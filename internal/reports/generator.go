package reports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/execlens/execlens/internal/database"
	"github.com/execlens/execlens/internal/metrics"
)

// RunGenerator executes the automated reporting and visualization pipeline.
func RunGenerator(ctx context.Context, db *database.DB, cfg ReportConfig) (*GenerationResult, error) {
	if cfg.OutputDir == "" {
		cfg.OutputDir = "reports"
	}
	if cfg.BaselineID == "" {
		cfg.BaselineID = "cpp"
	}
	if cfg.PracticalThresholdPercent <= 0 {
		cfg.PracticalThresholdPercent = 5.0
	}
	if cfg.ReportVersion == "" {
		cfg.ReportVersion = "1.0.0"
	}
	if cfg.ReadmePath == "" {
		cfg.ReadmePath = "README.md"
	}

	// 1. Identify and load runs from database
	runDetails, err := collectRunDetails(ctx, db, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to collect benchmark runs: %w", err)
	}

	if len(runDetails) == 0 {
		return nil, fmt.Errorf("no verified benchmark runs found matching the configuration")
	}

	// 2. Aggregate into WorkloadReportData
	workloads := aggregateWorkloads(runDetails, cfg.BaselineID)

	// 3. Build Summary & Environment metadata
	firstRun := runDetails[0]
	summary := buildSummary(runDetails, workloads, cfg.ReportVersion, firstRun.SystemInfo)

	// 4. Derive Deterministic Findings
	findings := GenerateFindings(workloads, cfg.PracticalThresholdPercent)

	// Update summary finding counts
	for _, f := range findings {
		if f.Status == metrics.AssessmentStatisticallySig {
			summary.StatSignificantCount++
		} else if f.Status == metrics.AssessmentInconclusive {
			summary.InconclusiveCount++
		}
	}

	// 5. Setup Directory Paths
	historyDir := filepath.Join(cfg.OutputDir, "history", summary.ReportID)
	latestDir := filepath.Join(cfg.OutputDir, "latest")
	chartsDir := filepath.Join(historyDir, "charts")

	if err := os.MkdirAll(chartsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create history charts directory: %w", err)
	}
	if err := os.MkdirAll(latestDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create latest directory: %w", err)
	}

	// 6. Generate All Vector SVG Charts
	generatedCharts, err := generateCharts(chartsDir, summary, workloads, firstRun.SystemInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to generate charts: %w", err)
	}

	// 7. Generate Data Exports (JSON, CSV, MD, HTML)
	var generatedFiles []string
	generatedFiles = append(generatedFiles, generatedCharts...)

	// summary.json
	summaryPath := filepath.Join(historyDir, "summary.json")
	summaryBytes, _ := json.MarshalIndent(summary, "", "  ")
	if err := os.WriteFile(summaryPath, summaryBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write summary.json: %w", err)
	}
	generatedFiles = append(generatedFiles, summaryPath)

	// raw-results.json
	rawPath := filepath.Join(historyDir, "raw-results.json")
	rawBytes, _ := json.MarshalIndent(runDetails, "", "  ")
	if err := os.WriteFile(rawPath, rawBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write raw-results.json: %w", err)
	}
	generatedFiles = append(generatedFiles, rawPath)

	// comparison.csv
	csvPath := filepath.Join(historyDir, "comparison.csv")
	csvBytes := generateComparisonCSV(runDetails)
	if err := os.WriteFile(csvPath, csvBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write comparison.csv: %w", err)
	}
	generatedFiles = append(generatedFiles, csvPath)

	// Git revision
	gitRev := getGitRevision()

	// Initial Manifest (to pass to markdown/html)
	manifest := ReportManifest{
		ManifestVersion: "1.0.0",
		ReportID:        summary.ReportID,
		CreatedAt:       summary.GeneratedAt,
		GitRevision:     gitRev,
		HostFingerprint: summary.Environment,
		StatisticalConfig: map[string]interface{}{
			"confidence_interval":         "Student's t 95%",
			"practical_threshold_percent": cfg.PracticalThresholdPercent,
			"central_tendency":            "Median",
		},
		WorkloadsCovered: summary.WorkloadsCovered,
		RunIDs:           summary.RunIDs,
	}

	// report.md
	mdPath := filepath.Join(historyDir, "report.md")
	mdContent := GenerateFullMarkdownReport(summary, &manifest, workloads, findings)
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write report.md: %w", err)
	}
	generatedFiles = append(generatedFiles, mdPath)

	// report.html
	htmlPath := filepath.Join(historyDir, "report.html")
	htmlContent := GenerateHTMLReport(summary, &manifest, workloads, findings)
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write report.html: %w", err)
	}
	generatedFiles = append(generatedFiles, htmlPath)

	// 8. Compute Checksums & Write Final manifest.json
	var checksums []FileChecksum
	for _, fpath := range generatedFiles {
		rel, _ := filepath.Rel(historyDir, fpath)
		data, err := os.ReadFile(fpath)
		if err == nil {
			hash := sha256.Sum256(data)
			checksums = append(checksums, FileChecksum{
				Path:      rel,
				SizeBytes: int64(len(data)),
				SHA256:    hex.EncodeToString(hash[:]),
			})
		}
	}
	manifest.Artifacts = checksums

	manifestPath := filepath.Join(historyDir, "manifest.json")
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(manifestPath, manifestBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write manifest.json: %w", err)
	}
	generatedFiles = append(generatedFiles, manifestPath)

	// 9. Sync History to latest/
	if err := copyDirectory(historyDir, latestDir); err != nil {
		return nil, fmt.Errorf("failed to copy artifacts to latest directory: %w", err)
	}

	// 10. Update README.md (if requested)
	readmeUpdated := false
	if cfg.UpdateReadme {
		if err := updateReadmeFile(cfg.ReadmePath, summary, workloads, findings); err != nil {
			return nil, fmt.Errorf("failed to update README.md: %w", err)
		}
		readmeUpdated = true
	}

	return &GenerationResult{
		ReportID:       summary.ReportID,
		OutputDir:      cfg.OutputDir,
		LatestDir:      latestDir,
		HistoryDir:     historyDir,
		Summary:        *summary,
		Manifest:       manifest,
		Findings:       findings,
		GeneratedFiles: generatedFiles,
		ReadmeUpdated:  readmeUpdated,
	}, nil
}

func collectRunDetails(ctx context.Context, db *database.DB, cfg ReportConfig) ([]database.RunDetail, error) {
	var runIDs []string

	if len(cfg.RunIDs) > 0 {
		runIDs = cfg.RunIDs
	} else {
		// List recent completed or failed runs
		runs, _, err := db.ListRuns(ctx, 50, 0, cfg.TrackFilter, cfg.WorkloadFilter, "")
		if err != nil {
			return nil, err
		}

		// Select the latest run per unique workload
		seenWorkloads := make(map[string]bool)
		for _, r := range runs {
			if !seenWorkloads[r.WorkloadID] {
				seenWorkloads[r.WorkloadID] = true
				runIDs = append(runIDs, r.ID)
			}
		}
	}

	if len(runIDs) == 0 {
		return nil, nil
	}

	var details []database.RunDetail
	for _, id := range runIDs {
		d, err := db.GetRunDetail(ctx, id, cfg.BaselineID)
		if err != nil {
			continue
		}
		if d != nil {
			details = append(details, *d)
		}
	}

	return details, nil
}

func aggregateWorkloads(runs []database.RunDetail, baselineID string) []WorkloadReportData {
	workloadMap := make(map[string]*WorkloadReportData)

	for _, r := range runs {
		w, ok := workloadMap[r.WorkloadID]
		if !ok {
			name := strings.ReplaceAll(r.WorkloadID, "_", " ")
			name = strings.Title(name)
			w = &WorkloadReportData{
				WorkloadID:    r.WorkloadID,
				WorkloadName:  name,
				Track:         r.Track,
				Parameters:    r.WorkloadParams,
				ExecutionMode: r.ExecutionMode,
				BaselineID:    baselineID,
				PrimaryRunID:  r.ID,
				CreatedAt:     r.CreatedAt,
			}
			workloadMap[r.WorkloadID] = w
		}

		// Baseline resolution: if workload is lowlevel, baseline might be "scalar"
		curBaseline := baselineID
		if r.Track == "lowlevel" {
			curBaseline = "scalar"
			w.BaselineID = "scalar"
		} else if r.Track == "gpu" {
			curBaseline = "cpu_reference"
			w.BaselineID = "cpu_reference"
		}

		for _, e := range r.Executions {
			view := ExecutionView{
				LanguageExecution: e,
				Speedup:           e.Speedup,
			}
			if e.PeakMemoryBytes > 0 && e.LanguageID != curBaseline {
				view.MemoryRatio = e.MemoryRatio
			}
			if len(e.Samples) > 1 {
				w.HasColdStartData = true
			}
			w.Executions = append(w.Executions, view)
		}
	}

	var result []WorkloadReportData
	for _, w := range workloadMap {
		result = append(result, *w)
	}

	// Sort workloads deterministically
	sort.Slice(result, func(i, j int) bool {
		return result[i].WorkloadID < result[j].WorkloadID
	})

	return result
}

func buildSummary(runs []database.RunDetail, workloads []WorkloadReportData, version string, sysInfo database.SystemInfo) *ReportSummary {
	now := time.Now().UTC()
	hashInput := fmt.Sprintf("%d_%s", now.UnixNano(), runs[0].ID)
	h := sha256.Sum256([]byte(hashInput))
	reportID := fmt.Sprintf("rep_%s_%s", now.Format("20060102"), hex.EncodeToString(h[:4]))

	summary := &ReportSummary{
		ReportID:         reportID,
		GeneratedAt:      now,
		GeneratorVersion: version,
		Environment:      sysInfo,
	}

	langSet := make(map[string]bool)
	trackSet := make(map[string]bool)
	workloadSet := make(map[string]bool)
	runSet := make(map[string]bool)

	for _, r := range runs {
		runSet[r.ID] = true
		trackSet[r.Track] = true
		workloadSet[r.WorkloadID] = true

		for _, e := range r.Executions {
			summary.TotalConfigurations++
			langSet[e.LanguageID] = true

			switch e.Status {
			case "COMPLETED":
				summary.SuccessfulExecutions++
			case "SKIPPED":
				summary.SkippedExecutions++
			case "FAILED":
				summary.FailedExecutions++
			}

			if e.ValidationPassed != nil && !*e.ValidationPassed {
				summary.ValidationFailures++
			}
		}
	}

	for k := range langSet {
		summary.LanguagesTested = append(summary.LanguagesTested, k)
	}
	sort.Strings(summary.LanguagesTested)

	for k := range trackSet {
		summary.TracksCovered = append(summary.TracksCovered, k)
	}
	sort.Strings(summary.TracksCovered)

	for k := range workloadSet {
		summary.WorkloadsCovered = append(summary.WorkloadsCovered, k)
	}
	sort.Strings(summary.WorkloadsCovered)

	for k := range runSet {
		summary.RunIDs = append(summary.RunIDs, k)
	}
	sort.Strings(summary.RunIDs)

	return summary
}

func generateCharts(chartsDir string, summary *ReportSummary, workloads []WorkloadReportData, sysInfo database.SystemInfo) ([]string, error) {
	var fileList []string

	// 1. overview.svg
	overviewSVG := RenderOverviewSVG(summary, sysInfo)
	pOverview := filepath.Join(chartsDir, "overview.svg")
	if err := os.WriteFile(pOverview, []byte(overviewSVG), 0644); err != nil {
		return nil, err
	}
	fileList = append(fileList, pOverview)

	// Primary workload for single-workload charts
	primary := workloads[0]

	// 2. runtime-comparison.svg
	runtimeSVG := RenderRuntimeComparisonSVG(primary)
	pRuntime := filepath.Join(chartsDir, "runtime-comparison.svg")
	if err := os.WriteFile(pRuntime, []byte(runtimeSVG), 0644); err != nil {
		return nil, err
	}
	fileList = append(fileList, pRuntime)

	// 3. throughput-comparison.svg
	thSVG := RenderThroughputComparisonSVG(primary)
	pTh := filepath.Join(chartsDir, "throughput-comparison.svg")
	if err := os.WriteFile(pTh, []byte(thSVG), 0644); err != nil {
		return nil, err
	}
	fileList = append(fileList, pTh)

	// 4. memory-comparison.svg
	memSVG := RenderMemoryComparisonSVG(primary)
	pMem := filepath.Join(chartsDir, "memory-comparison.svg")
	if err := os.WriteFile(pMem, []byte(memSVG), 0644); err != nil {
		return nil, err
	}
	fileList = append(fileList, pMem)

	// 5. variability.svg
	varSVG := RenderVariabilitySVG(primary)
	pVar := filepath.Join(chartsDir, "variability.svg")
	if err := os.WriteFile(pVar, []byte(varSVG), 0644); err != nil {
		return nil, err
	}
	fileList = append(fileList, pVar)

	// 6. scaling.svg
	scalingSVG := RenderScalingSVG(workloads)
	pScaling := filepath.Join(chartsDir, "scaling.svg")
	if err := os.WriteFile(pScaling, []byte(scalingSVG), 0644); err != nil {
		return nil, err
	}
	fileList = append(fileList, pScaling)

	// 7. startup-comparison.svg
	startupSVG := RenderStartupComparisonSVG(primary)
	pStartup := filepath.Join(chartsDir, "startup-comparison.svg")
	if err := os.WriteFile(pStartup, []byte(startupSVG), 0644); err != nil {
		return nil, err
	}
	fileList = append(fileList, pStartup)

	// 8. build-comparison.svg
	buildSVG := RenderBuildComparisonSVG(primary)
	pBuild := filepath.Join(chartsDir, "build-comparison.svg")
	if err := os.WriteFile(pBuild, []byte(buildSVG), 0644); err != nil {
		return nil, err
	}
	fileList = append(fileList, pBuild)

	// 9. workload-heatmap.svg
	heatmapSVG := RenderHeatmapSVG(workloads)
	pHeatmap := filepath.Join(chartsDir, "workload-heatmap.svg")
	if err := os.WriteFile(pHeatmap, []byte(heatmapSVG), 0644); err != nil {
		return nil, err
	}
	fileList = append(fileList, pHeatmap)

	// 10. Workload-specific charts
	for _, w := range workloads {
		wSVG := RenderWorkloadSpecificSVG(w)
		pW := filepath.Join(chartsDir, fmt.Sprintf("%s.svg", w.WorkloadID))
		if err := os.WriteFile(pW, []byte(wSVG), 0644); err != nil {
			return nil, err
		}
		fileList = append(fileList, pW)
	}

	return fileList, nil
}

func generateComparisonCSV(runs []database.RunDetail) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	_ = w.Write([]string{"# ExecLens Benchmark Consolidated Comparison Export"})
	_ = w.Write([]string{"Workload", "Track", "Language", "Status", "Median_ms", "Mean_ms", "CI95_Lower_ms", "CI95_Upper_ms", "StdDev_ms", "CV_percent", "Peak_Memory_MB", "Throughput_ops_sec", "Speedup_vs_Baseline", "Toolchain"})

	for _, r := range runs {
		for _, e := range r.Executions {
			medStr, meanStr, ciLStr, ciUStr, stdStr, cvStr, thStr, spStr := "N/A", "N/A", "N/A", "N/A", "N/A", "N/A", "N/A", "N/A"
			if e.Summary != nil {
				medStr = fmt.Sprintf("%.3f", e.Summary.MedianMs)
				meanStr = fmt.Sprintf("%.3f", e.Summary.MeanMs)
				ciLStr = fmt.Sprintf("%.3f", e.Summary.CI95LowerMs)
				ciUStr = fmt.Sprintf("%.3f", e.Summary.CI95UpperMs)
				stdStr = fmt.Sprintf("%.3f", e.Summary.StdDevMs)
				cvStr = fmt.Sprintf("%.2f", e.Summary.CVPercent)
				thStr = fmt.Sprintf("%.1f", e.Summary.ThroughputOpsPerSec)
				spStr = fmt.Sprintf("%.2f", e.Speedup)
			}
			_ = w.Write([]string{
				r.WorkloadID,
				r.Track,
				e.LanguageID,
				string(e.Status),
				medStr, meanStr, ciLStr, ciUStr, stdStr, cvStr,
				fmt.Sprintf("%.2f", e.PeakMemoryMB),
				thStr,
				spStr,
				e.ToolchainVersion,
			})
		}
	}

	w.Flush()
	return buf.Bytes()
}

func updateReadmeFile(readmePath string, summary *ReportSummary, workloads []WorkloadReportData, findings []ReportFinding) error {
	existingBytes, err := os.ReadFile(readmePath)
	if err != nil {
		return fmt.Errorf("failed to read README file: %w", err)
	}

	section := GenerateReadmeSection(summary, workloads, findings)
	updated, err := UpdateReadmeContent(string(existingBytes), section)
	if err != nil {
		return err
	}

	return os.WriteFile(readmePath, []byte(updated), 0644)
}

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}

func getGitRevision() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return "git-head"
}
