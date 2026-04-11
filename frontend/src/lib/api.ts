import type { Scope } from '@modulr/core-types';

const API_BASE_URL = import.meta.env['VITE_API_BASE_URL'] || 'http://localhost:8080/api';

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
];

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
    try {
      const data = await request<Scope[]>('/scopes');
      if (Array.isArray(data) && data.length > 0) {
        return data;
      }
    } catch {
      // Fallback keeps UI functional in local frontend-only mode.
    }
    return defaultScopes;
  },

  async createScope(scope: Omit<Scope, 'metadata'>): Promise<Scope> {
    try {
      return await request<Scope>('/scopes', {
        method: 'POST',
        body: JSON.stringify(scope),
      });
    } catch {
      return { ...scope, metadata: {} };
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
  },
};
