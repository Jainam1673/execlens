package languages

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

type PythonAdapter struct{}

func NewPythonAdapter() *PythonAdapter {
	return &PythonAdapter{}
}

func (p *PythonAdapter) ID() string {
	return "python"
}

func (p *PythonAdapter) DisplayName() string {
	return "Python (CPython)"
}

func (p *PythonAdapter) RequiredTools() []string {
	return []string{"python3"}
}

func (p *PythonAdapter) DetectAvailability(ctx context.Context) (bool, string, string) {
	return checkToolVersion(ctx, "python3", "--version")
}

func (p *PythonAdapter) IsCompiled() bool {
	return false
}

func (p *PythonAdapter) SourceSubdir() string {
	return "python"
}

func (p *PythonAdapter) BuildCommand(workloadID, sourceDir, targetBinPath string) (string, []string, string) {
	// Interpreted language; no explicit build command required.
	return "", nil, ""
}

func (p *PythonAdapter) ExecutionCommand(workloadID, targetBinPath string, workloadArgs []string) (string, []string, string) {
	scriptPath := filepath.Join(targetBinPath, fmt.Sprintf("%s.py", workloadID))
	args := append([]string{"-u", scriptPath}, workloadArgs...)
	return "python3", args, targetBinPath
}

func (p *PythonAdapter) BuildTimeout() time.Duration {
	return 0
}

func (p *PythonAdapter) ExecutionTimeout() time.Duration {
	return 90 * time.Second
}

func (p *PythonAdapter) DefaultWarmupIterations() int {
	return 0
}

func (p *PythonAdapter) WarmupPolicyDescription() string {
	return "CPython bytecode interpreter does not have a dynamic JIT compiler; 0 warm-up iterations needed."
}

func (p *PythonAdapter) EnvironmentVariables() map[string]string {
	return map[string]string{
		"PYTHONDONTWRITEBYTECODE": "1",
		"PYTHONUNBUFFERED":        "1",
	}
}
