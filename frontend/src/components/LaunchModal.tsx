import React, { useState } from 'react';
import { X, Play, AlertCircle, Info } from 'lucide-react';
import type { WorkloadContract, RuntimeInfo } from '../types';
import { createRun } from '../api';

interface LaunchModalProps {
  isOpen: boolean;
  onClose: () => void;
  workloads: WorkloadContract[];
  languages: RuntimeInfo[];
  onRunCreated: (runId: string) => void;
}

export const LaunchModal: React.FC<LaunchModalProps> = ({
  isOpen,
  onClose,
  workloads,
  languages,
  onRunCreated,
}) => {
  const [selectedWorkload, setSelectedWorkload] = useState<string>('sieve');
  const [selectedLangs, setSelectedLangs] = useState<string[]>(['cpp', 'rust', 'go', 'java', 'python', 'javascript']);
  const [iterations, setIterations] = useState<number>(5);
  const [warmup, setWarmup] = useState<number>(2);
  const [executionMode, setExecutionMode] = useState<string>('standard');
  const [params, setParams] = useState<Record<string, any>>({});
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen) return null;

  const currentWorkload = workloads.find((w) => w.id === selectedWorkload) || workloads[0];

  const handleToggleLang = (id: string) => {
    if (selectedLangs.includes(id)) {
      if (selectedLangs.length > 1) {
        setSelectedLangs(selectedLangs.filter((l) => l !== id));
      }
    } else {
      setSelectedLangs([...selectedLangs, id]);
    }
  };

  const handleParamChange = (name: string, value: any) => {
    setParams((prev) => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      // Merge defaults with custom params
      const workloadParams: Record<string, any> = {};
      if (currentWorkload) {
        currentWorkload.parameters.forEach((p) => {
          workloadParams[p.name] = params[p.name] !== undefined ? params[p.name] : p.default;
        });
      }

      const run = await createRun({
        workload_id: selectedWorkload,
        languages: selectedLangs,
        workload_params: workloadParams,
        iterations: Number(iterations),
        warmup_iterations: Number(warmup),
        execution_mode: executionMode,
      });

      onRunCreated(run.id);
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to start benchmark');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem' }}>
          <div>
            <h2 style={{ fontSize: '1.25rem', fontWeight: 700 }}>Launch Cross-Language Benchmark</h2>
            <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
              Configure workload parameters and select runtime environments.
            </p>
          </div>
          <button className="btn btn-secondary btn-sm" onClick={onClose} style={{ padding: '0.4rem' }}>
            <X size={18} />
          </button>
        </div>

        {error && (
          <div
            style={{
              padding: '0.75rem 1rem',
              backgroundColor: 'var(--error-bg)',
              border: '1px solid var(--error)',
              borderRadius: '8px',
              color: 'var(--error)',
              fontSize: '0.85rem',
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem',
              marginBottom: '1.5rem',
            }}
          >
            <AlertCircle size={16} /> {error}
          </div>
        )}

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
          {/* Workload Selection */}
          <div>
            <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '0.5rem' }}>
              Select Workload
            </label>
            <select
              value={selectedWorkload}
              onChange={(e) => setSelectedWorkload(e.target.value)}
              style={{
                width: '100%',
                padding: '0.65rem 0.85rem',
                backgroundColor: 'var(--bg-muted)',
                border: '1px solid var(--border-color)',
                borderRadius: '8px',
                color: 'var(--text-primary)',
                fontSize: '0.9rem',
              }}
            >
              {workloads.map((w) => (
                <option key={w.id} value={w.id}>
                  {w.name} ({w.id})
                </option>
              ))}
            </select>
            {currentWorkload && (
              <div
                style={{
                  marginTop: '0.5rem',
                  padding: '0.75rem',
                  backgroundColor: 'var(--bg-muted)',
                  borderRadius: '6px',
                  fontSize: '0.8rem',
                  color: 'var(--text-secondary)',
                }}
              >
                <p style={{ marginBottom: '0.25rem' }}>
                  <strong>Purpose:</strong> {currentWorkload.algorithm_purpose}
                </p>
                <p style={{ color: 'var(--text-muted)' }}>
                  <Info size={12} style={{ display: 'inline', marginRight: '4px' }} />
                  {currentWorkload.fairness_caveats}
                </p>
              </div>
            )}
          </div>

          {/* Dynamic Workload Parameters */}
          {currentWorkload && currentWorkload.parameters.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
              <label style={{ fontSize: '0.85rem', fontWeight: 600 }}>Workload Parameters</label>
              {currentWorkload.parameters.map((p) => (
                <div key={p.name} style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.8rem' }}>
                    <span>{p.description}</span>
                    <span style={{ color: 'var(--text-muted)' }}>Default: {p.default}</span>
                  </div>
                  {p.name === 'mode' ? (
                    <select
                      value={params[p.name] || p.default}
                      onChange={(e) => handleParamChange(p.name, e.target.value)}
                      style={{
                        padding: '0.5rem',
                        backgroundColor: 'var(--bg-muted)',
                        border: '1px solid var(--border-color)',
                        borderRadius: '6px',
                        color: 'var(--text-primary)',
                      }}
                    >
                      <option value="quicksort">quicksort (equivalent algorithmic implementation)</option>
                      <option value="stdlib">stdlib (runtime standard library sort)</option>
                    </select>
                  ) : (
                    <input
                      type={p.type === 'int' ? 'number' : 'text'}
                      defaultValue={p.default}
                      onChange={(e) =>
                        handleParamChange(p.name, p.type === 'int' ? parseInt(e.target.value, 10) : e.target.value)
                      }
                      style={{
                        padding: '0.5rem',
                        backgroundColor: 'var(--bg-muted)',
                        border: '1px solid var(--border-color)',
                        borderRadius: '6px',
                        color: 'var(--text-primary)',
                      }}
                    />
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Language Selection */}
          <div>
            <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '0.5rem' }}>
              Included Languages ({selectedLangs.length} selected)
            </label>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '0.5rem' }}>
              {languages.map((l) => (
                <label
                  key={l.id}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '0.5rem',
                    padding: '0.5rem 0.75rem',
                    backgroundColor: selectedLangs.includes(l.id) ? 'var(--accent-glow)' : 'var(--bg-muted)',
                    border: '1px solid',
                    borderColor: selectedLangs.includes(l.id) ? 'var(--accent-primary)' : 'var(--border-color)',
                    borderRadius: '6px',
                    cursor: l.is_available ? 'pointer' : 'not-allowed',
                    opacity: l.is_available ? 1 : 0.5,
                  }}
                >
                  <input
                    type="checkbox"
                    checked={selectedLangs.includes(l.id)}
                    disabled={!l.is_available}
                    onChange={() => handleToggleLang(l.id)}
                  />
                  <span style={{ fontSize: '0.85rem', fontWeight: 500 }}>{l.display_name}</span>
                  {!l.is_available && (
                    <span style={{ fontSize: '0.7rem', color: 'var(--error)', marginLeft: 'auto' }}>Missing</span>
                  )}
                </label>
              ))}
            </div>
          </div>

          {/* Execution Configuration: Iterations & Warmup */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '1rem' }}>
            <div>
              <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.25rem' }}>
                Timed Iterations
              </label>
              <input
                type="number"
                min={1}
                max={50}
                value={iterations}
                onChange={(e) => setIterations(Math.max(1, parseInt(e.target.value, 10) || 1))}
                style={{
                  width: '100%',
                  padding: '0.5rem',
                  backgroundColor: 'var(--bg-muted)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '6px',
                  color: 'var(--text-primary)',
                }}
              />
            </div>

            <div>
              <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.25rem' }}>
                Warmup Iterations
              </label>
              <input
                type="number"
                min={0}
                max={10}
                value={warmup}
                onChange={(e) => setWarmup(Math.max(0, parseInt(e.target.value, 10) || 0))}
                style={{
                  width: '100%',
                  padding: '0.5rem',
                  backgroundColor: 'var(--bg-muted)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '6px',
                  color: 'var(--text-primary)',
                }}
              />
            </div>

            <div>
              <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.25rem' }}>
                Execution Mode
              </label>
              <select
                value={executionMode}
                onChange={(e) => setExecutionMode(e.target.value)}
                style={{
                  width: '100%',
                  padding: '0.5rem',
                  backgroundColor: 'var(--bg-muted)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '6px',
                  color: 'var(--text-primary)',
                }}
              >
                <option value="standard">Standard (Warm)</option>
                <option value="cold_start">Cold Start (0 warmup)</option>
              </select>
            </div>
          </div>

          {/* Action Buttons */}
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '0.75rem', marginTop: '1rem' }}>
            <button type="button" className="btn btn-secondary" onClick={onClose} disabled={isSubmitting}>
              Cancel
            </button>
            <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
              {isSubmitting ? (
                'Submitting...'
              ) : (
                <>
                  <Play size={16} /> Start Benchmark
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
