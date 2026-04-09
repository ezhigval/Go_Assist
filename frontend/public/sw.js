/**
 * Service Worker for Modulr PWA
 * Offline support, caching, and background sync
 */

const CACHE_NAME = 'modulr-v1';
const STATIC_CACHE_NAME = 'modulr-static-v1';
const DYNAMIC_CACHE_NAME = 'modulr-dynamic-v1';

// Assets to cache immediately
const STATIC_ASSETS = [
  '/',
  '/index.html',
  '/manifest.json',
  '/vite.svg',
  '/tailwind.js',
  'https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap'
];

// ============================================================================
// INSTALLATION
// ============================================================================

self.addEventListener('install', (event) => {
  console.log('SW: Installing service worker');
  
  event.waitUntil(
    caches.open(STATIC_CACHE_NAME)
      .then((cache) => {
        console.log('SW: Caching static assets');
        return cache.addAll(STATIC_ASSETS);
      })
      .then(() => {
        console.log('SW: Static assets cached');
        return self.skipWaiting();
      })
      .catch((error) => {
        console.error('SW: Failed to cache static assets:', error);
      })
  );
});

// ============================================================================
// ACTIVATION
// ============================================================================

self.addEventListener('activate', (event) => {
  console.log('SW: Activating service worker');
  
  event.waitUntil(
    caches.keys()
      .then((cacheNames) => {
        return Promise.all(
          cacheNames.map((cacheName) => {
            if (cacheName !== CACHE_NAME && 
                cacheName !== STATIC_CACHE_NAME && 
                cacheName !== DYNAMIC_CACHE_NAME) {
              console.log('SW: Deleting old cache:', cacheName);
              return caches.delete(cacheName);
            }
          })
        );
      })
      .then(() => {
        console.log('SW: Old caches cleared');
        return self.clients.claim();
      })
      .catch((error) => {
        console.error('SW: Failed to activate:', error);
      })
  );
});

// ============================================================================
// FETCH STRATEGIES
// ============================================================================

self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);
  
  // Skip non-HTTP requests
  if (!request.url.startsWith('http')) {
    return;
  }
  
  // API requests - Network first, then cache
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(networkFirstStrategy(request));
    return;
  }
  
  // Static assets - Cache first, then network
  if (STATIC_ASSETS.some(asset => url.pathname === new URL(asset, self.location.origin).pathname)) {
    event.respondWith(cacheFirstStrategy(request));
    return;
  }
  
  // Dynamic content - Network first, then cache
  event.respondWith(networkFirstStrategy(request));
});

// ============================================================================
// CACHE STRATEGIES
// ============================================================================

// Cache First Strategy
async function cacheFirstStrategy(request) {
  try {
    const cachedResponse = await caches.match(request);
    
    if (cachedResponse) {
      console.log('SW: Serving from cache:', request.url);
      return cachedResponse;
    }
    
    // Try network
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      const cache = await caches.open(DYNAMIC_CACHE_NAME);
      cache.put(request, networkResponse.clone());
      console.log('SW: Cached new response:', request.url);
    }
    
    return networkResponse;
    
  } catch (error) {
    console.error('SW: Cache first strategy failed:', error);
    
    // Return offline page for navigation requests
    if (request.mode === 'navigate') {
      return caches.match('/index.html');
    }
    
    return new Response('Offline', {
      status: 503,
      statusText: 'Service Unavailable'
    });
  }
}

// Network First Strategy
async function networkFirstStrategy(request) {
  try {
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      const cache = await caches.open(DYNAMIC_CACHE_NAME);
      cache.put(request, networkResponse.clone());
      console.log('SW: Network response cached:', request.url);
    }
    
    return networkResponse;
    
  } catch (error) {
    console.log('SW: Network failed, trying cache:', request.url);
    
    const cachedResponse = await caches.match(request);
    
    if (cachedResponse) {
      return cachedResponse;
    }
    
    // Return offline page for navigation requests
    if (request.mode === 'navigate') {
      return caches.match('/index.html');
    }
    
    return new Response('Offline', {
      status: 503,
      statusText: 'Service Unavailable'
    });
  }
}

// ============================================================================
// BACKGROUND SYNC
// ============================================================================

self.addEventListener('sync', (event) => {
  console.log('SW: Background sync event:', event.tag);
  
  if (event.tag === 'background-sync') {
    event.waitUntil(doBackgroundSync());
  }
});

async function doBackgroundSync() {
  try {
    // Get all pending sync requests from IndexedDB
    const pendingRequests = await getPendingSyncRequests();
    
    console.log('SW: Processing', pendingRequests.length, 'pending sync requests');
    
    for (const request of pendingRequests) {
      try {
        await fetch(request.url, request.options);
        await removeSyncRequest(request.id);
        console.log('SW: Sync request completed:', request.id);
      } catch (error) {
        console.error('SW: Sync request failed:', request.id, error);
      }
    }
    
  } catch (error) {
    console.error('SW: Background sync failed:', error);
  }
}

// ============================================================================
// PUSH NOTIFICATIONS
// ============================================================================

self.addEventListener('push', (event) => {
  console.log('SW: Push notification received');
  
  if (!event.data) {
    return;
  }
  
  const options = event.data.json();
  
  event.waitUntil(
    self.registration.showNotification(options.title, {
      body: options.body,
      icon: options.icon || '/icon-192x192.png',
      badge: options.badge || '/badge-72x72.png',
      tag: options.tag,
      data: options.data,
      actions: options.actions,
      requireInteraction: options.requireInteraction || false,
      silent: options.silent || false
    })
  );
});

self.addEventListener('notificationclick', (event) => {
  console.log('SW: Notification clicked:', event.notification.tag);
  
  event.notification.close();
  
  if (event.action) {
    // Handle action buttons
    console.log('SW: Action clicked:', event.action);
    return;
  }
  
  // Focus or open the app
  event.waitUntil(
    clients.matchAll({ type: 'window' })
      .then((clientList) => {
        // Focus existing window if available
        for (const client of clientList) {
          if (client.url === self.location.origin && 'focus' in client) {
            return client.focus();
          }
        }
        
        // Open new window
        if (clients.openWindow) {
          return clients.openWindow('/');
        }
      })
  );
});

// ============================================================================
// MESSAGE HANDLING
// ============================================================================

self.addEventListener('message', (event) => {
  console.log('SW: Message received:', event.data);
  
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
  
  if (event.data && event.data.type === 'GET_VERSION') {
    event.ports[0].postMessage({ version: '1.0.0' });
  }
});

// ============================================================================
// INDEXEDDB HELPERS
// ============================================================================

async function getPendingSyncRequests() {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open('ModulrDB', 1);
    
    request.onerror = () => reject(request.error);
    
    request.onsuccess = () => {
      const db = request.result;
      const transaction = db.transaction(['syncRequests'], 'readonly');
      const store = transaction.objectStore('syncRequests');
      const getRequest = store.getAll();
      
      getRequest.onsuccess = () => resolve(getRequest.result || []);
      getRequest.onerror = () => reject(getRequest.error);
    };
    
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains('syncRequests')) {
        db.createObjectStore('syncRequests', { keyPath: 'id' });
      }
    };
  });
}

async function removeSyncRequest(id) {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open('ModulrDB', 1);
    
    request.onerror = () => reject(request.error);
    
    request.onsuccess = () => {
      const db = request.result;
      const transaction = db.transaction(['syncRequests'], 'readwrite');
      const store = transaction.objectStore('syncRequests');
      const deleteRequest = store.delete(id);
      
      deleteRequest.onsuccess = () => resolve();
      deleteRequest.onerror = () => reject(deleteRequest.error);
    };
  });
}

// ============================================================================
// CLEANUP
// ============================================================================

self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'CACHE_CLEANUP') {
    event.waitUntil(cleanupCache());
  }
});

async function cleanupCache() {
  try {
    const cache = await caches.open(DYNAMIC_CACHE_NAME);
    const requests = await cache.keys();
    const now = Date.now();
    const maxAge = 7 * 24 * 60 * 60 * 1000; // 7 days
    
    for (const request of requests) {
      const response = await cache.match(request);
      const date = response.headers.get('date');
      
      if (date && (now - new Date(date).getTime()) > maxAge) {
        await cache.delete(request);
        console.log('SW: Cleaned up expired cache entry:', request.url);
      }
    }
    
  } catch (error) {
    console.error('SW: Cache cleanup failed:', error);
  }
}

console.log('SW: Service worker script loaded');
