import { describe, it, expect } from 'vitest';
import type { CommitteeAppointment, OfficeBearerInfo } from '$lib/api/society';

describe('S07 Committee & Designations Frontend Logic & Validation', () => {
	describe('Appointment Form Validation', () => {
		function validateAppointmentForm(data: {
			memberId: string;
			designationId: string;
			startDate: string;
			endDate?: string;
		}) {
			const errors: Record<string, string> = {};
			if (!data.memberId) {
				errors.memberId = 'Please select an active society member.';
			}
			if (!data.designationId) {
				errors.designationId = 'Please select a designation.';
			}
			if (!data.startDate) {
				errors.startDate = 'Effective start date is required.';
			}
			if (data.endDate && data.startDate && data.endDate < data.startDate) {
				errors.endDate = 'End date cannot be prior to the start date.';
			}
			return {
				isValid: Object.keys(errors).length === 0,
				errors
			};
		}

		it('rejects empty fields', () => {
			const res = validateAppointmentForm({
				memberId: '',
				designationId: '',
				startDate: ''
			});
			expect(res.isValid).toBe(false);
			expect(res.errors.memberId).toBeDefined();
			expect(res.errors.designationId).toBeDefined();
			expect(res.errors.startDate).toBeDefined();
		});

		it('rejects end date prior to start date', () => {
			const res = validateAppointmentForm({
				memberId: 'mem-1',
				designationId: 'desig-1',
				startDate: '2026-10-15',
				endDate: '2026-10-10'
			});
			expect(res.isValid).toBe(false);
			expect(res.errors.endDate).toContain('prior to the start date');
		});

		it('accepts valid start and end dates', () => {
			const res = validateAppointmentForm({
				memberId: 'mem-1',
				designationId: 'desig-1',
				startDate: '2026-10-15',
				endDate: '2026-10-31'
			});
			expect(res.isValid).toBe(true);
			expect(res.errors).toEqual({});
		});

		it('accepts open-ended appointment without end date', () => {
			const res = validateAppointmentForm({
				memberId: 'mem-1',
				designationId: 'desig-1',
				startDate: '2026-10-15'
			});
			expect(res.isValid).toBe(true);
			expect(res.errors).toEqual({});
		});
	});

	describe('End Appointment Form Validation', () => {
		function validateEndAppointmentForm(data: {
			appointmentStartDate: string;
			endDate: string;
			endStatus: 'ENDED' | 'REVOKED';
		}) {
			const errors: Record<string, string> = {};
			if (!data.endDate) {
				errors.endDate = 'Effective end date is required.';
			} else if (data.appointmentStartDate && data.endDate < data.appointmentStartDate) {
				errors.endDate = `End date cannot precede the appointment start date (${data.appointmentStartDate}).`;
			}
			return {
				isValid: Object.keys(errors).length === 0,
				errors
			};
		}

		it('rejects empty end date', () => {
			const res = validateEndAppointmentForm({
				appointmentStartDate: '2026-04-01',
				endDate: '',
				endStatus: 'ENDED'
			});
			expect(res.isValid).toBe(false);
			expect(res.errors.endDate).toContain('is required');
		});

		it('rejects end date before appointment start date', () => {
			const res = validateEndAppointmentForm({
				appointmentStartDate: '2026-04-01',
				endDate: '2026-03-15',
				endStatus: 'ENDED'
			});
			expect(res.isValid).toBe(false);
			expect(res.errors.endDate).toContain('cannot precede the appointment start date');
		});

		it('accepts valid conclusion date equal to start date', () => {
			const res = validateEndAppointmentForm({
				appointmentStartDate: '2026-04-01',
				endDate: '2026-04-01',
				endStatus: 'REVOKED'
			});
			expect(res.isValid).toBe(true);
		});
	});

	describe('Designation Code Normalization & Creation Form', () => {
		function normalizeAndValidateDesignation(code: string, name: string) {
			const cleanCode = code.trim().toUpperCase().replace(/\s+/g, '_');
			const cleanName = name.trim();
			const errors: Record<string, string> = {};

			if (!cleanCode) {
				errors.code = 'Designation code is required.';
			} else if (!/^[A-Z0-9_]{2,50}$/.test(cleanCode)) {
				errors.code = 'Code must be 2-50 uppercase alphanumeric characters or underscores.';
			}

			if (!cleanName) {
				errors.name = 'Display name is required.';
			} else if (cleanName.length < 2 || cleanName.length > 100) {
				errors.name = 'Display name must be between 2 and 100 characters.';
			}

			return {
				normalizedCode: cleanCode,
				normalizedName: cleanName,
				isValid: Object.keys(errors).length === 0,
				errors
			};
		}

		it('normalizes spaces and lowercases to canonical uppercase codes', () => {
			const res = normalizeAndValidateDesignation('vice chairman', 'Vice Chairman');
			expect(res.normalizedCode).toBe('VICE_CHAIRMAN');
			expect(res.isValid).toBe(true);
		});

		it('rejects invalid characters in designation codes', () => {
			const res = normalizeAndValidateDesignation('VICE-CHAIRMAN!', 'Vice Chairman');
			expect(res.isValid).toBe(false);
			expect(res.errors.code).toBeDefined();
		});

		it('rejects short display name', () => {
			const res = normalizeAndValidateDesignation('AUD', 'A');
			expect(res.isValid).toBe(false);
			expect(res.errors.name).toBeDefined();
		});
	});

	describe('API Error Mapping for User Presentation', () => {
		function mapAppointmentError(code: string, fallback: string): string {
			switch (code) {
				case 'DESIGNATION_SLOT_OCCUPIED':
				case 'DESIGNATION_ALREADY_APPOINTED':
					return 'This designation is a single-holder position and already has an active occupant. Please end or revoke the current appointment first.';
				case 'OVERLAPPING_APPOINTMENT':
				case 'MEMBER_ALREADY_APPOINTED':
					return 'This member already holds an active appointment for this designation.';
				case 'CONFLICTING_DATE_RANGE':
					return 'Appointment dates conflict with an existing appointment term for this designation.';
				case 'MULTIPLE_DESIGNATIONS_DISALLOWED':
					return 'This designation cannot be held simultaneously with other active committee roles.';
				case 'INVALID_MEMBER_STATUS':
				case 'MEMBER_NOT_ACTIVE':
					return 'Appointments can only be conferred upon ACTIVE society members.';
				case 'DESIGNATION_INACTIVE':
					return 'The chosen designation is inactive and cannot accept new appointments.';
				default:
					return fallback;
			}
		}

		it('maps singleton occupied error to explanatory text', () => {
			const msg = mapAppointmentError('DESIGNATION_SLOT_OCCUPIED', 'Default error');
			expect(msg).toContain('single-holder position and already has an active occupant');
		});

		it('maps multiple designations disallowed error', () => {
			const msg = mapAppointmentError('MULTIPLE_DESIGNATIONS_DISALLOWED', 'Default error');
			expect(msg).toContain('cannot be held simultaneously');
		});

		it('maps conflicting date range error', () => {
			const msg = mapAppointmentError('CONFLICTING_DATE_RANGE', 'Default error');
			expect(msg).toContain('Appointment dates conflict with an existing appointment term');
		});
	});

	describe('Committee Overview & Appointment UI State Mapping', () => {
		// Mock helper simulating committee roster state management
		function computeRosterState(params: {
			overviewResult?: {
				status: 'fulfilled' | 'rejected';
				value?: {
					office_bearers?: Array<{
						designation_id: string;
						designation_code?: string;
						designation_name: string;
						member_id?: string;
						member_name?: string;
						member_phone_masked?: string;
						start_date?: string;
						expected_end_date?: string;
						status?: string;
					}>;
				};
				reason?: { statusCode?: number; message?: string };
			};
			appointmentsResult: {
				status: 'fulfilled' | 'rejected';
				value?: {
					appointments?: Array<{
						id: string;
						designation_id: string;
						designation_code: string;
						designation_name: string;
						member_id: string;
						member_name: string;
						member_phone: string;
						start_date: string;
						expected_end_date?: string | null;
						end_date?: string | null;
						status: 'ACTIVE' | 'ENDED' | 'REVOKED';
					}>;
				};
				reason?: { statusCode?: number; message?: string };
			};
			filter: 'ALL' | 'ACTIVE' | 'ENDED' | 'REVOKED';
		}) {
			let error: string | null = null;
			let appointments: CommitteeAppointment[] = [];

			if (params.appointmentsResult.status === 'fulfilled') {
				const raw = (params.appointmentsResult.value?.appointments || []) as CommitteeAppointment[];
				appointments =
					params.filter === 'ALL' ? raw : raw.filter((a) => a.status === params.filter);
			} else {
				error = params.appointmentsResult.reason?.message || 'Failed to load appointments';
			}

			if (params.overviewResult?.status === 'rejected') {
				if (params.appointmentsResult.status === 'rejected') {
					error = error || 'Failed to load committee data';
				} else if (params.overviewResult.reason?.statusCode !== 404) {
					error = params.overviewResult.reason?.message || 'Failed to load overview';
				}
			}

			// Office bearers derivation
			let officeBearers: OfficeBearerInfo[];
			if (
				params.overviewResult?.status === 'fulfilled' &&
				params.overviewResult.value?.office_bearers &&
				params.overviewResult.value.office_bearers.length > 0
			) {
				officeBearers = params.overviewResult.value.office_bearers;
			} else {
				// Derive from appointments
				const canonical = [
					{ code: 'CHAIRMAN', name: 'Chairman' },
					{ code: 'SECRETARY', name: 'Secretary' },
					{ code: 'TREASURER', name: 'Treasurer' }
				];
				officeBearers = canonical.map((c) => {
					const apt = appointments.find(
						(a) => a.designation_code === c.code && a.status === 'ACTIVE'
					);
					return {
						designation_id: apt?.designation_id || c.code.toLowerCase(),
						designation_code: c.code,
						designation_name: c.name,
						member_id: apt?.member_id,
						member_name: apt?.member_name,
						member_phone_masked: apt?.member_phone,
						start_date: apt?.start_date,
						expected_end_date: apt?.expected_end_date || undefined,
						status: apt?.status
					};
				});
			}

			return {
				error,
				appointments,
				officeBearers,
				displayedCount: appointments.length,
				isEmptyState: appointments.length === 0 && error === null
			};
		}

		it('Overview response with all three office-bearers', () => {
			const mockOfficeBearers = [
				{
					designation_id: 'd1',
					designation_code: 'CHAIRMAN',
					designation_name: 'Chairman',
					member_id: 'm1',
					member_name: 'Vijay Godbole',
					member_phone_masked: '+91 9822000101',
					start_date: '2026-01-01',
					expected_end_date: '2027-12-31',
					status: 'ACTIVE'
				},
				{
					designation_id: 'd2',
					designation_code: 'SECRETARY',
					designation_name: 'Secretary',
					member_id: 'm2',
					member_name: 'Milind Date',
					member_phone_masked: '+91 9822000103',
					start_date: '2026-01-01',
					expected_end_date: '2027-12-31',
					status: 'ACTIVE'
				},
				{
					designation_id: 'd3',
					designation_code: 'TREASURER',
					designation_name: 'Treasurer',
					member_id: 'm3',
					member_name: 'Sudhir Apte',
					member_phone_masked: '+91 9822000104',
					start_date: '2026-01-01',
					expected_end_date: '2027-12-31',
					status: 'ACTIVE'
				}
			];

			const state = computeRosterState({
				overviewResult: {
					status: 'fulfilled',
					value: { office_bearers: mockOfficeBearers }
				},
				appointmentsResult: {
					status: 'fulfilled',
					value: { appointments: [] }
				},
				filter: 'ACTIVE'
			});

			expect(state.officeBearers).toHaveLength(3);
			expect(state.officeBearers[0].member_name).toBe('Vijay Godbole');
			expect(state.officeBearers[0].status).toBe('ACTIVE');
			expect(state.officeBearers[1].member_name).toBe('Milind Date');
			expect(state.officeBearers[2].member_name).toBe('Sudhir Apte');
			expect(state.error).toBeNull();
		});

		it('Multiple committee members displayed in roster', () => {
			const appointments = [
				{
					id: 'a1',
					designation_id: 'd4',
					designation_code: 'COMMITTEE_MEMBER',
					designation_name: 'Committee Member',
					member_id: 'm4',
					member_name: 'Karan Saxena',
					member_phone: '+91 9822000115',
					start_date: '2026-10-11',
					status: 'ACTIVE' as const
				},
				{
					id: 'a2',
					designation_id: 'd4',
					designation_code: 'COMMITTEE_MEMBER',
					designation_name: 'Committee Member',
					member_id: 'm5',
					member_name: 'Rekha Chitnis',
					member_phone: '+91 9822000106',
					start_date: '2026-01-01',
					status: 'ACTIVE' as const
				}
			];

			const state = computeRosterState({
				overviewResult: {
					status: 'fulfilled',
					value: { office_bearers: [] }
				},
				appointmentsResult: {
					status: 'fulfilled',
					value: { appointments }
				},
				filter: 'ACTIVE'
			});

			expect(state.appointments).toHaveLength(2);
			expect(state.appointments[0].member_name).toBe('Karan Saxena');
			expect(state.appointments[1].member_name).toBe('Rekha Chitnis');
			expect(state.displayedCount).toBe(2);
		});

		it('Empty overview response handled cleanly', () => {
			const state = computeRosterState({
				overviewResult: {
					status: 'fulfilled',
					value: {
						office_bearers: [
							{ designation_id: 'd1', designation_code: 'CHAIRMAN', designation_name: 'Chairman' },
							{
								designation_id: 'd2',
								designation_code: 'SECRETARY',
								designation_name: 'Secretary'
							},
							{ designation_id: 'd3', designation_code: 'TREASURER', designation_name: 'Treasurer' }
						]
					}
				},
				appointmentsResult: {
					status: 'fulfilled',
					value: { appointments: [] }
				},
				filter: 'ACTIVE'
			});

			expect(state.officeBearers).toHaveLength(3);
			expect(state.officeBearers[0].member_id).toBeUndefined();
			expect(state.appointments).toHaveLength(0);
			expect(state.isEmptyState).toBe(true);
			expect(state.error).toBeNull();
		});

		it('Missing optional expected_end_date renders without errors', () => {
			const mockOfficeBearers = [
				{
					designation_id: 'd1',
					designation_code: 'CHAIRMAN',
					designation_name: 'Chairman',
					member_id: 'm1',
					member_name: 'Vijay Godbole',
					start_date: '2026-01-01',
					expected_end_date: undefined,
					status: 'ACTIVE'
				}
			];

			const state = computeRosterState({
				overviewResult: {
					status: 'fulfilled',
					value: { office_bearers: mockOfficeBearers }
				},
				appointmentsResult: {
					status: 'fulfilled',
					value: { appointments: [] }
				},
				filter: 'ACTIVE'
			});

			expect(state.officeBearers[0].expected_end_date).toBeUndefined();
			expect(state.error).toBeNull();
		});

		it('Overview request failure does not wipe appointments', () => {
			const appointments = [
				{
					id: 'a1',
					designation_id: 'd1',
					designation_code: 'CHAIRMAN',
					designation_name: 'Chairman',
					member_id: 'm1',
					member_name: 'Vijay Godbole',
					member_phone: '+91 9822000101',
					start_date: '2026-01-01',
					status: 'ACTIVE' as const
				}
			];

			const state = computeRosterState({
				overviewResult: {
					status: 'rejected',
					reason: { statusCode: 500, message: 'Server error on overview' }
				},
				appointmentsResult: {
					status: 'fulfilled',
					value: { appointments }
				},
				filter: 'ACTIVE'
			});

			// Appointments must still be populated
			expect(state.appointments).toHaveLength(1);
			expect(state.appointments[0].member_name).toBe('Vijay Godbole');
			// Derived office bearers still display Chairman
			expect(state.officeBearers[0].member_name).toBe('Vijay Godbole');
		});

		it('Appointment request success while overview fails with 404', () => {
			const appointments = [
				{
					id: 'a1',
					designation_id: 'd1',
					designation_code: 'CHAIRMAN',
					designation_name: 'Chairman',
					member_id: 'm1',
					member_name: 'Vijay Godbole',
					member_phone: '+91 9822000101',
					start_date: '2026-01-01',
					status: 'ACTIVE' as const
				},
				{
					id: 'a2',
					designation_id: 'd4',
					designation_code: 'COMMITTEE_MEMBER',
					designation_name: 'Committee Member',
					member_id: 'm2',
					member_name: 'Karan Saxena',
					member_phone: '+91 9822000115',
					start_date: '2026-10-11',
					status: 'ACTIVE' as const
				}
			];

			const state = computeRosterState({
				overviewResult: {
					status: 'rejected',
					reason: { statusCode: 404, message: 'Resource not found' }
				},
				appointmentsResult: {
					status: 'fulfilled',
					value: { appointments }
				},
				filter: 'ACTIVE'
			});

			// 404 on overview gracefully ignored when appointments succeeds
			expect(state.error).toBeNull();
			expect(state.appointments).toHaveLength(2);
			expect(state.displayedCount).toBe(2);
		});

		it('Supports ACTIVE, ENDED, REVOKED, and ALL filters', () => {
			const allAppointments = [
				{
					id: 'a1',
					designation_id: 'd1',
					designation_code: 'CHAIRMAN',
					designation_name: 'Chairman',
					member_id: 'm1',
					member_name: 'Vijay Godbole',
					member_phone: '+91 9822000101',
					start_date: '2026-01-01',
					status: 'ACTIVE' as const
				},
				{
					id: 'a2',
					designation_id: 'd2',
					designation_code: 'SECRETARY',
					designation_name: 'Secretary',
					member_id: 'm2',
					member_name: 'Old Secretary',
					member_phone: '+91 9822000000',
					start_date: '2025-01-01',
					end_date: '2025-12-31',
					status: 'ENDED' as const
				},
				{
					id: 'a3',
					designation_id: 'd3',
					designation_code: 'TREASURER',
					designation_name: 'Treasurer',
					member_id: 'm3',
					member_name: 'Revoked Treasurer',
					member_phone: '+91 9822000001',
					start_date: '2025-06-01',
					end_date: '2025-08-01',
					status: 'REVOKED' as const
				}
			];

			const stateAll = computeRosterState({
				appointmentsResult: { status: 'fulfilled', value: { appointments: allAppointments } },
				filter: 'ALL'
			});
			expect(stateAll.appointments).toHaveLength(3);

			const stateActive = computeRosterState({
				appointmentsResult: { status: 'fulfilled', value: { appointments: allAppointments } },
				filter: 'ACTIVE'
			});
			expect(stateActive.appointments).toHaveLength(1);
			expect(stateActive.appointments[0].status).toBe('ACTIVE');

			const stateEnded = computeRosterState({
				appointmentsResult: { status: 'fulfilled', value: { appointments: allAppointments } },
				filter: 'ENDED'
			});
			expect(stateEnded.appointments).toHaveLength(1);
			expect(stateEnded.appointments[0].status).toBe('ENDED');

			const stateRevoked = computeRosterState({
				appointmentsResult: { status: 'fulfilled', value: { appointments: allAppointments } },
				filter: 'REVOKED'
			});
			expect(stateRevoked.appointments).toHaveLength(1);
			expect(stateRevoked.appointments[0].status).toBe('REVOKED');
		});

		it('Correct mapping from API response envelope to UI state', () => {
			const envelope = {
				success: true,
				data: {
					appointments: [
						{
							id: 'a1',
							designation_id: 'd1',
							designation_code: 'CHAIRMAN',
							designation_name: 'Chairman',
							member_id: 'm1',
							member_name: 'Vijay Godbole',
							member_phone: '+91 9822000101',
							start_date: '2026-01-01',
							status: 'ACTIVE' as const
						}
					]
				},
				meta: 'test-meta'
			};

			const state = computeRosterState({
				appointmentsResult: { status: 'fulfilled', value: envelope.data },
				filter: 'ACTIVE'
			});

			expect(state.appointments).toHaveLength(1);
			expect(state.appointments[0].member_name).toBe('Vijay Godbole');
			expect(state.error).toBeNull();
		});

		it('No false zero appointments display when records exist', () => {
			const appointments = [
				{
					id: 'a1',
					designation_id: 'd1',
					designation_code: 'CHAIRMAN',
					designation_name: 'Chairman',
					member_id: 'm1',
					member_name: 'Vijay Godbole',
					member_phone: '+91 9822000101',
					start_date: '2026-01-01',
					status: 'ACTIVE' as const
				},
				{
					id: 'a2',
					designation_id: 'd2',
					designation_code: 'SECRETARY',
					designation_name: 'Secretary',
					member_id: 'm2',
					member_name: 'Milind Date',
					member_phone: '+91 9822000103',
					start_date: '2026-01-01',
					status: 'ACTIVE' as const
				},
				{
					id: 'a3',
					designation_id: 'd3',
					designation_code: 'TREASURER',
					designation_name: 'Treasurer',
					member_id: 'm3',
					member_name: 'Sudhir Apte',
					member_phone: '+91 9822000104',
					start_date: '2026-01-01',
					status: 'ACTIVE' as const
				},
				{
					id: 'a4',
					designation_id: 'd4',
					designation_code: 'COMMITTEE_MEMBER',
					designation_name: 'Committee Member',
					member_id: 'm4',
					member_name: 'Karan Saxena',
					member_phone: '+91 9822000115',
					start_date: '2026-10-11',
					status: 'ACTIVE' as const
				},
				{
					id: 'a5',
					designation_id: 'd4',
					designation_code: 'COMMITTEE_MEMBER',
					designation_name: 'Committee Member',
					member_id: 'm5',
					member_name: 'Rekha Chitnis',
					member_phone: '+91 9822000106',
					start_date: '2026-01-01',
					status: 'ACTIVE' as const
				}
			];

			const state = computeRosterState({
				overviewResult: {
					status: 'rejected',
					reason: { statusCode: 404, message: 'Not found' }
				},
				appointmentsResult: {
					status: 'fulfilled',
					value: { appointments }
				},
				filter: 'ACTIVE'
			});

			expect(state.displayedCount).toBe(5);
			expect(state.isEmptyState).toBe(false);
			expect(state.appointments.length).toBeGreaterThan(0);
		});
	});
});
