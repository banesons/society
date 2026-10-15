import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
	societyApi,
	type SocietyItem,
	type CreateSocietyPayload,
	type DeparturePreviewResponse
} from './society';
import { api, ApiClientError } from './client';

describe('Society Management Frontend API', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	const mockSociety: SocietyItem = {
		id: '11111111-1111-1111-1111-111111111111',
		society_code: 'SOC-MUM-7K4P2X',
		name: 'Palm Grove Residency',
		registration_number: 'REG/MH/2026/001',
		address_line1: 'Plot 42, Sector 18',
		address_line2: 'Near Central Park',
		locality: 'Kharghar',
		city: 'Navi Mumbai',
		state: 'Maharashtra',
		pincode: '410210',
		contact_email: 'contact@palmgrove.in',
		contact_phone: '+919876543210',
		status: 'PENDING',
		created_by: '22222222-2222-2222-2222-222222222222',
		updated_by: '22222222-2222-2222-2222-222222222222',
		created_at: '2026-10-10T12:00:00Z',
		updated_at: '2026-10-10T12:00:00Z'
	};

	describe('societyApi.listSocieties', () => {
		it('calls GET /platform/societies with query params', async () => {
			const mockResponse = {
				items: [mockSociety],
				total: 1,
				limit: 10,
				offset: 0
			};

			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockResponse);

			const result = await societyApi.listSocieties({
				limit: 10,
				offset: 0,
				search: 'Palm Grove',
				status: 'PENDING'
			});

			expect(result).toEqual(mockResponse);
			expect(getSpy).toHaveBeenCalledWith('/platform/societies', {
				params: {
					limit: 10,
					offset: 0,
					search: 'Palm Grove',
					status: 'PENDING'
				}
			});
		});

		it('propagates API client errors', async () => {
			vi.spyOn(api, 'get').mockRejectedValue(
				new ApiClientError('Forbidden', 'FORBIDDEN', 403, 'req-403')
			);

			await expect(societyApi.listSocieties()).rejects.toThrow('Forbidden');
		});
	});

	describe('societyApi.createSociety', () => {
		it('calls POST /platform/societies with valid payload and returns created society', async () => {
			const payload: CreateSocietyPayload = {
				name: 'Palm Grove Residency',
				registration_number: 'REG/MH/2026/001',
				address_line1: 'Plot 42, Sector 18',
				locality: 'Kharghar',
				city: 'Navi Mumbai',
				state: 'Maharashtra',
				pincode: '410210',
				contact_email: 'contact@palmgrove.in',
				contact_phone: '+919876543210'
			};

			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(mockSociety);

			const result = await societyApi.createSociety(payload);

			expect(result).toEqual(mockSociety);
			expect(result.society_code).toBe('SOC-MUM-7K4P2X');
			expect(result.status).toBe('PENDING');
			expect(postSpy).toHaveBeenCalledWith('/platform/societies', payload);
		});

		it('handles validation error from backend', async () => {
			vi.spyOn(api, 'post').mockRejectedValue(
				new ApiClientError('Invalid PIN code', 'INVALID_PINCODE', 400, 'req-400')
			);

			await expect(
				societyApi.createSociety({
					name: 'Bad Society',
					address_line1: 'Address',
					locality: 'Locality',
					city: 'City',
					state: 'State',
					pincode: '000000'
				})
			).rejects.toThrow('Invalid PIN code');
		});
	});

	describe('societyApi.getSociety', () => {
		it('calls GET /platform/societies/:id', async () => {
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockSociety);

			const result = await societyApi.getSociety(mockSociety.id);

			expect(result).toEqual(mockSociety);
			expect(getSpy).toHaveBeenCalledWith(`/platform/societies/${mockSociety.id}`);
		});
	});

	describe('societyApi.updateSociety', () => {
		it('calls PATCH /platform/societies/:id with update payload', async () => {
			const updatePayload = {
				name: 'Palm Grove Heights',
				address_line1: 'Updated Line 1'
			};
			const updatedSociety = { ...mockSociety, ...updatePayload };

			const patchSpy = vi.spyOn(api, 'patch').mockResolvedValue(updatedSociety);

			const result = await societyApi.updateSociety(mockSociety.id, updatePayload);

			expect(result).toEqual(updatedSociety);
			expect(patchSpy).toHaveBeenCalledWith(`/platform/societies/${mockSociety.id}`, updatePayload);
		});
	});

	describe('Lifecycle transitions', () => {
		it('calls POST /platform/societies/:id/activate', async () => {
			const activatedSociety = { ...mockSociety, status: 'ACTIVE' as const };
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(activatedSociety);

			const result = await societyApi.activateSociety(mockSociety.id);

			expect(result).toEqual(activatedSociety);
			expect(postSpy).toHaveBeenCalledWith(`/platform/societies/${mockSociety.id}/activate`, {
				reason: undefined
			});
		});

		it('calls GET /platform/society-suspension-reasons', async () => {
			const mockReasons = [
				{
					id: 1,
					code: 'PAYMENT_OVERDUE',
					label: 'Payment overdue',
					category: 'PAYMENT'
				}
			];
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockReasons);

			const result = await societyApi.getSuspensionReasons();

			expect(result).toEqual(mockReasons);
			expect(getSpy).toHaveBeenCalledWith('/platform/society-suspension-reasons');
		});

		it('calls POST /platform/societies/:id/suspend with reason_id and reason_details', async () => {
			const suspendedSociety = { ...mockSociety, status: 'SUSPENDED' as const };
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(suspendedSociety);

			const result = await societyApi.suspendSociety(mockSociety.id, {
				reason_id: 1,
				reason_details: 'Q3 dues unpaid'
			});

			expect(result).toEqual(suspendedSociety);
			expect(postSpy).toHaveBeenCalledWith(`/platform/societies/${mockSociety.id}/suspend`, {
				reason_id: 1,
				reason_details: 'Q3 dues unpaid'
			});
		});

		it('calls POST /platform/societies/:id/reactivate', async () => {
			const reactivatedSociety = { ...mockSociety, status: 'ACTIVE' as const };
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(reactivatedSociety);

			const result = await societyApi.reactivateSociety(mockSociety.id);

			expect(result).toEqual(reactivatedSociety);
			expect(postSpy).toHaveBeenCalledWith(`/platform/societies/${mockSociety.id}/reactivate`, {
				reason: undefined
			});
		});
	});

	describe('S04 Society Administration API', () => {
		const mockAssignment = {
			id: '33333333-3333-3333-3333-333333333333',
			society_id: mockSociety.id,
			user_id: '44444444-4444-4444-4444-444444444444',
			user_phone_masked: '+91 9820xxxx56',
			role: 'SOC_ADMIN',
			status: 'ACTIVE' as const,
			assigned_by: '22222222-2222-2222-2222-222222222222',
			created_at: '2026-10-12T10:00:00Z',
			updated_at: '2026-10-12T10:00:00Z'
		};

		it('calls GET /platform/societies/:id/admins with filters', async () => {
			const mockResponse = {
				items: [mockAssignment],
				total: 1,
				limit: 20,
				offset: 0
			};
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockResponse);

			const result = await societyApi.listSocietyAdmins(mockSociety.id, {
				status: 'ACTIVE',
				limit: 20,
				offset: 0
			});

			expect(result).toEqual(mockResponse);
			expect(getSpy).toHaveBeenCalledWith(`/platform/societies/${mockSociety.id}/admins`, {
				params: { status: 'ACTIVE', limit: 20, offset: 0 }
			});
		});

		it('calls POST /platform/societies/:id/admins and assigns SOC_ADMIN', async () => {
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(mockAssignment);

			const result = await societyApi.assignSocietyAdmin(mockSociety.id, {
				phone: '+919820123456',
				role: 'SOC_ADMIN'
			});

			expect(result).toEqual(mockAssignment);
			expect(result.role).toBe('SOC_ADMIN');
			expect(result.user_phone_masked).toBe('+91 9820xxxx56');
			expect(postSpy).toHaveBeenCalledWith(`/platform/societies/${mockSociety.id}/admins`, {
				phone: '+919820123456',
				role: 'SOC_ADMIN'
			});
		});

		it('propagates USER_NOT_FOUND error on unknown phone', async () => {
			vi.spyOn(api, 'post').mockRejectedValue(
				new ApiClientError('User not found', 'USER_NOT_FOUND', 404, 'req-404')
			);

			await expect(
				societyApi.assignSocietyAdmin(mockSociety.id, { phone: '+919999999999' })
			).rejects.toThrow('User not found');
		});

		it('propagates SOCIETY_ADMIN_ALREADY_ASSIGNED conflict error on duplicate assignment', async () => {
			vi.spyOn(api, 'post').mockRejectedValue(
				new ApiClientError(
					'User already assigned',
					'SOCIETY_ADMIN_ALREADY_ASSIGNED',
					409,
					'req-409'
				)
			);

			await expect(
				societyApi.assignSocietyAdmin(mockSociety.id, { phone: '+919820123456' })
			).rejects.toThrow('User already assigned');
		});

		it('calls DELETE /platform/societies/:id/admins/:assignment_id to revoke assignment', async () => {
			const revoked = {
				...mockAssignment,
				status: 'REVOKED' as const,
				revoked_by: '22222222-2222-2222-2222-222222222222',
				revoked_at: '2026-10-12T11:00:00Z'
			};
			const delSpy = vi.spyOn(api, 'delete').mockResolvedValue(revoked);

			const result = await societyApi.revokeSocietyAdmin(mockSociety.id, mockAssignment.id);

			expect(result.status).toBe('REVOKED');
			expect(delSpy).toHaveBeenCalledWith(
				`/platform/societies/${mockSociety.id}/admins/${mockAssignment.id}`
			);
		});

		it('calls GET /societies/:id/profile and retrieves verified profile', async () => {
			const mockProfile = {
				id: mockSociety.id,
				society_code: mockSociety.society_code,
				name: mockSociety.name,
				status: 'ACTIVE' as const,
				address_line1: mockSociety.address_line1,
				locality: mockSociety.locality,
				city: mockSociety.city,
				state: mockSociety.state,
				pincode: mockSociety.pincode
			};
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockProfile);

			const result = await societyApi.getSocietyProfile(mockSociety.id);

			expect(result).toEqual(mockProfile);
			expect(getSpy).toHaveBeenCalledWith(`/societies/${mockSociety.id}/profile`);
		});
	});

	describe('S05 Society Structure & Formation APIs', () => {
		const socId = mockSociety.id;

		it('calls GET /societies/:id/buildings to list buildings', async () => {
			const mockBuildings = {
				buildings: [
					{
						id: 'b-1',
						society_id: socId,
						name: 'Tower A',
						building_code: 'A',
						total_floors: 9,
						unit_count: 27,
						created_at: '2026-10-15T00:00:00Z',
						updated_at: '2026-10-15T00:00:00Z'
					}
				],
				total: 1
			};
			const spy = vi.spyOn(api, 'get').mockResolvedValue(mockBuildings);

			const result = await societyApi.listBuildings(socId);

			expect(result).toEqual(mockBuildings);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/buildings`);
		});

		it('calls POST /societies/:id/buildings to create building', async () => {
			const payload = { name: 'Tower B', building_code: 'B', total_floors: 12 };
			const mockBuilding = {
				id: 'b-2',
				society_id: socId,
				...payload,
				unit_count: 0,
				created_at: '2026-10-15T00:00:00Z',
				updated_at: '2026-10-15T00:00:00Z'
			};
			const spy = vi.spyOn(api, 'post').mockResolvedValue(mockBuilding);

			const result = await societyApi.createBuilding(socId, payload);

			expect(result).toEqual(mockBuilding);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/buildings`, payload);
		});

		it('calls POST /societies/:id/buildings/:building_id/wings to create wing', async () => {
			const payload = { name: 'Wing A', wing_code: 'A', total_floors: 9 };
			const mockWing = {
				id: 'w-1',
				society_id: socId,
				building_id: 'b-1',
				...payload,
				unit_count: 0,
				created_at: '2026-10-15T00:00:00Z',
				updated_at: '2026-10-15T00:00:00Z'
			};
			const spy = vi.spyOn(api, 'post').mockResolvedValue(mockWing);

			const result = await societyApi.createWing(socId, 'b-1', payload);

			expect(result).toEqual(mockWing);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/buildings/b-1/wings`, payload);
		});

		it('calls POST /societies/:id/units/preview to preview unit generation', async () => {
			const payload = {
				building_id: 'b-1',
				starting_floor: 1,
				total_floors: 9,
				units_per_floor: 3,
				starting_unit_number: 101,
				prefix: 'A'
			};
			const mockPreview = {
				total_units: 27,
				units: [
					{
						floor_number: 1,
						unit_number: '101',
						unit_identifier: 'A101',
						unit_type: 'RESIDENTIAL'
					},
					{ floor_number: 9, unit_number: '903', unit_identifier: 'A903', unit_type: 'RESIDENTIAL' }
				]
			};
			const spy = vi.spyOn(api, 'post').mockResolvedValue(mockPreview);

			const result = await societyApi.previewUnits(socId, payload);

			expect(result.total_units).toBe(27);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/units/preview`, payload);
		});

		it('calls POST /societies/:id/units/generate to persist generated units', async () => {
			const payload = {
				building_id: 'b-1',
				starting_floor: 1,
				total_floors: 9,
				units_per_floor: 3,
				starting_unit_number: 101,
				prefix: 'A'
			};
			const mockResponse = { generated_count: 27, message: 'Units generated successfully' };
			const spy = vi.spyOn(api, 'post').mockResolvedValue(mockResponse);

			const result = await societyApi.generateUnits(socId, payload);

			expect(result.generated_count).toBe(27);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/units/generate`, payload);
		});

		it('calls GET /societies/:id/units with pagination filters', async () => {
			const mockUnits = {
				units: [],
				total: 27,
				limit: 20,
				offset: 0
			};
			const spy = vi.spyOn(api, 'get').mockResolvedValue(mockUnits);

			const result = await societyApi.listUnits(socId, {
				building_id: 'b-1',
				limit: 20,
				offset: 0
			});

			expect(result.total).toBe(27);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/units`, {
				params: { building_id: 'b-1', wing_id: undefined, limit: 20, offset: 0 }
			});
		});

		it('calls GET /societies/:id/gates and POST /societies/:id/gates', async () => {
			const mockGate = {
				id: 'g-1',
				society_id: socId,
				name: 'Main Gate',
				gate_code: 'GATE-01',
				gate_type: 'VEHICLE_AND_PEDESTRIAN',
				status: 'ACTIVE',
				required_staff_per_shift: 2,
				created_at: '2026-10-15T00:00:00Z',
				updated_at: '2026-10-15T00:00:00Z'
			};
			const listSpy = vi.spyOn(api, 'get').mockResolvedValue({ gates: [mockGate], total: 1 });
			const createSpy = vi.spyOn(api, 'post').mockResolvedValue(mockGate);

			const gatesRes = await societyApi.listGates(socId);
			expect(gatesRes.gates).toHaveLength(1);
			expect(listSpy).toHaveBeenCalledWith(`/societies/${socId}/gates`);

			const created = await societyApi.createGate(socId, {
				name: 'Main Gate',
				gate_code: 'GATE-01',
				gate_type: 'VEHICLE_AND_PEDESTRIAN',
				status: 'ACTIVE',
				required_staff_per_shift: 2
			});
			expect(created.gate_code).toBe('GATE-01');
			expect(createSpy).toHaveBeenCalledWith(`/societies/${socId}/gates`, expect.any(Object));
		});

		it('calls GET and PUT /societies/:id/staffing', async () => {
			const mockStaffing = {
				id: 'sr-1',
				society_id: socId,
				shifts_per_day: 3,
				total_required_headcount: 10,
				notes: '24/7 security',
				created_at: '2026-10-15T00:00:00Z',
				updated_at: '2026-10-15T00:00:00Z'
			};
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockStaffing);
			const putSpy = vi.spyOn(api, 'put').mockResolvedValue(mockStaffing);

			const getRes = await societyApi.getStaffing(socId);
			expect(getRes.total_required_headcount).toBe(10);
			expect(getSpy).toHaveBeenCalledWith(`/societies/${socId}/staffing`);

			const putRes = await societyApi.setStaffing(socId, {
				shifts_per_day: 3,
				total_required_headcount: 10,
				notes: '24/7 security'
			});
			expect(putRes.shifts_per_day).toBe(3);
			expect(putSpy).toHaveBeenCalledWith(`/societies/${socId}/staffing`, expect.any(Object));
		});

		it('calls GET /societies/:id/formation-summary', async () => {
			const mockSummary = {
				total_buildings: 1,
				total_wings: 1,
				total_floors: 9,
				total_units: 27,
				total_gates: 1,
				required_staff_per_shift: 2,
				total_required_headcount: 10,
				registered_gatekeepers: 0,
				assigned_gatekeepers: 0
			};
			const spy = vi.spyOn(api, 'get').mockResolvedValue(mockSummary);

			const res = await societyApi.getFormationSummary(socId);

			expect(res.total_units).toBe(27);
			expect(res.registered_gatekeepers).toBe(0);
			expect(res.assigned_gatekeepers).toBe(0);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/formation-summary`);
		});
	});

	describe('S06 Members & Relationships API', () => {
		const socId = 'soc-123';
		const memberId = 'mem-456';
		const unitId = 'unit-789';

		it('calls GET /societies/:id/members with query params', async () => {
			const mockMemberList = {
				members: [
					{
						id: memberId,
						society_id: socId,
						first_name: 'Vikram',
						last_name: 'Malhotra',
						phone_masked: '+91 ******3210',
						status: 'ACTIVE' as const,
						created_at: '2026-10-16T00:00:00Z',
						updated_at: '2026-10-16T00:00:00Z'
					}
				],
				total: 1,
				limit: 50,
				offset: 0
			};
			const spy = vi.spyOn(api, 'get').mockResolvedValue(mockMemberList);

			const res = await societyApi.listMembers(socId, { status: 'ACTIVE', search: 'Vikram' });

			expect(res.members).toHaveLength(1);
			expect(res.members[0].first_name).toBe('Vikram');
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/members`, {
				params: {
					limit: 50,
					offset: 0,
					status: 'ACTIVE',
					search: 'Vikram'
				}
			});
		});

		it('calls POST /societies/:id/members to register member', async () => {
			const payload = {
				first_name: 'Ananya',
				phone: '+919830000000',
				email: 'ananya@example.com'
			};
			const mockResp = {
				member: {
					id: memberId,
					society_id: socId,
					first_name: 'Ananya',
					phone_masked: '+91 ******0000',
					status: 'ACTIVE' as const,
					created_at: '2026-10-16T00:00:00Z',
					updated_at: '2026-10-16T00:00:00Z'
				}
			};
			const spy = vi.spyOn(api, 'post').mockResolvedValue(mockResp);

			const res = await societyApi.registerMember(socId, payload);

			expect(res.member.first_name).toBe('Ananya');
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/members`, payload);
		});

		it('calls POST /societies/:id/members/:memberId/departure/preview', async () => {
			const mockPreview: DeparturePreviewResponse = {
				member_id: memberId,
				current_status: 'ACTIVE',
				version_hash: 'abc123hash',
				targeted_terminations: [],
				preserved_ownership_relationships: []
			};
			const spy = vi.spyOn(api, 'post').mockResolvedValue(mockPreview);

			const res = await societyApi.previewMemberDeparture(socId, memberId);

			expect(res.version_hash).toBe('abc123hash');
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/members/${memberId}/departure/preview`);
		});

		it('calls POST /societies/:id/members/:memberId/departure/confirm', async () => {
			const payload = {
				expected_version_hash: 'abc123hash',
				end_date: '2026-10-16',
				reason_code: 'TENANT_MOVED_OUT' as const,
				reason_details: 'Tenant moved out'
			};
			const mockConfirm = {
				message: 'Member departed successfully; occupancies ended and ownerships preserved'
			};
			const spy = vi.spyOn(api, 'post').mockResolvedValue(mockConfirm);

			const res = await societyApi.confirmMemberDeparture(socId, memberId, payload);

			expect(res.message).toContain('Member departed successfully');
			expect(spy).toHaveBeenCalledWith(
				`/societies/${socId}/members/${memberId}/departure/confirm`,
				payload
			);
		});

		it('calls GET and POST for unit relationships', async () => {
			const mockRelResp = {
				unit_id: unitId,
				total_ownership_share: 50,
				is_vacant: false,
				relationships: []
			};
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockRelResp);

			const getRes = await societyApi.listUnitRelationships(socId, unitId);
			expect(getRes.total_ownership_share).toBe(50);
			expect(getSpy).toHaveBeenCalledWith(`/societies/${socId}/units/${unitId}/relationships`);

			const createPayload = {
				member_id: memberId,
				relationship_type: 'PRIMARY_OWNER',
				is_occupant: true,
				ownership_share: 50
			};
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue({
				relationship: {
					id: 'rel-1',
					society_id: socId,
					unit_id: unitId,
					member_id: memberId,
					relationship_type: 'PRIMARY_OWNER' as const,
					is_occupant: true,
					ownership_share: 50,
					status: 'ACTIVE' as const,
					created_at: '2026-10-16T00:00:00Z',
					updated_at: '2026-10-16T00:00:00Z'
				}
			});

			const postRes = await societyApi.createUnitRelationship(socId, unitId, createPayload);
			expect(postRes.relationship.ownership_share).toBe(50);
			expect(postSpy).toHaveBeenCalledWith(
				`/societies/${socId}/units/${unitId}/relationships`,
				createPayload
			);
		});
	});

	describe('S07: Society Roles, Designations & Appointments API', () => {
		const socId = 'soc-777';
		const desigId = 'desig-111';
		const apptId = 'appt-222';
		const memberId = 'mem-333';

		it('calls listDesignations with include_inactive parameter', async () => {
			const mockResp = {
				designations: [
					{
						id: desigId,
						society_id: socId,
						code: 'CHAIRMAN',
						name: 'Chairman',
						is_system_default: true,
						is_active: true,
						allows_multiple_members: false,
						allows_multiple_designations: false,
						display_order: 1,
						created_at: '2026-10-18T00:00:00Z',
						updated_at: '2026-10-18T00:00:00Z'
					}
				]
			};
			const spy = vi.spyOn(api, 'get').mockResolvedValue(mockResp);

			const res = await societyApi.listDesignations(socId, true);
			expect(res.designations).toHaveLength(1);
			expect(res.designations[0].code).toBe('CHAIRMAN');
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/designations`, {
				params: { include_inactive: true }
			});
		});

		it('calls createDesignation with payload', async () => {
			const payload = {
				code: 'VICE_CHAIRMAN',
				name: 'Vice Chairman',
				allows_multiple_members: false,
				allows_multiple_designations: true,
				display_order: 4
			};
			const mockResp = {
				designation: {
					id: 'desig-vc',
					society_id: socId,
					...payload,
					is_system_default: false,
					is_active: true,
					created_at: '2026-10-18T00:00:00Z',
					updated_at: '2026-10-18T00:00:00Z'
				}
			};
			const spy = vi.spyOn(api, 'post').mockResolvedValue(mockResp);

			const res = await societyApi.createDesignation(socId, payload);
			expect(res.designation.code).toBe('VICE_CHAIRMAN');
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/designations`, payload);
		});

		it('calls updateDesignation and setDesignationStatus', async () => {
			const updatePayload = { name: 'Lead Treasurer', display_order: 3 };
			const mockUpdated = {
				designation: {
					id: desigId,
					society_id: socId,
					code: 'TREASURER',
					name: 'Lead Treasurer',
					is_system_default: true,
					is_active: true,
					allows_multiple_members: false,
					allows_multiple_designations: false,
					display_order: 3,
					created_at: '2026-10-18T00:00:00Z',
					updated_at: '2026-10-18T00:00:00Z'
				}
			};
			const patchSpy = vi.spyOn(api, 'patch').mockResolvedValue(mockUpdated);

			const updateRes = await societyApi.updateDesignation(socId, desigId, updatePayload);
			expect(updateRes.designation.name).toBe('Lead Treasurer');
			expect(patchSpy).toHaveBeenCalledWith(
				`/societies/${socId}/designations/${desigId}`,
				updatePayload
			);

			const toggleRes = await societyApi.setDesignationStatus(socId, desigId, false);
			expect(toggleRes.designation).toBeDefined();
			expect(patchSpy).toHaveBeenCalledWith(`/societies/${socId}/designations/${desigId}/status`, {
				is_active: false
			});
		});

		it('calls getCommitteeOverview', async () => {
			const mockOverview = {
				office_bearers: [
					{
						designation_id: desigId,
						designation_name: 'Chairman',
						appointment_id: apptId,
						member_id: memberId,
						member_name: 'Rajesh Sharma',
						member_phone_masked: '919876****10',
						start_date: '2026-04-01',
						status: 'ACTIVE'
					}
				],
				total_active_appointments: 1,
				active_designations_count: 5
			};
			const spy = vi.spyOn(api, 'get').mockResolvedValue(mockOverview);

			const res = await societyApi.getCommitteeOverview(socId);
			expect(res.total_active_appointments).toBe(1);
			expect(res.office_bearers[0].member_name).toBe('Rajesh Sharma');
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/committee/overview`);
		});

		it('calls listAppointments with status filter', async () => {
			const mockResp = {
				appointments: [
					{
						id: apptId,
						society_id: socId,
						member_id: memberId,
						designation_id: desigId,
						start_date: '2026-04-01',
						status: 'ACTIVE' as const,
						appointed_by: 'admin-1',
						created_at: '2026-04-01T00:00:00Z',
						updated_at: '2026-04-01T00:00:00Z',
						member_name: 'Rajesh Sharma',
						designation_name: 'Chairman'
					}
				]
			};
			const spy = vi.spyOn(api, 'get').mockResolvedValue(mockResp);

			const res = await societyApi.listAppointments(socId, { status: 'ACTIVE' });
			expect(res.appointments).toHaveLength(1);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/appointments`, {
				params: { status: 'ACTIVE', member_id: undefined, designation_id: undefined }
			});
		});

		it('calls createAppointment and endAppointment', async () => {
			const createPayload = {
				member_id: memberId,
				designation_id: desigId,
				start_date: '2026-10-18',
				notes: 'Appointed at AGM'
			};
			const mockAppt = {
				appointment: {
					id: apptId,
					society_id: socId,
					member_id: memberId,
					designation_id: desigId,
					start_date: '2026-10-18',
					status: 'ACTIVE' as const,
					appointed_by: 'admin-1',
					notes: 'Appointed at AGM',
					created_at: '2026-10-18T00:00:00Z',
					updated_at: '2026-10-18T00:00:00Z'
				}
			};
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(mockAppt);

			const createRes = await societyApi.createAppointment(socId, createPayload);
			expect(createRes.appointment.id).toBe(apptId);
			expect(postSpy).toHaveBeenCalledWith(`/societies/${socId}/appointments`, createPayload);

			const endPayload = {
				end_date: '2026-10-31',
				status: 'ENDED' as const,
				notes: 'Term completed'
			};
			const mockEnded = {
				appointment: {
					...mockAppt.appointment,
					end_date: '2026-10-31',
					status: 'ENDED' as const,
					ended_by: 'admin-1'
				}
			};
			postSpy.mockResolvedValueOnce(mockEnded);

			const endRes = await societyApi.endAppointment(socId, apptId, endPayload);
			expect(endRes.appointment.status).toBe('ENDED');
			expect(postSpy).toHaveBeenCalledWith(
				`/societies/${socId}/appointments/${apptId}/end`,
				endPayload
			);
		});
	});

	describe('Society Communication (S10 Announcements)', () => {
		const socId = 'soc-ann-1';
		const annId = 'ann-123';

		it('getAdminAnnouncements sends correct query params', async () => {
			const mockResp = {
				announcements: [],
				total: 0,
				limit: 20,
				offset: 0
			};
			const spy = vi.spyOn(api, 'get').mockResolvedValue(mockResp);

			const res = await societyApi.getAdminAnnouncements(socId, {
				status: 'PUBLISHED',
				category: 'GENERAL'
			});
			expect(res).toEqual(mockResp);
			expect(spy).toHaveBeenCalledWith(
				`/societies/${socId}/announcements/admin?status=PUBLISHED&category=GENERAL`
			);
		});

		it('createAnnouncement posts payload to admin endpoint', async () => {
			const payload = {
				title: 'Water Tank Cleaning Notice',
				body: 'Water supply will be suspended tomorrow between 10am and 2pm.',
				category: 'MAINTENANCE' as const,
				priority: 'URGENT' as const,
				audience_type: 'SOCIETY' as const
			};
			const mockResp = {
				announcement: { id: annId, ...payload, status: 'DRAFT' as const }
			};
			const spy = vi.spyOn(api, 'post').mockResolvedValue(mockResp);

			const res = await societyApi.createAnnouncement(socId, payload);
			expect(res).toEqual(mockResp);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/announcements`, payload);
		});

		it('publishAnnouncement posts to publish endpoint', async () => {
			const mockResp = {
				announcement: { id: annId, status: 'PUBLISHED' as const }
			};
			const spy = vi.spyOn(api, 'post').mockResolvedValue(mockResp);

			const res = await societyApi.publishAnnouncement(socId, annId);
			expect(res.announcement.status).toBe('PUBLISHED');
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/announcements/${annId}/publish`, {});
		});

		it('archiveAnnouncement posts to archive endpoint', async () => {
			const mockResp = {
				announcement: { id: annId, status: 'ARCHIVED' as const }
			};
			const spy = vi.spyOn(api, 'post').mockResolvedValue(mockResp);

			const res = await societyApi.archiveAnnouncement(socId, annId);
			expect(res.announcement.status).toBe('ARCHIVED');
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/announcements/${annId}/archive`, {});
		});

		it('getAnnouncementFeed queries resident feed', async () => {
			const mockResp = {
				announcements: [],
				total: 0,
				limit: 20,
				offset: 0,
				unread_count: 3
			};
			const spy = vi.spyOn(api, 'get').mockResolvedValue(mockResp);

			const res = await societyApi.getAnnouncementFeed(socId, { unread_only: true });
			expect(res.unread_count).toBe(3);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/announcements/feed?unread_only=true`);
		});

		it('markAnnouncementRead posts to mark-read endpoint', async () => {
			const mockResp = {
				announcement_id: annId,
				read: true
			};
			const spy = vi.spyOn(api, 'post').mockResolvedValue(mockResp);

			const res = await societyApi.markAnnouncementRead(socId, annId);
			expect(res.announcement_id).toBe(annId);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/announcements/${annId}/read`, {});
		});

		it('getUnreadAnnouncementCount queries unread count', async () => {
			const mockResp = { unread_count: 5 };
			const spy = vi.spyOn(api, 'get').mockResolvedValue(mockResp);

			const res = await societyApi.getUnreadAnnouncementCount(socId);
			expect(res.unread_count).toBe(5);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/announcements/unread-count`);
		});
	});

	describe('S11 Maintenance Foundation API', () => {
		const socId = 'soc-123';
		const catId = 'cat-111';
		const configId = 'cfg-222';
		const periodId = 'per-333';
		const billId = 'bill-444';

		it('listChargeCategories calls GET categories with status filter', async () => {
			const mockResp = { categories: [{ id: catId, name: 'Sinking Fund' }] };
			const spy = vi.spyOn(api, 'get').mockResolvedValue(mockResp);

			const res = await societyApi.listChargeCategories(socId, 'ACTIVE');
			expect(res.categories).toHaveLength(1);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/maintenance/categories?status=ACTIVE`);
		});

		it('createChargeCategory posts payload to categories endpoint', async () => {
			const payload = { name: 'Maintenance', description: 'Monthly fee' };
			const mockResp = { category: { id: catId, ...payload, status: 'ACTIVE' } };
			const spy = vi.spyOn(api, 'post').mockResolvedValue(mockResp);

			const res = await societyApi.createChargeCategory(socId, payload);
			expect(res.category.id).toBe(catId);
			expect(spy).toHaveBeenCalledWith(`/societies/${socId}/maintenance/categories`, payload);
		});

		it('updateChargeCategory sends PUT to category endpoint', async () => {
			const payload = { name: 'Updated Fee', status: 'INACTIVE' as const };
			const mockResp = { category: { id: catId, name: 'Updated Fee', status: 'INACTIVE' } };
			const spy = vi.spyOn(api, 'put').mockResolvedValue(mockResp);

			const res = await societyApi.updateChargeCategory(socId, catId, payload);
			expect(res.category.status).toBe('INACTIVE');
			expect(spy).toHaveBeenCalledWith(
				`/societies/${socId}/maintenance/categories/${catId}`,
				payload
			);
		});

		it('listChargeConfigs and createChargeConfig work correctly', async () => {
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue({ configs: [{ id: configId }] });
			await societyApi.listChargeConfigs(socId);
			expect(getSpy).toHaveBeenCalledWith(`/societies/${socId}/maintenance/configs`);

			const createPayload = {
				category_id: catId,
				name: 'Monthly Flat Rate',
				charge_type: 'FLAT_RECURRING' as const,
				amount: 3500,
				billing_frequency: 'MONTHLY' as const,
				effective_from: '2026-10-01'
			};
			const postSpy = vi
				.spyOn(api, 'post')
				.mockResolvedValue({ config: { id: configId, ...createPayload } });
			const res = await societyApi.createChargeConfig(socId, createPayload);
			expect(res.config.amount).toBe(3500);
			expect(postSpy).toHaveBeenCalledWith(
				`/societies/${socId}/maintenance/configs`,
				createPayload
			);
		});

		it('deactivateChargeConfig posts to deactivate endpoint', async () => {
			const spy = vi.spyOn(api, 'post').mockResolvedValue(undefined);
			await societyApi.deactivateChargeConfig(socId, configId);
			expect(spy).toHaveBeenCalledWith(
				`/societies/${socId}/maintenance/configs/${configId}/deactivate`,
				{}
			);
		});

		it('billing periods management and bill generation workflows', async () => {
			const getPeriodsSpy = vi.spyOn(api, 'get').mockResolvedValue({ periods: [{ id: periodId }] });
			await societyApi.listBillingPeriods(socId);
			expect(getPeriodsSpy).toHaveBeenCalledWith(`/societies/${socId}/maintenance/periods`);

			const periodPayload = {
				period_name: 'October 2026',
				period_start: '2026-10-01',
				period_end: '2026-10-31',
				due_date: '2026-11-10'
			};
			const createPeriodSpy = vi
				.spyOn(api, 'post')
				.mockResolvedValue({ period: { id: periodId, ...periodPayload } });
			await societyApi.createBillingPeriod(socId, periodPayload);
			expect(createPeriodSpy).toHaveBeenCalledWith(
				`/societies/${socId}/maintenance/periods`,
				periodPayload
			);

			const previewSpy = vi
				.spyOn(api, 'get')
				.mockResolvedValue({ preview: { period_id: periodId, eligible_unit_count: 5 } });
			const prevRes = await societyApi.previewBillGeneration(socId, periodId);
			expect(prevRes.preview.eligible_unit_count).toBe(5);
			expect(previewSpy).toHaveBeenCalledWith(
				`/societies/${socId}/maintenance/periods/${periodId}/preview`
			);

			const genSpy = vi
				.spyOn(api, 'post')
				.mockResolvedValue({ result: { bills_created: 5, units_skipped: 0 } });
			const genRes = await societyApi.generateBills(socId, periodId);
			expect(genRes.result.bills_created).toBe(5);
			expect(genSpy).toHaveBeenCalledWith(
				`/societies/${socId}/maintenance/periods/${periodId}/generate`,
				{}
			);
		});

		it('listBills, getBillDetails, and voidBill', async () => {
			const getBillsSpy = vi
				.spyOn(api, 'get')
				.mockResolvedValue({ items: [{ id: billId }], total: 1, limit: 10, offset: 0 });
			const listRes = await societyApi.listBills(socId, { status: 'ISSUED', limit: 10 });
			expect(listRes.total).toBe(1);
			expect(getBillsSpy).toHaveBeenCalledWith(
				`/societies/${socId}/maintenance/bills?status=ISSUED&limit=10`
			);

			const getDetailSpy = vi
				.spyOn(api, 'get')
				.mockResolvedValue({ bill: { id: billId, items: [] } });
			const detailRes = await societyApi.getBillDetails(socId, billId);
			expect(detailRes.bill.id).toBe(billId);
			expect(getDetailSpy).toHaveBeenCalledWith(`/societies/${socId}/maintenance/bills/${billId}`);

			const voidSpy = vi.spyOn(api, 'post').mockResolvedValue(undefined);
			await societyApi.voidBill(socId, billId, 'Duplicate billing error');
			expect(voidSpy).toHaveBeenCalledWith(`/societies/${socId}/maintenance/bills/${billId}/void`, {
				reason: 'Duplicate billing error'
			});
		});

		it('resident maintenance bills endpoints', async () => {
			const getResidentBillsSpy = vi
				.spyOn(api, 'get')
				.mockResolvedValue({ bills: [{ id: billId }], total: 1 });
			const myBillsRes = await societyApi.listResidentBills(socId);
			expect(myBillsRes.bills).toHaveLength(1);
			expect(getResidentBillsSpy).toHaveBeenCalledWith(`/societies/${socId}/maintenance/my-bills`);

			const getResidentDetailSpy = vi
				.spyOn(api, 'get')
				.mockResolvedValue({ bill: { id: billId, items: [] } });
			const myDetailRes = await societyApi.getResidentBillDetails(socId, billId);
			expect(myDetailRes.bill.id).toBe(billId);
			expect(getResidentDetailSpy).toHaveBeenCalledWith(
				`/societies/${socId}/maintenance/my-bills/${billId}`
			);
		});
	});
});
