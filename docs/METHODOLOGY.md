# ExecLens Benchmarking Methodology & Statistical Rigor

## 1. Benchmarking Philosophy

> **"Measure fairly. Compare deeply. Understand why."**

Benchmark results frequently mislead developers when they present simplistic, single-number rankings without context. ExecLens rejects claims that any programming language is "universally faster." Real-world execution performance is a function of:
- Specific algorithmic structures and data layouts.
- Hardware architecture, memory hierarchy, and cache behavior.
- Compiler optimization pipelines and code generation targets.
- Runtime execution models (Ahead-of-Time compilation, Just-in-Time compilation, or bytecode interpretation).
- Memory management and garbage collection strategies.

ExecLens reports measured empirical outcomes for defined workloads under recorded conditions, documenting trade-offs and measurement limitations honestly.

---

## 2. Measurement Isolation & Phases

### A. Separation of Compilation and Execution
In naive benchmark suites, compilation overhead or script loading is often lumped into runtime execution. ExecLens separates the lifecycle into explicit, independently recorded phases:

1. **Build Duration (`build_duration_ns`)**:
   Tracks source preparation, static analysis, and code generation (e.g. `g++ -O3`, `rustc -C opt-level=3`, `go build`, `javac`). Build time is isolated and never included in steady-state execution numbers.
2. **Correctness Validation**:
   Before recording timing data, each binary is executed in a validation pass against a deterministic expected result (e.g., matching prime count, sorting order validation, or cryptographic checksum). If validation fails, execution is halted and marked `FAILED` without accepting timing measurements.
3. **Warm-up Duration**:
   Managed runtimes with JIT compilers (such as Java's HotSpot C2 compiler and Node's V8 TurboFan) optimize bytecode dynamically based on profiling data. ExecLens provides explicit warm-up iterations (default: 2 for Java and Node, 0 for native C++, Rust, and Go) recorded separately with `is_warmup: true`. A dedicated **Cold Start** mode is available to measure initial un-optimized execution.
4. **Timed Execution Iterations**:
   The steady-state repetitions measured via high-precision monotonic timers (`CLOCK_MONOTONIC`).

### B. Memory Measurement Methodology
Peak resident memory is queried directly from the Linux kernel using the standard process usage statistics (`getrusage` / `ProcessState.SysUsage().(*syscall.Rusage).Maxrss`). On Linux, this captures the maximum Resident Set Size (RSS) in physical memory allocated by the process across its lifetime. Memory ratios are computed against an identified baseline only when valid physical measurements exist.

---

## 3. Statistical Analysis & Confidence Intervals

ExecLens avoids relying solely on the arithmetic mean, which is vulnerable to external OS scheduling spikes, page faults, or background interruptions.

### A. Central Tendency
- **Median ($M$)**: The primary central statistic reported, representing the 50th percentile of observations.
- **Mean ($\mu$)**: The arithmetic average $\mu = \frac{1}{N}\sum_{i=1}^N x_i$.

### B. Dispersion & Variability
- **Sample Standard Deviation ($s$)**:
  $$s = \sqrt{\frac{1}{N-1}\sum_{i=1}^N (x_i - \mu)^2}$$
- **Coefficient of Variation ($\text{CV}\%$)**:
  $$\text{CV} = \left(\frac{s}{\mu}\right) \times 100$$
  Provides a normalized indicator of measurement stability. Runs with $\text{CV} > 15\%$ indicate high environmental noise or GC pause variance.
- **Interquartile Range ($\text{IQR}$)**: The difference between the 75th percentile ($Q_3$) and 25th percentile ($Q_1$).

### C. 95% Confidence Intervals
ExecLens calculates the 95% Confidence Interval for the mean using the Student's $t$-distribution critical value $t_{0.025, N-1}$:
$$\text{Margin of Error} = t_{\alpha/2, N-1} \cdot \left(\frac{s}{\sqrt{N}}\right)$$
$$\text{CI}_{95\%} = [\mu - \text{Margin of Error}, \mu + \text{Margin of Error}]$$

### D. Qualitative Significance Categorization
ExecLens evaluates whether observed differences between a candidate and baseline represent meaningful performance divergences:

1. **`NEGLIGIBLE_DIFFERENCE`**:
   The median difference is below the practical significance threshold (5%), or 95% confidence intervals overlap with negligible magnitude.
2. **`INCONCLUSIVE`**:
   The median difference exceeds 5%, but the 95% confidence intervals overlap due to sample variance (noise).
3. **`STATISTICALLY_SIGNIFICANT`**:
   The candidate and baseline exhibit non-overlapping 95% confidence intervals, and the median difference exceeds the practical threshold.
4. **`PRACTICAL_DIFFERENCE`**:
   Clear divergence exceeding practical engineering significance.

---

## 4. Algorithmic Parity Across Languages

ExecLens designs workloads so that implementations perform equivalent algorithmic operations:

- **Sieve of Eratosthenes**:
  All language implementations allocate an 8-bit byte array (1 byte per integer candidate), avoiding bit-packed vectors (such as C++ `std::vector<bool>`) to preserve algorithmic parity in memory access patterns.
- **Integer Sorting**:
  ExecLens distinguishes two explicit modes:
  - `quicksort`: Implements strictly identical in-place median-of-three quicksort implementations across all languages.
  - `stdlib`: Evaluates the runtime standard library sort (pdqsort in Rust, introsort in C++, dual-pivot quicksort in Java, timsort in Python, V8 sort in Node).
- **Hash Table Operations**:
  Evaluates standard associative containers (`std::unordered_map`, `HashMap`, `map`, `dict`, `Map`). Differences in hash functions (SipHash, AES-NI, Murmur) are documented as genuine runtime characteristics.
- **JSON Processing**:
  Synthetic JSON records are generated with matching schema, parsed, filtered (`active == true`), and aggregated with matching arithmetic precision.
- **Matrix Multiplication**:
  $O(N^3)$ standard three-loop multiplication on contiguous row-major floating point arrays, isolating compiler loop vectorization and cache locality.
- **File Processing Stream**:
  Processes generated binary streams in standard 64KB buffers, hashing via standard SHA-256 digests. Page cache effects are noted honestly.

---

## 5. Environmental Factors & Limitations

- **Host CPU Throttling**: Turbo boost frequencies and thermal throttling can influence subsequent iterations. ExecLens defaults to single-concurrency execution to prevent core saturation.
- **OS Page Cache**: File I/O benchmarks measure buffered kernel read throughput and SHA-256 calculation rather than raw physical NVMe/SATA hardware latency.
- **Containerization**: Subprocess execution on a bare host may experience different memory paging characteristics than containerized Docker environments.
