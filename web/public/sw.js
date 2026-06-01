const CACHE = {
  STATIC: 'masaar-static-v2',
  SHELL: 'masaar-shell-v2',
  API: 'masaar-api-v2',
  PAGES: 'masaar-pages-v2',
}

const STATIC_EXT = /\.(js|css|woff2?|svg|png|jpg|jpeg|gif|ico|json)$/

const SYNC_CACHE = 'masaar-sync-queue'
const SYNC_TAG = 'masaar-sync'

// ── Install ──────────────────────────────────────────────────────────────────
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE.SHELL).then((cache) =>
      cache.addAll(['/offline.html', '/manifest.json'])
    )
  )
  self.skipWaiting()
})

// ── Activate ─────────────────────────────────────────────────────────────────
self.addEventListener('activate', (event) => {
  event.waitUntil(
    (async () => {
      // Clean old caches
      const keys = await caches.keys()
      const keep = Object.values(CACHE)
      await Promise.all(keys.filter((k) => !keep.includes(k)).map((k) => caches.delete(k)))

      // Enable navigation preload if supported
      if (self.registration.navigationPreload) {
        await self.registration.navigationPreload.enable()
      }

      self.clients.claim()
    })()
  )
})

// ── Fetch ────────────────────────────────────────────────────────────────────
self.addEventListener('fetch', (event) => {
  const { request } = event
  const url = new URL(request.url)

  // Skip non-http(s) requests
  if (!url.protocol.startsWith('http')) return

  // ── Static assets (JS, CSS, fonts, images) → Cache-first ──────────────────
  if (STATIC_EXT.test(url.pathname) || url.pathname.startsWith('/_next/static')) {
    event.respondWith(cacheFirst(request, CACHE.STATIC))
    return
  }

  // ── API GET → Stale-while-revalidate ──────────────────────────────────────
  if (request.method === 'GET' && url.pathname.startsWith('/api/')) {
    event.respondWith(staleWhileRevalidate(request, CACHE.API))
    return
  }

  // ── API mutations (POST, PUT, PATCH, DELETE) ───────────────────────────────
  if (request.method !== 'GET' && url.pathname.startsWith('/api/')) {
    event.respondWith(networkOrQueue(event))
    return
  }

  // ── Navigation → Network-first with offline fallback ──────────────────────
  if (request.mode === 'navigate') {
    event.respondWith(navigateOrFallback(event))
    return
  }

  // ── Everything else → Network-first ───────────────────────────────────────
  event.respondWith(networkFirst(request, CACHE.PAGES))
})

// ── Background Sync ──────────────────────────────────────────────────────────
self.addEventListener('sync', (event) => {
  if (event.tag === SYNC_TAG) {
    event.waitUntil(replaySyncQueue())
  }
})

// ── Message handler (client-triggered sync) ─────────────────────────────────
self.addEventListener('message', (event) => {
  if (event.data?.type === 'SYNC_NOW') {
    event.waitUntil(replaySyncQueue())
  }
  if (event.data?.type === 'SKIP_WAITING') {
    self.skipWaiting()
  }
  if (event.data?.type === 'GET_QUEUE_COUNT') {
    event.waitUntil(
      (async () => {
        const entries = await getSyncQueue()
        const client = await self.clients.get(event.source?.id || '')
        if (client) {
          client.postMessage({ type: 'QUEUE_COUNT', count: entries.length })
        }
      })()
    )
  }
})

// ── Caching strategies ──────────────────────────────────────────────────────

async function cacheFirst(request, cacheName) {
  const cached = await caches.match(request)
  if (cached) return cached
  try {
    const response = await fetch(request)
    if (response.ok) {
      const cache = await caches.open(cacheName)
      cache.put(request, response.clone())
    }
    return response
  } catch {
    return caches.match('/offline.html')
  }
}

async function staleWhileRevalidate(request, cacheName) {
  const cache = await caches.open(cacheName)
  const cached = await cache.match(request)

  const fetchPromise = fetch(request)
    .then((response) => {
      if (response.ok) cache.put(request, response.clone())
      return response
    })
    .catch(() => cached)

  return cached || fetchPromise
}

async function networkFirst(request, cacheName) {
  try {
    const response = await fetch(request)
    if (response.ok) {
      const cache = await caches.open(cacheName)
      cache.put(request, response.clone())
    }
    return response
  } catch {
    const cached = await caches.match(request)
    return cached || caches.match('/offline.html')
  }
}

async function networkOrQueue(event) {
  const { request } = event
  try {
    const response = await fetch(request.clone())
    return response
  } catch {
    // Queue the mutation for sync when back online
    await addToSyncQueue(request)
    return new Response(
      JSON.stringify({ queued: true, message: 'Request queued for sync when online' }),
      { status: 202, headers: { 'Content-Type': 'application/json' } }
    )
  }
}

async function navigateOrFallback(event) {
  // Try navigation preload first
  if (event.preloadResponse) {
    const preload = await event.preloadResponse.catch(() => null)
    if (preload) return preload
  }

  try {
    const response = await fetch(event.request)
    // Cache pages for offline navigation
    if (response.ok && response.type === 'basic') {
      const cache = await caches.open(CACHE.PAGES)
      cache.put(event.request, response.clone())
    }
    return response
  } catch {
    const cached = await caches.match(event.request)
    return cached || caches.match('/offline.html')
  }
}

// ── Sync Queue (IndexedDB-based) ─────────────────────────────────────────────

async function openSyncDB() {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(SYNC_CACHE, 1)
    req.onupgradeneeded = () => {
      const db = req.result
      if (!db.objectStoreNames.contains('requests')) {
        db.createObjectStore('requests', { keyPath: 'id', autoIncrement: true })
      }
    }
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

async function addToSyncQueue(request) {
  const body = await request.clone().text().catch(() => '')
  const entry = {
    url: request.url,
    method: request.method,
    headers: Object.fromEntries(request.headers.entries()),
    body: body || undefined,
    createdAt: Date.now(),
    retries: 0,
  }

  const db = await openSyncDB()
  const tx = db.transaction('requests', 'readwrite')
  tx.objectStore('requests').add(entry)
  await tx.done

  // Register a sync event (supported in Chromium-based browsers)
  if ('sync' in self.registration) {
    self.registration.sync.register(SYNC_TAG).catch(() => {})
  }

  // Also notify all clients so they can show a pending indicator
  const clients = await self.clients.matchAll()
  clients.forEach((client) =>
    client.postMessage({ type: 'QUEUED', url: request.url })
  )
}

async function getSyncQueue() {
  const db = await openSyncDB()
  const tx = db.transaction('requests', 'readonly')
  return new Promise((resolve) => {
    const result = tx.objectStore('requests').getAll()
    result.onsuccess = () => resolve(result.result || [])
    result.onerror = () => resolve([])
  })
}

async function replaySyncQueue() {
  const entries = await getSyncQueue()
  if (entries.length === 0) return

  const db = await openSyncDB()
  const results = { synced: 0, failed: 0 }

  for (const entry of entries) {
    try {
      const body = entry.body ? JSON.parse(entry.body) : undefined
      const response = await fetch(entry.url, {
        method: entry.method,
        headers: { ...entry.headers, 'X-Sync-Replay': 'true' },
        body: body ? JSON.stringify(body) : undefined,
      })

      if (response.ok) {
        const tx = db.transaction('requests', 'readwrite')
        tx.objectStore('requests').delete(entry.id)
        await tx.done
        results.synced++
      } else if (entry.retries >= 5) {
        // Drop after 5 failed retries
        const tx = db.transaction('requests', 'readwrite')
        tx.objectStore('requests').delete(entry.id)
        await tx.done
        results.failed++
      } else {
        // Increment retry count
        const tx = db.transaction('requests', 'readwrite')
        const store = tx.objectStore('requests')
        const record = await store.get(entry.id)
        if (record) {
          record.retries = (record.retries || 0) + 1
          store.put(record)
        }
        await tx.done
        results.failed++
      }
    } catch {
      results.failed++
    }
  }

  // Notify clients of sync result
  const clients = await self.clients.matchAll()
  clients.forEach((client) => client.postMessage({ type: 'SYNC_RESULT', ...results }))

  // If there are still pending entries, re-register sync
  const remaining = await getSyncQueue()
  if (remaining.length > 0 && 'sync' in self.registration) {
    self.registration.sync.register(SYNC_TAG).catch(() => {})
  }
}

// ── Periodic Background Sync (check for updates) ─────────────────────────────
self.addEventListener('periodicsync', (event) => {
  if (event.tag === 'masaar-update-check') {
    event.waitUntil(checkForUpdates())
  }
})

async function checkForUpdates() {
  // Invalidate API cache so next navigation fetches fresh data
  const cache = await caches.open(CACHE.API)
  const keys = await cache.keys()
  // Revalidate only the most important endpoints
  const priorityPaths = ['/api/v1/stats', '/api/v1/notifications']
  for (const req of keys) {
    if (priorityPaths.some((p) => req.url.includes(p))) {
      cache.delete(req)
    }
  }
}
