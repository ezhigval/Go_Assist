import type { Scope } from './core';

export interface ModuleCard {
  id: string;
  module: string;
  title: string;
  description: string;
  scope: Scope;
}
