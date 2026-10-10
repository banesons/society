import { describe, it, expect, vi, beforeEach } from 'vitest';
import { societyApi, type SocietyItem, type CreateSocietyPayload } from './society';
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
});
