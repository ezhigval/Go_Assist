/**
 * Telegram WebApp Provider
 * Integrates with Telegram Mini App SDK and provides platform-specific functionality
 */

import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import type { User, PlatformCapabilities, Theme } from '@modulr/core-types';

// ============================================================================
// TELEGRAM WEBAPP TYPES
// ============================================================================

interface TelegramWebApp {
  ready: () => void;
  expand: () => void;
  close: () => void;
  sendData: (data: string) => void;
  openLink: (url: string) => void;
  openTelegramLink: (url: string) => void;
  openInvoice: (url: string, callback: (status: string) => void) => void;
  showPopup: (params: PopupParams, callback: (buttonId: string) => void) => void;
  showAlert: (message: string, callback: () => void) => void;
  showConfirm: (message: string, callback: (confirmed: boolean) => void) => void;
  requestWriteAccess: (callback: (granted: boolean) => void) => void;
  requestContact: (callback: (shared: boolean) => void) => void;
  initData: string;
  initDataUnsafe: {
    query_id?: string;
    user?: {
      id: number;
      first_name: string;
      last_name?: string;
      username?: string;
      language_code?: string;
      is_premium?: boolean;
      photo_url?: string;
    };
    receiver?: {
      id: number;
      first_name: string;
      last_name?: string;
      username?: string;
      language_code?: string;
    };
    chat?: {
      id: number;
      first_name?: string;
      last_name?: string;
      username?: string;
      type: 'private' | 'group' | 'supergroup' | 'channel';
      title: string;
      photo_url?: string;
    };
    start_param?: string;
    auth_date: number;
    hash: string;
  };
  theme: {
    bg_color?: string;
    text_color?: string;
    hint_color?: string;
    link_color?: string;
    button_color?: string;
    button_text_color?: string;
    header_bg_color?: string;
    accent_text_color?: string;
    section_bg_color?: string;
    section_header_text_color?: string;
    subtitle_text_color?: string;
    destructive_text_color?: string;
  };
  colorScheme: 'light' | 'dark';
  viewport: {
    height: number;
    width: number;
    isExpanded: boolean;
    stable_height: number;
    expand: () => void;
    isStateStable: boolean;
  };
  onEvent: (eventType: string, callback: () => void) => void;
  offEvent: (eventType: string, callback: () => void) => void;
  MainButton: {
    text: string;
    color: string;
    textColor: string;
    isVisible: boolean;
    isActive: boolean;
    setText: (text: string) => void;
    onClick: (callback: () => void) => void;
    offClick: (callback: () => void) => void;
    show: () => void;
    hide: () => void;
    enable: () => void;
    disable: () => void;
    showProgress: (leaveActive?: boolean) => void;
    hideProgress: () => void;
    setParams: (params: { text?: string; color?: string; text_color?: string }) => void;
  };
  BackButton: {
    isVisible: boolean;
    show: () => void;
    hide: () => void;
    onClick: (callback: () => void) => void;
    offClick: (callback: () => void) => void;
  };
  SettingsButton: {
    isVisible: boolean;
    onClick: (callback: () => void) => void;
    offClick: (callback: () => void) => void;
  };
  HapticFeedback: {
    impactOccurred: (style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft') => void;
    notificationOccurred: (type: 'error' | 'success' | 'warning') => void;
    selectionChanged: () => void;
  };
  CloudStorage: {
    setItem: (key: string, value: string, callback: (error?: Error) => void) => void;
    getItem: (key: string, callback: (error?: Error, value?: string) => void) => void;
    removeItem: (key: string, callback: (error?: Error) => void) => void;
    getKeys: (callback: (error?: Error, keys?: string[]) => void) => void;
  };
  BiometricManager: {
    isBiometricAvailable: boolean;
    isAccessGranted: boolean;
    isAccessRequested: boolean;
    updateBiometricToken: (token: string, callback: (updated: boolean) => void) => void;
    authenticate: (reason: string, callback: (authenticated: boolean, token?: string) => void) => void;
    openSettings: () => void;
    requestAccess: (callback: (granted: boolean) => void) => void;
  };
}

interface PopupParams {
  title?: string;
  message: string;
  buttons: PopupButton[];
}

interface PopupButton {
  id: string;
  type?: 'default' | 'ok' | 'close' | 'cancel' | 'destructive';
  text: string;
}

// ============================================================================
// TELEGRAM CONTEXT TYPES
// ============================================================================

interface TelegramState {
  webApp: TelegramWebApp | null;
  user: User | null;
  theme: Theme;
  capabilities: PlatformCapabilities;
  isReady: boolean;
  error: string | null;
}

interface TelegramContextValue extends TelegramState {
  // Actions
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
  
  // Storage
  setStorageItem: (key: string, value: string) => Promise<void>;
  getStorageItem: (key: string) => Promise<string | null>;
  removeStorageItem: (key: string) => Promise<void>;
  
  // Biometrics
  authenticateBiometric: (reason: string) => Promise<boolean>;
  requestBiometricAccess: () => Promise<boolean>;
}

// ============================================================================
// CONTEXT
// ============================================================================

const TelegramContext = createContext<TelegramContextValue | undefined>(undefined);

interface TelegramProviderProps {
  children: ReactNode;
}

export function TelegramProvider({ children }: TelegramProviderProps) {
  const [state, setState] = useState<TelegramState>({
    webApp: null,
    user: null,
    theme: 'light',
    capabilities: {
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
    },
    isReady: false,
    error: null,
  });

  // ============================================================================
  // INITIALIZATION
  // ============================================================================

  const initialize = () => {
    if (typeof window === 'undefined') return;

    const telegramWebApp = (window as any).Telegram?.WebApp as TelegramWebApp;
    
    if (!telegramWebApp) {
      setState(prev => ({
        ...prev,
        error: 'Telegram WebApp SDK not available',
      }));
      return;
    }

    try {
      // Initialize WebApp
      telegramWebApp.ready();

      // Parse user data
      const user = parseTelegramUser(telegramWebApp.initDataUnsafe.user);

      // Parse theme
      const theme = telegramWebApp.colorScheme === 'dark' ? 'dark' : 'light';

      // Get capabilities
      const capabilities = parseTelegramCapabilities(telegramWebApp);

      setState({
        webApp: telegramWebApp,
        user,
        theme,
        capabilities,
        isReady: true,
        error: null,
      });

      // Set up event listeners
      setupEventListeners(telegramWebApp);

    } catch (error) {
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to initialize Telegram WebApp',
      }));
    }
  };

  useEffect(() => {
    initialize();
  }, []);

  // ============================================================================
  // EVENT LISTENERS
  // ============================================================================

  const setupEventListeners = (webApp: TelegramWebApp) => {
    // Viewport changes
    webApp.onEvent('viewportChanged', () => {
      setState(prev => ({
        ...prev,
        capabilities: {
          ...prev.capabilities,
          screenInfo: {
            ...prev.capabilities.screenInfo,
            width: webApp.viewport.width,
            height: webApp.viewport.height,
          },
        },
      }));
    });

    // Theme changes
    webApp.onEvent('themeChanged', () => {
      const theme = webApp.colorScheme === 'dark' ? 'dark' : 'light';
      setState(prev => ({ ...prev, theme }));
    });
  };

  // ============================================================================
  // UTILITY FUNCTIONS
  // ============================================================================

  const parseTelegramUser = (tgUser?: any): User | null => {
    if (!tgUser) return null;

    return {
      id: tgUser.id.toString(),
      username: tgUser.username || '',
      displayName: `${tgUser.first_name}${tgUser.last_name ? ' ' + tgUser.last_name : ''}`,
      avatar: tgUser.photo_url,
      email: undefined, // Telegram doesn't provide email
      roles: ['user'],
      preferences: {
        language: tgUser.language_code || 'ru',
        theme: 'auto',
        notifications: {
          push: true,
          email: false,
          inApp: true,
          types: ['system', 'reminder', 'message'],
        },
        privacy: {
          shareAnalytics: false,
          shareCrashReports: false,
          shareUsageData: false,
          dataRetention: '90days',
        },
        accessibility: {
          fontSize: 'base',
          highContrast: false,
          reduceMotion: false,
          screenReader: false,
          keyboardNavigation: true,
        },
      },
      session: {
        token: '', // Will be set by backend
        refreshToken: '',
        expiresAt: 0,
        isActive: true,
        lastActivity: Date.now(),
      },
    };
  };

  const parseTelegramCapabilities = (webApp: TelegramWebApp): PlatformCapabilities => {
    return {
      platform: 'telegram',
      hasNotifications: true,
      hasGeolocation: false,
      hasCamera: false,
      hasBiometrics: webApp.BiometricManager.isBiometricAvailable,
      hasOfflineSupport: false,
      hasVoiceInput: false,
      screenInfo: {
        width: webApp.viewport.width,
        height: webApp.viewport.height,
        density: window.devicePixelRatio || 1,
        orientation: webApp.viewport.width > webApp.viewport.height ? 'landscape' : 'portrait',
        isSmall: webApp.viewport.width < 768,
        isTouch: true,
      },
    };
  };

  // ============================================================================
  // UI ACTIONS
  // ============================================================================

  const showMainButton = (text: string, onClick: () => void) => {
    if (!state.webApp) return;

    state.webApp.MainButton.setText(text);
    state.webApp.MainButton.onClick(onClick);
    state.webApp.MainButton.show();
  };

  const hideMainButton = () => {
    if (!state.webApp) return;
    state.webApp.MainButton.hide();
  };

  const showBackButton = (onClick: () => void) => {
    if (!state.webApp) return;

    state.webApp.BackButton.onClick(onClick);
    state.webApp.BackButton.show();
  };

  const hideBackButton = () => {
    if (!state.webApp) return;
    state.webApp.BackButton.hide();
  };

  const showAlert = (message: string, callback?: () => void) => {
    if (!state.webApp) return;
    state.webApp.showAlert(message, callback);
  };

  const showConfirm = (message: string, callback: (confirmed: boolean) => void) => {
    if (!state.webApp) return;
    state.webApp.showConfirm(message, callback);
  };

  const hapticImpact = (style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft') => {
    if (!state.webApp) return;
    state.webApp.HapticFeedback.impactOccurred(style);
  };

  const hapticNotification = (type: 'error' | 'success' | 'warning') => {
    if (!state.webApp) return;
    state.webApp.HapticFeedback.notificationOccurred(type);
  };

  const expandViewport = () => {
    if (!state.webApp) return;
    state.webApp.viewport.expand();
  };

  const closeWebApp = () => {
    if (!state.webApp) return;
    state.webApp.close();
  };

  const sendDataToBot = (data: string) => {
    if (!state.webApp) return;
    state.webApp.sendData(data);
  };

  // ============================================================================
  // STORAGE ACTIONS
  // ============================================================================

  const setStorageItem = (key: string, value: string): Promise<void> => {
    return new Promise((resolve, reject) => {
      if (!state.webApp) {
        reject(new Error('Telegram WebApp not available'));
        return;
      }

      state.webApp.CloudStorage.setItem(key, value, (error) => {
        if (error) {
          reject(error);
        } else {
          resolve();
        }
      });
    });
  };

  const getStorageItem = (key: string): Promise<string | null> => {
    return new Promise((resolve, reject) => {
      if (!state.webApp) {
        reject(new Error('Telegram WebApp not available'));
        return;
      }

      state.webApp.CloudStorage.getItem(key, (error, value) => {
        if (error) {
          reject(error);
        } else {
          resolve(value || null);
        }
      });
    });
  };

  const removeStorageItem = (key: string): Promise<void> => {
    return new Promise((resolve, reject) => {
      if (!state.webApp) {
        reject(new Error('Telegram WebApp not available'));
        return;
      }

      state.webApp.CloudStorage.removeItem(key, (error) => {
        if (error) {
          reject(error);
        } else {
          resolve();
        }
      });
    });
  };

  // ============================================================================
  // BIOMETRIC ACTIONS
  // ============================================================================

  const authenticateBiometric = (reason: string): Promise<boolean> => {
    return new Promise((resolve, reject) => {
      if (!state.webApp) {
        reject(new Error('Telegram WebApp not available'));
        return;
      }

      state.webApp.BiometricManager.authenticate(reason, (authenticated, token) => {
        resolve(authenticated);
      });
    });
  };

  const requestBiometricAccess = (): Promise<boolean> => {
    return new Promise((resolve, reject) => {
      if (!state.webApp) {
        reject(new Error('Telegram WebApp not available'));
        return;
      }

      state.webApp.BiometricManager.requestAccess((granted) => {
        resolve(granted);
      });
    });
  };

  // ============================================================================
  // CONTEXT VALUE
  // ============================================================================

  const contextValue: TelegramContextValue = {
    ...state,
    initialize,
    showMainButton,
    hideMainButton,
    showBackButton,
    hideBackButton,
    showAlert,
    showConfirm,
    hapticImpact,
    hapticNotification,
    expandViewport,
    closeWebApp,
    sendDataToBot,
    setStorageItem,
    getStorageItem,
    removeStorageItem,
    authenticateBiometric,
    requestBiometricAccess,
  };

  return (
    <TelegramContext.Provider value={contextValue}>
      {children}
    </TelegramContext.Provider>
  );
}

// ============================================================================
// HOOK
// ============================================================================

export function useTelegram(): TelegramContextValue {
  const context = useContext(TelegramContext);
  
  if (context === undefined) {
    throw new Error('useTelegram must be used within a TelegramProvider');
  }
  
  return context;
}

// ============================================================================
// SPECIALIZED HOOKS
// ============================================================================

/**
 * Hook for Telegram theme
 */
export function useTelegramTheme(): Theme {
  return useTelegram().theme;
}

/**
 * Hook for Telegram user
 */
export function useTelegramUser(): User | null {
  return useTelegram().user;
}

/**
 * Hook for Telegram capabilities
 */
export function useTelegramCapabilities(): PlatformCapabilities {
  return useTelegram().capabilities;
}

/**
 * Hook for checking if Telegram is ready
 */
export function useTelegramReady(): boolean {
  return useTelegram().isReady;
}

// ============================================================================
// EXPORTS
// ============================================================================

export { TelegramContext };
export type { TelegramWebApp, PopupParams, PopupButton };
