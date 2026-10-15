import { env } from '$env/dynamic/public';
import { notificationsApi } from '$lib/api/notifications';
import { getFirebaseApp } from '$lib/firebase';
import type { Messaging } from 'firebase/messaging';

let fcmMessagingInstance: Messaging | null = null;

export function isPushSupported(): boolean {
	if (typeof window === 'undefined') return false;
	return 'Notification' in window && 'serviceWorker' in navigator && 'PushManager' in window;
}

export function getNotificationPermission(): NotificationPermission | 'unsupported' {
	if (!isPushSupported()) return 'unsupported';
	return Notification.permission;
}

export function getOrCreateDeviceId(): string {
	if (typeof window === 'undefined') return 'unknown_device';
	let deviceId = localStorage.getItem('chs_device_id');
	if (!deviceId) {
		deviceId = 'web_' + Math.random().toString(36).substring(2, 12) + '_' + Date.now();
		localStorage.setItem('chs_device_id', deviceId);
	}
	return deviceId;
}

async function getFirebaseMessaging() {
	if (fcmMessagingInstance) return fcmMessagingInstance;
	if (typeof window === 'undefined') return null;

	const app = getFirebaseApp();
	if (!app) return null;

	const { getMessaging } = await import('firebase/messaging');
	fcmMessagingInstance = getMessaging(app);
	return fcmMessagingInstance;
}

export async function requestPushPermission(): Promise<NotificationPermission> {
	if (!isPushSupported()) {
		throw new Error('Push notifications are not supported in this browser');
	}
	const permission = await Notification.requestPermission();
	return permission;
}

export async function registerPushDevice(): Promise<{
	token: string | null;
	status: 'granted' | 'denied' | 'unsupported' | 'error';
	error?: string;
}> {
	if (!isPushSupported()) {
		return { token: null, status: 'unsupported' };
	}

	try {
		const permission = await requestPushPermission();
		if (permission !== 'granted') {
			return { token: null, status: permission as 'denied' };
		}

		// Register service worker if not already registered
		const registration = await navigator.serviceWorker.register('/firebase-messaging-sw.js');
		await navigator.serviceWorker.ready;

		const messaging = await getFirebaseMessaging();
		if (!messaging) {
			return { token: null, status: 'error', error: 'Firebase messaging unavailable' };
		}

		const { getToken } = await import('firebase/messaging');
		const vapidKey =
			env.PUBLIC_FIREBASE_WEB_PUSH_VAPID_PUBLIC_KEY ||
			(env as Record<string, string | undefined>)['FIREBASE_WEB_PUSH_VAPID_PUBLIC_KEY'] ||
			'';

		// If vapid key is missing, mock or return error
		if (!vapidKey || vapidKey.startsWith('<')) {
			const mockToken = 'mock_web_fcm_token_' + getOrCreateDeviceId();
			await notificationsApi.registerDevice({
				device_id: getOrCreateDeviceId(),
				fcm_token: mockToken,
				platform: 'WEB'
			});
			return { token: mockToken, status: 'granted' };
		}

		const token = await getToken(messaging, {
			vapidKey,
			serviceWorkerRegistration: registration
		});

		if (token) {
			await notificationsApi.registerDevice({
				device_id: getOrCreateDeviceId(),
				fcm_token: token,
				platform: 'WEB'
			});
			return { token, status: 'granted' };
		}

		return { token: null, status: 'error', error: 'No registration token returned' };
	} catch (err: unknown) {
		const msg = err instanceof Error ? err.message : String(err);
		return { token: null, status: 'error', error: msg };
	}
}

export async function deactivateCurrentDevice(): Promise<void> {
	if (typeof window === 'undefined') return;
	const deviceId = localStorage.getItem('chs_device_id');
	if (deviceId) {
		try {
			await notificationsApi.deactivateDevice(deviceId);
		} catch {
			// Ignore network error on deactivation
		}
	}
}
