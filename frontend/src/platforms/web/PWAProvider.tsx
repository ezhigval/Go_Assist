import React, { createContext, useContext, useEffect, useMemo, useState } from 'react';
import type { Notification, PlatformCapabilities, Theme } from '@modulr/core-types';

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

interface PWAState {
  isInstallable: boolean;
  isInstalled: boolean;
  isOnline: boolean;
  serviceWorker: ServiceWorkerRegistration | null;
  installPrompt: BeforeInstallPromptEvent | null;
  pushSubscription: PushSubscription | null;
  notifications: Notification[];
  capabilities: PlatformCapabilities;
  theme: Theme;
}

interface PWAContextValue extends PWAState {
  installApp: () => Promise<boolean>;
  checkInstallable: () => void;
  updateServiceWorker: () => Promise<void>;
  skipWaiting: () => void;
  subscribeToPush: () => Promise<boolean>;
  unsubscribeFromPush: () => Promise<boolean>;
  requestNotificationPermission: () => Promise<boolean>;
  getStorageUsage: () => Promise<{ quota: number; usage: number }>;
  clearStorage: () => Promise<void>;
  setTheme: (theme: Theme) => void;
  checkOnlineStatus: () => boolean;
}

const PWAContext = createContext<PWAContextValue | undefined>(undefined);

export function PWAProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<PWAState>({
    isInstallable: false,
    isInstalled: false,
    isOnline: navigator.onLine,
    serviceWorker: null,
    installPrompt: null,
    pushSubscription: null,
    notifications: [],
    capabilities: {
      platform: 'web',
      hasNotifications: 'Notification' in window,
      hasGeolocation: 'geolocation' in navigator,
      hasCamera: 'mediaDevices' in navigator,
      hasBiometrics: false,
      hasOfflineSupport: 'serviceWorker' in navigator && 'caches' in window,
      hasVoiceInput: false,
      screenInfo: {
        width: window.innerWidth,
        height: window.innerHeight,
        density: window.devicePixelRatio || 1,
        orientation: window.innerWidth > window.innerHeight ? 'landscape' : 'portrait',
        isSmall: window.innerWidth < 768,
        isTouch: 'ontouchstart' in window,
      },
    },
    theme: 'light',
  });

  useEffect(() => {
    const onBeforeInstallPrompt = (event: Event) => {
      event.preventDefault();
      setState((prev) => ({
        ...prev,
        isInstallable: true,
        installPrompt: event as BeforeInstallPromptEvent,
      }));
    };

    const onInstalled = () => {
      setState((prev) => ({
        ...prev,
        isInstalled: true,
        isInstallable: false,
        installPrompt: null,
      }));
    };

    const onOnline = () => setState((prev) => ({ ...prev, isOnline: true }));
    const onOffline = () => setState((prev) => ({ ...prev, isOnline: false }));

    const isStandalone = window.matchMedia('(display-mode: standalone)').matches;
    setState((prev) => ({ ...prev, isInstalled: isStandalone }));

    window.addEventListener('beforeinstallprompt', onBeforeInstallPrompt);
    window.addEventListener('appinstalled', onInstalled);
    window.addEventListener('online', onOnline);
    window.addEventListener('offline', onOffline);

    if ('serviceWorker' in navigator) {
      void navigator.serviceWorker
        .register('/sw.js')
        .then((registration) => {
          setState((prev) => ({ ...prev, serviceWorker: registration }));
        })
        .catch(() => undefined);
    }

    return () => {
      window.removeEventListener('beforeinstallprompt', onBeforeInstallPrompt);
      window.removeEventListener('appinstalled', onInstalled);
      window.removeEventListener('online', onOnline);
      window.removeEventListener('offline', onOffline);
    };
  }, []);

  const contextValue = useMemo<PWAContextValue>(
    () => ({
      ...state,
      installApp: async () => {
        if (!state.installPrompt) return false;
        await state.installPrompt.prompt();
        const choice = await state.installPrompt.userChoice;
        const accepted = choice.outcome === 'accepted';
        setState((prev) => ({ ...prev, isInstalled: accepted, isInstallable: !accepted }));
        return accepted;
      },
      checkInstallable: () => {
        const isStandalone = window.matchMedia('(display-mode: standalone)').matches;
        setState((prev) => ({ ...prev, isInstalled: isStandalone }));
      },
      updateServiceWorker: async () => {
        await state.serviceWorker?.update();
      },
      skipWaiting: () => {
        state.serviceWorker?.waiting?.postMessage({ type: 'SKIP_WAITING' });
      },
      subscribeToPush: async () => false,
      unsubscribeFromPush: async () => true,
      requestNotificationPermission: async () => {
        if (!('Notification' in window)) return false;
        const permission = await Notification.requestPermission();
        return permission === 'granted';
      },
      getStorageUsage: async () => {
        if ('storage' in navigator && navigator.storage.estimate) {
          const estimate = await navigator.storage.estimate();
          return { quota: estimate.quota ?? 0, usage: estimate.usage ?? 0 };
        }
        return { quota: 0, usage: 0 };
      },
      clearStorage: async () => {
        localStorage.clear();
        sessionStorage.clear();
        if ('caches' in window) {
          const keys = await caches.keys();
          await Promise.all(keys.map((key) => caches.delete(key)));
        }
      },
      setTheme: (theme) => {
        setState((prev) => ({ ...prev, theme }));
        document.documentElement.classList.toggle('dark', theme === 'dark');
      },
      checkOnlineStatus: () => navigator.onLine,
    }),
    [state]
  );

  return <PWAContext.Provider value={contextValue}>{children}</PWAContext.Provider>;
}

export function usePWA(): PWAContextValue {
  const context = useContext(PWAContext);
  if (!context) {
    throw new Error('usePWA must be used inside PWAProvider');
  }
  return context;
}
