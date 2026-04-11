import React, { createContext, useContext, useEffect, useMemo, useState } from 'react';
import type { PlatformCapabilities, Theme } from '@modulr/core-types';

interface CameraOptions {
  quality?: number;
  allowEditing?: boolean;
  source?: 'camera' | 'photos';
  resultType?: 'uri' | 'base64';
}

interface ImagePickerOptions {
  multiple?: boolean;
  quality?: number;
  maxFiles?: number;
}

interface BiometricOptions {
  reason?: string;
  title?: string;
  subtitle?: string;
  description?: string;
}

interface NotificationAttachment {
  id: string;
  url: string;
}

interface NotificationSchedule {
  id?: string;
  title: string;
  body: string;
  schedule?: {
    at?: Date;
    repeats?: boolean;
    every?: 'minute' | 'hour' | 'day' | 'week' | 'month' | 'year';
  };
  sound?: string;
  badge?: number;
  attachments?: NotificationAttachment[];
}

interface CapacitorState {
  isNative: boolean;
  platform: 'ios' | 'android' | 'web';
  version: string;
  capabilities: PlatformCapabilities;
  permissions: Record<string, boolean>;
  biometricAvailable: boolean;
  networkStatus: {
    connected: boolean;
    connectionType: string;
  };
  appInfo: {
    version: string;
    build: string;
    id: string;
  };
  theme: Theme;
}

interface CapacitorContextValue extends CapacitorState {
  requestPermission: (permission: string) => Promise<boolean>;
  checkPermission: (permission: string) => Promise<boolean>;
  takePhoto: (options?: CameraOptions) => Promise<{ photo: string; data?: string }>;
  pickImage: (options?: ImagePickerOptions) => Promise<{ photos: string[] }>;
  getCurrentPosition: () => Promise<{ latitude: number; longitude: number }>;
  watchPosition: (callback: (position: { latitude: number; longitude: number }) => void) => Promise<string>;
  clearWatch: (watchId: string) => void;
  authenticate: (options?: BiometricOptions) => Promise<boolean>;
  checkBiometricAvailability: () => Promise<boolean>;
  scheduleNotification: (notification: NotificationSchedule) => Promise<string>;
  cancelNotification: (id: string) => Promise<void>;
  getPendingNotifications: () => Promise<NotificationSchedule[]>;
  getStorageInfo: () => Promise<{ quota: number; usage: number }>;
  clearStorage: () => Promise<void>;
  exitApp: () => Promise<void>;
  minimizeApp: () => Promise<void>;
  getLaunchUrl: () => Promise<string>;
  checkNetworkStatus: () => Promise<{ connected: boolean; connectionType: string }>;
  setTheme: (theme: Theme) => void;
}

const CapacitorContext = createContext<CapacitorContextValue | undefined>(undefined);

export function CapacitorProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<CapacitorState>({
    isNative: false,
    platform: 'web',
    version: '1.0.0',
    capabilities: {
      platform: 'mobile',
      hasNotifications: 'Notification' in window,
      hasGeolocation: 'geolocation' in navigator,
      hasCamera: 'mediaDevices' in navigator,
      hasBiometrics: false,
      hasOfflineSupport: 'serviceWorker' in navigator,
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
    permissions: {},
    biometricAvailable: false,
    networkStatus: {
      connected: navigator.onLine,
      connectionType: 'unknown',
    },
    appInfo: {
      version: '1.0.0',
      build: '1',
      id: 'modulr.web',
    },
    theme: 'light',
  });

  useEffect(() => {
    const onOnline = () => {
      setState((prev) => ({
        ...prev,
        networkStatus: { connected: true, connectionType: prev.networkStatus.connectionType },
      }));
    };
    const onOffline = () => {
      setState((prev) => ({
        ...prev,
        networkStatus: { connected: false, connectionType: prev.networkStatus.connectionType },
      }));
    };
    window.addEventListener('online', onOnline);
    window.addEventListener('offline', onOffline);

    const rawCapacitor = (window as unknown as { Capacitor?: { getPlatform?: () => string; isNativePlatform?: () => boolean } })
      .Capacitor;
    if (rawCapacitor?.getPlatform) {
      const platform = rawCapacitor.getPlatform();
      const isNative = typeof rawCapacitor.isNativePlatform === 'function' ? rawCapacitor.isNativePlatform() : platform !== 'web';
      setState((prev) => ({
        ...prev,
        isNative,
        platform: platform === 'ios' || platform === 'android' ? platform : 'web',
      }));
    }

    return () => {
      window.removeEventListener('online', onOnline);
      window.removeEventListener('offline', onOffline);
    };
  }, []);

  const contextValue = useMemo<CapacitorContextValue>(
    () => ({
      ...state,
      requestPermission: async (permission) => {
        setState((prev) => ({ ...prev, permissions: { ...prev.permissions, [permission]: true } }));
        return true;
      },
      checkPermission: async (permission) => Boolean(state.permissions[permission]),
      takePhoto: async () => {
        throw new Error('camera is unavailable in web fallback mode');
      },
      pickImage: async () => ({ photos: [] }),
      getCurrentPosition: async () =>
        new Promise((resolve, reject) => {
          navigator.geolocation.getCurrentPosition(
            (position) => {
              resolve({ latitude: position.coords.latitude, longitude: position.coords.longitude });
            },
            (error) => reject(error),
            { enableHighAccuracy: true }
          );
        }),
      watchPosition: async (callback) => {
        const watchId = navigator.geolocation.watchPosition((position) => {
          callback({ latitude: position.coords.latitude, longitude: position.coords.longitude });
        });
        return String(watchId);
      },
      clearWatch: (watchId) => navigator.geolocation.clearWatch(Number.parseInt(watchId, 10)),
      authenticate: async () => false,
      checkBiometricAvailability: async () => false,
      scheduleNotification: async (notification) => {
        if ('Notification' in window && Notification.permission === 'granted') {
          new Notification(notification.title, { body: notification.body });
          return notification.id ?? String(Date.now());
        }
        return notification.id ?? String(Date.now());
      },
      cancelNotification: async () => undefined,
      getPendingNotifications: async () => [],
      getStorageInfo: async () => {
        if ('storage' in navigator && navigator.storage.estimate) {
          const estimate = await navigator.storage.estimate();
          return { quota: estimate.quota ?? 0, usage: estimate.usage ?? 0 };
        }
        return { quota: 0, usage: 0 };
      },
      clearStorage: async () => {
        localStorage.clear();
        sessionStorage.clear();
      },
      exitApp: async () => undefined,
      minimizeApp: async () => undefined,
      getLaunchUrl: async () => window.location.href,
      checkNetworkStatus: async () => ({
        connected: navigator.onLine,
        connectionType: state.networkStatus.connectionType,
      }),
      setTheme: (theme) => {
        setState((prev) => ({ ...prev, theme }));
        document.documentElement.classList.toggle('dark', theme === 'dark');
      },
    }),
    [state]
  );

  return <CapacitorContext.Provider value={contextValue}>{children}</CapacitorContext.Provider>;
}

export function useCapacitor(): CapacitorContextValue {
  const context = useContext(CapacitorContext);
  if (!context) {
    throw new Error('useCapacitor must be used inside CapacitorProvider');
  }
  return context;
}
