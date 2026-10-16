package accelerators

import (
	"fmt"
	"path/filepath"
)

// AcceleratorWorkload defines a GPU benchmark.
type AcceleratorWorkload struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	SourceFile         string   `json:"source_file"`
	SupportedBackends  []string `json:"supported_backends"`
	DefaultSize        int64    `json:"default_size"`
	UnitName           string   `json:"unit_name"`
	HasCPUReference    bool     `json:"has_cpu_reference"`
	MeasurementMetrics []string `json:"measurement_metrics"`
}

var Catalog = []AcceleratorWorkload{
	{
		ID:                "cuda_vec_add_saxpy",
		Name:              "SAXPY & Vector Addition",
		Description:       "Element-wise vector addition and scaling (Y = a*X + Y) over 16M floats. Compares pure kernel execution vs PCIe transfers and CPU reference.",
		SourceFile:        "vec_add_saxpy.cu",
		SupportedBackends: []string{"cuda"},
		DefaultSize:       16777216,
		UnitName:          "elements",
		HasCPUReference:   true,
		MeasurementMetrics: []string{"kernel_ms", "h2d_ms", "d2h_ms", "alloc_ms", "effective_bandwidth_gbps", "gflops", "cpu_ref_ms"},
	},
	{
		ID:                "cuda_parallel_reduction",
		Name:              "Parallel Reduction (Tree & Warp Shuffle)",
		Description:       "Sum reduction of 16M elements using shared-memory tree reduction and warp shuffle instructions, evaluated against sequential CPU accumulation.",
		SourceFile:        "parallel_reduction.cu",
		SupportedBackends: []string{"cuda"},
		DefaultSize:       16777216,
		UnitName:          "elements",
		HasCPUReference:   true,
		MeasurementMetrics: []string{"kernel_ms", "h2d_ms", "d2h_ms", "alloc_ms", "effective_bandwidth_gbps", "gflops", "cpu_ref_ms"},
	},
	{
		ID:                "cuda_matrix_mul",
		Name:              "Dense Matrix Multiplication (GEMM)",
		Description:       "Matrix multiplication (1024x1024) comparing naive global memory kernel against tiled shared memory kernel (16x16 tiles) and cache-blocked CPU reference.",
		SourceFile:        "matrix_mul.cu",
		SupportedBackends: []string{"cuda"},
		DefaultSize:       1024,
		UnitName:          "dimension",
		HasCPUReference:   true,
		MeasurementMetrics: []string{"kernel_ms", "h2d_ms", "d2h_ms", "alloc_ms", "effective_bandwidth_gbps", "gflops", "cpu_ref_ms"},
	},
	{
		ID:                "cuda_convolution_1d",
		Name:              "1D Stencil Convolution",
		Description:       "1D filter convolution on 8M floats utilizing shared memory with halo cell caching to eliminate redundant DRAM accesses.",
		SourceFile:        "convolution_1d.cu",
		SupportedBackends: []string{"cuda"},
		DefaultSize:       8388608,
		UnitName:          "elements",
		HasCPUReference:   true,
		MeasurementMetrics: []string{"kernel_ms", "h2d_ms", "d2h_ms", "alloc_ms", "effective_bandwidth_gbps", "gflops", "cpu_ref_ms"},
	},
	{
		ID:                "cuda_histogram_scan",
		Name:              "256-Bin Histogram & Atomic Reduction",
		Description:       "Shared-memory atomic histogram accumulation over 16MB of data, avoiding global memory contention bottlenecks.",
		SourceFile:        "histogram_scan.cu",
		SupportedBackends: []string{"cuda"},
		DefaultSize:       16777216,
		UnitName:          "bytes",
		HasCPUReference:   true,
		MeasurementMetrics: []string{"kernel_ms", "h2d_ms", "d2h_ms", "alloc_ms", "effective_bandwidth_gbps", "gflops", "cpu_ref_ms"},
	},
	{
		ID:                "cuda_bandwidth_transfer",
		Name:              "PCIe Transfer Bandwidth (Pageable vs Pinned)",
		Description:       "Measures Host-to-Device and Device-to-Host transfer throughput over 64MB buffers comparing standard pageable host memory against page-locked pinned memory (cudaHostAlloc).",
		SourceFile:        "bandwidth_transfer.cu",
		SupportedBackends: []string{"cuda"},
		DefaultSize:       67108864,
		UnitName:          "bytes",
		HasCPUReference:   false,
		MeasurementMetrics: []string{"pinned_h2d_gbps", "pinned_d2h_gbps", "pageable_h2d_gbps", "pageable_d2h_gbps"},
	},
}

// GetWorkload looks up an accelerator workload by ID.
func GetWorkload(id string) (*AcceleratorWorkload, error) {
	for i := range Catalog {
		if Catalog[i].ID == id {
			return &Catalog[i], nil
		}
	}
	return nil, fmt.Errorf("unknown accelerator workload: %s", id)
}

// SourcePath returns absolute path to CUDA source file.
func (w *AcceleratorWorkload) SourcePath(benchmarksDir string) string {
	return filepath.Join(benchmarksDir, "cuda", w.SourceFile)
}
