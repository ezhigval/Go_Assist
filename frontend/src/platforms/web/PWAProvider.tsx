/**
 * PWA Provider - Progressive Web App functionality
 * Service Worker, Push Notifications, Offline Support
 */

import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import type { User, PlatformCapabilities, Theme, Notification } from '@modulr/core-types';

// ============================================================================
// PWA TYPES
// ============================================================================

interface PWAInstallPrompt {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

interface ServiceWorkerRegistration extends ServiceWorkerRegistration {
  waiting?: ServiceWorker;
  installing?: ServiceWorker;
}

interface PWAState {
  isInstallable: boolean;
  isInstalled: boolean;
  isOnline: boolean;
  serviceWorker: ServiceWorkerRegistration | null;
  installPrompt: PWAInstallPrompt | null;
  pushSubscription: PushSubscription | null;
  notifications: Notification[];
  capabilities: PlatformCapabilities;
  theme: Theme;
}

interface PWAContextValue extends PWAState {
  // Installation
  installApp: () => Promise<boolean>;
  checkInstallable: () => void;
  
  // Service Worker
  updateServiceWorker: () => Promise<void>;
  skipWaiting: () => void;
  
  // Push Notifications
  subscribeToPush: () => Promise<boolean>;
  unsubscribeFromPush: () => Promise<boolean>;
  requestNotificationPermission: () => Promise<boolean>;
  
  // Storage
  getStorageUsage: () => Promise<{ quota: number; usage: number }>;
  clearStorage: () => Promise<void>;
  
  // Theme
  setTheme: (theme: Theme) => void;
  
  // Network
  checkOnlineStatus: () => boolean;
}

// ============================================================================
// CONTEXT
// ============================================================================

const PWAContext = createContext<PWAContextValue | undefined>(undefined);

interface PWAProviderProps {
  children: ReactNode;
}

export function PWAProvider({ children }: PWAProviderProps) {
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
      hasNotifications: 'Notification' in window && 'PushManager' in window,
      hasGeolocation: 'geolocation' in navigator,
      hasCamera: 'mediaDevices' in navigator && 'getUserMedia' in navigator.mediaDevices,
      hasBiometrics: false,
      hasOfflineSupport: 'serviceWorker' in navigator && 'caches' in window,
      hasVoiceInput: 'webkitSpeechRecognition' in window || 'SpeechRecognition' in window,
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

  // ============================================================================
  // INITIALIZATION
  // ============================================================================

  useEffect(() => {
    initializePWA();
    setupEventListeners();
    
    return () => {
      cleanupEventListeners();
    };
  }, []);

  const initializePWA = async () => {
    // Register service worker
    await registerServiceWorker();
    
    // Check if app is installable
    checkInstallable();
    
    // Check if app is installed (running in standalone mode)
    checkInstalled();
    
    // Load theme
    loadTheme();
    
    // Check push subscription
    await checkPushSubscription();
    
    // Update capabilities
    updateCapabilities();
  };

  const setupEventListeners = () => {
    // Online/offline events
    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);
    
    // Install prompt event
    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
    
    // App installed event
    window.addEventListener('appinstalled', handleAppInstalled);
    
    // Theme change
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', handleThemeChange);
    
    // Screen resize
    window.addEventListener('resize', handleScreenResize);
  };

  const cleanupEventListeners = () => {
    window.removeEventListener('online', handleOnline);
    window.removeEventListener('offline', handleOffline);
    window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
    window.removeEventListener('appinstalled', handleAppInstalled);
    window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', handleThemeChange);
    window.removeEventListener('resize', handleScreenResize);
  };

  // ============================================================================
  // SERVICE WORKER
  // ============================================================================

  const registerServiceWorker = async () => {
    if (!('serviceWorker' in navigator)) {
      console.warn('Service Worker not supported');
      return;
    }

    try {
      const registration = await navigator.serviceWorker.register('/sw.js', {
        scope: '/',
      });

      setState(prev => ({ ...prev, serviceWorker: registration }));

      // Check for updates
      registration.addEventListener('updatefound', () => {
        const installingWorker = registration.installing;
        if (installingWorker) {
          installingWorker.addEventListener('statechange', () => {
            if (installingWorker.state === 'installed' && navigator.serviceWorker.controller) {
              // New version available
              console.log('New service worker available');
            }
          });
        }
      });

      console.log('Service Worker registered:', registration);
    } catch (error) {
      console.error('Service Worker registration failed:', error);
    }
  };

  const updateServiceWorker = async () => {
    if (!state.serviceWorker) return;

    try {
      await state.serviceWorker.update();
      console.log('Service Worker updated');
    } catch (error) {
      console.error('Service Worker update failed:', error);
    }
  };

  const skipWaiting = () => {
    if (state.serviceWorker?.waiting) {
      state.serviceWorker.waiting.postMessage({ type: 'SKIP_WAITING' });
    }
  };

  // ============================================================================
  // INSTALLATION
  // ============================================================================

  const handleBeforeInstallPrompt = (event: Event) => {
    event.preventDefault();
    const promptEvent = event as any;
    
    setState(prev => ({
      ...prev,
      isInstallable: true,
      installPrompt: promptEvent,
    }));
  };

  const handleAppInstalled = () => {
    setState(prev => ({
      ...prev,
      isInstalled: true,
      isInstallable: false,
      installPrompt: null,
    }));
    
    console.log('PWA installed successfully');
  };

  const checkInstallable = () => {
    // Check if running in standalone mode
    const isStandalone = window.matchMedia('(display-mode: standalone)').matches;
    
    setState(prev => ({
      ...prev,
      isInstalled: isStandalone,
    }));
  };

  const checkInstalled = () => {
    const isStandalone = 
      window.matchMedia('(display-mode: standalone)').matches ||
      ('standalone' in window.navigator && (window.navigator as any).standalone);
    
    setState(prev => ({
      ...prev,
      isInstalled: isStandalone,
    }));
  };

  const installApp = async (): Promise<boolean> => {
    if (!state.installPrompt) {
      return false;
    }

    try {
      await state.installPrompt.prompt();
      const choiceResult = await state.installPrompt.userChoice;
      
      if (choiceResult.outcome === 'accepted') {
        setState(prev => ({
          ...prev,
          isInstallable: false,
          installPrompt: null,
        }));
        
        console.log('PWA installation accepted');
        return true;
      }
      
      return false;
    } catch (error) {
      console.error('PWA installation failed:', error);
      return false;
    }
  };

  // ============================================================================
  // PUSH NOTIFICATIONS
  // ============================================================================

  const requestNotificationPermission = async (): Promise<boolean> => {
    if (!('Notification' in window)) {
      return false;
    }

    try {
      const permission = await Notification.requestPermission();
      return permission === 'granted';
    } catch (error) {
      console.error('Notification permission request failed:', error);
      return false;
    }
  };

  const subscribeToPush = async (): Promise<boolean> => {
    if (!('PushManager' in window) || !state.serviceWorker) {
      return false;
    }

    try {
      // Request notification permission first
      const hasPermission = await requestNotificationPermission();
      if (!hasPermission) {
        return false;
      }

      // Subscribe to push
      const subscription = await state.serviceWorker.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: process.env.VITE_VAPID_PUBLIC_KEY,
      });

      setState(prev => ({
        ...prev,
        pushSubscription: subscription,
      }));

      console.log('Push subscription successful:', subscription);
      return true;
    } catch (error) {
      console.error('Push subscription failed:', error);
      return false;
    }
  };

  const unsubscribeFromPush = async (): Promise<boolean> => {
    if (!state.pushSubscription) {
      return true;
    }

    try {
      await state.pushSubscription.unsubscribe();
      
      setState(prev => ({
        ...prev,
        pushSubscription: null,
      }));

      console.log('Push unsubscription successful');
      return true;
    } catch (error) {
      console.error('Push unsubscription failed:', error);
      return false;
    }
  };

  const checkPushSubscription = async () => {
    if (!('PushManager' in window) || !state.serviceWorker) {
      return;
    }

    try {
      const subscription = await state.serviceWorker.pushManager.getSubscription();
      setState(prev => ({
        ...prev,
        pushSubscription: subscription,
      }));
    } catch (error) {
      console.error('Push subscription check failed:', error);
    }
  };

  // ============================================================================
  // STORAGE
  // ============================================================================

  const getStorageUsage = async (): Promise<{ quota: number; usage: number }> => {
    if ('storage' in navigator && 'estimate' in navigator.storage) {
      try {
        const estimate = await navigator.storage.estimate();
        return {
          quota: estimate.quota || 0,
          usage: estimate.usage || 0,
        };
      } catch (error) {
        console.error('Storage estimate failed:', error);
      }
    }
    
    return { quota: 0, usage: 0 };
  };

  const clearStorage = async (): Promise<void> => {
    try {
      // Clear caches
      if ('caches' in window) {
        const cacheNames = await caches.keys();
        await Promise.all(cacheNames.map(name => caches.delete(name)));
      }
      
      // Clear localStorage
      localStorage.clear();
      
      // Clear sessionStorage
      sessionStorage.clear();
      
      console.log('Storage cleared successfully');
    } catch (error) {
      console.error('Storage clear failed:', error);
    }
  };

  // ============================================================================
  // THEME
  // ============================================================================

  const loadTheme = () => {
    const savedTheme = localStorage.getItem('modulr-theme') as Theme;
    const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    const theme = savedTheme || systemTheme;
    
    setState(prev => ({ ...prev, theme }));
    applyTheme(theme);
  };

  const setTheme = (theme: Theme) => {
    setState(prev => ({ ...prev, theme }));
    localStorage.setItem('modulr-theme', theme);
    applyTheme(theme);
  };

  const applyTheme = (theme: Theme) => {
    if (theme === 'dark') {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  };

  const handleThemeChange = (e: MediaQueryListEvent) => {
    const systemTheme = e.matches ? 'dark' : 'light';
    const savedTheme = localStorage.getItem('modulr-theme');
    
    if (!savedTheme || savedTheme === 'auto') {
      setTheme(systemTheme);
    }
  };

  // ============================================================================
  // NETWORK
  // ============================================================================

  const handleOnline = () => {
    setState(prev => ({ ...prev, isOnline: true }));
    console.log('App is online');
  };

  const handleOffline = () => {
    setState(prev => ({ ...prev, isOnline: false }));
    console.log('App is offline');
  };

  const checkOnlineStatus = (): boolean => {
    return navigator.onLine;
  };

  // ============================================================================
  // SCREEN
  // ============================================================================

  const handleScreenResize = () => {
    setState(prev => ({
      ...prev,
      capabilities: {
        ...prev.capabilities,
        screenInfo: {
          ...prev.capabilities.screenInfo,
          width: window.innerWidth,
          height: window.innerHeight,
          orientation: window.innerWidth > window.innerHeight ? 'landscape' : 'portrait',
          isSmall: window.innerWidth < 768,
        },
      },
    }));
  };

  const updateCapabilities = () => {
    setState(prev => ({
      ...prev,
      capabilities: {
        ...prev.capabilities,
        hasNotifications: 'Notification' in window && 'PushManager' in window,
        hasGeolocation: 'geolocation' in navigator,
        hasCamera: 'mediaDevices' in navigator && 'getUserMedia' in navigator.mediaDevices,
        hasOfflineSupport: 'serviceWorker' in navigator && 'caches' in window,
        hasVoiceInput: 'webkitSpeechRecognition' in window || 'SpeechRecognition' in window,
      },
    }));
  };

  // ============================================================================
  // CONTEXT VALUE
  // ============================================================================

  const contextValue: PWAContextValue = {
    ...state,
    installApp,
    checkInstallable,
    updateServiceWorker,
    skipWaiting,
    subscribeToPush,
    unsubscribeFromPush,
    requestNotificationPermission,
    getStorageUsage,
    clearStorage,
    setTheme,
    checkOnlineStatus,
  };

  return (
    <PWAContext.Provider value={contextValue}>
      {children}
    </PWAContext.Provider>
  );
}

// ============================================================================
// HOOK
// ============================================================================

export function usePWA(): PWAContextValue {
  const context = useContext(PWAContext);
  
  if (context === undefined) {
    throw new Error('usePWA must be used within a PWAProvider');
  }
  
  return context;
}

// ============================================================================
// SPECIALIZED HOOKS
// ============================================================================

/**
 * Hook for PWA installation
 */
export function usePWAInstall() {
  const { isInstallable, isInstalled, installApp, installPrompt } = usePWA();
  
  return {
    isInstallable,
    isInstalled,
    installApp,
    hasInstallPrompt: !!installPrompt,
  };
}

/**
 * Hook for network status
 */
export function useNetworkStatus() {
  const { isOnline, checkOnlineStatus } = usePWA();
  
  return {
    isOnline,
    checkOnlineStatus,
  };
}

/**
 * Hook for push notifications
 */
export function usePushNotifications() {
  const { 
    pushSubscription, 
    subscribeToPush, 
    unsubscribeFromPush, 
    requestNotificationPermission 
  } = usePWA();
  
  return {
    isSubscribed: !!pushSubscription,
    subscribeToPush,
    unsubscribeFromPush,
    requestPermission: requestNotificationPermission,
  };
}

/**
 * Hook for storage
 */
export function useStorage() {
  const { getStorageUsage, clearStorage } = usePWA();
  
  return {
    getUsage: getStorageUsage,
    clear: clearStorage,
  };
}

// ============================================================================
// EXPORTS
// ============================================================================

export { PWAContext };
export type { PWAInstallPrompt, ServiceWorkerRegistration };
