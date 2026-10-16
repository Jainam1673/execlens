import React from 'react';
import {
  Activity,
  CheckCircle2,
  XCircle,
  Cpu,
  Layers,
  Play,
  ArrowRight,
  ExternalLink,
} from 'lucide-react';
import type { DashboardStats, RuntimeInfo, Run } from '../types';

interface DashboardViewProps {
  stats: DashboardStats | null;
  languages: RuntimeInfo[];
  recentRuns: Run[];
  onOpenLaunch: () => void;
  onSelectRun: (runId: string) => void;
  onNavigateRuns: () => void;
}

export const DashboardView: React.FC<DashboardViewProps> = ({
  stats,
  languages,
  recentRuns,
  onOpenLaunch,
  onSelectRun,
  onNavigateRuns,
}) => {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '2rem' }}>
      {/* Top Banner / Welcome */}
      <div
        className="card"
        style={{
          background: 'linear-gradient(135deg, rgba(30, 41, 59, 0.7), rgba(15, 23, 42, 0.9))',
          borderColor: 'var(--border-color)',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: '1.5rem',
        }}
      >
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '0.5rem' }}>
            <h1 style={{ fontSize: '1.75rem', fontWeight: 800, letterSpacing: '-0.03em' }}>
              Execution Performance Lens
            </h1>
            <span className="badge badge-success">Production Engine</span>
          </div>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.95rem', maxWidth: '650px' }}>
            Empirical cross-language benchmark orchestration. Compare deterministic CPU, memory, matrix, hash table, JSON, and streaming I/O workloads with high-precision statistical verification.
          </p>
        </div>
        <button className="btn btn-primary" onClick={onOpenLaunch} style={{ padding: '0.75rem 1.5rem', fontSize: '1rem' }}>
          <Play size={18} /> Launch Benchmark
        </button>
      </div>

      {/* Metric Summary Cards */}
      <div className="grid-cols-4">
        <div className="card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem' }}>
            <span style={{ color: 'var(--text-muted)', fontSize: '0.8rem', fontWeight: 600 }}>TOTAL BENCHMARKS</span>
            <Activity size={18} color="var(--accent-primary)" />
          </div>
          <div style={{ fontSize: '2rem', fontWeight: 700 }}>{stats?.total_runs ?? 0}</div>
          <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginTop: '0.25rem' }}>
            Recorded benchmark runs
          </div>
        </div>

        <div className="card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem' }}>
            <span style={{ color: 'var(--text-muted)', fontSize: '0.8rem', fontWeight: 600 }}>COMPLETED RUNS</span>
            <CheckCircle2 size={18} color="var(--success)" />
          </div>
          <div style={{ fontSize: '2rem', fontWeight: 700, color: 'var(--success)' }}>
            {stats?.completed_runs ?? 0}
          </div>
          <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginTop: '0.25rem' }}>
            Validated & statistically measured
          </div>
        </div>

        <div className="card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem' }}>
            <span style={{ color: 'var(--text-muted)', fontSize: '0.8rem', fontWeight: 600 }}>FAILED / CANCELLED</span>
            <XCircle size={18} color={stats && stats.failed_runs > 0 ? "var(--error)" : "var(--text-muted)"} />
          </div>
          <div style={{ fontSize: '2rem', fontWeight: 700, color: stats && stats.failed_runs > 0 ? "var(--error)" : "var(--text-primary)" }}>
            {stats?.failed_runs ?? 0}
          </div>
          <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginTop: '0.25rem' }}>
            Diagnosed failure states
          </div>
        </div>

        <div className="card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem' }}>
            <span style={{ color: 'var(--text-muted)', fontSize: '0.8rem', fontWeight: 600 }}>ACTIVE RUNTIMES</span>
            <Cpu size={18} color="var(--accent-primary)" />
          </div>
          <div style={{ fontSize: '2rem', fontWeight: 700 }}>
            {stats?.available_languages ?? 0} / {stats?.supported_languages ?? 6}
          </div>
          <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginTop: '0.25rem' }}>
            Detected system compilers
          </div>
        </div>
      </div>

      {/* Language Toolchain Probing Grid */}
      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
          <div>
            <h2 className="card-title">Supported Programming Language Runtimes</h2>
            <p className="card-desc">Automatic probing of system toolchains, compiler versions, and warm-up policies.</p>
          </div>
          <span className="badge badge-neutral">Standard Suite (6)</span>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '1rem' }}>
          {languages.map((lang) => (
            <div
              key={lang.id}
              style={{
                backgroundColor: 'var(--bg-muted)',
                border: '1px solid var(--border-color)',
                borderRadius: '8px',
                padding: '1rem',
                display: 'flex',
                flexDirection: 'column',
                gap: '0.5rem',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={{ fontWeight: 600, fontSize: '0.95rem' }}>{lang.display_name}</span>
                {lang.is_available ? (
                  <span className="badge badge-success">Available</span>
                ) : (
                  <span className="badge badge-error">Missing</span>
                )}
              </div>
              <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                {lang.is_available ? (
                  <code>{lang.version}</code>
                ) : (
                  <span style={{ color: 'var(--error)' }}>{lang.missing_reason}</span>
                )}
              </div>
              <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '0.25rem' }}>
                <strong>Warm-up policy:</strong> {lang.default_warmup} iterations ({lang.is_compiled ? 'Native Compiled' : 'Bytecode / JIT'})
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Recent Runs Table or Empty Onboarding */}
      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
          <div>
            <h2 className="card-title">Recent Benchmark Executions</h2>
            <p className="card-desc">Persistent historical benchmark executions and sample observations.</p>
          </div>
          {recentRuns.length > 0 && (
            <button className="btn btn-secondary btn-sm" onClick={onNavigateRuns}>
              View All History <ArrowRight size={14} />
            </button>
          )}
        </div>

        {recentRuns.length === 0 ? (
          <div
            style={{
              padding: '3rem 1rem',
              textAlign: 'center',
              backgroundColor: 'var(--bg-muted)',
              borderRadius: '8px',
            }}
          >
            <Layers size={48} color="var(--text-muted)" style={{ margin: '0 auto 1rem' }} />
            <h3 style={{ fontSize: '1.1rem', fontWeight: 600, marginBottom: '0.5rem' }}>No benchmarks executed yet</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem', maxWidth: '450px', margin: '0 auto 1.5rem' }}>
              Get started by launching your first cross-language benchmark run. ExecLens will execute equivalent workloads under controlled conditions.
            </p>
            <button className="btn btn-primary" onClick={onOpenLaunch}>
              <Play size={16} /> Launch Sieve of Eratosthenes
            </button>
          </div>
        ) : (
          <div className="data-table-container">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Run ID</th>
                  <th>Workload</th>
                  <th>Status</th>
                  <th>Iterations</th>
                  <th>Execution Mode</th>
                  <th>Created (UTC)</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {recentRuns.map((r) => (
                  <tr key={r.id}>
                    <td>
                      <code style={{ fontWeight: 600 }}>{r.id}</code>
                    </td>
                    <td>
                      <strong>{r.workload_id}</strong>
                    </td>
                    <td>
                      <span
                        className={`badge ${
                          r.status === 'COMPLETED'
                            ? 'badge-success'
                            : r.status === 'FAILED'
                            ? 'badge-error'
                            : 'badge-warning'
                        }`}
                      >
                        {r.status}
                      </span>
                    </td>
                    <td>
                      {r.iterations} timed ({r.warmup_iterations} warmup)
                    </td>
                    <td>
                      <span className="badge badge-neutral">{r.execution_mode}</span>
                    </td>
                    <td style={{ color: 'var(--text-secondary)', fontSize: '0.8rem' }}>
                      {new Date(r.created_at).toLocaleString()}
                    </td>
                    <td>
                      <button
                        className="btn btn-secondary btn-sm"
                        onClick={() => onSelectRun(r.id)}
                      >
                        Explore Results <ExternalLink size={12} />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
};
