import { api } from '$lib/api/client';
import type { AuditListResponse, AuditLogItem } from '$lib/api/platform';

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
	},

	// S06: Members & Relationships
	listMembers(
		societyId: string,
		params?: { limit?: number; offset?: number; status?: string; search?: string }
	): Promise<MemberListResponse> {
		return api.get<MemberListResponse>(`/societies/${societyId}/members`, {
			params: {
				limit: params?.limit ?? 50,
				offset: params?.offset ?? 0,
				status: params?.status || undefined,
				search: params?.search || undefined
			}
		});
	},

	registerMember(
		societyId: string,
		payload: RegisterMemberPayload
	): Promise<{ member: SocietyMember }> {
		return api.post<{ member: SocietyMember }>(`/societies/${societyId}/members`, payload);
	},

	getMember(
		societyId: string,
		memberId: string
	): Promise<{ member: SocietyMember; relationships: UnitRelationship[] }> {
		return api.get<{ member: SocietyMember; relationships: UnitRelationship[] }>(
			`/societies/${societyId}/members/${memberId}`
		);
	},

	updateMember(
		societyId: string,
		memberId: string,
		payload: UpdateMemberPayload
	): Promise<{ member: SocietyMember }> {
		return api.patch<{ member: SocietyMember }>(
			`/societies/${societyId}/members/${memberId}`,
			payload
		);
	},

	previewMemberDeparture(societyId: string, memberId: string): Promise<DeparturePreviewResponse> {
		return api.post<DeparturePreviewResponse>(
			`/societies/${societyId}/members/${memberId}/departure/preview`
		);
	},

	confirmMemberDeparture(
		societyId: string,
		memberId: string,
		payload: ConfirmDeparturePayload
	): Promise<{ message: string }> {
		return api.post<{ message: string }>(
			`/societies/${societyId}/members/${memberId}/departure/confirm`,
			payload
		);
	},

	listUnitRelationships(
		societyId: string,
		unitId: string
	): Promise<{
		unit_id: string;
		total_ownership_share: number;
		is_vacant: boolean;
		relationships: UnitRelationship[];
	}> {
		return api.get<{
			unit_id: string;
			total_ownership_share: number;
			is_vacant: boolean;
			relationships: UnitRelationship[];
		}>(`/societies/${societyId}/units/${unitId}/relationships`);
	},

	createUnitRelationship(
		societyId: string,
		unitId: string,
		payload: CreateRelationshipPayload
	): Promise<{ relationship: UnitRelationship }> {
		return api.post<{ relationship: UnitRelationship }>(
			`/societies/${societyId}/units/${unitId}/relationships`,
			payload
		);
	},

	endUnitRelationship(
		societyId: string,
		unitId: string,
		relationshipId: string,
		payload: EndRelationshipPayload
	): Promise<{ relationship: UnitRelationship }> {
		return api.post<{ relationship: UnitRelationship }>(
			`/societies/${societyId}/units/${unitId}/relationships/${relationshipId}/end`,
			payload
		);
	},

	// S07: Society Roles & Designations
	listDesignations(
		societyId: string,
		includeInactive = false
	): Promise<{ designations: SocietyDesignation[] }> {
		return api.get<{ designations: SocietyDesignation[] }>(`/societies/${societyId}/designations`, {
			params: { include_inactive: includeInactive }
		});
	},

	createDesignation(
		societyId: string,
		payload: CreateDesignationPayload
	): Promise<{ designation: SocietyDesignation }> {
		return api.post<{ designation: SocietyDesignation }>(
			`/societies/${societyId}/designations`,
			payload
		);
	},

	updateDesignation(
		societyId: string,
		designationId: string,
		payload: UpdateDesignationPayload
	): Promise<{ designation: SocietyDesignation }> {
		return api.patch<{ designation: SocietyDesignation }>(
			`/societies/${societyId}/designations/${designationId}`,
			payload
		);
	},

	setDesignationStatus(
		societyId: string,
		designationId: string,
		isActive: boolean
	): Promise<{ designation: SocietyDesignation }> {
		return api.patch<{ designation: SocietyDesignation }>(
			`/societies/${societyId}/designations/${designationId}/status`,
			{ is_active: isActive }
		);
	},

	getCommitteeOverview(societyId: string): Promise<CommitteeOverview> {
		return api.get<CommitteeOverview>(`/societies/${societyId}/committee/overview`);
	},

	listAppointments(
		societyId: string,
		params?: { status?: string; member_id?: string; designation_id?: string }
	): Promise<{ appointments: CommitteeAppointment[] }> {
		return api.get<{ appointments: CommitteeAppointment[] }>(
			`/societies/${societyId}/appointments`,
			{
				params: {
					status: params?.status || undefined,
					member_id: params?.member_id || undefined,
					designation_id: params?.designation_id || undefined
				}
			}
		);
	},

	createAppointment(
		societyId: string,
		payload: CreateAppointmentPayload
	): Promise<{ appointment: CommitteeAppointment }> {
		return api.post<{ appointment: CommitteeAppointment }>(
			`/societies/${societyId}/appointments`,
			payload
		);
	},

	endAppointment(
		societyId: string,
		appointmentId: string,
		payload: EndAppointmentPayload
	): Promise<{ appointment: CommitteeAppointment }> {
		return api.post<{ appointment: CommitteeAppointment }>(
			`/societies/${societyId}/appointments/${appointmentId}/end`,
			payload
		);
	},

	// S08: Gate Foundation
	updateGate(
		societyId: string,
		gateId: string,
		payload: UpdateGatePayload
	): Promise<{ gate: Gate }> {
		return api.patch<{ gate: Gate }>(`/societies/${societyId}/gates/${gateId}`, payload);
	},

	listGatekeepers(
		societyId: string,
		status?: string
	): Promise<{ gatekeepers: SocietyAdminAssignment[] }> {
		return api.get<{ gatekeepers: SocietyAdminAssignment[] }>(
			`/societies/${societyId}/gatekeepers`,
			{
				params: { status }
			}
		);
	},

	assignGatekeeper(
		societyId: string,
		phone: string
	): Promise<{ gatekeeper: SocietyAdminAssignment }> {
		return api.post<{ gatekeeper: SocietyAdminAssignment }>(`/societies/${societyId}/gatekeepers`, {
			phone
		});
	},

	revokeGatekeeper(societyId: string, assignmentId: string): Promise<{ message: string }> {
		return api.delete<{ message: string }>(`/societies/${societyId}/gatekeepers/${assignmentId}`);
	},

	listShifts(societyId: string): Promise<{ shifts: GateShift[] }> {
		return api.get<{ shifts: GateShift[] }>(`/societies/${societyId}/shifts`);
	},

	createShift(societyId: string, payload: CreateGateShiftPayload): Promise<{ shift: GateShift }> {
		return api.post<{ shift: GateShift }>(`/societies/${societyId}/shifts`, payload);
	},

	listDutySessions(
		societyId: string,
		params?: { gate_id?: string; status?: string }
	): Promise<{ sessions: GateDutySession[] }> {
		return api.get<{ sessions: GateDutySession[] }>(`/societies/${societyId}/duty-sessions`, {
			params
		});
	},

	checkInDuty(
		societyId: string,
		gateId: string,
		payload?: { shift_id?: string }
	): Promise<{ session: GateDutySession }> {
		return api.post<{ session: GateDutySession }>(
			`/societies/${societyId}/gates/${gateId}/sessions/check-in`,
			payload ?? {}
		);
	},

	checkOutDuty(
		societyId: string,
		gateId: string,
		payload?: { handover_notes?: string }
	): Promise<{ session: GateDutySession }> {
		return api.post<{ session: GateDutySession }>(
			`/societies/${societyId}/gates/${gateId}/sessions/check-out`,
			payload ?? {}
		);
	},

	// S09: Gate Visitor Operations
	listVisits(
		societyId: string,
		params?: {
			status?: string;
			visit_type?: string;
			search?: string;
			unit_id?: string;
			limit?: number;
			offset?: number;
		}
	): Promise<{ visits: Visit[]; total: number; limit: number; offset: number }> {
		return api.get<{ visits: Visit[]; total: number; limit: number; offset: number }>(
			`/societies/${societyId}/visitors`,
			{ params }
		);
	},

	getVisitorDashboard(societyId: string): Promise<VisitorDashboardMetrics> {
		return api.get<VisitorDashboardMetrics>(`/societies/${societyId}/visitors/dashboard`);
	},

	getVisitDetails(
		societyId: string,
		visitId: string
	): Promise<{
		visit: Visit;
		passes: VisitorPass[];
		approvals: VisitApproval[];
		events: GateEvent[];
	}> {
		return api.get<{
			visit: Visit;
			passes: VisitorPass[];
			approvals: VisitApproval[];
			events: GateEvent[];
		}>(`/societies/${societyId}/visitors/${visitId}`);
	},

	registerExpectedVisit(
		societyId: string,
		payload: RegisterExpectedVisitPayload
	): Promise<{ visit: Visit; pass?: VisitorPass }> {
		return api.post<{ visit: Visit; pass?: VisitorPass }>(
			`/societies/${societyId}/visitors/expected`,
			payload
		);
	},

	registerWalkInVisit(
		societyId: string,
		payload: RegisterWalkInVisitPayload
	): Promise<{ visit: Visit }> {
		return api.post<{ visit: Visit }>(`/societies/${societyId}/visitors/walk-in`, payload);
	},

	approveVisit(
		societyId: string,
		visitId: string,
		payload?: { notes?: string }
	): Promise<{ visit: Visit }> {
		return api.post<{ visit: Visit }>(
			`/societies/${societyId}/visitors/${visitId}/approve`,
			payload ?? {}
		);
	},

	rejectVisit(
		societyId: string,
		visitId: string,
		payload?: { reason?: string }
	): Promise<{ visit: Visit }> {
		return api.post<{ visit: Visit }>(
			`/societies/${societyId}/visitors/${visitId}/reject`,
			payload ?? {}
		);
	},

	cancelVisit(societyId: string, visitId: string): Promise<{ visit: Visit }> {
		return api.post<{ visit: Visit }>(`/societies/${societyId}/visitors/${visitId}/cancel`, {});
	},

	issueVisitorPass(
		societyId: string,
		visitId: string,
		payload?: { valid_from?: string; valid_until?: string }
	): Promise<{ pass: VisitorPass; bearer_token: string }> {
		return api.post<{ pass: VisitorPass; bearer_token: string }>(
			`/societies/${societyId}/visitors/${visitId}/passes`,
			payload ?? {}
		);
	},

	verifyVisitorPass(
		societyId: string,
		tokenOrCode: string
	): Promise<{ pass: VisitorPass; visit: Visit }> {
		return api.post<{ pass: VisitorPass; visit: Visit }>(
			`/societies/${societyId}/visitors/passes/verify`,
			{ token_or_code: tokenOrCode }
		);
	},

	// S09 Resident Methods
	getMyUnits(societyId: string): Promise<{ units: ResidentUnit[] }> {
		return api.get<{ units: ResidentUnit[] }>(`/societies/${societyId}/units/my`);
	},

	updateUnitVisitorPolicy(
		societyId: string,
		unitId: string,
		requiresApproval: boolean
	): Promise<{ unit: ResidentUnit }> {
		return api.patch<{ unit: ResidentUnit }>(
			`/societies/${societyId}/units/${unitId}/visitor-policy`,
			{ requires_visitor_approval: requiresApproval }
		);
	},

	getPendingVisits(societyId: string): Promise<{ visits: Visit[] }> {
		return api.get<{ visits: Visit[] }>(`/societies/${societyId}/visitors/pending`);
	},

	checkInVisitor(
		societyId: string,
		visitId: string,
		payload: CheckInVisitorPayload
	): Promise<{ visit: Visit; event: GateEvent }> {
		return api.post<{ visit: Visit; event: GateEvent }>(
			`/societies/${societyId}/visitors/${visitId}/check-in`,
			payload
		);
	},

	checkOutVisitor(
		societyId: string,
		visitId: string,
		payload: CheckOutVisitorPayload
	): Promise<{ visit: Visit; event: GateEvent }> {
		return api.post<{ visit: Visit; event: GateEvent }>(
			`/societies/${societyId}/visitors/${visitId}/check-out`,
			payload
		);
	},

	// S10 Society Communication Methods
	createAnnouncement(
		societyId: string,
		payload: CreateAnnouncementPayload
	): Promise<{ announcement: Announcement }> {
		return api.post<{ announcement: Announcement }>(
			`/societies/${societyId}/announcements`,
			payload
		);
	},

	updateAnnouncement(
		societyId: string,
		announcementId: string,
		payload: UpdateAnnouncementPayload
	): Promise<{ announcement: Announcement }> {
		return api.put<{ announcement: Announcement }>(
			`/societies/${societyId}/announcements/${announcementId}`,
			payload
		);
	},

	publishAnnouncement(
		societyId: string,
		announcementId: string
	): Promise<{ announcement: Announcement }> {
		return api.post<{ announcement: Announcement }>(
			`/societies/${societyId}/announcements/${announcementId}/publish`,
			{}
		);
	},

	archiveAnnouncement(
		societyId: string,
		announcementId: string
	): Promise<{ announcement: Announcement }> {
		return api.post<{ announcement: Announcement }>(
			`/societies/${societyId}/announcements/${announcementId}/archive`,
			{}
		);
	},

	getAdminAnnouncements(
		societyId: string,
		filter?: AdminAnnouncementFilter
	): Promise<AdminAnnouncementListResponse> {
		const params = new URLSearchParams();
		if (filter?.status) params.set('status', filter.status);
		if (filter?.category) params.set('category', filter.category);
		if (filter?.search) params.set('search', filter.search);
		if (filter?.limit) params.set('limit', String(filter.limit));
		if (filter?.offset) params.set('offset', String(filter.offset));
		const qs = params.toString() ? `?${params.toString()}` : '';
		return api.get<AdminAnnouncementListResponse>(
			`/societies/${societyId}/announcements/admin${qs}`
		);
	},

	getAnnouncementFeed(
		societyId: string,
		filter?: AnnouncementFeedFilter
	): Promise<AnnouncementFeedResponse> {
		const params = new URLSearchParams();
		if (filter?.category) params.set('category', filter.category);
		if (filter?.unread_only) params.set('unread_only', 'true');
		if (filter?.search) params.set('search', filter.search);
		if (filter?.limit) params.set('limit', String(filter.limit));
		if (filter?.offset) params.set('offset', String(filter.offset));
		const qs = params.toString() ? `?${params.toString()}` : '';
		return api.get<AnnouncementFeedResponse>(`/societies/${societyId}/announcements/feed${qs}`);
	},

	getAnnouncementDetails(
		societyId: string,
		announcementId: string
	): Promise<{ announcement: Announcement }> {
		return api.get<{ announcement: Announcement }>(
			`/societies/${societyId}/announcements/${announcementId}`
		);
	},

	markAnnouncementRead(
		societyId: string,
		announcementId: string
	): Promise<{ announcement_id: string; read: boolean }> {
		return api.post<{ announcement_id: string; read: boolean }>(
			`/societies/${societyId}/announcements/${announcementId}/read`,
			{}
		);
	},

	getUnreadAnnouncementCount(societyId: string): Promise<{ unread_count: number }> {
		return api.get<{ unread_count: number }>(`/societies/${societyId}/announcements/unread-count`);
	},

	// Maintenance Foundation (S11)
	listChargeCategories(
		societyId: string,
		status?: string
	): Promise<{ categories: MaintenanceChargeCategory[] }> {
		const qs = status ? `?status=${encodeURIComponent(status)}` : '';
		return api.get<{ categories: MaintenanceChargeCategory[] }>(
			`/societies/${societyId}/maintenance/categories${qs}`
		);
	},

	createChargeCategory(
		societyId: string,
		payload: CreateChargeCategoryPayload
	): Promise<{ category: MaintenanceChargeCategory }> {
		return api.post<{ category: MaintenanceChargeCategory }>(
			`/societies/${societyId}/maintenance/categories`,
			payload
		);
	},

	updateChargeCategory(
		societyId: string,
		categoryId: string,
		payload: UpdateChargeCategoryPayload
	): Promise<{ category: MaintenanceChargeCategory }> {
		return api.put<{ category: MaintenanceChargeCategory }>(
			`/societies/${societyId}/maintenance/categories/${categoryId}`,
			payload
		);
	},

	listChargeConfigs(
		societyId: string,
		status?: string
	): Promise<{ configs: MaintenanceChargeConfig[] }> {
		const qs = status ? `?status=${encodeURIComponent(status)}` : '';
		return api.get<{ configs: MaintenanceChargeConfig[] }>(
			`/societies/${societyId}/maintenance/configs${qs}`
		);
	},

	createChargeConfig(
		societyId: string,
		payload: CreateChargeConfigPayload
	): Promise<{ config: MaintenanceChargeConfig }> {
		return api.post<{ config: MaintenanceChargeConfig }>(
			`/societies/${societyId}/maintenance/configs`,
			payload
		);
	},

	deactivateChargeConfig(societyId: string, configId: string): Promise<void> {
		return api.post<void>(`/societies/${societyId}/maintenance/configs/${configId}/deactivate`, {});
	},

	listBillingPeriods(
		societyId: string,
		status?: string
	): Promise<{ periods: MaintenanceBillingPeriod[] }> {
		const qs = status ? `?status=${encodeURIComponent(status)}` : '';
		return api.get<{ periods: MaintenanceBillingPeriod[] }>(
			`/societies/${societyId}/maintenance/periods${qs}`
		);
	},

	createBillingPeriod(
		societyId: string,
		payload: CreateBillingPeriodPayload
	): Promise<{ period: MaintenanceBillingPeriod }> {
		return api.post<{ period: MaintenanceBillingPeriod }>(
			`/societies/${societyId}/maintenance/periods`,
			payload
		);
	},

	previewBillGeneration(
		societyId: string,
		periodId: string
	): Promise<{ preview: BillGenerationPreview }> {
		return api.get<{ preview: BillGenerationPreview }>(
			`/societies/${societyId}/maintenance/periods/${periodId}/preview`
		);
	},

	generateBills(societyId: string, periodId: string): Promise<{ result: BillGenerationResult }> {
		return api.post<{ result: BillGenerationResult }>(
			`/societies/${societyId}/maintenance/periods/${periodId}/generate`,
			{}
		);
	},

	listBills(societyId: string, filter?: ListBillsFilter): Promise<ListBillsResponse> {
		const params = new URLSearchParams();
		if (filter?.period_id) params.set('period_id', filter.period_id);
		if (filter?.unit_id) params.set('unit_id', filter.unit_id);
		if (filter?.status) params.set('status', filter.status);
		if (filter?.limit) params.set('limit', String(filter.limit));
		if (filter?.offset) params.set('offset', String(filter.offset));
		const qs = params.toString() ? `?${params.toString()}` : '';
		return api.get<ListBillsResponse>(`/societies/${societyId}/maintenance/bills${qs}`);
	},

	getBillDetails(societyId: string, billId: string): Promise<{ bill: MaintenanceBillDetail }> {
		return api.get<{ bill: MaintenanceBillDetail }>(
			`/societies/${societyId}/maintenance/bills/${billId}`
		);
	},

	voidBill(societyId: string, billId: string, reason: string): Promise<void> {
		return api.post<void>(`/societies/${societyId}/maintenance/bills/${billId}/void`, { reason });
	},

	listResidentBills(
		societyId: string,
		filter?: { limit?: number; offset?: number }
	): Promise<ListResidentBillsResponse> {
		const params = new URLSearchParams();
		if (filter?.limit) params.set('limit', String(filter.limit));
		if (filter?.offset) params.set('offset', String(filter.offset));
		const qs = params.toString() ? `?${params.toString()}` : '';
		return api.get<ListResidentBillsResponse>(`/societies/${societyId}/maintenance/my-bills${qs}`);
	},

	getResidentBillDetails(
		societyId: string,
		billId: string
	): Promise<{ bill: MaintenanceBillDetail }> {
		return api.get<{ bill: MaintenanceBillDetail }>(
			`/societies/${societyId}/maintenance/my-bills/${billId}`
		);
	},

	recordPayment(
		societyId: string,
		payload: RecordPaymentPayload,
		idempotencyKey?: string
	): Promise<RecordPaymentResult> {
		const headers: Record<string, string> = {};
		if (idempotencyKey) {
			headers['Idempotency-Key'] = idempotencyKey;
		}
		return api.post<RecordPaymentResult>(`/societies/${societyId}/maintenance/payments`, payload, {
			headers
		});
	},

	reversePayment(
		societyId: string,
		paymentId: string,
		reason: string
	): Promise<{ message: string }> {
		return api.post<{ message: string }>(
			`/societies/${societyId}/maintenance/payments/${paymentId}/reverse`,
			{ reason }
		);
	},

	listCollectionRegister(
		societyId: string,
		filter?: ListCollectionRegisterFilter
	): Promise<ListCollectionRegisterResponse> {
		const params = new URLSearchParams();
		if (filter?.period_id) params.set('period_id', filter.period_id);
		if (filter?.status) params.set('status', filter.status);
		if (filter?.search) params.set('search', filter.search);
		if (filter?.limit) params.set('limit', String(filter.limit));
		if (filter?.offset) params.set('offset', String(filter.offset));
		const qs = params.toString() ? `?${params.toString()}` : '';
		return api.get<ListCollectionRegisterResponse>(
			`/societies/${societyId}/maintenance/collections/register${qs}`
		);
	},

	getCollectionSummary(
		societyId: string,
		periodId?: string
	): Promise<{ summary: CollectionSummary }> {
		const qs = periodId ? `?period_id=${periodId}` : '';
		return api.get<{ summary: CollectionSummary }>(
			`/societies/${societyId}/maintenance/collections/summary${qs}`
		);
	},

	getReceiptDetails(
		societyId: string,
		receiptId: string
	): Promise<{ receipt: MaintenanceReceipt }> {
		return api.get<{ receipt: MaintenanceReceipt }>(
			`/societies/${societyId}/maintenance/receipts/${receiptId}`
		);
	},

	listResidentPayments(
		societyId: string,
		filter?: { limit?: number; offset?: number }
	): Promise<ListResidentPaymentsResponse> {
		const params = new URLSearchParams();
		if (filter?.limit) params.set('limit', String(filter.limit));
		if (filter?.offset) params.set('offset', String(filter.offset));
		const qs = params.toString() ? `?${params.toString()}` : '';
		return api.get<ListResidentPaymentsResponse>(
			`/societies/${societyId}/maintenance/my-payments${qs}`
		);
	},

	getResidentReceipt(
		societyId: string,
		receiptId: string
	): Promise<{ receipt: MaintenanceReceipt }> {
		return api.get<{ receipt: MaintenanceReceipt }>(
			`/societies/${societyId}/maintenance/my-receipts/${receiptId}`
		);
	},

	// S13 Documents & Records Methods
	listDocumentCategories(
		societyId: string,
		includeInactive?: boolean
	): Promise<{ categories: DocumentCategory[] }> {
		const qs = includeInactive ? '?include_inactive=true' : '';
		return api.get<{ categories: DocumentCategory[] }>(
			`/societies/${societyId}/documents/categories${qs}`
		);
	},

	createDocumentCategory(
		societyId: string,
		payload: { name: string; description?: string; display_order?: number }
	): Promise<{ category: DocumentCategory }> {
		return api.post<{ category: DocumentCategory }>(
			`/societies/${societyId}/documents/categories`,
			payload
		);
	},

	listDocuments(societyId: string, params?: ListDocumentsFilter): Promise<ListDocumentsResponse> {
		const query = new URLSearchParams();
		if (params?.category_id) query.append('category_id', params.category_id);
		if (params?.classification) query.append('classification', params.classification);
		if (params?.status) query.append('status', params.status);
		if (params?.search) query.append('search', params.search);
		if (params?.building_id) query.append('building_id', params.building_id);
		if (params?.wing_id) query.append('wing_id', params.wing_id);
		if (params?.unit_id) query.append('unit_id', params.unit_id);
		if (params?.member_id) query.append('member_id', params.member_id);
		if (params?.limit !== undefined) query.append('limit', String(params.limit));
		if (params?.offset !== undefined) query.append('offset', String(params.offset));

		const qs = query.toString() ? `?${query.toString()}` : '';
		return api.get<ListDocumentsResponse>(`/societies/${societyId}/documents${qs}`);
	},

	getDocumentDetails(
		societyId: string,
		documentId: string
	): Promise<{ document: SocietyDocument }> {
		return api.get<{ document: SocietyDocument }>(
			`/societies/${societyId}/documents/${documentId}`
		);
	},

	uploadDocument(societyId: string, formData: FormData): Promise<{ document: SocietyDocument }> {
		return api.request<{ document: SocietyDocument }>(`/societies/${societyId}/documents`, {
			method: 'POST',
			body: formData
		});
	},

	updateDocument(
		societyId: string,
		documentId: string,
		payload: UpdateDocumentPayload
	): Promise<{ document: SocietyDocument }> {
		return api.put<{ document: SocietyDocument }>(
			`/societies/${societyId}/documents/${documentId}`,
			payload
		);
	},

	uploadDocumentVersion(
		societyId: string,
		documentId: string,
		formData: FormData
	): Promise<{ version: DocumentVersion }> {
		return api.request<{ version: DocumentVersion }>(
			`/societies/${societyId}/documents/${documentId}/versions`,
			{
				method: 'POST',
				body: formData
			}
		);
	},

	listDocumentVersions(
		societyId: string,
		documentId: string
	): Promise<{ versions: DocumentVersion[] }> {
		return api.get<{ versions: DocumentVersion[] }>(
			`/societies/${societyId}/documents/${documentId}/versions`
		);
	},

	downloadDocumentVersion(
		societyId: string,
		documentId: string,
		versionId: string,
		inline?: boolean
	): Promise<Blob> {
		const qs = inline ? '?inline=true' : '';
		return api.requestBlob(
			`/societies/${societyId}/documents/${documentId}/versions/${versionId}/download${qs}`
		);
	},

	archiveDocument(
		societyId: string,
		documentId: string,
		reason: string
	): Promise<{ document: SocietyDocument }> {
		return api.post<{ document: SocietyDocument }>(
			`/societies/${societyId}/documents/${documentId}/archive`,
			{ reason }
		);
	},

	restoreDocument(societyId: string, documentId: string): Promise<{ document: SocietyDocument }> {
		return api.post<{ document: SocietyDocument }>(
			`/societies/${societyId}/documents/${documentId}/restore`,
			{}
		);
	},

	getDocumentStorageMetrics(societyId: string): Promise<{ metrics: DocumentStorageMetrics }> {
		return api.get<{ metrics: DocumentStorageMetrics }>(
			`/societies/${societyId}/documents/metrics`
		);
	},

	// S14 Society Audit & Activity
	listSocietyAuditLogs(
		societyId: string,
		params?: {
			page?: number;
			limit?: number;
			offset?: number;
			action?: string;
			search?: string;
			actor_id?: string;
			target_user_id?: string;
			start_date?: string;
			end_date?: string;
		}
	): Promise<AuditListResponse> {
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
		if (params?.start_date) queryParams.start_date = params.start_date;
		if (params?.end_date) queryParams.end_date = params.end_date;

		return api.get<AuditListResponse>(`/societies/${societyId}/audit-logs`, {
			params: queryParams
		});
	},

	getSocietyAuditLog(societyId: string, eventId: string): Promise<{ item: AuditLogItem }> {
		return api.get<{ item: AuditLogItem }>(`/societies/${societyId}/audit-logs/${eventId}`);
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
	city?: string;
	state?: string;
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
	directionality: 'BIDIRECTIONAL' | 'INGRESS_ONLY' | 'EGRESS_ONLY';
	operational_status: 'ACTIVE' | 'INACTIVE' | 'MAINTENANCE' | 'EMERGENCY_OPEN';
	required_staff_per_shift: number;
	description?: string;
	created_at: string;
	updated_at: string;
}

export interface UpdateGatePayload {
	name?: string;
	directionality?: 'BIDIRECTIONAL' | 'INGRESS_ONLY' | 'EGRESS_ONLY';
	operational_status?: 'ACTIVE' | 'INACTIVE' | 'MAINTENANCE' | 'EMERGENCY_OPEN';
	required_staff_per_shift?: number;
	description?: string;
}

export interface GateShift {
	id: string;
	society_id: string;
	name: string;
	start_time: string;
	end_time: string;
	is_active: boolean;
	created_at: string;
	updated_at: string;
}

export interface CreateGateShiftPayload {
	name: string;
	start_time: string;
	end_time: string;
}

export interface GateDutySession {
	id: string;
	society_id: string;
	gate_id: string;
	gate_name: string;
	gate_code: string;
	user_id: string;
	user_phone: string;
	shift_id?: string;
	shift_name?: string;
	started_at: string;
	ended_at?: string;
	status: 'ACTIVE' | 'ENDED';
	handover_notes?: string;
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

// S06 Member & Unit Relationship Types
export interface SocietyMember {
	id: string;
	society_id: string;
	user_id?: string | null;
	first_name: string;
	last_name?: string | null;
	phone?: string;
	phone_masked: string;
	email?: string | null;
	status: 'ACTIVE' | 'INACTIVE' | 'LEFT';
	emergency_contact_name?: string | null;
	emergency_contact_phone?: string | null;
	unit_numbers?: string | null;
	active_units_count?: number;
	departure_reason_code?: DepartureReasonCode | null;
	departure_reason_details?: string | null;
	departed_at?: string | null;
	created_at: string;
	updated_at: string;
}

export type RelationshipType =
	| 'PRIMARY_OWNER'
	| 'SECONDARY_OWNER'
	| 'OWNER_FAMILY'
	| 'PRIMARY_TENANT'
	| 'SECONDARY_TENANT'
	| 'TENANT_FAMILY'
	| 'AUTHORIZED_OCCUPANT';

export interface UnitRelationship {
	id: string;
	society_id: string;
	unit_id: string;
	member_id: string;
	relationship_type: RelationshipType;
	is_occupant: boolean;
	ownership_share?: number | null;
	status: 'ACTIVE' | 'ENDED';
	start_date?: string | null;
	end_date?: string | null;
	notes?: string | null;
	created_at: string;
	updated_at: string;
	// Joined unit or member info
	first_name?: string;
	last_name?: string;
	phone?: string;
	phone_masked?: string;
	unit_identifier?: string;
	building_name?: string;
	floor_number?: number;
}

export interface MemberListResponse {
	members: SocietyMember[];
	total: number;
	limit: number;
	offset: number;
}

export interface RegisterMemberPayload {
	first_name: string;
	last_name?: string;
	phone: string;
	email?: string;
	emergency_contact_name?: string;
	emergency_contact_phone?: string;
}

export interface UpdateMemberPayload {
	first_name?: string;
	last_name?: string;
	email?: string;
	emergency_contact_name?: string;
	emergency_contact_phone?: string;
	status?: 'ACTIVE' | 'INACTIVE';
}

export interface CreateRelationshipPayload {
	member_id: string;
	relationship_type: string;
	is_occupant: boolean;
	ownership_share?: number;
	start_date?: string;
	notes?: string;
}

export interface EndRelationshipPayload {
	end_date: string;
	reason?: string;
}

export interface PreservedOwnershipItem {
	relationship_id: string;
	unit_identifier: string;
	building_name: string;
	relationship_type: string;
	ownership_share?: number | null;
	action: string;
	note: string;
}

export interface TargetedTerminationItem {
	relationship_id: string;
	unit_identifier: string;
	building_name: string;
	relationship_type: string;
	action: string;
	proposed_end_date: string;
}

export interface DeparturePreviewResponse {
	member_id: string;
	current_status: string;
	version_hash: string;
	preserved_ownership_relationships: PreservedOwnershipItem[];
	targeted_terminations: TargetedTerminationItem[];
}

export type DepartureReasonCode =
	| 'TENANT_MOVED_OUT'
	| 'LEASE_EXPIRED'
	| 'OWNERSHIP_TRANSFERRED'
	| 'MEMBER_RELOCATED'
	| 'FAMILY_MEMBER_DEPARTED'
	| 'MEMBERSHIP_TERMINATED'
	| 'DECEASED'
	| 'OTHER';

export interface DepartureReasonOption {
	code: DepartureReasonCode;
	label: string;
}

export const DEPARTURE_REASON_OPTIONS: DepartureReasonOption[] = [
	{ code: 'TENANT_MOVED_OUT', label: 'Tenant Moved Out' },
	{ code: 'LEASE_EXPIRED', label: 'Lease Expired' },
	{ code: 'OWNERSHIP_TRANSFERRED', label: 'Ownership Transferred' },
	{ code: 'MEMBER_RELOCATED', label: 'Member Relocated' },
	{ code: 'FAMILY_MEMBER_DEPARTED', label: 'Family Member Departed' },
	{ code: 'MEMBERSHIP_TERMINATED', label: 'Membership Terminated' },
	{ code: 'DECEASED', label: 'Deceased' },
	{ code: 'OTHER', label: 'Other' }
];

export interface ConfirmDeparturePayload {
	expected_version_hash: string;
	end_date: string;
	reason_code: DepartureReasonCode;
	reason_details?: string;
}

// S07: Designations & Committee Appointments
export interface SocietyDesignation {
	id: string;
	society_id: string;
	code: string;
	name: string;
	description?: string;
	is_system_default: boolean;
	is_active: boolean;
	allows_multiple_members: boolean;
	allows_multiple_designations: boolean;
	display_order: number;
	created_at: string;
	updated_at: string;
}

export interface CommitteeAppointment {
	id: string;
	society_id: string;
	member_id: string;
	designation_id: string;
	start_date: string;
	expected_end_date?: string | null;
	end_date?: string | null;
	status: 'ACTIVE' | 'ENDED' | 'REVOKED';
	notes?: string | null;
	appointed_by?: string;
	ended_by?: string | null;
	created_at: string;
	updated_at: string;
	// Joined fields
	member_name?: string;
	member_phone?: string;
	member_phone_masked?: string;
	member_status?: string;
	designation_code?: string;
	designation_name?: string;
	allows_multiple_members?: boolean;
}

export interface OfficeBearerInfo {
	designation_id: string;
	designation_code?: string;
	designation_name: string;
	appointment_id?: string;
	member_id?: string;
	member_name?: string;
	member_phone_masked?: string;
	start_date?: string;
	expected_end_date?: string;
	status?: string;
}

export interface CommitteeOverview {
	total_designations?: number;
	active_designations?: number;
	active_appointments?: number;
	key_positions_filled?: Record<string, boolean>;
	current_appointments?: CommitteeAppointment[];
	office_bearers: OfficeBearerInfo[];
	total_active_appointments?: number;
	active_designations_count?: number;
}

export interface CreateDesignationPayload {
	code: string;
	name: string;
	description?: string;
	allows_multiple_members?: boolean;
	allows_multiple_designations?: boolean;
	display_order?: number;
}

export interface UpdateDesignationPayload {
	name?: string;
	description?: string;
	allows_multiple_members?: boolean;
	allows_multiple_designations?: boolean;
	display_order?: number;
}

export interface CreateAppointmentPayload {
	member_id: string;
	designation_id: string;
	start_date: string;
	end_date?: string;
	notes?: string;
}

export interface EndAppointmentPayload {
	end_date: string;
	status?: 'ENDED' | 'REVOKED';
	notes?: string;
}

// S09 Visitor Models
export interface Visit {
	id: string;
	society_id: string;
	unit_id: string;
	unit_identifier: string;
	building_name: string;
	resident_id?: string;
	resident_name?: string;
	visitor_name: string;
	visitor_phone: string;
	visitor_phone_masked: string;
	purpose: string;
	visit_type: 'EXPECTED' | 'WALK_IN';
	status:
		| 'PENDING_APPROVAL'
		| 'APPROVED'
		| 'REJECTED'
		| 'CHECKED_IN'
		| 'CHECKED_OUT'
		| 'CANCELLED'
		| 'EXPIRED';
	vehicle_number?: string;
	expected_arrival?: string;
	valid_until?: string;
	check_in_at?: string;
	check_out_at?: string;
	entry_gate_id?: string;
	entry_gate_name?: string;
	exit_gate_id?: string;
	exit_gate_name?: string;
	rejection_reason?: string;
	created_by: string;
	created_at: string;
	updated_at: string;
}

export interface VisitorPass {
	id: string;
	society_id: string;
	visit_id: string;
	pass_code: string;
	valid_from: string;
	valid_until: string;
	status: 'ACTIVE' | 'CONSUMED' | 'REVOKED' | 'EXPIRED';
	consumed_at?: string;
	revoked_at?: string;
	bearer_token?: string;
	created_at: string;
	updated_at: string;
}

export interface VisitApproval {
	id: string;
	society_id: string;
	visit_id: string;
	decision: 'APPROVED' | 'REJECTED';
	decided_by: string;
	decided_by_name?: string;
	notes?: string;
	created_at: string;
}

export interface GateEvent {
	id: string;
	society_id: string;
	visit_id: string;
	gate_id: string;
	gate_name: string;
	session_id?: string;
	gatekeeper_id: string;
	event_type: 'ENTRY' | 'EXIT';
	direction: 'INGRESS' | 'EGRESS';
	vehicle_number?: string;
	notes?: string;
	created_at: string;
}

export interface VisitorDashboardMetrics {
	expected_today: number;
	awaiting_approval: number;
	currently_inside: number;
	completed_today: number;
}

export interface RegisterExpectedVisitPayload {
	unit_id: string;
	visitor_name: string;
	visitor_phone: string;
	purpose: string;
	vehicle_number?: string;
	expected_arrival?: string;
	valid_until?: string;
	requires_approval?: boolean;
}

export interface RegisterWalkInVisitPayload {
	unit_id: string;
	visitor_name: string;
	visitor_phone: string;
	purpose: string;
	vehicle_number?: string;
	requires_approval?: boolean;
}

export interface CheckInVisitorPayload {
	gate_id: string;
	vehicle_number?: string;
	notes?: string;
}

export interface CheckOutVisitorPayload {
	gate_id: string;
	notes?: string;
}

export interface ResidentUnit {
	id: string;
	society_id: string;
	building_id: string;
	building_name: string;
	wing_id?: string;
	wing_code?: string;
	floor_number: number;
	unit_number: string;
	unit_identifier: string;
	unit_type: string;
	relationship_type: string;
	is_occupant: boolean;
	requires_visitor_approval: boolean;
}

// S10 Communication Models
export type AnnouncementCategory =
	'GENERAL' | 'MAINTENANCE' | 'SECURITY' | 'EVENT' | 'EMERGENCY' | 'ADMINISTRATIVE';

export type AnnouncementPriority = 'NORMAL' | 'URGENT' | 'CRITICAL';
export type AnnouncementStatus = 'DRAFT' | 'PUBLISHED' | 'ARCHIVED';
export type AnnouncementAudienceType = 'SOCIETY' | 'BUILDING' | 'WING' | 'UNIT';

export interface Announcement {
	id: string;
	society_id: string;
	title: string;
	body: string;
	category: AnnouncementCategory;
	priority: AnnouncementPriority;
	status: AnnouncementStatus;
	audience_type: AnnouncementAudienceType;
	target_building_id?: string | null;
	target_wing_id?: string | null;
	target_unit_id?: string | null;
	target_label?: string | null;
	created_by: string;
	published_by?: string | null;
	published_at?: string | null;
	archived_at?: string | null;
	created_at: string;
	updated_at: string;
	// Resident feed enrichment
	is_read?: boolean;
	read_at?: string | null;
}

export interface CreateAnnouncementPayload {
	title: string;
	body: string;
	category?: AnnouncementCategory;
	priority?: AnnouncementPriority;
	audience_type: AnnouncementAudienceType;
	target_building_id?: string;
	target_wing_id?: string;
	target_unit_id?: string;
}

export interface UpdateAnnouncementPayload {
	title?: string;
	body?: string;
	category?: AnnouncementCategory;
	priority?: AnnouncementPriority;
	audience_type?: AnnouncementAudienceType;
	target_building_id?: string;
	target_wing_id?: string;
	target_unit_id?: string;
}

export interface AnnouncementFeedFilter {
	category?: string;
	unread_only?: boolean;
	search?: string;
	limit?: number;
	offset?: number;
}

export interface AdminAnnouncementFilter {
	status?: string;
	category?: string;
	search?: string;
	limit?: number;
	offset?: number;
}

export interface AnnouncementFeedResponse {
	announcements: Announcement[];
	total: number;
	unread_count: number;
	limit: number;
	offset: number;
}

export interface AdminAnnouncementListResponse {
	announcements: Announcement[];
	total: number;
	limit: number;
	offset: number;
}

// ==========================================
// S11 Maintenance Foundation Interfaces
// ==========================================

export interface MaintenanceChargeCategory {
	id: string;
	society_id: string;
	name: string;
	description?: string;
	status: 'ACTIVE' | 'INACTIVE';
	created_at: string;
	updated_at: string;
}

export interface CreateChargeCategoryPayload {
	name: string;
	description?: string;
}

export interface UpdateChargeCategoryPayload {
	name: string;
	description?: string;
	status?: 'ACTIVE' | 'INACTIVE';
}

export interface MaintenanceChargeConfig {
	id: string;
	society_id: string;
	category_id: string;
	category_name?: string;
	name: string;
	charge_type: 'FLAT_RECURRING' | 'PER_UNIT_RECURRING' | 'ONE_TIME';
	applicable_unit_type?: string;
	amount: number;
	billing_frequency: 'MONTHLY' | 'QUARTERLY' | 'ANNUAL' | 'ONE_TIME';
	effective_from: string;
	effective_until?: string | null;
	status: 'ACTIVE' | 'INACTIVE';
	created_at: string;
	updated_at: string;
}

export interface CreateChargeConfigPayload {
	category_id: string;
	name: string;
	charge_type: 'FLAT_RECURRING' | 'PER_UNIT_RECURRING' | 'ONE_TIME';
	applicable_unit_type?: string;
	amount: number;
	billing_frequency: 'MONTHLY' | 'QUARTERLY' | 'ANNUAL' | 'ONE_TIME';
	effective_from: string;
	effective_until?: string;
}

export interface MaintenanceBillingPeriod {
	id: string;
	society_id: string;
	period_name: string;
	period_start: string;
	period_end: string;
	due_date: string;
	status: 'ACTIVE' | 'CLOSED';
	created_by: string;
	created_at: string;
	updated_at: string;
}

export interface CreateBillingPeriodPayload {
	period_name: string;
	period_start: string;
	period_end: string;
	due_date: string;
}

export interface MaintenanceBillItem {
	id: string;
	bill_id: string;
	society_id: string;
	charge_category_id?: string;
	category_name: string;
	charge_config_id?: string;
	description: string;
	rate: number;
	quantity: number;
	amount: number;
	created_at: string;
}

export interface MaintenanceBill {
	id: string;
	society_id: string;
	unit_id: string;
	unit_identifier?: string;
	building_name?: string;
	wing_name?: string;
	billing_period_id: string;
	period_name?: string;
	bill_number: string;
	bill_date: string;
	due_date: string;
	currency: string;
	subtotal: number;
	total_amount: number;
	status: 'ISSUED' | 'OVERDUE' | 'VOIDED';
	raw_status: string;
	void_reason?: string;
	created_by: string;
	created_at: string;
	updated_at: string;
}

export interface MaintenanceBillDetail extends MaintenanceBill {
	items: MaintenanceBillItem[];
}

export interface BillGenerationPreview {
	period_id: string;
	period_name: string;
	period_start: string;
	period_end: string;
	due_date: string;
	eligible_unit_count: number;
	already_billed_count: number;
	units_to_bill_count: number;
	active_configs_count: number;
	estimated_total_per_unit: number;
	estimated_grand_total: number;
}

export interface BillGenerationResult {
	period_id: string;
	bills_created: number;
	units_skipped: number;
	total_billed_amount: number;
}

export interface ListBillsFilter {
	period_id?: string;
	unit_id?: string;
	status?: string;
	limit?: number;
	offset?: number;
}

export interface ListBillsResponse {
	items: MaintenanceBill[];
	total: number;
	limit: number;
	offset: number;
}

export interface ListResidentBillsResponse {
	bills: MaintenanceBill[];
	total: number;
}

export type PaymentMethod = 'CASH' | 'CHEQUE' | 'BANK_TRANSFER' | 'UPI_RECORDED_OFFLINE' | 'OTHER';

export type PaymentStatus = 'RECORDED' | 'REVERSED';
export type AllocationStatus = 'ACTIVE' | 'REVERSED';
export type ReceiptStatus = 'ISSUED' | 'REVERSED';
export type CollectionStatus = 'UNPAID' | 'PARTIALLY_PAID' | 'PAID' | 'VOIDED';

export interface MaintenancePayment {
	id: string;
	society_id: string;
	payment_reference: string;
	payment_method: PaymentMethod | string;
	amount: number;
	currency: string;
	payment_date: string;
	external_reference?: string;
	status: PaymentStatus | string;
	recorded_by: string;
	payer_name?: string;
	notes?: string;
	reversed_at?: string;
	reversed_by?: string;
	reversal_reason?: string;
	idempotency_key?: string;
	created_at: string;
	updated_at: string;
}

export interface MaintenancePaymentAllocation {
	id: string;
	society_id: string;
	payment_id: string;
	bill_id: string;
	allocated_amount: number;
	status: AllocationStatus | string;
	allocated_by: string;
	reversed_at?: string;
	created_at: string;
}

export interface MaintenanceReceipt {
	id: string;
	society_id: string;
	payment_id: string;
	receipt_number: string;
	receipt_date: string;
	total_amount: number;
	currency: string;
	issued_by: string;
	status: ReceiptStatus | string;
	reversed_at?: string;
	created_at: string;
	society_name?: string;
	society_code?: string;
	payment_reference?: string;
	payment_method?: string;
	external_reference?: string;
	payer_name?: string;
	bill_number?: string;
	unit_identifier?: string;
}

export interface RecordPaymentPayload {
	bill_id: string;
	amount: number;
	payment_method: string;
	payment_date: string;
	external_reference?: string;
	payer_name?: string;
	notes?: string;
	idempotency_key?: string;
}

export interface RecordPaymentResult {
	payment: MaintenancePayment;
	receipt: MaintenanceReceipt;
	allocation: MaintenancePaymentAllocation;
	remaining_balance: number;
}

export interface CollectionRegisterItem {
	bill_id: string;
	bill_number: string;
	unit_id: string;
	unit_identifier: string;
	period_id: string;
	period_name: string;
	due_date: string;
	total_amount: number;
	total_paid: number;
	outstanding_balance: number;
	bill_status: string;
	derived_status: string;
	collection_status: string;
	last_payment_date?: string;
}

export interface ListCollectionRegisterFilter {
	period_id?: string;
	status?: string;
	search?: string;
	limit?: number;
	offset?: number;
}

export interface ListCollectionRegisterResponse {
	items: CollectionRegisterItem[];
	total: number;
	limit: number;
	offset: number;
}

export interface CollectionSummary {
	total_billed: number;
	total_collected: number;
	total_outstanding: number;
	unpaid_bills_count: number;
	partially_paid_bills_count: number;
	paid_bills_count: number;
	reversed_payments_count: number;
	reversed_amount: number;
}

export interface ResidentPaymentItem {
	payment_id: string;
	payment_reference: string;
	receipt_id: string;
	receipt_number: string;
	bill_id: string;
	bill_number: string;
	unit_identifier: string;
	amount: number;
	payment_method: string;
	payment_date: string;
	status: string;
	created_at: string;
}

export interface ListResidentPaymentsResponse {
	payments: ResidentPaymentItem[];
	total: number;
	limit: number;
	offset: number;
}

// S13 Document Models
export type DocumentClassification = 'SOCIETY_RESIDENTS' | 'COMMITTEE' | 'SOC_ADMIN_ONLY';
export type DocumentStatus = 'ACTIVE' | 'ARCHIVED';

export interface DocumentCategory {
	id: string;
	society_id: string;
	name: string;
	description?: string;
	is_system: boolean;
	is_active: boolean;
	display_order: number;
	document_count?: number;
	created_at: string;
	updated_at: string;
}

export interface DocumentVersion {
	id: string;
	society_id: string;
	document_id: string;
	version_number: number;
	storage_path: string;
	file_name: string;
	file_size: number;
	mime_type: string;
	file_hash: string;
	change_summary?: string;
	uploaded_by: string;
	uploaded_by_name?: string;
	created_at: string;
}

export interface SocietyDocument {
	id: string;
	society_id: string;
	category_id: string;
	category_name?: string;
	document_number: string;
	title: string;
	description?: string;
	classification: DocumentClassification;
	status: DocumentStatus;
	current_version_id?: string;
	current_version?: DocumentVersion;
	building_id?: string;
	building_name?: string;
	wing_id?: string;
	wing_name?: string;
	unit_id?: string;
	unit_number?: string;
	member_id?: string;
	member_name?: string;
	created_by: string;
	created_by_name?: string;
	updated_by: string;
	created_at: string;
	updated_at: string;
	archived_at?: string;
	archived_by?: string;
	archived_reason?: string;
	version_count?: number;
}

export interface ListDocumentsFilter {
	category_id?: string;
	classification?: string;
	status?: string;
	search?: string;
	building_id?: string;
	wing_id?: string;
	unit_id?: string;
	member_id?: string;
	limit?: number;
	offset?: number;
}

export interface ListDocumentsResponse {
	documents: SocietyDocument[];
	total: number;
	limit: number;
	offset: number;
}

export interface UpdateDocumentPayload {
	title: string;
	description?: string;
	category_id: string;
	classification: DocumentClassification;
	building_id?: string;
	wing_id?: string;
	unit_id?: string;
	member_id?: string;
}

export interface DocumentStorageMetrics {
	total_documents: number;
	active_documents: number;
	archived_documents: number;
	total_versions: number;
	total_stored_bytes: number;
}
