import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './index.css';

type RuntimePlatform = 'telegram' | 'web' | 'mobile' | 'desktop';

function detectPlatform(): RuntimePlatform {
  const runtime = window as unknown as {
    Telegram?: unknown;
    Capacitor?: unknown;
    __TAURI__?: unknown;
  };
  if (typeof window !== 'undefined' && runtime.Telegram) {
    return 'telegram';
  }
  if (typeof window !== 'undefined' && runtime.Capacitor) {
    return 'mobile';
  }
  if (typeof window !== 'undefined' && runtime.__TAURI__) {
    return 'desktop';
  }
  return 'web';
}

async function initializePlatform(): Promise<void> {
  const platform = detectPlatform();

  if (platform === 'web' && 'serviceWorker' in navigator) {
    try {
      await navigator.serviceWorker.register('/sw.js');
    } catch (error) {
      console.warn('service worker registration failed', error);
    }
  }

  window.addEventListener('error', (event) => {
    console.error('global error', event.error);
  });
  window.addEventListener('unhandledrejection', (event) => {
    console.error('unhandled rejection', event.reason);
  });
}

function renderApp(): void {
  const rootElement = document.getElementById('root');
  if (!rootElement) {
    throw new Error('root element not found');
  }
  ReactDOM.createRoot(rootElement).render(
    <React.StrictMode>
      <App />
    </React.StrictMode>
  );
}

async function bootstrap(): Promise<void> {
  try {
    await initializePlatform();
  } finally {
    renderApp();
  }
}

void bootstrap();

export { detectPlatform, initializePlatform };
