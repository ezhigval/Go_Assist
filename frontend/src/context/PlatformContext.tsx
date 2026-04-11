import { createContext, useContext, useMemo } from 'react';
import type { Platform } from '@modulr/core-types';

interface PlatformContextValue {
  platform: Platform;
}

const PlatformContext = createContext<PlatformContextValue | undefined>(undefined);

function detectPlatform(): Platform {
  const runtime = window as unknown as {
    Telegram?: unknown;
    Capacitor?: unknown;
    __TAURI__?: unknown;
  };
  if (runtime.Telegram) return 'telegram';
  if (runtime.Capacitor) return 'mobile';
  if (runtime.__TAURI__) return 'desktop';
  return 'web';
}

export function PlatformProvider({ children }: { children: React.ReactNode }) {
  const value = useMemo<PlatformContextValue>(() => ({ platform: detectPlatform() }), []);
  return <PlatformContext.Provider value={value}>{children}</PlatformContext.Provider>;
}

export function usePlatform(): PlatformContextValue {
  const context = useContext(PlatformContext);
  if (!context) {
    throw new Error('usePlatform must be used inside PlatformProvider');
  }
  return context;
}
