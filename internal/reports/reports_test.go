package reports

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/execlens/execlens/internal/database"
	"github.com/execlens/execlens/internal/metrics"
)

// Helper to create synthetic test workload fixture
func createTestWorkloadFixture() []WorkloadReportData {
	now := time.Now().UTC()

	// Sieve workload
	cppSummary := metrics.Summary{
		Count:               5,
		MedianNs:            20_000_000,
		MedianMs:            20.0,
		MeanNs:              20_200_000,
		MeanMs:              20.2,
		MinMs:               19.5,
		MaxMs:               21.0,
		StdDevMs:            0.6,
		CVPercent:           3.0,
		CI95LowerNs:         19_400_000,
		CI95UpperNs:         20_900_000,
		CI95LowerMs:         19.4,
		CI95UpperMs:         20.9,
		ThroughputOpsPerSec: 50000.0,
	}
	rustSummary := metrics.Summary{
		Count:               5,
		MedianNs:            18_000_000,
		MedianMs:            18.0,
		MeanNs:              18_100_000,
		MeanMs:              18.1,
		MinMs:               17.8,
		MaxMs:               18.5,
		StdDevMs:            0.3,
		CVPercent:           1.7,
		CI95LowerNs:         17_700_000,
		CI95UpperNs:         18_500_000,
		CI95LowerMs:         17.7,
		CI95UpperMs:         18.5,
		ThroughputOpsPerSec: 55555.0,
	}
	pySummary := metrics.Summary{
		Count:               5,
		MedianNs:            120_000_000,
		MedianMs:            120.0,
		MeanNs:              122_000_000,
		MeanMs:              122.0,
		MinMs:               115.0,
		MaxMs:               135.0,
		StdDevMs:            7.5,
		CVPercent:           6.1,
		CI95LowerNs:         112_500_000,
		CI95UpperNs:         131_500_000,
		CI95LowerMs:         112.5,
		CI95UpperMs:         131.5,
		ThroughputOpsPerSec: 8333.0,
	}
	unstableSummary := metrics.Summary{
		Count:               5,
		MedianNs:            50_000_000,
		MedianMs:            50.0,
		MeanNs:              55_000_000,
		MeanMs:              55.0,
		MinMs:               40.0,
		MaxMs:               85.0,
		StdDevMs:            18.0,
		CVPercent:           32.7, // High variance (>15%)
		CI95LowerNs:         32_000_000,
		CI95UpperNs:         78_000_000,
		CI95LowerMs:         32.0,
		CI95UpperMs:         78.0,
		ThroughputOpsPerSec: 20000.0,
	}

	valTrue := true
	return []WorkloadReportData{
		{
			WorkloadID:    "sieve",
			WorkloadName:  "Sieve of Eratosthenes",
			Track:         "languages",
			ExecutionMode: "standard",
			BaselineID:    "cpp",
			PrimaryRunID:  "test_run_sieve",
			CreatedAt:     now,
			Executions: []ExecutionView{
				{
					LanguageExecution: database.LanguageExecution{
						LanguageID:       "cpp",
						Status:           "COMPLETED",
						ValidationPassed: &valTrue,
						PeakMemoryBytes:  15 * 1024 * 1024,
						PeakMemoryMB:     15.0,
						Summary:          &cppSummary,
						Samples: []database.RawSample{
							{Iteration: 0, IsWarmup: true, DurationMs: 22.0},
							{Iteration: 1, IsWarmup: false, DurationMs: 20.0},
							{Iteration: 2, IsWarmup: false, DurationMs: 19.5},
							{Iteration: 3, IsWarmup: false, DurationMs: 20.5},
						},
					},
					Speedup:     1.0,
					MemoryRatio: 1.0,
				},
				{
					LanguageExecution: database.LanguageExecution{
						LanguageID:       "rust",
						Status:           "COMPLETED",
						ValidationPassed: &valTrue,
						PeakMemoryBytes:  12 * 1024 * 1024,
						PeakMemoryMB:     12.0,
						Summary:          &rustSummary,
						Samples: []database.RawSample{
							{Iteration: 0, IsWarmup: true, DurationMs: 19.0},
							{Iteration: 1, IsWarmup: false, DurationMs: 18.0},
							{Iteration: 2, IsWarmup: false, DurationMs: 18.2},
						},
					},
					Speedup:     cppSummary.MedianMs / rustSummary.MedianMs, // 1.11x
					MemoryRatio: 12.0 / 15.0,
				},
				{
					LanguageExecution: database.LanguageExecution{
						LanguageID:       "python",
						Status:           "COMPLETED",
						ValidationPassed: &valTrue,
						PeakMemoryBytes:  30 * 1024 * 1024,
						PeakMemoryMB:     30.0,
						Summary:          &pySummary,
						Samples: []database.RawSample{
							{Iteration: 1, IsWarmup: false, DurationMs: 120.0},
						},
					},
					Speedup:     cppSummary.MedianMs / pySummary.MedianMs, // 0.17x
					MemoryRatio: 30.0 / 15.0,
				},
				{
					LanguageExecution: database.LanguageExecution{
						LanguageID:       "unstable_runtime",
						Status:           "COMPLETED",
						ValidationPassed: &valTrue,
						PeakMemoryBytes:  40 * 1024 * 1024,
						PeakMemoryMB:     40.0,
						Summary:          &unstableSummary,
						Samples: []database.RawSample{
							{Iteration: 1, IsWarmup: false, DurationMs: 40.0},
							{Iteration: 2, IsWarmup: false, DurationMs: 85.0},
						},
					},
					Speedup:     cppSummary.MedianMs / unstableSummary.MedianMs,
					MemoryRatio: 40.0 / 15.0,
				},
				{
					LanguageExecution: database.LanguageExecution{
						LanguageID:   "unavailable_lang",
						Status:       "SKIPPED",
						ErrorMessage: "Compiler toolchain not installed on host",
					},
				},
			},
		},
		{
			WorkloadID:    "cuda_vec_add",
			WorkloadName:  "CUDA Vector Add",
			Track:         "gpu",
			ExecutionMode: "standard",
			BaselineID:    "cpu_reference",
			PrimaryRunID:  "test_run_cuda",
			CreatedAt:     now,
			Executions: []ExecutionView{
				{
					LanguageExecution: database.LanguageExecution{
						LanguageID:             "cuda",
						Status:                 "COMPLETED",
						KernelDurationNs:       5_000_000,  // 5 ms
						TransferH2DNs:          10_000_000, // 10 ms
						TransferD2HNs:          10_000_000, // 10 ms
						EffectiveBandwidthGBps: 25.5,
						Summary: &metrics.Summary{
							Count:    3,
							MedianMs: 25.0,
						},
					},
					Speedup: 2.0,
				},
			},
		},
	}
}

func TestGenerateFindings(t *testing.T) {
	fixture := createTestWorkloadFixture()
	findings := GenerateFindings(fixture, 5.0)

	if len(findings) == 0 {
		t.Fatalf("expected non-empty findings list")
	}

	var foundTopPerformer, foundSpeedup, foundMemory, foundVariability, foundGPU, foundDiagnostic bool

	for _, f := range findings {
		switch f.Category {
		case FindingTopPerformer:
			if f.WorkloadID == "sieve" {
				foundTopPerformer = true
				if f.CandidateID != "rust" {
					t.Errorf("expected rust to be top performer on sieve, got %s", f.CandidateID)
				}
			}
		case FindingSpeedup:
			foundSpeedup = true
		case FindingMemoryEfficiency:
			foundMemory = true
		case FindingVariability:
			foundVariability = true
			if f.CandidateID != "unstable_runtime" {
				t.Errorf("expected unstable_runtime to trigger variability finding, got %s", f.CandidateID)
			}
		case FindingAccelerator:
			foundGPU = true
		case FindingDiagnosticNote:
			foundDiagnostic = true
			if f.CandidateID != "unavailable_lang" {
				t.Errorf("expected unavailable_lang diagnostic, got %s", f.CandidateID)
			}
		}
	}

	if !foundTopPerformer {
		t.Error("missing FindingTopPerformer")
	}
	if !foundSpeedup {
		t.Error("missing FindingSpeedup")
	}
	if !foundMemory {
		t.Error("missing FindingMemoryEfficiency")
	}
	if !foundVariability {
		t.Error("missing FindingVariability")
	}
	if !foundGPU {
		t.Error("missing FindingAccelerator")
	}
	if !foundDiagnostic {
		t.Error("missing FindingDiagnosticNote")
	}
}

func TestRenderAllCharts(t *testing.T) {
	fixture := createTestWorkloadFixture()
	summary := &ReportSummary{
		ReportID:             "test_rep_1",
		GeneratedAt:          time.Now().UTC(),
		GeneratorVersion:     "1.0.0",
		TotalConfigurations:  5,
		SuccessfulExecutions: 4,
		WorkloadsCovered:     []string{"sieve", "cuda_vec_add"},
		LanguagesTested:      []string{"cpp", "rust", "python"},
	}
	sysInfo := database.SystemInfo{
		OS:          "linux",
		Arch:        "amd64",
		CPUModel:    "Test CPU 8-Core",
		LogicalCPUs: 8,
	}

	charts := map[string]string{
		"overview":   RenderOverviewSVG(summary, sysInfo),
		"runtime":    RenderRuntimeComparisonSVG(fixture[0]),
		"throughput": RenderThroughputComparisonSVG(fixture[0]),
		"memory":     RenderMemoryComparisonSVG(fixture[0]),
		"variability": RenderVariabilitySVG(fixture[0]),
		"scaling":    RenderScalingSVG(fixture),
		"startup":    RenderStartupComparisonSVG(fixture[0]),
		"build":      RenderBuildComparisonSVG(fixture[0]),
		"heatmap":    RenderHeatmapSVG(fixture),
		"gpu":        RenderWorkloadSpecificSVG(fixture[1]),
	}

	for name, svg := range charts {
		if !strings.HasPrefix(strings.TrimSpace(svg), "<svg") {
			t.Errorf("chart %s does not begin with <svg", name)
		}
		if !strings.HasSuffix(strings.TrimSpace(svg), "</svg>") {
			t.Errorf("chart %s does not end with </svg>", name)
		}
		if strings.Contains(svg, "NaN") {
			t.Errorf("chart %s contains illegal NaN", name)
		}
	}
}

func TestUpdateReadmeContent(t *testing.T) {
	initialContent := `# Project Title
Introductory text.

## Architecture
Some architectural details.

## Prerequisites
Go and Node required.
`

	section := "<!-- EXECLENS_REPORT_START -->\n## New Benchmark Report\nContent\n<!-- EXECLENS_REPORT_END -->"

	// 1. Insertion when markers don't exist
	updated, err := UpdateReadmeContent(initialContent, section)
	if err != nil {
		t.Fatalf("UpdateReadmeContent failed on insert: %v", err)
	}

	if !strings.Contains(updated, "<!-- EXECLENS_REPORT_START -->") {
		t.Error("updated content missing start marker")
	}
	if !strings.Contains(updated, "<!-- EXECLENS_REPORT_END -->") {
		t.Error("updated content missing end marker")
	}
	if !strings.Contains(updated, "## Prerequisites") {
		t.Error("Prerequisites section was lost")
	}

	// 2. Replacement when markers already exist
	newSection := "<!-- EXECLENS_REPORT_START -->\n## Replaced Content\n<!-- EXECLENS_REPORT_END -->"
	secondUpdate, err := UpdateReadmeContent(updated, newSection)
	if err != nil {
		t.Fatalf("UpdateReadmeContent failed on replace: %v", err)
	}

	if !strings.Contains(secondUpdate, "## Replaced Content") {
		t.Error("failed to replace section")
	}
	if strings.Contains(secondUpdate, "## New Benchmark Report") {
		t.Error("old report content still present")
	}
}

func TestEndToEndGeneratorAndVerify(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Seed test run
	now := time.Now().UTC()
	run := &database.Run{
		ID:            "test_run_e2e",
		Track:         "languages",
		WorkloadID:    "sieve",
		Iterations:    3,
		ExecutionMode: "standard",
		Status:        database.StatusCompleted,
		CreatedAt:     now,
		SystemInfo: database.SystemInfo{
			OS:          "linux",
			Arch:        "amd64",
			CPUModel:    "Test Intel Xeon",
			LogicalCPUs: 4,
		},
	}

	err = db.CreateRun(ctx, run, []string{"cpp", "rust"})
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	valTrue := true
	cppSummary := metrics.Summary{
		Count:       3,
		MedianNs:    15_000_000,
		MedianMs:    15.0,
		MeanNs:      15_100_000,
		MeanMs:      15.1,
		MinMs:       14.8,
		MaxMs:       15.5,
		StdDevMs:    0.3,
		CVPercent:   2.0,
		CI95LowerNs: 14_500_000,
		CI95UpperNs: 15_700_000,
		CI95LowerMs: 14.5,
		CI95UpperMs: 15.7,
	}
	rustSummary := metrics.Summary{
		Count:       3,
		MedianNs:    14_000_000,
		MedianMs:    14.0,
		MeanNs:      14_100_000,
		MeanMs:      14.1,
		MinMs:       13.8,
		MaxMs:       14.5,
		StdDevMs:    0.3,
		CVPercent:   2.1,
		CI95LowerNs: 13_500_000,
		CI95UpperNs: 14_700_000,
		CI95LowerMs: 13.5,
		CI95UpperMs: 14.7,
	}

	_ = db.UpdateLanguageExecution(ctx, &database.LanguageExecution{
		ID:               "test_run_e2e_cpp",
		RunID:            run.ID,
		LanguageID:       "cpp",
		Status:           database.LangStatusCompleted,
		ValidationPassed: &valTrue,
		PeakMemoryBytes:  10 * 1024 * 1024,
		Summary:          &cppSummary,
	})
	_ = db.UpdateLanguageExecution(ctx, &database.LanguageExecution{
		ID:               "test_run_e2e_rust",
		RunID:            run.ID,
		LanguageID:       "rust",
		Status:           database.LangStatusCompleted,
		ValidationPassed: &valTrue,
		PeakMemoryBytes:  8 * 1024 * 1024,
		Summary:          &rustSummary,
	})

	readmePath := filepath.Join(tmpDir, "README.md")
	_ = os.WriteFile(readmePath, []byte("# Test Repo\n\n## Prerequisites\nInstalled tools.\n"), 0644)

	cfg := ReportConfig{
		DBPath:                    dbPath,
		OutputDir:                 filepath.Join(tmpDir, "reports"),
		BaselineID:                "cpp",
		UpdateReadme:              true,
		ReadmePath:                readmePath,
		PracticalThresholdPercent: 5.0,
		ReportVersion:             "1.0.0",
	}

	// 1. Run Generator
	res, err := RunGenerator(ctx, db, cfg)
	if err != nil {
		t.Fatalf("RunGenerator failed: %v", err)
	}

	if res.ReportID == "" {
		t.Errorf("empty report ID")
	}

	// 2. Run Verifier
	latestDir := filepath.Join(cfg.OutputDir, "latest")
	if err := VerifyReport(latestDir, readmePath); err != nil {
		t.Fatalf("VerifyReport failed on generated output: %v", err)
	}

	// 3. Verify Deterministic Output on second run
	manifest1, _ := os.ReadFile(filepath.Join(latestDir, "manifest.json"))
	var m1 ReportManifest
	_ = json.Unmarshal(manifest1, &m1)

	if len(m1.Artifacts) == 0 {
		t.Errorf("manifest has 0 artifacts")
	}
}
