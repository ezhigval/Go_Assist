import { type ReactNode } from 'react';
import { ScopeProvider } from './context/ScopeContext';
import { TelegramProvider } from './platforms/telegram/TelegramProvider';
import { PWAProvider } from './platforms/web/PWAProvider';
import { CapacitorProvider } from './platforms/mobile/CapacitorProvider';
import { TauriProvider } from './platforms/desktop/TauriProvider';
import { Header } from './components/layout/Header';
import { api } from './lib/api';
import { ControlPlaneDashboard } from './modules/control-plane/ControlPlaneDashboard';

const getPlatform = (): 'telegram' | 'web' | 'mobile' | 'desktop' => {
  if (typeof window !== 'undefined' && (window as any).Telegram?.WebApp) {
    return 'telegram';
  }
  if (typeof window !== 'undefined' && (window as any).Capacitor) {
    return 'mobile';
  }
  if (typeof window !== 'undefined' && (window as any).__TAURI__) {
    return 'desktop';
  }
  return 'web';
};

function App() {
  const platform = getPlatform();

  const renderPlatformProviders = (children: ReactNode) => {
    switch (platform) {
      case 'telegram':
        return (
          <TelegramProvider>
            <ScopeProvider apiClient={api}>
              {children}
            </ScopeProvider>
          </TelegramProvider>
        );
        
      case 'mobile':
        return (
          <CapacitorProvider>
            <ScopeProvider apiClient={api}>
              {children}
            </ScopeProvider>
          </CapacitorProvider>
        );
        
      case 'desktop':
        return (
          <TauriProvider>
            <ScopeProvider apiClient={api}>
              {children}
            </ScopeProvider>
          </TauriProvider>
        );
        
      case 'web':
      default:
        return (
          <PWAProvider>
            <ScopeProvider apiClient={api}>
              {children}
            </ScopeProvider>
          </PWAProvider>
        );
    }
  };

  return renderPlatformProviders(
    <div className="relative min-h-screen overflow-hidden">
      <div className="pointer-events-none absolute inset-x-0 top-[-12rem] h-[34rem] bg-[radial-gradient(circle_at_top_left,_rgba(217,119,6,0.24),_transparent_52%),radial-gradient(circle_at_top_right,_rgba(14,165,233,0.18),_transparent_48%),linear-gradient(180deg,_rgba(255,249,240,0.96),_rgba(246,238,226,0.88))]" />
      <div className="pointer-events-none absolute left-[-8rem] top-[22rem] h-72 w-72 rounded-full bg-emerald-300/20 blur-3xl" />
      <div className="pointer-events-none absolute right-[-6rem] top-[28rem] h-80 w-80 rounded-full bg-sky-300/20 blur-3xl" />

      <div className="relative">
        <Header
          title="Modulr Control Plane"
          showScopeSelector={true}
          showNotifications={false}
          showUserMenu={false}
          className="border-[color:var(--modulr-line)] bg-[rgba(255,251,245,0.88)] backdrop-blur-xl"
        />

        <main className="mx-auto max-w-7xl px-4 pb-12 pt-6 sm:px-6 lg:px-8">
          <ControlPlaneDashboard platform={platform} />
        </main>
      </div>
    </div>
  );
}

export default App;
