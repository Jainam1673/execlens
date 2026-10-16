package reports

import (
	"fmt"
	"html"
	"math"
	"strings"

	"github.com/execlens/execlens/internal/database"
)

// Language/Variant palette for consistent, accessible visual identification
var colorPalette = map[string]string{
	"cpp":           "#3b82f6", // blue
	"rust":          "#f97316", // orange
	"go":            "#06b6d4", // cyan
	"java":          "#eab308", // amber/yellow
	"python":        "#10b981", // emerald
	"javascript":    "#a855f7", // purple
	"scalar":        "#64748b", // slate
	"autovec":       "#3b82f6", // blue
	"simd_avx2":     "#06b6d4", // cyan
	"openmp":        "#8b5cf6", // violet
	"asm":           "#ec4899", // pink
	"aos":           "#f43f5e", // rose
	"soa":           "#10b981", // emerald
	"memcpy":        "#3b82f6", // blue
	"unroll64":      "#f59e0b", // amber
	"avx2_stream":   "#06b6d4", // cyan
	"cuda_kernel":   "#22c55e", // green
	"pcie_transfer": "#f59e0b", // amber
	"cpu_reference": "#64748b", // slate
	"default":       "#6366f1", // indigo
}

func getColor(key string) string {
	if c, ok := colorPalette[strings.ToLower(key)]; ok {
		return c
	}
	return colorPalette["default"]
}

// 1. RenderOverviewSVG generates the benchmark execution and health overview graphic
func RenderOverviewSVG(summary *ReportSummary, sysInfo database.SystemInfo) string {
	width := 820
	height := 260

	dateStr := summary.GeneratedAt.UTC().Format("2006-01-02 15:04:05 UTC")
	cpuModel := sysInfo.CPUModel
	if len(cpuModel) > 42 {
		cpuModel = cpuModel[:40] + "..."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, width, height, width, height))
	sb.WriteString(`<defs>
		<style>
			.title { font-family: system-ui, -apple-system, sans-serif; font-size: 16px; font-weight: 700; fill: #f8fafc; }
			.subtitle { font-family: system-ui, -apple-system, sans-serif; font-size: 12px; fill: #94a3b8; }
			.card-title { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; font-weight: 600; fill: #94a3b8; text-transform: uppercase; letter-spacing: 0.5px; }
			.card-val { font-family: system-ui, -apple-system, sans-serif; font-size: 24px; font-weight: 700; fill: #f8fafc; }
			.badge { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; font-weight: 500; }
		</style>
	</defs>`)

	// Background
	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" rx="8" fill="#0f172a" stroke="#1e293b" stroke-width="1"/>`, width, height))

	// Header
	sb.WriteString(fmt.Sprintf(`<text x="24" y="34" class="title">ExecLens — Benchmark Execution Overview</text>`))
	sb.WriteString(fmt.Sprintf(`<text x="24" y="54" class="subtitle">Report ID: %s | Host: %s (%s, %d vCPUs) | %s</text>`,
		html.EscapeString(summary.ReportID), html.EscapeString(cpuModel), html.EscapeString(sysInfo.OS), sysInfo.LogicalCPUs, dateStr))

	// 4 Stat Cards
	cards := []struct {
		label string
		val   string
		color string
	}{
		{"Configurations Tested", fmt.Sprintf("%d", summary.TotalConfigurations), "#38bdf8"},
		{"Successful Executions", fmt.Sprintf("%d", summary.SuccessfulExecutions), "#4ade80"},
		{"Statistically Significant", fmt.Sprintf("%d", summary.StatSignificantCount), "#a78bfa"},
		{"Inconclusive / Variance", fmt.Sprintf("%d", summary.InconclusiveCount), "#fbbf24"},
	}

	cardWidth := 180
	cardHeight := 75
	cardY := 75
	startX := 24
	gap := 15

	for i, c := range cards {
		x := startX + i*(cardWidth+gap)
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="6" fill="#1e293b" stroke="#334155" stroke-width="1"/>`, x, cardY, cardWidth, cardHeight))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="card-title">%s</text>`, x+14, cardY+24, c.label))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="card-val" fill="%s">%s</text>`, x+14, cardY+56, c.color, c.val))
	}

	// Bottom Summary Section: Coverage Tags
	sb.WriteString(`<rect x="24" y="165" width="772" height="75" rx="6" fill="#1e293b" stroke="#334155" stroke-width="1"/>`)
	sb.WriteString(`<text x="40" y="190" class="card-title">Coverage &amp; Integrity Summary</text>`)

	coverageText := fmt.Sprintf("Workloads: %s  |  Runtimes: %s  |  Skipped/Unavailable: %d  |  Validation Failures: %d",
		strings.Join(summary.WorkloadsCovered, ", "),
		strings.Join(summary.LanguagesTested, ", "),
		summary.SkippedExecutions,
		summary.ValidationFailures,
	)
	if len(coverageText) > 105 {
		coverageText = coverageText[:102] + "..."
	}
	sb.WriteString(fmt.Sprintf(`<text x="40" y="215" class="subtitle" fill="#cbd5e1">%s</text>`, html.EscapeString(coverageText)))

	sb.WriteString("</svg>")
	return sb.String()
}

// 2. RenderRuntimeComparisonSVG generates horizontal bar chart with 95% CI whiskers and speedup annotations
func RenderRuntimeComparisonSVG(w WorkloadReportData) string {
	width := 820
	baseHeight := 140
	barHeight := 28
	gap := 18

	var validExecs []ExecutionView
	for _, e := range w.Executions {
		if e.Status == "COMPLETED" && e.Summary != nil && e.Summary.MedianMs > 0 {
			validExecs = append(validExecs, e)
		}
	}

	if len(validExecs) == 0 {
		return renderEmptyChartSVG(width, 180, w.WorkloadName+" Execution Time", "No completed execution measurements available.")
	}

	totalHeight := baseHeight + len(validExecs)*(barHeight+gap)

	// Determine max value for X-axis scale (using CI95UpperMs)
	var maxMs float64 = 0.001
	for _, e := range validExecs {
		upper := e.Summary.CI95UpperMs
		if upper > maxMs {
			maxMs = upper
		}
	}
	maxMs = maxMs * 1.15 // padding

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, width, totalHeight, width, totalHeight))
	sb.WriteString(`<defs>
		<style>
			.title { font-family: system-ui, -apple-system, sans-serif; font-size: 15px; font-weight: 700; fill: #f8fafc; }
			.subtitle { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; fill: #94a3b8; }
			.axis-label { font-family: system-ui, -apple-system, sans-serif; font-size: 12px; font-weight: 600; fill: #f1f5f9; }
			.val-label { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; font-weight: 600; fill: #e2e8f0; }
			.grid-label { font-family: system-ui, -apple-system, sans-serif; font-size: 10px; fill: #64748b; }
			.ci-whisker { stroke: #cbd5e1; stroke-width: 1.5; }
		</style>
	</defs>`)

	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" rx="8" fill="#0f172a" stroke="#1e293b" stroke-width="1"/>`, width, totalHeight))

	// Header
	sb.WriteString(fmt.Sprintf(`<text x="24" y="32" class="title">%s — Median Execution Time (Lower is Faster)</text>`, html.EscapeString(w.WorkloadName)))
	sb.WriteString(fmt.Sprintf(`<text x="24" y="50" class="subtitle">Steady-state measured iterations with Student's t 95%% Confidence Interval whiskers | Baseline: %s</text>`, html.EscapeString(w.BaselineID)))

	chartLeft := 150
	chartRight := width - 110
	chartWidth := chartRight - chartLeft
	startY := 75

	// Grid lines
	gridSteps := 4
	for i := 0; i <= gridSteps; i++ {
		gx := chartLeft + (i * chartWidth / gridSteps)
		val := (float64(i) / float64(gridSteps)) * maxMs
		sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#334155" stroke-dasharray="2 2" stroke-width="1"/>`, gx, startY, gx, startY+len(validExecs)*(barHeight+gap)))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" class="grid-label">%.1f ms</text>`, gx, startY+len(validExecs)*(barHeight+gap)+18, val))
	}

	for i, e := range validExecs {
		y := startY + i*(barHeight+gap)
		barW := int((e.Summary.MedianMs / maxMs) * float64(chartWidth))
		if barW < 3 {
			barW = 3
		}

		color := getColor(e.LanguageID)
		isBaseline := e.LanguageID == w.BaselineID
		labelSuffix := ""
		if isBaseline {
			labelSuffix = " (Baseline)"
		}

		// Y-axis label
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="axis-label" text-anchor="end">%s%s</text>`, chartLeft-12, y+18, html.EscapeString(e.LanguageID), labelSuffix))

		// Bar
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="4" fill="%s" opacity="0.9"/>`, chartLeft, y, barW, barHeight, color))

		// 95% Confidence Interval Whisker
		ciLeftX := chartLeft + int((e.Summary.CI95LowerMs/maxMs)*float64(chartWidth))
		ciRightX := chartLeft + int((e.Summary.CI95UpperMs/maxMs)*float64(chartWidth))
		midY := y + barHeight/2

		sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" class="ci-whisker"/>`, ciLeftX, midY, ciRightX, midY))
		sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" class="ci-whisker"/>`, ciLeftX, midY-4, ciLeftX, midY+4))
		sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" class="ci-whisker"/>`, ciRightX, midY-4, ciRightX, midY+4))

		// Value label
		speedupStr := ""
		if e.Speedup > 0 && !isBaseline {
			speedupStr = fmt.Sprintf(" [%.2fx]", e.Speedup)
		}
		valText := fmt.Sprintf("%.2f ms%s", e.Summary.MedianMs, speedupStr)
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="val-label">%s</text>`, ciRightX+10, y+18, valText))
	}

	sb.WriteString("</svg>")
	return sb.String()
}

// 3. RenderThroughputComparisonSVG generates throughput comparison chart (ops/s or MB/s)
func RenderThroughputComparisonSVG(w WorkloadReportData) string {
	width := 820
	baseHeight := 140
	barHeight := 28
	gap := 18

	var validExecs []ExecutionView
	for _, e := range w.Executions {
		if e.Status == "COMPLETED" && e.Summary != nil && (e.Summary.ThroughputOpsPerSec > 0 || e.Summary.ThroughputMBPerSec > 0) {
			validExecs = append(validExecs, e)
		}
	}

	if len(validExecs) == 0 {
		return renderEmptyChartSVG(width, 180, w.WorkloadName+" Throughput", "No defined throughput metrics for this workload.")
	}

	useMB := validExecs[0].Summary.ThroughputMBPerSec > 0
	unit := "ops/s"
	if useMB {
		unit = "MB/s"
	}

	totalHeight := baseHeight + len(validExecs)*(barHeight+gap)

	var maxVal float64 = 0.001
	for _, e := range validExecs {
		val := e.Summary.ThroughputOpsPerSec
		if useMB {
			val = e.Summary.ThroughputMBPerSec
		}
		if val > maxVal {
			maxVal = val
		}
	}
	maxVal = maxVal * 1.15

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, width, totalHeight, width, totalHeight))
	sb.WriteString(`<defs>
		<style>
			.title { font-family: system-ui, -apple-system, sans-serif; font-size: 15px; font-weight: 700; fill: #f8fafc; }
			.subtitle { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; fill: #94a3b8; }
			.axis-label { font-family: system-ui, -apple-system, sans-serif; font-size: 12px; font-weight: 600; fill: #f1f5f9; }
			.val-label { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; font-weight: 600; fill: #e2e8f0; }
			.grid-label { font-family: system-ui, -apple-system, sans-serif; font-size: 10px; fill: #64748b; }
		</style>
	</defs>`)

	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" rx="8" fill="#0f172a" stroke="#1e293b" stroke-width="1"/>`, width, totalHeight))

	// Header
	sb.WriteString(fmt.Sprintf(`<text x="24" y="32" class="title">%s — Sustained Throughput (%s, Higher is Better)</text>`, html.EscapeString(w.WorkloadName), unit))
	sb.WriteString(fmt.Sprintf(`<text x="24" y="50" class="subtitle">Effective computational operations completed per second of median execution time</text>`))

	chartLeft := 150
	chartRight := width - 110
	chartWidth := chartRight - chartLeft
	startY := 75

	// Grid lines
	gridSteps := 4
	for i := 0; i <= gridSteps; i++ {
		gx := chartLeft + (i * chartWidth / gridSteps)
		val := (float64(i) / float64(gridSteps)) * maxVal
		valStr := formatLargeNumber(val)
		sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#334155" stroke-dasharray="2 2" stroke-width="1"/>`, gx, startY, gx, startY+len(validExecs)*(barHeight+gap)))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" class="grid-label">%s %s</text>`, gx, startY+len(validExecs)*(barHeight+gap)+18, valStr, unit))
	}

	for i, e := range validExecs {
		y := startY + i*(barHeight+gap)
		val := e.Summary.ThroughputOpsPerSec
		if useMB {
			val = e.Summary.ThroughputMBPerSec
		}

		barW := int((val / maxVal) * float64(chartWidth))
		if barW < 3 {
			barW = 3
		}

		color := getColor(e.LanguageID)

		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="axis-label" text-anchor="end">%s</text>`, chartLeft-12, y+18, html.EscapeString(e.LanguageID)))
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="4" fill="%s" opacity="0.9"/>`, chartLeft, y, barW, barHeight, color))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="val-label">%s %s</text>`, chartLeft+barW+8, y+18, formatLargeNumber(val), unit))
	}

	sb.WriteString("</svg>")
	return sb.String()
}

// 4. RenderMemoryComparisonSVG generates peak RSS physical memory usage chart
func RenderMemoryComparisonSVG(w WorkloadReportData) string {
	width := 820
	baseHeight := 140
	barHeight := 28
	gap := 18

	var validExecs []ExecutionView
	for _, e := range w.Executions {
		if e.PeakMemoryMB > 0 {
			validExecs = append(validExecs, e)
		}
	}

	if len(validExecs) == 0 {
		return renderEmptyChartSVG(width, 180, w.WorkloadName+" Peak Memory", "No physical memory observations captured.")
	}

	totalHeight := baseHeight + len(validExecs)*(barHeight+gap)

	var maxMB float64 = 0.001
	for _, e := range validExecs {
		if e.PeakMemoryMB > maxMB {
			maxMB = e.PeakMemoryMB
		}
	}
	maxMB = maxMB * 1.15

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, width, totalHeight, width, totalHeight))
	sb.WriteString(`<defs>
		<style>
			.title { font-family: system-ui, -apple-system, sans-serif; font-size: 15px; font-weight: 700; fill: #f8fafc; }
			.subtitle { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; fill: #94a3b8; }
			.axis-label { font-family: system-ui, -apple-system, sans-serif; font-size: 12px; font-weight: 600; fill: #f1f5f9; }
			.val-label { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; font-weight: 600; fill: #e2e8f0; }
			.grid-label { font-family: system-ui, -apple-system, sans-serif; font-size: 10px; fill: #64748b; }
		</style>
	</defs>`)

	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" rx="8" fill="#0f172a" stroke="#1e293b" stroke-width="1"/>`, width, totalHeight))

	// Header
	sb.WriteString(fmt.Sprintf(`<text x="24" y="32" class="title">%s — Peak Physical Memory Footprint (MaxRSS)</text>`, html.EscapeString(w.WorkloadName)))
	sb.WriteString(`<text x="24" y="50" class="subtitle">Peak Resident Set Size (RSS) recorded from Linux kernel getrusage | Lower is More Memory Efficient</text>`)

	chartLeft := 150
	chartRight := width - 110
	chartWidth := chartRight - chartLeft
	startY := 75

	// Grid lines
	gridSteps := 4
	for i := 0; i <= gridSteps; i++ {
		gx := chartLeft + (i * chartWidth / gridSteps)
		val := (float64(i) / float64(gridSteps)) * maxMB
		sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#334155" stroke-dasharray="2 2" stroke-width="1"/>`, gx, startY, gx, startY+len(validExecs)*(barHeight+gap)))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" class="grid-label">%.1f MB</text>`, gx, startY+len(validExecs)*(barHeight+gap)+18, val))
	}

	for i, e := range validExecs {
		y := startY + i*(barHeight+gap)
		barW := int((e.PeakMemoryMB / maxMB) * float64(chartWidth))
		if barW < 3 {
			barW = 3
		}

		color := getColor(e.LanguageID)
		ratioStr := ""
		if e.MemoryRatio > 0 && e.LanguageID != w.BaselineID {
			ratioStr = fmt.Sprintf(" (%.1fx vs baseline)", e.MemoryRatio)
		}

		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="axis-label" text-anchor="end">%s</text>`, chartLeft-12, y+18, html.EscapeString(e.LanguageID)))
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="4" fill="%s" opacity="0.9"/>`, chartLeft, y, barW, barHeight, color))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="val-label">%.1f MB%s</text>`, chartLeft+barW+8, y+18, e.PeakMemoryMB, ratioStr))
	}

	sb.WriteString("</svg>")
	return sb.String()
}

// 5. RenderVariabilitySVG generates distribution dot / sample spread plot showing individual observations
func RenderVariabilitySVG(w WorkloadReportData) string {
	width := 820
	baseHeight := 140
	rowHeight := 36
	gap := 14

	var validExecs []ExecutionView
	for _, e := range w.Executions {
		if e.Status == "COMPLETED" && len(e.Samples) > 0 {
			validExecs = append(validExecs, e)
		}
	}

	if len(validExecs) == 0 {
		return renderEmptyChartSVG(width, 180, w.WorkloadName+" Sample Variability", "No sample iterations recorded.")
	}

	totalHeight := baseHeight + len(validExecs)*(rowHeight+gap)

	// Find global min and max among non-warmup samples
	var minMs float64 = 1e9
	var maxMs float64 = 0.001

	for _, e := range validExecs {
		for _, s := range e.Samples {
			if !s.IsWarmup {
				if s.DurationMs < minMs {
					minMs = s.DurationMs
				}
				if s.DurationMs > maxMs {
					maxMs = s.DurationMs
				}
			}
		}
	}

	minMs = math.Max(0, minMs*0.85)
	maxMs = maxMs * 1.15
	span := maxMs - minMs
	if span <= 0 {
		span = 1
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, width, totalHeight, width, totalHeight))
	sb.WriteString(`<defs>
		<style>
			.title { font-family: system-ui, -apple-system, sans-serif; font-size: 15px; font-weight: 700; fill: #f8fafc; }
			.subtitle { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; fill: #94a3b8; }
			.axis-label { font-family: system-ui, -apple-system, sans-serif; font-size: 12px; font-weight: 600; fill: #f1f5f9; }
			.grid-label { font-family: system-ui, -apple-system, sans-serif; font-size: 10px; fill: #64748b; }
			.sample-dot { fill: #38bdf8; stroke: #0f172a; stroke-width: 1.5; }
			.median-tick { stroke: #f43f5e; stroke-width: 2.5; }
			.track-bg { fill: #1e293b; rx: 4; }
		</style>
	</defs>`)

	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" rx="8" fill="#0f172a" stroke="#1e293b" stroke-width="1"/>`, width, totalHeight))

	// Header
	sb.WriteString(fmt.Sprintf(`<text x="24" y="32" class="title">%s — Sample Dispersion &amp; Measurement Stability</text>`, html.EscapeString(w.WorkloadName)))
	sb.WriteString(`<text x="24" y="50" class="subtitle">Individual repetition sample dots (cyan) with median marker (red line) | Highlights environmental noise vs stability</text>`)

	chartLeft := 150
	chartRight := width - 110
	chartWidth := chartRight - chartLeft
	startY := 75

	// Grid
	gridSteps := 4
	for i := 0; i <= gridSteps; i++ {
		gx := chartLeft + (i * chartWidth / gridSteps)
		val := minMs + (float64(i)/float64(gridSteps))*span
		sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#334155" stroke-dasharray="2 2" stroke-width="1"/>`, gx, startY, gx, startY+len(validExecs)*(rowHeight+gap)))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" class="grid-label">%.1f ms</text>`, gx, startY+len(validExecs)*(rowHeight+gap)+18, val))
	}

	for i, e := range validExecs {
		y := startY + i*(rowHeight+gap)

		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="axis-label" text-anchor="end">%s</text>`, chartLeft-12, y+22, html.EscapeString(e.LanguageID)))
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" class="track-bg"/>`, chartLeft, y, chartWidth, rowHeight))

		// Plot sample dots
		for _, s := range e.Samples {
			if !s.IsWarmup {
				cx := chartLeft + int(((s.DurationMs-minMs)/span)*float64(chartWidth))
				sb.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="5" class="sample-dot"/>`, cx, y+rowHeight/2))
			}
		}

		// Median marker
		if e.Summary != nil {
			medX := chartLeft + int(((e.Summary.MedianMs-minMs)/span)*float64(chartWidth))
			sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" class="median-tick"/>`, medX, y+3, medX, y+rowHeight-3))
			sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="grid-label" fill="#f43f5e">CV %.1f%%</text>`, chartRight+12, y+22, e.Summary.CVPercent))
		}
	}

	sb.WriteString("</svg>")
	return sb.String()
}

// 6. RenderHeatmapSVG generates language-by-workload normalized speedup heatmap
func RenderHeatmapSVG(workloads []WorkloadReportData) string {
	width := 820
	cellWidth := 90
	cellHeight := 38
	headerHeight := 90
	legendHeight := 50
	labelWidth := 160

	// Gather unique languages
	langSet := make(map[string]bool)
	var languages []string
	for _, w := range workloads {
		for _, e := range w.Executions {
			if e.Status == "COMPLETED" && !langSet[e.LanguageID] {
				langSet[e.LanguageID] = true
				languages = append(languages, e.LanguageID)
			}
		}
	}

	if len(workloads) == 0 || len(languages) == 0 {
		return renderEmptyChartSVG(width, 220, "Workload vs Language Heatmap", "Insufficient completed multi-workload data.")
	}

	totalHeight := headerHeight + len(workloads)*cellHeight + legendHeight

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, width, totalHeight, width, totalHeight))
	sb.WriteString(`<defs>
		<style>
			.title { font-family: system-ui, -apple-system, sans-serif; font-size: 15px; font-weight: 700; fill: #f8fafc; }
			.subtitle { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; fill: #94a3b8; }
			.cell-text { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; font-weight: 600; text-anchor: middle; fill: #f8fafc; }
			.row-label { font-family: system-ui, -apple-system, sans-serif; font-size: 12px; font-weight: 500; fill: #cbd5e1; }
			.col-label { font-family: system-ui, -apple-system, sans-serif; font-size: 12px; font-weight: 600; text-anchor: middle; fill: #38bdf8; }
		</style>
	</defs>`)

	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" rx="8" fill="#0f172a" stroke="#1e293b" stroke-width="1"/>`, width, totalHeight))

	// Header
	sb.WriteString(`<text x="24" y="32" class="title">Cross-Workload Performance Heatmap (Speedup vs Baseline)</text>`)
	sb.WriteString(`<text x="24" y="50" class="subtitle">Normalized speedup ratio (baseline_time / candidate_time) | Green: Faster (&gt;1.0x), Red: Slower (&lt;1.0x)</text>`)

	// Column Headers
	startY := 80
	for j, lang := range languages {
		x := labelWidth + j*cellWidth + cellWidth/2
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="col-label">%s</text>`, x, startY-10, html.EscapeString(lang)))
	}

	// Rows
	for i, w := range workloads {
		y := startY + i*cellHeight

		// Row Label
		rowName := w.WorkloadID
		if len(rowName) > 18 {
			rowName = rowName[:16] + "..."
		}
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="row-label">%s</text>`, 24, y+24, html.EscapeString(rowName)))

		for j, lang := range languages {
			x := labelWidth + j*cellWidth

			// Find execution
			var exec *ExecutionView
			for _, e := range w.Executions {
				if e.LanguageID == lang && e.Status == "COMPLETED" {
					exec = &e
					break
				}
			}

			fillColor := "#1e293b" // default / missing
			textVal := "N/A"

			if exec != nil && exec.Summary != nil {
				speedup := exec.Speedup
				if speedup <= 0 {
					speedup = 1.0
				}
				textVal = fmt.Sprintf("%.2fx", speedup)

				if speedup >= 2.0 {
					fillColor = "#15803d" // deep green
				} else if speedup >= 1.1 {
					fillColor = "#166534" // green
				} else if speedup >= 0.95 {
					fillColor = "#334155" // slate (near parity)
				} else if speedup >= 0.5 {
					fillColor = "#991b1b" // red
				} else {
					fillColor = "#7f1d1d" // dark red
				}
			}

			sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="%s" stroke="#0f172a" stroke-width="1.5" rx="3"/>`, x, y, cellWidth, cellHeight, fillColor))
			sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="cell-text">%s</text>`, x+cellWidth/2, y+23, textVal))
		}
	}

	// Legend at bottom
	legY := startY + len(workloads)*cellHeight + 25
	sb.WriteString(fmt.Sprintf(`<g transform="translate(160, %d)">`, legY))
	sb.WriteString(`<rect x="0" y="0" width="16" height="12" fill="#15803d" rx="2"/><text x="22" y="10" class="subtitle">&gt;2.0x Fast</text>`)
	sb.WriteString(`<rect x="100" y="0" width="16" height="12" fill="#166534" rx="2"/><text x="122" y="10" class="subtitle">1.1x–2.0x</text>`)
	sb.WriteString(`<rect x="190" y="0" width="16" height="12" fill="#334155" rx="2"/><text x="212" y="10" class="subtitle">~Parity</text>`)
	sb.WriteString(`<rect x="270" y="0" width="16" height="12" fill="#991b1b" rx="2"/><text x="292" y="10" class="subtitle">0.5x–0.9x</text>`)
	sb.WriteString(`<rect x="360" y="0" width="16" height="12" fill="#7f1d1d" rx="2"/><text x="382" y="10" class="subtitle">&lt;0.5x Slow</text>`)
	sb.WriteString(`<rect x="460" y="0" width="16" height="12" fill="#1e293b" rx="2"/><text x="482" y="10" class="subtitle">N/A</text>`)
	sb.WriteString(`</g>`)

	sb.WriteString("</svg>")
	return sb.String()
}

// 7. RenderStartupComparisonSVG generates cold-start vs steady-state comparison
func RenderStartupComparisonSVG(w WorkloadReportData) string {
	width := 820
	baseHeight := 140
	barHeight := 22
	gap := 20

	var validExecs []ExecutionView
	for _, e := range w.Executions {
		if e.Status == "COMPLETED" && len(e.Samples) > 0 {
			validExecs = append(validExecs, e)
		}
	}

	if len(validExecs) == 0 {
		return renderEmptyChartSVG(width, 180, w.WorkloadName+" Startup vs Steady-State", "No sample iteration timing captured.")
	}

	totalHeight := baseHeight + len(validExecs)*(barHeight*2+gap)

	var maxMs float64 = 0.001
	for _, e := range validExecs {
		firstSample := e.Samples[0].DurationMs
		if firstSample > maxMs {
			maxMs = firstSample
		}
		if e.Summary != nil && e.Summary.MedianMs > maxMs {
			maxMs = e.Summary.MedianMs
		}
	}
	maxMs = maxMs * 1.15

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, width, totalHeight, width, totalHeight))
	sb.WriteString(`<defs>
		<style>
			.title { font-family: system-ui, -apple-system, sans-serif; font-size: 15px; font-weight: 700; fill: #f8fafc; }
			.subtitle { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; fill: #94a3b8; }
			.axis-label { font-family: system-ui, -apple-system, sans-serif; font-size: 12px; font-weight: 600; fill: #f1f5f9; }
			.val-label { font-family: system-ui, -apple-system, sans-serif; font-size: 10px; font-weight: 600; fill: #e2e8f0; }
			.grid-label { font-family: system-ui, -apple-system, sans-serif; font-size: 10px; fill: #64748b; }
		</style>
	</defs>`)

	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" rx="8" fill="#0f172a" stroke="#1e293b" stroke-width="1"/>`, width, totalHeight))

	// Header
	sb.WriteString(fmt.Sprintf(`<text x="24" y="32" class="title">%s — Cold Start vs Steady-State Execution</text>`, html.EscapeString(w.WorkloadName)))
	sb.WriteString(`<text x="24" y="50" class="subtitle">Comparison of Initial Run / Warmup Iteration (blue) vs Steady-State Median (green) | Highlights JIT warmup overhead</text>`)

	chartLeft := 150
	chartRight := width - 110
	chartWidth := chartRight - chartLeft
	startY := 75

	for i, e := range validExecs {
		y := startY + i*(barHeight*2+gap)
		firstMs := e.Samples[0].DurationMs
		steadyMs := e.Summary.MedianMs

		wFirst := int((firstMs / maxMs) * float64(chartWidth))
		wSteady := int((steadyMs / maxMs) * float64(chartWidth))

		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="axis-label" text-anchor="end">%s</text>`, chartLeft-12, y+24, html.EscapeString(e.LanguageID)))

		// Cold start bar
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="3" fill="#38bdf8"/>`, chartLeft, y, wFirst, barHeight))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="val-label">Cold: %.2f ms</text>`, chartLeft+wFirst+8, y+15, firstMs))

		// Steady-state bar
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="3" fill="#22c55e"/>`, chartLeft, y+barHeight+2, wSteady, barHeight))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="val-label">Steady: %.2f ms</text>`, chartLeft+wSteady+8, y+barHeight+17, steadyMs))
	}

	sb.WriteString("</svg>")
	return sb.String()
}

// 8. RenderBuildComparisonSVG generates compilation time vs binary size chart
func RenderBuildComparisonSVG(w WorkloadReportData) string {
	width := 820
	baseHeight := 140
	barHeight := 28
	gap := 18

	var validExecs []ExecutionView
	for _, e := range w.Executions {
		if e.BuildDurationMs > 0 || e.ExecutableSizeBytes > 0 {
			validExecs = append(validExecs, e)
		}
	}

	if len(validExecs) == 0 {
		return renderEmptyChartSVG(width, 180, w.WorkloadName+" Build Metrics", "No compiled toolchain build times recorded (interpreter or scripts).")
	}

	totalHeight := baseHeight + len(validExecs)*(barHeight+gap)

	var maxBuildMs float64 = 0.001
	for _, e := range validExecs {
		if e.BuildDurationMs > maxBuildMs {
			maxBuildMs = e.BuildDurationMs
		}
	}
	maxBuildMs = maxBuildMs * 1.15

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, width, totalHeight, width, totalHeight))
	sb.WriteString(`<defs>
		<style>
			.title { font-family: system-ui, -apple-system, sans-serif; font-size: 15px; font-weight: 700; fill: #f8fafc; }
			.subtitle { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; fill: #94a3b8; }
			.axis-label { font-family: system-ui, -apple-system, sans-serif; font-size: 12px; font-weight: 600; fill: #f1f5f9; }
			.val-label { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; font-weight: 600; fill: #e2e8f0; }
		</style>
	</defs>`)

	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" rx="8" fill="#0f172a" stroke="#1e293b" stroke-width="1"/>`, width, totalHeight))

	// Header
	sb.WriteString(fmt.Sprintf(`<text x="24" y="32" class="title">%s — Compilation Time &amp; Executable Binary Size</text>`, html.EscapeString(w.WorkloadName)))
	sb.WriteString(`<text x="24" y="50" class="subtitle">Build duration in milliseconds and compiled artifact size on disk</text>`)

	chartLeft := 150
	chartRight := width - 110
	chartWidth := chartRight - chartLeft
	startY := 75

	for i, e := range validExecs {
		y := startY + i*(barHeight+gap)
		barW := int((e.BuildDurationMs / maxBuildMs) * float64(chartWidth))
		if barW < 3 {
			barW = 3
		}

		color := getColor(e.LanguageID)
		sizeKB := float64(e.ExecutableSizeBytes) / 1024.0
		sizeStr := ""
		if sizeKB > 0 {
			if sizeKB > 1024 {
				sizeStr = fmt.Sprintf(" | Size: %.1f MB", sizeKB/1024.0)
			} else {
				sizeStr = fmt.Sprintf(" | Size: %.0f KB", sizeKB)
			}
		}

		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="axis-label" text-anchor="end">%s</text>`, chartLeft-12, y+18, html.EscapeString(e.LanguageID)))
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="4" fill="%s"/>`, chartLeft, y, barW, barHeight, color))
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="val-label">%.0f ms%s</text>`, chartLeft+barW+8, y+18, e.BuildDurationMs, sizeStr))
	}

	sb.WriteString("</svg>")
	return sb.String()
}

// 9. RenderScalingSVG generates line chart showing performance scaling across input sizes
func RenderScalingSVG(workloads []WorkloadReportData) string {
	width := 820
	height := 320

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, width, height, width, height))
	sb.WriteString(`<defs>
		<style>
			.title { font-family: system-ui, -apple-system, sans-serif; font-size: 15px; font-weight: 700; fill: #f8fafc; }
			.subtitle { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; fill: #94a3b8; }
			.axis-label { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; fill: #94a3b8; }
			.grid-line { stroke: #334155; stroke-dasharray: 2 2; stroke-width: 1; }
			.legend-text { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; fill: #cbd5e1; }
		</style>
	</defs>`)

	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" rx="8" fill="#0f172a" stroke="#1e293b" stroke-width="1"/>`, width, height))

	sb.WriteString(`<text x="24" y="32" class="title">Workload Input Scaling Progression</text>`)
	sb.WriteString(`<text x="24" y="50" class="subtitle">Empirical runtime scaling comparison across tested workload configurations</text>`)

	// Plot placeholder/grid if single configuration
	chartLeft := 80
	chartRight := width - 60
	chartTop := 80
	chartBottom := height - 60

	sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#64748b" stroke-width="1.5"/>`, chartLeft, chartBottom, chartRight, chartBottom))
	sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#64748b" stroke-width="1.5"/>`, chartLeft, chartTop, chartLeft, chartBottom))

	// If workloads are present, plot points per workload
	step := (chartRight - chartLeft) / int(math.Max(1, float64(len(workloads)+1)))
	for i, w := range workloads {
		x := chartLeft + (i+1)*step
		name := w.WorkloadID
		if len(name) > 10 {
			name = name[:8] + ".."
		}
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" class="axis-label">%s</text>`, x, chartBottom+18, html.EscapeString(name)))

		// Plot dots for executions
		for _, e := range w.Executions {
			if e.Status == "COMPLETED" && e.Summary != nil {
				// Normalize Y
				y := chartBottom - int(math.Min(float64(chartBottom-chartTop-20), (e.Summary.MedianMs/200.0)*float64(chartBottom-chartTop)))
				color := getColor(e.LanguageID)
				sb.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="4.5" fill="%s" opacity="0.8"/>`, x, y, color))
			}
		}
	}

	sb.WriteString("</svg>")
	return sb.String()
}

// 10. RenderWorkloadSpecificSVG generates dedicated chart for a specific workload
func RenderWorkloadSpecificSVG(w WorkloadReportData) string {
	// If GPU workload, render GPU breakdown chart
	if w.Track == "gpu" || strings.HasPrefix(w.WorkloadID, "cuda_") {
		return renderGPUBreakdownSVG(w)
	}
	// Default to detailed runtime comparison
	return RenderRuntimeComparisonSVG(w)
}

func renderGPUBreakdownSVG(w WorkloadReportData) string {
	width := 820
	height := 320

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, width, height, width, height))
	sb.WriteString(`<defs>
		<style>
			.title { font-family: system-ui, -apple-system, sans-serif; font-size: 15px; font-weight: 700; fill: #f8fafc; }
			.subtitle { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; fill: #94a3b8; }
			.axis-label { font-family: system-ui, -apple-system, sans-serif; font-size: 12px; font-weight: 600; fill: #f1f5f9; }
			.val-label { font-family: system-ui, -apple-system, sans-serif; font-size: 11px; font-weight: 600; fill: #e2e8f0; }
		</style>
	</defs>`)

	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" rx="8" fill="#0f172a" stroke="#1e293b" stroke-width="1"/>`, width, height))
	sb.WriteString(fmt.Sprintf(`<text x="24" y="32" class="title">%s — GPU Telemetry &amp; Execution Breakdown</text>`, html.EscapeString(w.WorkloadName)))
	sb.WriteString(`<text x="24" y="50" class="subtitle">Synchronized cudaEvent_t Kernel Duration vs PCIe Memory Transfers vs CPU Reference</text>`)

	chartLeft := 160
	chartWidth := width - 260
	startY := 85
	barHeight := 28
	gap := 20

	var execs []ExecutionView
	for _, e := range w.Executions {
		if e.Status == "COMPLETED" || e.Status == "SKIPPED" || e.Status == "FAILED" {
			execs = append(execs, e)
		}
	}

	for i, e := range execs {
		y := startY + i*(barHeight+gap)
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="axis-label" text-anchor="end">%s</text>`, chartLeft-12, y+18, html.EscapeString(e.LanguageID)))

		if e.Status == "COMPLETED" && e.KernelDurationNs > 0 {
			kernelMs := float64(e.KernelDurationNs) / 1e6
			h2dMs := float64(e.TransferH2DNs) / 1e6
			d2hMs := float64(e.TransferD2HNs) / 1e6
			totalMs := kernelMs + h2dMs + d2hMs
			if totalMs <= 0 {
				totalMs = 1
			}

			kW := int((kernelMs / totalMs) * float64(chartWidth))
			hW := int((h2dMs / totalMs) * float64(chartWidth))
			dW := chartWidth - kW - hW

			sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="#22c55e" rx="3 0 0 3"/>`, chartLeft, y, kW, barHeight))
			sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="#f59e0b"/>`, chartLeft+kW, y, hW, barHeight))
			sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="#ef4444" rx="0 3 3 0"/>`, chartLeft+kW+hW, y, dW, barHeight))

			sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="val-label">Kernel: %.2f ms | H2D: %.2f ms | D2H: %.2f ms</text>`, chartLeft+chartWidth+10, y+18, kernelMs, h2dMs, d2hMs))
		} else {
			sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="#334155" rx="3"/>`, chartLeft, y, chartWidth, barHeight))
			note := e.ErrorMessage
			if len(note) > 55 {
				note = note[:52] + "..."
			}
			sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="val-label" fill="#94a3b8">%s: %s</text>`, chartLeft+10, y+18, e.Status, html.EscapeString(note)))
		}
	}

	sb.WriteString("</svg>")
	return sb.String()
}

func renderEmptyChartSVG(width, height int, title, message string) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">
		<rect width="%d" height="%d" rx="8" fill="#0f172a" stroke="#1e293b" stroke-width="1"/>
		<text x="24" y="32" font-family="system-ui, sans-serif" font-size="15" font-weight="700" fill="#f8fafc">%s</text>
		<text x="24" y="52" font-family="system-ui, sans-serif" font-size="11" fill="#94a3b8">%s</text>
		<text x="%d" y="%d" font-family="system-ui, sans-serif" font-size="13" font-weight="500" fill="#64748b" text-anchor="middle">No compatible observations recorded.</text>
	</svg>`, width, height, width, height, width, height, html.EscapeString(title), html.EscapeString(message), width/2, height/2+20)
}

func formatLargeNumber(n float64) string {
	if n >= 1e9 {
		return fmt.Sprintf("%.2fB", n/1e9)
	}
	if n >= 1e6 {
		return fmt.Sprintf("%.1fM", n/1e6)
	}
	if n >= 1e3 {
		return fmt.Sprintf("%.0fK", n/1e3)
	}
	return fmt.Sprintf("%.1f", n)
}
