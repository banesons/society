import { describe, it, expect, vi, beforeEach } from 'vitest';
import { platformApi } from './platform';
import { api, ApiClientError } from './client';
import { auth } from '../auth.svelte';

interface TestAuth {
	user: unknown;
	isAuthenticated: boolean;
	clearSession(): void;
}

const testAuth = auth as unknown as TestAuth;

describe('Platform Administration Frontend API & State', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		testAuth.clearSession();
	});

	describe('Role Detection & Authorization State', () => {
		it('detects unauthenticated state', () => {
			expect(auth.isAuthenticated).toBe(false);
			expect(auth.isSuperAdmin).toBe(false);
			expect(auth.isPlatformAdmin).toBe(false);
		});

		it('identifies ordinary user without platform role', () => {
			testAuth.user = {
				id: 'u-1',
				phone: '+91 9999xxxx01',
				status: 'ACTIVE',
				platform_roles: []
			};
			testAuth.isAuthenticated = true;

			expect(auth.isAuthenticated).toBe(true);
			expect(auth.isSuperAdmin).toBe(false);
			expect(auth.isPlatformAdmin).toBe(false);
		});

		it('identifies active SUPER_ADMIN role correctly', () => {
			testAuth.user = {
				id: 'u-super',
				phone: '+91 9967xxxx01',
				status: 'ACTIVE',
				platform_roles: ['SUPER_ADMIN']
			};
			testAuth.isAuthenticated = true;

			expect(auth.isAuthenticated).toBe(true);
			expect(auth.isSuperAdmin).toBe(true);
			expect(auth.isPlatformAdmin).toBe(true);
		});

		it('identifies regular ADMIN role without Super Admin access', () => {
			testAuth.user = {
				id: 'u-admin',
				phone: '+91 9820xxxx56',
				status: 'ACTIVE',
				platform_roles: ['ADMIN']
			};
			testAuth.isAuthenticated = true;

			expect(auth.isAuthenticated).toBe(true);
			expect(auth.isSuperAdmin).toBe(false);
			expect(auth.isPlatformAdmin).toBe(true);
		});
	});

	describe('platformApi.listAdmins', () => {
		it('calls GET /platform/admins with query parameters', async () => {
			const mockResponse = {
				items: [
					{
						id: 'r-1',
						user_id: 'u-1',
						phone: '+91 9967xxxx01',
						role: 'SUPER_ADMIN',
						status: 'ACTIVE',
						created_at: '2026-10-10T10:00:00Z',
						updated_at: '2026-10-10T10:00:00Z'
					}
				],
				total: 1,
				limit: 10,
				offset: 0
			};

			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockResponse);

			const result = await platformApi.listAdmins({
				limit: 10,
				offset: 0,
				role: 'SUPER_ADMIN',
				status: 'ACTIVE'
			});

			expect(result).toEqual(mockResponse);
			expect(getSpy).toHaveBeenCalledWith('/platform/admins', {
				params: {
					limit: 10,
					offset: 0,
					role: 'SUPER_ADMIN',
					status: 'ACTIVE'
				}
			});
		});

		it('propagates API client errors on network failure', async () => {
			vi.spyOn(api, 'get').mockRejectedValue(
				new ApiClientError('Forbidden', 'FORBIDDEN', 403, 'req-123')
			);

			await expect(platformApi.listAdmins()).rejects.toThrow('Forbidden');
		});
	});

	describe('platformApi.createAdmin', () => {
		it('calls POST /platform/admins with phone payload', async () => {
			const mockResponse = {
				id: 'r-2',
				user_id: 'u-2',
				phone: '+91 9820xxxx56',
				role: 'ADMIN',
				status: 'ACTIVE',
				created_at: '2026-10-10T14:00:00Z'
			};

			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(mockResponse);

			const result = await platformApi.createAdmin('9820123456');

			expect(result).toEqual(mockResponse);
			expect(postSpy).toHaveBeenCalledWith('/platform/admins', { phone: '9820123456' });
		});

		it('handles duplicate admin conflict error (409 ADMIN_ALREADY_EXISTS)', async () => {
			vi.spyOn(api, 'post').mockRejectedValue(
				new ApiClientError(
					'User already holds a platform role',
					'ADMIN_ALREADY_EXISTS',
					409,
					'req-409'
				)
			);

			await expect(platformApi.createAdmin('9820123456')).rejects.toThrow(
				'User already holds a platform role'
			);
		});
	});

	describe('platformApi.updateAdminStatus', () => {
		it('calls PATCH /platform/admins/{id}/status with status and reason', async () => {
			const mockResponse = {
				id: 'r-2',
				user_id: 'u-2',
				role: 'ADMIN',
				status: 'DISABLED',
				updated_at: '2026-10-10T15:00:00Z'
			};

			const patchSpy = vi.spyOn(api, 'patch').mockResolvedValue(mockResponse);

			const result = await platformApi.updateAdminStatus('r-2', 'DISABLED', 'Security review');

			expect(result).toEqual(mockResponse);
			expect(patchSpy).toHaveBeenCalledWith('/platform/admins/r-2/status', {
				status: 'DISABLED',
				reason: 'Security review'
			});
		});

		it('handles self-disable rejection (400 CANNOT_DISABLE_SELF)', async () => {
			vi.spyOn(api, 'patch').mockRejectedValue(
				new ApiClientError('Super Admin cannot disable self', 'CANNOT_DISABLE_SELF', 400, 'req-400')
			);

			await expect(
				platformApi.updateAdminStatus('r-super', 'DISABLED', 'Self disable')
			).rejects.toThrow('Super Admin cannot disable self');
		});
	});

	describe('platformApi.listAuditLogs', () => {
		it('calls GET /platform/audit-logs with action filter and pagination', async () => {
			const mockResponse = {
				items: [
					{
						id: 'a-1',
						actor_id: 'u-super',
						action: 'ADMIN_DISABLED',
						target_user_id: 'u-2',
						details: { reason: 'Security review' },
						client_ip: '127.0.0.1',
						created_at: '2026-10-10T15:00:00Z'
					}
				],
				total: 1,
				limit: 20,
				offset: 0
			};

			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockResponse);

			const result = await platformApi.listAuditLogs({
				limit: 20,
				offset: 0,
				action: 'ADMIN_DISABLED'
			});

			expect(result).toEqual(mockResponse);
			expect(getSpy).toHaveBeenCalledWith('/platform/audit-logs', {
				params: {
					limit: 20,
					offset: 0,
					action: 'ADMIN_DISABLED'
				}
			});
		});
	});
});
