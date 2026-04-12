/**
 * Header component - Main application header
 * Responsive design with platform-specific features
 */

import { useState } from 'react';
import { cn } from '../../lib/utils';
import { Button } from '../ui/Button';
import { useScope } from '../../context/ScopeContext';
import { useTelegram } from '../../platforms/telegram/TelegramProvider';
import { usePWA } from '../../platforms/web/PWAProvider';
import { useCapacitor } from '../../platforms/mobile/CapacitorProvider';
import { useTauri } from '../../platforms/desktop/TauriProvider';

// ============================================================================
// HEADER PROPS
// ============================================================================

export interface HeaderProps {
  title?: string;
  showBackButton?: boolean;
  showMenuButton?: boolean;
  showScopeSelector?: boolean;
  showNotifications?: boolean;
  showUserMenu?: boolean;
  onBackClick?: () => void;
  onMenuClick?: () => void;
  className?: string;
}

// ============================================================================
// HEADER COMPONENT
// ============================================================================

export function Header({
  title = 'Modulr',
  showBackButton = false,
  showMenuButton = true,
  showScopeSelector = true,
  showNotifications = true,
  showUserMenu = true,
  onBackClick,
  onMenuClick,
  className,
}: HeaderProps) {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  
  // Platform hooks
  const telegram = useTelegram();
  const pwa = usePWA();
  const mobile = useCapacitor();
  const desktop = useTauri();
  
  // Scope context
  const { activeScope, setActiveScope, availableScopes } = useScope();
  
  // Determine current platform
  const isTelegram = telegram.isReady;
  const isMobile = mobile.isNative;
  const isDesktop = desktop.isDesktop;
  const isPWA = pwa.isInstalled || pwa.isInstallable;
  
  // Platform-specific features
  const showHamburgerMenu = isMobile || isPWA;
  const showWindowControls = isDesktop;
  const showTelegramBackButton = isTelegram && showBackButton;
  
  // ============================================================================
  // EVENT HANDLERS
  // ============================================================================

  const handleBackClick = () => {
    if (isTelegram) {
      // Use Telegram's back button functionality
      telegram.hideBackButton();
    }
    
    onBackClick?.();
  };

  const handleMenuClick = () => {
    if (showHamburgerMenu) {
      setIsMenuOpen(!isMenuOpen);
    }
    
    onMenuClick?.();
  };

  const handleScopeChange = (scope: typeof activeScope) => {
    setActiveScope(scope);
    setIsMenuOpen(false);
  };

  const handleNotificationClick = () => {
    // Open notifications panel
    console.log('Open notifications');
    setIsMenuOpen(false);
  };

  const handleUserMenuClick = () => {
    // Open user menu
    console.log('Open user menu');
    setIsMenuOpen(false);
  };

  const handleWindowControl = (action: 'minimize' | 'maximize' | 'close') => {
    switch (action) {
      case 'minimize':
        desktop.minimizeWindow();
        break;
      case 'maximize':
        if (desktop.windowInfo.isMaximized) {
          desktop.unmaximizeWindow();
        } else {
          desktop.maximizeWindow();
        }
        break;
      case 'close':
        desktop.closeWindow();
        break;
    }
  };

  // ============================================================================
  // PLATFORM-SPECIFIC COMPONENTS
  // ============================================================================

  const TelegramBackButton = () => {
    if (!showTelegramBackButton) return null;
    
    return (
      <Button
        variant="ghost"
        size="sm"
        onClick={handleBackClick}
        className="text-gray-600 hover:text-gray-900"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
        </svg>
      </Button>
    );
  };

  const HamburgerMenu = () => {
    if (!showHamburgerMenu || !showMenuButton) return null;
    
    return (
      <Button
        variant="ghost"
        size="sm"
        onClick={handleMenuClick}
        className="text-gray-600 hover:text-gray-900"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </Button>
    );
  };

  const WindowControls = () => {
    if (!showWindowControls) return null;
    
    return (
      <div className="flex items-center space-x-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => handleWindowControl('minimize')}
          className="text-gray-600 hover:text-gray-900 p-1"
        >
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <rect x="2" y="9" width="16" height="2" />
          </svg>
        </Button>
        
        <Button
          variant="ghost"
          size="sm"
          onClick={() => handleWindowControl('maximize')}
          className="text-gray-600 hover:text-gray-900 p-1"
        >
          {desktop.windowInfo.isMaximized ? (
            <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
              <path d="M9 9h6v6H9zM3 3h6v6H3zM15 15h6v6h-6z" />
            </svg>
          ) : (
            <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
              <path d="M3 3h18v18H3z" />
            </svg>
          )}
        </Button>
        
        <Button
          variant="ghost"
          size="sm"
          onClick={() => handleWindowControl('close')}
          className="text-gray-600 hover:text-red-600 p-1"
        >
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
          </svg>
        </Button>
      </div>
    );
  };

  const ScopeSelector = () => {
    if (!showScopeSelector) return null;
    
    return (
      <div className="flex items-center space-x-2">
        <span className="text-sm text-gray-500">Scope:</span>
        <select
          value={`${activeScope.segment}:${activeScope.tags.join(',')}`}
          onChange={(e) => {
            const [segment, ...tags] = e.target.value.split(':');
            const scope = {
              segment: segment as any,
              tags: tags.join(',').split(',').filter(Boolean),
              metadata: {},
            };
            handleScopeChange(scope);
          }}
          className="text-sm font-medium border-0 bg-transparent focus:outline-none focus:ring-2 focus:ring-primary-500 rounded px-2 py-1"
        >
          {availableScopes.map((scope) => (
            <option
              key={`${scope.segment}:${scope.tags.join(',')}`}
              value={`${scope.segment}:${scope.tags.join(',')}`}
            >
              {scope.segment.charAt(0).toUpperCase() + scope.segment.slice(1)}
              {scope.tags.length > 0 && ` (${scope.tags.join(', ')})`}
            </option>
          ))}
        </select>
        
        {/* Scope indicator */}
        <div
          className="w-3 h-3 rounded-full border-2 border-gray-300"
          style={{ backgroundColor: getScopeColor(activeScope.segment) }}
        />
      </div>
    );
  };

  const NotificationButton = () => {
    if (!showNotifications) return null;
    
    return (
      <Button
        variant="ghost"
        size="sm"
        onClick={handleNotificationClick}
        className="text-gray-600 hover:text-gray-900 relative"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
        </svg>
        
        {/* Notification badge */}
        <span className="absolute -top-1 -right-1 w-2 h-2 bg-red-500 rounded-full" />
      </Button>
    );
  };

  const UserMenuButton = () => {
    if (!showUserMenu) return null;
    
    return (
      <Button
        variant="ghost"
        size="sm"
        onClick={handleUserMenuClick}
        className="text-gray-600 hover:text-gray-900"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
        </svg>
      </Button>
    );
  };

  // ============================================================================
  // MOBILE MENU
  // ============================================================================

  const MobileMenu = () => {
    if (!isMenuOpen) return null;
    
    return (
      <div className="absolute top-full left-0 right-0 bg-white border-t border-gray-200 shadow-lg z-50">
        <div className="p-4 space-y-4">
          {/* Scope selector in mobile menu */}
          <div>
            <h3 className="text-sm font-medium text-gray-900 mb-2">Scope</h3>
            <div className="space-y-2">
              {availableScopes.map((scope) => (
                <button
                  key={`${scope.segment}:${scope.tags.join(',')}`}
                  onClick={() => handleScopeChange(scope)}
                  className={cn(
                    "w-full text-left px-3 py-2 rounded-md text-sm",
                    `${scope.segment}:${scope.tags.join(',')}` === `${activeScope.segment}:${activeScope.tags.join(',')}`
                      ? "bg-primary-100 text-primary-700"
                      : "text-gray-700 hover:bg-gray-100"
                  )}
                >
                  <div className="flex items-center justify-between">
                    <span>
                      {scope.segment.charAt(0).toUpperCase() + scope.segment.slice(1)}
                      {scope.tags.length > 0 && ` (${scope.tags.join(', ')})`}
                    </span>
                    <div
                      className="w-2 h-2 rounded-full"
                      style={{ backgroundColor: getScopeColor(scope.segment) }}
                    />
                  </div>
                </button>
              ))}
            </div>
          </div>
          
          {/* Notifications */}
          <button
            onClick={handleNotificationClick}
            className="w-full text-left px-3 py-2 rounded-md text-sm text-gray-700 hover:bg-gray-100"
          >
            Notifications
          </button>
          
          {/* User menu */}
          <button
            onClick={handleUserMenuClick}
            className="w-full text-left px-3 py-2 rounded-md text-sm text-gray-700 hover:bg-gray-100"
          >
            Account Settings
          </button>
        </div>
      </div>
    );
  };

  // ============================================================================
  // RENDER
  // ============================================================================

  return (
    <header className={cn(
      "bg-white border-b border-gray-200 shadow-sm",
      "relative z-40",
      className
    )}>
      <div className="px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Left side */}
          <div className="flex items-center space-x-4">
            {/* Window controls (desktop) */}
            {showWindowControls && <WindowControls />}
            
            {/* Back button */}
            <TelegramBackButton />
            
            {/* Menu button */}
            <HamburgerMenu />
            
            {/* Title */}
            <h1 className="text-xl font-semibold text-gray-900 truncate">
              {title}
            </h1>
          </div>
          
          {/* Center - Scope selector */}
          <div className="hidden md:flex items-center flex-1 justify-center">
            <ScopeSelector />
          </div>
          
          {/* Right side */}
          <div className="flex items-center space-x-2">
            {/* Notifications */}
            <NotificationButton />
            
            {/* User menu */}
            <UserMenuButton />
          </div>
        </div>
      </div>
      
      {/* Mobile menu */}
      <MobileMenu />
    </header>
  );
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

function getScopeColor(segment: string): string {
  const colors: Record<string, string> = {
    personal: '#10b981',
    family: '#f59e0b',
    work: '#3b82f6',
    business: '#0ea5e9',
    health: '#ef4444',
    travel: '#8b5cf6',
    pets: '#f97316',
    assets: '#64748b',
  };
  
  return colors[segment] || '#6b7280';
}
