import React, { createContext, useContext, useEffect, useMemo, useState } from 'react';
import type { Notification, PlatformCapabilities, Theme } from '@modulr/core-types';

interface TauriState {
  isDesktop: boolean;
  platform: 'windows' | 'macos' | 'linux';
  arch: 'x86_64' | 'aarch64';
  version: string;
  capabilities: PlatformCapabilities;
  systemInfo: {
    osType: string;
    osVersion: string;
    arch: string;
    hostname: string;
    cpuCount: number;
    totalMemory: number;
  };
  appInfo: {
    version: string;
    build: string;
    name: string;
    tauriVersion: string;
  };
  windowInfo: {
    isFocused: boolean;
    isVisible: boolean;
    isFullscreen: boolean;
    isMaximized: boolean;
    width: number;
    height: number;
  };
  theme: Theme;
  notifications: Notification[];
}

interface TauriContextValue extends TauriState {
  minimizeWindow: () => Promise<void>;
  maximizeWindow: () => Promise<void>;
  unmaximizeWindow: () => Promise<void>;
  closeWindow: () => Promise<void>;
  hideWindow: () => Promise<void>;
  showWindow: () => Promise<void>;
  setFullscreen: (fullscreen: boolean) => Promise<void>;
  centerWindow: () => Promise<void>;
  showNotification: (title: string, body: string) => Promise<void>;
  openExternal: (url: string) => Promise<void>;
  openFile: (path: string) => Promise<void>;
  showInFolder: (path: string) => Promise<void>;
  readFile: (path: string) => Promise<string>;
  writeFile: (path: string, content: string) => Promise<void>;
  exists: (path: string) => Promise<boolean>;
  mkdir: (path: string) => Promise<void>;
  readClipboard: () => Promise<string>;
  writeClipboard: (content: string) => Promise<void>;
  registerShortcut: (shortcut: string, action: () => void) => Promise<void>;
  unregisterShortcut: (shortcut: string) => Promise<void>;
  setTrayIcon: (icon: string) => Promise<void>;
  setTrayTooltip: (tooltip: string) => Promise<void>;
  enableAutoStart: () => Promise<void>;
  disableAutoStart: () => Promise<void>;
  isAutoStartEnabled: () => Promise<boolean>;
  setTheme: (theme: Theme) => void;
  restartApp: () => Promise<void>;
  getLaunchArgs: () => Promise<string[]>;
}

const TauriContext = createContext<TauriContextValue | undefined>(undefined);

export function TauriProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<TauriState>({
    isDesktop: false,
    platform: 'windows',
    arch: 'x86_64',
    version: '1.0.0',
    capabilities: {
      platform: 'desktop',
      hasNotifications: true,
      hasGeolocation: false,
      hasCamera: false,
      hasBiometrics: false,
      hasOfflineSupport: true,
      hasVoiceInput: false,
      screenInfo: {
        width: window.innerWidth,
        height: window.innerHeight,
        density: window.devicePixelRatio || 1,
        orientation: window.innerWidth > window.innerHeight ? 'landscape' : 'portrait',
        isSmall: window.innerWidth < 1024,
        isTouch: 'ontouchstart' in window,
      },
    },
    systemInfo: {
      osType: 'unknown',
      osVersion: 'unknown',
      arch: 'unknown',
      hostname: 'unknown',
      cpuCount: 0,
      totalMemory: 0,
    },
    appInfo: {
      version: '1.0.0',
      build: '1',
      name: 'Modulr',
      tauriVersion: '1.0.0',
    },
    windowInfo: {
      isFocused: true,
      isVisible: true,
      isFullscreen: false,
      isMaximized: false,
      width: window.innerWidth,
      height: window.innerHeight,
    },
    theme: 'light',
    notifications: [],
  });

  useEffect(() => {
    const runtime = window as unknown as { __TAURI__?: unknown };
    const hasTauri = Boolean(runtime.__TAURI__);
    if (hasTauri) {
      setState((prev) => ({ ...prev, isDesktop: true }));
    }
  }, []);

  const contextValue = useMemo<TauriContextValue>(
    () => ({
      ...state,
      minimizeWindow: async () => undefined,
      maximizeWindow: async () => {
        setState((prev) => ({ ...prev, windowInfo: { ...prev.windowInfo, isMaximized: true } }));
      },
      unmaximizeWindow: async () => {
        setState((prev) => ({ ...prev, windowInfo: { ...prev.windowInfo, isMaximized: false } }));
      },
      closeWindow: async () => {
        window.close();
      },
      hideWindow: async () => undefined,
      showWindow: async () => undefined,
      setFullscreen: async (fullscreen) => {
        setState((prev) => ({ ...prev, windowInfo: { ...prev.windowInfo, isFullscreen: fullscreen } }));
      },
      centerWindow: async () => undefined,
      showNotification: async (title, body) => {
        if ('Notification' in window && Notification.permission === 'granted') {
          new Notification(title, { body });
        }
      },
      openExternal: async (url) => {
        window.open(url, '_blank', 'noopener,noreferrer');
      },
      openFile: async () => undefined,
      showInFolder: async () => undefined,
      readFile: async () => '',
      writeFile: async () => undefined,
      exists: async () => false,
      mkdir: async () => undefined,
      readClipboard: async () => navigator.clipboard.readText(),
      writeClipboard: async (content) => {
        await navigator.clipboard.writeText(content);
      },
      registerShortcut: async () => undefined,
      unregisterShortcut: async () => undefined,
      setTrayIcon: async () => undefined,
      setTrayTooltip: async () => undefined,
      enableAutoStart: async () => undefined,
      disableAutoStart: async () => undefined,
      isAutoStartEnabled: async () => false,
      setTheme: (theme) => {
        setState((prev) => ({ ...prev, theme }));
        document.documentElement.classList.toggle('dark', theme === 'dark');
      },
      restartApp: async () => window.location.reload(),
      getLaunchArgs: async () => [],
    }),
    [state]
  );

  return <TauriContext.Provider value={contextValue}>{children}</TauriContext.Provider>;
}

export function useTauri(): TauriContextValue {
  const context = useContext(TauriContext);
  if (!context) {
    throw new Error('useTauri must be used inside TauriProvider');
  }
  return context;
}
