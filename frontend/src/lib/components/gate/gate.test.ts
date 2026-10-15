import { describe, it, expect } from 'vitest';
import { normalizePhone, formatPhoneDisplay } from '$lib/phone';

describe('S08 Gate Foundation Frontend Logic & Validation', () => {
	describe('Gate Configuration Validation', () => {
		function validateGateConfig(data: {
			name: string;
			directionality: string;
			operationalStatus: string;
			requiredStaff: number;
		}) {
			const errors: Record<string, string> = {};
			if (!data.name.trim()) {
				errors.name = 'Gate name cannot be blank';
			}
			const validDirs = ['BIDIRECTIONAL', 'INGRESS_ONLY', 'EGRESS_ONLY'];
			if (!validDirs.includes(data.directionality)) {
				errors.directionality = 'Invalid directionality selection';
			}
			const validStatuses = ['ACTIVE', 'INACTIVE', 'MAINTENANCE', 'EMERGENCY_OPEN'];
			if (!validStatuses.includes(data.operationalStatus)) {
				errors.operationalStatus = 'Invalid operational status';
			}
			if (data.requiredStaff < 0) {
				errors.requiredStaff = 'Required staff cannot be negative';
			}
			return {
				isValid: Object.keys(errors).length === 0,
				errors
			};
		}

		it('validates correct gate configuration', () => {
			const res = validateGateConfig({
				name: 'North Inbound Gate',
				directionality: 'INGRESS_ONLY',
				operationalStatus: 'ACTIVE',
				requiredStaff: 2
			});
			expect(res.isValid).toBe(true);
		});

		it('rejects blank name and negative staff count', () => {
			const res = validateGateConfig({
				name: '',
				directionality: 'BIDIRECTIONAL',
				operationalStatus: 'ACTIVE',
				requiredStaff: -1
			});
			expect(res.isValid).toBe(false);
			expect(res.errors.name).toBeDefined();
			expect(res.errors.requiredStaff).toBeDefined();
		});

		it('rejects invalid directionality and status', () => {
			const res = validateGateConfig({
				name: 'Main Gate',
				directionality: 'UNKNOWN_DIR',
				operationalStatus: 'INVALID_STATUS',
				requiredStaff: 1
			});
			expect(res.isValid).toBe(false);
			expect(res.errors.directionality).toBeDefined();
			expect(res.errors.operationalStatus).toBeDefined();
		});
	});

	describe('Gatekeeper Assignment Phone Normalization', () => {
		it('normalizes valid Indian mobile phone', () => {
			const phone = '9820012345';
			const normalized = normalizePhone(phone);
			expect(normalized).toBe('919820012345');
			expect(formatPhoneDisplay(normalized)).toBe('+91 98200 12345');
		});

		it('rejects invalid short phone numbers', () => {
			expect(() => normalizePhone('12345')).toThrow();
		});
	});

	describe('Shift Schedule Validation', () => {
		function validateShift(data: { name: string; startTime: string; endTime: string }) {
			const errors: Record<string, string> = {};
			if (!data.name.trim()) {
				errors.name = 'Shift name is required';
			}
			if (!data.startTime) {
				errors.startTime = 'Start time is required';
			}
			if (!data.endTime) {
				errors.endTime = 'End time is required';
			}
			return {
				isValid: Object.keys(errors).length === 0,
				errors
			};
		}

		it('accepts valid shift schedule', () => {
			const res = validateShift({
				name: 'Morning Shift',
				startTime: '06:00',
				endTime: '14:00'
			});
			expect(res.isValid).toBe(true);
		});

		it('rejects empty shift name', () => {
			const res = validateShift({
				name: '',
				startTime: '06:00',
				endTime: '14:00'
			});
			expect(res.isValid).toBe(false);
			expect(res.errors.name).toBeDefined();
		});
	});

	describe('Duty Check-In Gate Availability', () => {
		function canCheckInToGate(gate: { status: string; operational_status: string }): boolean {
			return gate.status === 'ACTIVE' && gate.operational_status !== 'INACTIVE';
		}

		it('allows check-in to active open gate', () => {
			expect(canCheckInToGate({ status: 'ACTIVE', operational_status: 'ACTIVE' })).toBe(true);
		});

		it('allows check-in to emergency open gate', () => {
			expect(canCheckInToGate({ status: 'ACTIVE', operational_status: 'EMERGENCY_OPEN' })).toBe(
				true
			);
		});

		it('rejects check-in to inactive gate', () => {
			expect(canCheckInToGate({ status: 'INACTIVE', operational_status: 'ACTIVE' })).toBe(false);
			expect(canCheckInToGate({ status: 'ACTIVE', operational_status: 'INACTIVE' })).toBe(false);
		});
	});

	describe('API Error Message Formatting', () => {
		function formatGateError(code: string): string {
			switch (code) {
				case 'SESSION_ALREADY_ACTIVE':
					return 'You are already checked into an active duty session. Please check out first.';
				case 'GATE_INACTIVE':
					return 'This gate is currently marked inactive or closed for operations.';
				case 'GATEKEEPER_ALREADY_ASSIGNED':
					return 'This user already holds an active gatekeeper assignment in this society';
				case 'SHIFT_NAME_EXISTS':
					return 'A shift with this name already exists in this society';
				case 'GATE_NOT_FOUND':
					return 'Target gate could not be found';
				default:
					return 'An unexpected perimeter error occurred';
			}
		}

		it('maps SESSION_ALREADY_ACTIVE correctly', () => {
			expect(formatGateError('SESSION_ALREADY_ACTIVE')).toContain(
				'already checked into an active duty session'
			);
		});

		it('maps GATE_INACTIVE correctly', () => {
			expect(formatGateError('GATE_INACTIVE')).toContain('closed for operations');
		});

		it('maps GATEKEEPER_ALREADY_ASSIGNED correctly', () => {
			expect(formatGateError('GATEKEEPER_ALREADY_ASSIGNED')).toContain(
				'already holds an active gatekeeper assignment'
			);
		});

		it('maps SHIFT_NAME_EXISTS correctly', () => {
			expect(formatGateError('SHIFT_NAME_EXISTS')).toContain('already exists');
		});
	});
});
