import React, { useEffect, useState } from 'react';
import {
  Activity,
  Layers,
  BarChart3,
  GitCompare,
  Moon,
  Sun,
  Play,
} from 'lucide-react';
import type { DashboardStats, RuntimeInfo, WorkloadContract, Run } from './types';
import { fetchStats, fetchLanguages, fetchWorkloads, fetchRuns } from './api';
import { DashboardView } from './components/DashboardView';
import { LaunchModal } from './components/LaunchModal';
import { RunDetailView } from './components/RunDetailView';
import { CompareView } from './components/CompareView';
import { HistoryView } from './components/HistoryView';

type ViewMode = 'dashboard' | 'runs' | 'run-detail' | 'compare';

export const App: React.FC = () => {
  const [view, setView] = useState<ViewMode>('dashboard');
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const [isLaunchOpen, setIsLaunchOpen] = useState<boolean>(false);
  const [theme, setTheme] = useState<'dark' | 'light'>('dark');

  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [languages, setLanguages] = useState<RuntimeInfo[]>([]);
  const [workloads, setWorkloads] = useState<WorkloadContract[]>([]);
  const [recentRuns, setRecentRuns] = useState<Run[]>([]);

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  const toggleTheme = () => {
    setTheme((t) => (t === 'dark' ? 'light' : 'dark'));
  };

  const loadData = async () => {
    try {
      const [s, l, w, r] = await Promise.all([
        fetchStats(),
        fetchLanguages(),
        fetchWorkloads(),
        fetchRuns(10, 0),
      ]);
      setStats(s);
      setLanguages(l);
      setWorkloads(w);
      setRecentRuns(r.runs);
    } catch (err) {
      console.error('Error loading initial data:', err);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleSelectRun = (runId: string) => {
    setSelectedRunId(runId);
    setView('run-detail');
  };

  const handleRunCreated = (runId: string) => {
    loadData();
    handleSelectRun(runId);
  };

  return (
    <div className="app-container">
      {/* Sidebar Navigation */}
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-icon">
            <Activity size={20} />
          </div>
          <div>
            <div className="brand-title">ExecLens</div>
            <div className="brand-tagline">Measure fairly. Compare deeply.</div>
          </div>
        </div>

        <nav className="nav-links">
          <button
            className={`nav-item ${view === 'dashboard' ? 'active' : ''}`}
            onClick={() => setView('dashboard')}
          >
            <Layers size={18} />
            <span>Dashboard</span>
          </button>

          <button
            className={`nav-item ${view === 'runs' ? 'active' : ''}`}
            onClick={() => setView('runs')}
          >
            <BarChart3 size={18} />
            <span>Benchmark History</span>
          </button>

          <button
            className={`nav-item ${view === 'compare' ? 'active' : ''}`}
            onClick={() => setView('compare')}
          >
            <GitCompare size={18} />
            <span>Run Comparison</span>
          </button>
        </nav>

        {/* Quick Launch Button in Sidebar */}
        <div style={{ marginTop: 'auto', paddingTop: '1.5rem', borderTop: '1px solid var(--border-color)' }}>
          <button
            className="btn btn-primary"
            style={{ width: '100%', justifyContent: 'center', marginBottom: '1rem' }}
            onClick={() => setIsLaunchOpen(true)}
          >
            <Play size={16} /> New Benchmark
          </button>

          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>v1.0.0</span>
            <button
              className="btn btn-secondary btn-sm"
              onClick={toggleTheme}
              title={`Switch to ${theme === 'dark' ? 'Light' : 'Dark'} mode`}
              style={{ padding: '0.35rem 0.6rem' }}
            >
              {theme === 'dark' ? <Sun size={14} /> : <Moon size={14} />}
            </button>
          </div>
        </div>
      </aside>

      {/* Main Content Area */}
      <main className="main-content">
        <header className="topbar">
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
            <span style={{ fontWeight: 600, fontSize: '0.9rem', color: 'var(--text-secondary)' }}>
              {view === 'dashboard'
                ? 'System Overview'
                : view === 'runs'
                ? 'Historical Records'
                : view === 'compare'
                ? 'Comparative Analysis'
                : 'Execution Results'}
            </span>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
            <button className="btn btn-primary btn-sm" onClick={() => setIsLaunchOpen(true)}>
              <Play size={14} /> Launch
            </button>
          </div>
        </header>

        <div className="content-body">
          {view === 'dashboard' && (
            <DashboardView
              stats={stats}
              languages={languages}
              recentRuns={recentRuns}
              onOpenLaunch={() => setIsLaunchOpen(true)}
              onSelectRun={handleSelectRun}
              onNavigateRuns={() => setView('runs')}
            />
          )}

          {view === 'runs' && (
            <HistoryView
              workloads={workloads}
              onSelectRun={handleSelectRun}
              onOpenLaunch={() => setIsLaunchOpen(true)}
            />
          )}

          {view === 'run-detail' && selectedRunId && (
            <RunDetailView
              runId={selectedRunId}
              onBack={() => setView('dashboard')}
            />
          )}

          {view === 'compare' && (
            <CompareView
              onBack={() => setView('dashboard')}
              onSelectRun={handleSelectRun}
            />
          )}
        </div>
      </main>

      {/* Launch Benchmark Modal */}
      <LaunchModal
        isOpen={isLaunchOpen}
        onClose={() => setIsLaunchOpen(false)}
        workloads={workloads}
        languages={languages}
        onRunCreated={handleRunCreated}
      />
    </div>
  );
};

export default App;
