package lowlevel

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/execlens/execlens/internal/database"
	"github.com/execlens/execlens/internal/engine"
	"github.com/execlens/execlens/internal/metrics"
)

// Runner manages execution of low-level CPU & compiler lab benchmarks.
type Runner struct {
	db            *database.DB
	benchmarksDir string
}

func NewRunner(db *database.DB, benchmarksDir string) *Runner {
	return &Runner{
		db:            db,
		benchmarksDir: benchmarksDir,
	}
}

// ExecuteRun executes all variants or compiler configs for a low-level benchmark run.
func (r *Runner) ExecuteRun(ctx context.Context, run *database.Run) error {
	wl, err := GetWorkload(run.WorkloadID)
	if err != nil {
		_ = r.db.UpdateRunStatus(ctx, run.ID, database.StatusFailed, err.Error())
		return err
	}

	detail, err := r.db.GetRunDetail(ctx, run.ID, "")
	if err != nil {
		_ = r.db.UpdateRunStatus(ctx, run.ID, database.StatusFailed, err.Error())
		return err
	}

	_ = r.db.UpdateRunStatus(ctx, run.ID, database.StatusPreparing, "")

	tempDir, err := os.MkdirTemp("", fmt.Sprintf("execlens_lowlevel_%s_*", run.ID))
	if err != nil {
		_ = r.db.UpdateRunStatus(ctx, run.ID, database.StatusFailed, "failed to create temp dir")
		return err
	}
	defer os.RemoveAll(tempDir)

	_ = r.db.UpdateRunStatus(ctx, run.ID, database.StatusRunning, "")

	sourcePath := wl.SourcePath(r.benchmarksDir)
	hasAnySuccess := false
	var lastErr error

	for _, execRecord := range detail.Executions {
		if ctx.Err() != nil {
			_ = r.db.UpdateRunStatus(ctx, run.ID, database.StatusCancelled, "cancelled by user")
			return ctx.Err()
		}

		err := r.executeVariant(ctx, run, &execRecord, wl, sourcePath, tempDir)
		if err != nil {
			lastErr = err
		} else if execRecord.Status == database.LangStatusCompleted {
			hasAnySuccess = true
		}
	}

	if hasAnySuccess {
		_ = r.db.UpdateRunStatus(ctx, run.ID, database.StatusCompleted, "")
	} else {
		errMsg := "All low-level executions failed"
		if lastErr != nil {
			errMsg = lastErr.Error()
		}
		_ = r.db.UpdateRunStatus(ctx, run.ID, database.StatusFailed, errMsg)
	}

	return nil
}

func (r *Runner) executeVariant(ctx context.Context, run *database.Run, execRecord *database.LanguageExecution, wl *LowLevelWorkload, sourcePath, tempDir string) error {
	// Parse compiler and variant from execution ID:
	// Format could be:
	// 1. Variant ID directly (e.g. "simd_avx2", "autovec", "scalar") with default GCC -O3
	// 2. Or "gcc-O0", "gcc-O3", "clang-O3", "clang-Ofast"
	variantID := execRecord.LanguageID
	compilerName := "gcc"
	optLevel := "-O3"

	if strings.Contains(variantID, "-O") {
		parts := strings.Split(variantID, "-")
		if len(parts) >= 2 {
			compilerName = parts[0]
			optLevel = "-" + parts[1]
			variantID = "" // uses default variant
		}
	}

	cfg := BuildConfig{
		CompilerName: compilerName,
		OptLevel:     optLevel,
	}

	execRecord.Status = database.LangStatusBuilding
	_ = r.db.UpdateLanguageExecution(ctx, execRecord)

	binPath := filepath.Join(tempDir, fmt.Sprintf("bin_%s", execRecord.ID))
	artifacts, err := BuildLowLevel(ctx, sourcePath, binPath, cfg)

	execRecord.CompilerName = cfg.CompilerName
	execRecord.OptLevel = cfg.OptLevel
	execRecord.CompilerFlags = fmt.Sprintf("%s -std=c++20 -mavx2 -mfma", cfg.OptLevel)
	execRecord.TargetArch = "x86_64"

	if artifacts != nil {
		execRecord.BuildCommand = artifacts.BuildCommand
		execRecord.BuildDurationNs = artifacts.BuildDuration.Nanoseconds()
		execRecord.BuildStdout = artifacts.BuildStdout
		execRecord.BuildStderr = artifacts.BuildStderr
		execRecord.BuildExitCode = artifacts.ExitCode
		execRecord.ExecutableSizeBytes = artifacts.BinarySizeBytes
		execRecord.GeneratedAssembly = artifacts.AssemblyCode
		execRecord.CompilerRemarks = artifacts.OptimizerRemarks
	}

	if err != nil {
		execRecord.Status = database.LangStatusFailed
		execRecord.ErrorMessage = err.Error()
		_ = r.db.UpdateLanguageExecution(ctx, execRecord)
		return err
	}

	// Determine workload arguments
	sizeVal := wl.DefaultSize
	if s, ok := run.WorkloadParams["size"]; ok {
		switch v := s.(type) {
		case float64:
			sizeVal = int64(v)
		case int64:
			sizeVal = v
		case int:
			sizeVal = int64(v)
		}
	}

	sizeFlag := "--size"
	if wl.ID == "lowlevel_memcpy" {
		sizeFlag = "--bytes"
	}

	var runArgs []string
	runArgs = append(runArgs, sizeFlag, fmt.Sprintf("%d", sizeVal))
	if variantID != "" {
		runArgs = append(runArgs, "--variant", variantID)
	}

	// 1. Validation Run
	execRecord.Status = database.LangStatusValidating
	_ = r.db.UpdateLanguageExecution(ctx, execRecord)

	valArgs := append(runArgs, "--validate")
	vRes, err := runProcess(ctx, binPath, valArgs, 15*time.Second)
	if err != nil || vRes.ExitCode != 0 {
		valPassed := false
		execRecord.ValidationPassed = &valPassed
		execRecord.Status = database.LangStatusFailed
		execRecord.ErrorMessage = fmt.Sprintf("Validation failed: exit code %d, output: %s", vRes.ExitCode, vRes.Stdout)
		_ = r.db.UpdateLanguageExecution(ctx, execRecord)
		return fmt.Errorf("validation failed: %s", execRecord.ErrorMessage)
	}

	// Parse validation JSON output
	var valData struct {
		Verified bool   `json:"verified"`
		Checksum string `json:"checksum"`
	}
	_ = json.Unmarshal([]byte(vRes.Stdout), &valData)
	valPassed := valData.Verified
	execRecord.ValidationPassed = &valPassed
	execRecord.ValidationOutput = vRes.Stdout
	execRecord.ValidationExpect = "verified: true"

	if !valPassed {
		execRecord.Status = database.LangStatusFailed
		execRecord.ErrorMessage = "Correctness verification failed (result outside tolerance)"
		_ = r.db.UpdateLanguageExecution(ctx, execRecord)
		return fmt.Errorf("correctness failed")
	}

	// 2. Warm-up Iterations
	warmupCount := run.WarmupIterations
	if run.ExecutionMode == "cold_start" {
		warmupCount = 0
	}

	if warmupCount > 0 {
		execRecord.Status = database.LangStatusWarmingUp
		_ = r.db.UpdateLanguageExecution(ctx, execRecord)

		for w := 0; w < warmupCount; w++ {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			wRes, err := runProcess(ctx, binPath, runArgs, 30*time.Second)
			if err != nil {
				continue
			}
			sample := &database.RawSample{
				LanguageExecutionID: execRecord.ID,
				Iteration:           w + 1,
				IsWarmup:            true,
				DurationNs:          wRes.DurationNs,
				MemoryBytes:         wRes.MemoryBytes,
				ExitCode:            wRes.ExitCode,
				CreatedAt:           time.Now().UTC(),
			}
			_ = r.db.AddRawSample(ctx, sample)
		}
	}

	// 3. Timed Measurement Iterations
	execRecord.Status = database.LangStatusRunning
	_ = r.db.UpdateLanguageExecution(ctx, execRecord)

	var durationsNs []int64
	var peakMemory int64

	for iter := 1; iter <= run.Iterations; iter++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		res, err := runProcess(ctx, binPath, runArgs, 30*time.Second)
		if err != nil {
			execRecord.Status = database.LangStatusFailed
			execRecord.ErrorMessage = fmt.Sprintf("Iteration %d failed: %v", iter, err)
			_ = r.db.UpdateLanguageExecution(ctx, execRecord)
			return err
		}

		durationsNs = append(durationsNs, res.DurationNs)
		if res.MemoryBytes > peakMemory {
			peakMemory = res.MemoryBytes
		}

		sample := &database.RawSample{
			LanguageExecutionID: execRecord.ID,
			Iteration:           iter,
			IsWarmup:            false,
			DurationNs:          res.DurationNs,
			MemoryBytes:         res.MemoryBytes,
			ExitCode:            res.ExitCode,
			CreatedAt:           time.Now().UTC(),
		}
		_ = r.db.AddRawSample(ctx, sample)
	}

	// 4. Summarize and complete
	var durations []time.Duration
	for _, ns := range durationsNs {
		durations = append(durations, time.Duration(ns))
	}
	if summary, err := metrics.CalculateSummary(durations, peakMemory, sizeVal, 0); err == nil {
		execRecord.Summary = &summary
	}
	execRecord.PeakMemoryBytes = peakMemory
	execRecord.Status = database.LangStatusCompleted
	now := time.Now().UTC()
	execRecord.CompletedAt = &now

	_ = r.db.UpdateLanguageExecution(ctx, execRecord)
	return nil
}

type processOutcome struct {
	DurationNs  int64
	MemoryBytes int64
	ExitCode    int
	Stdout      string
	Stderr      string
}

func runProcess(ctx context.Context, binPath string, args []string, timeout time.Duration) (processOutcome, error) {
	res := engine.RunCommand(ctx, binPath, args, "", nil, timeout)
	outcome := processOutcome{
		DurationNs:  res.Duration.Nanoseconds(),
		MemoryBytes: res.PeakMemory,
		ExitCode:    res.ExitCode,
		Stdout:      res.Stdout,
		Stderr:      res.Stderr,
	}
	if res.ExitCode != 0 || res.TimedOut || res.Cancelled {
		return outcome, fmt.Errorf("process exited with code %d: %s", res.ExitCode, res.ErrorMessage)
	}
	return outcome, nil
}
