-- Multitrack expansion: Low-Level Compiler Lab and GPU / Accelerator Lab

ALTER TABLE runs ADD COLUMN track TEXT DEFAULT 'languages';

CREATE INDEX IF NOT EXISTS idx_runs_track ON runs(track);

ALTER TABLE language_executions ADD COLUMN compiler_name TEXT;
ALTER TABLE language_executions ADD COLUMN compiler_flags TEXT;
ALTER TABLE language_executions ADD COLUMN opt_level TEXT;
ALTER TABLE language_executions ADD COLUMN target_arch TEXT;
ALTER TABLE language_executions ADD COLUMN executable_size_bytes INTEGER DEFAULT 0;
ALTER TABLE language_executions ADD COLUMN generated_assembly TEXT;
ALTER TABLE language_executions ADD COLUMN compiler_remarks TEXT;

ALTER TABLE language_executions ADD COLUMN kernel_duration_ns INTEGER DEFAULT 0;
ALTER TABLE language_executions ADD COLUMN transfer_h2d_ns INTEGER DEFAULT 0;
ALTER TABLE language_executions ADD COLUMN transfer_d2h_ns INTEGER DEFAULT 0;
ALTER TABLE language_executions ADD COLUMN alloc_duration_ns INTEGER DEFAULT 0;
ALTER TABLE language_executions ADD COLUMN effective_bandwidth_gbps REAL DEFAULT 0.0;
ALTER TABLE language_executions ADD COLUMN gflops REAL DEFAULT 0.0;
ALTER TABLE language_executions ADD COLUMN gpu_device_name TEXT;
ALTER TABLE language_executions ADD COLUMN gpu_compute_capability TEXT;
ALTER TABLE language_executions ADD COLUMN gpu_driver_version TEXT;
