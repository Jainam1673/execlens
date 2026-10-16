package database

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestDatabaseLifecycleAndRuns(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_execlens.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Verify initial stats
	stats, err := db.GetDashboardStats(ctx)
	if err != nil {
		t.Fatalf("failed to get dashboard stats: %v", err)
	}
	if stats.TotalRuns != 0 {
		t.Errorf("expected 0 total runs, got %d", stats.TotalRuns)
	}

	// Create a run
	runID := "test_run_123"
	run := &Run{
		ID:               runID,
		WorkloadID:       "sieve",
		WorkloadParams:   map[string]interface{}{"limit": 1000000},
		Iterations:       5,
		WarmupIterations: 2,
		ExecutionMode:    "standard",
		Status:           StatusQueued,
		CreatedAt:        time.Now().UTC(),
		SystemInfo: SystemInfo{
			OS:          "linux",
			Arch:        "amd64",
			CPUModel:    "Test CPU",
			LogicalCPUs: 8,
			TotalMemory: 1024 * 1024 * 1024 * 16,
		},
	}

	err = db.CreateRun(ctx, run, []string{"go", "rust", "cpp"})
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	// Fetch run
	fetched, err := db.GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("failed to get run: %v", err)
	}
	if fetched == nil || fetched.ID != runID {
		t.Fatalf("fetched run mismatch: %+v", fetched)
	}

	// Update run status
	err = db.UpdateRunStatus(ctx, runID, StatusRunning, "")
	if err != nil {
		t.Fatalf("failed to update run status: %v", err)
	}

	// Add a raw sample
	sample := &RawSample{
		LanguageExecutionID: runID + "_go",
		Iteration:           1,
		IsWarmup:            false,
		DurationNs:          12500000, // 12.5ms
		MemoryBytes:         1024 * 1024 * 4,
		ExitCode:            0,
		CreatedAt:           time.Now().UTC(),
	}
	err = db.AddRawSample(ctx, sample)
	if err != nil {
		t.Fatalf("failed to add raw sample: %v", err)
	}

	// Update language execution
	valPassed := true
	exec := &LanguageExecution{
		ID:               runID + "_go",
		Status:           LangStatusCompleted,
		ToolchainVersion: "go1.27.1",
		ValidationPassed: &valPassed,
		PeakMemoryBytes:  1024 * 1024 * 4,
	}
	err = db.UpdateLanguageExecution(ctx, exec)
	if err != nil {
		t.Fatalf("failed to update language execution: %v", err)
	}

	// Query run detail
	detail, err := db.GetRunDetail(ctx, runID, "go")
	if err != nil {
		t.Fatalf("failed to get run detail: %v", err)
	}
	if len(detail.Executions) != 3 {
		t.Errorf("expected 3 executions, got %d", len(detail.Executions))
	}

	// Test reconciliation on restart
	err = db.ReconcileInterruptedRuns(ctx)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}
	reconciled, _ := db.GetRun(ctx, runID)
	if reconciled.Status != StatusFailed {
		t.Errorf("expected status FAILED after reconciliation, got %s", reconciled.Status)
	}
}
