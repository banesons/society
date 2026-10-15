import { api } from '$lib/api/client';

export interface NotificationItem {
	id: string;
	recipient_user_id: string;
	society_id?: string | null;
	type: string;
	title: string;
	body: string;
	action_url?: string;
	metadata: Record<string, unknown>;
	reference_type?: string;
	reference_id?: string;
	read_at?: string | null;
	created_at: string;
}

export interface NotificationListResponse {
	items: NotificationItem[];
	total: number;
	page: number;
	limit: number;
	unread_count: number;
}

export interface UnreadCountResponse {
	unread_count: number;
}

export interface MarkReadResponse {
	id: string;
	read_at: string;
}

export interface MarkAllReadResponse {
	marked_count: number;
}

export interface UserDeviceItem {
	id: string;
	user_id: string;
	device_id: string;
	platform: 'WEB' | 'ANDROID' | 'IOS';
	user_agent?: string;
	client_ip?: string;
	status: 'ACTIVE' | 'INACTIVE' | 'REVOKED';
	last_seen_at: string;
	created_at: string;
	updated_at: string;
}

export interface RegisterDevicePayload {
	device_id: string;
	fcm_token: string;
	platform: 'WEB' | 'ANDROID' | 'IOS';
}

export interface ListNotificationsParams {
	page?: number;
	limit?: number;
	unread_only?: boolean;
	society_id?: string;
}

export const notificationsApi = {
	async list(params: ListNotificationsParams = {}): Promise<NotificationListResponse> {
		const query: Record<string, string | number | boolean | undefined> = {
			page: params.page || 1,
			limit: params.limit || 20
		};
		if (params.unread_only) query.unread_only = true;
		if (params.society_id) query.society_id = params.society_id;

		return await api.get<NotificationListResponse>('/notifications', { params: query });
	},

	async getUnreadCount(societyId?: string): Promise<number> {
		const query: Record<string, string | undefined> = {};
		if (societyId) query.society_id = societyId;

		const res = await api.get<UnreadCountResponse>('/notifications/unread-count', {
			params: query
		});
		return res.unread_count;
	},

	async markAsRead(id: string): Promise<MarkReadResponse> {
		return await api.patch<MarkReadResponse>(`/notifications/${id}/read`);
	},

	async markAllAsRead(societyId?: string): Promise<MarkAllReadResponse> {
		const query: Record<string, string | undefined> = {};
		if (societyId) query.society_id = societyId;

		return await api.post<MarkAllReadResponse>('/notifications/mark-all-read', undefined, {
			params: query
		});
	},

	async registerDevice(payload: RegisterDevicePayload): Promise<UserDeviceItem> {
		return await api.post<UserDeviceItem>('/notifications/devices', payload);
	},

	async deactivateDevice(deviceId: string): Promise<{ success: boolean }> {
		return await api.delete<{ success: boolean }>(`/notifications/devices/${deviceId}`);
	},

	async listDevices(): Promise<UserDeviceItem[]> {
		const res = await api.get<{ devices: UserDeviceItem[] }>('/notifications/devices');
		return res.devices;
	}
};
