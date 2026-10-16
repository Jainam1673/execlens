package reports

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GenerateFullMarkdownReport produces the comprehensive `report.md` artifact.
func GenerateFullMarkdownReport(summary *ReportSummary, manifest *ReportManifest, workloads []WorkloadReportData, findings []ReportFinding) string {
	var sb strings.Builder

	sb.WriteString("# ExecLens Performance Engineering Report\n\n")
	sb.WriteString("> **Measure fairly. Compare deeply. Understand why.**\n\n")

	sb.WriteString(fmt.Sprintf("- **Report ID**: `%s`\n", summary.ReportID))
	sb.WriteString(fmt.Sprintf("- **Generated At**: %s\n", summary.GeneratedAt.UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("- **Generator Version**: `v%s`\n", summary.GeneratorVersion))
	sb.WriteString(fmt.Sprintf("- **Git Revision**: `%s`\n", manifest.GitRevision))
	sb.WriteString(fmt.Sprintf("- **Host Environment**: %s (%s, %d vCPUs) | CPU: `%s`\n",
		summary.Environment.OS, summary.Environment.Arch, summary.Environment.LogicalCPUs, summary.Environment.CPUModel))
	sb.WriteString(fmt.Sprintf("- **Total Configurations Tested**: %d (%d completed, %d skipped/unavailable, %d validation failures)\n\n",
		summary.TotalConfigurations, summary.SuccessfulExecutions, summary.SkippedExecutions, summary.ValidationFailures))

	sb.WriteString("---\n\n")

	// Executive Visual Summary
	sb.WriteString("## 1. Executive Visual Summary\n\n")
	sb.WriteString("![Benchmark Overview](charts/overview.svg)\n\n")

	sb.WriteString("### Cross-Workload Performance Heatmap\n\n")
	sb.WriteString("The heatmap summarizes relative speedup normalized against the baseline across all tested workloads. Green cells denote accelerated performance; red cells denote slower execution; grey/dark cells denote statistical parity or unavailable toolchains.\n\n")
	sb.WriteString("![Cross-Workload Heatmap](charts/workload-heatmap.svg)\n\n")

	sb.WriteString("---\n\n")

	// Deterministic Findings Section
	sb.WriteString("## 2. Key Deterministic Performance Findings\n\n")
	sb.WriteString("The following findings were derived programmatically by the ExecLens Deterministic Findings Engine from verified sample distributions and Student's t 95% Confidence Intervals:\n\n")

	for _, f := range findings {
		statusBadge := "📊 **MEASURED**"
		if f.Status == "STATISTICALLY_SIGNIFICANT" {
			statusBadge = "✅ **STATISTICALLY SIGNIFICANT**"
		} else if f.Status == "INCONCLUSIVE" {
			statusBadge = "⚠️ **INCONCLUSIVE (HIGH VARIANCE)**"
		} else if f.Category == FindingDiagnosticNote {
			statusBadge = "ℹ️ **DIAGNOSTIC NOTE**"
		}

		sb.WriteString(fmt.Sprintf("### %s: %s\n\n", f.ID, f.Title))
		sb.WriteString(fmt.Sprintf("- **Status**: %s\n", statusBadge))
		sb.WriteString(fmt.Sprintf("- **Workload & Scope**: `%s` (%s)\n", f.WorkloadID, f.Scope))
		sb.WriteString(fmt.Sprintf("- **Comparison**: `%s` vs `%s` baseline\n", f.CandidateID, f.BaselineID))
		sb.WriteString(fmt.Sprintf("- **Effect Size**: **%s** (%s)\n", f.EffectSize, f.Uncertainty))
		sb.WriteString(fmt.Sprintf("- **Explanation**: %s\n", f.Explanation))
		sb.WriteString(fmt.Sprintf("- **Caveat / Limitations**: *%s*\n\n", f.Limitations))
	}

	sb.WriteString("---\n\n")

	// Workload-by-Workload Section
	sb.WriteString("## 3. Workload-by-Workload Detailed Results\n\n")

	for _, w := range workloads {
		sb.WriteString(fmt.Sprintf("### Workload: %s (`%s`)\n\n", w.WorkloadName, w.WorkloadID))
		sb.WriteString(fmt.Sprintf("- **Track**: `%s` | **Execution Mode**: `%s` | **Baseline**: `%s`\n", w.Track, w.ExecutionMode, w.BaselineID))

		paramsBytes, _ := json.Marshal(w.Parameters)
		sb.WriteString(fmt.Sprintf("- **Workload Parameters**: `%s`\n\n", string(paramsBytes)))

		// Embed Workload Chart
		chartPath := fmt.Sprintf("charts/%s.svg", w.WorkloadID)
		sb.WriteString(fmt.Sprintf("![%s Chart](%s)\n\n", w.WorkloadName, chartPath))

		// Summary Table
		sb.WriteString("| Implementation | Status | Median (ms) | Mean (ms) | 95% CI (ms) | StdDev (CV%) | Peak RSS | Throughput | Relative Speedup |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")

		for _, e := range w.Executions {
			if e.Summary != nil && e.Status == "COMPLETED" {
				throughputStr := "N/A"
				if e.Summary.ThroughputOpsPerSec > 0 {
					throughputStr = fmt.Sprintf("%.0f ops/s", e.Summary.ThroughputOpsPerSec)
				} else if e.Summary.ThroughputMBPerSec > 0 {
					throughputStr = fmt.Sprintf("%.1f MB/s", e.Summary.ThroughputMBPerSec)
				}

				speedupStr := "1.00x (Baseline)"
				if e.LanguageID != w.BaselineID && e.Speedup > 0 {
					speedupStr = fmt.Sprintf("**%.2fx**", e.Speedup)
				}

				sb.WriteString(fmt.Sprintf("| **%s** | %s | %.2f | %.2f | [%.2f, %.2f] | ±%.2f (%.1f%%) | %.1f MB | %s | %s |\n",
					e.LanguageID, e.Status, e.Summary.MedianMs, e.Summary.MeanMs,
					e.Summary.CI95LowerMs, e.Summary.CI95UpperMs,
					e.Summary.StdDevMs, e.Summary.CVPercent,
					e.PeakMemoryMB, throughputStr, speedupStr,
				))
			} else {
				sb.WriteString(fmt.Sprintf("| **%s** | %s | N/A | N/A | N/A | N/A | %.1f MB | N/A | N/A |\n",
					e.LanguageID, e.Status, e.PeakMemoryMB))
			}
		}

		// Collapsible Notes & Diagnostic Details
		sb.WriteString("\n<details>\n<summary>Inspect Execution Notes, Toolchain Flags & Compiler Remarks</summary>\n\n")
		for _, e := range w.Executions {
			sb.WriteString(fmt.Sprintf("#### %s (`%s`)\n", e.LanguageID, e.ToolchainVersion))
			if e.CompilerFlags != "" {
				sb.WriteString(fmt.Sprintf("- **Compiler Flags**: `%s` (Opt: `%s`)\n", e.CompilerFlags, e.OptLevel))
			}
			if e.BuildDurationMs > 0 {
				sb.WriteString(fmt.Sprintf("- **Build Duration**: %.1f ms | **Binary Size**: %d bytes\n", e.BuildDurationMs, e.ExecutableSizeBytes))
			}
			if e.KernelDurationNs > 0 {
				sb.WriteString(fmt.Sprintf("- **GPU Kernel Timing**: %.2f ms | **H2D**: %.2f ms | **D2H**: %.2f ms | **Bandwidth**: %.1f GB/s\n",
					float64(e.KernelDurationNs)/1e6, float64(e.TransferH2DNs)/1e6, float64(e.TransferD2HNs)/1e6, e.EffectiveBandwidthGBps))
			}
			if e.ErrorMessage != "" {
				sb.WriteString(fmt.Sprintf("- **Diagnostic Message**: %s\n", e.ErrorMessage))
			}
			if e.CompilerRemarks != "" {
				sb.WriteString(fmt.Sprintf("- **Compiler Remarks**:\n```\n%s\n```\n", e.CompilerRemarks))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("</details>\n\n---\n\n")
	}

	// Methodology & Reproducibility Section
	sb.WriteString("## 4. Methodology, Reproducibility & Provenance\n\n")
	sb.WriteString("ExecLens adheres to strict systems benchmarking methodology:\n")
	sb.WriteString("1. **Separation of Build & Runtime**: Compilation duration is tracked independently and never conflated with steady-state execution.\n")
	sb.WriteString("2. **Deterministic Correctness**: Outputs are validated against expected values before accepting timing samples.\n")
	sb.WriteString("3. **Kernel RSS Memory Tracking**: Memory reflects peak physical Resident Set Size (RSS) via Linux kernel `getrusage`.\n")
	sb.WriteString("4. **Statistical Rigor**: Reports Median, Student's t 95% Confidence Intervals, and sample Coefficient of Variation (CV%).\n\n")

	sb.WriteString("### Reproduction Command\n\n")
	sb.WriteString("To reproduce this exact benchmark report and all generated artifacts on your machine:\n\n")
	sb.WriteString("```bash\n# Execute benchmarks and generate report\nmake benchmark-report\n```\n\n")

	sb.WriteString("### Manifest & Raw Data Links\n\n")
	sb.WriteString("- [Reproducibility Manifest](manifest.json)\n")
	sb.WriteString("- [Aggregated Summary Dataset](summary.json)\n")
	sb.WriteString("- [Complete Raw Observations](raw-results.json)\n")
	sb.WriteString("- [Tabular Comparison CSV](comparison.csv)\n\n")

	sb.WriteString("---\n*Generated by ExecLens Automated Benchmark Reporting System — Measure fairly. Compare deeply. Understand why.*\n")
	return sb.String()
}

// GenerateReadmeSection produces the formatted Markdown text to insert between markers in README.md.
func GenerateReadmeSection(summary *ReportSummary, workloads []WorkloadReportData, findings []ReportFinding) string {
	var sb strings.Builder

	sb.WriteString("<!-- EXECLENS_REPORT_START -->\n")
	sb.WriteString("## Latest Verified Benchmark Report\n\n")

	dateStr := summary.GeneratedAt.UTC().Format("2006-01-02 15:04 UTC")
	sb.WriteString(fmt.Sprintf("> **Report ID**: `%s` | **Date**: %s | **Host**: `%s` (%s, %d vCPUs) | **Generator**: `v%s`\n\n",
		summary.ReportID, dateStr, summary.Environment.CPUModel, summary.Environment.OS, summary.Environment.LogicalCPUs, summary.GeneratorVersion))

	sb.WriteString(fmt.Sprintf("ExecLens tested **%d configurations** across **%d workloads** and **%d runtimes** with %d successful executions (%d skipped/unavailable, %d inconclusive due to environmental variance).\n\n",
		summary.TotalConfigurations, len(summary.WorkloadsCovered), len(summary.LanguagesTested), summary.SuccessfulExecutions, summary.SkippedExecutions, summary.InconclusiveCount))

	// Executive Summary Charts
	sb.WriteString("### Executive Visual Summary\n\n")
	sb.WriteString("![Benchmark Overview](reports/latest/charts/overview.svg)\n\n")
	sb.WriteString("![Performance Heatmap](reports/latest/charts/workload-heatmap.svg)\n\n")

	// Top Findings Summary
	sb.WriteString("### Key Automated Performance Findings\n\n")
	count := 0
	for _, f := range findings {
		if f.Category == FindingTopPerformer || f.Category == FindingSpeedup || f.Category == FindingMemoryEfficiency {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", f.Title, f.Explanation))
			count++
			if count >= 6 {
				break
			}
		}
	}
	sb.WriteString("\n")

	// Workload-by-Workload Summary with collapsible details
	sb.WriteString("### Workload-by-Workload Results\n\n")

	for _, w := range workloads {
		sb.WriteString(fmt.Sprintf("<details>\n<summary><strong>%s (%s) — View Benchmark Findings &amp; Visuals</strong></summary>\n\n", w.WorkloadName, w.WorkloadID))

		chartRelPath := fmt.Sprintf("reports/latest/charts/%s.svg", w.WorkloadID)
		sb.WriteString(fmt.Sprintf("![%s Chart](%s)\n\n", w.WorkloadName, chartRelPath))

		sb.WriteString("| Implementation | Status | Median Time | 95% Confidence Interval | Peak Memory | Throughput | Relative Speedup |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")

		for _, e := range w.Executions {
			if e.Summary != nil && e.Status == "COMPLETED" {
				speedupStr := "1.00x (Baseline)"
				if e.LanguageID != w.BaselineID && e.Speedup > 0 {
					speedupStr = fmt.Sprintf("**%.2fx**", e.Speedup)
				}

				throughputStr := "N/A"
				if e.Summary.ThroughputOpsPerSec > 0 {
					throughputStr = fmt.Sprintf("%.0f ops/s", e.Summary.ThroughputOpsPerSec)
				} else if e.Summary.ThroughputMBPerSec > 0 {
					throughputStr = fmt.Sprintf("%.1f MB/s", e.Summary.ThroughputMBPerSec)
				}

				sb.WriteString(fmt.Sprintf("| **%s** | %s | %.2f ms | [%.2f, %.2f] ms | %.1f MB | %s | %s |\n",
					e.LanguageID, e.Status, e.Summary.MedianMs, e.Summary.CI95LowerMs, e.Summary.CI95UpperMs,
					e.PeakMemoryMB, throughputStr, speedupStr))
			} else {
				sb.WriteString(fmt.Sprintf("| **%s** | %s | N/A | N/A | %.1f MB | N/A | N/A |\n",
					e.LanguageID, e.Status, e.PeakMemoryMB))
			}
		}

		sb.WriteString("\n</details>\n\n")
	}

	// Provenance & Full Report Links
	sb.WriteString("### Complete Reports & Reproducibility Links\n\n")
	sb.WriteString("- 📄 **Full Markdown Report**: [reports/latest/report.md](reports/latest/report.md)\n")
	sb.WriteString("- 🌐 **Interactive HTML Report**: [reports/latest/report.html](reports/latest/report.html)\n")
	sb.WriteString("- 📊 **Summary Metrics**: [reports/latest/summary.json](reports/latest/summary.json)\n")
	sb.WriteString("- 🔬 **Raw Observations**: [reports/latest/raw-results.json](reports/latest/raw-results.json)\n")
	sb.WriteString("- 📑 **Comparison CSV**: [reports/latest/comparison.csv](reports/latest/comparison.csv)\n")
	sb.WriteString("- 🛡️ **Reproducibility Manifest**: [reports/latest/manifest.json](reports/latest/manifest.json)\n\n")

	sb.WriteString("```bash\n# Reproduce this report from existing benchmark data\nmake report\n```\n\n")
	sb.WriteString("<!-- EXECLENS_REPORT_END -->")

	return sb.String()
}

// UpdateReadmeContent replaces content strictly between the markers, or inserts them cleanly.
func UpdateReadmeContent(existingContent, newReportSection string) (string, error) {
	const startMarker = "<!-- EXECLENS_REPORT_START -->"
	const endMarker = "<!-- EXECLENS_REPORT_END -->"

	startIdx := strings.Index(existingContent, startMarker)
	endIdx := strings.Index(existingContent, endMarker)

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		// Existing markers found: replace only within bounds
		before := existingContent[:startIdx]
		after := existingContent[endIdx+len(endMarker):]
		return before + newReportSection + after, nil
	}

	// Markers do not exist: find designated insertion point (before Prerequisites or at end of Key Features)
	prereqIdx := strings.Index(existingContent, "## Prerequisites")
	if prereqIdx != -1 {
		before := existingContent[:prereqIdx]
		after := existingContent[prereqIdx:]
		return before + newReportSection + "\n\n---\n\n" + after, nil
	}

	// Fallback: append
	return existingContent + "\n\n---\n\n" + newReportSection + "\n", nil
}
