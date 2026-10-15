// Firebase Cloud Messaging Service Worker for CHS
// Handles background push events when the application is not in active focus.

self.addEventListener('install', () => {
	self.skipWaiting();
});

self.addEventListener('activate', (event) => {
	event.waitUntil(self.clients.claim());
});

self.addEventListener('push', (event) => {
	if (!event.data) return;

	try {
		const payload = event.data.json();
		const notification = payload.notification || {};
		const data = payload.data || {};

		const title = notification.title || data.title || 'CHS Notice';
		const options = {
			body: notification.body || data.body || 'You have a new update.',
			icon: '/favicon.svg',
			badge: '/favicon.svg',
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
				icon: '/favicon.svg'
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
