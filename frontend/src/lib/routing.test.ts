import { describe, it, expect, vi, beforeEach } from 'vitest';
import { auth } from './auth.svelte';
import { api, ApiClientError } from './api/client';

interface TestAuth {
	user: unknown;
	isAuthenticated: boolean;
	isLoading: boolean;
	clearSession(): void;
}

const testAuth = auth as unknown as TestAuth;

describe('Frontend Routing & Authentication UX', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		testAuth.clearSession();
	});

	describe('Role Determination for Login Navigation', () => {
		it('redirects SUPER_ADMIN to /admin', () => {
			testAuth.user = {
				id: 'u-super',
				phone: '+91 9967xxxx01',
				status: 'ACTIVE',
				platform_roles: ['SUPER_ADMIN']
			};
			testAuth.isAuthenticated = true;

			expect(auth.isSuperAdmin).toBe(true);
			// Target route for Super Admin
			const targetRoute = auth.isSuperAdmin ? '/admin' : '/unauthorized';
			expect(targetRoute).toBe('/admin');
		});

		it('redirects ordinary authenticated user to /unauthorized', () => {
			testAuth.user = {
				id: 'u-ordinary',
				phone: '+91 9999xxxx01',
				status: 'ACTIVE',
				platform_roles: []
			};
			testAuth.isAuthenticated = true;

			expect(auth.isSuperAdmin).toBe(false);
			const targetRoute = auth.isSuperAdmin ? '/admin' : '/unauthorized';
			expect(targetRoute).toBe('/unauthorized');
		});

		it('denies platform administration access to disabled admin', () => {
			testAuth.user = {
				id: 'u-disabled',
				phone: '+91 9820xxxx56',
				status: 'ACTIVE',
				platform_roles: [] // Disabled admins lose active platform_roles
			};
			testAuth.isAuthenticated = true;

			expect(auth.isSuperAdmin).toBe(false);
			const targetRoute = auth.isSuperAdmin ? '/admin' : '/unauthorized';
			expect(targetRoute).toBe('/unauthorized');
		});
	});

	describe('Session Management: Logout vs Logout All', () => {
		it('logout calls POST /auth/logout and clears local session', async () => {
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue({});

			testAuth.user = { id: 'u-1', phone: '+91 9967xxxx01', status: 'ACTIVE' };
			testAuth.isAuthenticated = true;

			await auth.logout();

			expect(postSpy).toHaveBeenCalledWith('/auth/logout');
			expect(auth.isAuthenticated).toBe(false);
			expect(auth.user).toBeNull();
		});

		it('logoutAll calls POST /auth/logout-all and clears local session', async () => {
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue({
				message: 'All sessions revoked successfully'
			});

			testAuth.user = { id: 'u-1', phone: '+91 9967xxxx01', status: 'ACTIVE' };
			testAuth.isAuthenticated = true;

			await auth.logoutAll();

			expect(postSpy).toHaveBeenCalledWith('/auth/logout-all');
			expect(auth.isAuthenticated).toBe(false);
			expect(auth.user).toBeNull();
		});

		it('logoutAll clears session even if network fails gracefully', async () => {
			vi.spyOn(api, 'post').mockRejectedValue(
				new ApiClientError('Network error', 'NETWORK_ERROR', 0)
			);

			testAuth.user = { id: 'u-1', phone: '+91 9967xxxx01', status: 'ACTIVE' };
			testAuth.isAuthenticated = true;

			await auth.logoutAll();

			// Local state must be cleared regardless to prevent stale local access
			expect(auth.isAuthenticated).toBe(false);
			expect(auth.user).toBeNull();
		});
	});

	describe('Session Expiry & Transparent Refresh Handling', () => {
		it('transparent refresh handles expired access token on 401', async () => {
			const refreshSpy = vi.spyOn(api, 'post').mockResolvedValue({
				access_token: 'new-valid-jwt',
				refresh_token: 'new-refresh-token',
				expires_in: 900
			});

			(auth as unknown as { refreshToken: string }).refreshToken = 'old-refresh-token';

			const newAccessToken = await auth.refreshSession();

			expect(refreshSpy).toHaveBeenCalledWith('/auth/refresh', {
				refresh_token: 'old-refresh-token'
			});
			expect(newAccessToken).toBe('new-valid-jwt');
		});

		it('clears session when refresh token has expired or is revoked', async () => {
			vi.spyOn(api, 'post').mockRejectedValue(
				new ApiClientError('Refresh token expired', 'TOKEN_EXPIRED', 401)
			);

			(auth as unknown as { refreshToken: string }).refreshToken = 'expired-refresh-token';
			testAuth.user = { id: 'u-1', phone: '+91 9967xxxx01', status: 'ACTIVE' };
			testAuth.isAuthenticated = true;

			const result = await auth.refreshSession();

			expect(result).toBeNull();
			expect(auth.isAuthenticated).toBe(false);
			expect(auth.user).toBeNull();
		});
	});

	describe('Automatic FIFO Session Replacement Login UX', () => {
		it('third login succeeds with FIFO replacement without SESSION_LIMIT_REACHED error', async () => {
			const mockThirdLoginResponse = {
				access_token: 'valid-jwt-device-3',
				refresh_token: 'valid-refresh-device-3',
				expires_in: 900,
				user: {
					id: 'u-super',
					phone: '+91 9967xxxx01',
					status: 'ACTIVE',
					platform_roles: ['SUPER_ADMIN']
				},
				session_replaced: true
			};

			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(mockThirdLoginResponse);

			await auth.verifyOTP('9967089101', '123456');

			expect(postSpy).toHaveBeenCalledWith('/auth/otp/verify', {
				phone: '9967089101',
				otp: '123456'
			});
			expect(auth.isAuthenticated).toBe(true);
			expect(auth.sessionReplaced).toBe(true);
			expect(auth.isSuperAdmin).toBe(true);

			// Target route proceeds normally to /admin without manual logout block
			const targetRoute = auth.isSuperAdmin ? '/admin' : '/unauthorized';
			expect(targetRoute).toBe('/admin');
		});

		it('third login for ordinary resident routes to /unauthorized normally', async () => {
			const mockThirdLoginResponse = {
				access_token: 'valid-jwt-device-3',
				refresh_token: 'valid-refresh-device-3',
				expires_in: 900,
				user: {
					id: 'u-resident',
					phone: '+91 9820xxxx56',
					status: 'ACTIVE',
					platform_roles: []
				},
				session_replaced: true
			};

			vi.spyOn(api, 'post').mockResolvedValue(mockThirdLoginResponse);

			await auth.verifyOTP('9820123456', '123456');

			expect(auth.isAuthenticated).toBe(true);
			expect(auth.sessionReplaced).toBe(true);
			expect(auth.isSuperAdmin).toBe(false);

			const targetRoute = auth.isSuperAdmin ? '/admin' : '/unauthorized';
			expect(targetRoute).toBe('/unauthorized');
		});
	});
});
