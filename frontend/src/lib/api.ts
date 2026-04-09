/**
 * API Client for Modulr backend
 * Unified interface for all backend communication
 */

import type { 
  ApiResponse, 
  ApiError, 
  ResponseMeta, 
  PaginationMeta,
  SearchQuery,
  SearchResult,
  Scope,
  User,
  Notification
} from '@modulr/core-types';

// ============================================================================
// API CONFIGURATION
// ============================================================================

export interface ApiConfig {
  baseURL: string;
  timeout: number;
  retries: number;
  retryDelay: number;
  headers: Record<string, string>;
}

const defaultConfig: ApiConfig = {
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api',
  timeout: 10000,
  retries: 3,
  retryDelay: 1000,
  headers: {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
  },
};

// ============================================================================
// API CLIENT
// ============================================================================

export class ApiClient {
  private config: ApiConfig;
  private token: string | null = null;

  constructor(config: Partial<ApiConfig> = {}) {
    this.config = { ...defaultConfig, ...config };
  }

  // ============================================================================
  // AUTHENTICATION
  // ============================================================================

  setToken(token: string): void {
    this.token = token;
  }

  clearToken(): void {
    this.token = null;
  }

  private getAuthHeaders(): Record<string, string> {
    return this.token ? { Authorization: `Bearer ${this.token}` } : {};
  }

  // ============================================================================
  // HTTP METHODS
  // ============================================================================

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    const url = `${this.config.baseURL}${endpoint}`;
    const headers = {
      ...this.config.headers,
      ...this.getAuthHeaders(),
      ...options.headers,
    };

    let lastError: Error;

    for (let attempt = 0; attempt <= this.config.retries; attempt++) {
      try {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

        const response = await fetch(url, {
          ...options,
          headers,
          signal: controller.signal,
        });

        clearTimeout(timeoutId);

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new ApiError(
            errorData.code || `HTTP_${response.status}`,
            errorData.message || `HTTP ${response.status}: ${response.statusText}`,
            errorData.details
          );
        }

        const data = await response.json();
        
        return {
          data,
          meta: {
            timing: {
              duration: 0, // Will be set by wrapper
              timestamp: Date.now(),
              cacheHit: false,
            },
          },
        };

      } catch (error) {
        lastError = error instanceof Error ? error : new Error('Unknown error');
        
        if (attempt === this.config.retries) {
          break;
        }

        // Wait before retry
        await new Promise(resolve => 
          setTimeout(resolve, this.config.retryDelay * Math.pow(2, attempt))
        );
      }
    }

    return {
      error: lastError instanceof ApiError ? lastError : new ApiError(
        'NETWORK_ERROR',
        lastError?.message || 'Network request failed'
      ),
    };
  }

  async get<T>(endpoint: string, params?: Record<string, any>): Promise<ApiResponse<T>> {
    const url = new URL(endpoint, this.config.baseURL);
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          url.searchParams.append(key, String(value));
        }
      });
    }

    return this.request<T>(url.pathname + url.search);
  }

  async post<T>(endpoint: string, data?: any): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async put<T>(endpoint: string, data?: any): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async patch<T>(endpoint: string, data?: any): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T>(endpoint: string): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'DELETE',
    });
  }

  // ============================================================================
  // AUTHENTICATION ENDPOINTS
  // ============================================================================

  async login(credentials: { username: string; password: string }): Promise<ApiResponse<{ token: string; user: User }>> {
    return this.post('/auth/login', credentials);
  }

  async refreshToken(refreshToken: string): Promise<ApiResponse<{ token: string }>> {
    return this.post('/auth/refresh', { refreshToken });
  }

  async logout(): Promise<ApiResponse<void>> {
    return this.post('/auth/logout');
  }

  async getCurrentUser(): Promise<ApiResponse<User>> {
    return this.get('/auth/me');
  }

  // ============================================================================
  // USER ENDPOINTS
  // ============================================================================

  async updateProfile(data: Partial<User>): Promise<ApiResponse<User>> {
    return this.put('/users/profile', data);
  }

  async updatePreferences(preferences: User['preferences']): Promise<ApiResponse<User>> {
    return this.put('/users/preferences', preferences);
  }

  // ============================================================================
  // SCOPE ENDPOINTS
  // ============================================================================

  async getScopes(): Promise<ApiResponse<Scope[]>> {
    return this.get('/scopes');
  }

  async createScope(scope: Omit<Scope, 'metadata'>): Promise<ApiResponse<Scope>> {
    return this.post('/scopes', scope);
  }

  async updateScope(id: string, updates: Partial<Scope>): Promise<ApiResponse<Scope>> {
    return this.put(`/scopes/${id}`, updates);
  }

  async deleteScope(id: string): Promise<ApiResponse<void>> {
    return this.delete(`/scopes/${id}`);
  }

  // ============================================================================
  // SEARCH ENDPOINTS
  // ============================================================================

  async search<T>(query: SearchQuery): Promise<ApiResponse<SearchResult<T>>> {
    return this.post('/search', query);
  }

  // ============================================================================
  // NOTIFICATION ENDPOINTS
  // ============================================================================

  async getNotifications(params?: { 
    page?: number; 
    limit?: number; 
    unread?: boolean;
  }): Promise<ApiResponse<{ notifications: Notification[]; pagination: PaginationMeta }>> {
    return this.get('/notifications', params);
  }

  async markNotificationAsRead(id: string): Promise<ApiResponse<void>> {
    return this.patch(`/notifications/${id}/read`);
  }

  async markAllNotificationsAsRead(): Promise<ApiResponse<void>> {
    return this.patch('/notifications/read-all');
  }

  // ============================================================================
  // EVENT ENDPOINTS
  // ============================================================================

  async emitEvent(eventName: string, payload: any): Promise<ApiResponse<void>> {
    return this.post('/events', { name: eventName, payload });
  }

  // ============================================================================
  // FILE ENDPOINTS
  // ============================================================================

  async uploadFile(file: File, onProgress?: (progress: number) => void): Promise<ApiResponse<{ id: string; url: string }>> {
    const formData = new FormData();
    formData.append('file', file);

    return new Promise((resolve) => {
      const xhr = new XMLHttpRequest();

      if (onProgress) {
        xhr.upload.addEventListener('progress', (event) => {
          if (event.lengthComputable) {
            const progress = (event.loaded / event.total) * 100;
            onProgress(progress);
          }
        });
      }

      xhr.addEventListener('load', () => {
        try {
          const response = JSON.parse(xhr.responseText);
          resolve({ data: response });
        } catch (error) {
          resolve({
            error: new ApiError('PARSE_ERROR', 'Failed to parse response')
          });
        }
      });

      xhr.addEventListener('error', () => {
        resolve({
          error: new ApiError('UPLOAD_ERROR', 'File upload failed')
        });
      });

      xhr.open('POST', `${this.config.baseURL}/files/upload`);
      
      // Set headers
      Object.entries({
        ...this.config.headers,
        ...this.getAuthHeaders(),
      }).forEach(([key, value]) => {
        xhr.setRequestHeader(key, value);
      });

      xhr.send(formData);
    });
  }

  async getFileUrl(fileId: string): Promise<ApiResponse<{ url: string }>> {
    return this.get(`/files/${fileId}/url`);
  }

  async deleteFile(fileId: string): Promise<ApiResponse<void>> {
    return this.delete(`/files/${fileId}`);
  }

  // ============================================================================
  // HEALTH ENDPOINTS
  // ============================================================================

  async healthCheck(): Promise<ApiResponse<{ status: string; timestamp: number }>> {
    return this.get('/health');
  }

  async getMetrics(): Promise<ApiResponse<{ 
    uptime: number; 
    memory: number; 
    requests: number;
    errors: number;
  }>> {
    return this.get('/metrics');
  }
}

// ============================================================================
// API ERROR CLASS
// ============================================================================

export class ApiError extends Error {
  constructor(
    public code: string,
    message: string,
    public details?: Record<string, any>
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

// ============================================================================
// GLOBAL API INSTANCE
// ============================================================================

export const api = new ApiClient();

// ============================================================================
// API HOOKS (for React Query integration)
// ============================================================================

export const apiHooks = {
  // Auth hooks
  useLogin: () => ({
    mutationFn: (credentials: { username: string; password: string }) => 
      api.login(credentials),
  }),

  useRefreshToken: () => ({
    mutationFn: (refreshToken: string) => 
      api.refreshToken(refreshToken),
  }),

  useLogout: () => ({
    mutationFn: () => api.logout(),
  }),

  useCurrentUser: () => ({
    queryKey: ['currentUser'],
    queryFn: () => api.getCurrentUser(),
  }),

  // User hooks
  useUpdateProfile: () => ({
    mutationFn: (data: Partial<User>) => api.updateProfile(data),
  }),

  useUpdatePreferences: () => ({
    mutationFn: (preferences: User['preferences']) => api.updatePreferences(preferences),
  }),

  // Scope hooks
  useScopes: () => ({
    queryKey: ['scopes'],
    queryFn: () => api.getScopes(),
  }),

  useCreateScope: () => ({
    mutationFn: (scope: Omit<Scope, 'metadata'>) => api.createScope(scope),
  }),

  useUpdateScope: () => ({
    mutationFn: ({ id, updates }: { id: string; updates: Partial<Scope> }) => 
      api.updateScope(id, updates),
  }),

  useDeleteScope: () => ({
    mutationFn: (id: string) => api.deleteScope(id),
  }),

  // Search hooks
  useSearch: <T>() => ({
    mutationFn: (query: SearchQuery) => api.search<T>(query),
  }),

  // Notification hooks
  useNotifications: (params?: { 
    page?: number; 
    limit?: number; 
    unread?: boolean;
  }) => ({
    queryKey: ['notifications', params],
    queryFn: () => api.getNotifications(params),
  }),

  useMarkNotificationAsRead: () => ({
    mutationFn: (id: string) => api.markNotificationAsRead(id),
  }),

  useMarkAllNotificationsAsRead: () => ({
    mutationFn: () => api.markAllNotificationsAsRead(),
  }),

  // Event hooks
  useEmitEvent: () => ({
    mutationFn: ({ eventName, payload }: { eventName: string; payload: any }) => 
      api.emitEvent(eventName, payload),
  }),

  // File hooks
  useUploadFile: () => ({
    mutationFn: ({ file, onProgress }: { file: File; onProgress?: (progress: number) => void }) => 
      api.uploadFile(file, onProgress),
  }),

  useGetFileUrl: () => ({
    queryKey: ['fileUrl'],
    queryFn: (fileId: string) => api.getFileUrl(fileId),
  }),

  useDeleteFile: () => ({
    mutationFn: (fileId: string) => api.deleteFile(fileId),
  }),

  // Health hooks
  useHealthCheck: () => ({
    queryKey: ['health'],
    queryFn: () => api.healthCheck(),
    refetchInterval: 30000, // Check every 30 seconds
  }),

  useMetrics: () => ({
    queryKey: ['metrics'],
    queryFn: () => api.getMetrics(),
    refetchInterval: 60000, // Check every minute
  }),
};

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

/**
 * Check if API response has an error
 */
export function hasError<T>(response: ApiResponse<T>): response is ApiResponse<T> & { error: ApiError } {
  return !!response.error;
}

/**
 * Get data from API response or throw error
 */
export function getData<T>(response: ApiResponse<T>): T {
  if (hasError(response)) {
    throw response.error;
  }
  return response.data!;
}

/**
 * Create API error from fetch error
 */
export function createApiError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error;
  }

  if (error instanceof Error) {
    return new ApiError('UNKNOWN_ERROR', error.message);
  }

  return new ApiError('UNKNOWN_ERROR', 'An unknown error occurred');
}

// ============================================================================
// EXPORTS
// ============================================================================

export { ApiClient };
export type { ApiConfig };
