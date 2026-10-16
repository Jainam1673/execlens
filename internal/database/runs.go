package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/execlens/execlens/internal/metrics"
)

// CreateRun inserts a new run and creates initial pending language executions within a transaction.
func (d *DB) CreateRun(ctx context.Context, run *Run, languageIDs []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	paramsJSON := ToJSON(run.WorkloadParams)
	systemJSON := ToJSON(run.SystemInfo)
	nowStr := run.CreatedAt.UTC().Format(time.RFC3339)

	if run.Track == "" {
		run.Track = "languages"
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO runs (id, track, workload_id, workload_params, iterations, warmup_iterations, execution_mode, status, created_at, system_info)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, run.ID, run.Track, run.WorkloadID, paramsJSON, run.Iterations, run.WarmupIterations, run.ExecutionMode, run.Status, nowStr, systemJSON)
	if err != nil {
		return fmt.Errorf("failed to insert run: %w", err)
	}

	for _, langID := range languageIDs {
		execID := fmt.Sprintf("%s_%s", run.ID, langID)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO language_executions (id, run_id, language_id, status, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, execID, run.ID, langID, LangStatusPending, nowStr)
		if err != nil {
			return fmt.Errorf("failed to insert language execution: %w", err)
		}
	}

	return tx.Commit()
}

// UpdateRunStatus updates the overall status of a run.
func (d *DB) UpdateRunStatus(ctx context.Context, runID string, status RunStatus, errorMessage string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	nowStr := time.Now().UTC().Format(time.RFC3339)

	var query string
	var args []interface{}

	switch status {
	case StatusRunning:
		query = "UPDATE runs SET status = ?, started_at = COALESCE(started_at, ?) WHERE id = ?"
		args = []interface{}{status, nowStr, runID}
	case StatusCompleted, StatusFailed, StatusCancelled:
		query = "UPDATE runs SET status = ?, completed_at = ?, error_message = ? WHERE id = ?"
		args = []interface{}{status, nowStr, errorMessage, runID}
	default:
		query = "UPDATE runs SET status = ?, error_message = ? WHERE id = ?"
		args = []interface{}{status, errorMessage, runID}
	}

	_, err := d.conn.ExecContext(ctx, query, args...)
	return err
}

// UpdateLanguageExecution updates build, validation, summary, and status of a language execution.
func (d *DB) UpdateLanguageExecution(ctx context.Context, exec *LanguageExecution) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var summaryJSON string
	if exec.Summary != nil {
		summaryJSON = ToJSON(exec.Summary)
	}

	var completedStr *string
	if exec.CompletedAt != nil {
		s := exec.CompletedAt.UTC().Format(time.RFC3339)
		completedStr = &s
	}

	var valPassed *int
	if exec.ValidationPassed != nil {
		val := 0
		if *exec.ValidationPassed {
			val = 1
		}
		valPassed = &val
	}

	query := `
		UPDATE language_executions
		SET status = ?,
		    toolchain_version = ?,
		    build_command = ?,
		    build_duration_ns = ?,
		    build_stdout = ?,
		    build_stderr = ?,
		    build_exit_code = ?,
		    validation_passed = ?,
		    validation_output = ?,
		    validation_expected = ?,
		    summary_json = ?,
		    peak_memory_bytes = ?,
		    error_message = ?,
		    completed_at = ?,
		    compiler_name = ?,
		    compiler_flags = ?,
		    opt_level = ?,
		    target_arch = ?,
		    executable_size_bytes = ?,
		    generated_assembly = ?,
		    compiler_remarks = ?,
		    kernel_duration_ns = ?,
		    transfer_h2d_ns = ?,
		    transfer_d2h_ns = ?,
		    alloc_duration_ns = ?,
		    effective_bandwidth_gbps = ?,
		    gflops = ?,
		    gpu_device_name = ?,
		    gpu_compute_capability = ?,
		    gpu_driver_version = ?
		WHERE id = ?
	`

	_, err := d.conn.ExecContext(ctx, query,
		exec.Status,
		exec.ToolchainVersion,
		exec.BuildCommand,
		exec.BuildDurationNs,
		exec.BuildStdout,
		exec.BuildStderr,
		exec.BuildExitCode,
		valPassed,
		exec.ValidationOutput,
		exec.ValidationExpect,
		summaryJSON,
		exec.PeakMemoryBytes,
		exec.ErrorMessage,
		completedStr,
		exec.CompilerName,
		exec.CompilerFlags,
		exec.OptLevel,
		exec.TargetArch,
		exec.ExecutableSizeBytes,
		exec.GeneratedAssembly,
		exec.CompilerRemarks,
		exec.KernelDurationNs,
		exec.TransferH2DNs,
		exec.TransferD2HNs,
		exec.AllocDurationNs,
		exec.EffectiveBandwidthGBps,
		exec.GFLOPS,
		exec.GPUDeviceName,
		exec.GPUComputeCapability,
		exec.GPUDriverVersion,
		exec.ID,
	)

	return err
}

// AddRawSample records an individual execution observation.
func (d *DB) AddRawSample(ctx context.Context, sample *RawSample) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	nowStr := sample.CreatedAt.UTC().Format(time.RFC3339)
	isWarmupInt := 0
	if sample.IsWarmup {
		isWarmupInt = 1
	}

	res, err := d.conn.ExecContext(ctx, `
		INSERT INTO raw_samples (language_execution_id, iteration, is_warmup, duration_ns, memory_bytes, exit_code, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, sample.LanguageExecutionID, sample.Iteration, isWarmupInt, sample.DurationNs, sample.MemoryBytes, sample.ExitCode, nowStr)
	if err != nil {
		return err
	}

	sample.ID, _ = res.LastInsertId()
	return nil
}

// GetRun retrieves a single run by ID.
func (d *DB) GetRun(ctx context.Context, runID string) (*Run, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	row := d.conn.QueryRowContext(ctx, `
		SELECT id, track, workload_id, workload_params, iterations, warmup_iterations, execution_mode,
		       status, created_at, started_at, completed_at, error_message, system_info
		FROM runs WHERE id = ?
	`, runID)

	var r Run
	var paramsJSON, systemJSON, createdStr string
	var trackStr, startedStr, completedStr, errMsg sql.NullString

	err := row.Scan(&r.ID, &trackStr, &r.WorkloadID, &paramsJSON, &r.Iterations, &r.WarmupIterations, &r.ExecutionMode,
		&r.Status, &createdStr, &startedStr, &completedStr, &errMsg, &systemJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	if trackStr.Valid && trackStr.String != "" {
		r.Track = trackStr.String
	} else {
		r.Track = "languages"
	}

	r.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	if startedStr.Valid {
		t, _ := time.Parse(time.RFC3339, startedStr.String)
		r.StartedAt = &t
	}
	if completedStr.Valid {
		t, _ := time.Parse(time.RFC3339, completedStr.String)
		r.CompletedAt = &t
	}
	if errMsg.Valid {
		r.ErrorMessage = errMsg.String
	}

	_ = json.Unmarshal([]byte(paramsJSON), &r.WorkloadParams)
	_ = json.Unmarshal([]byte(systemJSON), &r.SystemInfo)

	return &r, nil
}

// GetRunDetail retrieves a full run with executions and samples, recalculating relative metrics against baseline.
func (d *DB) GetRunDetail(ctx context.Context, runID string, baselineLangID string) (*RunDetail, error) {
	run, err := d.GetRun(ctx, runID)
	if err != nil || run == nil {
		return nil, err
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.conn.QueryContext(ctx, `
		SELECT id, run_id, language_id, status, toolchain_version, build_command,
		       build_duration_ns, build_stdout, build_stderr, build_exit_code,
		       validation_passed, validation_output, validation_expected,
		       summary_json, peak_memory_bytes, error_message, created_at, completed_at,
		       compiler_name, compiler_flags, opt_level, target_arch,
		       executable_size_bytes, generated_assembly, compiler_remarks,
		       kernel_duration_ns, transfer_h2d_ns, transfer_d2h_ns, alloc_duration_ns,
		       effective_bandwidth_gbps, gflops, gpu_device_name, gpu_compute_capability, gpu_driver_version
		FROM language_executions
		WHERE run_id = ?
		ORDER BY language_id ASC
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to query language executions: %w", err)
	}
	defer rows.Close()

	var executions []LanguageExecution
	for rows.Next() {
		var e LanguageExecution
		var createdStr string
		var toolchain, buildCmd, bStdout, bStderr, valOut, valExp, summaryJSON, errMsg, compStr sql.NullString
		var valPassed sql.NullInt64
		var compName, compFlags, optLvl, tgtArch, genAsm, compRemarks sql.NullString
		var gpuDev, gpuCC, gpuDrv sql.NullString

		err := rows.Scan(
			&e.ID, &e.RunID, &e.LanguageID, &e.Status, &toolchain, &buildCmd,
			&e.BuildDurationNs, &bStdout, &bStderr, &e.BuildExitCode,
			&valPassed, &valOut, &valExp,
			&summaryJSON, &e.PeakMemoryBytes, &errMsg, &createdStr, &compStr,
			&compName, &compFlags, &optLvl, &tgtArch,
			&e.ExecutableSizeBytes, &genAsm, &compRemarks,
			&e.KernelDurationNs, &e.TransferH2DNs, &e.TransferD2HNs, &e.AllocDurationNs,
			&e.EffectiveBandwidthGBps, &e.GFLOPS, &gpuDev, &gpuCC, &gpuDrv,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		e.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		if compStr.Valid {
			t, _ := time.Parse(time.RFC3339, compStr.String)
			e.CompletedAt = &t
		}
		if toolchain.Valid {
			e.ToolchainVersion = toolchain.String
		}
		if buildCmd.Valid {
			e.BuildCommand = buildCmd.String
		}
		if bStdout.Valid {
			e.BuildStdout = bStdout.String
		}
		if bStderr.Valid {
			e.BuildStderr = bStderr.String
		}
		if errMsg.Valid {
			e.ErrorMessage = errMsg.String
		}
		if valOut.Valid {
			e.ValidationOutput = valOut.String
		}
		if valExp.Valid {
			e.ValidationExpect = valExp.String
		}
		if valPassed.Valid {
			p := valPassed.Int64 == 1
			e.ValidationPassed = &p
		}

		if compName.Valid {
			e.CompilerName = compName.String
		}
		if compFlags.Valid {
			e.CompilerFlags = compFlags.String
		}
		if optLvl.Valid {
			e.OptLevel = optLvl.String
		}
		if tgtArch.Valid {
			e.TargetArch = tgtArch.String
		}
		if genAsm.Valid {
			e.GeneratedAssembly = genAsm.String
		}
		if compRemarks.Valid {
			e.CompilerRemarks = compRemarks.String
		}

		if gpuDev.Valid {
			e.GPUDeviceName = gpuDev.String
		}
		if gpuCC.Valid {
			e.GPUComputeCapability = gpuCC.String
		}
		if gpuDrv.Valid {
			e.GPUDriverVersion = gpuDrv.String
		}

		e.BuildDurationMs = float64(e.BuildDurationNs) / 1e6
		e.PeakMemoryMB = float64(e.PeakMemoryBytes) / (1024 * 1024)
		e.KernelDurationMs = float64(e.KernelDurationNs) / 1e6
		e.TransferH2DMs = float64(e.TransferH2DNs) / 1e6
		e.TransferD2HMs = float64(e.TransferD2HNs) / 1e6
		e.AllocDurationMs = float64(e.AllocDurationNs) / 1e6

		if summaryJSON.Valid && summaryJSON.String != "" {
			var s metrics.Summary
			if err := json.Unmarshal([]byte(summaryJSON.String), &s); err == nil {
				e.Summary = &s
			}
		}

		executions = append(executions, e)
	}

	// Fetch raw samples for each execution
	for i := range executions {
		sRows, err := d.conn.QueryContext(ctx, `
			SELECT id, language_execution_id, iteration, is_warmup, duration_ns, memory_bytes, exit_code, created_at
			FROM raw_samples
			WHERE language_execution_id = ?
			ORDER BY iteration ASC, id ASC
		`, executions[i].ID)
		if err == nil {
			for sRows.Next() {
				var s RawSample
				var isWarmupInt int
				var createdStr string
				if err := sRows.Scan(&s.ID, &s.LanguageExecutionID, &s.Iteration, &isWarmupInt, &s.DurationNs, &s.MemoryBytes, &s.ExitCode, &createdStr); err == nil {
					s.IsWarmup = isWarmupInt == 1
					s.DurationMs = float64(s.DurationNs) / 1e6
					s.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
					executions[i].Samples = append(executions[i].Samples, s)
				}
			}
			sRows.Close()
		}
	}

	// Compute relative metrics if a baseline is specified or pick the first completed execution
	var baselineExec *LanguageExecution
	if baselineLangID != "" {
		for i := range executions {
			if executions[i].LanguageID == baselineLangID && executions[i].Summary != nil {
				baselineExec = &executions[i]
				break
			}
		}
	}
	// Fallback to first completed with valid summary if specified baseline is not available
	if baselineExec == nil {
		for i := range executions {
			if executions[i].Summary != nil {
				baselineExec = &executions[i]
				break
			}
		}
	}

	if baselineExec != nil && baselineExec.Summary != nil {
		for i := range executions {
			if executions[i].Summary != nil {
				baseNs := baselineExec.Summary.MedianNs
				if baseNs <= 0 {
					baseNs = baselineExec.Summary.MedianMs * 1e6
				}
				candNs := executions[i].Summary.MedianNs
				if candNs <= 0 {
					candNs = executions[i].Summary.MedianMs * 1e6
				}
				executions[i].Speedup = metrics.CalculateSpeedup(baseNs, candNs)
				executions[i].MemoryRatio = metrics.CalculateMemoryRatio(baselineExec.PeakMemoryBytes, executions[i].PeakMemoryBytes)
				assessment, note := metrics.AssessDifference(*baselineExec.Summary, *executions[i].Summary, 5.0)
				executions[i].Significance = assessment
				executions[i].SignificanceNote = note
			}
		}
	}

	return &RunDetail{
		Run:        *run,
		Executions: executions,
	}, nil
}

// ListRuns returns paginated benchmark history with optional filters.
func (d *DB) ListRuns(ctx context.Context, limit, offset int, track string, workloadID string, status string) ([]Run, int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var whereClauses []string
	var args []interface{}

	if track != "" {
		whereClauses = append(whereClauses, "track = ?")
		args = append(args, track)
	}
	if workloadID != "" {
		whereClauses = append(whereClauses, "workload_id = ?")
		args = append(args, workloadID)
	}
	if status != "" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, status)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + whereClauses[0]
		for i := 1; i < len(whereClauses); i++ {
			whereSQL += " AND " + whereClauses[i]
		}
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(1) FROM runs %s", whereSQL)
	err := d.conn.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count runs: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, track, workload_id, workload_params, iterations, warmup_iterations, execution_mode,
		       status, created_at, started_at, completed_at, error_message, system_info
		FROM runs
		%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereSQL)

	queryArgs := append(args, limit, offset)
	rows, err := d.conn.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query runs: %w", err)
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		var r Run
		var paramsJSON, systemJSON, createdStr string
		var trackStr, startedStr, completedStr, errMsg sql.NullString

		if err := rows.Scan(&r.ID, &trackStr, &r.WorkloadID, &paramsJSON, &r.Iterations, &r.WarmupIterations, &r.ExecutionMode,
			&r.Status, &createdStr, &startedStr, &completedStr, &errMsg, &systemJSON); err != nil {
			return nil, 0, fmt.Errorf("failed to scan run: %w", err)
		}

		if trackStr.Valid && trackStr.String != "" {
			r.Track = trackStr.String
		} else {
			r.Track = "languages"
		}

		r.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		if startedStr.Valid {
			t, _ := time.Parse(time.RFC3339, startedStr.String)
			r.StartedAt = &t
		}
		if completedStr.Valid {
			t, _ := time.Parse(time.RFC3339, completedStr.String)
			r.CompletedAt = &t
		}
		if errMsg.Valid {
			r.ErrorMessage = errMsg.String
		}

		_ = json.Unmarshal([]byte(paramsJSON), &r.WorkloadParams)
		_ = json.Unmarshal([]byte(systemJSON), &r.SystemInfo)

		runs = append(runs, r)
	}

	return runs, total, nil
}

// SummaryStats returns counts for total, completed, and failed runs.
type DashboardStats struct {
	TotalRuns     int        `json:"total_runs"`
	CompletedRuns int        `json:"completed_runs"`
	FailedRuns    int        `json:"failed_runs"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
}

// GetDashboardStats computes real summary counters.
func (d *DB) GetDashboardStats(ctx context.Context) (DashboardStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var s DashboardStats
	_ = d.conn.QueryRowContext(ctx, "SELECT COUNT(1) FROM runs").Scan(&s.TotalRuns)
	_ = d.conn.QueryRowContext(ctx, "SELECT COUNT(1) FROM runs WHERE status = 'COMPLETED'").Scan(&s.CompletedRuns)
	_ = d.conn.QueryRowContext(ctx, "SELECT COUNT(1) FROM runs WHERE status = 'FAILED'").Scan(&s.FailedRuns)

	var lastCreated sql.NullString
	_ = d.conn.QueryRowContext(ctx, "SELECT created_at FROM runs ORDER BY created_at DESC LIMIT 1").Scan(&lastCreated)
	if lastCreated.Valid {
		t, err := time.Parse(time.RFC3339, lastCreated.String)
		if err == nil {
			s.LastRunAt = &t
		}
	}

	return s, nil
}

// ReconcileInterruptedRuns reconciles any non-terminal runs left over from process shutdown or crashes.
func (d *DB) ReconcileInterruptedRuns(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	nowStr := time.Now().UTC().Format(time.RFC3339)
	_, err := d.conn.ExecContext(ctx, `
		UPDATE runs
		SET status = 'FAILED',
		    completed_at = ?,
		    error_message = 'Application restarted while run was in progress'
		WHERE status IN ('QUEUED', 'PREPARING', 'RUNNING', 'VALIDATING')
	`, nowStr)
	return err
}
