import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ROUTES, resolvePostLoginRoute, getContextualBackDestination } from './routing';
import { auth } from './auth.svelte';
import { toast } from './toast.svelte';
import { api } from './api/client';

describe('Application Shell & UX Refactor Integration', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		toast.clear();
		(auth as unknown as { clearSession(): void }).clearSession();
	});

	describe('Canonical Route Structure & Constants', () => {
		it('defines standard public and authenticated application routes', () => {
			expect(ROUTES.HOME).toBe('/');
			expect(ROUTES.LOGIN).toBe('/app/login');
			expect(ROUTES.APP).toBe('/app');
			expect(ROUTES.DASHBOARD).toBe('/app/dashboard');
			expect(ROUTES.SOCIETIES).toBe('/app/societies');
			expect(ROUTES.ADMIN).toBe('/app/admin');
			expect(ROUTES.DEVELOPER).toBe('/developer');
			expect(ROUTES.UNAUTHORIZED).toBe('/unauthorized');
		});

		it('ensures post-login routing never defaults to public marketing home', () => {
			const superUser = {
				id: 'u-1',
				phone: '919967089101',
				status: 'ACTIVE',
				platform_roles: ['SUPER_ADMIN']
			};
			const res = resolvePostLoginRoute(superUser);
			expect(res.type).toBe('navigate');
			if (res.type === 'navigate') {
				expect(res.path).not.toBe('/');
				expect(res.path).toBe(ROUTES.ADMIN);
			}
		});

		it('ensures contextual back navigation from /admin returns to /app/dashboard (Back to Dashboard)', () => {
			const superUser = {
				id: 'u-1',
				phone: '919967089101',
				status: 'ACTIVE',
				platform_roles: ['SUPER_ADMIN']
			};
			const dest = getContextualBackDestination(superUser, '/admin');
			expect(dest.href).toBe('/app/dashboard');
			expect(dest.label).toBe('Back to Dashboard');
		});

		it('ensures contextual back navigation from /app/admin returns to /app/dashboard (Back to Dashboard)', () => {
			const superUser = {
				id: 'u-1',
				phone: '919967089101',
				status: 'ACTIVE',
				platform_roles: ['SUPER_ADMIN']
			};
			const dest = getContextualBackDestination(superUser, '/app/admin');
			expect(dest.href).toBe('/app/dashboard');
			expect(dest.label).toBe('Back to Dashboard');
		});

		it('ensures ordinary society members navigate back to /app/dashboard instead of marketing homepage', () => {
			const member = {
				id: 'u-2',
				phone: '919820000001',
				status: 'ACTIVE',
				platform_roles: []
			};
			const dest = getContextualBackDestination(member);
			expect(dest.href).toBe('/app/dashboard');
			expect(dest.label).toBe('Back to Dashboard');
		});
	});

	describe('Session Replacement Feedback via Toast System', () => {
		it('triggers a concise info toast only when session_replaced is true', async () => {
			vi.spyOn(api, 'post').mockResolvedValue({
				access_token: 'fake-access',
				refresh_token: 'fake-refresh',
				expires_in: 900,
				session_replaced: true,
				user: {
					id: 'u-replaced',
					phone: '+919967089101',
					status: 'ACTIVE',
					platform_roles: ['SUPER_ADMIN']
				}
			});

			await auth.verifyOTP('9967089101', '123456');

			expect(auth.sessionReplaced).toBe(true);
			expect(toast.toasts).toHaveLength(1);
			expect(toast.toasts[0].type).toBe('info');
			expect(toast.toasts[0].message).toBe(
				'Signed in successfully. Your oldest session was signed out.'
			);
		});

		it('does NOT trigger a session replacement toast during normal login', async () => {
			vi.spyOn(api, 'post').mockResolvedValue({
				access_token: 'fake-access',
				refresh_token: 'fake-refresh',
				expires_in: 900,
				session_replaced: false,
				user: {
					id: 'u-normal',
					phone: '+919967089101',
					status: 'ACTIVE',
					platform_roles: ['SUPER_ADMIN']
				}
			});

			await auth.verifyOTP('9967089101', '123456');

			expect(auth.sessionReplaced).toBe(false);
			expect(toast.toasts).toHaveLength(0);
		});
	});

	describe('Responsive Shell & Viewport Classification (Phase 1)', () => {
		it('correctly classifies mobile viewports at 320x651 and 375x812 below 768px', async () => {
			const { screen, DeviceType } = await import('./screen.svelte');

			// Test mobile viewport 320x651 (reported defect dimension)
			screen.width = 320;
			expect(screen.isMobile).toBe(true);
			expect(screen.isTablet).toBe(false);
			expect(screen.isDesktop).toBe(false);
			expect(screen.device).toBe(DeviceType.Mobile);

			// Test mobile viewport 375x812 (standard iPhone)
			screen.width = 375;
			expect(screen.isMobile).toBe(true);
			expect(screen.isTablet).toBe(false);
			expect(screen.isDesktop).toBe(false);
			expect(screen.device).toBe(DeviceType.Mobile);
		});

		it('correctly classifies tablet and desktop viewports at 768x1024, 1024x768, and 1440x900', async () => {
			const { screen, DeviceType } = await import('./screen.svelte');

			// Test tablet viewport 768x1024 (iPad portrait, threshold of desktop sidebar)
			screen.width = 768;
			expect(screen.isMobile).toBe(false);
			expect(screen.isTablet).toBe(true);
			expect(screen.isDesktop).toBe(false);
			expect(screen.device).toBe(DeviceType.Tablet);

			// Test tablet/desktop threshold 1024x768 (iPad landscape)
			screen.width = 1024;
			expect(screen.isMobile).toBe(false);
			expect(screen.isTablet).toBe(false);
			expect(screen.isDesktop).toBe(true);
			expect(screen.device).toBe(DeviceType.Desktop);

			// Test desktop viewport 1440x900 (MacBook / large desktop)
			screen.width = 1440;
			expect(screen.isMobile).toBe(false);
			expect(screen.isTablet).toBe(false);
			expect(screen.isDesktop).toBe(true);
			expect(screen.device).toBe(DeviceType.Desktop);
		});
	});
});
