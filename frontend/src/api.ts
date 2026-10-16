import type {
  DashboardStats,
  RuntimeInfo,
  WorkloadContract,
  Run,
  RunDetail,
  ComparisonReport,
  TrackInfo,
  LowLevelWorkload,
  CompilerInfo,
  AcceleratorBackend,
  AcceleratorWorkload,
} from './types';

const API_BASE = '/api/v1';

export async function fetchStats(): Promise<DashboardStats> {
  const res = await fetch(`${API_BASE}/stats`);
  if (!res.ok) throw new Error(`Failed to fetch stats: ${res.statusText}`);
  return res.json();
}

export async function fetchTracks(): Promise<TrackInfo[]> {
  const res = await fetch(`${API_BASE}/tracks`);
  if (!res.ok) throw new Error(`Failed to fetch tracks: ${res.statusText}`);
  const data = await res.json();
  return data.tracks || [];
}

export async function fetchLanguages(): Promise<RuntimeInfo[]> {
  const res = await fetch(`${API_BASE}/languages`);
  if (!res.ok) throw new Error(`Failed to fetch languages: ${res.statusText}`);
  const data = await res.json();
  return data.languages || [];
}

export async function fetchWorkloads(): Promise<WorkloadContract[]> {
  const res = await fetch(`${API_BASE}/workloads`);
  if (!res.ok) throw new Error(`Failed to fetch workloads: ${res.statusText}`);
  const data = await res.json();
  return data.workloads || [];
}

export async function fetchLowLevelCatalog(): Promise<{ workloads: LowLevelWorkload[]; compilers: Record<string, CompilerInfo> }> {
  const res = await fetch(`${API_BASE}/lowlevel/catalog`);
  if (!res.ok) throw new Error(`Failed to fetch low-level catalog: ${res.statusText}`);
  return res.json();
}

export async function fetchAccelerators(): Promise<AcceleratorBackend[]> {
  const res = await fetch(`${API_BASE}/accelerators`);
  if (!res.ok) throw new Error(`Failed to fetch accelerators: ${res.statusText}`);
  const data = await res.json();
  return data.accelerators || [];
}

export async function fetchAcceleratorCatalog(): Promise<AcceleratorWorkload[]> {
  const res = await fetch(`${API_BASE}/accelerators/catalog`);
  if (!res.ok) throw new Error(`Failed to fetch accelerator catalog: ${res.statusText}`);
  const data = await res.json();
  return data.workloads || [];
}

export async function fetchRuns(limit = 20, offset = 0, track?: string, workloadId?: string, status?: string): Promise<{ runs: Run[]; total: number }> {
  const params = new URLSearchParams({
    limit: limit.toString(),
    offset: offset.toString(),
  });
  if (track) params.append('track', track);
  if (workloadId) params.append('workload_id', workloadId);
  if (status) params.append('status', status);

  const res = await fetch(`${API_BASE}/runs?${params.toString()}`);
  if (!res.ok) throw new Error(`Failed to fetch runs: ${res.statusText}`);
  return res.json();
}

export async function fetchRun(runId: string, baseline?: string): Promise<RunDetail> {
  const url = baseline
    ? `${API_BASE}/runs/${runId}?baseline=${encodeURIComponent(baseline)}`
    : `${API_BASE}/runs/${runId}`;
  const res = await fetch(url);
  if (!res.ok) throw new Error(`Failed to fetch run ${runId}: ${res.statusText}`);
  return res.json();
}

export async function createRun(payload: {
  track?: string;
  workload_id: string;
  languages: string[];
  workload_params?: Record<string, any>;
  iterations: number;
  warmup_iterations?: number;
  execution_mode?: string;
}): Promise<Run> {
  const res = await fetch(`${API_BASE}/runs`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const errData = await res.json().catch(() => ({}));
    throw new Error(errData.error || `Failed to create run: ${res.statusText}`);
  }
  return res.json();
}

export async function cancelRun(runId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/runs/${runId}/cancel`, { method: 'POST' });
  if (!res.ok) throw new Error(`Failed to cancel run: ${res.statusText}`);
}

export async function fetchComparison(runIds: string[]): Promise<ComparisonReport> {
  const res = await fetch(`${API_BASE}/compare?run_ids=${runIds.join(',')}`);
  if (!res.ok) throw new Error(`Failed to compare runs: ${res.statusText}`);
  return res.json();
}

export async function fetchReport(runId: string): Promise<string> {
  const res = await fetch(`${API_BASE}/runs/${runId}/report`);
  if (!res.ok) throw new Error(`Failed to fetch report: ${res.statusText}`);
  return res.text();
}
