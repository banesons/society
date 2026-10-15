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
	actor_name: string;
	actor_phone: string;
	action: string;
	action_label: string;
	target_user_id?: string;
	target_display?: string;
	society_id?: string;
	society_name?: string;
	society_code?: string;
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
	pagination?: {
		page: number;
		limit: number;
		total_items: number;
		total_pages: number;
	};
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
		page?: number;
		limit?: number;
		offset?: number;
		action?: string;
		search?: string;
		actor_id?: string;
		target_user_id?: string;
		society_id?: string;
		start_date?: string;
		end_date?: string;
	}): Promise<AuditListResponse> {
		const queryParams: Record<string, string | number | boolean | undefined> = {
			limit: params?.limit ?? 20
		};
		if (params?.offset !== undefined) {
			queryParams.offset = params.offset;
		}
		if (params?.page !== undefined) {
			queryParams.page = params.page;
		}
		if (params?.action) queryParams.action = params.action;
		if (params?.search) queryParams.search = params.search;
		if (params?.actor_id) queryParams.actor_id = params.actor_id;
		if (params?.target_user_id) queryParams.target_user_id = params.target_user_id;
		if (params?.society_id) queryParams.society_id = params.society_id;
		if (params?.start_date) queryParams.start_date = params.start_date;
		if (params?.end_date) queryParams.end_date = params.end_date;

		return api.get<AuditListResponse>('/platform/audit-logs', { params: queryParams });
	},

	getAuditLog(id: string): Promise<{ item: AuditLogItem }> {
		return api.get<{ item: AuditLogItem }>(`/platform/audit-logs/${id}`);
	}
};
