// Unified Firebase Cloud Messaging + Progressive Web App (PWA) Service Worker for CHS
// Handles background push events, offline caching, and asset stale-while-revalidate.

const CACHE_NAME = 'chs-pwa-v1';
const OFFLINE_URL = '/offline.html';

const PRECACHE_ASSETS = [
	OFFLINE_URL,
	'/manifest.webmanifest',
	'/favicon.svg',
	'/icons/icon-192.png',
	'/icons/icon-512.png',
	'/icons/icon.svg'
];

self.addEventListener('install', (event) => {
	event.waitUntil(
		caches.open(CACHE_NAME).then((cache) => {
			return cache.addAll(PRECACHE_ASSETS);
		})
	);
	self.skipWaiting();
});

self.addEventListener('activate', (event) => {
	event.waitUntil(
		caches.keys().then((cacheNames) => {
			return Promise.all(
				cacheNames.map((name) => {
					if (name !== CACHE_NAME) {
						return caches.delete(name);
					}
				})
			);
		}).then(() => self.clients.claim())
	);
});

// Fetch event: Network-first for navigations with offline fallback; stale-while-revalidate for static assets
self.addEventListener('fetch', (event) => {
	const request = event.request;

	// Only intercept GET requests
	if (request.method !== 'GET') {
		return;
	}

	const url = new URL(request.url);

	// Never intercept backend API requests
	if (url.pathname.startsWith('/api/') || url.pathname.startsWith('/auth/')) {
		return;
	}

	// 1. Navigation requests: Network first -> Offline HTML fallback
	if (request.mode === 'navigate') {
		event.respondWith(
			fetch(request).catch(async () => {
				const cache = await caches.open(CACHE_NAME);
				const cachedFallback = await cache.match(OFFLINE_URL);
				return cachedFallback || Response.error();
			})
		);
		return;
	}

	// 2. Static Fonts & Media: Stale-while-revalidate
	const isStaticAsset =
		url.origin === self.location.origin &&
		(url.pathname.startsWith('/icons/') ||
			url.pathname.endsWith('.svg') ||
			url.pathname.endsWith('.png') ||
			url.pathname.endsWith('.jpg') ||
			url.pathname.endsWith('.woff2') ||
			url.pathname.endsWith('.css'));

	const isGoogleFont =
		url.hostname === 'fonts.googleapis.com' || url.hostname === 'fonts.gstatic.com';

	if (isStaticAsset || isGoogleFont) {
		event.respondWith(
			caches.open(CACHE_NAME).then(async (cache) => {
				const cachedResponse = await cache.match(request);
				const fetchPromise = fetch(request)
					.then((networkResponse) => {
						if (networkResponse && networkResponse.status === 200) {
							cache.put(request, networkResponse.clone());
						}
						return networkResponse;
					})
					.catch(() => cachedResponse);

				return cachedResponse || fetchPromise;
			})
		);
	}
});

// Firebase Cloud Messaging (FCM) Background Push Notifications
self.addEventListener('push', (event) => {
	if (!event.data) return;

	try {
		const payload = event.data.json();
		const notification = payload.notification || {};
		const data = payload.data || {};

		const title = notification.title || data.title || 'CHS Notice';
		const options = {
			body: notification.body || data.body || 'You have a new update.',
			icon: '/icons/icon-192.png',
			badge: '/icons/icon-192.png',
			data: {
				action_url: data.action_url || '/',
				notification_id: data.notification_id
			}
		};

		event.waitUntil(self.registration.showNotification(title, options));
	} catch {
		// Non-JSON payload fallback
		const text = event.data.text();
		event.waitUntil(
			self.registration.showNotification('CHS Notice', {
				body: text,
				icon: '/icons/icon-192.png',
				badge: '/icons/icon-192.png'
			})
		);
	}
});

self.addEventListener('notificationclick', (event) => {
	event.notification.close();
	const targetUrl = (event.notification.data && event.notification.data.action_url) || '/';

	event.waitUntil(
		self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
			for (const client of clientList) {
				if (client.url && 'focus' in client) {
					client.navigate(targetUrl);
					return client.focus();
				}
			}
			if (self.clients.openWindow) {
				return self.clients.openWindow(targetUrl);
			}
		})
	);
});
