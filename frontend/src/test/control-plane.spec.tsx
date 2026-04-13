import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ScopeProvider } from '../context/ScopeContext';
import { api } from '../lib/api';
import { ControlPlaneDashboard } from '../modules/control-plane/ControlPlaneDashboard';

describe('ControlPlaneDashboard', () => {
  beforeEach(() => {
    api.resetControlPlaneState();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders v2 control plane snapshot', async () => {
    render(
      <ScopeProvider apiClient={api}>
        <ControlPlaneDashboard platform="web" />
      </ScopeProvider>
    );

    expect(await screen.findByText('Control Plane')).toBeInTheDocument();
    expect(await screen.findByText('Runtime Core Bus')).toBeInTheDocument();
    expect(screen.getByText('Plugin registry')).toBeInTheDocument();
    expect(await screen.findByText('Tracker')).toBeInTheDocument();
    expect(screen.getByText('Operator backend')).toBeInTheDocument();
  });

  it('persists module toggles into local control plane state', async () => {
    const user = userEvent.setup();

    render(
      <ScopeProvider apiClient={api}>
        <ControlPlaneDashboard platform="web" />
      </ScopeProvider>
    );

    const button = await screen.findByRole('button', { name: 'toggle-module-tracker' });
    await user.click(button);

    await waitFor(async () => {
      const snapshot = await api.getControlPlane();
      const tracker = snapshot.modules.find((module) => module.id === 'tracker');
      expect(tracker?.enabled).toBe(false);
    });

    expect(await screen.findByText('Config updated')).toBeInTheDocument();
  });

  it('surfaces backend diagnostics from /api/health on first screen', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = String(input);
      if (url.endsWith('/health')) {
        return new Response(
          JSON.stringify({
            ok: true,
            checked_at: '2026-04-13T11:22:33Z',
            mode: 'persistent',
            persist_enabled: true,
            persist_path: '/tmp/controlplane.json',
            plugin_dir: 'plugins/manifests',
            plugin_manifests: 3,
            snapshot_updated_at: '2026-04-13T11:21:00Z',
          }),
          {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          }
        );
      }
      throw new Error('offline');
    });
    vi.stubGlobal('fetch', fetchMock);

    render(
      <ScopeProvider apiClient={api}>
        <ControlPlaneDashboard platform="web" />
      </ScopeProvider>
    );

    expect(await screen.findByText('Operator backend')).toBeInTheDocument();
    expect(await screen.findAllByText('backend online')).toHaveLength(2);
    expect(screen.getByText('/tmp/controlplane.json')).toBeInTheDocument();
    expect(screen.getByText('plugins/manifests')).toBeInTheDocument();
  });

  it('meets minimal web ux criteria for operator flow', async () => {
    const user = userEvent.setup();

    render(
      <ScopeProvider apiClient={api}>
        <ControlPlaneDashboard platform="web" />
      </ScopeProvider>
    );

    expect(await screen.findByText('Platform')).toBeInTheDocument();
    expect(screen.getByText('Active Scope')).toBeInTheDocument();
    expect(screen.getByText('Operator backend')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'add-scope' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'rotate-broker-runtime-core' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'toggle-module-tracker' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'rotate-plugin-finance-sync' })).toBeInTheDocument();
    expect(await screen.findByText('Control plane booted')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'rotate-broker-runtime-core' }));

    await waitFor(async () => {
      const snapshot = await api.getControlPlane();
      const runtimeLane = snapshot.brokers.find((broker) => broker.id === 'runtime-core');
      expect(runtimeLane?.mode).toBe('nats');
    });

    expect(await screen.findByText('Config updated')).toBeInTheDocument();
  });
});
