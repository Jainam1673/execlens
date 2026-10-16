package languages

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

type JSAdapter struct{}

func NewJSAdapter() *JSAdapter {
	return &JSAdapter{}
}

func (j *JSAdapter) ID() string {
	return "javascript"
}

func (j *JSAdapter) DisplayName() string {
	return "JavaScript (Node.js / V8)"
}

func (j *JSAdapter) RequiredTools() []string {
	return []string{"node"}
}

func (j *JSAdapter) DetectAvailability(ctx context.Context) (bool, string, string) {
	return checkToolVersion(ctx, "node", "--version")
}

func (j *JSAdapter) IsCompiled() bool {
	return false
}

func (j *JSAdapter) SourceSubdir() string {
	return "javascript"
}

func (j *JSAdapter) BuildCommand(workloadID, sourceDir, targetBinPath string) (string, []string, string) {
	// Interpreted / V8 JIT; no separate compilation step
	return "", nil, ""
}

func (j *JSAdapter) ExecutionCommand(workloadID, targetBinPath string, workloadArgs []string) (string, []string, string) {
	scriptPath := filepath.Join(targetBinPath, fmt.Sprintf("%s.js", workloadID))
	args := append([]string{scriptPath}, workloadArgs...)
	return "node", args, targetBinPath
}

func (j *JSAdapter) BuildTimeout() time.Duration {
	return 0
}

func (j *JSAdapter) ExecutionTimeout() time.Duration {
	return 60 * time.Second
}

func (j *JSAdapter) DefaultWarmupIterations() int {
	return 2
}

func (j *JSAdapter) WarmupPolicyDescription() string {
	return "V8 runtime uses Ignition interpreter and TurboFan optimizing JIT compiler; 2 warm-up iterations allow TurboFan compilation of hot functions."
}

func (j *JSAdapter) EnvironmentVariables() map[string]string {
	return map[string]string{
		"NODE_ENV": "production",
	}
}
