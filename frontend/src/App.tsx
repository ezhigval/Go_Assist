/**
 * Main App component - Root of the application
 * Platform detection and provider setup
 */

import React from 'react';
import { ScopeProvider } from './context/ScopeContext';
import { TelegramProvider } from './platforms/telegram/TelegramProvider';
import { PWAProvider } from './platforms/web/PWAProvider';
import { CapacitorProvider } from './platforms/mobile/CapacitorProvider';
import { TauriProvider } from './platforms/desktop/TauriProvider';
import { Header } from './components/layout/Header';
import { eventBus } from './lib/eventBus';
import { api } from './lib/api';

// ============================================================================
// PLATFORM DETECTION
// ============================================================================

const getPlatform = (): 'telegram' | 'web' | 'mobile' | 'desktop' => {
  // Check for Telegram WebApp
  if (typeof window !== 'undefined' && (window as any).Telegram?.WebApp) {
    return 'telegram';
  }
  
  // Check for Capacitor (mobile)
  if (typeof window !== 'undefined' && (window as any).Capacitor) {
    return 'mobile';
  }
  
  // Check for Tauri (desktop)
  if (typeof window !== 'undefined' && (window as any).__TAURI__) {
    return 'desktop';
  }
  
  // Default to web
  return 'web';
};

// ============================================================================
// MAIN APP COMPONENT
// ============================================================================

function App() {
  const platform = getPlatform();
  
  // ============================================================================
  // PLATFORM-SPECIFIC PROVIDERS
  // ============================================================================
  
  const renderPlatformProviders = (children: React.ReactNode) => {
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
  
  // ============================================================================
  // MAIN CONTENT
  // ============================================================================
  
  const MainContent = () => {
    return (
      <div className="min-h-screen bg-gray-50">
        {/* Header */}
        <Header
          title="Modulr"
          showScopeSelector={true}
          showNotifications={true}
          showUserMenu={true}
        />
        
        {/* Main content area */}
        <main className="container mx-auto px-4 py-8">
          <div className="space-y-6">
            {/* Welcome section */}
            <div className="text-center">
              <h1 className="text-3xl font-bold text-gray-900 mb-2">
                Welcome to Modulr
              </h1>
              <p className="text-gray-600">
                Your personal assistant for productivity and organization
              </p>
            </div>
            
            {/* Platform indicator */}
            <div className="flex justify-center">
              <div className="inline-flex items-center px-3 py-1 rounded-full text-sm bg-blue-100 text-blue-800">
                <span className="font-medium">Platform:</span>
                <span className="ml-1 font-bold">{platform}</span>
              </div>
            </div>
            
            {/* Feature cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <h3 className="text-lg font-semibold text-gray-900 mb-2">
                  Task Management
                </h3>
                <p className="text-gray-600">
                  Organize your tasks and projects with smart prioritization
                </p>
              </div>
              
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <h3 className="text-lg font-semibold text-gray-900 mb-2">
                  Calendar Integration
                </h3>
                <p className="text-gray-600">
                  Sync your events and never miss an important deadline
                </p>
              </div>
              
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <h3 className="text-lg font-semibold text-gray-900 mb-2">
                  AI Assistant
                </h3>
                <p className="text-gray-600">
                  Get intelligent suggestions and automate your workflow
                </p>
              </div>
            </div>
            
            {/* Quick actions */}
            <div className="flex justify-center space-x-4">
              <button
                onClick={() => eventBus.emit('test-event', { message: 'Hello from Modulr!' })}
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
              >
                Test Event Bus
              </button>
              
              <button
                onClick={() => api.healthCheck()}
                className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 transition-colors"
              >
                Check API Status
              </button>
            </div>
          </div>
        </main>
      </div>
    );
  };
  
  // ============================================================================
  // RENDER
  // ============================================================================
  
  return renderPlatformProviders(<MainContent />);
}

// ============================================================================
// EXPORTS
// ============================================================================

export default App;
