import React, { createContext, useContext, useEffect, useMemo, useState } from 'react';
import type { PlatformCapabilities, Theme, User } from '@modulr/core-types';

interface TelegramWebApp {
  ready?: () => void;
  expand?: () => void;
  close?: () => void;
  sendData?: (data: string) => void;
  showAlert?: (message: string, callback?: () => void) => void;
  showConfirm?: (message: string, callback: (confirmed: boolean) => void) => void;
  colorScheme?: 'light' | 'dark';
  initDataUnsafe?: {
    user?: {
      id: number;
      username?: string;
      first_name?: string;
      last_name?: string;
      language_code?: string;
    };
  };
  MainButton?: {
    setText: (text: string) => void;
    onClick: (cb: () => void) => void;
    show: () => void;
    hide: () => void;
  };
  BackButton?: {
    onClick: (cb: () => void) => void;
    show: () => void;
    hide: () => void;
  };
}

interface TelegramState {
  webApp: TelegramWebApp | null;
  user: User | null;
  theme: Theme;
  capabilities: PlatformCapabilities;
  isReady: boolean;
  error: string | null;
}

interface TelegramContextValue extends TelegramState {
  initialize: () => void;
  showMainButton: (text: string, onClick: () => void) => void;
  hideMainButton: () => void;
  showBackButton: (onClick: () => void) => void;
  hideBackButton: () => void;
  showAlert: (message: string, callback?: () => void) => void;
  showConfirm: (message: string, callback: (confirmed: boolean) => void) => void;
  hapticImpact: (style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft') => void;
  hapticNotification: (type: 'error' | 'success' | 'warning') => void;
  expandViewport: () => void;
  closeWebApp: () => void;
  sendDataToBot: (data: string) => void;
  setStorageItem: (key: string, value: string) => Promise<void>;
  getStorageItem: (key: string) => Promise<string | null>;
  removeStorageItem: (key: string) => Promise<void>;
  authenticateBiometric: (reason: string) => Promise<boolean>;
  requestBiometricAccess: () => Promise<boolean>;
}

const fallbackCapabilities: PlatformCapabilities = {
  platform: 'telegram',
  hasNotifications: true,
  hasGeolocation: false,
  hasCamera: false,
  hasBiometrics: false,
  hasOfflineSupport: false,
  hasVoiceInput: false,
  screenInfo: {
    width: 0,
    height: 0,
    density: 1,
    orientation: 'portrait',
    isSmall: true,
    isTouch: true,
  },
};

const TelegramContext = createContext<TelegramContextValue | undefined>(undefined);

function parseUser(webApp: TelegramWebApp | null): User | null {
  const tgUser = webApp?.initDataUnsafe?.user;
  if (!tgUser) return null;
  return {
    id: String(tgUser.id),
    username: tgUser.username ?? '',
    displayName: [tgUser.first_name, tgUser.last_name].filter(Boolean).join(' ') || tgUser.username || 'Telegram User',
    roles: ['user'],
    preferences: {
      language: tgUser.language_code || 'ru',
      theme: 'auto',
      notifications: { push: true, email: false, inApp: true, types: ['system', 'reminder', 'message'] },
      privacy: { shareAnalytics: false, shareCrashReports: false, shareUsageData: false, dataRetention: '90days' },
      accessibility: { fontSize: 'base', highContrast: false, reduceMotion: false, screenReader: false, keyboardNavigation: true },
    },
    session: {
      token: '',
      refreshToken: '',
      expiresAt: 0,
      isActive: true,
      lastActivity: Date.now(),
    },
  };
}

export function TelegramProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<TelegramState>({
    webApp: null,
    user: null,
    theme: 'light',
    capabilities: fallbackCapabilities,
    isReady: false,
    error: null,
  });

  const initialize = () => {
    const maybeTelegram = (window as unknown as { Telegram?: { WebApp?: TelegramWebApp } }).Telegram?.WebApp ?? null;
    if (!maybeTelegram) {
      setState((prev) => ({ ...prev, isReady: false, error: 'Telegram WebApp SDK not found' }));
      return;
    }

    maybeTelegram.ready?.();
    maybeTelegram.expand?.();

    setState((prev) => ({
      ...prev,
      webApp: maybeTelegram,
      user: parseUser(maybeTelegram),
      theme: maybeTelegram.colorScheme === 'dark' ? 'dark' : 'light',
      isReady: true,
      error: null,
      capabilities: {
        ...prev.capabilities,
        screenInfo: {
          ...prev.capabilities.screenInfo,
          width: window.innerWidth,
          height: window.innerHeight,
          density: window.devicePixelRatio || 1,
          orientation: window.innerWidth > window.innerHeight ? 'landscape' : 'portrait',
          isSmall: window.innerWidth < 768,
          isTouch: true,
        },
      },
    }));
  };

  useEffect(() => {
    initialize();
  }, []);

  const contextValue = useMemo<TelegramContextValue>(
    () => ({
      ...state,
      initialize,
      showMainButton: (text, onClick) => {
        state.webApp?.MainButton?.setText(text);
        state.webApp?.MainButton?.onClick(onClick);
        state.webApp?.MainButton?.show();
      },
      hideMainButton: () => state.webApp?.MainButton?.hide(),
      showBackButton: (onClick) => {
        state.webApp?.BackButton?.onClick(onClick);
        state.webApp?.BackButton?.show();
      },
      hideBackButton: () => state.webApp?.BackButton?.hide(),
      showAlert: (message, callback) => state.webApp?.showAlert?.(message, callback),
      showConfirm: (message, callback) => state.webApp?.showConfirm?.(message, callback),
      hapticImpact: () => undefined,
      hapticNotification: () => undefined,
      expandViewport: () => state.webApp?.expand?.(),
      closeWebApp: () => state.webApp?.close?.(),
      sendDataToBot: (data) => state.webApp?.sendData?.(data),
      setStorageItem: async (key, value) => {
        localStorage.setItem(`tg:${key}`, value);
      },
      getStorageItem: async (key) => localStorage.getItem(`tg:${key}`),
      removeStorageItem: async (key) => {
        localStorage.removeItem(`tg:${key}`);
      },
      authenticateBiometric: async () => false,
      requestBiometricAccess: async () => false,
    }),
    [state]
  );

  return <TelegramContext.Provider value={contextValue}>{children}</TelegramContext.Provider>;
}

export function useTelegram(): TelegramContextValue {
  const context = useContext(TelegramContext);
  if (!context) {
    throw new Error('useTelegram must be used inside TelegramProvider');
  }
  return context;
}
