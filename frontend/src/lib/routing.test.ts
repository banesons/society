import { describe, it, expect, vi, beforeEach } from 'vitest';
import { auth, type UserProfile } from './auth.svelte';
import { api, ApiClientError } from './api/client';

interface TestAuth {
	user: unknown;
	isAuthenticated: boolean;
	isLoading: boolean;
	clearSession(): void;
}

const testAuth = auth as unknown as TestAuth;

import {
	resolvePostLoginRoute,
	getContextualBackDestination,
	sanitizeReturnTo,
	ROUTES,
	appRoutes
} from './routing';

describe('Frontend Routing & Authentication UX', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		testAuth.clearSession();
	});

	describe('Canonical App Route Constants & Builders', () => {
		it('defines standard /app route constants', () => {
			expect(ROUTES.APP).toBe('/app');
			expect(ROUTES.DASHBOARD).toBe('/app/dashboard');
			expect(ROUTES.SOCIETIES).toBe('/app/societies');
			expect(ROUTES.LOGIN).toBe('/app/login');
			expect(ROUTES.ADMIN).toBe('/app/admin');
		});

		it('builds canonical society route with appRoutes.society', () => {
			expect(appRoutes.society('soc-123')).toBe('/app/societies/soc-123');
			expect(appRoutes.society('soc-123', '#maintenance')).toBe(
				'/app/societies/soc-123#maintenance'
			);
			expect(appRoutes.society('soc-123', 'announcements')).toBe(
				'/app/societies/soc-123#announcements'
			);
		});

		it('builds canonical login with returnTo via appRoutes.login', () => {
			expect(appRoutes.login()).toBe('/app/login');
			expect(appRoutes.login('/app/dashboard')).toBe('/app/login?returnTo=%2Fapp%2Fdashboard');
		});
	});

	describe('Safe ReturnTo Redirection Handling', () => {
		it('accepts safe relative paths', () => {
			expect(sanitizeReturnTo('/app/dashboard')).toBe('/app/dashboard');
			expect(sanitizeReturnTo('/app/societies/soc-123#announcements')).toBe(
				'/app/societies/soc-123#announcements'
			);
			expect(sanitizeReturnTo('/developer')).toBe('/developer');
		});

		it('rejects protocol-relative and external URLs (open redirect defense)', () => {
			expect(sanitizeReturnTo('//evil.com')).toBeNull();
			expect(sanitizeReturnTo('https://evil.com/phish')).toBeNull();
			expect(sanitizeReturnTo('http://evil.com')).toBeNull();
			expect(sanitizeReturnTo('javascript:alert(1)')).toBeNull();
		});

		it('rejects recursive login return destinations', () => {
			expect(sanitizeReturnTo('/app/login')).toBeNull();
			expect(sanitizeReturnTo('/login')).toBeNull();
			expect(sanitizeReturnTo('/app/login?returnTo=/app/dashboard')).toBeNull();
		});

		it('handles null or empty inputs gracefully', () => {
			expect(sanitizeReturnTo(null)).toBeNull();
			expect(sanitizeReturnTo(undefined)).toBeNull();
			expect(sanitizeReturnTo('')).toBeNull();
			expect(sanitizeReturnTo('   ')).toBeNull();
		});
	});

	describe('Documented Post-Login Routing Resolution Policy', () => {
		it('redirects SUPER_ADMIN to /app/admin', () => {
			const user = {
				id: 'u-super',
				phone: '+91 9967xxxx01',
				status: 'ACTIVE',
				platform_roles: ['SUPER_ADMIN'],
				society_assignments: []
			};
			const res = resolvePostLoginRoute(user);
			expect(res).toEqual({ type: 'navigate', path: '/app/admin' });
		});

		it('redirects platform ADMIN to /app/admin without implicit society access', () => {
			const user = {
				id: 'u-admin',
				phone: '+91 8422xxxx01',
				status: 'ACTIVE',
				platform_roles: ['ADMIN'],
				society_assignments: []
			};
			const res = resolvePostLoginRoute(user);
			expect(res).toEqual({ type: 'navigate', path: '/app/admin' });
		});

		it('honors safe preferredReturnTo when provided upon login', () => {
			const user = {
				id: 'u-soc-admin',
				phone: '+91 9820xxxx56',
				status: 'ACTIVE',
				platform_roles: [],
				society_assignments: [
					{
						society_id: '0a2c153c-32ca-458e-a090-81a4687b7a76',
						society_code: 'SOC-PUN-GVR202',
						society_name: 'Greenview Residency',
						role: 'SOC_ADMIN',
						status: 'ACTIVE' as const
					}
				]
			};
			const res = resolvePostLoginRoute(
				user,
				'/app/societies/0a2c153c-32ca-458e-a090-81a4687b7a76#resident-maintenance'
			);
			expect(res).toEqual({
				type: 'navigate',
				path: '/app/societies/0a2c153c-32ca-458e-a090-81a4687b7a76#resident-maintenance'
			});
		});

		it('redirects unassigned user (0 accessible societies) to /unauthorized', () => {
			const user = {
				id: 'u-unassigned',
				phone: '+91 9999xxxx01',
				status: 'ACTIVE',
				platform_roles: [],
				society_assignments: []
			};
			const res = resolvePostLoginRoute(user);
			expect(res).toEqual({ type: 'unauthorized', reason: 'NO_ACCESSIBLE_SOCIETIES' });
		});

		it('navigates directly to society overview under /app/societies/:id when exactly 1 society is accessible', () => {
			const user = {
				id: 'u-soc-admin',
				phone: '+91 9820xxxx56',
				status: 'ACTIVE',
				platform_roles: [],
				society_assignments: [
					{
						society_id: '0a2c153c-32ca-458e-a090-81a4687b7a76',
						society_code: 'SOC-PUN-GVR202',
						society_name: 'Greenview Residency',
						role: 'SOC_ADMIN',
						status: 'ACTIVE' as const
					}
				]
			};
			const res = resolvePostLoginRoute(user);
			expect(res).toEqual({
				type: 'navigate',
				path: '/app/societies/0a2c153c-32ca-458e-a090-81a4687b7a76'
			});
		});

		it('presents society selector when multiple societies are accessible', () => {
			const assignments = [
				{
					society_id: '0a2c153c-32ca-458e-a090-81a4687b7a76',
					society_code: 'SOC-PUN-GVR202',
					society_name: 'Greenview Residency',
					role: 'SOC_ADMIN',
					status: 'ACTIVE' as const
				},
				{
					society_id: 'ecd1369f-c49c-45b9-b36d-68e1234f1042',
					society_code: 'SOC-MUM-SRM101',
					society_name: 'Shree Ram CHS',
					role: 'SOC_ADMIN',
					status: 'ACTIVE' as const
				}
			];
			const user = {
				id: 'u-multi-soc',
				phone: '+91 9820xxxx56',
				status: 'ACTIVE',
				platform_roles: [],
				society_assignments: assignments
			};
			const res = resolvePostLoginRoute(user);
			expect(res).toEqual({
				type: 'select_society',
				assignments
			});
		});

		it('resolves unauthenticated state to unauthorized', () => {
			const res = resolvePostLoginRoute(null);
			expect(res).toEqual({ type: 'unauthorized', reason: 'UNAUTHENTICATED' });
		});
	});

	describe('Contextual Back Navigation Destination', () => {
		it('returns /app/admin for Super Admin', () => {
			const user = {
				id: 'u-super',
				phone: '919967089101',
				status: 'ACTIVE',
				platform_roles: ['SUPER_ADMIN']
			};
			const dest = getContextualBackDestination(user);
			expect(dest.href).toBe('/app/admin');
			expect(dest.label).toBe('Return to Admin Console');
		});

		it('returns /app/admin for Platform Admin', () => {
			const user = {
				id: 'u-admin',
				phone: '918422862001',
				status: 'ACTIVE',
				platform_roles: ['ADMIN']
			};
			const dest = getContextualBackDestination(user);
			expect(dest.href).toBe('/app/admin');
			expect(dest.label).toBe('Return to Admin Console');
		});

		it('returns /app/dashboard (Back to Dashboard) for ordinary society administrators and members', () => {
			const user = {
				id: 'u-soc',
				phone: '919820000001',
				status: 'ACTIVE',
				platform_roles: []
			};
			const dest = getContextualBackDestination(user);
			expect(dest.href).toBe('/app/dashboard');
			expect(dest.label).toBe('Back to Dashboard');
		});

		it('returns /app/dashboard (Back to Dashboard) when navigating back from /admin or /app/admin', () => {
			const user = {
				id: 'u-super',
				phone: '919967089101',
				status: 'ACTIVE',
				platform_roles: ['SUPER_ADMIN']
			};
			const dest = getContextualBackDestination(user, '/admin');
			expect(dest.href).toBe('/app/dashboard');
			expect(dest.label).toBe('Back to Dashboard');
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

			await auth.verifyOTP('919967089101', '123456');

			expect(postSpy).toHaveBeenCalledWith('/auth/otp/verify', {
				phone: '919967089101',
				otp: '123456'
			});
			expect(auth.isAuthenticated).toBe(true);
			expect(auth.sessionReplaced).toBe(true);
			expect(auth.isSuperAdmin).toBe(true);

			// Target route proceeds to /app/admin
			const targetRoute = auth.isSuperAdmin ? '/app/admin' : '/unauthorized';
			expect(targetRoute).toBe('/app/admin');
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

			const targetRoute = auth.isSuperAdmin ? '/app/admin' : '/unauthorized';
			expect(targetRoute).toBe('/unauthorized');
		});
	});

	describe('Dedicated Gatekeeper Routing & Gate Console Paths', () => {
		it('defines appRoutes.gate and login builders', () => {
			expect(appRoutes.gate('soc-123')).toBe('/app/societies/soc-123/gate');
			expect(appRoutes.login('/app/societies/soc-1/gate')).toBe(
				'/app/login?returnTo=%2Fapp%2Fsocieties%2Fsoc-1%2Fgate'
			);
		});

		it('sanitizes /app/login to prevent self-loop', () => {
			expect(sanitizeReturnTo('/app/login')).toBeNull();
			expect(sanitizeReturnTo('/app/login?returnTo=/app/dashboard')).toBeNull();
		});

		it('routes single-assignment GATEKEEPER directly to gate console', () => {
			const user = {
				id: 'u-guard',
				phone: '+91 9876543210',
				status: 'ACTIVE',
				platform_roles: [],
				society_assignments: [
					{
						society_id: 'soc-greenview',
						society_name: 'Greenview Residency',
						role: 'GATEKEEPER'
					}
				]
			};

			const resolution = resolvePostLoginRoute(user as unknown as UserProfile);
			expect(resolution).toEqual({
				type: 'navigate',
				path: '/app/societies/soc-greenview/gate'
			});
		});

		it('routes ordinary resident to society hub', () => {
			const user = {
				id: 'u-resident',
				phone: '+91 9876543210',
				status: 'ACTIVE',
				platform_roles: [],
				society_assignments: [
					{
						society_id: 'soc-greenview',
						society_name: 'Greenview Residency',
						role: 'RESIDENT'
					}
				]
			};

			const resolution = resolvePostLoginRoute(user as unknown as UserProfile);
			expect(resolution).toEqual({
				type: 'navigate',
				path: '/app/societies/soc-greenview'
			});
		});
	});
});
