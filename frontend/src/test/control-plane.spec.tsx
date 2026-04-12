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
  });
});
