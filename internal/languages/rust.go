package languages

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

type RustAdapter struct{}

func NewRustAdapter() *RustAdapter {
	return &RustAdapter{}
}

func (r *RustAdapter) ID() string {
	return "rust"
}

func (r *RustAdapter) DisplayName() string {
	return "Rust (rustc)"
}

func (r *RustAdapter) RequiredTools() []string {
	return []string{"rustc"}
}

func (r *RustAdapter) DetectAvailability(ctx context.Context) (bool, string, string) {
	return checkToolVersion(ctx, "rustc", "--version")
}

func (r *RustAdapter) IsCompiled() bool {
	return true
}

func (r *RustAdapter) SourceSubdir() string {
	return "rust"
}

func (r *RustAdapter) BuildCommand(workloadID, sourceDir, targetBinPath string) (string, []string, string) {
	srcFile := filepath.Join(sourceDir, fmt.Sprintf("%s.rs", workloadID))
	args := []string{
		"-C", "opt-level=3",
		"-C", "codegen-units=1",
		"-o", targetBinPath,
		srcFile,
	}
	return "rustc", args, sourceDir
}

func (r *RustAdapter) ExecutionCommand(workloadID, targetBinPath string, workloadArgs []string) (string, []string, string) {
	return targetBinPath, workloadArgs, filepath.Dir(targetBinPath)
}

func (r *RustAdapter) BuildTimeout() time.Duration {
	return 60 * time.Second
}

func (r *RustAdapter) ExecutionTimeout() time.Duration {
	return 60 * time.Second
}

func (r *RustAdapter) DefaultWarmupIterations() int {
	return 0
}

func (r *RustAdapter) WarmupPolicyDescription() string {
	return "Native machine code compiled with rustc opt-level=3 requires 0 warm-up iterations (no JIT)."
}

func (r *RustAdapter) EnvironmentVariables() map[string]string {
	return map[string]string{
		"RUST_BACKTRACE": "1",
	}
}
