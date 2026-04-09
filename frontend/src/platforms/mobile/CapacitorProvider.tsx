/**
 * Capacitor Provider - Mobile platform functionality
 * Native API access for iOS and Android
 */

import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import type { 
  User, 
  PlatformCapabilities, 
  Theme, 
  Notification 
} from '@modulr/core-types';

// ============================================================================
// CAPACITOR TYPES
// ============================================================================

interface CapacitorState {
  isNative: boolean;
  platform: 'ios' | 'android' | 'web';
  version: string;
  capabilities: PlatformCapabilities;
  permissions: {
    camera: boolean;
    geolocation: boolean;
    notifications: boolean;
    microphone: boolean;
    photos: boolean;
    contacts: boolean;
  };
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
  // Permissions
  requestPermission: (permission: string) => Promise<boolean>;
  checkPermission: (permission: string) => Promise<boolean>;
  
  // Camera
  takePhoto: (options?: CameraOptions) => Promise<{ photo: string; data?: string }>;
  pickImage: (options?: ImagePickerOptions) => Promise<{ photos: string[] }>;
  
  // Geolocation
  getCurrentPosition: () => Promise<{ latitude: number; longitude: number }>;
  watchPosition: (callback: (position: { latitude: number; longitude: number }) => void) => Promise<string>;
  clearWatch: (watchId: string) => void;
  
  // Biometrics
  authenticate: (options?: BiometricOptions) => Promise<boolean>;
  checkBiometricAvailability: () => Promise<boolean>;
  
  // Notifications
  scheduleNotification: (notification: NotificationSchedule) => Promise<string>;
  cancelNotification: (id: string) => Promise<void>;
  getPendingNotifications: () => Promise<NotificationSchedule[]>;
  
  // Storage
  getStorageInfo: () => Promise<{ quota: number; usage: number }>;
  clearStorage: () => Promise<void>;
  
  // App
  exitApp: () => Promise<void>;
  minimizeApp: () => Promise<void>;
  getLaunchUrl: () => Promise<string>;
  
  // Network
  checkNetworkStatus: () => Promise<{ connected: boolean; connectionType: string }>;
  
  // Theme
  setTheme: (theme: Theme) => void;
}

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

interface NotificationAttachment {
  id: string;
  url: string;
}

// ============================================================================
// CONTEXT
// ============================================================================

const CapacitorContext = createContext<CapacitorContextValue | undefined>(undefined);

interface CapacitorProviderProps {
  children: ReactNode;
}

export function CapacitorProvider({ children }: CapacitorProviderProps) {
  const [state, setState] = useState<CapacitorState>({
    isNative: false,
    platform: 'web',
    version: '1.0.0',
    capabilities: {
      platform: 'mobile',
      hasNotifications: false,
      hasGeolocation: false,
      hasCamera: false,
      hasBiometrics: false,
      hasOfflineSupport: false,
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
    permissions: {
      camera: false,
      geolocation: false,
      notifications: false,
      microphone: false,
      photos: false,
      contacts: false,
    },
    biometricAvailable: false,
    networkStatus: {
      connected: navigator.onLine,
      connectionType: 'unknown',
    },
    appInfo: {
      version: '1.0.0',
      build: '1',
      id: 'com.modulr.app',
    },
    theme: 'light',
  });

  // ============================================================================
  // INITIALIZATION
  // ============================================================================

  useEffect(() => {
    initializeCapacitor();
    setupEventListeners();
    
    return () => {
      cleanupEventListeners();
    };
  }, []);

  const initializeCapacitor = async () => {
    // Check if running on native platform
    const isNative = await checkIsNative();
    
    if (isNative) {
      await initializeNativeFeatures();
    }
    
    // Load theme
    loadTheme();
    
    // Update capabilities
    updateCapabilities();
  };

  const checkIsNative = async (): Promise<boolean> => {
    try {
      // Dynamic import of Capacitor
      const { Capacitor } = await import('@capacitor/core');
      const isNative = Capacitor.isNativePlatform;
      
      setState(prev => ({
        ...prev,
        isNative,
        platform: Capacitor.getPlatform() as 'ios' | 'android' | 'web',
      }));
      
      return isNative;
    } catch {
      return false;
    }
  };

  const initializeNativeFeatures = async () => {
    try {
      // Import Capacitor plugins
      const [
        { Device },
        { Network },
        { App },
        { Preferences },
        { Camera },
        { Geolocation },
        { PushNotifications },
        { Haptics },
        { StatusBar },
        { SplashScreen },
      ] = await Promise.all([
        import('@capacitor/device'),
        import('@capacitor/network'),
        import('@capacitor/app'),
        import('@capacitor/preferences'),
        import('@capacitor/camera'),
        import('@capacitor/geolocation'),
        import('@capacitor/push-notifications'),
        import('@capacitor/haptics'),
        import('@capacitor/status-bar'),
        import('@capacitor/splash-screen'),
      ]);

      // Get device info
      const deviceInfo = await Device.getInfo();
      
      // Get network status
      const networkStatus = await Network.getStatus();
      
      // Get app info
      const appInfo = await App.getInfo();
      
      // Check permissions
      const permissions = await checkAllPermissions();
      
      // Check biometric availability
      const biometricAvailable = await checkBiometricAvailability();
      
      setState(prev => ({
        ...prev,
        version: deviceInfo.platformVersion,
        networkStatus: {
          connected: networkStatus.connected,
          connectionType: networkStatus.connectionType,
        },
        appInfo: {
          version: appInfo.version,
          build: appInfo.build,
          id: appInfo.id,
        },
        permissions,
        biometricAvailable,
      }));

      // Hide splash screen
      await SplashScreen.hide();

      // Set status bar style
      await StatusBar.setStyle({ style: 'LIGHT' });

    } catch (error) {
      console.error('Failed to initialize native features:', error);
    }
  };

  const setupEventListeners = () => {
    // Network status listener
    if (state.isNative) {
      setupNetworkListener();
    }
    
    // App state listener
    setupAppStateListener();
  };

  const setupNetworkListener = async () => {
    try {
      const { Network } = await import('@capacitor/network');
      
      Network.addListener('networkStatusChange', (status) => {
        setState(prev => ({
          ...prev,
          networkStatus: {
            connected: status.connected,
            connectionType: status.connectionType,
          },
        }));
      });
    } catch (error) {
      console.error('Failed to setup network listener:', error);
    }
  };

  const setupAppStateListener = async () => {
    try {
      const { App } = await import('@capacitor/app');
      
      App.addListener('appStateChange', ({ isActive }) => {
        if (isActive) {
          // App became active
          console.log('App became active');
        } else {
          // App became inactive
          console.log('App became inactive');
        }
      });
    } catch (error) {
      console.error('Failed to setup app state listener:', error);
    }
  };

  const cleanupEventListeners = () => {
    // Cleanup will be handled by Capacitor automatically
  };

  // ============================================================================
  // PERMISSIONS
  // ============================================================================

  const checkAllPermissions = async () => {
    const permissions = {
      camera: await checkPermission('camera'),
      geolocation: await checkPermission('geolocation'),
      notifications: await checkPermission('notifications'),
      microphone: await checkPermission('microphone'),
      photos: await checkPermission('photos'),
      contacts: await checkPermission('contacts'),
    };
    
    return permissions;
  };

  const checkPermission = async (permission: string): Promise<boolean> => {
    if (!state.isNative) return false;
    
    try {
      const { Permissions } = await import('@capacitor/permissions');
      
      let permissionType;
      switch (permission) {
        case 'camera':
          permissionType = Permissions['camera'];
          break;
        case 'geolocation':
          permissionType = Permissions['location'];
          break;
        case 'notifications':
          permissionType = Permissions['notifications'];
          break;
        case 'microphone':
          permissionType = Permissions['microphone'];
          break;
        case 'photos':
          permissionType = Permissions['photos'];
          break;
        case 'contacts':
          permissionType = Permissions['contacts'];
          break;
        default:
          return false;
      }
      
      const result = await Permissions.check({ permissions: [permissionType] });
      return result[permissionType] === 'granted';
    } catch (error) {
      console.error(`Failed to check ${permission} permission:`, error);
      return false;
    }
  };

  const requestPermission = async (permission: string): Promise<boolean> => {
    if (!state.isNative) return false;
    
    try {
      const { Permissions } = await import('@capacitor/permissions');
      
      let permissionType;
      switch (permission) {
        case 'camera':
          permissionType = Permissions['camera'];
          break;
        case 'geolocation':
          permissionType = Permissions['location'];
          break;
        case 'notifications':
          permissionType = Permissions['notifications'];
          break;
        case 'microphone':
          permissionType = Permissions['microphone'];
          break;
        case 'photos':
          permissionType = Permissions['photos'];
          break;
        case 'contacts':
          permissionType = Permissions['contacts'];
          break;
        default:
          return false;
      }
      
      const result = await Permissions.request({ permissions: [permissionType] });
      const granted = result[permissionType] === 'granted';
      
      // Update permissions state
      setState(prev => ({
        ...prev,
        permissions: {
          ...prev.permissions,
          [permission]: granted,
        },
      }));
      
      return granted;
    } catch (error) {
      console.error(`Failed to request ${permission} permission:`, error);
      return false;
    }
  };

  // ============================================================================
  // CAMERA
  // ============================================================================

  const takePhoto = async (options?: CameraOptions): Promise<{ photo: string; data?: string }> => {
    if (!state.isNative) {
      throw new Error('Camera is only available on native platforms');
    }
    
    try {
      const { Camera } = await import('@capacitor/camera');
      
      const result = await Camera.getPhoto({
        quality: options?.quality || 90,
        allowEditing: options?.allowEditing || false,
        resultType: options?.resultType === 'base64' ? 'base64' : 'uri',
        source: options?.source === 'photos' ? 'PHOTOS' : 'CAMERA',
      });
      
      return {
        photo: result.webPath || result.base64String || '',
        data: result.base64String,
      };
    } catch (error) {
      console.error('Failed to take photo:', error);
      throw error;
    }
  };

  const pickImage = async (options?: ImagePickerOptions): Promise<{ photos: string[] }> => {
    if (!state.isNative) {
      throw new Error('Image picker is only available on native platforms');
    }
    
    try {
      const { Camera } = await import('@capacitor/camera');
      
      const result = await Camera.pickImages({
        multiple: options?.multiple || false,
        quality: options?.quality || 90,
        limit: options?.maxFiles || 1,
      });
      
      return {
        photos: result.photos.map(photo => photo.webPath || ''),
      };
    } catch (error) {
      console.error('Failed to pick images:', error);
      throw error;
    }
  };

  // ============================================================================
  // GEOLOCATION
  // ============================================================================

  const getCurrentPosition = async (): Promise<{ latitude: number; longitude: number }> => {
    if (!state.isNative) {
      // Fallback to browser geolocation
      return new Promise((resolve, reject) => {
        navigator.geolocation.getCurrentPosition(
          (position) => resolve({
            latitude: position.coords.latitude,
            longitude: position.coords.longitude,
          }),
          reject
        );
      });
    }
    
    try {
      const { Geolocation } = await import('@capacitor/geolocation');
      
      const position = await Geolocation.getCurrentPosition({
        enableHighAccuracy: true,
        timeout: 10000,
      });
      
      return {
        latitude: position.coords.latitude,
        longitude: position.coords.longitude,
      };
    } catch (error) {
      console.error('Failed to get current position:', error);
      throw error;
    }
  };

  const watchPosition = async (callback: (position: { latitude: number; longitude: number }) => void): Promise<string> => {
    if (!state.isNative) {
      // Fallback to browser geolocation
      const watchId = navigator.geolocation.watchPosition(
        (position) => callback({
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
        })
      );
      return watchId.toString();
    }
    
    try {
      const { Geolocation } = await import('@capacitor/geolocation');
      
      const watchId = await Geolocation.watchPosition({
        enableHighAccuracy: true,
        timeout: 10000,
      }, (position, err) => {
        if (err) {
          console.error('Geolocation watch error:', err);
          return;
        }
        
        callback({
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
        });
      });
      
      return watchId;
    } catch (error) {
      console.error('Failed to watch position:', error);
      throw error;
    }
  };

  const clearWatch = (watchId: string) => {
    if (!state.isNative) {
      navigator.geolocation.clearWatch(parseInt(watchId));
      return;
    }
    
    try {
      const { Geolocation } = require('@capacitor/geolocation');
      Geolocation.clearWatch({ id: watchId });
    } catch (error) {
      console.error('Failed to clear position watch:', error);
    }
  };

  // ============================================================================
  // BIOMETRICS
  ============================================================================

  const authenticate = async (options?: BiometricOptions): Promise<boolean> => {
    if (!state.isNative || !state.biometricAvailable) {
      return false;
    }
    
    try {
      const { BiometricAuth } = await import('@capacitor/biometric-auth');
      
      const result = await BiometricAuth.authenticate({
        reason: options?.reason || 'Authenticate to access Modulr',
        title: options?.title || 'Authentication',
        subtitle: options?.subtitle || '',
        description: options?.description || '',
      });
      
      return result.success;
    } catch (error) {
      console.error('Biometric authentication failed:', error);
      return false;
    }
  };

  const checkBiometricAvailability = async (): Promise<boolean> => {
    if (!state.isNative) return false;
    
    try {
      const { BiometricAuth } = await import('@capacitor/biometric-auth');
      
      const result = await BiometricAuth.isAvailable();
      return result.isAvailable;
    } catch (error) {
      console.error('Failed to check biometric availability:', error);
      return false;
    }
  };

  // ============================================================================
  // NOTIFICATIONS
  ============================================================================

  const scheduleNotification = async (notification: NotificationSchedule): Promise<string> => {
    if (!state.isNative) {
      // Fallback to browser notification
      if ('Notification' in window && Notification.permission === 'granted') {
        const browserNotification = new Notification(notification.title, {
          body: notification.body,
          icon: '/icon-192x192.png',
        });
        
        setTimeout(() => browserNotification.close(), 5000);
        return Date.now().toString();
      }
      
      throw new Error('Notifications not available');
    }
    
    try {
      const { LocalNotifications } = await import('@capacitor/local-notifications');
      
      const notificationData = {
        id: notification.id || Date.now().toString(),
        title: notification.title,
        body: notification.body,
        schedule: notification.schedule ? {
          at: notification.schedule.at,
          repeats: notification.schedule.repeats,
          every: notification.schedule.every,
        } : undefined,
        sound: notification.sound || 'default',
        attachments: notification.attachments || [],
      };
      
      await LocalNotifications.schedule({
        notifications: [notificationData],
      });
      
      return notificationData.id;
    } catch (error) {
      console.error('Failed to schedule notification:', error);
      throw error;
    }
  };

  const cancelNotification = async (id: string) => {
    if (!state.isNative) return;
    
    try {
      const { LocalNotifications } = await import('@capacitor/local-notifications');
      
      await LocalNotifications.cancel({
        notifications: [{ id }],
      });
    } catch (error) {
      console.error('Failed to cancel notification:', error);
    }
  };

  const getPendingNotifications = async (): Promise<NotificationSchedule[]> => {
    if (!state.isNative) return [];
    
    try {
      const { LocalNotifications } = await import('@capacitor/local-notifications');
      
      const result = await LocalNotifications.getPending();
      return result.notifications.map(notification => ({
        id: notification.id,
        title: notification.title,
        body: notification.body,
        schedule: notification.schedule,
        sound: notification.sound,
        attachments: notification.attachments || [],
      }));
    } catch (error) {
      console.error('Failed to get pending notifications:', error);
      return [];
    }
  };

  // ============================================================================
  // STORAGE
  ============================================================================

  const getStorageInfo = async (): Promise<{ quota: number; usage: number }> => {
    if (!state.isNative) {
      // Fallback to browser storage estimate
      if ('storage' in navigator && 'estimate' in navigator.storage) {
        const estimate = await navigator.storage.estimate();
        return {
          quota: estimate.quota || 0,
          usage: estimate.usage || 0,
        };
      }
      
      return { quota: 0, usage: 0 };
    }
    
    try {
      // Capacitor doesn't provide storage info, use browser fallback
      if ('storage' in navigator && 'estimate' in navigator.storage) {
        const estimate = await navigator.storage.estimate();
        return {
          quota: estimate.quota || 0,
          usage: estimate.usage || 0,
        };
      }
      
      return { quota: 0, usage: 0 };
    } catch (error) {
      console.error('Failed to get storage info:', error);
      return { quota: 0, usage: 0 };
    }
  };

  const clearStorage = async (): Promise<void> => {
    if (!state.isNative) {
      // Fallback to browser storage
      localStorage.clear();
      sessionStorage.clear();
      
      if ('caches' in window) {
        const cacheNames = await caches.keys();
        await Promise.all(cacheNames.map(name => caches.delete(name)));
      }
      
      return;
    }
    
    try {
      const { Preferences } = await import('@capacitor/preferences');
      await Preferences.clear();
      
      // Also clear browser storage for web compatibility
      localStorage.clear();
      sessionStorage.clear();
    } catch (error) {
      console.error('Failed to clear storage:', error);
    }
  };

  // ============================================================================
  // APP
  ============================================================================

  const exitApp = async () => {
    if (!state.isNative) return;
    
    try {
      const { App } = await import('@capacitor/app');
      await App.exitApp();
    } catch (error) {
      console.error('Failed to exit app:', error);
    }
  };

  const minimizeApp = async () => {
    if (!state.isNative) return;
    
    try {
      const { App } = await import('@capacitor/app');
      await App.minimizeApp();
    } catch (error) {
      console.error('Failed to minimize app:', error);
    }
  };

  const getLaunchUrl = async (): Promise<string> => {
    if (!state.isNative) return window.location.href;
    
    try {
      const { App } = await import('@capacitor/app');
      const launchUrl = await App.getLaunchUrl();
      return launchUrl?.url || window.location.href;
    } catch (error) {
      console.error('Failed to get launch URL:', error);
      return window.location.href;
    }
  };

  // ============================================================================
  // NETWORK
  ============================================================================

  const checkNetworkStatus = async (): Promise<{ connected: boolean; connectionType: string }> => {
    if (!state.isNative) {
      return {
        connected: navigator.onLine,
        connectionType: 'unknown',
      };
    }
    
    try {
      const { Network } = await import('@capacitor/network');
      const status = await Network.getStatus();
      
      setState(prev => ({
        ...prev,
        networkStatus: {
          connected: status.connected,
          connectionType: status.connectionType,
        },
      }));
      
      return {
        connected: status.connected,
        connectionType: status.connectionType,
      };
    } catch (error) {
      console.error('Failed to check network status:', error);
      return {
        connected: navigator.onLine,
        connectionType: 'unknown',
      };
    }
  };

  // ============================================================================
  // THEME
  ============================================================================

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

  const updateCapabilities = () => {
    setState(prev => ({
      ...prev,
      capabilities: {
        ...prev.capabilities,
        hasNotifications: state.isNative || ('Notification' in window && 'PushManager' in window),
        hasGeolocation: state.isNative || 'geolocation' in navigator,
        hasCamera: state.isNative || ('mediaDevices' in navigator && 'getUserMedia' in navigator.mediaDevices),
        hasBiometrics: state.isNative && state.biometricAvailable,
        hasOfflineSupport: state.isNative || ('serviceWorker' in navigator && 'caches' in window),
        hasVoiceInput: state.isNative || 'webkitSpeechRecognition' in window || 'SpeechRecognition' in window,
      },
    }));
  };

  // ============================================================================
  // CONTEXT VALUE
  ============================================================================

  const contextValue: CapacitorContextValue = {
    ...state,
    requestPermission,
    checkPermission,
    takePhoto,
    pickImage,
    getCurrentPosition,
    watchPosition,
    clearWatch,
    authenticate,
    checkBiometricAvailability,
    scheduleNotification,
    cancelNotification,
    getPendingNotifications,
    getStorageInfo,
    clearStorage,
    exitApp,
    minimizeApp,
    getLaunchUrl,
    checkNetworkStatus,
    setTheme,
  };

  return (
    <CapacitorContext.Provider value={contextValue}>
      {children}
    </CapacitorContext.Provider>
  );
}

// ============================================================================
// HOOK
// ============================================================================

export function useCapacitor(): CapacitorContextValue {
  const context = useContext(CapacitorContext);
  
  if (context === undefined) {
    throw new Error('useCapacitor must be used within a CapacitorProvider');
  }
  
  return context;
}

// ============================================================================
// SPECIALIZED HOOKS
// ============================================================================

/**
 * Hook for camera functionality
 */
export function useCamera() {
  const { permissions, requestPermission, takePhoto, pickImage } = useCapacitor();
  
  return {
    hasPermission: permissions.camera,
    requestPermission: () => requestPermission('camera'),
    takePhoto,
    pickImage,
  };
}

/**
 * Hook for geolocation
 */
export function useGeolocation() {
  const { permissions, requestPermission, getCurrentPosition, watchPosition, clearWatch } = useCapacitor();
  
  return {
    hasPermission: permissions.geolocation,
    requestPermission: () => requestPermission('geolocation'),
    getCurrentPosition,
    watchPosition,
    clearWatch,
  };
}

/**
 * Hook for biometrics
 */
export function useBiometrics() {
  const { biometricAvailable, authenticate, checkBiometricAvailability } = useCapacitor();
  
  return {
    isAvailable: biometricAvailable,
    authenticate,
    checkAvailability: checkBiometricAvailability,
  };
}

/**
 * Hook for network status
 */
export function useNetwork() {
  const { networkStatus, checkNetworkStatus } = useCapacitor();
  
  return {
    ...networkStatus,
    checkStatus: checkNetworkStatus,
  };
}

// ============================================================================
// EXPORTS
// ============================================================================

export { CapacitorContext };
export type { 
  CapacitorState, 
  CameraOptions, 
  ImagePickerOptions, 
  BiometricOptions, 
  NotificationSchedule,
  NotificationAttachment 
};
