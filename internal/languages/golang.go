package languages

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

type GoAdapter struct{}

func NewGoAdapter() *GoAdapter {
	return &GoAdapter{}
}

func (g *GoAdapter) ID() string {
	return "go"
}

func (g *GoAdapter) DisplayName() string {
	return "Go"
}

func (g *GoAdapter) RequiredTools() []string {
	return []string{"go"}
}

func (g *GoAdapter) DetectAvailability(ctx context.Context) (bool, string, string) {
	return checkToolVersion(ctx, "go", "version")
}

func (g *GoAdapter) IsCompiled() bool {
	return true
}

func (g *GoAdapter) SourceSubdir() string {
	return "go"
}

func (g *GoAdapter) BuildCommand(workloadID, sourceDir, targetBinPath string) (string, []string, string) {
	srcFile := fmt.Sprintf("%s.go", workloadID)
	args := []string{
		"build",
		"-trimpath",
		"-o", targetBinPath,
		srcFile,
	}
	return "go", args, sourceDir
}

func (g *GoAdapter) ExecutionCommand(workloadID, targetBinPath string, workloadArgs []string) (string, []string, string) {
	return targetBinPath, workloadArgs, filepath.Dir(targetBinPath)
}

func (g *GoAdapter) BuildTimeout() time.Duration {
	return 30 * time.Second
}

func (g *GoAdapter) ExecutionTimeout() time.Duration {
	return 60 * time.Second
}

func (g *GoAdapter) DefaultWarmupIterations() int {
	return 0
}

func (g *GoAdapter) WarmupPolicyDescription() string {
	return "Statically compiled native binary requires 0 warm-up iterations (no JIT compilation)."
}

func (g *GoAdapter) EnvironmentVariables() map[string]string {
	return map[string]string{
		"GODEBUG": "madvdontneed=1",
	}
}
