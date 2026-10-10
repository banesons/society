import { api } from '$lib/api/client';

export interface SocietyItem {
	id: string;
	society_code: string;
	name: string;
	registration_number?: string;
	address_line1: string;
	address_line2?: string;
	locality: string;
	city: string;
	state: string;
	pincode: string;
	contact_email?: string;
	contact_phone?: string;
	status: 'PENDING' | 'ACTIVE' | 'SUSPENDED';
	created_by: string;
	updated_by?: string;
	created_at: string;
	updated_at: string;
}

export interface SocietyListResponse {
	items: SocietyItem[];
	total: number;
	limit: number;
	offset: number;
}

export interface CreateSocietyPayload {
	name: string;
	registration_number?: string;
	address_line1: string;
	address_line2?: string;
	locality: string;
	city: string;
	state: string;
	pincode: string;
	contact_email?: string;
	contact_phone?: string;
}

export interface UpdateSocietyPayload {
	name?: string;
	registration_number?: string;
	address_line1?: string;
	address_line2?: string;
	locality?: string;
	city?: string;
	state?: string;
	pincode?: string;
	contact_email?: string;
	contact_phone?: string;
}

export interface SuspensionReason {
	id: number;
	code: string;
	label: string;
	description?: string;
	category: string;
	allows_manual?: boolean;
	allows_automated?: boolean;
	is_active?: boolean;
}

export interface SuspendSocietyPayload {
	reason_id?: number;
	reason_code?: string;
	reason_details?: string;
}

export const societyApi = {
	listSocieties(params?: {
		limit?: number;
		offset?: number;
		status?: string;
		search?: string;
	}): Promise<SocietyListResponse> {
		return api.get<SocietyListResponse>('/platform/societies', {
			params: {
				limit: params?.limit ?? 20,
				offset: params?.offset ?? 0,
				status: params?.status || undefined,
				search: params?.search || undefined
			}
		});
	},

	createSociety(payload: CreateSocietyPayload): Promise<SocietyItem> {
		return api.post<SocietyItem>('/platform/societies', payload);
	},

	getSociety(id: string): Promise<SocietyItem> {
		return api.get<SocietyItem>(`/platform/societies/${id}`);
	},

	updateSociety(id: string, payload: UpdateSocietyPayload): Promise<SocietyItem> {
		return api.patch<SocietyItem>(`/platform/societies/${id}`, payload);
	},

	getSuspensionReasons(): Promise<SuspensionReason[]> {
		return api.get<SuspensionReason[]>('/platform/society-suspension-reasons');
	},

	activateSociety(id: string, reason?: string): Promise<SocietyItem> {
		return api.post<SocietyItem>(`/platform/societies/${id}/activate`, { reason });
	},

	suspendSociety(id: string, payload: SuspendSocietyPayload): Promise<SocietyItem> {
		return api.post<SocietyItem>(`/platform/societies/${id}/suspend`, payload);
	},

	reactivateSociety(id: string, reason?: string): Promise<SocietyItem> {
		return api.post<SocietyItem>(`/platform/societies/${id}/reactivate`, { reason });
	},

	// S04 Society Administration Methods (Platform SUPER_ADMIN)
	listSocietyAdmins(
		societyId: string,
		params?: { status?: string; limit?: number; offset?: number }
	): Promise<SocietyAdminListResponse> {
		return api.get<SocietyAdminListResponse>(`/platform/societies/${societyId}/admins`, {
			params: {
				status: params?.status || undefined,
				limit: params?.limit ?? 20,
				offset: params?.offset ?? 0
			}
		});
	},

	assignSocietyAdmin(
		societyId: string,
		payload: { phone?: string; user_id?: string; role?: string }
	): Promise<SocietyAdminAssignment> {
		return api.post<SocietyAdminAssignment>(`/platform/societies/${societyId}/admins`, payload);
	},

	revokeSocietyAdmin(societyId: string, assignmentId: string): Promise<SocietyAdminAssignment> {
		return api.delete<SocietyAdminAssignment>(
			`/platform/societies/${societyId}/admins/${assignmentId}`
		);
	},

	// S04 Society Scoped Operational Methods (SOC_ADMIN)
	getSocietyProfile(societyId: string): Promise<SocietyProfile> {
		return api.get<SocietyProfile>(`/societies/${societyId}/profile`);
	},

	// S05 Society Structure & Formation Methods
	listBuildings(societyId: string): Promise<{ buildings: Building[]; total: number }> {
		return api.get<{ buildings: Building[]; total: number }>(`/societies/${societyId}/buildings`);
	},

	createBuilding(societyId: string, payload: CreateBuildingPayload): Promise<Building> {
		return api.post<Building>(`/societies/${societyId}/buildings`, payload);
	},

	createWing(societyId: string, buildingId: string, payload: CreateWingPayload): Promise<Wing> {
		return api.post<Wing>(`/societies/${societyId}/buildings/${buildingId}/wings`, payload);
	},

	previewUnits(societyId: string, payload: GenerateUnitsParams): Promise<UnitPreviewResult> {
		return api.post<UnitPreviewResult>(`/societies/${societyId}/units/preview`, payload);
	},

	generateUnits(
		societyId: string,
		payload: GenerateUnitsParams
	): Promise<{ generated_count: number; message: string }> {
		return api.post<{ generated_count: number; message: string }>(
			`/societies/${societyId}/units/generate`,
			payload
		);
	},

	listUnits(
		societyId: string,
		params?: { building_id?: string; wing_id?: string; limit?: number; offset?: number }
	): Promise<{ units: Unit[]; total: number; limit: number; offset: number }> {
		return api.get<{ units: Unit[]; total: number; limit: number; offset: number }>(
			`/societies/${societyId}/units`,
			{
				params: {
					building_id: params?.building_id || undefined,
					wing_id: params?.wing_id || undefined,
					limit: params?.limit ?? 50,
					offset: params?.offset ?? 0
				}
			}
		);
	},

	listGates(societyId: string): Promise<{ gates: Gate[]; total: number }> {
		return api.get<{ gates: Gate[]; total: number }>(`/societies/${societyId}/gates`);
	},

	createGate(societyId: string, payload: CreateGatePayload): Promise<Gate> {
		return api.post<Gate>(`/societies/${societyId}/gates`, payload);
	},

	getStaffing(societyId: string): Promise<StaffingRequirement> {
		return api.get<StaffingRequirement>(`/societies/${societyId}/staffing`);
	},

	setStaffing(societyId: string, payload: UpdateStaffingPayload): Promise<StaffingRequirement> {
		return api.put<StaffingRequirement>(`/societies/${societyId}/staffing`, payload);
	},

	getFormationSummary(societyId: string): Promise<FormationSummary> {
		return api.get<FormationSummary>(`/societies/${societyId}/formation-summary`);
	}
};

export interface SocietyAdminAssignment {
	id: string;
	society_id: string;
	user_id: string;
	user_phone_masked: string;
	role: string;
	status: 'ACTIVE' | 'REVOKED';
	assigned_by: string;
	revoked_by?: string;
	revoked_at?: string;
	created_at: string;
	updated_at: string;
}

export interface SocietyAdminListResponse {
	items: SocietyAdminAssignment[];
	total: number;
	limit: number;
	offset: number;
}

export interface UserSocietyAssignment {
	society_id: string;
	society_code: string;
	society_name: string;
	role: string;
	status: 'ACTIVE';
}

export interface SocietyProfile {
	id: string;
	society_code: string;
	name: string;
	status: 'ACTIVE' | 'PENDING' | 'SUSPENDED';
	address_line1: string;
	address_line2?: string;
	locality: string;
	city: string;
	state: string;
	pincode: string;
	contact_email?: string;
	contact_phone?: string;
}

// S05 Structure Models
export interface Building {
	id: string;
	society_id: string;
	name: string;
	building_code: string;
	total_floors: number;
	wings?: Wing[];
	unit_count: number;
	created_at: string;
	updated_at: string;
}

export interface Wing {
	id: string;
	society_id: string;
	building_id: string;
	name: string;
	wing_code: string;
	total_floors: number;
	unit_count: number;
	created_at: string;
	updated_at: string;
}

export interface Unit {
	id: string;
	society_id: string;
	building_id: string;
	wing_id?: string;
	floor_number: number;
	unit_number: string;
	unit_identifier: string;
	unit_type: string;
	status: string;
	created_at: string;
	updated_at: string;
}

export interface Gate {
	id: string;
	society_id: string;
	name: string;
	gate_code: string;
	gate_type: string;
	status: string;
	required_staff_per_shift: number;
	description?: string;
	created_at: string;
	updated_at: string;
}

export interface StaffingRequirement {
	id?: string;
	society_id: string;
	shifts_per_day: number;
	total_required_headcount: number;
	notes?: string;
	updated_by?: string;
	created_at?: string;
	updated_at?: string;
}

export interface FormationSummary {
	total_buildings: number;
	total_wings: number;
	total_floors: number;
	total_units: number;
	total_gates: number;
	required_staff_per_shift: number;
	total_required_headcount: number;
	registered_gatekeepers: number;
	assigned_gatekeepers: number;
}

export interface CreateBuildingPayload {
	name: string;
	building_code: string;
	total_floors: number;
}

export interface CreateWingPayload {
	name: string;
	wing_code: string;
	total_floors: number;
}

export interface GenerateUnitsParams {
	building_id: string;
	wing_id?: string;
	starting_floor: number;
	total_floors: number;
	units_per_floor: number;
	starting_unit_number: number;
	prefix?: string;
	floor_overrides?: Record<number, number>;
	unit_type?: string;
}

export interface GeneratedUnitPreview {
	floor_number: number;
	unit_number: string;
	unit_identifier: string;
	unit_type: string;
}

export interface UnitPreviewResult {
	total_units: number;
	units: GeneratedUnitPreview[];
}

export interface CreateGatePayload {
	name: string;
	gate_code: string;
	gate_type: string;
	status: string;
	required_staff_per_shift: number;
	description?: string;
}

export interface UpdateStaffingPayload {
	shifts_per_day: number;
	total_required_headcount: number;
	notes?: string;
}
