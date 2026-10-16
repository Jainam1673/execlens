# ExecLens Automated Benchmark Methodology & Reporting Standards

## 1. Provenance and Measurement Integrity

> **"Measure fairly. Compare deeply. Understand why."**

The ExecLens reporting system transforms raw, auditable benchmark observations into peer-reviewable engineering reports and publication-quality vector visualizations.

To ensure transparency and reproducibility, every benchmark report enforces the following scientific principles:

1. **Independent Execution Phases**:
   - **Build Duration (`build_duration_ns`)**: Time required for static code compilation and optimization is tracked separately and never blended into execution runtime.
   - **Correctness Validation**: Before timing samples are accepted, each execution runs a validation pass against a known deterministic contract (e.g. prime counts, sort order checksums, or SHA-256 digests). Unverified or corrupted executions are marked `FAILED` and excluded from speedup rankings.
   - **Warm-Up vs Steady-State**: Runtimes with dynamic JIT compilation (Java HotSpot, Node.js V8) execute dedicated warm-up passes tagged `is_warmup: true`. Only subsequent steady-state iterations are incorporated into statistical summaries.
   - **Kernel MaxRSS Memory**: Physical resident memory is captured via the Linux kernel `getrusage` interface, avoiding unreliable language-internal runtime heuristics.

2. **Statistical Transparency**:
   - Central tendency is primarily reported as the **Median** to prevent distortion by sporadic OS context switches.
   - Sample dispersion is represented via **Sample Standard Deviation ($s$)**, **Interquartile Range ($\text{IQR}$)**, and **Coefficient of Variation ($\text{CV}\%$)**.
   - **Student's t 95% Confidence Intervals** are computed for every candidate:
     $$\text{CI}_{95\%} = \left[\mu - t_{\alpha/2, N-1} \cdot \frac{s}{\sqrt{N}}, \, \mu + t_{\alpha/2, N-1} \cdot \frac{s}{\sqrt{N}}\right]$$
   - Differences where confidence intervals overlap are classified explicitly as `INCONCLUSIVE` or `NEGLIGIBLE_DIFFERENCE`.

3. **Multi-Track Measurement Boundaries**:
   - **Languages Track**: Algorithmic parity across C++, Rust, Go, Java, Python, and JavaScript.
   - **Low-Level Code Lab**: Evaluates scalar vs compiler auto-vectorized loops vs explicit AVX2 SIMD intrinsics vs OpenMP vs inline x86_64 assembly; AoS vs SoA memory layouts; pointer chasing vs contiguous streaming; memory copying mechanisms.
   - **CUDA & GPU Compute Lab**: Strictly isolates pure kernel execution via `cudaEvent_t` from PCIe Host-to-Device (H2D) and Device-to-Host (D2H) data movement. CPU references are compared against both pure kernel and end-to-end wall clock separately.

4. **Honest Reporting & Failure Transparency**:
   - Missing compilers, unsupported devices, or hardware-toolchain capability mismatches (such as newer CUDA toolchains dropping older microarchitectures) are marked `SKIPPED` with exact diagnostic reasons.
   - ExecLens never fabricates measurements, simulates synthetic numbers as real results, or silently substitutes CPU execution for requested GPU runs.

---

## 2. Report Output Architecture

Reports are generated deterministically into the `reports/` directory:

```
reports/
├── latest/                      # Symlink / mirror of most recent report
│   ├── summary.json             # High-level coverage and execution statistics
│   ├── raw-results.json         # Complete, immutable raw iteration observations
│   ├── comparison.csv           # Tabular export for spreadsheets and R/Python
│   ├── manifest.json            # Provenance manifest with SHA-256 artifact checksums
│   ├── report.md                # Detailed Markdown engineering report
│   ├── report.html              # Standalone interactive HTML report
│   └── charts/                  # Vector SVG charts
│       ├── overview.svg
│       ├── runtime-comparison.svg
│       ├── throughput-comparison.svg
│       ├── memory-comparison.svg
│       ├── variability.svg
│       ├── scaling.svg
│       ├── startup-comparison.svg
│       ├── build-comparison.svg
│       ├── workload-heatmap.svg
│       └── <workload_id>.svg
└── history/
    └── rep_<date>_<hash>/       # Preserved historical run records
```

---

## 3. Workflow Commands

- `make benchmark`: Executes benchmark workloads to collect verified data in SQLite.
- `make report`: Runs the automated reporting generator, renders SVG charts, updates `README.md`, and generates manifests.
- `make report-html`: Generates the standalone interactive HTML report.
- `make report-verify`: Audits all generated report files, links, schemas, and README markers.
- `make benchmark-report`: Runs end-to-end benchmark collection, report generation, and verification.
- `make clean-report`: Cleans generated reports.
