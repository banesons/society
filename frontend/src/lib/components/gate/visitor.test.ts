import { describe, it, expect, vi, beforeEach } from 'vitest';
import { societyApi, type Visit, type VisitorDashboardMetrics } from '$lib/api/society';
import { api, ApiClientError } from '$lib/api/client';
import { normalizePhone } from '$lib/phone';

describe('S09 Gate Visitor Operations Frontend Logic & API', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	const mockSocietyId = '0a2c153c-32ca-458e-a090-81a4687b7a76';
	const mockVisit: Visit = {
		id: 'e6f47701-4475-4702-8a90-3b60098f41e0',
		society_id: mockSocietyId,
		unit_id: '038bf8c8-b219-4aa8-9f3b-fa2a549a1f11',
		unit_identifier: '101',
		building_name: 'Tower A',
		visitor_name: 'Rajesh Sharma',
		visitor_phone: '+919820011223',
		visitor_phone_masked: '+919820xxxx23',
		purpose: 'GUEST',
		visit_type: 'EXPECTED',
		status: 'APPROVED',
		expected_arrival: '2026-10-12T14:00:00Z',
		valid_until: '2026-10-12T22:00:00Z',
		created_by: 'user-1',
		created_at: '2026-10-12T00:15:00Z',
		updated_at: '2026-10-12T00:15:00Z'
	};

	describe('Visitor Form Validation Logic', () => {
		function validateVisitorInput(data: {
			unitId: string;
			visitorName: string;
			visitorPhone: string;
			purpose: string;
		}) {
			const errors: Record<string, string> = {};
			if (!data.unitId) {
				errors.unitId = 'Destination unit is required';
			}
			if (!data.visitorName.trim()) {
				errors.visitorName = 'Visitor name is required';
			}
			if (!data.visitorPhone.trim()) {
				errors.visitorPhone = 'Visitor phone is required';
			} else {
				try {
					normalizePhone(data.visitorPhone);
				} catch {
					errors.visitorPhone = 'Invalid phone number format';
				}
			}
			if (!data.purpose) {
				errors.purpose = 'Purpose of visit is required';
			}
			return {
				isValid: Object.keys(errors).length === 0,
				errors
			};
		}

		it('validates correct visitor registration form', () => {
			const res = validateVisitorInput({
				unitId: 'unit-123',
				visitorName: 'Amit Patel',
				visitorPhone: '+91 98200 12345',
				purpose: 'DELIVERY'
			});
			expect(res.isValid).toBe(true);
		});

		it('flags missing unit, blank name, and invalid phone number', () => {
			const res = validateVisitorInput({
				unitId: '',
				visitorName: '   ',
				visitorPhone: '1234',
				purpose: ''
			});
			expect(res.isValid).toBe(false);
			expect(res.errors.unitId).toBeDefined();
			expect(res.errors.visitorName).toBeDefined();
			expect(res.errors.visitorPhone).toBeDefined();
			expect(res.errors.purpose).toBeDefined();
		});
	});

	describe('Pass Code and Token Verification Format', () => {
		function isPassExpired(validUntil: string): boolean {
			return new Date(validUntil).getTime() < Date.now();
		}

		it('identifies expired vs active validity windows', () => {
			const futureDate = new Date(Date.now() + 3600000).toISOString();
			const pastDate = new Date(Date.now() - 3600000).toISOString();

			expect(isPassExpired(futureDate)).toBe(false);
			expect(isPassExpired(pastDate)).toBe(true);
		});
	});

	describe('societyApi Visitor Operations Methods', () => {
		it('listVisits calls GET /societies/{id}/visitors with params', async () => {
			const mockRes = {
				visits: [mockVisit],
				total: 1,
				limit: 20,
				offset: 0
			};
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockRes);

			const result = await societyApi.listVisits(mockSocietyId, {
				status: 'CHECKED_IN',
				limit: 20,
				offset: 0
			});

			expect(result).toEqual(mockRes);
			expect(getSpy).toHaveBeenCalledWith(`/societies/${mockSocietyId}/visitors`, {
				params: { status: 'CHECKED_IN', limit: 20, offset: 0 }
			});
		});

		it('getVisitorDashboard calls GET /societies/{id}/visitors/dashboard', async () => {
			const mockMetrics: VisitorDashboardMetrics = {
				expected_today: 5,
				awaiting_approval: 2,
				currently_inside: 3,
				completed_today: 10
			};
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockMetrics);

			const result = await societyApi.getVisitorDashboard(mockSocietyId);
			expect(result).toEqual(mockMetrics);
			expect(getSpy).toHaveBeenCalledWith(`/societies/${mockSocietyId}/visitors/dashboard`);
		});

		it('checkInVisitor calls POST /societies/{id}/visitors/{visitId}/check-in', async () => {
			const checkInRes = {
				visit: {
					...mockVisit,
					status: 'CHECKED_IN' as const,
					check_in_at: new Date().toISOString()
				},
				event: {
					id: 'event-1',
					society_id: mockSocietyId,
					visit_id: mockVisit.id,
					gate_id: 'gate-1',
					gate_name: 'Main Gate',
					gatekeeper_id: 'guard-1',
					event_type: 'ENTRY' as const,
					direction: 'INGRESS' as const,
					created_at: new Date().toISOString()
				}
			};
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(checkInRes);

			const payload = { gate_id: 'gate-1', vehicle_number: 'MH02CB1234' };
			const result = await societyApi.checkInVisitor(mockSocietyId, mockVisit.id, payload);

			expect(result.visit.status).toBe('CHECKED_IN');
			expect(postSpy).toHaveBeenCalledWith(
				`/societies/${mockSocietyId}/visitors/${mockVisit.id}/check-in`,
				payload
			);
		});

		it('checkOutVisitor calls POST /societies/{id}/visitors/{visitId}/check-out', async () => {
			const checkOutRes = {
				visit: {
					...mockVisit,
					status: 'CHECKED_OUT' as const,
					check_out_at: new Date().toISOString()
				},
				event: {
					id: 'event-2',
					society_id: mockSocietyId,
					visit_id: mockVisit.id,
					gate_id: 'gate-1',
					gate_name: 'Main Gate',
					gatekeeper_id: 'guard-1',
					event_type: 'EXIT' as const,
					direction: 'EGRESS' as const,
					created_at: new Date().toISOString()
				}
			};
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(checkOutRes);

			const payload = { gate_id: 'gate-1', notes: 'Normal exit' };
			const result = await societyApi.checkOutVisitor(mockSocietyId, mockVisit.id, payload);

			expect(result.visit.status).toBe('CHECKED_OUT');
			expect(postSpy).toHaveBeenCalledWith(
				`/societies/${mockSocietyId}/visitors/${mockVisit.id}/check-out`,
				payload
			);
		});

		it('handles duplicate check-in conflict error (409 Conflict)', async () => {
			vi.spyOn(api, 'post').mockRejectedValue(
				new ApiClientError(
					'Visitor is already checked in',
					'VISIT_ALREADY_CHECKED_IN',
					409,
					'req-409'
				)
			);

			await expect(
				societyApi.checkInVisitor(mockSocietyId, mockVisit.id, { gate_id: 'gate-1' })
			).rejects.toThrow('Visitor is already checked in');
		});

		it('handles pass verification call and propagates error on expired pass', async () => {
			vi.spyOn(api, 'post').mockRejectedValue(
				new ApiClientError('Visitor pass has expired', 'PASS_EXPIRED', 409, 'req-exp')
			);

			await expect(societyApi.verifyVisitorPass(mockSocietyId, 'GP-EXPIRED')).rejects.toThrow(
				'Visitor pass has expired'
			);
		});

		it('registerWalkInVisit sends payload without requiring client-side approval flag', async () => {
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue({ visit: mockVisit });
			const payload = {
				unit_id: 'unit-123',
				visitor_name: 'Walk-in Guest',
				visitor_phone: '+919820011223',
				purpose: 'GUEST'
			};
			const res = await societyApi.registerWalkInVisit(mockSocietyId, payload);
			expect(res.visit).toBeDefined();
			expect(postSpy).toHaveBeenCalledWith(`/societies/${mockSocietyId}/visitors/walk-in`, payload);
		});

		it('getMyUnits calls GET /societies/{id}/units/my', async () => {
			const mockUnits = [
				{
					id: 'unit-1',
					unit_identifier: '101',
					floor_number: 1,
					building_name: 'Tower A',
					relationship_type: 'PRIMARY_OWNER',
					requires_visitor_approval: false
				}
			];
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue({ units: mockUnits });
			const res = await societyApi.getMyUnits(mockSocietyId);
			expect(res.units).toEqual(mockUnits);
			expect(getSpy).toHaveBeenCalledWith(`/societies/${mockSocietyId}/units/my`);
		});

		it('updateUnitVisitorPolicy calls PATCH /societies/{id}/units/{unitId}/visitor-policy', async () => {
			const mockUnit = {
				id: 'unit-1',
				unit_identifier: '101',
				floor_number: 1,
				building_name: 'Tower A',
				relationship_type: 'PRIMARY_OWNER',
				requires_visitor_approval: true
			};
			const patchSpy = vi.spyOn(api, 'patch').mockResolvedValue({
				unit: mockUnit
			});
			const res = await societyApi.updateUnitVisitorPolicy(mockSocietyId, 'unit-1', true);
			expect(res.unit.requires_visitor_approval).toBe(true);
			expect(patchSpy).toHaveBeenCalledWith(
				`/societies/${mockSocietyId}/units/unit-1/visitor-policy`,
				{ requires_visitor_approval: true }
			);
		});

		it('getPendingVisits calls GET /societies/{id}/visitors/pending', async () => {
			const mockPending = [{ ...mockVisit, status: 'PENDING_APPROVAL' as const }];
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue({ visits: mockPending });
			const res = await societyApi.getPendingVisits(mockSocietyId);
			expect(res.visits).toEqual(mockPending);
			expect(getSpy).toHaveBeenCalledWith(`/societies/${mockSocietyId}/visitors/pending`);
		});

		it('handles DUTY_SESSION_REQUIRED error on gate check-in', async () => {
			vi.spyOn(api, 'post').mockRejectedValue(
				new ApiClientError(
					'Active duty session required for this gate',
					'DUTY_SESSION_REQUIRED',
					403,
					'req-duty'
				)
			);

			await expect(
				societyApi.checkInVisitor(mockSocietyId, mockVisit.id, { gate_id: 'gate-1' })
			).rejects.toThrow('Active duty session required for this gate');
		});

		it('handles OCCUPANT_UNAUTHORIZED error on cross-unit approval attempt', async () => {
			vi.spyOn(api, 'post').mockRejectedValue(
				new ApiClientError(
					'Not authorized to approve visits for this unit',
					'OCCUPANT_UNAUTHORIZED',
					403,
					'req-cross'
				)
			);

			await expect(
				societyApi.approveVisit(mockSocietyId, mockVisit.id, { notes: 'Unauthorized' })
			).rejects.toThrow('Not authorized to approve visits for this unit');
		});
	});
});
