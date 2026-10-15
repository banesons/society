import { describe, it, expect, vi, beforeEach } from 'vitest';
import { notificationsApi } from './notifications';
import { api } from './client';
import { isPushSupported, getNotificationPermission, getOrCreateDeviceId } from '$lib/push';

describe('Notifications API Client', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it('fetches notification list with pagination parameters', async () => {
		const mockResponse = {
			items: [
				{
					id: 'notif-1',
					recipient_user_id: 'user-1',
					type: 'ANNOUNCEMENT_PUBLISHED',
					title: 'Water Outage Notice',
					body: 'Maintenance on tank tomorrow',
					action_url: '/society/123#announcements',
					metadata: {},
					read_at: null,
					created_at: '2026-10-14T10:00:00Z'
				}
			],
			total: 1,
			page: 1,
			limit: 20,
			unread_count: 1
		};

		vi.spyOn(api, 'get').mockResolvedValue(mockResponse);

		const res = await notificationsApi.list({ page: 1, limit: 20, society_id: 'soc-1' });
		expect(res.items.length).toBe(1);
		expect(res.unread_count).toBe(1);
		expect(api.get).toHaveBeenCalledWith('/notifications', {
			params: { page: 1, limit: 20, society_id: 'soc-1' }
		});
	});

	it('fetches unread count', async () => {
		vi.spyOn(api, 'get').mockResolvedValue({ unread_count: 5 });

		const count = await notificationsApi.getUnreadCount('soc-1');
		expect(count).toBe(5);
		expect(api.get).toHaveBeenCalledWith('/notifications/unread-count', {
			params: { society_id: 'soc-1' }
		});
	});

	it('marks a single notification read', async () => {
		const mockReadAt = '2026-10-14T10:30:00Z';
		vi.spyOn(api, 'patch').mockResolvedValue({ id: 'notif-1', read_at: mockReadAt });

		const res = await notificationsApi.markAsRead('notif-1');
		expect(res.id).toBe('notif-1');
		expect(res.read_at).toBe(mockReadAt);
		expect(api.patch).toHaveBeenCalledWith('/notifications/notif-1/read');
	});

	it('marks all notifications read', async () => {
		vi.spyOn(api, 'post').mockResolvedValue({ marked_count: 3 });

		const res = await notificationsApi.markAllAsRead('soc-1');
		expect(res.marked_count).toBe(3);
		expect(api.post).toHaveBeenCalledWith('/notifications/mark-all-read', undefined, {
			params: { society_id: 'soc-1' }
		});
	});

	it('registers a device token', async () => {
		const mockDevice = {
			id: 'dev-rec-1',
			user_id: 'user-1',
			device_id: 'chrome-xyz',
			platform: 'WEB' as const,
			status: 'ACTIVE' as const,
			last_seen_at: '2026-10-14T10:00:00Z',
			created_at: '2026-10-14T10:00:00Z',
			updated_at: '2026-10-14T10:00:00Z'
		};

		vi.spyOn(api, 'post').mockResolvedValue(mockDevice);

		const res = await notificationsApi.registerDevice({
			device_id: 'chrome-xyz',
			fcm_token: 'fcm-tok-123',
			platform: 'WEB'
		});

		expect(res.device_id).toBe('chrome-xyz');
		expect(api.post).toHaveBeenCalledWith('/notifications/devices', {
			device_id: 'chrome-xyz',
			fcm_token: 'fcm-tok-123',
			platform: 'WEB'
		});
	});

	it('deactivates a registered device', async () => {
		vi.spyOn(api, 'delete').mockResolvedValue({ success: true });

		const res = await notificationsApi.deactivateDevice('chrome-xyz');
		expect(res.success).toBe(true);
		expect(api.delete).toHaveBeenCalledWith('/notifications/devices/chrome-xyz');
	});
});

describe('Push Helpers', () => {
	it('detects push support based on window features', () => {
		expect(typeof isPushSupported()).toBe('boolean');
	});

	it('detects permission status or unsupported', () => {
		const perm = getNotificationPermission();
		expect(['default', 'granted', 'denied', 'unsupported']).toContain(perm);
	});

	it('generates and persists device ID in localStorage if window present', () => {
		const devId = getOrCreateDeviceId();
		expect(devId).toBeTruthy();
		const devId2 = getOrCreateDeviceId();
		expect(devId2).toBe(devId);
	});
});
