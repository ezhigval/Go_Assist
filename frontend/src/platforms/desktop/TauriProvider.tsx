/**
 * Tauri Provider - Desktop platform functionality
 * Native desktop integration for Windows, macOS, Linux
 */

import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import type { 
  User, 
  PlatformCapabilities, 
  Theme, 
  Notification 
} from '@modulr/core-types';

// ============================================================================
// TAURI TYPES
// ============================================================================

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
  // Window management
  minimizeWindow: () => Promise<void>;
  maximizeWindow: () => Promise<void>;
  unmaximizeWindow: () => Promise<void>;
  closeWindow: () => Promise<void>;
  hideWindow: () => Promise<void>;
  showWindow: () => Promise<void>;
  setFullscreen: (fullscreen: boolean) => Promise<void>;
  centerWindow: () => Promise<void>;
  
  // System integration
  showNotification: (title: string, body: string) => Promise<void>;
  openExternal: (url: string) => Promise<void>;
  openFile: (path: string) => Promise<void>;
  showInFolder: (path: string) => Promise<void>;
  
  // File system
  readFile: (path: string) => Promise<string>;
  writeFile: (path: string, content: string) => Promise<void>;
  exists: (path: string) => Promise<boolean>;
  mkdir: (path: string) => Promise<void>;
  
  // Clipboard
  readClipboard: () => Promise<string>;
  writeClipboard: (content: string) => Promise<void>;
  
  // Global shortcuts
  registerShortcut: (shortcut: string, action: () => void) => Promise<void>;
  unregisterShortcut: (shortcut: string) => Promise<void>;
  
  // System tray
  setTrayIcon: (icon: string) => Promise<void>;
  setTrayTooltip: (tooltip: string) => Promise<void>;
  
  // Auto-start
  enableAutoStart: () => Promise<void>;
  disableAutoStart: () => Promise<void>;
  isAutoStartEnabled: () => Promise<boolean>;
  
  // Theme
  setTheme: (theme: Theme) => void;
  
  // App lifecycle
  restartApp: () => Promise<void>;
  getLaunchArgs: () => Promise<string[]>;
}

// ============================================================================
// CONTEXT
// ============================================================================

const TauriContext = createContext<TauriContextValue | undefined>(undefined);

interface TauriProviderProps {
  children: ReactNode;
}

export function TauriProvider({ children }: TauriProviderProps) {
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

  // ============================================================================
  // INITIALIZATION
  // ============================================================================

  useEffect(() => {
    initializeTauri();
    setupEventListeners();
    
    return () => {
      cleanupEventListeners();
    };
  }, []);

  const initializeTauri = async () => {
    // Check if running on desktop platform
    const isDesktop = await checkIsDesktop();
    
    if (isDesktop) {
      await initializeDesktopFeatures();
    }
    
    // Load theme
    loadTheme();
    
    // Update capabilities
    updateCapabilities();
  };

  const checkIsDesktop = async (): Promise<boolean> => {
    try {
      // Dynamic import of Tauri
      const { invoke } = await import('@tauri-apps/api/tauri');
      
      try {
        // Try to call a simple Tauri command
        await invoke('get_platform_info');
        return true;
      } catch {
        return false;
      }
    } catch {
      return false;
    }
  };

  const initializeDesktopFeatures = async () => {
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      const { listen } = await import('@tauri-apps/api/event');
      const { appWindow } = await import('@tauri-apps/api/window');
      const { platform } = await import('@tauri-apps/api/os');
      
      // Get system info
      const systemInfo = await invoke('get_system_info') as TauriState['systemInfo'];
      
      // Get app info
      const appInfo = await invoke('get_app_info') as TauriState['appInfo'];
      
      // Get platform info
      const platformInfo = await platform();
      
      // Get window info
      const windowInfo = {
        isFocused: await appWindow.isFocused(),
        isVisible: await appWindow.isVisible(),
        isFullscreen: await appWindow.isFullscreen(),
        isMaximized: await appWindow.isMaximized(),
        width: await appWindow.innerSize().then(s => s.width),
        height: await appWindow.innerSize().then(s => s.height),
      };
      
      setState(prev => ({
        ...prev,
        isDesktop: true,
        platform: platformInfo as 'windows' | 'macos' | 'linux',
        arch: platformInfo.arch as 'x86_64' | 'aarch64',
        systemInfo,
        appInfo,
        windowInfo,
      }));
      
      // Setup window event listeners
      setupWindowListeners();
      
    } catch (error) {
      console.error('Failed to initialize desktop features:', error);
    }
  };

  const setupWindowListeners = async () => {
    try {
      const { appWindow } = await import('@tauri-apps/api/window');
      const { listen } = await import('@tauri-apps/api/event');
      
      // Window focus events
      await listen('tauri://focus', () => {
        setState(prev => ({
          ...prev,
          windowInfo: { ...prev.windowInfo, isFocused: true },
        }));
      });
      
      await listen('tauri://blur', () => {
        setState(prev => ({
          ...prev,
          windowInfo: { ...prev.windowInfo, isFocused: false },
        }));
      });
      
      // Window resize events
      await listen('tauri://resize', () => {
        appWindow.innerSize().then(size => {
          setState(prev => ({
            ...prev,
            windowInfo: { ...prev.windowInfo, width: size.width, height: size.height },
          }));
        });
      });
      
      // Window maximize events
      await listen('tauri://maximized', () => {
        setState(prev => ({
          ...prev,
          windowInfo: { ...prev.windowInfo, isMaximized: true },
        }));
      });
      
      await listen('tauri://unmaximized', () => {
        setState(prev => ({
          ...prev,
          windowInfo: { ...prev.windowInfo, isMaximized: false },
        }));
      });
      
      // Fullscreen events
      await listen('tauri://fullscreen', () => {
        setState(prev => ({
          ...prev,
          windowInfo: { ...prev.windowInfo, isFullscreen: true },
        }));
      });
      
      await listen('tauri://unfullscreen', () => {
        setState(prev => ({
          ...prev,
          windowInfo: { ...prev.windowInfo, isFullscreen: false },
        }));
      });
      
    } catch (error) {
      console.error('Failed to setup window listeners:', error);
    }
  };

  const setupEventListeners = () => {
    // Window resize listener
    window.addEventListener('resize', handleWindowResize);
    
    // Theme change listener
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', handleThemeChange);
  };

  const cleanupEventListeners = () => {
    window.removeEventListener('resize', handleWindowResize);
    window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', handleThemeChange);
  };

  // ============================================================================
  // WINDOW MANAGEMENT
  // ============================================================================

  const minimizeWindow = async () => {
    if (!state.isDesktop) return;
    
    try {
      const { appWindow } = await import('@tauri-apps/api/window');
      await appWindow.minimize();
    } catch (error) {
      console.error('Failed to minimize window:', error);
    }
  };

  const maximizeWindow = async () => {
    if (!state.isDesktop) return;
    
    try {
      const { appWindow } = await import('@tauri-apps/api/window');
      await appWindow.maximize();
    } catch (error) {
      console.error('Failed to maximize window:', error);
    }
  };

  const unmaximizeWindow = async () => {
    if (!state.isDesktop) return;
    
    try {
      const { appWindow } = await import('@tauri-apps/api/window');
      await appWindow.unmaximize();
    } catch (error) {
      console.error('Failed to unmaximize window:', error);
    }
  };

  const closeWindow = async () => {
    if (!state.isDesktop) return;
    
    try {
      const { appWindow } = await import('@tauri-apps/api/window');
      await appWindow.close();
    } catch (error) {
      console.error('Failed to close window:', error);
    }
  };

  const hideWindow = async () => {
    if (!state.isDesktop) return;
    
    try {
      const { appWindow } = await import('@tauri-apps/api/window');
      await appWindow.hide();
    } catch (error) {
      console.error('Failed to hide window:', error);
    }
  };

  const showWindow = async () => {
    if (!state.isDesktop) return;
    
    try {
      const { appWindow } = await import('@tauri-apps/api/window');
      await appWindow.show();
      await appWindow.setFocus();
    } catch (error) {
      console.error('Failed to show window:', error);
    }
  };

  const setFullscreen = async (fullscreen: boolean) => {
    if (!state.isDesktop) return;
    
    try {
      const { appWindow } = await import('@tauri-apps/api/window');
      if (fullscreen) {
        await appWindow.setFullscreen(true);
      } else {
        await appWindow.setFullscreen(false);
      }
    } catch (error) {
      console.error('Failed to set fullscreen:', error);
    }
  };

  const centerWindow = async () => {
    if (!state.isDesktop) return;
    
    try {
      const { appWindow } = await import('@tauri-apps/api/window');
      await appWindow.center();
    } catch (error) {
      console.error('Failed to center window:', error);
    }
  };

  // ============================================================================
  // SYSTEM INTEGRATION
  // ============================================================================

  const showNotification = async (title: string, body: string) => {
    if (!state.isDesktop) {
      // Fallback to browser notification
      if ('Notification' in window && Notification.permission === 'granted') {
        new Notification(title, { body });
      }
      return;
    }
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      await invoke('show_notification', { title, body });
    } catch (error) {
      console.error('Failed to show notification:', error);
    }
  };

  const openExternal = async (url: string) => {
    if (!state.isDesktop) {
      window.open(url, '_blank');
      return;
    }
    
    try {
      const { open } = await import('@tauri-apps/api/shell');
      await open(url);
    } catch (error) {
      console.error('Failed to open external URL:', error);
    }
  };

  const openFile = async (path: string) => {
    if (!state.isDesktop) return;
    
    try {
      const { open } = await import('@tauri-apps/api/shell');
      await open(path);
    } catch (error) {
      console.error('Failed to open file:', error);
    }
  };

  const showInFolder = async (path: string) => {
    if (!state.isDesktop) return;
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      await invoke('show_in_folder', { path });
    } catch (error) {
      console.error('Failed to show in folder:', error);
    }
  };

  // ============================================================================
  // FILE SYSTEM
  // ============================================================================

  const readFile = async (path: string): Promise<string> => {
    if (!state.isDesktop) {
      throw new Error('File system access is only available on desktop');
    }
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      return await invoke('read_file', { path });
    } catch (error) {
      console.error('Failed to read file:', error);
      throw error;
    }
  };

  const writeFile = async (path: string, content: string) => {
    if (!state.isDesktop) {
      throw new Error('File system access is only available on desktop');
    }
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      await invoke('write_file', { path, content });
    } catch (error) {
      console.error('Failed to write file:', error);
      throw error;
    }
  };

  const exists = async (path: string): Promise<boolean> => {
    if (!state.isDesktop) return false;
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      return await invoke('file_exists', { path });
    } catch (error) {
      console.error('Failed to check file existence:', error);
      return false;
    }
  };

  const mkdir = async (path: string) => {
    if (!state.isDesktop) return;
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      await invoke('create_directory', { path });
    } catch (error) {
      console.error('Failed to create directory:', error);
    }
  };

  // ============================================================================
  // CLIPBOARD
  // ============================================================================

  const readClipboard = async (): Promise<string> => {
    if (!state.isDesktop) {
      // Fallback to browser clipboard
      try {
        return await navigator.clipboard.readText();
      } catch {
        return '';
      }
    }
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      return await invoke('read_clipboard');
    } catch (error) {
      console.error('Failed to read clipboard:', error);
      return '';
    }
  };

  const writeClipboard = async (content: string) => {
    if (!state.isDesktop) {
      // Fallback to browser clipboard
      try {
        await navigator.clipboard.writeText(content);
      } catch (error) {
        console.error('Failed to write to browser clipboard:', error);
      }
      return;
    }
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      await invoke('write_clipboard', { content });
    } catch (error) {
      console.error('Failed to write to clipboard:', error);
    }
  };

  // ============================================================================
  // GLOBAL SHORTCUTS
  // ============================================================================

  const registerShortcut = async (shortcut: string, action: () => void) => {
    if (!state.isDesktop) return;
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      await invoke('register_shortcut', { shortcut });
      
      // Store the action locally
      const shortcuts = (window as any).__modulr_shortcuts || {};
      shortcuts[shortcut] = action;
      (window as any).__modulr_shortcuts = shortcuts;
      
    } catch (error) {
      console.error('Failed to register shortcut:', error);
    }
  };

  const unregisterShortcut = async (shortcut: string) => {
    if (!state.isDesktop) return;
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      await invoke('unregister_shortcut', { shortcut });
      
      // Remove the action locally
      const shortcuts = (window as any).__modulr_shortcuts || {};
      delete shortcuts[shortcut];
      (window as any).__modulr_shortcuts = shortcuts;
      
    } catch (error) {
      console.error('Failed to unregister shortcut:', error);
    }
  };

  // ============================================================================
  // SYSTEM TRAY
  // ============================================================================

  const setTrayIcon = async (icon: string) => {
    if (!state.isDesktop) return;
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      await invoke('set_tray_icon', { icon });
    } catch (error) {
      console.error('Failed to set tray icon:', error);
    }
  };

  const setTrayTooltip = async (tooltip: string) => {
    if (!state.isDesktop) return;
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      await invoke('set_tray_tooltip', { tooltip });
    } catch (error) {
      console.error('Failed to set tray tooltip:', error);
    }
  };

  // ============================================================================
  // AUTO-START
  // ============================================================================

  const enableAutoStart = async () => {
    if (!state.isDesktop) return;
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      await invoke('enable_auto_start');
    } catch (error) {
      console.error('Failed to enable auto-start:', error);
    }
  };

  const disableAutoStart = async () => {
    if (!state.isDesktop) return;
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      await invoke('disable_auto_start');
    } catch (error) {
      console.error('Failed to disable auto-start:', error);
    }
  };

  const isAutoStartEnabled = async (): Promise<boolean> => {
    if (!state.isDesktop) return false;
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      return await invoke('is_auto_start_enabled');
    } catch (error) {
      console.error('Failed to check auto-start status:', error);
      return false;
    }
  };

  // ============================================================================
  // APP LIFECYCLE
  // ============================================================================

  const restartApp = async () => {
    if (!state.isDesktop) {
      window.location.reload();
      return;
    }
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      await invoke('restart_app');
    } catch (error) {
      console.error('Failed to restart app:', error);
    }
  };

  const getLaunchArgs = async (): Promise<string[]> => {
    if (!state.isDesktop) return [];
    
    try {
      const { invoke } = await import('@tauri-apps/api/tauri');
      return await invoke('get_launch_args');
    } catch (error) {
      console.error('Failed to get launch args:', error);
      return [];
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

  const handleWindowResize = () => {
    setState(prev => ({
      ...prev,
      capabilities: {
        ...prev.capabilities,
        screenInfo: {
          ...prev.capabilities.screenInfo,
          width: window.innerWidth,
          height: window.innerHeight,
          orientation: window.innerWidth > window.innerHeight ? 'landscape' : 'portrait',
          isSmall: window.innerWidth < 1024,
        },
      },
    }));
  };

  const updateCapabilities = () => {
    setState(prev => ({
      ...prev,
      capabilities: {
        ...prev.capabilities,
        hasNotifications: state.isDesktop || ('Notification' in window && 'PushManager' in window),
        hasGeolocation: 'geolocation' in navigator,
        hasCamera: 'mediaDevices' in navigator && 'getUserMedia' in navigator.mediaDevices,
        hasBiometrics: false,
        hasOfflineSupport: true,
        hasVoiceInput: 'webkitSpeechRecognition' in window || 'SpeechRecognition' in window,
      },
    }));
  };

  // ============================================================================
  // CONTEXT VALUE
  // ============================================================================

  const contextValue: TauriContextValue = {
    ...state,
    minimizeWindow,
    maximizeWindow,
    unmaximizeWindow,
    closeWindow,
    hideWindow,
    showWindow,
    setFullscreen,
    centerWindow,
    showNotification,
    openExternal,
    openFile,
    showInFolder,
    readFile,
    writeFile,
    exists,
    mkdir,
    readClipboard,
    writeClipboard,
    registerShortcut,
    unregisterShortcut,
    setTrayIcon,
    setTrayTooltip,
    enableAutoStart,
    disableAutoStart,
    isAutoStartEnabled,
    setTheme,
    restartApp,
    getLaunchArgs,
  };

  return (
    <TauriContext.Provider value={contextValue}>
      {children}
    </TauriContext.Provider>
  );
}

// ============================================================================
// HOOK
// ============================================================================

export function useTauri(): TauriContextValue {
  const context = useContext(TauriContext);
  
  if (context === undefined) {
    throw new Error('useTauri must be used within a TauriProvider');
  }
  
  return context;
}

// ============================================================================
// SPECIALIZED HOOKS
// ============================================================================

/**
 * Hook for window management
 */
export function useWindow() {
  const { windowInfo, minimizeWindow, maximizeWindow, unmaximizeWindow, closeWindow, setFullscreen } = useTauri();
  
  return {
    ...windowInfo,
    minimize: minimizeWindow,
    maximize: maximizeWindow,
    unmaximize: unmaximizeWindow,
    close: closeWindow,
    setFullscreen,
  };
}

/**
 * Hook for file system
 */
export function useFileSystem() {
  const { readFile, writeFile, exists, mkdir } = useTauri();
  
  return {
    read: readFile,
    write: writeFile,
    exists,
    createDirectory: mkdir,
  };
}

/**
 * Hook for clipboard
 */
export function useClipboard() {
  const { readClipboard, writeClipboard } = useTauri();
  
  return {
    read: readClipboard,
    write: writeClipboard,
  };
}

/**
 * Hook for system integration
 */
export function useSystemIntegration() {
  const { showNotification, openExternal, openFile, showInFolder } = useTauri();
  
  return {
    showNotification,
    openExternal,
    openFile,
    showInFolder,
  };
}

// ============================================================================
// EXPORTS
// ============================================================================

export { TauriContext };
