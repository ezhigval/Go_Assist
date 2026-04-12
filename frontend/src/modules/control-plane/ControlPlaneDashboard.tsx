import { useEffect, useState } from 'react';
import { Button } from '../../components/ui/Button';
import { useScope } from '../../context/ScopeContext';
import { api, scopeKey } from '../../lib/api';
import { eventBus } from '../../lib/eventBus';
import { cn } from '../../lib/utils';
import { AllSegments, type EventName, type Segment } from '../../types/core';
import type {
  BrokerLane,
  ControlPlaneSnapshot,
  ModuleControl,
  PluginControl,
  PluginStatus,
} from '../../types/control-plane';

type RuntimePlatform = 'telegram' | 'web' | 'mobile' | 'desktop';
type HealthState = 'checking' | 'online' | 'offline';

interface DashboardEvent {
  id: string;
  eventName: EventName;
  title: string;
  detail: string;
  timestamp: number;
}

const pluginStatusCycle: PluginStatus[] = ['disabled', 'staged', 'enabled'];

const statusTone: Record<PluginStatus | BrokerLane['status'] | HealthState, string> = {
  enabled: 'bg-emerald-100 text-emerald-800',
  staged: 'bg-amber-100 text-amber-800',
  disabled: 'bg-stone-200 text-stone-700',
  ready: 'bg-emerald-100 text-emerald-800',
  planned: 'bg-sky-100 text-sky-800',
  degraded: 'bg-rose-100 text-rose-800',
  checking: 'bg-stone-200 text-stone-700',
  online: 'bg-emerald-100 text-emerald-800',
  offline: 'bg-rose-100 text-rose-800',
};

function parseTags(raw: string): string[] {
  return Array.from(
    new Set(
      raw
        .split(',')
        .map((tag) => tag.trim().toLowerCase())
        .filter(Boolean)
    )
  );
}

function formatEventTime(timestamp: number): string {
  return new Date(timestamp).toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
  });
}

function nextPluginStatus(current: PluginStatus): PluginStatus {
  const index = pluginStatusCycle.indexOf(current);
  if (index === -1) {
    return pluginStatusCycle[0]!;
  }
  return pluginStatusCycle[(index + 1) % pluginStatusCycle.length] ?? pluginStatusCycle[0]!;
}

interface ControlPlaneDashboardProps {
  platform: RuntimePlatform;
}

export function ControlPlaneDashboard({ platform }: ControlPlaneDashboardProps) {
  const {
    activeScope,
    availableScopes,
    createScope,
    deleteScope,
    setActiveScope,
    loadScopes,
    error,
    clearError,
    isLoading,
  } = useScope();
  const [snapshot, setSnapshot] = useState<ControlPlaneSnapshot>(() => api.getControlPlaneSnapshot());
  const [health, setHealth] = useState<HealthState>('checking');
  const [timeline, setTimeline] = useState<DashboardEvent[]>([]);
  const [newScopeSegment, setNewScopeSegment] = useState<Segment>('work');
  const [newScopeTags, setNewScopeTags] = useState('ops, priority');

  const refreshSnapshot = async () => {
    const nextSnapshot = await api.getControlPlane();
    setSnapshot(nextSnapshot);
  };

  const appendEvent = (eventName: EventName, title: string, detail: string) => {
    setTimeline((current) => [
      {
        id: `${eventName}-${Date.now()}-${Math.random().toString(16).slice(2, 7)}`,
        eventName,
        title,
        detail,
        timestamp: Date.now(),
      },
      ...current,
    ].slice(0, 8));
  };

  useEffect(() => {
    let alive = true;

    const bootstrap = async () => {
      const snapshotPromise = api.getControlPlane();
      const healthPromise = api.healthCheck().catch(() => ({ ok: false }));

      const nextSnapshot = await snapshotPromise;
      if (!alive) {
        return;
      }
      setSnapshot(nextSnapshot);

      const nextHealth = await healthPromise;
      if (!alive) {
        return;
      }
      setHealth(nextHealth.ok ? 'online' : 'offline');
    };

    const unsubscribeStartup = eventBus.on('v1.system.startup', (event) => {
      appendEvent('v1.system.startup', 'Control plane booted', JSON.stringify(event.payload));
    });
    const unsubscribePreferences = eventBus.on('v1.user.preferences.updated', (event) => {
      appendEvent('v1.user.preferences.updated', 'Config updated', JSON.stringify(event.payload));
    });

    void bootstrap();
    eventBus.emit('v1.system.startup', {
      platform,
      surface: 'control-plane',
    });

    return () => {
      alive = false;
      unsubscribeStartup();
      unsubscribePreferences();
    };
  }, [platform]);

  const handleCreateScope = async () => {
    const tags = parseTags(newScopeTags);
    await createScope({
      segment: newScopeSegment,
      tags,
    });
    await refreshSnapshot();
    eventBus.emit('v1.user.preferences.updated', {
      entity: 'scope',
      action: 'created',
      scope: `${newScopeSegment}:${tags.join(',')}`,
    });
  };

  const handleDeleteScope = async (id: string) => {
    await deleteScope(id);
    await refreshSnapshot();
    eventBus.emit('v1.user.preferences.updated', {
      entity: 'scope',
      action: 'deleted',
      scope: id,
    });
  };

  const handleToggleTag = async (tag: string) => {
    const currentKey = scopeKey(activeScope);
    const nextTags = activeScope.tags.includes(tag)
      ? activeScope.tags.filter((value) => value !== tag)
      : [...activeScope.tags, tag];

    const updatedScope = await api.updateScopeTags(currentKey, nextTags);
    setActiveScope(updatedScope);
    await loadScopes();
    await refreshSnapshot();
    eventBus.emit('v1.user.preferences.updated', {
      entity: 'scope_tags',
      scope: activeScope.segment,
      tags: nextTags,
    });
  };

  const handleToggleModule = async (module: ModuleControl) => {
    const nextModule = await api.updateModule(module.id, { enabled: !module.enabled });
    setSnapshot((current) =>
      current
        ? {
            ...current,
            modules: current.modules.map((item) => (item.id === nextModule.id ? nextModule : item)),
          }
        : current
    );
    eventBus.emit('v1.user.preferences.updated', {
      entity: 'module',
      module: module.id,
      enabled: nextModule.enabled,
    });
  };

  const handleRotateBroker = async (broker: BrokerLane) => {
    const nextBroker = await api.cycleBrokerMode(broker.id);
    setSnapshot((current) =>
      current
        ? {
            ...current,
            brokers: current.brokers.map((item) => (item.id === nextBroker.id ? nextBroker : item)),
          }
        : current
    );
    eventBus.emit('v1.user.preferences.updated', {
      entity: 'broker',
      broker: broker.id,
      mode: nextBroker.mode,
      status: nextBroker.status,
    });
  };

  const handleRotatePlugin = async (plugin: PluginControl) => {
    const nextPlugin = await api.updatePlugin(plugin.id, { status: nextPluginStatus(plugin.status) });
    setSnapshot((current) =>
      current
        ? {
            ...current,
            plugins: current.plugins.map((item) => (item.id === nextPlugin.id ? nextPlugin : item)),
          }
        : current
    );
    eventBus.emit('v1.user.preferences.updated', {
      entity: 'plugin',
      plugin: plugin.id,
      status: nextPlugin.status,
    });
  };

  const enabledModules = snapshot?.modules.filter((item) => item.enabled).length ?? 0;
  const enabledPlugins = snapshot?.plugins.filter((item) => item.status === 'enabled').length ?? 0;

  return (
    <div className="space-y-8">
      <section className="glass-card overflow-hidden">
        <div className="grid gap-6 lg:grid-cols-[1.5fr,0.9fr]">
          <div className="space-y-5">
            <div className="inline-flex items-center gap-2 rounded-full border border-white/70 bg-white/70 px-4 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-stone-500">
              Modulr v2.0
              <span className={cn('rounded-full px-2 py-0.5 text-[10px] font-bold tracking-[0.18em]', statusTone[health])}>
                {health}
              </span>
            </div>
            <div className="space-y-3">
              <h2 className="text-4xl font-semibold tracking-tight text-stone-950 sm:text-5xl">
                Control Plane
              </h2>
              <p className="max-w-2xl text-base leading-7 text-stone-600">
                Dashboard для перехода из single-process runtime в управляемый v2.0 режим:
                consumer groups, plugin registry и scope/tag configuration без правки кода.
              </p>
            </div>
            <div className="flex flex-wrap gap-3">
              <div className="rounded-2xl border border-white/70 bg-white/80 px-4 py-3">
                <div className="text-xs uppercase tracking-[0.2em] text-stone-500">Platform</div>
                <div className="mt-1 text-lg font-semibold text-stone-950">{platform}</div>
              </div>
              <div className="rounded-2xl border border-white/70 bg-white/80 px-4 py-3">
                <div className="text-xs uppercase tracking-[0.2em] text-stone-500">Active Scope</div>
                <div className="mt-1 text-lg font-semibold text-stone-950">
                  {activeScope.segment}
                  {activeScope.tags.length > 0 && ` · ${activeScope.tags.join(', ')}`}
                </div>
              </div>
            </div>
          </div>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
            <MetricCard label="Scope presets" value={availableScopes.length} hint="Persisted presets from control plane storage" />
            <MetricCard label="Enabled modules" value={enabledModules} hint="Runtime actions currently admitted by UI control plane" />
            <MetricCard label="Enabled plugins" value={enabledPlugins} hint="Versioned manifests staged or live for fanout" />
            <MetricCard label="Broker lanes" value={snapshot?.brokers.length ?? 0} hint="Each lane defines topic, mode and consumer-group shape" />
          </div>
        </div>
      </section>

      {error && (
        <section className="glass-card flex items-center justify-between gap-4 border-rose-200 bg-rose-50/90">
          <p className="text-sm text-rose-800">{error}</p>
          <Button variant="secondary" size="sm" onClick={clearError}>
            Dismiss
          </Button>
        </section>
      )}

      <section className="grid gap-6 xl:grid-cols-[1.2fr,0.8fr]">
        <div className="glass-card space-y-5">
          <SectionHeading eyebrow="Scope presets" title="Context switching in a couple clicks" />
          <div className="grid gap-3 md:grid-cols-2">
            {availableScopes.map((scope) => {
              const id = scopeKey(scope);
              const isActive = id === scopeKey(activeScope);
              return (
                <div key={id} className={cn('rounded-3xl border px-4 py-4 transition', isActive ? 'border-amber-400 bg-amber-50/90' : 'border-stone-200 bg-white/70')}>
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <div className="text-sm uppercase tracking-[0.2em] text-stone-500">{scope.segment}</div>
                      <div className="mt-2 text-sm text-stone-700">
                        {scope.tags.length > 0 ? scope.tags.join(', ') : 'No tags yet'}
                      </div>
                    </div>
                    <div className="flex gap-2">
                      {isActive ? (
                        <span className="rounded-full bg-stone-900 px-3 py-1 text-xs font-semibold text-white">Active</span>
                      ) : (
                        <Button
                          variant="secondary"
                          size="sm"
                          aria-label={`use-scope-${id}`}
                          onClick={() => setActiveScope(scope)}
                        >
                          Use
                        </Button>
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={`remove-scope-${id}`}
                        onClick={() => handleDeleteScope(id)}
                        disabled={availableScopes.length <= 1}
                      >
                        Remove
                      </Button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          <div className="rounded-3xl border border-dashed border-stone-300 bg-white/60 p-4">
            <div className="grid gap-3 md:grid-cols-[0.7fr,1fr,auto]">
              <label className="space-y-2 text-sm text-stone-600">
                <span className="font-medium text-stone-900">Segment</span>
                <select
                  value={newScopeSegment}
                  onChange={(event) => setNewScopeSegment(event.target.value as Segment)}
                  aria-label="new-scope-segment"
                  className="w-full rounded-2xl border border-stone-200 bg-white px-3 py-2 text-stone-900 outline-none ring-0"
                >
                  {AllSegments.map((segment) => (
                    <option key={segment} value={segment}>
                      {segment}
                    </option>
                  ))}
                </select>
              </label>
              <label className="space-y-2 text-sm text-stone-600">
                <span className="font-medium text-stone-900">Tags</span>
                <input
                  value={newScopeTags}
                  onChange={(event) => setNewScopeTags(event.target.value)}
                  aria-label="new-scope-tags"
                  placeholder="ops, priority"
                  className="w-full rounded-2xl border border-stone-200 bg-white px-3 py-2 text-stone-900 outline-none ring-0"
                />
              </label>
              <div className="flex items-end">
                <Button aria-label="add-scope" onClick={handleCreateScope} loading={isLoading}>
                  Add scope
                </Button>
              </div>
            </div>
          </div>

          <div className="space-y-3">
            <div className="text-sm font-medium text-stone-900">Quick tags for current scope</div>
            <div className="flex flex-wrap gap-2">
              {(snapshot?.tagPresets ?? []).map((tag) => {
                const active = activeScope.tags.includes(tag);
                return (
                  <button
                    key={tag}
                    type="button"
                    onClick={() => handleToggleTag(tag)}
                    aria-label={`toggle-tag-${tag}`}
                    aria-pressed={active}
                    className={cn(
                      'rounded-full px-4 py-2 text-sm font-medium transition',
                      active ? 'bg-stone-900 text-white' : 'bg-white/70 text-stone-700 hover:bg-white'
                    )}
                  >
                    {active ? '−' : '+'} {tag}
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        <div className="glass-card space-y-5">
          <SectionHeading eyebrow="Broker lanes" title="Distributed bus rollout" />
          <div className="space-y-3">
            {(snapshot?.brokers ?? []).map((broker) => (
              <div key={broker.id} className="rounded-3xl border border-stone-200 bg-white/75 p-4">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-sm uppercase tracking-[0.2em] text-stone-500">{broker.topic}</div>
                    <div className="mt-1 text-lg font-semibold text-stone-950">{broker.title}</div>
                  </div>
                  <span className={cn('rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em]', statusTone[broker.status])}>
                    {broker.status}
                  </span>
                </div>
                <p className="mt-3 text-sm leading-6 text-stone-600">{broker.notes}</p>
                <div className="mt-4 flex flex-wrap gap-2">
                  <span className="rounded-full bg-stone-100 px-3 py-1 text-xs font-medium text-stone-700">
                    mode: {broker.mode}
                  </span>
                  {broker.consumerGroups.map((group) => (
                    <span key={group.id} className="rounded-full bg-white px-3 py-1 text-xs font-medium text-stone-700">
                      {group.id} · {group.consumers} workers · lag {group.lag}
                    </span>
                  ))}
                </div>
                <div className="mt-4">
                  <Button
                    variant="secondary"
                    size="sm"
                    aria-label={`rotate-broker-${broker.id}`}
                    onClick={() => handleRotateBroker(broker)}
                  >
                    Rotate mode
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.1fr,0.9fr]">
        <div className="glass-card space-y-5">
          <SectionHeading eyebrow="Runtime modules" title="Dispatch admission and queue shape" />
          <div className="grid gap-3">
            {(snapshot?.modules ?? []).map((module) => (
              <div key={module.id} className="rounded-3xl border border-stone-200 bg-white/75 p-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="space-y-2">
                    <div className="text-sm uppercase tracking-[0.2em] text-stone-500">{module.dispatchMode}</div>
                    <div className="text-xl font-semibold text-stone-950">{module.title}</div>
                    <p className="max-w-2xl text-sm leading-6 text-stone-600">{module.description}</p>
                  </div>
                  <Button
                    variant={module.enabled ? 'secondary' : 'primary'}
                    size="sm"
                    aria-label={`toggle-module-${module.id}`}
                    onClick={() => handleToggleModule(module)}
                  >
                    {module.enabled ? 'Pause' : 'Enable'}
                  </Button>
                </div>
                <div className="mt-4 flex flex-wrap gap-2">
                  <span className="rounded-full bg-stone-100 px-3 py-1 text-xs font-medium text-stone-700">
                    group: {module.consumerGroup}
                  </span>
                  <span className="rounded-full bg-stone-100 px-3 py-1 text-xs font-medium text-stone-700">
                    latency: {module.latencyBudgetMs}ms
                  </span>
                  {module.allowedScopes.map((scope) => (
                    <span key={scope} className="rounded-full bg-white px-3 py-1 text-xs font-medium text-stone-700">
                      {scope}
                    </span>
                  ))}
                  {module.tags.map((tag) => (
                    <span key={tag} className="rounded-full bg-amber-50 px-3 py-1 text-xs font-medium text-amber-800">
                      #{tag}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="glass-card space-y-5">
          <SectionHeading eyebrow="Plugin registry" title="Versioned external execution" />
          <div className="space-y-3">
            {(snapshot?.plugins ?? []).map((plugin) => (
              <div key={plugin.id} className="rounded-3xl border border-stone-200 bg-white/75 p-4">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <div className="text-sm uppercase tracking-[0.2em] text-stone-500">{plugin.runtime} · {plugin.protocol}</div>
                    <div className="mt-1 text-lg font-semibold text-stone-950">
                      {plugin.id}
                      <span className="ml-2 text-sm font-medium text-stone-500">v{plugin.version}</span>
                    </div>
                  </div>
                  <Button
                    variant="secondary"
                    size="sm"
                    aria-label={`rotate-plugin-${plugin.id}`}
                    onClick={() => handleRotatePlugin(plugin)}
                  >
                    {plugin.status}
                  </Button>
                </div>
                <p className="mt-3 text-sm leading-6 text-stone-600">{plugin.description}</p>
                <div className="mt-3 rounded-2xl bg-stone-100/80 px-3 py-2 text-xs text-stone-600">
                  {plugin.entry}
                </div>
                <div className="mt-4 space-y-2">
                  {plugin.capabilities.map((capability) => (
                    <div key={`${plugin.id}-${capability.module}`} className="rounded-2xl bg-white px-3 py-2 text-sm text-stone-700">
                      <span className="font-semibold text-stone-900">{capability.module}</span>
                      {' · '}
                      {capability.actions.join(', ')}
                      {' · '}
                      {capability.scopes.join(', ')}
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="glass-card space-y-5" aria-live="polite">
        <SectionHeading eyebrow="Event trace" title="Live control-plane audit" />
        <div className="grid gap-3 md:grid-cols-2">
          {timeline.map((item) => (
            <div key={item.id} className="rounded-3xl border border-stone-200 bg-white/75 p-4">
              <div className="flex items-center justify-between gap-3">
                <div className="text-sm font-semibold text-stone-950">{item.title}</div>
                <div className="text-xs uppercase tracking-[0.18em] text-stone-500">{formatEventTime(item.timestamp)}</div>
              </div>
              <div className="mt-2 text-xs uppercase tracking-[0.18em] text-stone-500">{item.eventName}</div>
              <div className="mt-3 text-sm leading-6 text-stone-600">{item.detail}</div>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

function MetricCard({ label, value, hint }: { label: string; value: number; hint: string }) {
  return (
    <div className="rounded-[26px] border border-white/70 bg-white/85 p-4 shadow-[0_18px_50px_rgba(53,31,15,0.08)]">
      <div className="text-xs uppercase tracking-[0.22em] text-stone-500">{label}</div>
      <div className="mt-2 text-3xl font-semibold text-stone-950">{value}</div>
      <p className="mt-2 text-sm leading-6 text-stone-600">{hint}</p>
    </div>
  );
}

function SectionHeading({ eyebrow, title }: { eyebrow: string; title: string }) {
  return (
    <div className="space-y-2">
      <div className="text-xs font-semibold uppercase tracking-[0.24em] text-stone-500">{eyebrow}</div>
      <h3 className="text-2xl font-semibold tracking-tight text-stone-950">{title}</h3>
    </div>
  );
}
