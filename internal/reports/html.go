package reports

import (
	"fmt"
	"html"
	"strings"
)

// GenerateHTMLReport produces a standalone, responsive, self-contained HTML report.
func GenerateHTMLReport(summary *ReportSummary, manifest *ReportManifest, workloads []WorkloadReportData, findings []ReportFinding) string {
	var sb strings.Builder

	dateStr := summary.GeneratedAt.UTC().Format("2006-01-02 15:04:05 UTC")

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>ExecLens — Benchmark Engineering Report</title>
  <style>
    :root {
      --bg: #0b0f19;
      --card-bg: #111827;
      --border: #1f2937;
      --border-accent: #374151;
      --text: #f9fafb;
      --text-muted: #9ca3af;
      --accent: #38bdf8;
      --accent-green: #22c55e;
      --accent-red: #ef4444;
      --accent-amber: #f59e0b;
      --accent-purple: #a855f7;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      background-color: var(--bg);
      color: var(--text);
      line-height: 1.6;
      padding: 0 1rem 3rem 1rem;
    }
    .container { max-width: 1200px; margin: 0 auto; }
    header {
      padding: 2.5rem 0 1.5rem 0;
      border-bottom: 1px solid var(--border);
      margin-bottom: 2rem;
    }
    .badge {
      display: inline-block;
      padding: 0.25rem 0.6rem;
      border-radius: 9999px;
      font-size: 0.75rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.5px;
      background: #1e293b;
      color: var(--accent);
      border: 1px solid #334155;
      margin-bottom: 0.75rem;
    }
    h1 { font-size: 2.25rem; font-weight: 800; color: #fff; margin-bottom: 0.5rem; letter-spacing: -0.5px; }
    .tagline { color: var(--text-muted); font-size: 1.1rem; }
    .meta-bar {
      display: flex;
      flex-wrap: wrap;
      gap: 1.5rem;
      margin-top: 1.25rem;
      font-size: 0.875rem;
      color: var(--text-muted);
    }
    .meta-bar strong { color: var(--text); }
    .nav-tabs {
      display: flex;
      gap: 0.5rem;
      border-bottom: 1px solid var(--border);
      margin-bottom: 2rem;
      overflow-x: auto;
    }
    .tab-btn {
      background: none;
      border: none;
      padding: 0.75rem 1.25rem;
      font-size: 0.95rem;
      font-weight: 600;
      color: var(--text-muted);
      cursor: pointer;
      border-bottom: 2px solid transparent;
      transition: all 0.2s;
    }
    .tab-btn:hover { color: var(--text); }
    .tab-btn.active { color: var(--accent); border-bottom-color: var(--accent); }
    .tab-content { display: none; }
    .tab-content.active { display: block; }
    .stats-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
      gap: 1rem;
      margin-bottom: 2rem;
    }
    .stat-card {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 0.5rem;
      padding: 1.25rem;
    }
    .stat-label { font-size: 0.75rem; text-transform: uppercase; color: var(--text-muted); font-weight: 600; }
    .stat-val { font-size: 1.85rem; font-weight: 700; color: #fff; margin-top: 0.25rem; }
    .card {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 0.5rem;
      padding: 1.5rem;
      margin-bottom: 2rem;
    }
    .card h2 { font-size: 1.35rem; font-weight: 700; margin-bottom: 1rem; color: #fff; }
    .card h3 { font-size: 1.1rem; font-weight: 600; margin: 1.25rem 0 0.75rem 0; color: #e2e8f0; }
    .chart-container {
      background: #0f172a;
      border: 1px solid var(--border);
      border-radius: 0.5rem;
      padding: 1rem;
      margin: 1.25rem 0;
      text-align: center;
      overflow-x: auto;
    }
    .chart-container img { max-width: 100%; height: auto; }
    table {
      width: 100%;
      border-collapse: collapse;
      margin: 1rem 0;
      font-size: 0.875rem;
    }
    th {
      text-align: left;
      padding: 0.75rem 1rem;
      background: #1f2937;
      color: #9ca3af;
      font-weight: 600;
      border-bottom: 1px solid var(--border-accent);
    }
    td {
      padding: 0.75rem 1rem;
      border-bottom: 1px solid var(--border);
      color: #e5e7eb;
    }
    tr:hover td { background: #1a2234; }
    .finding-card {
      background: #131d2e;
      border-left: 4px solid var(--accent);
      border-radius: 0 0.5rem 0.5rem 0;
      padding: 1.25rem;
      margin-bottom: 1rem;
    }
    .finding-card.stat-sig { border-left-color: var(--accent-green); }
    .finding-card.inconclusive { border-left-color: var(--accent-amber); }
    .finding-title { font-weight: 700; font-size: 1.05rem; color: #fff; margin-bottom: 0.25rem; }
    .finding-meta { font-size: 0.8rem; color: var(--text-muted); margin-bottom: 0.5rem; }
    .finding-desc { font-size: 0.9rem; color: #cbd5e1; }
    .finding-limitation { font-size: 0.8rem; color: #94a3b8; font-style: italic; margin-top: 0.5rem; }
    footer {
      margin-top: 3rem;
      padding-top: 1.5rem;
      border-top: 1px solid var(--border);
      text-align: center;
      font-size: 0.85rem;
      color: var(--text-muted);
    }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <span class="badge">ExecLens Performance Lab</span>
      <h1>Automated Benchmark Engineering Report</h1>
      <p class="tagline">Measure fairly. Compare deeply. Understand why.</p>
      <div class="meta-bar">`)

	sb.WriteString(fmt.Sprintf(`<span><strong>Report ID:</strong> %s</span>`, html.EscapeString(summary.ReportID)))
	sb.WriteString(fmt.Sprintf(`<span><strong>Date:</strong> %s</span>`, dateStr))
	sb.WriteString(fmt.Sprintf(`<span><strong>Host:</strong> %s (%s, %d vCPUs)</span>`, html.EscapeString(summary.Environment.CPUModel), summary.Environment.OS, summary.Environment.LogicalCPUs))
	sb.WriteString(fmt.Sprintf(`<span><strong>Git Revision:</strong> %s</span>`, manifest.GitRevision))
	sb.WriteString(fmt.Sprintf(`<span><strong>Version:</strong> v%s</span>`, summary.GeneratorVersion))

	sb.WriteString(`</div>
    </header>

    <nav class="nav-tabs">
      <button class="tab-btn active" onclick="openTab('summary')">Executive Summary</button>
      <button class="tab-btn" onclick="openTab('findings')">Deterministic Findings</button>
      <button class="tab-btn" onclick="openTab('workloads')">Workload Breakdown</button>
      <button class="tab-btn" onclick="openTab('methodology')">Methodology &amp; Provenance</button>
    </nav>

    <!-- Tab 1: Executive Summary -->
    <div id="tab-summary" class="tab-content active">
      <div class="stats-grid">`)

	sb.WriteString(fmt.Sprintf(`
        <div class="stat-card">
          <div class="stat-label">Total Configurations</div>
          <div class="stat-val" style="color: #38bdf8;">%d</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Successful Executions</div>
          <div class="stat-val" style="color: #4ade80;">%d</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Statistically Significant</div>
          <div class="stat-val" style="color: #a78bfa;">%d</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Skipped / Inconclusive</div>
          <div class="stat-val" style="color: #fbbf24;">%d</div>
        </div>`,
		summary.TotalConfigurations, summary.SuccessfulExecutions, summary.StatSignificantCount, summary.SkippedExecutions+summary.InconclusiveCount))

	sb.WriteString(`</div>
      <div class="card">
        <h2>Benchmark Execution Coverage</h2>
        <div class="chart-container">
          <img src="charts/overview.svg" alt="ExecLens Execution Overview">
        </div>
      </div>

      <div class="card">
        <h2>Cross-Workload Normalized Speedup Heatmap</h2>
        <p class="tagline" style="font-size: 0.9rem; margin-bottom: 1rem;">Normalized speedup matrix relative to baseline. Green represents accelerated execution; red denotes slower execution.</p>
        <div class="chart-container">
          <img src="charts/workload-heatmap.svg" alt="Cross-Workload Heatmap">
        </div>
      </div>
    </div>

    <!-- Tab 2: Deterministic Findings -->
    <div id="tab-findings" class="tab-content">
      <div class="card">
        <h2>Programmatically Derived Findings</h2>
        <p class="tagline" style="font-size: 0.9rem; margin-bottom: 1.5rem;">Derived strictly from verified observations and non-overlapping 95% Confidence Intervals:</p>`)

	for _, f := range findings {
		cardClass := "finding-card"
		if f.Status == "STATISTICALLY_SIGNIFICANT" {
			cardClass += " stat-sig"
		} else if f.Status == "INCONCLUSIVE" {
			cardClass += " inconclusive"
		}

		sb.WriteString(fmt.Sprintf(`
        <div class="%s">
          <div class="finding-title">%s</div>
          <div class="finding-meta">ID: <code>%s</code> | Workload: <code>%s</code> | Status: <strong>%s</strong> | Effect: <strong>%s</strong></div>
          <div class="finding-desc">%s</div>
          <div class="finding-limitation">Caveat: %s</div>
        </div>`,
			cardClass, html.EscapeString(f.Title), f.ID, f.WorkloadID, f.Status, html.EscapeString(f.EffectSize), html.EscapeString(f.Explanation), html.EscapeString(f.Limitations)))
	}

	sb.WriteString(`</div>
    </div>

    <!-- Tab 3: Workload Breakdown -->
    <div id="tab-workloads" class="tab-content">`)

	for _, w := range workloads {
		sb.WriteString(fmt.Sprintf(`
      <div class="card">
        <h2>%s (<code>%s</code>)</h2>
        <div class="meta-bar" style="margin-top: 0; margin-bottom: 1rem;">
          <span><strong>Track:</strong> %s</span>
          <span><strong>Mode:</strong> %s</span>
          <span><strong>Baseline:</strong> %s</span>
        </div>
        <div class="chart-container">
          <img src="charts/%s.svg" alt="%s Chart">
        </div>
        <table>
          <thead>
            <tr>
              <th>Implementation</th>
              <th>Status</th>
              <th>Median (ms)</th>
              <th>Mean (ms)</th>
              <th>95%% CI (ms)</th>
              <th>StdDev (CV%%)</th>
              <th>Peak RSS</th>
              <th>Throughput</th>
              <th>Relative Speedup</th>
            </tr>
          </thead>
          <tbody>`,
			html.EscapeString(w.WorkloadName), w.WorkloadID, w.Track, w.ExecutionMode, w.BaselineID, w.WorkloadID, html.EscapeString(w.WorkloadName)))

		for _, e := range w.Executions {
			if e.Summary != nil && e.Status == "COMPLETED" {
				speedupStr := "1.00x (Baseline)"
				if e.LanguageID != w.BaselineID && e.Speedup > 0 {
					speedupStr = fmt.Sprintf("<strong>%.2fx</strong>", e.Speedup)
				}
				thStr := "N/A"
				if e.Summary.ThroughputOpsPerSec > 0 {
					thStr = fmt.Sprintf("%.0f ops/s", e.Summary.ThroughputOpsPerSec)
				} else if e.Summary.ThroughputMBPerSec > 0 {
					thStr = fmt.Sprintf("%.1f MB/s", e.Summary.ThroughputMBPerSec)
				}

				sb.WriteString(fmt.Sprintf(`
            <tr>
              <td><strong>%s</strong></td>
              <td>%s</td>
              <td>%.2f</td>
              <td>%.2f</td>
              <td>[%.2f, %.2f]</td>
              <td>±%.2f (%.1f%%)</td>
              <td>%.1f MB</td>
              <td>%s</td>
              <td>%s</td>
            </tr>`,
					e.LanguageID, e.Status, e.Summary.MedianMs, e.Summary.MeanMs,
					e.Summary.CI95LowerMs, e.Summary.CI95UpperMs, e.Summary.StdDevMs, e.Summary.CVPercent,
					e.PeakMemoryMB, thStr, speedupStr))
			} else {
				sb.WriteString(fmt.Sprintf(`
            <tr>
              <td><strong>%s</strong></td>
              <td>%s</td>
              <td colspan="7"><em>%s</em></td>
            </tr>`, e.LanguageID, e.Status, html.EscapeString(e.ErrorMessage)))
			}
		}

		sb.WriteString(`</tbody>
        </table>
      </div>`)
	}

	sb.WriteString(`</div>

    <!-- Tab 4: Methodology & Provenance -->
    <div id="tab-methodology" class="tab-content">
      <div class="card">
        <h2>Measurement Methodology &amp; Reproducibility</h2>
        <p>ExecLens guarantees benchmark provenance through controlled isolation:</p>
        <ul style="margin: 1rem 0 1rem 2rem; color: #cbd5e1;">
          <li><strong>Phase Isolation:</strong> Build time, correctness validation, warm-up iterations, and steady-state runs are timed independently.</li>
          <li><strong>Kernel MaxRSS:</strong> Peak physical memory is queried via Linux kernel <code>getrusage</code>.</li>
          <li><strong>Statistical Rigor:</strong> Student's t 95% Confidence Intervals are calculated for all mean durations.</li>
          <li><strong>Truthful Hardware Diagnostics:</strong> Missing or architecture-incompatible accelerators are logged transparently without silent fallback.</li>
        </ul>
        <h3>Reproducibility Manifest</h3>
        <p>The manifest contains SHA-256 cryptographic checksums of all generated charts, CSV tables, and raw JSON observations.</p>
        <div style="margin-top: 1rem;">
          <a href="manifest.json" style="color: var(--accent); margin-right: 1.5rem;">Download Manifest (JSON)</a>
          <a href="raw-results.json" style="color: var(--accent); margin-right: 1.5rem;">Raw Observations (JSON)</a>
          <a href="comparison.csv" style="color: var(--accent);">Tabular Results (CSV)</a>
        </div>
      </div>
    </div>

    <footer>
      <p>Generated by <strong>ExecLens Automated Benchmark Reporting System</strong> | Measure fairly. Compare deeply. Understand why.</p>
    </footer>
  </div>

  <script>
    function openTab(tabName) {
      document.querySelectorAll('.tab-content').forEach(el => el.classList.remove('active'));
      document.querySelectorAll('.tab-btn').forEach(el => el.classList.remove('active'));
      document.getElementById('tab-' + tabName).classList.add('active');
      event.target.classList.add('active');
    }
  </script>
</body>
</html>`)

	return sb.String()
}
