package tests

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/execlens/execlens/internal/database"
	"github.com/execlens/execlens/internal/engine"
	"github.com/execlens/execlens/internal/jobs"
	"github.com/execlens/execlens/internal/languages"
	"github.com/execlens/execlens/internal/reports"
	"github.com/execlens/execlens/internal/workloads"
)

func TestEndToEndBenchmarkRun(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "integration.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	langReg := languages.NewDefaultRegistry()
	wlReg := workloads.NewRegistry()

	// Locate benchmarks directory relative to repo root
	benchmarksDir, err := filepath.Abs("../benchmarks")
	if err != nil {
		t.Fatalf("failed to resolve benchmarks dir: %v", err)
	}

	executor := engine.NewExecutor(db, langReg, wlReg, benchmarksDir)
	pool := jobs.NewWorkerPool(db, executor, 1)
	defer func() {
		_ = pool.Shutdown(context.Background())
	}()

	runID := "int_run_sieve"
	params := map[string]interface{}{"limit": 1000}

	run := &database.Run{
		ID:               runID,
		WorkloadID:       "sieve",
		WorkloadParams:   params,
		Iterations:       2,
		WarmupIterations: 1,
		ExecutionMode:    "standard",
		Status:           database.StatusQueued,
		CreatedAt:        time.Now().UTC(),
		SystemInfo:       engine.CollectSystemInfo(),
	}

	// Test Go and Python in integration test
	testLangs := []string{"go", "python"}
	if err := db.CreateRun(context.Background(), run, testLangs); err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	// Submit and wait for execution
	if err := pool.Submit(runID); err != nil {
		t.Fatalf("failed to submit run: %v", err)
	}

	// Wait up to 30 seconds for completion
	deadline := time.Now().Add(30 * time.Second)
	var finalDetail *database.RunDetail

	for time.Now().Before(deadline) {
		detail, err := db.GetRunDetail(context.Background(), runID, "go")
		if err == nil && detail != nil {
			if detail.Status == database.StatusCompleted || detail.Status == database.StatusFailed {
				finalDetail = detail
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	if finalDetail == nil {
		t.Fatalf("timed out waiting for benchmark run to finish")
	}

	if finalDetail.Status != database.StatusCompleted {
		t.Fatalf("expected run status COMPLETED, got %s (error: %s)", finalDetail.Status, finalDetail.ErrorMessage)
	}

	if len(finalDetail.Executions) != 2 {
		t.Fatalf("expected 2 language executions, got %d", len(finalDetail.Executions))
	}

	for _, e := range finalDetail.Executions {
		if e.Status != database.LangStatusCompleted {
			t.Errorf("expected execution %s to be COMPLETED, got %s (err: %s)", e.LanguageID, e.Status, e.ErrorMessage)
		}
		if e.ValidationPassed == nil || !*e.ValidationPassed {
			t.Errorf("expected validation to pass for %s", e.LanguageID)
		}
		if e.Summary == nil || e.Summary.MedianMs <= 0 {
			t.Errorf("expected valid summary metrics for %s", e.LanguageID)
		}
	}

	// Verify CSV and JSON exports
	jsonExport, err := reports.ExportRunJSON(finalDetail)
	if err != nil || len(jsonExport) == 0 {
		t.Errorf("JSON export failed: %v", err)
	}

	csvExport, err := reports.ExportRunCSV(finalDetail)
	if err != nil || len(csvExport) == 0 {
		t.Errorf("CSV export failed: %v", err)
	}
}

func TestRunCancellation(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "cancel.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	langReg := languages.NewDefaultRegistry()
	wlReg := workloads.NewRegistry()
	benchmarksDir, _ := filepath.Abs("../benchmarks")

	executor := engine.NewExecutor(db, langReg, wlReg, benchmarksDir)
	pool := jobs.NewWorkerPool(db, executor, 1)
	defer func() {
		_ = pool.Shutdown(context.Background())
	}()

	runID := "run_cancel_test"
	run := &database.Run{
		ID:               runID,
		WorkloadID:       "sieve",
		WorkloadParams:   map[string]interface{}{"limit": 50000000}, // Large limit
		Iterations:       20,
		WarmupIterations: 2,
		ExecutionMode:    "standard",
		Status:           database.StatusQueued,
		CreatedAt:        time.Now().UTC(),
		SystemInfo:       engine.CollectSystemInfo(),
	}

	if err := db.CreateRun(context.Background(), run, []string{"python"}); err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	if err := pool.Submit(runID); err != nil {
		t.Fatalf("failed to submit run: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	// Issue cancellation
	if err := pool.Cancel(runID); err != nil {
		t.Fatalf("failed to cancel run: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	r, _ := db.GetRun(context.Background(), runID)
	if r.Status != database.StatusCancelled && r.Status != database.StatusFailed {
		t.Errorf("expected CANCELLED or FAILED, got %s", r.Status)
	}
}
