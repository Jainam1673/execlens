package lowlevel

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// CompilerInfo holds detected compiler availability and version.
type CompilerInfo struct {
	Name        string `json:"name"`        // "gcc" or "clang"
	Executable  string `json:"executable"`  // "g++" or "clang++"
	IsAvailable bool   `json:"is_available"`
	Version     string `json:"version"`
	Path        string `json:"path"`
}

// DetectCompilers probes for GCC (g++) and Clang (clang++) on the host.
func DetectCompilers(ctx context.Context) map[string]CompilerInfo {
	res := make(map[string]CompilerInfo)

	// Probe GCC
	gccPath, err := exec.LookPath("g++")
	if err == nil {
		out, err := exec.CommandContext(ctx, gccPath, "--version").Output()
		ver := "unknown"
		if err == nil && len(out) > 0 {
			ver = strings.Split(string(out), "\n")[0]
		}
		res["gcc"] = CompilerInfo{
			Name:        "gcc",
			Executable:  "g++",
			IsAvailable: true,
			Version:     ver,
			Path:        gccPath,
		}
	} else {
		res["gcc"] = CompilerInfo{
			Name:        "gcc",
			Executable:  "g++",
			IsAvailable: false,
		}
	}

	// Probe Clang
	clangPath, err := exec.LookPath("clang++")
	if err == nil {
		out, err := exec.CommandContext(ctx, clangPath, "--version").Output()
		ver := "unknown"
		if err == nil && len(out) > 0 {
			ver = strings.Split(string(out), "\n")[0]
		}
		res["clang"] = CompilerInfo{
			Name:        "clang",
			Executable:  "clang++",
			IsAvailable: true,
			Version:     ver,
			Path:        clangPath,
		}
	} else {
		res["clang"] = CompilerInfo{
			Name:        "clang",
			Executable:  "clang++",
			IsAvailable: false,
		}
	}

	return res
}

// CompileArtifacts contains the build outcome and machine-code analysis.
type CompileArtifacts struct {
	BinaryPath       string
	BinarySizeBytes  int64
	AssemblyCode     string
	OptimizerRemarks string
	BuildDuration    time.Duration
	BuildCommand     string
	BuildStdout      string
	BuildStderr      string
	ExitCode         int
}

// BuildConfiguration defines a specific compiler configuration experiment.
type BuildConfig struct {
	CompilerName string // "gcc" or "clang"
	OptLevel     string // "-O0", "-O1", "-O2", "-O3", "-Ofast"
	ExtraFlags   []string
	TargetArch   string
}

// BuildLowLevel compiles the source file into a binary, captures assembly, and records vector remarks.
func BuildLowLevel(ctx context.Context, sourcePath string, outBinaryPath string, cfg BuildConfig) (*CompileArtifacts, error) {
	compilerBin := "g++"
	if cfg.CompilerName == "clang" {
		compilerBin = "clang++"
	}

	optLevel := cfg.OptLevel
	if optLevel == "" {
		optLevel = "-O3"
	}

	targetArch := cfg.TargetArch
	if targetArch == "" {
		targetArch = runtime.GOARCH
	}

	// Base compilation flags
	commonFlags := []string{
		optLevel,
		"-std=c++20",
		"-mavx2",
		"-mfma",
	}

	// Add OpenMP if gcc
	if cfg.CompilerName == "gcc" {
		commonFlags = append(commonFlags, "-fopenmp")
	}

	commonFlags = append(commonFlags, cfg.ExtraFlags...)

	// 1. Build the executable binary
	buildArgs := append(commonFlags, sourcePath, "-o", outBinaryPath)
	tStart := time.Now()
	cmd := exec.CommandContext(ctx, compilerBin, buildArgs...)
	outBytes, err := cmd.CombinedOutput()
	bDuration := time.Since(tStart)

	art := &CompileArtifacts{
		BinaryPath:    outBinaryPath,
		BuildDuration: bDuration,
		BuildCommand:  fmt.Sprintf("%s %s", compilerBin, strings.Join(buildArgs, " ")),
		BuildStdout:   string(outBytes),
		ExitCode:      0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			art.ExitCode = exitErr.ExitCode()
		} else {
			art.ExitCode = 1
		}
		art.BuildStderr = err.Error()
		return art, fmt.Errorf("build failed: %s", string(outBytes))
	}

	// Read binary size
	if fi, err := os.Stat(outBinaryPath); err == nil {
		art.BinarySizeBytes = fi.Size()
	}

	// 2. Extract Intel syntax assembly (-S -masm=intel)
	asmPath := outBinaryPath + ".s"
	var asmArgs []string
	if cfg.CompilerName == "clang" {
		asmArgs = append(commonFlags, "-S", "-mllvm", "--x86-asm-syntax=intel", sourcePath, "-o", asmPath)
	} else {
		asmArgs = append(commonFlags, "-S", "-masm=intel", "-fverbose-asm", sourcePath, "-o", asmPath)
	}

	_ = exec.CommandContext(ctx, compilerBin, asmArgs...).Run()
	if asmData, err := os.ReadFile(asmPath); err == nil {
		// Cap assembly size to first 3000 lines or ~150KB for responsive inspection
		lines := strings.Split(string(asmData), "\n")
		if len(lines) > 2000 {
			art.AssemblyCode = strings.Join(lines[:2000], "\n") + "\n\n... [truncated for display] ..."
		} else {
			art.AssemblyCode = string(asmData)
		}
		_ = os.Remove(asmPath)
	}

	// 3. Extract compiler optimization / vector remarks
	var remarkArgs []string
	if cfg.CompilerName == "clang" {
		remarkArgs = append(commonFlags, "-c", "-Rpass=loop-vectorize", sourcePath, "-o", filepath.Join(os.TempDir(), "remark.o"))
	} else {
		remarkArgs = append(commonFlags, "-c", "-fopt-info-vec-optimized", sourcePath, "-o", filepath.Join(os.TempDir(), "remark.o"))
	}
	remarkOutput, _ := exec.CommandContext(ctx, compilerBin, remarkArgs...).CombinedOutput()
	art.OptimizerRemarks = strings.TrimSpace(string(remarkOutput))
	_ = os.Remove(filepath.Join(os.TempDir(), "remark.o"))

	return art, nil
}
