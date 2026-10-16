import React, { useEffect, useState } from 'react';
import { AlertTriangle, ArrowLeft } from 'lucide-react';
import type { Run, ComparisonReport, ComparisonWarning, RunDetail, LanguageExecution } from '../types';
import { fetchRuns, fetchComparison } from '../api';

interface CompareViewProps {
  onBack: () => void;
  onSelectRun?: (runId: string) => void;
}

export const CompareView: React.FC<CompareViewProps> = ({ onBack }) => {
  const [runs, setRuns] = useState<Run[]>([]);
  const [selectedRunIds, setSelectedRunIds] = useState<string[]>([]);
  const [comparison, setComparison] = useState<ComparisonReport | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);

  useEffect(() => {
    loadRecentRuns();
  }, []);

  const loadRecentRuns = async () => {
    try {
      setIsLoading(true);
      const res = await fetchRuns(20, 0, undefined, 'COMPLETED');
      setRuns(res.runs);
      if (res.runs.length >= 2) {
        // Pre-select first 2 completed runs
        const initialSelected = [res.runs[0].id, res.runs[1].id];
        setSelectedRunIds(initialSelected);
        loadComparison(initialSelected);
      }
    } catch (err: any) {
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  const loadComparison = async (ids: string[]) => {
    if (ids.length < 2) {
      setComparison(null);
      return;
    }
    try {
      const data = await fetchComparison(ids);
      setComparison(data);
    } catch (err: any) {
      console.error(err);
    }
  };

  const handleToggleRun = (id: string) => {
    let next: string[];
    if (selectedRunIds.includes(id)) {
      next = selectedRunIds.filter((r: string) => r !== id);
    } else {
      next = [...selectedRunIds, id];
    }
    setSelectedRunIds(next);
    if (next.length >= 2) {
      loadComparison(next);
    } else {
      setComparison(null);
    }
  };

  if (isLoading && runs.length === 0) {
    return (
      <div style={{ textAlign: 'center', padding: '3rem 1rem' }}>
        <p style={{ color: 'var(--text-secondary)' }}>Loading completed runs for comparison...</p>
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '2rem' }}>
      {/* Header */}
      <div className="card">
        <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', marginBottom: '0.5rem' }}>
          <button className="btn btn-secondary btn-sm" onClick={onBack}>
            <ArrowLeft size={16} /> Back
          </button>
          <h1 style={{ fontSize: '1.5rem', fontWeight: 700 }}>Run Comparison & Compatibility Matrix</h1>
        </div>
        <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem' }}>
          Select two or more benchmark runs to evaluate performance differences across executions, configurations, or hardware environments.
        </p>
      </div>

      {/* Run Selection Selector */}
      <div className="card">
        <h3 className="card-title">Available Completed Runs ({runs.length})</h3>
        <p className="card-desc">Check runs to include in the comparative matrix (at least 2 required):</p>

        {runs.length === 0 ? (
          <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem' }}>No completed runs found to compare.</p>
        ) : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '0.75rem' }}>
            {runs.map((r: Run) => (
              <label
                key={r.id}
                style={{
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: '0.75rem',
                  padding: '0.75rem',
                  backgroundColor: selectedRunIds.includes(r.id) ? 'var(--accent-glow)' : 'var(--bg-muted)',
                  border: '1px solid',
                  borderColor: selectedRunIds.includes(r.id) ? 'var(--accent-primary)' : 'var(--border-color)',
                  borderRadius: '8px',
                  cursor: 'pointer',
                }}
              >
                <input
                  type="checkbox"
                  checked={selectedRunIds.includes(r.id)}
                  onChange={() => handleToggleRun(r.id)}
                  style={{ marginTop: '0.2rem' }}
                />
                <div>
                  <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                    <code style={{ fontWeight: 600 }}>{r.id}</code>
                    <span className="badge badge-neutral">{r.workload_id}</span>
                  </div>
                  <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginTop: '0.25rem' }}>
                    Mode: {r.execution_mode} &bull; {r.iterations} iters
                  </div>
                  <div style={{ fontSize: '0.7rem', color: 'var(--text-muted)' }}>
                    {new Date(r.created_at).toLocaleString()}
                  </div>
                </div>
              </label>
            ))}
          </div>
        )}
      </div>

      {/* Compatibility Warnings Banner */}
      {comparison && comparison.warnings.length > 0 && (
        <div className="card" style={{ backgroundColor: 'var(--warning-bg)', borderColor: 'var(--warning)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.5rem', color: 'var(--warning)' }}>
            <AlertTriangle size={20} />
            <h3 style={{ fontSize: '1rem', fontWeight: 700 }}>Comparability Warnings Detected</h3>
          </div>
          <p style={{ fontSize: '0.85rem', color: 'var(--text-primary)', marginBottom: '0.75rem' }}>
            Fair benchmark comparison requires equivalent workloads, matching input parameters, and stable hardware conditions:
          </p>
          <ul style={{ paddingLeft: '1.5rem', fontSize: '0.85rem', color: 'var(--text-primary)' }}>
            {comparison.warnings.map((w: ComparisonWarning, idx: number) => (
              <li key={idx} style={{ marginBottom: '0.25rem' }}>
                <strong>[{w.type}]</strong> {w.message}
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Comparison Results Matrix */}
      {comparison && comparison.runs.length > 0 && (
        <div className="card">
          <h3 className="card-title">Side-by-Side Performance Comparison</h3>
          <p className="card-desc">Comparing median execution duration (ms) across selected runs.</p>

          <div className="data-table-container">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Language</th>
                  {comparison.runs.map((r: RunDetail) => (
                    <th key={r.id}>
                      <div>{r.id}</div>
                      <div style={{ fontSize: '0.7rem', color: 'var(--text-muted)', fontWeight: 400 }}>
                        {r.workload_id} ({r.execution_mode})
                      </div>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {['cpp', 'rust', 'go', 'java', 'python', 'javascript'].map((langId) => (
                  <tr key={langId}>
                    <td>
                      <strong style={{ textTransform: 'uppercase' }}>{langId}</strong>
                    </td>
                    {comparison.runs.map((r: RunDetail) => {
                      const exec = r.executions.find((e: LanguageExecution) => e.language_id === langId);
                      return (
                        <td key={r.id}>
                          {exec && exec.summary ? (
                            <div>
                              <strong>{exec.summary.median_ms.toFixed(2)} ms</strong>
                              <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
                                ±{exec.summary.std_dev_ms.toFixed(2)} ms &bull; {exec.peak_memory_mb.toFixed(1)} MB
                              </div>
                            </div>
                          ) : (
                            <span style={{ color: 'var(--text-muted)' }}>
                              {exec ? exec.status : 'Not Run'}
                            </span>
                          )}
                        </td>
                      );
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};
