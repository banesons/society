import { describe, it, expect, vi, beforeEach } from 'vitest';
import { auth } from './auth.svelte';
import { api } from './api/client';

interface TestAuth {
	user: unknown;
	isAuthenticated: boolean;
	clearSession(): void;
}

const testAuth = auth as unknown as TestAuth;

describe('AuthStore', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		testAuth.clearSession();
	});

	it('initializes with unauthenticated default state', () => {
		expect(auth.isAuthenticated).toBe(false);
		expect(auth.user).toBeNull();
		expect(auth.isLoginOpen).toBe(false);
	});

	it('toggles login dialog modal state', () => {
		expect(auth.isLoginOpen).toBe(false);
		auth.openLogin();
		expect(auth.isLoginOpen).toBe(true);
		auth.closeLogin();
		expect(auth.isLoginOpen).toBe(false);
	});

	it('sendOTP calls api.post with phone and returns cooldown', async () => {
		const postSpy = vi.spyOn(api, 'post').mockResolvedValue({ resend_after_seconds: 60 });

		const res = await auth.sendOTP('919967089101');
		expect(res.resend_after_seconds).toBe(60);
		expect(postSpy).toHaveBeenCalledWith('/auth/otp/send', { phone: '919967089101' });
	});

	it('verifyOTP sets user profile and marks authenticated', async () => {
		const mockResult = {
			access_token: 'fake-jwt',
			refresh_token: 'fake-refresh',
			expires_in: 900,
			user: {
				id: 'usr-123',
				phone: '919967089101',
				status: 'ACTIVE'
			}
		};

		vi.spyOn(api, 'post').mockResolvedValue(mockResult);

		await auth.verifyOTP('919967089101', '123456');

		expect(auth.isAuthenticated).toBe(true);
		expect(auth.user).toEqual(mockResult.user);
		expect(auth.sessionReplaced).toBe(false);
	});

	it('verifyOTP sets sessionReplaced=true when reported by backend on 3rd login', async () => {
		const mockResult = {
			access_token: 'fake-jwt-3',
			refresh_token: 'fake-refresh-3',
			expires_in: 900,
			user: {
				id: 'usr-123',
				phone: '919967089101',
				status: 'ACTIVE'
			},
			session_replaced: true
		};

		vi.spyOn(api, 'post').mockResolvedValue(mockResult);

		await auth.verifyOTP('919967089101', '123456');

		expect(auth.isAuthenticated).toBe(true);
		expect(auth.sessionReplaced).toBe(true);

		// Dismiss notice
		auth.dismissSessionReplacedNotice();
		expect(auth.sessionReplaced).toBe(false);
	});

	it('logout clears session state and calls backend', async () => {
		const postSpy = vi.spyOn(api, 'post').mockResolvedValue({});

		// Set active state
		testAuth.user = { id: '1', phone: '919967089101', status: 'ACTIVE' };
		testAuth.isAuthenticated = true;
		auth.sessionReplaced = true;

		await auth.logout();

		expect(postSpy).toHaveBeenCalledWith('/auth/logout');
		expect(auth.isAuthenticated).toBe(false);
		expect(auth.user).toBeNull();
		expect(auth.sessionReplaced).toBe(false);
	});

	describe('Session Restoration & Reload Resilience (Phase 3)', () => {
		it('restores valid session on init() without racing route guards', async () => {
			const mockSessionStorage: Record<string, string> = {
				chs_refresh_token: 'valid-stored-refresh-token'
			};
			vi.stubGlobal('sessionStorage', {
				getItem: (key: string) => mockSessionStorage[key] ?? null,
				setItem: (key: string, val: string) => {
					mockSessionStorage[key] = val;
				},
				removeItem: (key: string) => {
					delete mockSessionStorage[key];
				}
			});

			const postSpy = vi.spyOn(api, 'post').mockResolvedValue({
				access_token: 'new-restored-access',
				refresh_token: 'new-rotated-refresh',
				expires_in: 900
			});

			const getSpy = vi.spyOn(api, 'get').mockResolvedValue({
				id: 'usr-restored',
				phone: '919967089101',
				status: 'ACTIVE',
				platform_roles: ['SUPER_ADMIN']
			});

			await auth.init();

			expect(postSpy).toHaveBeenCalledWith('/auth/refresh', {
				refresh_token: 'valid-stored-refresh-token'
			});
			expect(getSpy).toHaveBeenCalledWith('/auth/me');
			expect(auth.isAuthenticated).toBe(true);
			expect(auth.user?.id).toBe('usr-restored');
			expect(auth.isLoading).toBe(false);
			expect(auth.networkError).toBe(false);
		});

		it('deduplicates concurrent refresh requests to prevent token rotation race condition', async () => {
			const mockSessionStorage: Record<string, string> = {
				chs_refresh_token: 'race-prevention-token'
			};
			vi.stubGlobal('sessionStorage', {
				getItem: (key: string) => mockSessionStorage[key] ?? null,
				setItem: (key: string, val: string) => {
					mockSessionStorage[key] = val;
				},
				removeItem: (key: string) => {
					delete mockSessionStorage[key];
				}
			});

			// Setup refresh token on store
			(auth as unknown as { refreshToken: string }).refreshToken = 'race-prevention-token';

			let resolveRefresh: (res: unknown) => void;
			const refreshPromise = new Promise((resolve) => {
				resolveRefresh = resolve;
			});

			const postSpy = vi.spyOn(api, 'post').mockImplementation(() => refreshPromise as never);

			// Fire 3 simultaneous refresh operations (e.g. from 3 concurrent 401 API responses)
			const p1 = auth.refreshSession();
			const p2 = auth.refreshSession();
			const p3 = auth.refreshSession();

			// Ensure only ONE network call was dispatched
			expect(postSpy).toHaveBeenCalledTimes(1);

			// Resolve the single network call
			resolveRefresh!({
				access_token: 'shared-rotated-token',
				refresh_token: 'new-rotated-refresh',
				expires_in: 900
			});

			const [res1, res2, res3] = await Promise.all([p1, p2, p3]);

			expect(res1).toBe('shared-rotated-token');
			expect(res2).toBe('shared-rotated-token');
			expect(res3).toBe('shared-rotated-token');
			expect(postSpy).toHaveBeenCalledTimes(1);
		});

		it('preserves saved session and marks networkError=true on temporary network drop', async () => {
			const mockSessionStorage: Record<string, string> = {
				chs_refresh_token: 'network-drop-token'
			};
			vi.stubGlobal('sessionStorage', {
				getItem: (key: string) => mockSessionStorage[key] ?? null,
				setItem: (key: string, val: string) => {
					mockSessionStorage[key] = val;
				},
				removeItem: (key: string) => {
					delete mockSessionStorage[key];
				}
			});

			const { ApiClientError } = await import('./api/client');
			vi.spyOn(api, 'post').mockRejectedValue(
				new ApiClientError('Failed to fetch', 'NETWORK_ERROR', 0)
			);

			await auth.init();

			// Session storage must NOT be cleared on network drop!
			expect(mockSessionStorage['chs_refresh_token']).toBe('network-drop-token');
			expect(auth.isAuthenticated).toBe(false);
			expect(auth.networkError).toBe(true);
			expect(auth.isLoading).toBe(false);
		});

		it('clears session and sets sessionReplaced=true when server returns SESSION_REVOKED (401)', async () => {
			const mockSessionStorage: Record<string, string> = {
				chs_refresh_token: 'revoked-token'
			};
			vi.stubGlobal('sessionStorage', {
				getItem: (key: string) => mockSessionStorage[key] ?? null,
				setItem: (key: string, val: string) => {
					mockSessionStorage[key] = val;
				},
				removeItem: (key: string) => {
					delete mockSessionStorage[key];
				}
			});

			const { ApiClientError } = await import('./api/client');
			vi.spyOn(api, 'post').mockRejectedValue(
				new ApiClientError('Session revoked', 'SESSION_REVOKED', 401)
			);

			await auth.init();

			// Session storage MUST be purged on authoritative revocation
			expect(mockSessionStorage['chs_refresh_token']).toBeUndefined();
			expect(auth.isAuthenticated).toBe(false);
			expect(auth.sessionReplaced).toBe(true);
			expect(auth.user).toBeNull();
		});
	});
});
