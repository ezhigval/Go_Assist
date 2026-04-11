import type {
  EventBus as EventBusContract,
  EventHandler,
  EventName,
  EventOptions,
  TypedEvent,
} from '@modulr/core-types';

function newEventID(): string {
  return `evt_${Date.now()}_${Math.floor(Math.random() * 1_000_000)}`;
}

class InMemoryEventBus implements EventBusContract {
  private handlers = new Map<EventName, Set<EventHandler>>();

  on<T>(eventName: EventName, handler: EventHandler<T>): () => void {
    const bucket = this.handlers.get(eventName) ?? new Set<EventHandler>();
    bucket.add(handler as EventHandler);
    this.handlers.set(eventName, bucket);
    return () => this.off(eventName, handler);
  }

  off<T>(eventName: EventName, handler: EventHandler<T>): void {
    const bucket = this.handlers.get(eventName);
    if (!bucket) return;
    bucket.delete(handler as EventHandler);
    if (bucket.size === 0) {
      this.handlers.delete(eventName);
    }
  }

  emit<T>(eventName: EventName, payload: T, options?: EventOptions): void {
    const event: TypedEvent<T> = {
      id: newEventID(),
      name: eventName,
      payload,
      source: options?.source ?? 'frontend',
      traceId: options?.traceId ?? `tr_${Date.now()}`,
      timestamp: Date.now(),
    };
    if (options?.context) {
      event.context = options.context;
    }

    const bucket = this.handlers.get(eventName);
    if (!bucket) return;
    for (const handler of bucket) {
      void handler(event);
    }
  }

  once<T>(eventName: EventName, handler: EventHandler<T>): () => void {
    const wrapper: EventHandler<T> = (event) => {
      this.off(eventName, wrapper);
      return handler(event);
    };
    return this.on(eventName, wrapper);
  }

  clear(): void {
    this.handlers.clear();
  }

  listenerCount(eventName: EventName): number {
    return this.handlers.get(eventName)?.size ?? 0;
  }
}

export const eventBus = new InMemoryEventBus();
