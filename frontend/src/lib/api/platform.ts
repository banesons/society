import { api } from '$lib/api/client';

export interface AdminItem {
	id: string;
	user_id: string;
	phone: string;
	role: 'SUPER_ADMIN' | 'ADMIN';
	status: 'ACTIVE' | 'DISABLED';
	created_by?: string;
	created_at: string;
	updated_at: string;
}

export interface AdminListResponse {
	items: AdminItem[];
	total: number;
	limit: number;
	offset: number;
}

export interface CreateAdminPayload {
	phone: string;
}

export interface CreateAdminResponse {
	id: string;
	user_id: string;
	phone: string;
	role: string;
	status: string;
	created_at: string;
}

export interface UpdateAdminStatusPayload {
	status: 'ACTIVE' | 'DISABLED';
	reason?: string;
}

export interface UpdateAdminStatusResponse {
	id: string;
	user_id: string;
	role: string;
	status: string;
	updated_at: string;
}

export interface AuditLogItem {
	id: string;
	actor_id: string;
	action: string;
	target_user_id?: string;
	details: Record<string, unknown>;
	client_ip?: string;
	user_agent?: string;
	created_at: string;
}

export interface AuditListResponse {
	items: AuditLogItem[];
	total: number;
	limit: number;
	offset: number;
}

export const platformApi = {
	listAdmins(params?: {
		limit?: number;
		offset?: number;
		role?: string;
		status?: string;
	}): Promise<AdminListResponse> {
		return api.get<AdminListResponse>('/platform/admins', {
			params: {
				limit: params?.limit ?? 20,
				offset: params?.offset ?? 0,
				role: params?.role || undefined,
				status: params?.status || undefined
			}
		});
	},

	createAdmin(phone: string): Promise<CreateAdminResponse> {
		return api.post<CreateAdminResponse>('/platform/admins', { phone });
	},

	updateAdminStatus(
		id: string,
		status: 'ACTIVE' | 'DISABLED',
		reason?: string
	): Promise<UpdateAdminStatusResponse> {
		return api.patch<UpdateAdminStatusResponse>(`/platform/admins/${id}/status`, {
			status,
			reason: reason || undefined
		});
	},

	listAuditLogs(params?: {
		limit?: number;
		offset?: number;
		action?: string;
	}): Promise<AuditListResponse> {
		return api.get<AuditListResponse>('/platform/audit-logs', {
			params: {
				limit: params?.limit ?? 50,
				offset: params?.offset ?? 0,
				action: params?.action || undefined
			}
		});
	}
};
