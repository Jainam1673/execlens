package reports

import (
	"time"

	"github.com/execlens/execlens/internal/database"
	"github.com/execlens/execlens/internal/metrics"
)

// ReportConfig controls the report generation pipeline.
type ReportConfig struct {
	DBPath                   string   `json:"db_path"`
	OutputDir                string   `json:"output_dir"`
	RunIDs                   []string `json:"run_ids,omitempty"`
	WorkloadFilter           string   `json:"workload_filter,omitempty"`
	TrackFilter              string   `json:"track_filter,omitempty"`
	BaselineID               string   `json:"baseline_id,omitempty"` // default "cpp" or "scalar"
	UpdateReadme             bool     `json:"update_readme"`
	ReadmePath               string   `json:"readme_path"`
	PracticalThresholdPercent float64 `json:"practical_threshold_percent"` // default 5.0%
	ReportVersion            string   `json:"report_version"`
}

// DefaultReportConfig returns sensible defaults.
func DefaultReportConfig() ReportConfig {
	return ReportConfig{
		OutputDir:                "reports",
		BaselineID:               "cpp",
		UpdateReadme:             true,
		ReadmePath:               "README.md",
		PracticalThresholdPercent: 5.0,
		ReportVersion:            "1.0.0",
	}
}

// FindingCategory classifies the nature of an automated performance finding.
type FindingCategory string

const (
	FindingTopPerformer      FindingCategory = "TOP_PERFORMER"
	FindingSpeedup           FindingCategory = "SPEEDUP"
	FindingMemoryEfficiency  FindingCategory = "MEMORY_EFFICIENCY"
	FindingVariability       FindingCategory = "VARIABILITY"
	FindingCompilerOpt       FindingCategory = "COMPILER_OPTIMIZATION"
	FindingAccelerator       FindingCategory = "ACCELERATOR_TELEMETRY"
	FindingInconclusive      FindingCategory = "INCONCLUSIVE"
	FindingDiagnosticNote    FindingCategory = "DIAGNOSTIC_NOTE"
)

// ReportFinding represents a single deterministic written finding.
type ReportFinding struct {
	ID                 string                     `json:"id"`
	Title              string                     `json:"title"`
	Category           FindingCategory            `json:"category"`
	WorkloadID         string                     `json:"workload_id"`
	Track              string                     `json:"track"`
	Scope              string                     `json:"scope"`
	BaselineID         string                     `json:"baseline_id"`
	CandidateID        string                     `json:"candidate_id"`
	MetricName         string                     `json:"metric_name"`
	MetricUnit         string                     `json:"metric_unit"`
	BaselineValue      float64                    `json:"baseline_value"`
	CandidateValue     float64                    `json:"candidate_value"`
	EffectSize         string                     `json:"effect_size"`
	SampleCount        int                        `json:"sample_count"`
	Uncertainty        string                     `json:"uncertainty"`
	Status             metrics.SignificanceAssessment `json:"status"`
	Explanation        string                     `json:"explanation"`
	Limitations        string                     `json:"limitations"`
	ChartRelativePath  string                     `json:"chart_relative_path,omitempty"`
	RawDataRelativePath string                    `json:"raw_data_relative_path,omitempty"`
}

// ExecutionView encapsulates an execution with its calculated metrics for reporting.
type ExecutionView struct {
	database.LanguageExecution
	Speedup     float64 `json:"speedup"`
	MemoryRatio float64 `json:"memory_ratio"`
}

// WorkloadReportData groups all valid executions for a workload across runs.
type WorkloadReportData struct {
	WorkloadID       string                 `json:"workload_id"`
	WorkloadName     string                 `json:"workload_name"`
	Track            string                 `json:"track"`
	Description      string                 `json:"description"`
	Parameters       map[string]interface{} `json:"parameters"`
	ExecutionMode    string                 `json:"execution_mode"`
	BaselineID       string                 `json:"baseline_id"`
	Executions       []ExecutionView        `json:"executions"`
	PrimaryRunID     string                 `json:"primary_run_id"`
	CreatedAt        time.Time              `json:"created_at"`
	HasColdStartData bool                   `json:"has_cold_start_data"`
}

// ReportSummary contains high-level coverage and execution statistics.
type ReportSummary struct {
	ReportID             string                 `json:"report_id"`
	GeneratedAt          time.Time              `json:"generated_at"`
	GeneratorVersion     string                 `json:"generator_version"`
	Environment          database.SystemInfo    `json:"environment"`
	TotalConfigurations  int                    `json:"total_configurations"`
	SuccessfulExecutions int                    `json:"successful_executions"`
	FailedExecutions     int                    `json:"failed_executions"`
	SkippedExecutions    int                    `json:"skipped_executions"`
	ValidationFailures   int                    `json:"validation_failures"`
	InconclusiveCount    int                    `json:"inconclusive_count"`
	StatSignificantCount int                    `json:"statistically_significant_count"`
	WorkloadsCovered     []string               `json:"workloads_covered"`
	LanguagesTested      []string               `json:"languages_tested"`
	TracksCovered        []string               `json:"tracks_covered"`
	RunIDs               []string               `json:"run_ids"`
}

// FileChecksum records file verification information.
type FileChecksum struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

// ReportManifest captures the provenance and reproducibility metadata of a generated report.
type ReportManifest struct {
	ManifestVersion   string            `json:"manifest_version"`
	ReportID          string            `json:"report_id"`
	CreatedAt         time.Time         `json:"created_at"`
	GitRevision       string            `json:"git_revision"`
	HostFingerprint   database.SystemInfo `json:"host_fingerprint"`
	StatisticalConfig map[string]interface{} `json:"statistical_config"`
	WorkloadsCovered  []string          `json:"workloads_covered"`
	RunIDs            []string          `json:"run_ids"`
	Artifacts         []FileChecksum    `json:"artifacts"`
	OmittedMetrics    []string          `json:"omitted_metrics,omitempty"`
}

// GenerationResult provides status after a report has been produced.
type GenerationResult struct {
	ReportID        string            `json:"report_id"`
	OutputDir       string            `json:"output_dir"`
	LatestDir       string            `json:"latest_dir"`
	HistoryDir      string            `json:"history_dir"`
	Summary         ReportSummary     `json:"summary"`
	Manifest        ReportManifest    `json:"manifest"`
	Findings        []ReportFinding   `json:"findings"`
	GeneratedFiles  []string          `json:"generated_files"`
	ReadmeUpdated   bool              `json:"readme_updated"`
}
