package reports

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/execlens/execlens/internal/database"
)

// ExportRunJSON serializes the run and all executions with raw samples into clean JSON.
func ExportRunJSON(detail *database.RunDetail) ([]byte, error) {
	return json.MarshalIndent(detail, "", "  ")
}

// ExportRunCSV formats the run executions and samples as CSV.
func ExportRunCSV(detail *database.RunDetail) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write metadata header comments
	_ = writer.Write([]string{"# ExecLens Benchmark Run Export"})
	_ = writer.Write([]string{"# Run ID", detail.ID})
	_ = writer.Write([]string{"# Workload", detail.WorkloadID})
	_ = writer.Write([]string{"# OS", detail.SystemInfo.OS, "Arch", detail.SystemInfo.Arch, "CPU", detail.SystemInfo.CPUModel})
	_ = writer.Write([]string{"# Date UTC", detail.CreatedAt.Format(time.RFC3339)})
	_ = writer.Write([]string{}) // empty line

	// Write Summary Table
	_ = writer.Write([]string{"--- Summary Table ---"})
	_ = writer.Write([]string{
		"Language", "Status", "Toolchain", "Median (ms)", "Mean (ms)", "Min (ms)", "Max (ms)",
		"StdDev (ms)", "CV %", "95% CI Lower (ms)", "95% CI Upper (ms)", "Peak Memory (MB)",
		"Throughput Ops/sec", "Speedup Relative to Baseline",
	})

	for _, e := range detail.Executions {
		if e.Summary != nil {
			_ = writer.Write([]string{
				e.LanguageID,
				string(e.Status),
				e.ToolchainVersion,
				fmt.Sprintf("%.3f", e.Summary.MedianMs),
				fmt.Sprintf("%.3f", e.Summary.MeanMs),
				fmt.Sprintf("%.3f", e.Summary.MinMs),
				fmt.Sprintf("%.3f", e.Summary.MaxMs),
				fmt.Sprintf("%.3f", e.Summary.StdDevMs),
				fmt.Sprintf("%.2f%%", e.Summary.CVPercent),
				fmt.Sprintf("%.3f", e.Summary.CI95LowerMs),
				fmt.Sprintf("%.3f", e.Summary.CI95UpperMs),
				fmt.Sprintf("%.2f", e.PeakMemoryMB),
				fmt.Sprintf("%.1f", e.Summary.ThroughputOpsPerSec),
				fmt.Sprintf("%.2fx", e.Speedup),
			})
		} else {
			_ = writer.Write([]string{
				e.LanguageID,
				string(e.Status),
				e.ToolchainVersion,
				"N/A", "N/A", "N/A", "N/A", "N/A", "N/A", "N/A", "N/A",
				fmt.Sprintf("%.2f", e.PeakMemoryMB),
				"N/A", "N/A",
			})
		}
	}

	_ = writer.Write([]string{})
	_ = writer.Write([]string{"--- Raw Samples ---"})
	_ = writer.Write([]string{"Language", "Iteration", "Is Warmup", "Duration (ms)", "Memory (MB)", "Exit Code"})

	for _, e := range detail.Executions {
		for _, s := range e.Samples {
			_ = writer.Write([]string{
				e.LanguageID,
				fmt.Sprintf("%d", s.Iteration),
				fmt.Sprintf("%t", s.IsWarmup),
				fmt.Sprintf("%.4f", s.DurationMs),
				fmt.Sprintf("%.2f", float64(s.MemoryBytes)/(1024*1024)),
				fmt.Sprintf("%d", s.ExitCode),
			})
		}
	}

	writer.Flush()
	return buf.Bytes(), nil
}

// ComparisonWarning flags environmental or parameter divergence between compared runs.
type ComparisonWarning struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// ComparisonReport provides side-by-side run analysis.
type ComparisonReport struct {
	Runs     []database.RunDetail `json:"runs"`
	Warnings []ComparisonWarning  `json:"warnings"`
}

// CompareRuns evaluates two or more runs for comparability and produces side-by-side results.
func CompareRuns(runs []database.RunDetail) ComparisonReport {
	report := ComparisonReport{
		Runs:     runs,
		Warnings: []ComparisonWarning{},
	}

	if len(runs) < 2 {
		return report
	}

	first := runs[0]
	for i := 1; i < len(runs); i++ {
		curr := runs[i]

		// Check Workload compatibility
		if curr.WorkloadID != first.WorkloadID {
			report.Warnings = append(report.Warnings, ComparisonWarning{
				Type:    "WORKLOAD_MISMATCH",
				Message: fmt.Sprintf("Run '%s' tests '%s' while Run '%s' tests '%s'. Comparing different workloads is invalid.", first.ID, first.WorkloadID, curr.ID, curr.WorkloadID),
			})
		}

		// Check Workload Parameters
		firstParams, _ := json.Marshal(first.WorkloadParams)
		currParams, _ := json.Marshal(curr.WorkloadParams)
		if string(firstParams) != string(currParams) {
			report.Warnings = append(report.Warnings, ComparisonWarning{
				Type:    "PARAMETER_MISMATCH",
				Message: fmt.Sprintf("Run '%s' and '%s' have different workload parameters (%s vs %s).", first.ID, curr.ID, string(firstParams), string(currParams)),
			})
		}

		// Check Execution Mode
		if curr.ExecutionMode != first.ExecutionMode {
			report.Warnings = append(report.Warnings, ComparisonWarning{
				Type:    "MODE_MISMATCH",
				Message: fmt.Sprintf("Run '%s' used '%s' mode, while Run '%s' used '%s'.", first.ID, first.ExecutionMode, curr.ID, curr.ExecutionMode),
			})
		}

		// Check Environment
		if curr.SystemInfo.CPUModel != first.SystemInfo.CPUModel || curr.SystemInfo.OS != first.SystemInfo.OS {
			report.Warnings = append(report.Warnings, ComparisonWarning{
				Type:    "ENVIRONMENT_MISMATCH",
				Message: fmt.Sprintf("Hardware/OS divergence between runs: '%s' (%s %s) vs '%s' (%s %s).", first.ID, first.SystemInfo.OS, first.SystemInfo.CPUModel, curr.ID, curr.SystemInfo.OS, curr.SystemInfo.CPUModel),
			})
		}
	}

	return report
}

// GenerateMarkdownReport produces a detailed text report for a run.
func GenerateMarkdownReport(detail *database.RunDetail) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# ExecLens Benchmark Report: %s\n\n", detail.WorkloadID))
	sb.WriteString(fmt.Sprintf("- **Run ID**: `%s`\n", detail.ID))
	sb.WriteString(fmt.Sprintf("- **Date (UTC)**: %s\n", detail.CreatedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("- **Status**: %s\n", detail.Status))
	sb.WriteString(fmt.Sprintf("- **Iterations**: %d timed, %d warmup\n", detail.Iterations, detail.WarmupIterations))
	sb.WriteString(fmt.Sprintf("- **Execution Mode**: %s\n", detail.ExecutionMode))
	sb.WriteString(fmt.Sprintf("- **Host Environment**: %s (%s), %d vCPUs, CPU: %s\n\n",
		detail.SystemInfo.OS, detail.SystemInfo.Arch, detail.SystemInfo.LogicalCPUs, detail.SystemInfo.CPUModel))

	paramsJSON, _ := json.Marshal(detail.WorkloadParams)
	sb.WriteString(fmt.Sprintf("**Workload Parameters**: `%s`\n\n", string(paramsJSON)))

	sb.WriteString("## Results Summary\n\n")
	sb.WriteString("| Language | Status | Median Time | Mean Time | 95% CI | StdDev (CV%) | Peak Memory | Throughput | Speedup |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")

	for _, e := range detail.Executions {
		if e.Summary != nil {
			sb.WriteString(fmt.Sprintf("| **%s** | %s | %.2f ms | %.2f ms | [%.2f, %.2f] ms | ±%.2f ms (%.1f%%) | %.1f MB | %.0f ops/s | **%.2fx** |\n",
				e.LanguageID, e.Status, e.Summary.MedianMs, e.Summary.MeanMs, e.Summary.CI95LowerMs, e.Summary.CI95UpperMs,
				e.Summary.StdDevMs, e.Summary.CVPercent, e.PeakMemoryMB, e.Summary.ThroughputOpsPerSec, e.Speedup))
		} else {
			sb.WriteString(fmt.Sprintf("| **%s** | %s | N/A | N/A | N/A | N/A | %.1f MB | N/A | N/A |\n",
				e.LanguageID, e.Status, e.PeakMemoryMB))
		}
	}

	sb.WriteString("\n## Statistical and Fairness Notes\n\n")
	for _, e := range detail.Executions {
		if e.SignificanceNote != "" {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", e.LanguageID, e.SignificanceNote))
		}
		if e.ErrorMessage != "" {
			sb.WriteString(fmt.Sprintf("- **%s (Error)**: %s\n", e.LanguageID, e.ErrorMessage))
		}
	}

	sb.WriteString("\n---\n*Generated by ExecLens — Measure fairly. Compare deeply. Understand why.*\n")
	return sb.String()
}
