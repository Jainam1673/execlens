-- Initial schema for ExecLens

CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    workload_id TEXT NOT NULL,
    workload_params TEXT NOT NULL, -- JSON
    iterations INTEGER NOT NULL,
    warmup_iterations INTEGER NOT NULL,
    execution_mode TEXT NOT NULL, -- "standard", "cold_start"
    status TEXT NOT NULL, -- QUEUED, PREPARING, RUNNING, VALIDATING, COMPLETED, FAILED, CANCELLED
    created_at TEXT NOT NULL,
    started_at TEXT,
    completed_at TEXT,
    error_message TEXT,
    system_info TEXT NOT NULL -- JSON
);

CREATE INDEX IF NOT EXISTS idx_runs_created_at ON runs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_workload_id ON runs(workload_id);

CREATE TABLE IF NOT EXISTS language_executions (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    language_id TEXT NOT NULL,
    status TEXT NOT NULL, -- PENDING, BUILDING, WARMING_UP, RUNNING, VALIDATING, COMPLETED, FAILED, SKIPPED
    toolchain_version TEXT,
    build_command TEXT,
    build_duration_ns INTEGER DEFAULT 0,
    build_stdout TEXT,
    build_stderr TEXT,
    build_exit_code INTEGER DEFAULT 0,
    validation_passed INTEGER,
    validation_output TEXT,
    validation_expected TEXT,
    summary_json TEXT, -- JSON Metrics Summary
    peak_memory_bytes INTEGER DEFAULT 0,
    error_message TEXT,
    created_at TEXT NOT NULL,
    completed_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_lang_exec_run_id ON language_executions(run_id);
CREATE INDEX IF NOT EXISTS idx_lang_exec_language_id ON language_executions(language_id);

CREATE TABLE IF NOT EXISTS raw_samples (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    language_execution_id TEXT NOT NULL REFERENCES language_executions(id) ON DELETE CASCADE,
    iteration INTEGER NOT NULL,
    is_warmup INTEGER NOT NULL DEFAULT 0,
    duration_ns INTEGER NOT NULL,
    memory_bytes INTEGER DEFAULT 0,
    exit_code INTEGER NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_raw_samples_exec_id ON raw_samples(language_execution_id);
