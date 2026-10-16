package languages

import (
	"context"
	"time"
)

// LanguageAdapter defines the complete interface and contract for a programming language runtime.
type LanguageAdapter interface {
	// ID returns the stable identifier (e.g., "cpp", "rust", "go", "java", "python", "javascript").
	ID() string

	// DisplayName returns human-readable name (e.g., "C++ (GCC)", "Rust (rustc)").
	DisplayName() string

	// RequiredTools returns a list of binary names required by this adapter (e.g. ["g++"]).
	RequiredTools() []string

	// DetectAvailability checks if required tools exist on host and returns version or error.
	DetectAvailability(ctx context.Context) (isAvailable bool, version string, missingReason string)

	// IsCompiled indicates if this language requires an explicit compilation step before running.
	IsCompiled() bool

	// BuildCommand returns the binary and arguments to compile the workload.
	// sourceDir is the root directory containing the source code.
	// targetBinPath is the output binary path for compiled languages.
	BuildCommand(workloadID, sourceDir, targetBinPath string) (executable string, args []string, workingDir string)

	// ExecutionCommand returns the command to execute a single benchmark run.
	// targetBinPath is either the compiled binary or the entry script path.
	// workloadParams are passed as CLI arguments (e.g. --limit 1000000 --json).
	ExecutionCommand(workloadID, targetBinPath string, workloadArgs []string) (executable string, args []string, workingDir string)

	// BuildTimeout returns maximum duration allowed for building the benchmark.
	BuildTimeout() time.Duration

	// ExecutionTimeout returns maximum duration allowed for a single execution iteration.
	ExecutionTimeout() time.Duration

	// DefaultWarmupIterations returns runtime-appropriate warm-up count (e.g., 2 for JIT, 0 for native).
	DefaultWarmupIterations() int

	// WarmupPolicyDescription describes why this language has this warm-up requirement.
	WarmupPolicyDescription() string

	// EnvironmentVariables returns restricted environment variables passed to execution.
	EnvironmentVariables() map[string]string

	// SourceSubdir returns the directory under benchmarks/ where this language's code resides.
	SourceSubdir() string
}

// RuntimeInfo describes the detected status of a language on the current host.
type RuntimeInfo struct {
	ID                   string   `json:"id"`
	DisplayName          string   `json:"display_name"`
	IsAvailable          bool     `json:"is_available"`
	Version              string   `json:"version,omitempty"`
	MissingReason        string   `json:"missing_reason,omitempty"`
	RequiredTools        []string `json:"required_tools"`
	IsCompiled           bool     `json:"is_compiled"`
	DefaultWarmup        int      `json:"default_warmup"`
	WarmupPolicyDesc     string   `json:"warmup_policy_description"`
}
