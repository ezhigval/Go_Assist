/**
 * Scope Context - manages active scope and tags across the application
 * Provides context-aware functionality for all components
 */

import { createContext, useContext, useReducer, useEffect, ReactNode } from 'react';
import type { Scope, Segment, User } from '@modulr/core-types';

// ============================================================================
// SCOPE STATE TYPES
// ============================================================================

export interface ScopeState {
  activeScope: Scope;
  availableScopes: Scope[];
  recentScopes: Scope[];
  user: User | null;
  isLoading: boolean;
  error: string | null;
}

export interface ScopeContextValue extends ScopeState {
  // Actions
  setActiveScope: (scope: Scope) => void;
  addTag: (tag: string) => void;
  removeTag: (tag: string) => void;
  updateScope: (updates: Partial<Scope>) => void;
  createScope: (scope: Omit<Scope, 'metadata'>) => Promise<void>;
  deleteScope: (scopeId: string) => Promise<void>;
  loadScopes: () => Promise<void>;
  clearError: () => void;
  
  // Computed values
  currentSegment: Segment;
  scopeColor: string;
  hasMultipleScopes: boolean;
}

// ============================================================================
// ACTION TYPES
// ============================================================================

type ScopeAction =
  | { type: 'SET_LOADING'; payload: boolean }
  | { type: 'SET_ERROR'; payload: string | null }
  | { type: 'SET_USER'; payload: User }
  | { type: 'SET_ACTIVE_SCOPE'; payload: Scope }
  | { type: 'SET_AVAILABLE_SCOPES'; payload: Scope[] }
  | { type: 'ADD_RECENT_SCOPE'; payload: Scope }
  | { type: 'UPDATE_SCOPE'; payload: Partial<Scope> }
  | { type: 'ADD_TAG'; payload: string }
  | { type: 'REMOVE_TAG'; payload: string }
  | { type: 'CLEAR_ERROR' };

// ============================================================================
// INITIAL STATE
// ============================================================================

const defaultScope: Scope = {
  segment: 'personal',
  tags: [],
  metadata: {},
};

const initialState: ScopeState = {
  activeScope: defaultScope,
  availableScopes: [defaultScope],
  recentScopes: [],
  user: null,
  isLoading: false,
  error: null,
};

// ============================================================================
// REDUCER
// ============================================================================

function scopeReducer(state: ScopeState, action: ScopeAction): ScopeState {
  switch (action.type) {
    case 'SET_LOADING':
      return {
        ...state,
        isLoading: action.payload,
        error: action.payload ? null : state.error,
      };

    case 'SET_ERROR':
      return {
        ...state,
        error: action.payload,
        isLoading: false,
      };

    case 'SET_USER':
      return {
        ...state,
        user: action.payload,
      };

    case 'SET_ACTIVE_SCOPE':
      return {
        ...state,
        activeScope: action.payload,
        // Add to recent scopes if not already there
        recentScopes: [
          action.payload,
          ...state.recentScopes.filter(scope => 
            scope.segment !== action.payload.segment || 
            !arraysEqual(scope.tags, action.payload.tags)
          ),
        ].slice(0, 10), // Keep only last 10
      };

    case 'SET_AVAILABLE_SCOPES':
      return {
        ...state,
        availableScopes: action.payload,
      };

    case 'ADD_RECENT_SCOPE':
      return {
        ...state,
        recentScopes: [
          action.payload,
          ...state.recentScopes.filter(scope => 
            scope.segment !== action.payload.segment || 
            !arraysEqual(scope.tags, action.payload.tags)
          ),
        ].slice(0, 10),
      };

    case 'UPDATE_SCOPE':
      return {
        ...state,
        activeScope: {
          ...state.activeScope,
          ...action.payload,
        },
      };

    case 'ADD_TAG':
      const newTags = [...state.activeScope.tags, action.payload];
      return {
        ...state,
        activeScope: {
          ...state.activeScope,
          tags: Array.from(new Set(newTags)), // Remove duplicates
        },
      };

    case 'REMOVE_TAG':
      return {
        ...state,
        activeScope: {
          ...state.activeScope,
          tags: state.activeScope.tags.filter(tag => tag !== action.payload),
        },
      };

    case 'CLEAR_ERROR':
      return {
        ...state,
        error: null,
      };

    default:
      return state;
  }
}

// ============================================================================
// SCOPE COLORS
// ============================================================================

const scopeColors: Record<Segment, string> = {
  personal: '#10b981',
  family: '#f59e0b',
  work: '#3b82f6',
  business: '#0ea5e9',
  health: '#ef4444',
  travel: '#8b5cf6',
  pets: '#f97316',
  assets: '#64748b',
};

// ============================================================================
// CONTEXT PROVIDER
// ============================================================================

const ScopeContext = createContext<ScopeContextValue | undefined>(undefined);

interface ScopeProviderProps {
  children: ReactNode;
  initialUser?: User | null;
  apiClient?: {
    getScopes: () => Promise<Scope[]>;
    createScope: (scope: Omit<Scope, 'metadata'>) => Promise<Scope>;
    deleteScope: (scopeId: string) => Promise<void>;
  };
}

export function ScopeProvider({ children, initialUser = null, apiClient }: ScopeProviderProps) {
  const [state, dispatch] = useReducer(scopeReducer, {
    ...initialState,
    user: initialUser,
  });

  // Load scopes on mount
  useEffect(() => {
    if (apiClient) {
      loadScopes();
    }
  }, [apiClient]);

  // Persist active scope to localStorage
  useEffect(() => {
    if (typeof window !== 'undefined') {
      try {
        localStorage.setItem('modulr-active-scope', JSON.stringify(state.activeScope));
      } catch (error) {
        console.warn('Failed to save active scope to localStorage:', error);
      }
    }
  }, [state.activeScope]);

  // Load active scope from localStorage on mount
  useEffect(() => {
    if (typeof window !== 'undefined') {
      try {
        const saved = localStorage.getItem('modulr-active-scope');
        if (saved) {
          const savedScope = JSON.parse(saved) as Scope;
          if (isValidScope(savedScope)) {
            dispatch({ type: 'SET_ACTIVE_SCOPE', payload: savedScope });
          }
        }
      } catch (error) {
        console.warn('Failed to load active scope from localStorage:', error);
      }
    }
  }, []);

  // ============================================================================
  // ACTIONS
  // ============================================================================

  const setActiveScope = (scope: Scope) => {
    if (isValidScope(scope)) {
      dispatch({ type: 'SET_ACTIVE_SCOPE', payload: scope });
    }
  };

  const addTag = (tag: string) => {
    if (tag && tag.trim()) {
      dispatch({ type: 'ADD_TAG', payload: tag.trim() });
    }
  };

  const removeTag = (tag: string) => {
    dispatch({ type: 'REMOVE_TAG', payload: tag });
  };

  const updateScope = (updates: Partial<Scope>) => {
    dispatch({ type: 'UPDATE_SCOPE', payload: updates });
  };

  const createScope = async (scope: Omit<Scope, 'metadata'>) => {
    if (!apiClient) {
      dispatch({ type: 'SET_ERROR', payload: 'API client not available' });
      return;
    }

    dispatch({ type: 'SET_LOADING', payload: true });

    try {
      const newScope = await apiClient.createScope(scope);
      dispatch({ type: 'SET_AVAILABLE_SCOPES', payload: [...state.availableScopes, newScope] });
      dispatch({ type: 'SET_ACTIVE_SCOPE', payload: newScope });
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : 'Failed to create scope' });
    } finally {
      dispatch({ type: 'SET_LOADING', payload: false });
    }
  };

  const deleteScope = async (scopeId: string) => {
    if (!apiClient) {
      dispatch({ type: 'SET_ERROR', payload: 'API client not available' });
      return;
    }

    dispatch({ type: 'SET_LOADING', payload: true });

    try {
      await apiClient.deleteScope(scopeId);
      const updatedScopes = state.availableScopes.filter(scope => 
        `${scope.segment}:${scope.tags.join(',')}` !== scopeId
      );
      dispatch({ type: 'SET_AVAILABLE_SCOPES', payload: updatedScopes });
      
      // If we deleted the active scope, switch to default
      if (state.activeScope && `${state.activeScope.segment}:${state.activeScope.tags.join(',')}` === scopeId) {
        dispatch({ type: 'SET_ACTIVE_SCOPE', payload: defaultScope });
      }
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : 'Failed to delete scope' });
    } finally {
      dispatch({ type: 'SET_LOADING', payload: false });
    }
  };

  const loadScopes = async () => {
    if (!apiClient) {
      return;
    }

    dispatch({ type: 'SET_LOADING', payload: true });

    try {
      const scopes = await apiClient.getScopes();
      dispatch({ type: 'SET_AVAILABLE_SCOPES', payload: scopes });
    } catch (error) {
      dispatch({ type: 'SET_ERROR', payload: error instanceof Error ? error.message : 'Failed to load scopes' });
    } finally {
      dispatch({ type: 'SET_LOADING', payload: false });
    }
  };

  const clearError = () => {
    dispatch({ type: 'CLEAR_ERROR' });
  };

  // ============================================================================
  // COMPUTED VALUES
  // ============================================================================

  const currentSegment = state.activeScope.segment;
  const scopeColor = scopeColors[currentSegment];
  const hasMultipleScopes = state.availableScopes.length > 1;

  // ============================================================================
  // CONTEXT VALUE
  // ============================================================================

  const contextValue: ScopeContextValue = {
    ...state,
    setActiveScope,
    addTag,
    removeTag,
    updateScope,
    createScope,
    deleteScope,
    loadScopes,
    clearError,
    currentSegment,
    scopeColor,
    hasMultipleScopes,
  };

  return (
    <ScopeContext.Provider value={contextValue}>
      {children}
    </ScopeContext.Provider>
  );
}

// ============================================================================
// HOOK
// ============================================================================

export function useScope(): ScopeContextValue {
  const context = useContext(ScopeContext);
  
  if (context === undefined) {
    throw new Error('useScope must be used within a ScopeProvider');
  }
  
  return context;
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

function isValidScope(scope: unknown): scope is Scope {
  return (
    typeof scope === 'object' &&
    scope !== null &&
    'segment' in scope &&
    typeof scope.segment === 'string' &&
    'tags' in scope &&
    Array.isArray(scope.tags) &&
    scope.tags.every((tag: unknown) => typeof tag === 'string')
  );
}

function arraysEqual<T>(a: T[], b: T[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false;
  }
  return true;
}

// ============================================================================
// HOOK VARIANTS
// ============================================================================

/**
 * Hook for getting only the current scope
 */
export function useCurrentScope(): Scope {
  return useScope().activeScope;
}

/**
 * Hook for getting scope color
 */
export function useScopeColor(): string {
  return useScope().scopeColor;
}

/**
 * Hook for getting current segment
 */
export function useCurrentSegment(): Segment {
  return useScope().currentSegment;
}

/**
 * Hook for scope loading state
 */
export function useScopeLoading(): boolean {
  return useScope().isLoading;
}

/**
 * Hook for scope error state
 */
export function useScopeError(): string | null {
  return useScope().error;
}

// ============================================================================
// EXPORTS
// ============================================================================

export { ScopeContext };
