package accelerators

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// AcceleratorBackend describes the status and capabilities of an accelerator execution adapter.
type AcceleratorBackend struct {
	ID                string `json:"id"`                 // "cuda", "opencl", "sycl", "hip"
	DisplayName       string `json:"display_name"`        // "NVIDIA CUDA", "OpenCL", "Intel oneAPI SYCL", "AMD ROCm / HIP"
	IsAvailable       bool   `json:"is_available"`
	ToolkitVersion    string `json:"toolkit_version,omitempty"`
	DriverVersion     string `json:"driver_version,omitempty"`
	DeviceName        string `json:"device_name,omitempty"`
	ComputeCapability string `json:"compute_capability,omitempty"`
	MissingReason     string `json:"missing_reason,omitempty"`
	CompilerPath      string `json:"compiler_path,omitempty"`
}

// Registry manages accelerator adapters.
type Registry struct {
	backends map[string]AcceleratorBackend
}

// NewRegistry initializes and detects all accelerator backends.
func NewRegistry(ctx context.Context) *Registry {
	r := &Registry{
		backends: make(map[string]AcceleratorBackend),
	}
	r.detectAll(ctx)
	return r
}

func (r *Registry) detectAll(ctx context.Context) {
	r.backends["cuda"] = detectCUDA(ctx)
	r.backends["opencl"] = detectOpenCL(ctx)
	r.backends["sycl"] = detectSYCL(ctx)
	r.backends["hip"] = detectHIP(ctx)
}

func (r *Registry) GetAll() []AcceleratorBackend {
	var list []AcceleratorBackend
	order := []string{"cuda", "opencl", "sycl", "hip"}
	for _, id := range order {
		if b, ok := r.backends[id]; ok {
			list = append(list, b)
		}
	}
	return list
}

func (r *Registry) Get(id string) (AcceleratorBackend, error) {
	b, ok := r.backends[id]
	if !ok {
		return AcceleratorBackend{}, fmt.Errorf("unknown accelerator backend: %s", id)
	}
	return b, nil
}

func detectCUDA(ctx context.Context) AcceleratorBackend {
	b := AcceleratorBackend{
		ID:          "cuda",
		DisplayName: "NVIDIA CUDA",
		IsAvailable: false,
	}

	nvccPath, err := exec.LookPath("nvcc")
	if err != nil {
		// Also check standard /opt/cuda/bin/nvcc
		if p, err2 := exec.LookPath("/opt/cuda/bin/nvcc"); err2 == nil {
			nvccPath = p
		}
	}

	if nvccPath == "" {
		b.MissingReason = "CUDA compiler driver (nvcc) not found in PATH or /opt/cuda/bin"
		return b
	}
	b.CompilerPath = nvccPath

	// Detect nvcc version
	if verOut, err := exec.CommandContext(ctx, nvccPath, "--version").Output(); err == nil {
		lines := strings.Split(string(verOut), "\n")
		for _, l := range lines {
			if strings.Contains(l, "release") {
				b.ToolkitVersion = strings.TrimSpace(l)
				break
			}
		}
	}

	// Detect GPU device via nvidia-smi
	smiPath, err := exec.LookPath("nvidia-smi")
	if err != nil {
		b.MissingReason = "NVIDIA System Management Interface (nvidia-smi) not found; GPU driver may not be installed"
		return b
	}

	queryOut, err := exec.CommandContext(ctx, smiPath, "--query-gpu=name,driver_version,compute_cap", "--format=csv,noheader").Output()
	if err != nil || len(queryOut) == 0 {
		b.MissingReason = "nvidia-smi failed to query GPU device properties"
		return b
	}

	parts := strings.Split(strings.TrimSpace(string(queryOut)), ",")
	if len(parts) >= 3 {
		b.DeviceName = strings.TrimSpace(parts[0])
		b.DriverVersion = strings.TrimSpace(parts[1])
		b.ComputeCapability = strings.TrimSpace(parts[2])
	}

	// Check architectural compatibility:
	// CUDA 13.x requires compute capability >= 7.5 (Turing). Pascal (6.1) was dropped in CUDA 13.
	if strings.Contains(b.ToolkitVersion, "release 13.") {
		ccFloat, _ := strconv.ParseFloat(b.ComputeCapability, 64)
		if ccFloat < 7.5 {
			b.IsAvailable = false
			b.MissingReason = fmt.Sprintf(
				"CUDA Toolkit 13.x dropped support for pre-Turing architectures (requires CC >= 7.5). Host device '%s' has Compute Capability %s (Pascal), which is unsupported by the installed nvcc.",
				b.DeviceName, b.ComputeCapability,
			)
			return b
		}
	}

	b.IsAvailable = true
	return b
}

func detectOpenCL(ctx context.Context) AcceleratorBackend {
	b := AcceleratorBackend{
		ID:          "opencl",
		DisplayName: "OpenCL",
		IsAvailable: false,
	}

	if clPath, err := exec.LookPath("clinfo"); err == nil {
		out, err := exec.CommandContext(ctx, clPath, "--raw").Output()
		if err == nil && len(out) > 0 {
			b.IsAvailable = true
			b.ToolkitVersion = "OpenCL ICD Loader detected"
			return b
		}
	}

	b.MissingReason = "OpenCL platform headers or clinfo ICD loader not found on host"
	return b
}

func detectSYCL(ctx context.Context) AcceleratorBackend {
	b := AcceleratorBackend{
		ID:          "sycl",
		DisplayName: "Intel oneAPI SYCL",
		IsAvailable: false,
	}

	if dpcppPath, err := exec.LookPath("dpcpp"); err == nil {
		b.IsAvailable = true
		b.CompilerPath = dpcppPath
		b.ToolkitVersion = "Intel DPC++/SYCL Compiler"
		return b
	} else if icxPath, err := exec.LookPath("icx"); err == nil {
		b.IsAvailable = true
		b.CompilerPath = icxPath
		b.ToolkitVersion = "Intel oneAPI ICX"
		return b
	}

	b.MissingReason = "SYCL toolchain (dpcpp or icx) not found on host"
	return b
}

func detectHIP(ctx context.Context) AcceleratorBackend {
	b := AcceleratorBackend{
		ID:          "hip",
		DisplayName: "AMD ROCm / HIP",
		IsAvailable: false,
	}

	if hipccPath, err := exec.LookPath("hipcc"); err == nil {
		b.IsAvailable = true
		b.CompilerPath = hipccPath
		b.ToolkitVersion = "AMD HIP Compiler (hipcc)"
		return b
	}

	b.MissingReason = "AMD ROCm toolkit or hipcc compiler not found on host"
	return b
}
