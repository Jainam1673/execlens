package reports

import (
	"fmt"
	"math"
	"sort"

	"github.com/execlens/execlens/internal/metrics"
)

// GenerateFindings analyzes completed workload benchmark data and deterministically generates
// structured findings according to defined statistical and engineering criteria.
func GenerateFindings(workloads []WorkloadReportData, practicalThresholdPercent float64) []ReportFinding {
	var findings []ReportFinding
	findingIndex := 1

	for _, w := range workloads {
		// Filter valid executions with calculated summaries
		var validExecs []ExecutionView
		var skippedOrFailed []ExecutionView

		for _, e := range w.Executions {
			if e.Status == "COMPLETED" && e.Summary != nil && e.Summary.Count > 0 {
				validExecs = append(validExecs, e)
			} else {
				skippedOrFailed = append(skippedOrFailed, e)
			}
		}

		// 1. Top Performer Finding (if at least 1 valid execution exists)
		if len(validExecs) > 0 {
			// Sort by median duration ascending
			sort.Slice(validExecs, func(i, j int) bool {
				return validExecs[i].Summary.MedianMs < validExecs[j].Summary.MedianMs
			})

			best := validExecs[0]
			findingID := fmt.Sprintf("find_%03d", findingIndex)
			findingIndex++

			title := fmt.Sprintf("%s Lowest Observed Median Runtime: %s", w.WorkloadName, best.LanguageID)
			explanation := fmt.Sprintf(
				"%s recorded the lowest observed median execution time of %.2f ms across %d iterations.",
				best.LanguageID, best.Summary.MedianMs, best.Summary.Count,
			)

			f := ReportFinding{
				ID:                 findingID,
				Title:              title,
				Category:           FindingTopPerformer,
				WorkloadID:         w.WorkloadID,
				Track:              w.Track,
				Scope:              fmt.Sprintf("%s (%s mode)", w.WorkloadName, w.ExecutionMode),
				BaselineID:         w.BaselineID,
				CandidateID:        best.LanguageID,
				MetricName:         "Median Execution Time",
				MetricUnit:         "ms",
				BaselineValue:      0, // determined if baseline exists below
				CandidateValue:     best.Summary.MedianMs,
				EffectSize:         fmt.Sprintf("%.2f ms", best.Summary.MedianMs),
				SampleCount:        best.Summary.Count,
				Uncertainty:        fmt.Sprintf("95%% CI [%.2f, %.2f] ms, CV %.1f%%", best.Summary.CI95LowerMs, best.Summary.CI95UpperMs, best.Summary.CVPercent),
				Status:             metrics.AssessmentStatisticallySig,
				Explanation:        explanation,
				Limitations:        "Applies strictly to this specific workload, input configuration, and host environment.",
				ChartRelativePath:  fmt.Sprintf("charts/%s.svg", w.WorkloadID),
				RawDataRelativePath: "raw-results.json",
			}
			findings = append(findings, f)
		}

		// 2. Relative Speedup & Statistical Significance (versus baseline)
		var baselineExec *ExecutionView
		for i := range validExecs {
			if validExecs[i].LanguageID == w.BaselineID {
				baselineExec = &validExecs[i]
				break
			}
		}

		if baselineExec != nil && baselineExec.Summary != nil {
			for _, cand := range validExecs {
				if cand.LanguageID == w.BaselineID {
					continue
				}

				baseNs := baselineExec.Summary.MedianNs
				if baseNs <= 0 {
					baseNs = baselineExec.Summary.MedianMs * 1e6
				}
				candNs := cand.Summary.MedianNs
				if candNs <= 0 {
					candNs = cand.Summary.MedianMs * 1e6
				}
				speedup := metrics.CalculateSpeedup(baseNs, candNs)
				status, assessNote := metrics.AssessDifference(*baselineExec.Summary, *cand.Summary, practicalThresholdPercent)

				findingID := fmt.Sprintf("find_%03d", findingIndex)
				findingIndex++

				var title string
				var category FindingCategory

				if status == metrics.AssessmentInconclusive {
					category = FindingInconclusive
					title = fmt.Sprintf("%s: Comparison Inconclusive for %s vs %s", w.WorkloadName, cand.LanguageID, baselineExec.LanguageID)
				} else if speedup >= 1.0 {
					category = FindingSpeedup
					title = fmt.Sprintf("%s: %s Observed %.2fx Faster than %s", w.WorkloadName, cand.LanguageID, speedup, baselineExec.LanguageID)
				} else {
					category = FindingSpeedup
					title = fmt.Sprintf("%s: %s Observed %.2fx Slower than %s", w.WorkloadName, cand.LanguageID, 1.0/speedup, baselineExec.LanguageID)
				}

				explanation := fmt.Sprintf(
					"%s achieved median %.2f ms vs %s baseline %.2f ms (%.2fx relative speedup). %s",
					cand.LanguageID, cand.Summary.MedianMs, baselineExec.LanguageID, baselineExec.Summary.MedianMs, speedup, assessNote,
				)

				f := ReportFinding{
					ID:                 findingID,
					Title:              title,
					Category:           category,
					WorkloadID:         w.WorkloadID,
					Track:              w.Track,
					Scope:              fmt.Sprintf("Comparison against %s baseline", baselineExec.LanguageID),
					BaselineID:         baselineExec.LanguageID,
					CandidateID:        cand.LanguageID,
					MetricName:         "Relative Speedup",
					MetricUnit:         "x",
					BaselineValue:      baselineExec.Summary.MedianMs,
					CandidateValue:     cand.Summary.MedianMs,
					EffectSize:         fmt.Sprintf("%.2fx", speedup),
					SampleCount:        cand.Summary.Count,
					Uncertainty:        fmt.Sprintf("Candidate 95%% CI [%.2f, %.2f] ms vs Baseline [%.2f, %.2f] ms", cand.Summary.CI95LowerMs, cand.Summary.CI95UpperMs, baselineExec.Summary.CI95LowerMs, baselineExec.Summary.CI95UpperMs),
					Status:             status,
					Explanation:        explanation,
					Limitations:        "Speedup is non-transferable across differing algorithmic patterns, I/O profiles, or unverified hardware configurations.",
					ChartRelativePath:  "charts/runtime-comparison.svg",
					RawDataRelativePath: "comparison.csv",
				}
				findings = append(findings, f)
			}
		}

		// 3. Memory Footprint Efficiency
		if len(validExecs) > 1 {
			// Find lowest peak memory
			sort.Slice(validExecs, func(i, j int) bool {
				return validExecs[i].PeakMemoryMB < validExecs[j].PeakMemoryMB
			})

			lowestMem := validExecs[0]
			highestMem := validExecs[len(validExecs)-1]

			if lowestMem.PeakMemoryMB > 0 && highestMem.PeakMemoryMB > lowestMem.PeakMemoryMB {
				findingID := fmt.Sprintf("find_%03d", findingIndex)
				findingIndex++

				memRatio := highestMem.PeakMemoryMB / math.Max(0.1, lowestMem.PeakMemoryMB)
				title := fmt.Sprintf("%s Memory Efficiency: %s Used %.1f MB Peak RSS", w.WorkloadName, lowestMem.LanguageID, lowestMem.PeakMemoryMB)
				explanation := fmt.Sprintf(
					"%s demonstrated the lowest physical memory footprint (%.1f MB peak RSS) compared to %s (%.1f MB peak RSS, %.1fx difference).",
					lowestMem.LanguageID, lowestMem.PeakMemoryMB, highestMem.LanguageID, highestMem.PeakMemoryMB, memRatio,
				)

				f := ReportFinding{
					ID:                 findingID,
					Title:              title,
					Category:           FindingMemoryEfficiency,
					WorkloadID:         w.WorkloadID,
					Track:              w.Track,
					Scope:              "Process Lifetime Peak Resident Set Size (MaxRSS)",
					BaselineID:         highestMem.LanguageID,
					CandidateID:        lowestMem.LanguageID,
					MetricName:         "Peak RSS Memory",
					MetricUnit:         "MB",
					BaselineValue:      highestMem.PeakMemoryMB,
					CandidateValue:     lowestMem.PeakMemoryMB,
					EffectSize:         fmt.Sprintf("%.1f MB (%.1fx less)", lowestMem.PeakMemoryMB, memRatio),
					SampleCount:        lowestMem.Summary.Count,
					Uncertainty:        "Captured directly from OS kernel getrusage.",
					Status:             metrics.AssessmentPractical,
					Explanation:        explanation,
					Limitations:        "Captures peak physical RSS of child process; does not isolate sub-allocator arena fragmentation from active working sets.",
					ChartRelativePath:  "charts/memory-comparison.svg",
					RawDataRelativePath: "raw-results.json",
				}
				findings = append(findings, f)
			}
		}

		// 4. Sample Variability & Stability Warnings
		for _, e := range validExecs {
			if e.Summary.CVPercent > 15.0 {
				findingID := fmt.Sprintf("find_%03d", findingIndex)
				findingIndex++

				title := fmt.Sprintf("%s Stability Notice: High Measurement Variance in %s (CV %.1f%%)", w.WorkloadName, e.LanguageID, e.Summary.CVPercent)
				explanation := fmt.Sprintf(
					"%s showed high dispersion across iterations (StdDev ±%.2f ms, CV %.1f%% exceeding the 15%% threshold).",
					e.LanguageID, e.Summary.StdDevMs, e.Summary.CVPercent,
				)

				f := ReportFinding{
					ID:                 findingID,
					Title:              title,
					Category:           FindingVariability,
					WorkloadID:         w.WorkloadID,
					Track:              w.Track,
					Scope:              "Inter-sample coefficient of variation",
					BaselineID:         w.BaselineID,
					CandidateID:        e.LanguageID,
					MetricName:         "Coefficient of Variation",
					MetricUnit:         "%",
					BaselineValue:      15.0,
					CandidateValue:     e.Summary.CVPercent,
					EffectSize:         fmt.Sprintf("%.1f%%", e.Summary.CVPercent),
					SampleCount:        e.Summary.Count,
					Uncertainty:        fmt.Sprintf("StdDev ±%.2f ms across %d samples", e.Summary.StdDevMs, e.Summary.Count),
					Status:             metrics.AssessmentInconclusive,
					Explanation:        explanation,
					Limitations:        "High variance indicates JIT compilation de-optimization, garbage collection pauses, or background OS core scheduling preemption.",
					ChartRelativePath:  "charts/variability.svg",
					RawDataRelativePath: "raw-results.json",
				}
				findings = append(findings, f)
			}
		}

		// 5. GPU Accelerator Specific Telemetry & Crossover Findings
		for _, e := range validExecs {
			if e.KernelDurationNs > 0 && e.TransferH2DNs > 0 {
				findingID := fmt.Sprintf("find_%03d", findingIndex)
				findingIndex++

				kernelMs := float64(e.KernelDurationNs) / 1e6
				h2dMs := float64(e.TransferH2DNs) / 1e6
				d2hMs := float64(e.TransferD2HNs) / 1e6
				transferMs := h2dMs + d2hMs
				totalMs := e.Summary.MedianMs

				title := fmt.Sprintf("%s GPU Execution Breakdown: Kernel (%.2f ms) vs PCIe Transfer (%.2f ms)", w.WorkloadName, kernelMs, transferMs)
				explanation := fmt.Sprintf(
					"GPU pure kernel execution via cudaEvent_t required %.2f ms, while PCIe bus data movement (H2D + D2H) consumed %.2f ms (%.1f%% of end-to-end wall clock). Effective bandwidth was %.1f GB/s.",
					kernelMs, transferMs, (transferMs/totalMs)*100.0, e.EffectiveBandwidthGBps,
				)

				f := ReportFinding{
					ID:                 findingID,
					Title:              title,
					Category:           FindingAccelerator,
					WorkloadID:         w.WorkloadID,
					Track:              "gpu",
					Scope:              "CUDA Kernel vs PCIe Transfer Breakdown",
					BaselineID:         "cpu_reference",
					CandidateID:        e.LanguageID,
					MetricName:         "PCIe Transfer Overhead",
					MetricUnit:         "ms",
					BaselineValue:      kernelMs,
					CandidateValue:     transferMs,
					EffectSize:         fmt.Sprintf("%.1f%% of wall clock", (transferMs/totalMs)*100.0),
					SampleCount:        e.Summary.Count,
					Uncertainty:        fmt.Sprintf("Synchronized cudaEvent_t kernel timing (%.2f ms)", kernelMs),
					Status:             metrics.AssessmentStatisticallySig,
					Explanation:        explanation,
					Limitations:        "PCIe latency dominates small payloads; kernel speedup manifests only when compute intensity offsets bus transfers.",
					ChartRelativePath:  fmt.Sprintf("charts/%s.svg", w.WorkloadID),
					RawDataRelativePath: "raw-results.json",
				}
				findings = append(findings, f)
			}
		}

		// 6. Skipped or Unavailable Findings (Diagnostic notes)
		for _, e := range skippedOrFailed {
			findingID := fmt.Sprintf("find_%03d", findingIndex)
			findingIndex++

			title := fmt.Sprintf("%s Configuration Unavailable: %s (%s)", w.WorkloadName, e.LanguageID, e.Status)
			explanation := fmt.Sprintf("Execution %s was marked %s. %s", e.LanguageID, e.Status, e.ErrorMessage)

			f := ReportFinding{
				ID:                 findingID,
				Title:              title,
				Category:           FindingDiagnosticNote,
				WorkloadID:         w.WorkloadID,
				Track:              w.Track,
				Scope:              "Toolchain & Device Capability Validation",
				BaselineID:         w.BaselineID,
				CandidateID:        e.LanguageID,
				MetricName:         "Execution Status",
				MetricUnit:         "status",
				BaselineValue:      0,
				CandidateValue:     0,
				EffectSize:         string(e.Status),
				SampleCount:        0,
				Uncertainty:        "N/A",
				Status:             metrics.AssessmentInconclusive,
				Explanation:        explanation,
				Limitations:        "Reported transparently without silent fallback or fabricated values.",
				ChartRelativePath:  "charts/overview.svg",
				RawDataRelativePath: "raw-results.json",
			}
			findings = append(findings, f)
		}
	}

	return findings
}
