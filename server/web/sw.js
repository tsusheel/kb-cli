/**
 * KB-CLI Web Fuzzy Finder — Service Worker
 * Caches app shell for offline use and caches read-only API requests (items, tags, stats)
 */

const CACHE_VERSION = 'v4';
const SHELL_CACHE = `kb-shell-${CACHE_VERSION}`;
const DATA_CACHE = `kb-data-${CACHE_VERSION}`;

const SHELL_ASSETS = [
  '/',
  '/index.html',
  '/style.css',
  '/app.js',
  '/assets/site.webmanifest',
  '/manifest.json',
  '/assets/favicon.ico',
  '/assets/favicon.svg',
  '/assets/favicon-96x96.png',
  '/assets/apple-touch-icon.png',
  '/assets/web-app-manifest-192x192.png',
  '/assets/web-app-manifest-512x512.png'
];

// Precache shell on install
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(SHELL_CACHE).then((cache) => {
      return cache.addAll(SHELL_ASSETS);
    }).then(() => self.skipWaiting())
  );
});

// Clean up old caches on activate
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) => {
      return Promise.all(
        keys.map((key) => {
          if (key !== SHELL_CACHE && key !== DATA_CACHE) {
            return caches.delete(key);
          }
        })
      );
    }).then(() => self.clients.claim())
  );
});

// Handle fetch requests
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // 1. Only handle same-origin or fonts
  if (url.origin !== self.location.origin && !url.hostname.includes('fonts.g')) {
    return;
  }

  // 2. Bypass cache for mutations / auth / sync
  if (request.method !== 'GET') {
    return;
  }

  // 3. API Read Endpoints: Network-First with Cache Fallback
  if (url.pathname.startsWith('/api/items') || 
      url.pathname.startsWith('/api/tags') || 
      url.pathname.startsWith('/api/stats') ||
      url.pathname.startsWith('/api/notes/')) {
    event.respondWith(
      fetch(request)
        .then((response) => {
          if (response.ok) {
            const clone = response.clone();
            caches.open(DATA_CACHE).then((cache) => cache.put(request, clone));
          }
          return response;
        })
        .catch(async () => {
          const cached = await caches.match(request);
          if (cached) {
            return cached;
          }
          // Return offline JSON message
          return new Response(
            JSON.stringify({ offline: true, error: 'Offline mode: data not cached yet' }),
            {
              status: 503,
              headers: { 'Content-Type': 'application/json' }
            }
          );
        })
    );
    return;
  }

  // 4. Static assets & SPA navigation: Stale-While-Revalidate
  event.respondWith(
    caches.match(request).then((cachedResponse) => {
      const fetchPromise = fetch(request)
        .then((networkResponse) => {
          if (networkResponse && networkResponse.status === 200) {
            const clone = networkResponse.clone();
            caches.open(SHELL_CACHE).then((cache) => cache.put(request, clone));
          }
          return networkResponse;
        })
        .catch(() => {
          // If offline and navigating to a page, fallback to cached index.html
          if (request.mode === 'navigate') {
            return caches.match('/index.html') || caches.match('/');
          }
        });

      return cachedResponse || fetchPromise;
    })
  );
});

// Allow client to trigger immediate update activation
self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
});
