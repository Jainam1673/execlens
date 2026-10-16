export type RunStatus =
  | 'QUEUED'
  | 'PREPARING'
  | 'RUNNING'
  | 'VALIDATING'
  | 'COMPLETED'
  | 'FAILED'
  | 'CANCELLED';

export type LanguageExecutionStatus =
  | 'PENDING'
  | 'BUILDING'
  | 'WARMING_UP'
  | 'RUNNING'
  | 'VALIDATING'
  | 'COMPLETED'
  | 'FAILED'
  | 'SKIPPED';

export type SignificanceAssessment =
  | 'INCONCLUSIVE'
  | 'NEGLIGIBLE_DIFFERENCE'
  | 'PRACTICAL_DIFFERENCE'
  | 'STATISTICALLY_SIGNIFICANT';

export interface SystemInfo {
  os: string;
  arch: string;
  cpu_model: string;
  logical_cpus: number;
  total_memory_bytes: number;
  hostname?: string;
}

export interface MetricsSummary {
  count: number;
  min_ns: number;
  max_ns: number;
  mean_ns: number;
  median_ns: number;
  std_dev_ns: number;
  iqr_ns: number;
  cv_percent: number;
  min_ms: number;
  max_ms: number;
  mean_ms: number;
  median_ms: number;
  std_dev_ms: number;
  ci95_lower_ms: number;
  ci95_upper_ms: number;
  peak_memory_bytes: number;
  peak_memory_mb: number;
  throughput_ops_per_sec?: number;
  throughput_mb_per_sec?: number;
}

export interface RawSample {
  id: number;
  language_execution_id: string;
  iteration: number;
  is_warmup: boolean;
  duration_ns: number;
  duration_ms: number;
  memory_bytes: number;
  exit_code: number;
  created_at: string;
}

export interface LanguageExecution {
  id: string;
  run_id: string;
  language_id: string;
  status: LanguageExecutionStatus;
  toolchain_version: string;
  build_command?: string;
  build_duration_ns: number;
  build_duration_ms: number;
  build_stdout?: string;
  build_stderr?: string;
  build_exit_code: number;
  validation_passed?: boolean;
  validation_output?: string;
  validation_expected?: string;
  summary?: MetricsSummary;
  peak_memory_bytes: number;
  peak_memory_mb: number;
  error_message?: string;
  created_at: string;
  completed_at?: string;
  samples?: RawSample[];
  speedup?: number;
  memory_ratio?: number;
  significance?: SignificanceAssessment;
  significance_note?: string;
}

export interface Run {
  id: string;
  workload_id: string;
  workload_params: Record<string, any>;
  iterations: number;
  warmup_iterations: number;
  execution_mode: string;
  status: RunStatus;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  error_message?: string;
  system_info: SystemInfo;
}

export interface RunDetail extends Run {
  executions: LanguageExecution[];
}

export interface RuntimeInfo {
  id: string;
  display_name: string;
  is_available: boolean;
  version?: string;
  missing_reason?: string;
  required_tools: string[];
  is_compiled: boolean;
  default_warmup: number;
  warmup_policy_description: string;
}

export interface WorkloadParam {
  name: string;
  type: string;
  default: any;
  description: string;
}

export interface WorkloadContract {
  id: string;
  version: string;
  name: string;
  description: string;
  algorithm_purpose: string;
  parameters: WorkloadParam[];
  supported_modes: string[];
  fairness_caveats: string;
}

export interface DashboardStats {
  total_runs: number;
  completed_runs: number;
  failed_runs: number;
  last_run_at?: string;
  supported_languages: number;
  available_languages: number;
}

export interface ComparisonWarning {
  type: string;
  message: string;
}

export interface ComparisonReport {
  runs: RunDetail[];
  warnings: ComparisonWarning[];
}
