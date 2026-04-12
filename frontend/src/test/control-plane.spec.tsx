import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it } from 'vitest';
import { ScopeProvider } from '../context/ScopeContext';
import { api } from '../lib/api';
import { ControlPlaneDashboard } from '../modules/control-plane/ControlPlaneDashboard';

describe('ControlPlaneDashboard', () => {
  beforeEach(() => {
    api.resetControlPlaneState();
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

  it('meets minimal web ux criteria for operator flow', async () => {
    const user = userEvent.setup();

    render(
      <ScopeProvider apiClient={api}>
        <ControlPlaneDashboard platform="web" />
      </ScopeProvider>
    );

    expect(await screen.findByText('Platform')).toBeInTheDocument();
    expect(screen.getByText('Active Scope')).toBeInTheDocument();
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
