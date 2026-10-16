package languages

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

type CPPAdapter struct{}

func NewCPPAdapter() *CPPAdapter {
	return &CPPAdapter{}
}

func (c *CPPAdapter) ID() string {
	return "cpp"
}

func (c *CPPAdapter) DisplayName() string {
	return "C++ (GCC/G++)"
}

func (c *CPPAdapter) RequiredTools() []string {
	return []string{"g++"}
}

func (c *CPPAdapter) DetectAvailability(ctx context.Context) (bool, string, string) {
	return checkToolVersion(ctx, "g++", "--version")
}

func (c *CPPAdapter) IsCompiled() bool {
	return true
}

func (c *CPPAdapter) SourceSubdir() string {
	return "cpp"
}

func (c *CPPAdapter) BuildCommand(workloadID, sourceDir, targetBinPath string) (string, []string, string) {
	srcFile := filepath.Join(sourceDir, fmt.Sprintf("%s.cpp", workloadID))
	args := []string{
		"-O3",
		"-std=c++20",
		"-Wall",
		"-o", targetBinPath,
		srcFile,
	}
	return "g++", args, sourceDir
}

func (c *CPPAdapter) ExecutionCommand(workloadID, targetBinPath string, workloadArgs []string) (string, []string, string) {
	return targetBinPath, workloadArgs, filepath.Dir(targetBinPath)
}

func (c *CPPAdapter) BuildTimeout() time.Duration {
	return 30 * time.Second
}

func (c *CPPAdapter) ExecutionTimeout() time.Duration {
	return 60 * time.Second
}

func (c *CPPAdapter) DefaultWarmupIterations() int {
	return 0
}

func (c *CPPAdapter) WarmupPolicyDescription() string {
	return "Native machine code executed via GCC -O3 requires 0 warm-up iterations (no JIT warmup)."
}

func (c *CPPAdapter) EnvironmentVariables() map[string]string {
	return map[string]string{}
}
