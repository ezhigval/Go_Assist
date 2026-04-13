import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ScopeProvider } from '../context/ScopeContext';
import { api } from '../lib/api';
import { ControlPlaneDashboard } from '../modules/control-plane/ControlPlaneDashboard';

describe('ControlPlaneDashboard', () => {
  beforeEach(() => {
    api.resetControlPlaneState();
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new Error('offline');
      })
    );
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

    await waitFor(() => {
      const snapshot = api.getControlPlaneSnapshot();
      const tracker = snapshot.modules.find((module) => module.id === 'tracker');
      expect(tracker?.enabled).toBe(false);
    });

    expect(await screen.findByText('Config updated')).toBeInTheDocument();
  });

  it('persists module settings edits into local control plane state', async () => {
    const user = userEvent.setup();

    render(
      <ScopeProvider apiClient={api}>
        <ControlPlaneDashboard platform="web" />
      </ScopeProvider>
    );

    await user.selectOptions(await screen.findByLabelText('module-dispatch-tracker'), 'fanout');
    await user.clear(screen.getByLabelText('module-group-tracker'));
    await user.type(screen.getByLabelText('module-group-tracker'), 'tracker-priority');
    await user.clear(screen.getByLabelText('module-latency-tracker'));
    await user.type(screen.getByLabelText('module-latency-tracker'), '320');
    await user.click(screen.getByRole('button', { name: 'toggle-module-scope-tracker-travel' }));
    await user.click(screen.getByRole('button', { name: 'save-module-tracker' }));

    await waitFor(() => {
      const snapshot = api.getControlPlaneSnapshot();
      const tracker = snapshot.modules.find((module) => module.id === 'tracker');
      expect(tracker?.dispatchMode).toBe('fanout');
      expect(tracker?.consumerGroup).toBe('tracker-priority');
      expect(tracker?.latencyBudgetMs).toBe(320);
      expect(tracker?.allowedScopes).toContain('travel');
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
    expect(screen.getByRole('button', { name: 'reload-plugin-manifests' })).toBeEnabled();
  });

  it('reloads plugin manifests from backend without restarting dashboard', async () => {
    const user = userEvent.setup();
    const baseSnapshot = api.getControlPlaneSnapshot();
    const reloadedSnapshot = {
      ...baseSnapshot,
      updatedAt: '2026-04-13T11:25:00Z',
      plugins: baseSnapshot.plugins.map((plugin) =>
        plugin.id === 'finance-sync'
          ? { ...plugin, version: '2.0.0', entry: 'bin/finance-sync-v2' }
          : plugin
      ),
    };

    let reloads = 0;
    const fetchMock = vi.fn(async (input: string | URL | Request, init?: RequestInit) => {
      const url = String(input);
      const method = init?.method ?? 'GET';
      if (url.endsWith('/control-plane/plugins/reload') && method === 'POST') {
        reloads += 1;
        return new Response(JSON.stringify(reloadedSnapshot), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }
      if (url.endsWith('/control-plane')) {
        return new Response(JSON.stringify(baseSnapshot), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }
      if (url.endsWith('/health')) {
        return new Response(
          JSON.stringify({
            ok: true,
            checked_at: reloads > 0 ? '2026-04-13T11:26:00Z' : '2026-04-13T11:22:33Z',
            mode: 'persistent',
            persist_enabled: true,
            persist_path: '/tmp/controlplane.json',
            plugin_dir: 'plugins/manifests',
            plugin_manifests: reloads > 0 ? 4 : 3,
            snapshot_updated_at: reloads > 0 ? '2026-04-13T11:25:00Z' : baseSnapshot.updatedAt,
          }),
          {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          }
        );
      }
      throw new Error(`unexpected request: ${method} ${url}`);
    });
    vi.stubGlobal('fetch', fetchMock);

    render(
      <ScopeProvider apiClient={api}>
        <ControlPlaneDashboard platform="web" />
      </ScopeProvider>
    );

    const reloadButton = await screen.findByRole('button', { name: 'reload-plugin-manifests' });
    await waitFor(() => expect(reloadButton).toBeEnabled());
    await user.click(reloadButton);

    await waitFor(() => {
      expect(screen.getByText('v2.0.0')).toBeInTheDocument();
      expect(screen.getByText('bin/finance-sync-v2')).toBeInTheDocument();
    });

    expect(await screen.findByText('Config updated')).toBeInTheDocument();
  });

  it('persists plugin settings edits into local control plane state', async () => {
    render(
      <ScopeProvider apiClient={api}>
        <ControlPlaneDashboard platform="web" />
      </ScopeProvider>
    );

    fireEvent.change(await screen.findByLabelText('plugin-description-finance-sync'), {
      target: { value: 'Ledger sync for staged operator rollout.' },
    });
    fireEvent.change(screen.getByLabelText('plugin-capability-actions-finance-sync-0'), {
      target: { value: 'create_transaction, reconcile' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'toggle-plugin-scope-finance-sync-0-assets' }));
    fireEvent.click(screen.getByRole('button', { name: 'save-plugin-finance-sync' }));

    await waitFor(() => {
      const snapshot = api.getControlPlaneSnapshot();
      const financeSync = snapshot.plugins.find((plugin) => plugin.id === 'finance-sync');
      expect(financeSync?.description).toBe('Ledger sync for staged operator rollout.');
      expect(financeSync?.capabilities[0]?.actions).toEqual(['create_transaction', 'reconcile']);
      expect(financeSync?.capabilities[0]?.scopes).toContain('assets');
    });

    expect(await screen.findByText('Config updated')).toBeInTheDocument();
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
    expect(screen.getByRole('button', { name: 'save-module-tracker' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'rotate-plugin-finance-sync' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'save-plugin-finance-sync' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'reload-plugin-manifests' })).toBeInTheDocument();
    expect(await screen.findByText('Control plane booted')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'rotate-broker-runtime-core' }));

    await waitFor(() => {
      const snapshot = api.getControlPlaneSnapshot();
      const runtimeLane = snapshot.brokers.find((broker) => broker.id === 'runtime-core');
      expect(runtimeLane?.mode).toBe('nats');
    });

    expect(await screen.findByText('Config updated')).toBeInTheDocument();
  });
});
