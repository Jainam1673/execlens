package database

import (
	"encoding/json"
	"time"

	"github.com/execlens/execlens/internal/metrics"
)

// RunStatus defines the lifecycle states of a benchmark run.
type RunStatus string

const (
	StatusQueued     RunStatus = "QUEUED"
	StatusPreparing  RunStatus = "PREPARING"
	StatusRunning    RunStatus = "RUNNING"
	StatusValidating RunStatus = "VALIDATING"
	StatusCompleted  RunStatus = "COMPLETED"
	StatusFailed     RunStatus = "FAILED"
	StatusCancelled  RunStatus = "CANCELLED"
)

// LanguageExecutionStatus defines the lifecycle of a specific language within a run.
type LanguageExecutionStatus string

const (
	LangStatusPending    LanguageExecutionStatus = "PENDING"
	LangStatusBuilding   LanguageExecutionStatus = "BUILDING"
	LangStatusWarmingUp  LanguageExecutionStatus = "WARMING_UP"
	LangStatusRunning    LanguageExecutionStatus = "RUNNING"
	LangStatusValidating LanguageExecutionStatus = "VALIDATING"
	LangStatusCompleted  LanguageExecutionStatus = "COMPLETED"
	LangStatusFailed     LanguageExecutionStatus = "FAILED"
	LangStatusSkipped    LanguageExecutionStatus = "SKIPPED"
)

// SystemInfo stores captured hardware and operating environment details.
type SystemInfo struct {
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	CPUModel    string `json:"cpu_model"`
	LogicalCPUs int    `json:"logical_cpus"`
	TotalMemory uint64 `json:"total_memory_bytes"`
	Hostname    string `json:"hostname,omitempty"`
}

// Run represents a benchmark execution request and its state.
type Run struct {
	ID               string                 `json:"id"`
	WorkloadID       string                 `json:"workload_id"`
	WorkloadParams   map[string]interface{} `json:"workload_params"`
	Iterations       int                    `json:"iterations"`
	WarmupIterations int                    `json:"warmup_iterations"`
	ExecutionMode    string                 `json:"execution_mode"`
	Status           RunStatus              `json:"status"`
	CreatedAt        time.Time              `json:"created_at"`
	StartedAt        *time.Time             `json:"started_at,omitempty"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	SystemInfo       SystemInfo             `json:"system_info"`
}

// RawSample represents an individual timed execution sample.
type RawSample struct {
	ID                  int64     `json:"id"`
	LanguageExecutionID string    `json:"language_execution_id"`
	Iteration           int       `json:"iteration"`
	IsWarmup            bool      `json:"is_warmup"`
	DurationNs          int64     `json:"duration_ns"`
	DurationMs          float64   `json:"duration_ms"`
	MemoryBytes         int64     `json:"memory_bytes"`
	ExitCode            int       `json:"exit_code"`
	CreatedAt           time.Time `json:"created_at"`
}

// LanguageExecution represents the execution outcome for a specific language in a run.
type LanguageExecution struct {
	ID               string                  `json:"id"`
	RunID            string                  `json:"run_id"`
	LanguageID       string                  `json:"language_id"`
	Status           LanguageExecutionStatus `json:"status"`
	ToolchainVersion string                  `json:"toolchain_version"`
	BuildCommand     string                  `json:"build_command,omitempty"`
	BuildDurationNs  int64                   `json:"build_duration_ns"`
	BuildDurationMs  float64                 `json:"build_duration_ms"`
	BuildStdout      string                  `json:"build_stdout,omitempty"`
	BuildStderr      string                  `json:"build_stderr,omitempty"`
	BuildExitCode    int                     `json:"build_exit_code"`
	ValidationPassed *bool                   `json:"validation_passed,omitempty"`
	ValidationOutput string                  `json:"validation_output,omitempty"`
	ValidationExpect string                  `json:"validation_expected,omitempty"`
	Summary          *metrics.Summary        `json:"summary,omitempty"`
	PeakMemoryBytes  int64                   `json:"peak_memory_bytes"`
	PeakMemoryMB     float64                 `json:"peak_memory_mb"`
	ErrorMessage     string                  `json:"error_message,omitempty"`
	CreatedAt        time.Time               `json:"created_at"`
	CompletedAt      *time.Time              `json:"completed_at,omitempty"`
	Samples          []RawSample             `json:"samples,omitempty"`

	// Derived relative metrics (computed on demand relative to chosen baseline)
	Speedup         float64                    `json:"speedup,omitempty"`
	MemoryRatio     float64                    `json:"memory_ratio,omitempty"`
	Significance    metrics.SignificanceAssessment `json:"significance,omitempty"`
	SignificanceNote string                    `json:"significance_note,omitempty"`
}

// RunDetail contains a run and all its associated language executions.
type RunDetail struct {
	Run
	Executions []LanguageExecution `json:"executions"`
}

// Helper to marshal JSON string for database storage
func ToJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
