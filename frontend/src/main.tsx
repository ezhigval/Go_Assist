/**
 * Main entry point for the application
 * Sets up React root and initializes platform-specific features
 */

import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';

// ============================================================================
// PLATFORM INITIALIZATION
// ============================================================================

const initializePlatform = async () => {
  // Initialize platform-specific features
  if (typeof window !== 'undefined') {
    // Set up global error handling
    window.addEventListener('error', (event) => {
      console.error('Global error:', event.error);
    });

    window.addEventListener('unhandledrejection', (event) => {
      console.error('Unhandled promise rejection:', event.reason);
    });

    // Initialize platform-specific features
    const platform = getPlatform();
    
    switch (platform) {
      case 'telegram':
        await initializeTelegram();
        break;
      case 'mobile':
        await initializeMobile();
        break;
      case 'desktop':
        await initializeDesktop();
        break;
      case 'web':
      default:
        await initializeWeb();
        break;
    }
  }
};

const getPlatform = (): 'telegram' | 'web' | 'mobile' | 'desktop' => {
  if ((window as any).Telegram?.WebApp) {
    return 'telegram';
  }
  
  if ((window as any).Capacitor) {
    return 'mobile';
  }
  
  if ((window as any).__TAURI__) {
    return 'desktop';
  }
  
  return 'web';
};

const initializeTelegram = async () => {
  const webApp = (window as any).Telegram?.WebApp;
  if (webApp) {
    webApp.ready();
    webApp.expand();
    
    // Set theme colors
    const colorScheme = webApp.colorScheme;
    if (colorScheme === 'dark') {
      document.documentElement.classList.add('dark');
    }
    
    // Set up back button if needed
    if (webApp.BackButton) {
      webApp.BackButton.hide();
    }
    
    console.log('Telegram WebApp initialized');
  }
};

const initializeMobile = async () => {
  // Initialize Capacitor-specific features
  try {
    const { Capacitor } = await import('@capacitor/core');
    console.log(`Capacitor initialized on ${Capacitor.getPlatform()}`);
  } catch (error) {
    console.error('Failed to initialize Capacitor:', error);
  }
};

const initializeDesktop = async () => {
  // Initialize Tauri-specific features
  try {
    const { invoke } = await import('@tauri-apps/api/tauri');
    console.log('Tauri initialized');
  } catch (error) {
    console.error('Failed to initialize Tauri:', error);
  }
};

const initializeWeb = async () => {
  // Initialize web-specific features
  if ('serviceWorker' in navigator) {
    try {
      const registration = await navigator.serviceWorker.register('/sw.js');
      console.log('Service Worker registered:', registration);
    } catch (error) {
      console.error('Service Worker registration failed:', error);
    }
  }
  
  // Check for PWA installation prompt
  window.addEventListener('beforeinstallprompt', (event) => {
    console.log('PWA installation prompt available');
    (window as any).__PWA_INSTALL_PROMPT = event;
  });
  
  console.log('Web platform initialized');
};

// ============================================================================
// RENDER APPLICATION
// ============================================================================

const renderApp = () => {
  const rootElement = document.getElementById('root');
  
  if (!rootElement) {
    throw new Error('Root element not found');
  }
  
  const root = ReactDOM.createRoot(rootElement);
  
  root.render(
    <React.StrictMode>
      <App />
    </React.StrictMode>
  );
};

// ============================================================================
// STARTUP
// ============================================================================

const startApp = async () => {
  try {
    await initializePlatform();
    renderApp();
    console.log('Modulr app started successfully');
  } catch (error) {
    console.error('Failed to start app:', error);
    
    // Fallback render
    renderApp();
  }
};

// Start the application
startApp();

// ============================================================================
// EXPORTS FOR TESTING
// ============================================================================

export { initializePlatform, getPlatform };
