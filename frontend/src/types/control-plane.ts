import type { Scope, Segment } from './core';

export type BrokerMode = 'memory' | 'nats' | 'kafka';
export type BrokerStatus = 'ready' | 'planned' | 'degraded';
export type AckPolicy = 'at_least_once' | 'exactly_once';

export interface BrokerConsumerGroup {
  id: string;
  consumers: number;
  lag: number;
  ackPolicy: AckPolicy;
}

export interface BrokerLane {
  id: string;
  title: string;
  topic: string;
  mode: BrokerMode;
  status: BrokerStatus;
  notes: string;
  consumerGroups: BrokerConsumerGroup[];
}

export type ModuleDispatchMode = 'inline' | 'queued' | 'fanout';

export interface ModuleControl {
  id: string;
  title: string;
  description: string;
  enabled: boolean;
  dispatchMode: ModuleDispatchMode;
  consumerGroup: string;
  allowedScopes: Segment[];
  tags: string[];
  latencyBudgetMs: number;
}

export type PluginRuntime = 'process' | 'wasm';
export type PluginProtocol = 'grpc' | 'stdio';
export type PluginStatus = 'enabled' | 'staged' | 'disabled';
export type ControlPlaneHealthMode = 'memory' | 'persistent' | 'fallback';

export interface PluginCapability {
  module: string;
  actions: string[];
  scopes: Segment[];
}

export interface PluginControl {
  id: string;
  version: string;
  runtime: PluginRuntime;
  protocol: PluginProtocol;
  status: PluginStatus;
  entry: string;
  description: string;
  capabilities: PluginCapability[];
}

export interface ControlPlaneHealth {
  ok: boolean;
  checkedAt: string;
  mode: ControlPlaneHealthMode;
  persistEnabled: boolean;
  persistPath?: string;
  pluginDir?: string;
  pluginManifests: number;
  snapshotUpdatedAt: string;
}

export interface ControlPlaneSnapshot {
  updatedAt: string;
  scopes: Scope[];
  tagPresets: string[];
  brokers: BrokerLane[];
  modules: ModuleControl[];
  plugins: PluginControl[];
}
