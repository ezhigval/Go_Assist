/**
 * Client-side EventBus implementation
 * Provides typed event communication between components and backend synchronization
 */

import React from 'react';
import type { 
  EventBus as IEventBus, 
  EventName, 
  EventHandler, 
  TypedEvent, 
  EventOptions, 
  Logger 
} from '@modulr/core-types';

// ============================================================================
// EVENT BUS IMPLEMENTATION
// ============================================================================

export class EventBus implements IEventBus {
  private listeners = new Map<EventName, Set<EventHandler>>();
  private onceListeners = new Map<EventName, Set<EventHandler>>();
  private maxListeners = 100;
  private debug = false;
  private logger: Logger;

  constructor(logger?: Logger) {
    this.logger = logger || console;
    this.debug = process.env.NODE_ENV === 'development';
  }

  /**
   * Subscribe to an event
   * @param eventName - Event name to listen for
   * @param handler - Event handler function
   * @returns Unsubscribe function
   */
  on<T = unknown>(eventName: EventName, handler: EventHandler<T>): () => void {
    if (this.debug) {
      this.logger.debug(`[EventBus] Subscribing to ${eventName}`);
    }

    if (!this.listeners.has(eventName)) {
      this.listeners.set(eventName, new Set());
    }

    const handlers = this.listeners.get(eventName)!;
    
    if (handlers.size >= this.maxListeners) {
      this.logger.warn(`[EventBus] Too many listeners for ${eventName} (${handlers.size})`);
    }

    handlers.add(handler);

    // Return unsubscribe function
    return () => {
      if (this.debug) {
        this.logger.debug(`[EventBus] Unsubscribing from ${eventName}`);
      }
      handlers.delete(handler);
      
      if (handlers.size === 0) {
        this.listeners.delete(eventName);
      }
    };
  }

  /**
   * Subscribe to an event only once
   * @param eventName - Event name to listen for
   * @param handler - Event handler function
   * @returns Unsubscribe function
   */
  once<T = unknown>(eventName: EventName, handler: EventHandler<T>): () => void {
    if (this.debug) {
      this.logger.debug(`[EventBus] Subscribing once to ${eventName}`);
    }

    if (!this.onceListeners.has(eventName)) {
      this.onceListeners.set(eventName, new Set());
    }

    const handlers = this.onceListeners.get(eventName)!;
    handlers.add(handler);

    // Return unsubscribe function
    return () => {
      if (this.debug) {
        this.logger.debug(`[EventBus] Unsubscribing once from ${eventName}`);
      }
      handlers.delete(handler);
      
      if (handlers.size === 0) {
        this.onceListeners.delete(eventName);
      }
    };
  }

  /**
   * Emit an event
   * @param eventName - Event name
   * @param payload - Event payload
   * @param options - Event options
   */
  emit<T = unknown>(eventName: EventName, payload: T, options?: EventOptions): void {
    const event: TypedEvent<T> = {
      id: this.generateEventId(),
      name: eventName,
      payload,
      source: options?.source || 'client',
      traceId: options?.traceId || this.generateTraceId(),
      timestamp: Date.now(),
      context: options?.context,
    };

    if (this.debug) {
      this.logger.debug(`[EventBus] Emitting ${eventName}`, event);
    }

    // Handle regular listeners
    const handlers = this.listeners.get(eventName);
    if (handlers) {
      // Create a copy to avoid issues with handlers being removed during iteration
      const handlersArray = Array.from(handlers);
      
      for (const handler of handlersArray) {
        try {
          handler(event);
        } catch (error) {
          this.logger.error(`[EventBus] Error in handler for ${eventName}`, error as Error);
        }
      }
    }

    // Handle once listeners
    const onceHandlers = this.onceListeners.get(eventName);
    if (onceHandlers) {
      // Create a copy and clear the set before calling handlers
      const onceHandlersArray = Array.from(onceHandlers);
      this.onceListeners.delete(eventName);
      
      for (const handler of onceHandlersArray) {
        try {
          handler(event);
        } catch (error) {
          this.logger.error(`[EventBus] Error in once handler for ${eventName}`, error as Error);
        }
      }
    }
  }

  /**
   * Unsubscribe from an event
   * @param eventName - Event name
   * @param handler - Event handler function
   */
  off<T = unknown>(eventName: EventName, handler: EventHandler<T>): void {
    if (this.debug) {
      this.logger.debug(`[EventBus] Removing handler for ${eventName}`);
    }

    const handlers = this.listeners.get(eventName);
    if (handlers) {
      handlers.delete(handler);
      
      if (handlers.size === 0) {
        this.listeners.delete(eventName);
      }
    }

    const onceHandlers = this.onceListeners.get(eventName);
    if (onceHandlers) {
      onceHandlers.delete(handler);
      
      if (onceHandlers.size === 0) {
        this.onceListeners.delete(eventName);
      }
    }
  }

  /**
   * Clear all listeners
   */
  clear(): void {
    if (this.debug) {
      this.logger.debug('[EventBus] Clearing all listeners');
    }
    
    this.listeners.clear();
    this.onceListeners.clear();
  }

  /**
   * Get the number of listeners for an event
   * @param eventName - Event name
   * @returns Number of listeners
   */
  listenerCount(eventName: EventName): number {
    const regularCount = this.listeners.get(eventName)?.size || 0;
    const onceCount = this.onceListeners.get(eventName)?.size || 0;
    return regularCount + onceCount;
  }

  /**
   * Get all event names with listeners
   * @returns Array of event names
   */
  eventNames(): EventName[] {
    const names = new Set<EventName>();
    
    for (const eventName of this.listeners.keys()) {
      names.add(eventName);
    }
    
    for (const eventName of this.onceListeners.keys()) {
      names.add(eventName);
    }
    
    return Array.from(names);
  }

  /**
   * Set debug mode
   * @param enabled - Enable debug logging
   */
  setDebug(enabled: boolean): void {
    this.debug = enabled;
  }

  /**
   * Set maximum number of listeners per event
   * @param max - Maximum listeners
   */
  setMaxListeners(max: number): void {
    this.maxListeners = max;
  }

  // ============================================================================
  // PRIVATE METHODS
  // ============================================================================

  /**
   * Generate a unique event ID
   * @returns Event ID
   */
  private generateEventId(): string {
    return `evt_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Generate a trace ID for event tracking
   * @returns Trace ID
   */
  private generateTraceId(): string {
    return `trace_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}

// ============================================================================
// GLOBAL EVENT BUS INSTANCE
// ============================================================================

// Create a singleton instance for the entire application
export const eventBus = new EventBus();

// ============================================================================
// EVENT BUS UTILITIES
// ============================================================================

/**
 * Wait for an event to be emitted
 * @param eventName - Event name to wait for
 * @param timeout - Timeout in milliseconds (optional)
 * @returns Promise that resolves with the event
 */
export function waitForEvent<T = unknown>(
  eventName: EventName, 
  timeout?: number
): Promise<TypedEvent<T>> {
  return new Promise((resolve, reject) => {
    let timeoutId: NodeJS.Timeout | undefined;

    const unsubscribe = eventBus.once<T>(eventName, (event) => {
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
      resolve(event);
    });

    if (timeout) {
      timeoutId = setTimeout(() => {
        unsubscribe();
        reject(new Error(`Timeout waiting for event ${eventName}`));
      }, timeout);
    }
  });
}

/**
 * Emit an event and wait for a response
 * @param eventName - Event name to emit
 * @param payload - Event payload
 * @param responseEventName - Response event name
 * @param timeout - Timeout in milliseconds (optional)
 * @returns Promise that resolves with the response event
 */
export async function emitAndWait<TRequest = unknown, TResponse = unknown>(
  eventName: EventName,
  payload: TRequest,
  responseEventName: EventName,
  timeout?: number
): Promise<TypedEvent<TResponse>> {
  const traceId = eventBus['generateTraceId']();
  
  // Emit the request event
  eventBus.emit(eventName, payload, { traceId });
  
  // Wait for the response
  return waitForEvent<TResponse>(responseEventName, timeout);
}

/**
 * Create an event emitter with a specific source
 * @param source - Source name
 * @returns Event emitter functions
 */
export function createEventEmitter(source: string) {
  return {
    emit: <T = unknown>(eventName: EventName, payload: T, options?: Omit<EventOptions, 'source'>) => {
      eventBus.emit(eventName, payload, { ...options, source });
    },
    
    on: <T = unknown>(eventName: EventName, handler: EventHandler<T>) => {
      return eventBus.on(eventName, handler);
    },
    
    once: <T = unknown>(eventName: EventName, handler: EventHandler<T>) => {
      return eventBus.once(eventName, handler);
    },
    
    off: <T = unknown>(eventName: EventName, handler: EventHandler<T>) => {
      return eventBus.off(eventName, handler);
    },
  };
}

// ============================================================================
// EVENT BUS HOOKS
// ============================================================================

/**
 * React hook for subscribing to events
 * @param eventName - Event name to listen for
 * @param handler - Event handler function
 * @param deps - Dependency array (optional)
 */
export function useEventBus<T = unknown>(
  eventName: EventName, 
  handler: EventHandler<T>, 
  deps: React.DependencyList = []
): void {
  React.useEffect(() => {
    const unsubscribe = eventBus.on(eventName, handler);
    return unsubscribe;
  }, deps);
}

/**
 * React hook for subscribing to an event once
 * @param eventName - Event name to listen for
 * @param handler - Event handler function
 * @param deps - Dependency array (optional)
 */
export function useEventBusOnce<T = unknown>(
  eventName: EventName, 
  handler: EventHandler<T>, 
  deps: React.DependencyList = []
): void {
  React.useEffect(() => {
    const unsubscribe = eventBus.once(eventName, handler);
    return unsubscribe;
  }, deps);
}

/**
 * React hook for emitting events
 * @returns Event emitter function
 */
export function useEventEmitter(): {
  emit: <T = unknown>(eventName: EventName, payload: T, options?: EventOptions) => void;
} {
  return React.useMemo(() => ({
    emit: <T = unknown>(eventName: EventName, payload: T, options?: EventOptions) => {
      eventBus.emit(eventName, payload, options);
    },
  }), []);
}

// ============================================================================
// EVENT BUS MIDDLEWARE
// ============================================================================

export interface EventMiddleware {
  (event: TypedEvent, next: () => void): void;
}

/**
 * Add middleware to the event bus
 * @param middleware - Middleware function
 */
export function addMiddleware(middleware: EventMiddleware): void {
  // Store middleware in a global registry
  if (!globalThis.modulrEventMiddleware) {
    globalThis.modulrEventMiddleware = [];
  }
  globalThis.modulrEventMiddleware.push(middleware);
}

/**
 * Remove middleware from the event bus
 * @param middleware - Middleware function to remove
 */
export function removeMiddleware(middleware: EventMiddleware): void {
  if (globalThis.modulrEventMiddleware) {
    const index = globalThis.modulrEventMiddleware.indexOf(middleware);
    if (index > -1) {
      globalThis.modulrEventMiddleware.splice(index, 1);
    }
  }
}

// ============================================================================
// TYPE DECLARATIONS
// ============================================================================

declare global {
  var modulrEventMiddleware: EventMiddleware[] | undefined;
}
