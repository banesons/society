import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
	getFirebaseConfig,
	isFirebaseConfigured,
	getFirebaseApp,
	initPerformance,
	getPerformanceInstance,
	traceOperation
} from './firebase';

describe('Firebase & Performance Module', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it('reads firebase configuration from environment', () => {
		const config = getFirebaseConfig();
		expect(config).toHaveProperty('apiKey');
		expect(config).toHaveProperty('authDomain');
		expect(config).toHaveProperty('projectId');
		expect(config).toHaveProperty('appId');
	});

	it('checks if firebase is configured', () => {
		const isConfigured = isFirebaseConfigured();
		expect(typeof isConfigured).toBe('boolean');
	});

	it('returns null or app instance safely', () => {
		const app = getFirebaseApp();
		// In node test environment, if window is undefined, it returns null
		// or if mocked/configured, returns app instance
		if (typeof window === 'undefined') {
			expect(app).toBeNull();
		} else {
			expect(app === null || typeof app === 'object').toBe(true);
		}
	});

	it('initPerformance handles environment safely', async () => {
		const perf = await initPerformance();
		if (typeof window === 'undefined' || !isFirebaseConfigured()) {
			expect(perf).toBeNull();
		}
	});

	it('getPerformanceInstance returns cached instance or null', () => {
		const perf = getPerformanceInstance();
		expect(perf === null || typeof perf === 'object').toBe(true);
	});

	it('traceOperation executes wrapped callback and returns result', async () => {
		const mockResult = { data: 'test_success' };
		const result = await traceOperation('test_trace', async () => {
			return mockResult;
		});
		expect(result).toEqual(mockResult);
	});

	it('traceOperation propagates callback errors', async () => {
		const testError = new Error('simulated operation failure');
		await expect(
			traceOperation('failing_trace', async () => {
				throw testError;
			})
		).rejects.toThrow('simulated operation failure');
	});
});
