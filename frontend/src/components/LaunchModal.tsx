import React, { useState, useEffect } from 'react';
import { X, Play, AlertCircle, Cpu, Zap, Layers } from 'lucide-react';
import type { WorkloadContract, RuntimeInfo, LowLevelWorkload, CompilerInfo, AcceleratorBackend, AcceleratorWorkload } from '../types';
import { createRun, fetchLowLevelCatalog, fetchAccelerators, fetchAcceleratorCatalog } from '../api';

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
  const [activeTrack, setActiveTrack] = useState<'languages' | 'lowlevel' | 'accelerator'>('languages');

  // Languages Track State
  const [selectedWorkload, setSelectedWorkload] = useState<string>('sieve');
  const [selectedLangs, setSelectedLangs] = useState<string[]>(['cpp', 'rust', 'go', 'java', 'python', 'javascript']);
  const [iterations, setIterations] = useState<number>(5);
  const [warmup, setWarmup] = useState<number>(2);
  const [executionMode, setExecutionMode] = useState<string>('standard');
  const [params, setParams] = useState<Record<string, any>>({});

  // Low-Level Lab State
  const [lowLevelWorkloads, setLowLevelWorkloads] = useState<LowLevelWorkload[]>([]);
  const [compilers, setCompilers] = useState<Record<string, CompilerInfo>>({});
  const [selectedLLWorkload, setSelectedLLWorkload] = useState<string>('lowlevel_reduction');
  const [selectedVariants, setSelectedVariants] = useState<string[]>(['scalar', 'autovec', 'simd_avx2', 'openmp', 'asm']);
  const [llSize, setLLSize] = useState<number>(10000000);

  // GPU Accelerator State
  const [accelerators, setAccelerators] = useState<AcceleratorBackend[]>([]);
  const [accelWorkloads, setAccelWorkloads] = useState<AcceleratorWorkload[]>([]);
  const [selectedAccelWorkload, setSelectedAccelWorkload] = useState<string>('cuda_vec_add_saxpy');
  const [accelSize, setAccelSize] = useState<number>(16777216);

  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      fetchLowLevelCatalog()
        .then((res) => {
          setLowLevelWorkloads(res.workloads);
          setCompilers(res.compilers);
        })
        .catch(console.error);

      fetchAccelerators().then(setAccelerators).catch(console.error);
      fetchAcceleratorCatalog().then(setAccelWorkloads).catch(console.error);
    }
  }, [isOpen]);

  if (!isOpen) return null;

  const currentWorkload = workloads.find((w) => w.id === selectedWorkload) || workloads[0];
  const currentLLWorkload = lowLevelWorkloads.find((w) => w.id === selectedLLWorkload) || lowLevelWorkloads[0];
  const currentAccelWorkload = accelWorkloads.find((w) => w.id === selectedAccelWorkload) || accelWorkloads[0];
  const cudaBackend = accelerators.find((a) => a.id === 'cuda');

  const handleToggleLang = (id: string) => {
    if (selectedLangs.includes(id)) {
      if (selectedLangs.length > 1) {
        setSelectedLangs(selectedLangs.filter((l) => l !== id));
      }
    } else {
      setSelectedLangs([...selectedLangs, id]);
    }
  };

  const handleToggleVariant = (id: string) => {
    if (selectedVariants.includes(id)) {
      if (selectedVariants.length > 1) {
        setSelectedVariants(selectedVariants.filter((v) => v !== id));
      }
    } else {
      setSelectedVariants([...selectedVariants, id]);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      if (activeTrack === 'languages') {
        const workloadParams: Record<string, any> = {};
        if (currentWorkload) {
          currentWorkload.parameters.forEach((p) => {
            workloadParams[p.name] = params[p.name] !== undefined ? params[p.name] : p.default;
          });
        }

        const run = await createRun({
          track: 'languages',
          workload_id: selectedWorkload,
          languages: selectedLangs,
          workload_params: workloadParams,
          iterations: Number(iterations),
          warmup_iterations: Number(warmup),
          execution_mode: executionMode,
        });

        onRunCreated(run.id);
        onClose();
      } else if (activeTrack === 'lowlevel') {
        const run = await createRun({
          track: 'lowlevel',
          workload_id: selectedLLWorkload,
          languages: selectedVariants,
          workload_params: { size: Number(llSize) },
          iterations: Number(iterations),
          warmup_iterations: 1,
          execution_mode: 'standard',
        });

        onRunCreated(run.id);
        onClose();
      } else if (activeTrack === 'accelerator') {
        const run = await createRun({
          track: 'accelerator',
          workload_id: selectedAccelWorkload,
          languages: ['cuda'],
          workload_params: { size: Number(accelSize) },
          iterations: Number(iterations),
          warmup_iterations: 1,
          execution_mode: 'standard',
        });

        onRunCreated(run.id);
        onClose();
      }
    } catch (err: any) {
      setError(err.message || 'Failed to start benchmark');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="modal-overlay">
      <div className="modal-content" style={{ maxWidth: '640px', maxHeight: '90vh', overflowY: 'auto' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
          <div>
            <h2 style={{ fontSize: '1.25rem', fontWeight: 700 }}>Launch Performance Benchmark</h2>
            <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
              Select experimental track, workload, and target execution options.
            </p>
          </div>
          <button className="btn btn-secondary btn-sm" onClick={onClose} style={{ padding: '0.4rem' }}>
            <X size={18} />
          </button>
        </div>

        {/* Track Selection Tabs */}
        <div style={{ display: 'flex', gap: '0.5rem', marginBottom: '1.5rem', borderBottom: '1px solid var(--border-color)', paddingBottom: '0.75rem' }}>
          <button
            type="button"
            className={`btn btn-sm ${activeTrack === 'languages' ? 'btn-primary' : 'btn-secondary'}`}
            onClick={() => setActiveTrack('languages')}
          >
            <Layers size={14} /> Languages Suite
          </button>
          <button
            type="button"
            className={`btn btn-sm ${activeTrack === 'lowlevel' ? 'btn-primary' : 'btn-secondary'}`}
            onClick={() => setActiveTrack('lowlevel')}
          >
            <Cpu size={14} /> Low-Level CPU Lab
          </button>
          <button
            type="button"
            className={`btn btn-sm ${activeTrack === 'accelerator' ? 'btn-primary' : 'btn-secondary'}`}
            onClick={() => setActiveTrack('accelerator')}
          >
            <Zap size={14} /> GPU Accelerator Lab
          </button>
        </div>

        {error && (
          <div
            style={{
              padding: '0.75rem 1rem',
              backgroundColor: 'var(--error-bg)',
              border: '1px solid var(--error)',
              borderRadius: '6px',
              color: 'var(--error)',
              fontSize: '0.85rem',
              marginBottom: '1rem',
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem',
            }}
          >
            <AlertCircle size={16} />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>
          {/* TRACK 1: LANGUAGES */}
          {activeTrack === 'languages' && (
            <>
              <div>
                <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.4rem' }}>
                  Workload
                </label>
                <select
                  value={selectedWorkload}
                  onChange={(e) => {
                    setSelectedWorkload(e.target.value);
                    setParams({});
                  }}
                  style={{
                    width: '100%',
                    padding: '0.5rem 0.75rem',
                    backgroundColor: 'var(--bg-muted)',
                    border: '1px solid var(--border-color)',
                    borderRadius: '6px',
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
                  <p style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginTop: '0.35rem' }}>
                    {currentWorkload.description}
                  </p>
                )}
              </div>

              <div>
                <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.4rem' }}>
                  Target Languages ({selectedLangs.length} selected)
                </label>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '0.5rem' }}>
                  {languages.map((l) => (
                    <button
                      key={l.id}
                      type="button"
                      disabled={!l.is_available}
                      onClick={() => handleToggleLang(l.id)}
                      style={{
                        padding: '0.5rem',
                        borderRadius: '6px',
                        border: selectedLangs.includes(l.id)
                          ? '1px solid var(--accent-primary)'
                          : '1px solid var(--border-color)',
                        backgroundColor: selectedLangs.includes(l.id)
                          ? 'rgba(56, 189, 248, 0.1)'
                          : 'var(--bg-muted)',
                        color: selectedLangs.includes(l.id) ? 'var(--accent-primary)' : 'var(--text-primary)',
                        fontSize: '0.8rem',
                        fontWeight: 600,
                        cursor: l.is_available ? 'pointer' : 'not-allowed',
                        opacity: l.is_available ? 1 : 0.4,
                        textAlign: 'center',
                      }}
                    >
                      {l.display_name}
                    </button>
                  ))}
                </div>
              </div>
            </>
          )}

          {/* TRACK 2: LOW-LEVEL CPU LAB */}
          {activeTrack === 'lowlevel' && (
            <>
              <div>
                <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.4rem' }}>
                  Low-Level CPU Workload
                </label>
                <select
                  value={selectedLLWorkload}
                  onChange={(e) => {
                    setSelectedLLWorkload(e.target.value);
                    const found = lowLevelWorkloads.find((w) => w.id === e.target.value);
                    if (found) {
                      setSelectedVariants(found.variants.map((v) => v.id));
                      setLLSize(found.default_size);
                    }
                  }}
                  style={{
                    width: '100%',
                    padding: '0.5rem 0.75rem',
                    backgroundColor: 'var(--bg-muted)',
                    border: '1px solid var(--border-color)',
                    borderRadius: '6px',
                    color: 'var(--text-primary)',
                    fontSize: '0.9rem',
                  }}
                >
                  {lowLevelWorkloads.map((w) => (
                    <option key={w.id} value={w.id}>
                      {w.name} ({w.id})
                    </option>
                  ))}
                </select>
                {currentLLWorkload && (
                  <p style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginTop: '0.35rem' }}>
                    {currentLLWorkload.description}
                  </p>
                )}
              </div>

              <div>
                <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.4rem' }}>
                  Target Implementations / Variants ({selectedVariants.length} selected)
                </label>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.4rem' }}>
                  {currentLLWorkload?.variants.map((v) => (
                    <label
                      key={v.id}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '0.5rem',
                        padding: '0.4rem 0.6rem',
                        borderRadius: '4px',
                        backgroundColor: selectedVariants.includes(v.id) ? 'rgba(56, 189, 248, 0.08)' : 'var(--bg-muted)',
                        cursor: 'pointer',
                        fontSize: '0.8rem',
                      }}
                    >
                      <input
                        type="checkbox"
                        checked={selectedVariants.includes(v.id)}
                        onChange={() => handleToggleVariant(v.id)}
                      />
                      <div>
                        <strong>{v.name}</strong> — <span style={{ color: 'var(--text-secondary)' }}>{v.description}</span>
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              <div>
                <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.4rem' }}>
                  Input Size ({currentLLWorkload?.unit_name || 'elements'})
                </label>
                <input
                  type="number"
                  value={llSize}
                  onChange={(e) => setLLSize(Number(e.target.value))}
                  style={{
                    width: '100%',
                    padding: '0.4rem 0.6rem',
                    backgroundColor: 'var(--bg-muted)',
                    border: '1px solid var(--border-color)',
                    borderRadius: '6px',
                    color: 'var(--text-primary)',
                  }}
                />
              </div>

              <div style={{ padding: '0.5rem 0.75rem', backgroundColor: 'var(--bg-muted)', borderRadius: '6px', fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
                <strong>Host Compilers:</strong> GCC ({compilers.gcc?.is_available ? '✅ Available' : '❌'}), Clang ({compilers.clang?.is_available ? '✅ Available' : '❌'}). Both Intel syntax assembly and compiler remarks are captured automatically.
              </div>
            </>
          )}

          {/* TRACK 3: GPU ACCELERATOR LAB */}
          {activeTrack === 'accelerator' && (
            <>
              <div>
                <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.4rem' }}>
                  GPU Compute Workload
                </label>
                <select
                  value={selectedAccelWorkload}
                  onChange={(e) => {
                    setSelectedAccelWorkload(e.target.value);
                    const found = accelWorkloads.find((w) => w.id === e.target.value);
                    if (found) setAccelSize(found.default_size);
                  }}
                  style={{
                    width: '100%',
                    padding: '0.5rem 0.75rem',
                    backgroundColor: 'var(--bg-muted)',
                    border: '1px solid var(--border-color)',
                    borderRadius: '6px',
                    color: 'var(--text-primary)',
                    fontSize: '0.9rem',
                  }}
                >
                  {accelWorkloads.map((w) => (
                    <option key={w.id} value={w.id}>
                      {w.name}
                    </option>
                  ))}
                </select>
                {currentAccelWorkload && (
                  <p style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginTop: '0.35rem' }}>
                    {currentAccelWorkload.description}
                  </p>
                )}
              </div>

              {/* Hardware & Diagnostics Status Banner */}
              <div
                style={{
                  padding: '0.75rem',
                  borderRadius: '6px',
                  border: cudaBackend?.is_available ? '1px solid var(--success)' : '1px solid var(--warning)',
                  backgroundColor: cudaBackend?.is_available ? 'rgba(52, 211, 153, 0.08)' : 'rgba(251, 191, 36, 0.08)',
                  fontSize: '0.8rem',
                }}
              >
                <div style={{ fontWeight: 600, marginBottom: '0.25rem' }}>
                  {cudaBackend?.is_available ? '✅ CUDA Hardware Detected & Supported' : '⚠️ Accelerator Diagnostic Notice'}
                </div>
                <div>
                  {cudaBackend?.is_available ? (
                    <span>Device: {cudaBackend.device_name} (CC {cudaBackend.compute_capability}, Driver {cudaBackend.driver_version})</span>
                  ) : (
                    <span>{cudaBackend?.missing_reason || 'CUDA adapter unavailable on host.'}</span>
                  )}
                </div>
              </div>

              <div>
                <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.4rem' }}>
                  Dataset / Dimension Size
                </label>
                <input
                  type="number"
                  value={accelSize}
                  onChange={(e) => setAccelSize(Number(e.target.value))}
                  style={{
                    width: '100%',
                    padding: '0.4rem 0.6rem',
                    backgroundColor: 'var(--bg-muted)',
                    border: '1px solid var(--border-color)',
                    borderRadius: '6px',
                    color: 'var(--text-primary)',
                  }}
                />
              </div>
            </>
          )}

          {/* Common Iterations Controls */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem', borderTop: '1px solid var(--border-color)', paddingTop: '1rem' }}>
            <div>
              <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.4rem' }}>
                Timed Iterations (3-20)
              </label>
              <input
                type="number"
                min={1}
                max={20}
                value={iterations}
                onChange={(e) => setIterations(Number(e.target.value))}
                style={{
                  width: '100%',
                  padding: '0.4rem 0.6rem',
                  backgroundColor: 'var(--bg-muted)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '6px',
                  color: 'var(--text-primary)',
                }}
              />
            </div>

            <div>
              <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.4rem' }}>
                Warmup Iterations (0-5)
              </label>
              <input
                type="number"
                min={0}
                max={5}
                value={warmup}
                onChange={(e) => setWarmup(Number(e.target.value))}
                style={{
                  width: '100%',
                  padding: '0.4rem 0.6rem',
                  backgroundColor: 'var(--bg-muted)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '6px',
                  color: 'var(--text-primary)',
                }}
              />
            </div>

            <div>
              <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: 600, marginBottom: '0.4rem' }}>
                Execution Mode
              </label>
              <select
                value={executionMode}
                onChange={(e) => setExecutionMode(e.target.value)}
                style={{
                  width: '100%',
                  padding: '0.4rem 0.6rem',
                  backgroundColor: 'var(--bg-muted)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '6px',
                  color: 'var(--text-primary)',
                  fontSize: '0.85rem',
                }}
              >
                <option value="standard">Standard (Warmed-up)</option>
                <option value="cold_start">Cold Start (No warmup)</option>
                <option value="quicksort">Quicksort Algorithm</option>
                <option value="stdlib">Standard Library Sort</option>
              </select>
            </div>
          </div>

          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '0.75rem', marginTop: '0.5rem' }}>
            <button type="button" className="btn btn-secondary" onClick={onClose} disabled={isSubmitting}>
              Cancel
            </button>
            <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
              <Play size={16} />
              {isSubmitting ? 'Starting...' : 'Execute Benchmark'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
