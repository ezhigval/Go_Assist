/**
 * Core type definitions for Modulr frontend
 * These types align with backend event system and provide type safety
 */

// ============================================================================
// SCOPES AND SEGMENTS
// ============================================================================

export type Segment = 
  | 'personal'
  | 'family'
  | 'work'
  | 'business'
  | 'health'
  | 'travel'
  | 'pets'
  | 'assets';

export const AllSegments: Segment[] = [
  'personal',
  'family',
  'work',
  'business',
  'health',
  'travel',
  'pets',
  'assets',
];

export interface Scope {
  segment: Segment;
  tags: string[];
  metadata?: Record<string, unknown>;
}

export interface Context {
  scope: Scope;
  user: User;
  platform: Platform;
  timestamp: number;
}

// ============================================================================
// PLATFORMS
// ============================================================================

export type Platform = 
  | 'telegram'
  | 'web'
  | 'mobile'
  | 'desktop'
  | 'wearable';

export interface PlatformCapabilities {
  platform: Platform;
  hasNotifications: boolean;
  hasGeolocation: boolean;
  hasCamera: boolean;
  hasBiometrics: boolean;
  hasOfflineSupport: boolean;
  hasVoiceInput: boolean;
  screenInfo: ScreenInfo;
}

export interface ScreenInfo {
  width: number;
  height: number;
  density: number;
  orientation: 'portrait' | 'landscape';
  isSmall: boolean;
  isTouch: boolean;
}

// ============================================================================
// USER AND AUTHENTICATION
// ============================================================================

export interface User {
  id: string;
  username: string;
  displayName: string;
  avatar?: string;
  email?: string;
  roles: Role[];
  preferences: UserPreferences;
  session: UserSession;
}

export type Role = 
  | 'user'
  | 'admin'
  | 'moderator'
  | 'developer';

export interface UserPreferences {
  language: string;
  theme: Theme;
  notifications: NotificationPreferences;
  privacy: PrivacyPreferences;
  accessibility: AccessibilityPreferences;
}

export type Theme = 
  | 'light'
  | 'dark'
  | 'auto';

export interface NotificationPreferences {
  push: boolean;
  email: boolean;
  inApp: boolean;
  types: NotificationType[];
}

export type NotificationType = 
  | 'system'
  | 'reminder'
  | 'message'
  | 'achievement'
  | 'deadline'
  | 'suggestion';

export interface PrivacyPreferences {
  shareAnalytics: boolean;
  shareCrashReports: boolean;
  shareUsageData: boolean;
  dataRetention: DataRetention;
}

export type DataRetention = 
  | '30days'
  | '90days'
  | '1year'
  | 'forever';

export interface AccessibilityPreferences {
  fontSize: FontSize;
  highContrast: boolean;
  reduceMotion: boolean;
  screenReader: boolean;
  keyboardNavigation: boolean;
}

export type FontSize = 
  | 'xs'
  | 'sm'
  | 'base'
  | 'lg'
  | 'xl'
  | '2xl';

export interface UserSession {
  token: string;
  refreshToken: string;
  expiresAt: number;
  isActive: boolean;
  lastActivity: number;
}

// ============================================================================
// EVENTS AND EVENT BUS
// ============================================================================

export interface BaseEvent {
  id: string;
  name: EventName;
  payload: unknown;
  source: string;
  traceId: string;
  timestamp: number;
  context?: Record<string, unknown>;
}

export type EventName = 
  // System events
  | 'v1.system.startup'
  | 'v1.system.shutdown'
  | 'v1.system.error'
  
  // User events
  | 'v1.user.login'
  | 'v1.user.logout'
  | 'v1.user.preferences.updated'
  
  // Calendar events
  | 'v1.calendar.created'
  | 'v1.calendar.meeting.created'
  | 'v1.calendar.updated'
  | 'v1.calendar.deleted'
  
  // Task/Tracker events
  | 'v1.todo.created'
  | 'v1.todo.due'
  | 'v1.todo.completed'
  | 'v1.tracker.milestone.reached'
  
  // Finance events
  | 'v1.finance.transaction.created'
  | 'v1.finance.budget.exceeded'
  | 'v1.finance.subscription.created'
  
  // Knowledge events
  | 'v1.knowledge.article.saved'
  | 'v1.knowledge.query'
  | 'v1.knowledge.recommendation'
  
  // Communication events
  | 'v1.email.received'
  | 'v1.email.sent'
  | 'v1.transport.file.received'
  
  // AI events
  | 'v1.ai.suggestion'
  | 'v1.ai.decision'
  | 'v1.orchestrator.fallback.requested'
  
  // Notification events
  | 'v1.notification.push'
  | 'v1.reminder.on_route';

export interface TypedEvent<T = unknown> extends BaseEvent {
  payload: T;
  name: EventName;
}

export interface EventHandler<T = unknown> {
  (event: TypedEvent<T>): void | Promise<void>;
}

export interface EventBus {
  on<T = unknown>(eventName: EventName, handler: EventHandler<T>): () => void;
  off<T = unknown>(eventName: EventName, handler: EventHandler<T>): void;
  emit<T = unknown>(eventName: EventName, payload: T, options?: EventOptions): void;
  once<T = unknown>(eventName: EventName, handler: EventHandler<T>): () => void;
  clear(): void;
  listenerCount(eventName: EventName): number;
}

export interface EventOptions {
  source?: string;
  traceId?: string;
  context?: Record<string, unknown>;
  immediate?: boolean;
}

// ============================================================================
// AI DECISIONS AND SUGGESTIONS
// ============================================================================

export interface AIDecision {
  id: string;
  target: string;
  action: string;
  parameters: Record<string, unknown>;
  confidence: number;
  scope: string;
  createdAt: number;
  metadata?: Record<string, unknown>;
}

export interface AISuggestion {
  id: string;
  type: SuggestionType;
  title: string;
  description: string;
  action: SuggestionAction;
  confidence: number;
  urgency: Urgency;
  scope: Scope;
  metadata?: Record<string, unknown>;
  expiresAt?: number;
}

export type SuggestionType = 
  | 'action'
  | 'reminder'
  | 'insight'
  | 'recommendation'
  | 'automation';

export interface SuggestionAction {
  label: string;
  type: ActionType;
  payload: Record<string, unknown>;
  requiresConfirmation: boolean;
}

export type ActionType = 
  | 'create'
  | 'update'
  | 'delete'
  | 'navigate'
  | 'external'
  | 'custom';

export type Urgency = 
  | 'low'
  | 'medium'
  | 'high'
  | 'critical';

// ============================================================================
// API RESPONSES AND ERRORS
// ============================================================================

export interface ApiResponse<T = unknown> {
  data?: T;
  error?: ApiError;
  meta?: ResponseMeta;
}

export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
  stack?: string;
}

export interface ResponseMeta {
  pagination?: PaginationMeta;
  timing?: TimingMeta;
  version?: string;
}

export interface PaginationMeta {
  page: number;
  limit: number;
  total: number;
  totalPages: number;
  hasNext: boolean;
  hasPrev: boolean;
}

export interface TimingMeta {
  duration: number;
  timestamp: number;
  cacheHit: boolean;
}

// ============================================================================
// OFFLINE AND SYNCHRONIZATION
// ============================================================================

export interface OfflineAction {
  id: string;
  type: ActionType;
  payload: Record<string, unknown>;
  timestamp: number;
  retryCount: number;
  status: OfflineStatus;
  error?: string;
}

export type OfflineStatus = 
  | 'pending'
  | 'syncing'
  | 'completed'
  | 'failed'
  | 'conflict';

export interface SyncConflict {
  id: string;
  localAction: OfflineAction;
  serverAction: OfflineAction;
  resolution?: ConflictResolution;
}

export interface ConflictResolution {
  strategy: ConflictStrategy;
  winner: 'local' | 'server' | 'manual';
  mergedData?: Record<string, unknown>;
}

export type ConflictStrategy = 
  | 'local_wins'
  | 'server_wins'
  | 'merge'
  | 'manual';

// ============================================================================
// NOTIFICATIONS
// ============================================================================

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  body: string;
  data?: Record<string, unknown>;
  actions?: NotificationAction[];
  timestamp: number;
  read: boolean;
  expiresAt?: number;
  priority: NotificationPriority;
}

export interface NotificationAction {
  id: string;
  label: string;
  action: string;
  autoDismiss?: boolean;
}

export type NotificationPriority = 
  | 'low'
  | 'normal'
  | 'high'
  | 'urgent';

// ============================================================================
// SEARCH AND FILTERING
// ============================================================================

export interface SearchQuery {
  query: string;
  filters?: SearchFilter[];
  sort?: SortOption;
  pagination?: PaginationOptions;
  scope?: Scope;
}

export interface SearchFilter {
  field: string;
  operator: FilterOperator;
  value: unknown;
}

export type FilterOperator = 
  | 'eq'
  | 'ne'
  | 'gt'
  | 'gte'
  | 'lt'
  | 'lte'
  | 'in'
  | 'nin'
  | 'contains'
  | 'startsWith'
  | 'endsWith';

export interface SortOption {
  field: string;
  direction: 'asc' | 'desc';
}

export interface PaginationOptions {
  page: number;
  limit: number;
}

export interface SearchResult<T = unknown> {
  items: T[];
  total: number;
  page: number;
  limit: number;
  hasMore: boolean;
  facets?: SearchFacet[];
}

export interface SearchFacet {
  field: string;
  values: FacetValue[];
}

export interface FacetValue {
  value: string;
  count: number;
  selected?: boolean;
}

// ============================================================================
// VALIDATION AND UTILITIES
// ============================================================================

export interface ValidationResult {
  isValid: boolean;
  errors: ValidationError[];
  warnings: ValidationWarning[];
}

export interface ValidationError {
  field: string;
  message: string;
  code: string;
}

export interface ValidationWarning {
  field: string;
  message: string;
  code: string;
}

export interface Logger {
  debug(message: string, ...args: unknown[]): void;
  info(message: string, ...args: unknown[]): void;
  warn(message: string, ...args: unknown[]): void;
  error(message: string, error?: Error, ...args: unknown[]): void;
}

// ============================================================================
// THEME AND STYLING
// ============================================================================

export interface ThemeConfig {
  mode: Theme;
  colors: ThemeColors;
  typography: ThemeTypography;
  spacing: ThemeSpacing;
  shadows: ThemeShadows;
}

export interface ThemeColors {
  primary: ColorPalette;
  secondary: ColorPalette;
  accent: ColorPalette;
  neutral: ColorPalette;
  semantic: SemanticColors;
  scope: ScopeColors;
}

export interface ColorPalette {
  50: string;
  100: string;
  200: string;
  300: string;
  400: string;
  500: string;
  600: string;
  700: string;
  800: string;
  900: string;
}

export interface SemanticColors {
  success: string;
  warning: string;
  error: string;
  info: string;
}

export type ScopeColors = {
  [key in Segment]: string;
};

export interface ThemeTypography {
  fontFamily: {
    sans: string[];
    mono: string[];
    display: string[];
  };
  fontSize: Record<FontSize, string>;
  fontWeight: Record<string, number>;
  lineHeight: Record<string, number>;
}

export interface ThemeSpacing {
  xs: string;
  sm: string;
  md: string;
  lg: string;
  xl: string;
  '2xl': string;
  '3xl': string;
}

export interface ThemeShadows {
  soft: string;
  medium: string;
  strong: string;
}
