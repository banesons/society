import { initializeApp, getApps, getApp, type FirebaseApp } from 'firebase/app';
import { getPerformance, trace, type FirebasePerformance } from 'firebase/performance';
import { env } from '$env/dynamic/public';

let performanceInstance: FirebasePerformance | null = null;

export function getFirebaseConfig() {
	return {
		apiKey: env.PUBLIC_FIREBASE_API_KEY || '',
		authDomain: env.PUBLIC_FIREBASE_AUTH_DOMAIN || '',
		projectId: env.PUBLIC_FIREBASE_PROJECT_ID || 'chsneu-2021',
		storageBucket: env.PUBLIC_FIREBASE_STORAGE_BUCKET || '',
		messagingSenderId: env.PUBLIC_FIREBASE_MESSAGING_SENDER_ID || '799965614182',
		appId: env.PUBLIC_FIREBASE_APP_ID || '',
		measurementId: env.PUBLIC_FIREBASE_MEASUREMENT_ID || ''
	};
}

export function isFirebaseConfigured(): boolean {
	const config = getFirebaseConfig();
	return Boolean(config.apiKey && config.projectId && config.appId);
}

export function getFirebaseApp(): FirebaseApp | null {
	if (typeof window === 'undefined') {
		return null;
	}

	try {
		if (getApps().length > 0) {
			return getApp();
		}

		const config = getFirebaseConfig();
		if (!config.apiKey || !config.projectId || !config.appId) {
			return null;
		}

		return initializeApp(config);
	} catch (err) {
		console.warn('[Firebase] Initialization error:', err);
		return null;
	}
}

export async function initPerformance(): Promise<FirebasePerformance | null> {
	if (performanceInstance) {
		return performanceInstance;
	}

	if (typeof window === 'undefined') {
		return null;
	}

	const app = getFirebaseApp();
	if (!app) {
		return null;
	}

	try {
		performanceInstance = getPerformance(app);
		return performanceInstance;
	} catch (err) {
		console.warn('[Firebase Performance] Failed to initialize performance monitoring:', err);
		return null;
	}
}

export function getPerformanceInstance(): FirebasePerformance | null {
	return performanceInstance;
}

export async function traceOperation<T>(
	traceName: string,
	operation: () => Promise<T>
): Promise<T> {
	const perf = performanceInstance;
	if (!perf || typeof window === 'undefined') {
		return operation();
	}

	try {
		const t = trace(perf, traceName);
		t.start();
		try {
			const result = await operation();
			t.stop();
			return result;
		} catch (err) {
			t.putAttribute('error', 'true');
			t.stop();
			throw err;
		}
	} catch {
		return operation();
	}
}
