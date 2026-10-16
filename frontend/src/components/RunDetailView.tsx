import React, { useEffect, useState } from 'react';
import {
  Download,
  FileText,
  RotateCw,
  XCircle,
  CheckCircle,
  AlertTriangle,
  BarChart2,
  Terminal,
} from 'lucide-react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
  Cell,
} from 'recharts';
import type { RunDetail, LanguageExecution } from '../types';
import { fetchRun, cancelRun, fetchReport } from '../api';

interface RunDetailViewProps {
  runId: string;
  onBack: () => void;
}

const COLORS = ['#38bdf8', '#818cf8', '#34d399', '#fbbf24', '#f87171', '#c084fc'];

export const RunDetailView: React.FC<RunDetailViewProps> = ({ runId, onBack }) => {
  const [detail, setDetail] = useState<RunDetail | null>(null);
  const [baseline, setBaseline] = useState<string>('');
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [reportText, setReportText] = useState<string | null>(null);
  const [inspectExecution, setInspectExecution] = useState<LanguageExecution | null>(null);

  const loadData = async (selectedBaseline?: string) => {
    try {
      const data = await fetchRun(runId, selectedBaseline || baseline);
      setDetail(data);
      if (!baseline && data.executions.length > 0) {
        // Default baseline to first completed execution
        const completed = data.executions.find((e) => e.summary !== undefined);
        if (completed) {
          setBaseline(completed.language_id);
        }
      }
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Failed to fetch run details');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(() => {
      if (
        detail &&
        ['QUEUED', 'PREPARING', 'RUNNING', 'VALIDATING'].includes(detail.status)
      ) {
        loadData();
      }
    }, 2000);
    return () => clearInterval(interval);
  }, [runId, baseline, detail?.status]);

  const handleBaselineChange = (newBaseline: string) => {
    setBaseline(newBaseline);
    loadData(newBaseline);
  };

  const handleCancel = async () => {
    try {
      await cancelRun(runId);
      loadData();
    } catch (err: any) {
      alert(`Error cancelling run: ${err.message}`);
    }
  };

  const handleViewReport = async () => {
    try {
      const md = await fetchReport(runId);
      setReportText(md);
    } catch (err: any) {
      alert(`Error loading report: ${err.message}`);
    }
  };

  if (isLoading && !detail) {
    return (
      <div style={{ textAlign: 'center', padding: '4rem 1rem' }}>
        <RotateCw size={32} className="spin" style={{ margin: '0 auto 1rem' }} />
        <p style={{ color: 'var(--text-secondary)' }}>Loading benchmark execution measurements...</p>
      </div>
    );
  }

  if (error || !detail) {
    return (
      <div className="card" style={{ textAlign: 'center', padding: '3rem 1rem' }}>
        <AlertTriangle size={40} color="var(--error)" style={{ margin: '0 auto 1rem' }} />
        <h3 style={{ fontSize: '1.25rem', fontWeight: 600, marginBottom: '0.5rem' }}>Failed to load run</h3>
        <p style={{ color: 'var(--text-secondary)', marginBottom: '1.5rem' }}>{error}</p>
        <button className="btn btn-secondary" onClick={onBack}>
          Back to Dashboard
        </button>
      </div>
    );
  }

  const isTerminal = ['COMPLETED', 'FAILED', 'CANCELLED'].includes(detail.status);

  // Prepare chart datasets
  const timeData = detail.executions
    .filter((e) => e.summary !== undefined)
    .map((e) => ({
      name: e.language_id.toUpperCase(),
      median_ms: parseFloat(e.summary!.median_ms.toFixed(2)),
      mean_ms: parseFloat(e.summary!.mean_ms.toFixed(2)),
      std_dev_ms: parseFloat(e.summary!.std_dev_ms.toFixed(2)),
      speedup: parseFloat((e.speedup || 1.0).toFixed(2)),
      memory_mb: parseFloat(e.peak_memory_mb.toFixed(1)),
      throughput: e.summary!.throughput_ops_per_sec
        ? Math.round(e.summary!.throughput_ops_per_sec)
        : Math.round(e.summary!.throughput_mb_per_sec || 0),
    }));

  const buildData = detail.executions
    .filter((e) => e.build_duration_ms > 0)
    .map((e) => ({
      name: e.language_id.toUpperCase(),
      build_ms: parseFloat(e.build_duration_ms.toFixed(1)),
      execution_ms: e.summary ? parseFloat(e.summary.median_ms.toFixed(1)) : 0,
    }));

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '2rem' }}>
      {/* Top Header Card */}
      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '1rem' }}>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '0.5rem' }}>
              <button className="btn btn-secondary btn-sm" onClick={onBack}>
                &larr; Back
              </button>
              <h1 style={{ fontSize: '1.5rem', fontWeight: 700 }}>
                {detail.workload_id.toUpperCase()} Benchmark
              </h1>
              <span
                className={`badge ${
                  detail.status === 'COMPLETED'
                    ? 'badge-success'
                    : detail.status === 'FAILED'
                    ? 'badge-error'
                    : 'badge-warning'
                }`}
              >
                {detail.status}
              </span>
            </div>
            <p style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
              Run ID: <code>{detail.id}</code> &bull; Created: {new Date(detail.created_at).toUTCString()}
            </p>
          </div>

          {/* Action Toolbar */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
            {!isTerminal && (
              <button className="btn btn-danger btn-sm" onClick={handleCancel}>
                <XCircle size={14} /> Cancel Run
              </button>
            )}

            <button className="btn btn-secondary btn-sm" onClick={handleViewReport}>
              <FileText size={14} /> View Markdown Report
            </button>

            <a
              href={`/api/v1/runs/${detail.id}/export?format=json`}
              download={`execlens_${detail.id}.json`}
              className="btn btn-secondary btn-sm"
            >
              <Download size={14} /> JSON
            </a>

            <a
              href={`/api/v1/runs/${detail.id}/export?format=csv`}
              download={`execlens_${detail.id}.csv`}
              className="btn btn-secondary btn-sm"
            >
              <Download size={14} /> CSV
            </a>
          </div>
        </div>

        {/* Environmental Metadata Sub-bar */}
        <div
          style={{
            marginTop: '1.25rem',
            paddingTop: '1.25rem',
            borderTop: '1px solid var(--border-subtle)',
            display: 'flex',
            gap: '2rem',
            flexWrap: 'wrap',
            fontSize: '0.85rem',
            color: 'var(--text-secondary)',
          }}
        >
          <div>
            <strong style={{ color: 'var(--text-primary)' }}>System:</strong> {detail.system_info.os} ({detail.system_info.arch}), {detail.system_info.logical_cpus} Cores
          </div>
          <div>
            <strong style={{ color: 'var(--text-primary)' }}>CPU:</strong> {detail.system_info.cpu_model}
          </div>
          <div>
            <strong style={{ color: 'var(--text-primary)' }}>Config:</strong> {detail.iterations} Timed Iterations ({detail.warmup_iterations} Warmup)
          </div>
          <div>
            <strong style={{ color: 'var(--text-primary)' }}>Mode:</strong> {detail.execution_mode}
          </div>
        </div>
      </div>

      {/* Baseline Selection Control */}
      <div
        className="card"
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: '1rem 1.5rem',
          flexWrap: 'wrap',
          gap: '1rem',
          backgroundColor: 'var(--bg-muted)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
          <BarChart2 size={20} color="var(--accent-primary)" />
          <div>
            <span style={{ fontWeight: 600, fontSize: '0.9rem' }}>Comparative Speedup Baseline</span>
            <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
              Speedup = Baseline Median Time / Candidate Median Time. Above 1.0x indicates candidate was faster.
            </p>
          </div>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <label style={{ fontSize: '0.85rem', fontWeight: 600 }}>Baseline:</label>
          <select
            value={baseline}
            onChange={(e) => handleBaselineChange(e.target.value)}
            style={{
              padding: '0.4rem 0.8rem',
              backgroundColor: 'var(--bg-surface)',
              border: '1px solid var(--border-color)',
              borderRadius: '6px',
              color: 'var(--text-primary)',
              fontWeight: 600,
            }}
          >
            {detail.executions
              .filter((e) => e.summary !== undefined)
              .map((e) => (
                <option key={e.language_id} value={e.language_id}>
                  {e.language_id.toUpperCase()} ({e.toolchain_version})
                </option>
              ))}
          </select>
        </div>
      </div>

      {/* Interactive Charts Grid */}
      <div className="grid-cols-2">
        {/* Median Execution Duration Bar Chart */}
        <div className="card">
          <h3 className="card-title">Median Execution Time (Lower is Better)</h3>
          <p className="card-desc">Duration of steady-state workload repetitions in milliseconds.</p>
          <div style={{ width: '100%', height: 280 }}>
            <ResponsiveContainer>
              <BarChart data={timeData} margin={{ top: 10, right: 10, left: -10, bottom: 20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border-subtle)" />
                <XAxis dataKey="name" stroke="var(--text-secondary)" fontSize={12} />
                <YAxis stroke="var(--text-secondary)" fontSize={12} unit="ms" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'var(--bg-surface)',
                    borderColor: 'var(--border-color)',
                    color: 'var(--text-primary)',
                    borderRadius: '8px',
                  }}
                  formatter={(val: any) => [`${val} ms`, 'Median Duration']}
                />
                <Bar dataKey="median_ms" radius={[4, 4, 0, 0]}>
                  {timeData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.name.toLowerCase() === baseline.toLowerCase() ? '#6366f1' : COLORS[index % COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Speedup Multiplier Relative to Baseline */}
        <div className="card">
          <h3 className="card-title">Relative Speedup (Higher is Better)</h3>
          <p className="card-desc">Computed relative to {baseline.toUpperCase()} (1.0x baseline marked).</p>
          <div style={{ width: '100%', height: 280 }}>
            <ResponsiveContainer>
              <BarChart data={timeData} margin={{ top: 10, right: 10, left: -10, bottom: 20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border-subtle)" />
                <XAxis dataKey="name" stroke="var(--text-secondary)" fontSize={12} />
                <YAxis stroke="var(--text-secondary)" fontSize={12} unit="x" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'var(--bg-surface)',
                    borderColor: 'var(--border-color)',
                    color: 'var(--text-primary)',
                    borderRadius: '8px',
                  }}
                  formatter={(val: any) => [`${val}x`, 'Speedup']}
                />
                <Bar dataKey="speedup" fill="var(--accent-primary)" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Peak Resident Memory Chart */}
        <div className="card">
          <h3 className="card-title">Peak Resident Memory (MaxRSS)</h3>
          <p className="card-desc">Maximum physical RAM consumed (MB) measured by OS kernel rusage.</p>
          <div style={{ width: '100%', height: 280 }}>
            <ResponsiveContainer>
              <BarChart data={timeData} margin={{ top: 10, right: 10, left: -10, bottom: 20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border-subtle)" />
                <XAxis dataKey="name" stroke="var(--text-secondary)" fontSize={12} />
                <YAxis stroke="var(--text-secondary)" fontSize={12} unit="MB" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'var(--bg-surface)',
                    borderColor: 'var(--border-color)',
                    color: 'var(--text-primary)',
                    borderRadius: '8px',
                  }}
                  formatter={(val: any) => [`${val} MB`, 'Peak Memory']}
                />
                <Bar dataKey="memory_mb" fill="#10b981" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Throughput Chart */}
        <div className="card">
          <h3 className="card-title">Throughput (Operations / sec)</h3>
          <p className="card-desc">Completed logical operations divided by median execution time.</p>
          <div style={{ width: '100%', height: 280 }}>
            <ResponsiveContainer>
              <BarChart data={timeData} margin={{ top: 10, right: 10, left: 10, bottom: 20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border-subtle)" />
                <XAxis dataKey="name" stroke="var(--text-secondary)" fontSize={12} />
                <YAxis stroke="var(--text-secondary)" fontSize={12} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'var(--bg-surface)',
                    borderColor: 'var(--border-color)',
                    color: 'var(--text-primary)',
                    borderRadius: '8px',
                  }}
                  formatter={(val: any) => [`${val.toLocaleString()} ops/s`, 'Throughput']}
                />
                <Bar dataKey="throughput" fill="#f59e0b" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      {/* Build Duration vs Execution Duration Breakdown */}
      {buildData.length > 0 && (
        <div className="card">
          <h3 className="card-title">Compilation vs Steady-State Execution Duration</h3>
          <p className="card-desc">
            Independent tracking: compilation duration is separated from algorithmic execution duration.
          </p>
          <div style={{ width: '100%', height: 260 }}>
            <ResponsiveContainer>
              <BarChart data={buildData} margin={{ top: 10, right: 10, left: 0, bottom: 20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border-subtle)" />
                <XAxis dataKey="name" stroke="var(--text-secondary)" fontSize={12} />
                <YAxis stroke="var(--text-secondary)" fontSize={12} unit="ms" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'var(--bg-surface)',
                    borderColor: 'var(--border-color)',
                    color: 'var(--text-primary)',
                    borderRadius: '8px',
                  }}
                />
                <Legend />
                <Bar dataKey="build_ms" name="Build / Compile Duration (ms)" fill="#818cf8" radius={[4, 4, 0, 0]} />
                <Bar dataKey="execution_ms" name="Execution Median (ms)" fill="#38bdf8" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* Comprehensive Results Data Table */}
      <div className="card">
        <h3 className="card-title">Execution Measurements & Statistical Summary</h3>
        <p className="card-desc">
          Detailed metrics across all languages, including Student's t 95% Confidence Intervals and variability.
        </p>

        <div className="data-table-container">
          <table className="data-table">
            <thead>
              <tr>
                <th>Language</th>
                <th>Status</th>
                <th>Toolchain</th>
                <th>Median</th>
                <th>Mean</th>
                <th>95% CI</th>
                <th>StdDev (CV%)</th>
                <th>Peak RAM</th>
                <th>Throughput</th>
                <th>Speedup</th>
                <th>Validation</th>
                <th>Diagnostics</th>
              </tr>
            </thead>
            <tbody>
              {detail.executions.map((e) => (
                <tr key={e.id}>
                  <td>
                    <strong style={{ fontSize: '0.95rem' }}>{e.language_id.toUpperCase()}</strong>
                  </td>
                  <td>
                    <span
                      className={`badge ${
                        e.status === 'COMPLETED'
                          ? 'badge-success'
                          : e.status === 'FAILED'
                          ? 'badge-error'
                          : 'badge-neutral'
                      }`}
                    >
                      {e.status}
                    </span>
                  </td>
                  <td style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                    <code>{e.toolchain_version || 'N/A'}</code>
                  </td>
                  <td>
                    <strong>{e.summary ? `${e.summary.median_ms.toFixed(2)} ms` : 'N/A'}</strong>
                  </td>
                  <td>{e.summary ? `${e.summary.mean_ms.toFixed(2)} ms` : 'N/A'}</td>
                  <td style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                    {e.summary ? `[${e.summary.ci95_lower_ms.toFixed(2)}, ${e.summary.ci95_upper_ms.toFixed(2)}]` : 'N/A'}
                  </td>
                  <td style={{ fontSize: '0.8rem' }}>
                    {e.summary ? `±${e.summary.std_dev_ms.toFixed(2)} ms (${e.summary.cv_percent.toFixed(1)}%)` : 'N/A'}
                  </td>
                  <td>{e.peak_memory_mb ? `${e.peak_memory_mb.toFixed(1)} MB` : 'N/A'}</td>
                  <td>
                    {e.summary?.throughput_ops_per_sec
                      ? `${Math.round(e.summary.throughput_ops_per_sec).toLocaleString()} ops/s`
                      : 'N/A'}
                  </td>
                  <td>
                    {e.speedup !== undefined && e.speedup > 0 ? (
                      <span
                        style={{
                          fontWeight: 700,
                          color: e.speedup >= 1.0 ? 'var(--success)' : 'var(--warning)',
                        }}
                      >
                        {e.speedup.toFixed(2)}x
                      </span>
                    ) : (
                      'N/A'
                    )}
                  </td>
                  <td>
                    {e.validation_passed === true ? (
                      <span style={{ color: 'var(--success)', display: 'flex', alignItems: 'center', gap: '4px' }}>
                        <CheckCircle size={14} /> Passed
                      </span>
                    ) : e.validation_passed === false ? (
                      <span style={{ color: 'var(--error)', display: 'flex', alignItems: 'center', gap: '4px' }}>
                        <XCircle size={14} /> Failed
                      </span>
                    ) : (
                      <span style={{ color: 'var(--text-muted)' }}>Pending</span>
                    )}
                  </td>
                  <td>
                    <button
                      className="btn btn-secondary btn-sm"
                      onClick={() => setInspectExecution(e)}
                    >
                      <Terminal size={12} /> Inspect
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Statistical Assessment & Fairness Notes */}
      <div className="card">
        <h3 className="card-title">Statistical Significance & Fairness Notes</h3>
        <p className="card-desc">
          Rigorous analysis differentiating true performance differences from measurement variance and confidence interval overlap.
        </p>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
          {detail.executions.map((e) => (
            <div
              key={e.id}
              style={{
                padding: '0.85rem 1rem',
                backgroundColor: 'var(--bg-muted)',
                borderRadius: '8px',
                borderLeft: '4px solid',
                borderLeftColor:
                  e.significance === 'STATISTICALLY_SIGNIFICANT'
                    ? 'var(--success)'
                    : e.significance === 'NEGLIGIBLE_DIFFERENCE'
                    ? 'var(--text-muted)'
                    : 'var(--warning)',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.25rem' }}>
                <strong>{e.language_id.toUpperCase()}</strong>
                <span className="badge badge-neutral">{e.significance || 'EVALUATED'}</span>
              </div>
              <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                {e.significance_note || (e.error_message ? `Error: ${e.error_message}` : 'Execution completed successfully.')}
              </p>
            </div>
          ))}
        </div>
      </div>

      {/* Report Modal */}
      {reportText && (
        <div className="modal-overlay">
          <div className="modal-content" style={{ maxWidth: '800px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
              <h2 style={{ fontSize: '1.25rem', fontWeight: 700 }}>Markdown Benchmark Report</h2>
              <button className="btn btn-secondary btn-sm" onClick={() => setReportText(null)}>
                <XCircle size={16} />
              </button>
            </div>
            <pre
              style={{
                backgroundColor: 'var(--bg-muted)',
                padding: '1rem',
                borderRadius: '8px',
                fontSize: '0.85rem',
                overflowX: 'auto',
                whiteSpace: 'pre-wrap',
                maxHeight: '60vh',
              }}
            >
              {reportText}
            </pre>
            <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '1rem' }}>
              <button
                className="btn btn-primary"
                onClick={() => {
                  navigator.clipboard.writeText(reportText);
                  alert('Report copied to clipboard!');
                }}
              >
                Copy to Clipboard
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Execution Diagnostics Modal */}
      {inspectExecution && (
        <div className="modal-overlay">
          <div className="modal-content" style={{ maxWidth: '750px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
              <h2 style={{ fontSize: '1.25rem', fontWeight: 700 }}>
                Diagnostics: {inspectExecution.language_id.toUpperCase()}
              </h2>
              <button className="btn btn-secondary btn-sm" onClick={() => setInspectExecution(null)}>
                <XCircle size={16} />
              </button>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem', fontSize: '0.85rem' }}>
              <div>
                <strong>Build Command:</strong>
                <pre style={{ backgroundColor: 'var(--bg-muted)', padding: '0.5rem', borderRadius: '4px', marginTop: '4px' }}>
                  {inspectExecution.build_command || 'No explicit build step required (Interpreted / JIT)'}
                </pre>
              </div>

              {inspectExecution.build_stderr && (
                <div>
                  <strong style={{ color: 'var(--error)' }}>Build Stderr:</strong>
                  <pre style={{ backgroundColor: 'var(--bg-muted)', padding: '0.5rem', borderRadius: '4px', marginTop: '4px', color: 'var(--error)' }}>
                    {inspectExecution.build_stderr}
                  </pre>
                </div>
              )}

              <div>
                <strong>Validation Output:</strong>
                <pre style={{ backgroundColor: 'var(--bg-muted)', padding: '0.5rem', borderRadius: '4px', marginTop: '4px' }}>
                  {inspectExecution.validation_output || 'N/A'}
                </pre>
              </div>

              <div>
                <strong>Individual Repetition Samples ({inspectExecution.samples?.length || 0}):</strong>
                <div style={{ maxHeight: '200px', overflowY: 'auto', marginTop: '4px' }}>
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>#</th>
                        <th>Type</th>
                        <th>Duration (ms)</th>
                        <th>Memory (MB)</th>
                        <th>Exit Code</th>
                      </tr>
                    </thead>
                    <tbody>
                      {inspectExecution.samples?.map((s) => (
                        <tr key={s.id}>
                          <td>{s.iteration}</td>
                          <td>
                            <span className={`badge ${s.is_warmup ? 'badge-neutral' : 'badge-success'}`}>
                              {s.is_warmup ? 'Warmup' : 'Timed'}
                            </span>
                          </td>
                          <td>{s.duration_ms.toFixed(3)} ms</td>
                          <td>{(s.memory_bytes / (1024 * 1024)).toFixed(1)} MB</td>
                          <td>{s.exit_code}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
