package accelerators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/execlens/execlens/internal/database"
	"github.com/execlens/execlens/internal/engine"
	"github.com/execlens/execlens/internal/metrics"
)

// Runner manages accelerator and GPU benchmark builds and executions.
type Runner struct {
	db            *database.DB
	registry      *Registry
	benchmarksDir string
}

func NewRunner(db *database.DB, registry *Registry, benchmarksDir string) *Runner {
	return &Runner{
		db:            db,
		registry:      registry,
		benchmarksDir: benchmarksDir,
	}
}

// ExecuteRun executes all accelerator variants/backends for a run.
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

	tempDir, err := os.MkdirTemp("", fmt.Sprintf("execlens_accel_%s_*", run.ID))
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

		backendID := execRecord.LanguageID // e.g. "cuda", "opencl", "sycl", "hip"
		if backendID == "" {
			backendID = "cuda"
		}

		backend, err := r.registry.Get(backendID)
		if err != nil || !backend.IsAvailable {
			// Fail honestly and record exact diagnostics without fabricating numbers
			execRecord.Status = database.LangStatusSkipped
			if !backend.IsAvailable {
				execRecord.ErrorMessage = fmt.Sprintf("Accelerator backend unavailable: %s", backend.MissingReason)
			} else {
				execRecord.ErrorMessage = err.Error()
			}
			_ = r.db.UpdateLanguageExecution(ctx, &execRecord)
			continue
		}

		err = r.executeCUDABackend(ctx, run, &execRecord, wl, backend, sourcePath, tempDir)
		if err != nil {
			lastErr = err
		} else if execRecord.Status == database.LangStatusCompleted {
			hasAnySuccess = true
		}
	}

	if hasAnySuccess {
		_ = r.db.UpdateRunStatus(ctx, run.ID, database.StatusCompleted, "")
	} else {
		errMsg := "No accelerator executions succeeded (hardware/toolchain unavailable or execution failed)"
		if lastErr != nil {
			errMsg = lastErr.Error()
		}
		_ = r.db.UpdateRunStatus(ctx, run.ID, database.StatusFailed, errMsg)
	}

	return nil
}

type cudaRunOutput struct {
	Workload               string  `json:"workload"`
	DeviceName             string  `json:"device_name"`
	ComputeCapability      string  `json:"compute_capability"`
	Size                   int64   `json:"size"`
	Bytes                  int64   `json:"bytes"`
	AllocMs                float64 `json:"alloc_ms"`
	H2DMs                  float64 `json:"h2d_ms"`
	KernelMs               float64 `json:"kernel_ms"`
	D2HMs                  float64 `json:"d2h_ms"`
	TotalMs                float64 `json:"total_ms"`
	EffectiveBandwidthGBps float64 `json:"effective_bandwidth_gbps"`
	GFLOPS                 float64 `json:"gflops"`
	CPURefMs               float64 `json:"cpu_ref_ms"`
	Verified               bool    `json:"verified"`
}

func (r *Runner) executeCUDABackend(ctx context.Context, run *database.Run, execRecord *database.LanguageExecution, wl *AcceleratorWorkload, backend AcceleratorBackend, sourcePath, tempDir string) error {
	binPath := filepath.Join(tempDir, fmt.Sprintf("cuda_%s", wl.ID))

	// 1. Compile CUDA kernel
	execRecord.Status = database.LangStatusBuilding
	_ = r.db.UpdateLanguageExecution(ctx, execRecord)

	archFlag := fmt.Sprintf("-arch=sm_%s", strings.ReplaceAll(backend.ComputeCapability, ".", ""))
	nvccArgs := []string{
		"-O3",
		"-std=c++17",
		archFlag,
		sourcePath,
		"-o", binPath,
	}

	execRecord.BuildCommand = fmt.Sprintf("%s %s", backend.CompilerPath, strings.Join(nvccArgs, " "))
	tStart := time.Now()
	bCmd := exec.CommandContext(ctx, backend.CompilerPath, nvccArgs...)
	bOut, err := bCmd.CombinedOutput()
	bDuration := time.Since(tStart)

	execRecord.BuildDurationNs = bDuration.Nanoseconds()
	execRecord.BuildStdout = string(bOut)

	if err != nil {
		execRecord.Status = database.LangStatusFailed
		execRecord.ErrorMessage = fmt.Sprintf("nvcc compilation failed: %s", string(bOut))
		_ = r.db.UpdateLanguageExecution(ctx, execRecord)
		return err
	}

	// 2. Validation
	execRecord.Status = database.LangStatusValidating
	_ = r.db.UpdateLanguageExecution(ctx, execRecord)

	valRes := engine.RunCommand(ctx, binPath, []string{"--validate"}, "", nil, 30*time.Second)
	if valRes.ExitCode != 0 {
		valPassed := false
		execRecord.ValidationPassed = &valPassed
		execRecord.Status = database.LangStatusFailed
		execRecord.ErrorMessage = fmt.Sprintf("CUDA validation failed with exit code %d: %s", valRes.ExitCode, valRes.Stdout)
		_ = r.db.UpdateLanguageExecution(ctx, execRecord)
		return fmt.Errorf("CUDA validation failed")
	}

	var parsedOut cudaRunOutput
	_ = json.Unmarshal([]byte(valRes.Stdout), &parsedOut)
	valPassed := parsedOut.Verified
	execRecord.ValidationPassed = &valPassed
	execRecord.ValidationOutput = valRes.Stdout
	execRecord.ValidationExpect = "verified: true"

	// 3. Execution iterations
	execRecord.Status = database.LangStatusRunning
	_ = r.db.UpdateLanguageExecution(ctx, execRecord)

	var durationsNs []int64
	var peakMemory int64
	var lastTelemetry cudaRunOutput

	for iter := 1; iter <= run.Iterations; iter++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		res := engine.RunCommand(ctx, binPath, nil, "", nil, 60*time.Second)
		if res.ExitCode != 0 {
			execRecord.Status = database.LangStatusFailed
			execRecord.ErrorMessage = fmt.Sprintf("CUDA execution iteration %d failed: %s", iter, res.Stderr)
			_ = r.db.UpdateLanguageExecution(ctx, execRecord)
			return fmt.Errorf("CUDA iteration failed")
		}

		_ = json.Unmarshal([]byte(res.Stdout), &lastTelemetry)

		durationsNs = append(durationsNs, res.Duration.Nanoseconds())
		if res.PeakMemory > peakMemory {
			peakMemory = res.PeakMemory
		}

		sample := &database.RawSample{
			LanguageExecutionID: execRecord.ID,
			Iteration:           iter,
			IsWarmup:            false,
			DurationNs:          res.Duration.Nanoseconds(),
			MemoryBytes:         res.PeakMemory,
			ExitCode:            res.ExitCode,
			CreatedAt:           time.Now().UTC(),
		}
		_ = r.db.AddRawSample(ctx, sample)
	}

	// Store GPU metrics breakdown
	execRecord.GPUDeviceName = backend.DeviceName
	execRecord.GPUComputeCapability = backend.ComputeCapability
	execRecord.GPUDriverVersion = backend.DriverVersion
	execRecord.KernelDurationNs = int64(lastTelemetry.KernelMs * 1e6)
	execRecord.TransferH2DNs = int64(lastTelemetry.H2DMs * 1e6)
	execRecord.TransferD2HNs = int64(lastTelemetry.D2HMs * 1e6)
	execRecord.AllocDurationNs = int64(lastTelemetry.AllocMs * 1e6)
	execRecord.EffectiveBandwidthGBps = lastTelemetry.EffectiveBandwidthGBps
	execRecord.GFLOPS = lastTelemetry.GFLOPS

	var durations []time.Duration
	for _, ns := range durationsNs {
		durations = append(durations, time.Duration(ns))
	}
	if summary, err := metrics.CalculateSummary(durations, peakMemory, wl.DefaultSize, 0); err == nil {
		execRecord.Summary = &summary
	}
	execRecord.PeakMemoryBytes = peakMemory
	execRecord.Status = database.LangStatusCompleted
	now := time.Now().UTC()
	execRecord.CompletedAt = &now

	_ = r.db.UpdateLanguageExecution(ctx, execRecord)
	return nil
}
