import type { Scope, Segment } from '@modulr/core-types';
import type {
  BrokerLane,
  BrokerMode,
  BrokerStatus,
  ControlPlaneSnapshot,
  ModuleControl,
  PluginControl,
} from '../types/control-plane';

const API_BASE_URL = import.meta.env['VITE_API_BASE_URL'] || 'http://localhost:8080/api';
const CONTROL_PLANE_STORAGE_KEY = 'modulr-control-plane-v2';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  if (!response.ok) {
    throw new Error(`API ${response.status}: ${response.statusText}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

const defaultScopes: Scope[] = [
  { segment: 'personal', tags: [], metadata: {} },
  { segment: 'family', tags: [], metadata: {} },
  { segment: 'work', tags: [], metadata: {} },
  { segment: 'business', tags: [], metadata: {} },
  { segment: 'business', tags: ['ops'], metadata: { source: 'v2-control-plane' } },
  { segment: 'travel', tags: ['handoff'], metadata: { source: 'v2-control-plane' } },
];

let memorySnapshot = buildDefaultControlPlane();

export function scopeKey(scope: Pick<Scope, 'segment' | 'tags'>): string {
  return `${scope.segment}:${normalizeTags(scope.tags).join(',')}`;
}

function buildDefaultControlPlane(): ControlPlaneSnapshot {
  return {
    updatedAt: new Date().toISOString(),
    scopes: clone(defaultScopes),
    tagPresets: ['ops', 'focus', 'handoff', 'priority', 'audit', 'automation'],
    brokers: [
      {
        id: 'runtime-core',
        title: 'Runtime Core Bus',
        topic: 'runtime.events',
        mode: 'memory',
        status: 'ready',
        notes: 'Single-process baseline for orchestrator + metrics + transport responses.',
        consumerGroups: [
          { id: 'orchestrator', consumers: 2, lag: 0, ackPolicy: 'at_least_once' },
          { id: 'metrics', consumers: 1, lag: 0, ackPolicy: 'at_least_once' },
        ],
      },
      {
        id: 'plugin-fanout',
        title: 'Plugin Fanout Lane',
        topic: 'plugins.dispatch',
        mode: 'nats',
        status: 'planned',
        notes: 'Reserved lane for v2 plugin workloads and backpressure separation.',
        consumerGroups: [
          { id: 'plugin-workers', consumers: 3, lag: 4, ackPolicy: 'at_least_once' },
        ],
      },
    ],
    modules: [
      {
        id: 'tracker',
        title: 'Tracker',
        description: 'Напоминания, планы и milestone flow для scoped productivity.',
        enabled: true,
        dispatchMode: 'queued',
        consumerGroup: 'tracker-workers',
        allowedScopes: ['personal', 'work', 'business'],
        tags: ['reminders', 'milestones'],
        latencyBudgetMs: 250,
      },
      {
        id: 'finance',
        title: 'Finance',
        description: 'Транзакции и журнал расходов с отдельной очередью для business scope.',
        enabled: true,
        dispatchMode: 'fanout',
        consumerGroup: 'finance-ledger',
        allowedScopes: ['personal', 'business', 'assets'],
        tags: ['ledger', 'vat', 'budget'],
        latencyBudgetMs: 180,
      },
      {
        id: 'knowledge',
        title: 'Knowledge',
        description: 'Ноты и query capture с мягким fallback в локальное хранилище.',
        enabled: true,
        dispatchMode: 'inline',
        consumerGroup: 'knowledge-cache',
        allowedScopes: ['personal', 'work', 'travel'],
        tags: ['notes', 'search'],
        latencyBudgetMs: 120,
      },
      {
        id: 'notifications',
        title: 'Notifications',
        description: 'Delivery path для outcome/fallback и cross-platform уведомлений.',
        enabled: false,
        dispatchMode: 'fanout',
        consumerGroup: 'notify-broadcast',
        allowedScopes: ['personal', 'family', 'work', 'business'],
        tags: ['push', 'transport'],
        latencyBudgetMs: 90,
      },
    ],
    plugins: [
      {
        id: 'finance-sync',
        version: '1.0.0',
        runtime: 'process',
        protocol: 'grpc',
        status: 'enabled',
        entry: 'plugins/finance-sync/bin/finance-sync',
        description: 'External ledger adapter for business finance dispatch.',
        capabilities: [
          { module: 'finance', actions: ['create_transaction', 'sync'], scopes: ['business', 'work'] },
        ],
      },
      {
        id: 'tracker-plan',
        version: '1.1.0',
        runtime: 'wasm',
        protocol: 'stdio',
        status: 'staged',
        entry: 'plugins/tracker-plan/tracker-plan.wasm',
        description: 'Planned sandbox plugin for decomposition and milestone shaping.',
        capabilities: [
          { module: 'tracker', actions: ['create_task', 'create_reminder'], scopes: ['personal', 'work'] },
        ],
      },
      {
        id: 'audit-mirror',
        version: '0.9.0',
        runtime: 'process',
        protocol: 'stdio',
        status: 'disabled',
        entry: 'plugins/audit-mirror/bin/audit-mirror',
        description: 'Mirror consumer for regulated audit trails before broker rollout.',
        capabilities: [
          { module: 'knowledge', actions: ['save_note'], scopes: ['business'] },
        ],
      },
    ],
  };
}

function normalizeTags(tags: string[]): string[] {
  return Array.from(
    new Set(
      tags
        .map((tag) => tag.trim().toLowerCase())
        .filter(Boolean)
    )
  );
}

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

function canUseStorage(): boolean {
  if (typeof window === 'undefined' || typeof window.localStorage === 'undefined') {
    return false;
  }
  return (
    typeof window.localStorage.getItem === 'function' &&
    typeof window.localStorage.setItem === 'function'
  );
}

function readLocalSnapshot(): ControlPlaneSnapshot {
  if (!canUseStorage()) {
    return clone(memorySnapshot);
  }

  const raw = window.localStorage.getItem(CONTROL_PLANE_STORAGE_KEY);
  if (!raw) {
    const initial = buildDefaultControlPlane();
    memorySnapshot = clone(initial);
    window.localStorage.setItem(CONTROL_PLANE_STORAGE_KEY, JSON.stringify(initial));
    return initial;
  }

  try {
    const parsed = JSON.parse(raw) as ControlPlaneSnapshot;
    memorySnapshot = clone(parsed);
    return parsed;
  } catch {
    const fallback = buildDefaultControlPlane();
    memorySnapshot = clone(fallback);
    window.localStorage.setItem(CONTROL_PLANE_STORAGE_KEY, JSON.stringify(fallback));
    return fallback;
  }
}

function writeLocalSnapshot(snapshot: ControlPlaneSnapshot): ControlPlaneSnapshot {
  const next = {
    ...snapshot,
    updatedAt: new Date().toISOString(),
  };
  memorySnapshot = clone(next);
  if (canUseStorage()) {
    window.localStorage.setItem(CONTROL_PLANE_STORAGE_KEY, JSON.stringify(next));
  }
  return next;
}

function updateLocalSnapshot(mutator: (snapshot: ControlPlaneSnapshot) => ControlPlaneSnapshot): ControlPlaneSnapshot {
  const next = mutator(clone(readLocalSnapshot()));
  return writeLocalSnapshot(next);
}

function findScopeIndex(scopes: Scope[], id: string): number {
  return scopes.findIndex((scope) => scopeKey(scope) === id);
}

function rotateBrokerMode(mode: BrokerMode): { mode: BrokerMode; status: BrokerStatus } {
  switch (mode) {
    case 'memory':
      return { mode: 'nats', status: 'planned' };
    case 'nats':
      return { mode: 'kafka', status: 'degraded' };
    case 'kafka':
    default:
      return { mode: 'memory', status: 'ready' };
  }
}

const defaultScopesResponse = clone(defaultScopes);

export const api = {
  async healthCheck(): Promise<{ ok: boolean }> {
    try {
      await request<unknown>('/health');
      return { ok: true };
    } catch {
      return { ok: false };
    }
  },

  async getScopes(): Promise<Scope[]> {
    const localScopes = readLocalSnapshot().scopes;
    try {
      const data = await request<Scope[]>('/scopes');
      if (Array.isArray(data) && data.length > 0) {
        writeLocalSnapshot({
          ...readLocalSnapshot(),
          scopes: data.map((scope) => ({
            ...scope,
            tags: normalizeTags(scope.tags),
          })),
        });
        return data;
      }
    } catch {
      // Fallback keeps UI functional in local frontend-only mode.
    }
    return localScopes.length > 0 ? localScopes : defaultScopesResponse;
  },

  async createScope(scope: Omit<Scope, 'metadata'>): Promise<Scope> {
    const normalizedScope: Scope = {
      segment: scope.segment,
      tags: normalizeTags(scope.tags),
      metadata: { source: 'v2-control-plane' },
    };
    try {
      const created = await request<Scope>('/scopes', {
        method: 'POST',
        body: JSON.stringify(normalizedScope),
      });
      updateLocalSnapshot((snapshot) => {
        const id = scopeKey(created);
        const index = findScopeIndex(snapshot.scopes, id);
        if (index >= 0) {
          snapshot.scopes[index] = created;
        } else {
          snapshot.scopes.push(created);
        }
        return snapshot;
      });
      return created;
    } catch {
      updateLocalSnapshot((snapshot) => {
        const id = scopeKey(normalizedScope);
        if (findScopeIndex(snapshot.scopes, id) === -1) {
          snapshot.scopes.push(normalizedScope);
        }
        return snapshot;
      });
      return normalizedScope;
    }
  },

  async deleteScope(scopeId: string): Promise<void> {
    try {
      await request<void>(`/scopes/${encodeURIComponent(scopeId)}`, {
        method: 'DELETE',
      });
    } catch {
      // Ignore in local fallback mode.
    }
    updateLocalSnapshot((snapshot) => ({
      ...snapshot,
      scopes: snapshot.scopes.filter((scope) => scopeKey(scope) !== scopeId),
    }));
  },

  async updateScopeTags(scopeId: string, tags: string[]): Promise<Scope> {
    const nextTags = normalizeTags(tags);
    try {
      const updated = await request<Scope>(`/scopes/${encodeURIComponent(scopeId)}`, {
        method: 'PATCH',
        body: JSON.stringify({ tags: nextTags }),
      });
      updateLocalSnapshot((snapshot) => {
        const index = findScopeIndex(snapshot.scopes, scopeId);
        if (index >= 0) {
          snapshot.scopes[index] = updated;
        } else {
          snapshot.scopes.push(updated);
        }
        return snapshot;
      });
      return updated;
    } catch {
      let updatedScope: Scope | undefined;
      updateLocalSnapshot((snapshot) => {
        const index = findScopeIndex(snapshot.scopes, scopeId);
        if (index >= 0) {
          const currentScope = snapshot.scopes[index];
          if (!currentScope) {
            return snapshot;
          }
          updatedScope = {
            segment: currentScope.segment,
            tags: nextTags,
            metadata: currentScope.metadata ?? {},
          };
          snapshot.scopes[index] = updatedScope;
          return snapshot;
        }

        const [segment] = scopeId.split(':');
        updatedScope = {
          segment: segment as Segment,
          tags: nextTags,
          metadata: { source: 'v2-control-plane' },
        };
        snapshot.scopes.push(updatedScope);
        return snapshot;
      });
      if (!updatedScope) {
        throw new Error(`Unknown scope ${scopeId}`);
      }
      return updatedScope;
    }
  },

  async getControlPlane(): Promise<ControlPlaneSnapshot> {
    try {
      const remote = await request<ControlPlaneSnapshot>('/control-plane');
      return writeLocalSnapshot(remote);
    } catch {
      return readLocalSnapshot();
    }
  },

  async updateModule(moduleId: string, patch: Partial<ModuleControl>): Promise<ModuleControl> {
    try {
      const remote = await request<ModuleControl>(`/control-plane/modules/${encodeURIComponent(moduleId)}`, {
        method: 'PATCH',
        body: JSON.stringify(patch),
      });
      updateLocalSnapshot((snapshot) => ({
        ...snapshot,
        modules: snapshot.modules.map((module) => (module.id === moduleId ? remote : module)),
      }));
      return remote;
    } catch {
      let updated: ModuleControl | null = null;
      updateLocalSnapshot((snapshot) => {
        snapshot.modules = snapshot.modules.map((module) => {
          if (module.id !== moduleId) {
            return module;
          }
          updated = { ...module, ...patch };
          return updated;
        });
        return snapshot;
      });
      if (!updated) {
        throw new Error(`Unknown module ${moduleId}`);
      }
      return updated;
    }
  },

  async updatePlugin(pluginId: string, patch: Partial<PluginControl>): Promise<PluginControl> {
    try {
      const remote = await request<PluginControl>(`/control-plane/plugins/${encodeURIComponent(pluginId)}`, {
        method: 'PATCH',
        body: JSON.stringify(patch),
      });
      updateLocalSnapshot((snapshot) => ({
        ...snapshot,
        plugins: snapshot.plugins.map((plugin) => (plugin.id === pluginId ? remote : plugin)),
      }));
      return remote;
    } catch {
      let updated: PluginControl | null = null;
      updateLocalSnapshot((snapshot) => {
        snapshot.plugins = snapshot.plugins.map((plugin) => {
          if (plugin.id !== pluginId) {
            return plugin;
          }
          updated = { ...plugin, ...patch };
          return updated;
        });
        return snapshot;
      });
      if (!updated) {
        throw new Error(`Unknown plugin ${pluginId}`);
      }
      return updated;
    }
  },

  async cycleBrokerMode(brokerId: string): Promise<BrokerLane> {
    try {
      const remote = await request<BrokerLane>(`/control-plane/brokers/${encodeURIComponent(brokerId)}/cycle`, {
        method: 'POST',
      });
      updateLocalSnapshot((snapshot) => ({
        ...snapshot,
        brokers: snapshot.brokers.map((broker) => (broker.id === brokerId ? remote : broker)),
      }));
      return remote;
    } catch {
      let updated: BrokerLane | null = null;
      updateLocalSnapshot((snapshot) => {
        snapshot.brokers = snapshot.brokers.map((broker) => {
          if (broker.id !== brokerId) {
            return broker;
          }
          const next = rotateBrokerMode(broker.mode);
          updated = { ...broker, ...next };
          return updated;
        });
        return snapshot;
      });
      if (!updated) {
        throw new Error(`Unknown broker ${brokerId}`);
      }
      return updated;
    }
  },

  resetControlPlaneState(): void {
    const initial = buildDefaultControlPlane();
    memorySnapshot = clone(initial);
    if (canUseStorage()) {
      window.localStorage.setItem(CONTROL_PLANE_STORAGE_KEY, JSON.stringify(initial));
    }
  },
};
