package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/execlens/execlens/internal/database"
	"github.com/execlens/execlens/internal/languages"
	"github.com/execlens/execlens/internal/metrics"
	"github.com/execlens/execlens/internal/workloads"
)

// Executor orchestrates build, validation, and benchmarking of workloads.
type Executor struct {
	db            *database.DB
	langRegistry  *languages.Registry
	workloadReg   *workloads.Registry
	benchmarksDir string
}

func NewExecutor(db *database.DB, langReg *languages.Registry, wlReg *workloads.Registry, benchmarksDir string) *Executor {
	absDir, err := filepath.Abs(benchmarksDir)
	if err == nil {
		benchmarksDir = absDir
	}
	return &Executor{
		db:            db,
		langRegistry:  langReg,
		workloadReg:   wlReg,
		benchmarksDir: benchmarksDir,
	}
}

// ExecuteRun executes all language benchmarks for a given run.
func (e *Executor) ExecuteRun(ctx context.Context, runID string) error {
	run, err := e.db.GetRun(ctx, runID)
	if err != nil || run == nil {
		return fmt.Errorf("run not found: %s", runID)
	}

	workload, err := e.workloadReg.Get(run.WorkloadID)
	if err != nil {
		_ = e.db.UpdateRunStatus(ctx, runID, database.StatusFailed, err.Error())
		return err
	}

	detail, err := e.db.GetRunDetail(ctx, runID, "")
	if err != nil {
		_ = e.db.UpdateRunStatus(ctx, runID, database.StatusFailed, err.Error())
		return err
	}

	_ = e.db.UpdateRunStatus(ctx, runID, database.StatusPreparing, "")

	// Create temporary sandbox directory for compiled binaries and datasets
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("execlens_%s_*", runID))
	if err != nil {
		_ = e.db.UpdateRunStatus(ctx, runID, database.StatusFailed, "failed to create temp dir")
		return err
	}
	defer os.RemoveAll(tempDir)

	// Prepare dataset if required by workload contract
	var datasetPath string
	if workload.PrepareDataset != nil {
		p, err := workload.PrepareDataset(run.WorkloadParams, tempDir)
		if err != nil {
			_ = e.db.UpdateRunStatus(ctx, runID, database.StatusFailed, fmt.Sprintf("dataset preparation failed: %v", err))
			return err
		}
		datasetPath = p
	}

	_ = e.db.UpdateRunStatus(ctx, runID, database.StatusRunning, "")

	hasAnySuccess := false
	var lastErr error

	for _, exec := range detail.Executions {
		if ctx.Err() != nil {
			_ = e.db.UpdateRunStatus(ctx, runID, database.StatusCancelled, "Run cancelled by user")
			return ctx.Err()
		}

		err := e.executeLanguage(ctx, run, &exec, workload, datasetPath, tempDir)
		if err != nil {
			lastErr = err
		} else if exec.Status == database.LangStatusCompleted {
			hasAnySuccess = true
		}
	}

	if ctx.Err() != nil {
		_ = e.db.UpdateRunStatus(ctx, runID, database.StatusCancelled, "Run cancelled by user")
		return ctx.Err()
	}

	if hasAnySuccess {
		_ = e.db.UpdateRunStatus(ctx, runID, database.StatusCompleted, "")
	} else {
		errMsg := "All language executions failed"
		if lastErr != nil {
			errMsg = lastErr.Error()
		}
		_ = e.db.UpdateRunStatus(ctx, runID, database.StatusFailed, errMsg)
	}

	return nil
}

func (e *Executor) executeLanguage(ctx context.Context, run *database.Run, execRecord *database.LanguageExecution, workload *workloads.WorkloadContract, datasetPath, tempDir string) error {
	adapter, err := e.langRegistry.Get(execRecord.LanguageID)
	if err != nil {
		execRecord.Status = database.LangStatusFailed
		execRecord.ErrorMessage = err.Error()
		_ = e.db.UpdateLanguageExecution(ctx, execRecord)
		return err
	}

	// 1. Probing availability
	isAvail, version, missing := adapter.DetectAvailability(ctx)
	if !isAvail {
		execRecord.Status = database.LangStatusSkipped
		execRecord.ErrorMessage = fmt.Sprintf("Toolchain unavailable: %s", missing)
		_ = e.db.UpdateLanguageExecution(ctx, execRecord)
		return nil
	}
	execRecord.ToolchainVersion = version

	sourceDir := filepath.Join(e.benchmarksDir, adapter.SourceSubdir())
	langTempDir := filepath.Join(tempDir, adapter.ID())
	_ = os.MkdirAll(langTempDir, 0755)

	targetBin := filepath.Join(langTempDir, fmt.Sprintf("bin_%s", workload.ID))
	if adapter.ID() == "java" {
		targetBin = langTempDir // Java expects directory for classes
	} else if !adapter.IsCompiled() {
		targetBin = sourceDir // Interpreted languages execute source file directly
	}

	// 2. Compilation / Build (if applicable)
	if adapter.IsCompiled() {
		execRecord.Status = database.LangStatusBuilding
		_ = e.db.UpdateLanguageExecution(ctx, execRecord)

		bCmd, bArgs, bDir := adapter.BuildCommand(workload.ID, sourceDir, targetBin)
		execRecord.BuildCommand = fmt.Sprintf("%s %s", bCmd, strings.Join(bArgs, " "))

		bRes := RunCommand(ctx, bCmd, bArgs, bDir, adapter.EnvironmentVariables(), adapter.BuildTimeout())
		execRecord.BuildDurationNs = bRes.Duration.Nanoseconds()
		execRecord.BuildStdout = bRes.Stdout
		execRecord.BuildStderr = bRes.Stderr
		execRecord.BuildExitCode = bRes.ExitCode

		if bRes.ExitCode != 0 || bRes.TimedOut || bRes.Cancelled {
			execRecord.Status = database.LangStatusFailed
			execRecord.ErrorMessage = fmt.Sprintf("Build failed with code %d: %s", bRes.ExitCode, bRes.ErrorMessage)
			_ = e.db.UpdateLanguageExecution(ctx, execRecord)
			return fmt.Errorf("build failed: %s", execRecord.ErrorMessage)
		}
	}

	cliArgs := workload.ToCLIArgs(run.WorkloadParams, datasetPath)
	eCmd, eArgs, eDir := adapter.ExecutionCommand(workload.ID, targetBin, cliArgs)

	// 3. Correctness Validation
	execRecord.Status = database.LangStatusValidating
	_ = e.db.UpdateLanguageExecution(ctx, execRecord)

	valRes := RunCommand(ctx, eCmd, eArgs, eDir, adapter.EnvironmentVariables(), adapter.ExecutionTimeout())
	if valRes.ExitCode != 0 {
		valPassed := false
		execRecord.ValidationPassed = &valPassed
		execRecord.ValidationOutput = valRes.Stderr
		execRecord.Status = database.LangStatusFailed
		execRecord.ErrorMessage = fmt.Sprintf("Validation run failed with exit code %d: %s", valRes.ExitCode, valRes.ErrorMessage)
		_ = e.db.UpdateLanguageExecution(ctx, execRecord)
		return fmt.Errorf("validation run failed: %s", execRecord.ErrorMessage)
	}

	valPassed, actualOut, expectedOut := workload.ValidateResult(run.WorkloadParams, valRes.Stdout)
	execRecord.ValidationPassed = &valPassed
	execRecord.ValidationOutput = actualOut
	execRecord.ValidationExpect = expectedOut

	if !valPassed {
		execRecord.Status = database.LangStatusFailed
		execRecord.ErrorMessage = fmt.Sprintf("Correctness validation failed: got '%s', expected '%s'", actualOut, expectedOut)
		_ = e.db.UpdateLanguageExecution(ctx, execRecord)
		return fmt.Errorf("correctness validation failed: %s", execRecord.ErrorMessage)
	}

	// 4. Warm-up Iterations
	warmupCount := run.WarmupIterations
	if run.ExecutionMode == "cold_start" {
		warmupCount = 0
	}

	if warmupCount > 0 {
		execRecord.Status = database.LangStatusWarmingUp
		_ = e.db.UpdateLanguageExecution(ctx, execRecord)

		for w := 0; w < warmupCount; w++ {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			wRes := RunCommand(ctx, eCmd, eArgs, eDir, adapter.EnvironmentVariables(), adapter.ExecutionTimeout())
			sample := &database.RawSample{
				LanguageExecutionID: execRecord.ID,
				Iteration:           w + 1,
				IsWarmup:            true,
				DurationNs:          wRes.Duration.Nanoseconds(),
				MemoryBytes:         wRes.PeakMemory,
				ExitCode:            wRes.ExitCode,
				CreatedAt:           time.Now().UTC(),
			}
			_ = e.db.AddRawSample(ctx, sample)
		}
	}

	// 5. Timed Measured Iterations
	execRecord.Status = database.LangStatusRunning
	_ = e.db.UpdateLanguageExecution(ctx, execRecord)

	durations := make([]time.Duration, 0, run.Iterations)
	var maxPeakMemory int64 = 0

	for i := 0; i < run.Iterations; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		mRes := RunCommand(ctx, eCmd, eArgs, eDir, adapter.EnvironmentVariables(), adapter.ExecutionTimeout())
		if mRes.ExitCode != 0 {
			execRecord.Status = database.LangStatusFailed
			execRecord.ErrorMessage = fmt.Sprintf("Iteration %d failed with code %d: %s", i+1, mRes.ExitCode, mRes.ErrorMessage)
			_ = e.db.UpdateLanguageExecution(ctx, execRecord)
			return fmt.Errorf("iteration %d failed: %s", i+1, execRecord.ErrorMessage)
		}

		durations = append(durations, mRes.Duration)
		if mRes.PeakMemory > maxPeakMemory {
			maxPeakMemory = mRes.PeakMemory
		}

		sample := &database.RawSample{
			LanguageExecutionID: execRecord.ID,
			Iteration:           i + 1,
			IsWarmup:            false,
			DurationNs:          mRes.Duration.Nanoseconds(),
			MemoryBytes:         mRes.PeakMemory,
			ExitCode:            mRes.ExitCode,
			CreatedAt:           time.Now().UTC(),
		}
		_ = e.db.AddRawSample(ctx, sample)
	}

	// 6. Statistical Summary
	totalOps := workload.OperationsCount(run.WorkloadParams)
	totalBytes := workload.TotalBytesCount(run.WorkloadParams)

	summary, err := metrics.CalculateSummary(durations, maxPeakMemory, totalOps, totalBytes)
	if err == nil {
		execRecord.Summary = &summary
	}
	execRecord.PeakMemoryBytes = maxPeakMemory
	execRecord.Status = database.LangStatusCompleted
	now := time.Now().UTC()
	execRecord.CompletedAt = &now

	_ = e.db.UpdateLanguageExecution(ctx, execRecord)
	return nil
}
