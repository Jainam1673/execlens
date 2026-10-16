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

## 5. Low-Level CPU Microbenchmarking & Compiler Exploration

The Low-Level Code Lab evaluates performance characteristics at the CPU micro-architecture and compiler optimization levels.

### A. Auto-Vectorization vs Explicit SIMD vs Handwritten Assembly
A common myth in systems programming is that handwritten assembly or explicit SIMD intrinsics are unconditionally faster than modern compiler-generated code. ExecLens benchmarks this empirically:
1. **Scalar Baseline**: Compiled with `-fno-tree-vectorize -fno-vectorize` to establish single-element scalar baseline performance.
2. **Auto-Vectorized Code**: Enables the compiler's auto-vectorizer (`-O3 -mavx2 -mfma`). Modern compilers (GCC and Clang) frequently perform loop unrolling, vector register re-association, and instruction scheduling that can match or exceed hand-coded intrinsics.
3. **Explicit SIMD Intrinsics**: Uses AVX2 / FMA intrinsics (such as `_mm256_fmadd_pd` and `_mm256_loadu_pd`). Demonstrates deterministic vectorization when compiler heuristics fail to auto-vectorize complex control flows.
4. **Inline x86_64 Assembly**: Directly invokes vectorized assembly instructions (`vaddps`, `vmulps`, etc.). Demonstrates register allocation tradeoffs and instruction-level parallelism limitations.

### B. Memory Layout: AoS vs SoA
Data layout dramatically alters cache line utilization:
- **Array of Structures (AoS)**: In particle simulation (`struct { float x, y, z, vx, vy, vz, mass; }`), streaming updates to position (`x += vx * dt`) load entire 32-byte structures into L1 cache lines, but only utilize 16 bytes per element, discarding 50% of the loaded cache bandwidth.
- **Structure of Arrays (SoA)**: Separates fields into independent contiguous arrays (`float x[]`, `float vx[]`). Every cache line loaded contains 100% useful payload for SIMD vector loads (`_mm256_load_ps`), doubling effective memory bandwidth and enabling 8-wide single-instruction updates.

### C. Contiguous Access vs Cache Prefetcher Defeat
Modern CPU hardware prefetchers detect linear and constant-stride memory access patterns, preloading lines into L1/L2 caches before instructions execute:
- **Contiguous Array Streaming**: Linear traversal achieves maximum throughput by saturating memory bandwidth with hardware prefetching.
- **Pseudo-Random Pointer Chasing**: Elements are chained into a Hamiltonian cycle spanning a buffer larger than the CPU Last-Level Cache (LLC). Each dereference depends on the previous load (`curr = curr->next`), completely defeating hardware prefetchers and measuring pure DRAM random access latency (60–80 ns).

### D. Memory Copy Mechanisms & Streaming Stores
Memory copying exhibits stark performance boundaries depending on cache interaction:
- **glibc `memcpy`**: Highly tuned standard library implementation utilizing architecture-specific dynamic dispatch (AVX-512/AVX2 with unrolled block copying).
- **Scalar Byte Copy**: Single-byte `char*` loop, bottlenecked by instruction dispatch and single-byte memory transactions.
- **Unrolled 64-bit Copy**: Copies 8 bytes per iteration with 4x unrolling, reducing loop branch overhead.
- **AVX2 Streaming Non-Temporal Stores (`_mm256_stream_si256`)**:
  Requires strict 32-byte memory alignment (allocated via `posix_memalign`). Uses non-temporal streaming store instructions that bypass the CPU cache hierarchy entirely, writing directly to DRAM via write-combining buffers. This avoids cache pollution when processing multi-megabyte datasets that will not be immediately re-read.

### E. Compiler Optimization Diagnostics
ExecLens compiles C++ sources under GCC and Clang with optimization flags `-O0` through `-Ofast` and captures:
- **Intel-Syntax Assembly (`-S -masm=intel`)**: Preserved for user inspection to understand code generation differences.
- **Vectorization Remarks (`-fopt-info-vec-optimized` / `-Rpass=loop-vectorize`)**: Extracts compiler diagnostics explaining which loops were vectorized, unrolled, or rejected.

---

## 6. CUDA & GPU Measurement Boundaries

Benchmarking GPU execution requires strict separation of asynchronous boundaries. Naive host timers produce fundamentally invalid GPU metrics.

### A. Asynchronous Execution & `cudaEvent_t` Timing
CUDA kernel launches (`kernel<<<grid, block>>>`) are asynchronous non-blocking calls on the host CPU. A standard CPU timer started before the launch and stopped immediately after will measure only the launch overhead (microseconds), not the actual execution on the GPU.
ExecLens uses synchronized GPU event timing:
```cpp
cudaEvent_t start, stop;
cudaEventCreate(&start);
cudaEventCreate(&stop);

cudaEventRecord(start, 0);
kernel<<<grid, block>>>(...);
cudaEventRecord(stop, 0);

cudaEventSynchronize(stop);
float ms = 0.0f;
cudaEventElapsedTime(&ms, start, stop);
```

### B. Isolated Measurement Breakdown
ExecLens breaks down GPU execution into five distinct metrics:
1. **Allocation Duration (`alloc_duration_ns`)**: Time spent allocating and freeing device memory (`cudaMalloc`, `cudaFree`).
2. **Host-to-Device Transfer (`transfer_h2d_ns`)**: PCIe transfer time moving input vectors from system RAM to GPU VRAM (`cudaMemcpyHostToDevice`).
3. **Pure Kernel Duration (`kernel_duration_ns`)**: High-precision `cudaEvent_t` time spent executing computation on GPU Streaming Multiprocessors (SMs).
4. **Device-to-Host Transfer (`transfer_d2h_ns`)**: PCIe transfer time copying verified results back to system memory (`cudaMemcpyDeviceToHost`).
5. **End-to-End Wall-Clock Time**: Total duration encompassing allocation, transfers, kernel execution, and device synchronization.

### C. Effective Bandwidth & GFLOPS Calculation
- **Effective Bandwidth**:
  $$\text{Bandwidth (GB/s)} = \frac{\text{Bytes Read} + \text{Bytes Written}}{\text{Kernel Time (ns)}}$$
- **Computational Throughput**:
  $$\text{GFLOPS} = \frac{\text{Floating Point Operations}}{\text{Kernel Time (ns)}}$$

### D. PCIe Transfer: Pageable vs Pinned Memory
Transfer speed across the PCIe bus depends heavily on host memory allocation:
- **Pageable Host Memory** (`malloc`): The CUDA driver must first copy data into a temporary pinned buffer before initiating Direct Memory Access (DMA) over PCIe, halving effective transfer bandwidth.
- **Pinned Host Memory** (`cudaHostAlloc`): Allocates page-locked physical memory that the GPU DMA engine can access directly without CPU intervention, saturating the physical PCIe bus bandwidth (e.g., 6–12+ GB/s on PCIe Gen3/4).

### E. Fair Heterogeneous CPU vs GPU Comparisons
Comparing CPU and GPU performance is invalid if boundaries are mixed:
- **Kernel-Only vs CPU Algorithm**: Compares the compute efficiency of the GPU Streaming Multiprocessors against the CPU cache and vector ALUs, assuming data already resides in registers/VRAM.
- **End-to-End vs CPU Algorithm**: Accounts for PCIe data movement penalty. For small workloads (e.g., vectors under 100K elements), PCIe transfer overhead frequently exceeds kernel execution, making the CPU faster end-to-end. ExecLens presents both metrics side-by-side so engineers understand the crossover point.

### F. Honest Failure & Hardware Diagnostics
ExecLens never fabricates benchmark numbers or silently substitutes a CPU implementation when a GPU run is requested.
When GPU hardware is missing, drivers are unloaded, or the installed CUDA toolkit version does not support the physical device architecture (such as CUDA 13.3 dropping Compute Capability 6.1 Pascal devices), ExecLens marks the run as `SKIPPED` and provides the exact diagnostic explanation (e.g., `CUDA Toolkit nvcc dropped support for compute_61 architecture`).

---

## 7. Environmental Factors & Limitations

- **Host CPU Throttling**: Turbo boost frequencies and thermal throttling can influence subsequent iterations. ExecLens defaults to single-concurrency execution to prevent core saturation.
- **GPU Clock Scaling & Warm-Up**: GPU boost clocks scale dynamically with temperature and power limits. ExecLens performs explicit warm-up kernel launches to stabilize clock frequencies prior to timed repetitions.
- **OS Page Cache**: File I/O benchmarks measure buffered kernel read throughput and SHA-256 calculation rather than raw physical NVMe/SATA hardware latency.
- **Containerization**: Subprocess execution on a bare host may experience different memory paging characteristics than containerized Docker environments.

