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

		const resend = await auth.sendOTP('+919967089101');
		expect(resend).toBe(60);
		expect(postSpy).toHaveBeenCalledWith('/auth/otp/send', { phone: '+919967089101' });
	});

	it('verifyOTP sets user profile and marks authenticated', async () => {
		const mockResult = {
			access_token: 'fake-jwt',
			refresh_token: 'fake-refresh',
			expires_in: 900,
			user: {
				id: 'usr-123',
				phone: '+91 9967xxxx01',
				status: 'ACTIVE'
			}
		};

		vi.spyOn(api, 'post').mockResolvedValue(mockResult);

		await auth.verifyOTP('+919967089101', '123456');

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
				phone: '+91 9967xxxx01',
				status: 'ACTIVE'
			},
			session_replaced: true
		};

		vi.spyOn(api, 'post').mockResolvedValue(mockResult);

		await auth.verifyOTP('+919967089101', '123456');

		expect(auth.isAuthenticated).toBe(true);
		expect(auth.sessionReplaced).toBe(true);

		// Dismiss notice
		auth.dismissSessionReplacedNotice();
		expect(auth.sessionReplaced).toBe(false);
	});

	it('logout clears session state and calls backend', async () => {
		const postSpy = vi.spyOn(api, 'post').mockResolvedValue({});

		// Set active state
		testAuth.user = { id: '1', phone: '+91 9967xxxx01', status: 'ACTIVE' };
		testAuth.isAuthenticated = true;
		auth.sessionReplaced = true;

		await auth.logout();

		expect(postSpy).toHaveBeenCalledWith('/auth/logout');
		expect(auth.isAuthenticated).toBe(false);
		expect(auth.user).toBeNull();
		expect(auth.sessionReplaced).toBe(false);
	});
});
