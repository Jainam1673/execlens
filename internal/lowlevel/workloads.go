package lowlevel

import (
	"fmt"
	"path/filepath"
)

// LowLevelWorkload defines a benchmark in the Low-Level Code Lab.
type LowLevelWorkload struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	SourceFile  string            `json:"source_file"`
	Variants    []WorkloadVariant `json:"variants"`
	DefaultSize int64             `json:"default_size"`
	UnitName    string            `json:"unit_name"`
}

type WorkloadVariant struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var Catalog = []LowLevelWorkload{
	{
		ID:          "lowlevel_reduction",
		Name:        "Array Reduction & Vectorization",
		Description: "Sum reduction of 10M doubles comparing scalar loop, auto-vectorization, explicit AVX2 SIMD intrinsics, OpenMP, and inline assembly",
		SourceFile:  "reduction.cpp",
		DefaultSize: 10000000,
		UnitName:    "elements",
		Variants: []WorkloadVariant{
			{ID: "scalar", Name: "Scalar Loop", Description: "Loop with compiler vectorization explicitly disabled"},
			{ID: "autovec", Name: "Auto-Vectorized", Description: "Standard loop optimized by compiler vectorizer"},
			{ID: "simd_avx2", Name: "AVX2 SIMD Intrinsics", Description: "Hand-crafted 256-bit AVX2 vector intrinsics with 4 accumulators"},
			{ID: "openmp", Name: "Parallel OpenMP", Description: "Multi-threaded CPU reduction using OpenMP pragma"},
			{ID: "asm", Name: "Inline Assembly", Description: "Handwritten x86_64 AVX vector assembly loop"},
		},
	},
	{
		ID:          "lowlevel_dot_product",
		Name:        "Vector Dot Product (FMA)",
		Description: "Double-precision dot product comparing scalar, auto-vectorized, and AVX2+FMA fused multiply-add intrinsics",
		SourceFile:  "dot_product.cpp",
		DefaultSize: 10000000,
		UnitName:    "elements",
		Variants: []WorkloadVariant{
			{ID: "scalar", Name: "Scalar Loop", Description: "Scalar multiply-accumulate without vectorization"},
			{ID: "autovec", Name: "Auto-Vectorized", Description: "Compiler-vectorized dot product loop"},
			{ID: "simd_fma", Name: "AVX2 + FMA SIMD", Description: "Explicit _mm256_fmadd_pd intrinsics with 4-way unrolling"},
		},
	},
	{
		ID:          "lowlevel_layout",
		Name:        "Memory Layout: AoS vs SoA",
		Description: "3D particle physics update comparing Array-of-Structures (AoS) cache strides against Structure-of-Arrays (SoA) contiguous SIMD vectorization",
		SourceFile:  "layout_memory.cpp",
		DefaultSize: 5000000,
		UnitName:    "particles",
		Variants: []WorkloadVariant{
			{ID: "aos", Name: "Array of Structures (AoS)", Description: "Interleaved 32-byte struct layout Particle[N]"},
			{ID: "soa", Name: "Structure of Arrays (SoA)", Description: "Separated contiguous arrays float x[N], y[N], z[N]"},
		},
	},
	{
		ID:          "lowlevel_pointer_chase",
		Name:        "Memory Access: Contiguous vs Pointer Chasing",
		Description: "Evaluates CPU hardware prefetcher efficiency: sequential array scanning vs pseudo-random cycle pointer chasing across L1/L2/L3 caches and DRAM",
		SourceFile:  "pointer_chase.cpp",
		DefaultSize: 4000000,
		UnitName:    "elements",
		Variants: []WorkloadVariant{
			{ID: "contiguous", Name: "Contiguous Array", Description: "Sequential linear memory scan with hardware prefetching"},
			{ID: "pointer_chasing", Name: "Pointer Chasing", Description: "Unpredictable cycle node traversal defeating prefetchers"},
		},
	},
	{
		ID:          "lowlevel_memcpy",
		Name:        "Memory Bandwidth & Non-Temporal Streaming",
		Description: "Memory copy throughput comparing glibc memcpy, scalar byte loop, 64-bit unrolled copy, and AVX2 non-temporal streaming stores (_mm256_stream_si256)",
		SourceFile:  "memory_copy.cpp",
		DefaultSize: 64000000,
		UnitName:    "bytes",
		Variants: []WorkloadVariant{
			{ID: "libc_memcpy", Name: "glibc memcpy", Description: "System library optimized memory copy"},
			{ID: "scalar_loop", Name: "Scalar Loop", Description: "Byte-by-byte scalar copy loop"},
			{ID: "unroll64", Name: "64-bit Unrolled", Description: "64-bit uint64 unrolled loop"},
			{ID: "simd_streaming", Name: "AVX2 Streaming Store", Description: "Non-temporal cache-bypassing stores (_mm256_stream_si256)"},
		},
	},
}

// GetWorkload looks up a low-level workload by ID.
func GetWorkload(id string) (*LowLevelWorkload, error) {
	for i := range Catalog {
		if Catalog[i].ID == id {
			return &Catalog[i], nil
		}
	}
	return nil, fmt.Errorf("unknown low-level workload: %s", id)
}

// SourcePath returns absolute path to source file within benchmarks/lowlevel.
func (w *LowLevelWorkload) SourcePath(benchmarksDir string) string {
	return filepath.Join(benchmarksDir, "lowlevel", w.SourceFile)
}
