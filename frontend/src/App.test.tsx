import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import App from './App';

// Mock fetch APIs
vi.stubGlobal('fetch', vi.fn((url: string) => {
  if (url.includes('/api/v1/stats')) {
    return Promise.resolve({
      ok: true,
      json: () => Promise.resolve({
        total_runs: 5,
        completed_runs: 4,
        failed_runs: 1,
        supported_languages: 6,
        available_languages: 6,
      }),
    });
  }
  if (url.includes('/api/v1/languages')) {
    return Promise.resolve({
      ok: true,
      json: () => Promise.resolve({
        languages: [
          { id: 'go', display_name: 'Go', is_available: true, version: 'go1.27.1', default_warmup: 0 },
          { id: 'rust', display_name: 'Rust (rustc)', is_available: true, version: 'rustc 1.97.1', default_warmup: 0 },
        ],
      }),
    });
  }
  if (url.includes('/api/v1/workloads')) {
    return Promise.resolve({
      ok: true,
      json: () => Promise.resolve({
        workloads: [
          { id: 'sieve', name: 'Sieve of Eratosthenes', parameters: [], supported_modes: ['standard'] },
        ],
      }),
    });
  }
  if (url.includes('/api/v1/runs')) {
    return Promise.resolve({
      ok: true,
      json: () => Promise.resolve({
        runs: [],
        total: 0,
      }),
    });
  }
  return Promise.reject(new Error('Unknown endpoint'));
}));

describe('ExecLens Frontend Application', () => {
  it('renders branding title and navigation', () => {
    render(<App />);
    expect(screen.getByText('ExecLens')).toBeInTheDocument();
    expect(screen.getByText('Measure fairly. Compare deeply.')).toBeInTheDocument();
    expect(screen.getByText('Dashboard')).toBeInTheDocument();
    expect(screen.getByText('Benchmark History')).toBeInTheDocument();
    expect(screen.getByText('Run Comparison')).toBeInTheDocument();
  });

  it('renders the topbar action button', () => {
    render(<App />);
    const buttons = screen.getAllByRole('button', { name: /launch/i });
    expect(buttons.length).toBeGreaterThan(0);
  });
});
