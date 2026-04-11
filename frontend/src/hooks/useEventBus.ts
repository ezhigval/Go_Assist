import { useEffect } from 'react';
import type { EventHandler, EventName } from '@modulr/core-types';
import { eventBus } from '../lib/eventBus';

export function useEventBus<T>(eventName: EventName, handler: EventHandler<T>): void {
  useEffect(() => {
    return eventBus.on(eventName, handler);
  }, [eventName, handler]);
}
