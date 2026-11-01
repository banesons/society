import { api } from '$lib/api/client';

export interface UserPersonalProfile {
	id: string;
	phone: string;
	phone_raw: string;
	first_name: string;
	last_name: string;
	display_name: string;
	email: string;
	profile_photo_url?: string | null;
	status: string;
	platform_roles?: string[];
	society_assignments?: Array<{
		society_id: string;
		society_name: string;
		role: string;
	}>;
}

export interface UpdatePersonalProfileParams {
	first_name?: string;
	last_name?: string;
	email?: string;
	profile_photo_url?: string | null;
}

export async function getPersonalProfile(): Promise<UserPersonalProfile> {
	return api.get<UserPersonalProfile>('/auth/profile');
}

export async function updatePersonalProfile(
	params: UpdatePersonalProfileParams
): Promise<UserPersonalProfile> {
	return api.patch<UserPersonalProfile>('/auth/profile', params);
}

export interface ActiveSession {
	id: string;
	device_label: string;
	browser: string;
	os: string;
	is_current: boolean;
	created_at: string;
	last_active_at: string;
}

export interface ActiveSessionsResponse {
	sessions: ActiveSession[];
	max_active_sessions: number;
}

export async function getActiveSessions(): Promise<ActiveSessionsResponse> {
	return api.get<ActiveSessionsResponse>('/auth/sessions');
}

export async function revokeActiveSession(sessionId: string): Promise<{ message: string }> {
	return api.delete<{ message: string }>(`/auth/sessions/${sessionId}`);
}

export interface UserActivityEvent {
	id: string;
	action: string;
	category: 'ACCOUNT' | 'VISITOR' | 'SOCIETY' | 'SECURITY';
	title?: string;
	summary?: string;
	display_title?: string;
	display_summary?: string;
	timestamp: string;
	society_id?: string;
	society_name?: string;
	unit_identifier?: string;
	gate_name?: string;
	details?: Record<string, unknown>;
}

export interface UserActivityTimelineResponse {
	events: UserActivityEvent[];
	total: number;
	limit: number;
	offset: number;
	category: string;
}

export async function getUserActivityTimeline(
	category?: string,
	limit?: number,
	offset?: number
): Promise<UserActivityTimelineResponse> {
	return api.get<UserActivityTimelineResponse>('/auth/profile/activity', {
		params: {
			category: category && category !== 'ALL' ? category : undefined,
			limit: limit ?? 20,
			offset: offset ?? 0
		}
	});
}
