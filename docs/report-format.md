# ExecLens Report Format & Data Schema Specification

This document specifies the file structures, JSON schemas, CSV formats, and finding data models used by the ExecLens Automated Reporting System.

---

## 1. Directory Structure

```
reports/latest/
├── summary.json          # High-level aggregate metrics and execution coverage
├── raw-results.json      # Complete raw run records and iteration samples
├── comparison.csv        # Normalized comparison table for external analysis
├── manifest.json         # Cryptographic provenance manifest
├── report.md             # Reviewable Markdown report
├── report.html           # Standalone self-contained HTML report
└── charts/               # Publication-grade vector SVGs
    ├── overview.svg
    ├── runtime-comparison.svg
    ├── throughput-comparison.svg
    ├── memory-comparison.svg
    ├── variability.svg
    ├── scaling.svg
    ├── startup-comparison.svg
    ├── build-comparison.svg
    ├── workload-heatmap.svg
    └── <workload_id>.svg
```

---

## 2. File Specifications

### 2.1 `summary.json`

Provides high-level counters, coverage, and environment fingerprint.

```json
{
  "report_id": "rep_20261017_a1b2c3d4",
  "generated_at": "2026-10-17T03:45:00Z",
  "generator_version": "1.0.0",
  "environment": {
    "os": "linux",
    "arch": "amd64",
    "cpu_model": "Intel Core i7-10510U CPU @ 1.80GHz",
    "logical_cpus": 8,
    "total_memory_bytes": 16500000000
  },
  "total_configurations": 18,
  "successful_executions": 15,
  "failed_executions": 0,
  "skipped_executions": 3,
  "validation_failures": 0,
  "inconclusive_count": 2,
  "statistically_significant_count": 10,
  "workloads_covered": ["lowlevel_reduction", "sieve", "sort"],
  "languages_tested": ["cpp", "go", "java", "javascript", "python", "rust"],
  "tracks_covered": ["languages", "lowlevel"],
  "run_ids": ["run_593d6a685947", "run_ad2a14f69db8", "run_d09a1707b89f"]
}
```

### 2.2 `manifest.json`

Records cryptographic SHA-256 checksums of every generated artifact to guarantee provenance and detect tampering.

```json
{
  "manifest_version": "1.0.0",
  "report_id": "rep_20261017_a1b2c3d4",
  "created_at": "2026-10-17T03:45:00Z",
  "git_revision": "7e53c5e",
  "host_fingerprint": { ... },
  "statistical_config": {
    "central_tendency": "Median",
    "confidence_interval": "Student's t 95%",
    "practical_threshold_percent": 5.0
  },
  "workloads_covered": ["sieve", "sort", "lowlevel_reduction"],
  "run_ids": ["run_ad2a14f69db8", "run_d09a1707b89f"],
  "artifacts": [
    {
      "path": "charts/overview.svg",
      "size_bytes": 4820,
      "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    },
    {
      "path": "report.md",
      "size_bytes": 12450,
      "sha256": "5f4dcc3b5aa765d61d8327deb882cf99..."
    }
  ]
}
```

### 2.3 `comparison.csv`

Tabular data export suitable for ingestion by Python Pandas, R, or spreadsheet applications.

Columns:
1. `Workload`: Workload identifier (`sieve`, `sort`, `lowlevel_reduction`, etc.)
2. `Track`: Execution track (`languages`, `lowlevel`, `gpu`)
3. `Language`: Implementation / variant (`cpp`, `rust`, `scalar`, `avx2_stream`)
4. `Status`: Execution outcome (`COMPLETED`, `SKIPPED`, `FAILED`)
5. `Median_ms`: Median execution duration in milliseconds
6. `Mean_ms`: Arithmetic mean duration in milliseconds
7. `CI95_Lower_ms`: Lower bound of Student's t 95% Confidence Interval
8. `CI95_Upper_ms`: Upper bound of Student's t 95% Confidence Interval
9. `StdDev_ms`: Sample standard deviation in milliseconds
10. `CV_percent`: Normalized Coefficient of Variation ($\frac{s}{\mu} \times 100$)
11. `Peak_Memory_MB`: Peak physical resident set size (MaxRSS) in megabytes
12. `Throughput_ops_sec`: Effective throughput operations per second
13. `Speedup_vs_Baseline`: Normalized speedup ratio ($\frac{\text{Baseline Median}}{\text{Candidate Median}}$)
14. `Toolchain`: Compiler / runtime version string

---

## 3. Deterministic Findings Data Model

Each programmatic finding follows this schema:

| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | string | Unique deterministic identifier (e.g. `find_001`) |
| `title` | string | Short, descriptive finding statement |
| `category` | string | `TOP_PERFORMER`, `SPEEDUP`, `MEMORY_EFFICIENCY`, `VARIABILITY`, `COMPILER_OPTIMIZATION`, `ACCELERATOR_TELEMETRY`, `INCONCLUSIVE`, `DIAGNOSTIC_NOTE` |
| `workload_id` | string | Workload scope |
| `track` | string | Benchmark track |
| `scope` | string | Algorithmic mode or comparison boundary |
| `baseline_id` | string | Identified baseline for comparison |
| `candidate_id` | string | Implementation evaluated |
| `metric_name` | string | Evaluated metric (e.g. `Median Execution Time`) |
| `metric_unit` | string | Metric unit (`ms`, `MB`, `ops/s`, `x`, `%`) |
| `baseline_value` | float | Measured baseline central value |
| `candidate_value` | float | Measured candidate central value |
| `effect_size` | string | Quantified divergence (e.g. `1.45x`, `12.4 MB`) |
| `sample_count` | int | Iteration count ($N$) |
| `uncertainty` | string | Uncertainty bounds (e.g. `95% CI [23.1, 24.8] ms`) |
| `status` | string | Significance classification (`STATISTICALLY_SIGNIFICANT`, `PRACTICAL_DIFFERENCE`, `NEGLIGIBLE_DIFFERENCE`, `INCONCLUSIVE`) |
| `explanation` | string | Neutral, empirical explanation of the finding |
| `limitations` | string | Confounding factors and measurement limits |

---

## 4. README.md Integration Markers

README integration updates only content bounded by the markers:

```markdown
<!-- EXECLENS_REPORT_START -->
... Generated content inserted here ...
<!-- EXECLENS_REPORT_END -->
```

Any edits outside of these boundary tags are strictly preserved.
