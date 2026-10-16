import React, { useEffect, useState } from 'react';
import { ExternalLink, RotateCw } from 'lucide-react';
import type { Run, WorkloadContract } from '../types';
import { fetchRuns } from '../api';

interface HistoryViewProps {
  workloads: WorkloadContract[];
  onSelectRun: (runId: string) => void;
  onOpenLaunch: () => void;
}

export const HistoryView: React.FC<HistoryViewProps> = ({
  workloads,
  onSelectRun,
  onOpenLaunch,
}) => {
  const [runs, setRuns] = useState<Run[]>([]);
  const [total, setTotal] = useState<number>(0);
  const [trackFilter, setTrackFilter] = useState<string>('');
  const [workloadFilter, setWorkloadFilter] = useState<string>('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [page, setPage] = useState<number>(0);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const limit = 15;

  const loadData = async () => {
    setIsLoading(true);
    try {
      const res = await fetchRuns(limit, page * limit, trackFilter || undefined, workloadFilter || undefined, statusFilter || undefined);
      setRuns(res.runs);
      setTotal(res.total);
    } catch (err: any) {
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [page, trackFilter, workloadFilter, statusFilter]);

  const totalPages = Math.ceil(total / limit);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '2rem' }}>
      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem', flexWrap: 'wrap', gap: '1rem' }}>
          <div>
            <h1 style={{ fontSize: '1.5rem', fontWeight: 700 }}>Benchmark Execution History</h1>
            <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem' }}>
              Browse and filter all historical benchmark runs ({total} recorded).
            </p>
          </div>
          <button className="btn btn-primary" onClick={onOpenLaunch}>
            + New Benchmark
          </button>
        </div>

        {/* Filter Controls */}
        <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap', marginBottom: '1.5rem' }}>
          <div>
            <label style={{ display: 'block', fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '0.25rem' }}>
              Filter Track
            </label>
            <select
              value={trackFilter}
              onChange={(e) => {
                setTrackFilter(e.target.value);
                setPage(0);
              }}
              style={{
                padding: '0.4rem 0.8rem',
                backgroundColor: 'var(--bg-muted)',
                border: '1px solid var(--border-color)',
                borderRadius: '6px',
                color: 'var(--text-primary)',
                fontSize: '0.85rem',
              }}
            >
              <option value="">All Tracks</option>
              <option value="languages">Languages Track</option>
              <option value="lowlevel">Low-Level CPU Lab</option>
              <option value="accelerator">GPU Accelerator Lab</option>
            </select>
          </div>

          <div>
            <label style={{ display: 'block', fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '0.25rem' }}>
              Filter Workload
            </label>
            <select
              value={workloadFilter}
              onChange={(e) => {
                setWorkloadFilter(e.target.value);
                setPage(0);
              }}
              style={{
                padding: '0.4rem 0.8rem',
                backgroundColor: 'var(--bg-muted)',
                border: '1px solid var(--border-color)',
                borderRadius: '6px',
                color: 'var(--text-primary)',
                fontSize: '0.85rem',
              }}
            >
              <option value="">All Workloads</option>
              {workloads.map((w) => (
                <option key={w.id} value={w.id}>
                  {w.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label style={{ display: 'block', fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '0.25rem' }}>
              Filter Status
            </label>
            <select
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setPage(0);
              }}
              style={{
                padding: '0.4rem 0.8rem',
                backgroundColor: 'var(--bg-muted)',
                border: '1px solid var(--border-color)',
                borderRadius: '6px',
                color: 'var(--text-primary)',
                fontSize: '0.85rem',
              }}
            >
              <option value="">All Statuses</option>
              <option value="COMPLETED">Completed</option>
              <option value="FAILED">Failed</option>
              <option value="RUNNING">Running</option>
              <option value="CANCELLED">Cancelled</option>
            </select>
          </div>
        </div>

        {/* Table */}
        {isLoading ? (
          <div style={{ textAlign: 'center', padding: '3rem 1rem' }}>
            <RotateCw size={24} className="spin" style={{ margin: '0 auto 0.5rem' }} />
            <p style={{ color: 'var(--text-secondary)' }}>Loading history...</p>
          </div>
        ) : runs.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '3rem 1rem', color: 'var(--text-muted)' }}>
            No benchmark runs matched the filter criteria.
          </div>
        ) : (
          <>
            <div className="data-table-container">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Run ID</th>
                    <th>Track</th>
                    <th>Workload</th>
                    <th>Status</th>
                    <th>Iterations</th>
                    <th>Mode</th>
                    <th>Environment</th>
                    <th>Timestamp (UTC)</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {runs.map((r) => (
                    <tr key={r.id}>
                      <td>
                        <code>{r.id}</code>
                      </td>
                      <td>
                        <span
                          className={`badge ${
                            r.track === 'lowlevel'
                              ? 'badge-warning'
                              : r.track === 'accelerator'
                              ? 'badge-primary'
                              : 'badge-neutral'
                          }`}
                        >
                          {r.track === 'lowlevel' ? 'Low-Level CPU' : r.track === 'accelerator' ? 'GPU Lab' : 'Languages'}
                        </span>
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
                        {r.iterations} ({r.warmup_iterations} warmup)
                      </td>
                      <td>
                        <span className="badge badge-neutral">{r.execution_mode}</span>
                      </td>
                      <td style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                        {r.system_info.os} ({r.system_info.arch}), {r.system_info.logical_cpus} vCPUs
                      </td>
                      <td style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                        {new Date(r.created_at).toLocaleString()}
                      </td>
                      <td>
                        <button
                          className="btn btn-secondary btn-sm"
                          onClick={() => onSelectRun(r.id)}
                        >
                          Explore <ExternalLink size={12} />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '1.5rem' }}>
                <span style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                  Page {page + 1} of {totalPages}
                </span>
                <div style={{ display: 'flex', gap: '0.5rem' }}>
                  <button
                    className="btn btn-secondary btn-sm"
                    disabled={page === 0}
                    onClick={() => setPage((p) => Math.max(0, p - 1))}
                  >
                    Previous
                  </button>
                  <button
                    className="btn btn-secondary btn-sm"
                    disabled={page + 1 >= totalPages}
                    onClick={() => setPage((p) => p + 1)}
                  >
                    Next
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};
