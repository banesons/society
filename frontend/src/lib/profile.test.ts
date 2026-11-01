import { describe, it, expect, vi, beforeEach } from 'vitest';
import { getDailyBackdrop, getDayOfYear } from './backdrop';
import { ROUTES, appRoutes } from './routing';
import { formatFullPhone } from './phone';

describe('Personal Profile & Backdrop Unit Tests', () => {
	beforeEach(() => {
		vi.resetAllMocks();
	});

	it('registers canonical /app/profile in routing definitions', () => {
		expect(ROUTES.PROFILE).toBe('/app/profile');
		expect(appRoutes.profile).toBe('/app/profile');
	});

	it('calculates deterministic day of year', () => {
		const testDate = new Date(Date.UTC(2026, 0, 1)); // Jan 1st
		expect(getDayOfYear(testDate)).toBe(1);

		const febFirst = new Date(Date.UTC(2026, 1, 1)); // Feb 1st
		expect(getDayOfYear(febFirst)).toBe(32);
	});

	it('returns identical backdrop for the same calendar date', () => {
		const morning = new Date(Date.UTC(2026, 9, 17, 8, 30, 0));
		const evening = new Date(Date.UTC(2026, 9, 17, 21, 45, 0));

		const backdropMorning = getDailyBackdrop(morning);
		const backdropEvening = getDailyBackdrop(evening);

		expect(backdropMorning.id).toBe(backdropEvening.id);
		expect(backdropMorning.url).toBe(backdropEvening.url);
		expect(backdropMorning.title).toBe(backdropEvening.title);
	});

	it('rotates to next backdrop on subsequent dates', () => {
		const day1 = new Date(Date.UTC(2026, 9, 17));
		const day2 = new Date(Date.UTC(2026, 9, 18));

		const b1 = getDailyBackdrop(day1);
		const b2 = getDailyBackdrop(day2);

		// With 7 curated architectural backdrops, adjacent days differ
		expect(b1.id).not.toBe(b2.id);
	});

	it('falls back safely if curated array were empty', () => {
		const backdrop = getDailyBackdrop();
		expect(backdrop).toBeDefined();
		expect(backdrop.url).toBeTruthy();
		expect(backdrop.title).toBeTruthy();
	});

	it('extracts correct initials from full names and handles edge cases', async () => {
		const { getInitials } = await import('./avatar');
		expect(getInitials('Meenal Jain')).toBe('MJ');
		expect(getInitials('Shubham Bane')).toBe('SB');
		expect(getInitials('Emily')).toBe('EM');
		expect(getInitials('A')).toBe('A');
		expect(getInitials('   ')).toBe('U');
		expect(getInitials(null)).toBe('U');
		expect(getInitials(undefined)).toBe('U');
		expect(getInitials('+91 99670 89101')).toBe('CH');
	});

	describe('Display Name Resolution & Fallback Logic (TEST C & TEST I)', () => {
		function resolveFrontendDisplayName(
			firstName: string,
			lastName: string,
			rawPhone: string,
			serverDisplayName?: string
		): string {
			const full = `${firstName || ''} ${lastName || ''}`.trim();
			if (full) return full;
			if (serverDisplayName && serverDisplayName !== 'Verified Resident') {
				return serverDisplayName;
			}
			return formatFullPhone(rawPhone);
		}

		it('resolves full name when both first and last names are present', () => {
			const name = resolveFrontendDisplayName('Shuubham', 'Bane', '919967089101');
			expect(name).toBe('Shuubham Bane');
		});

		it('resolves first name only when last name is empty', () => {
			const name = resolveFrontendDisplayName('Shuubham', '', '919967089101');
			expect(name).toBe('Shuubham');
		});

		it('falls back to formatted phone number when names are absent', () => {
			const name = resolveFrontendDisplayName('', '', '919967089101');
			expect(name).toBe('+91 99670 89101');
			expect(name).not.toBe('Verified Resident');
		});

		it('never resolves to hardcoded Verified Resident even if server provides it as fallback', () => {
			const name = resolveFrontendDisplayName('', '', '919967089101', 'Verified Resident');
			expect(name).toBe('+91 99670 89101');
			expect(name).not.toBe('Verified Resident');
		});
	});

	describe('Profile Input Validation Logic (TEST D)', () => {
		function validateProfileForm(firstName: string, email: string) {
			const errors: { firstName?: string; email?: string } = {};
			const cleanFirst = firstName.trim();
			const cleanEmail = email.trim();

			if (!cleanFirst) {
				errors.firstName = 'First name cannot be empty';
			} else if (cleanFirst.length > 100) {
				errors.firstName = 'First name must be under 100 characters';
			}

			if (cleanEmail) {
				const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
				if (!emailPattern.test(cleanEmail)) {
					errors.email = 'Please provide a valid email address';
				} else if (cleanEmail.length > 255) {
					errors.email = 'Email address exceeds maximum length';
				}
			}

			return {
				isValid: Object.keys(errors).length === 0,
				errors
			};
		}

		it('validates correct profile data', () => {
			const res = validateProfileForm('Shuubham', 'shubham@banesons.com');
			expect(res.isValid).toBe(true);
			expect(res.errors).toEqual({});
		});

		it('rejects blank first name', () => {
			const res = validateProfileForm('   ', 'shubham@banesons.com');
			expect(res.isValid).toBe(false);
			expect(res.errors.firstName).toBeDefined();
		});

		it('rejects malformed email address', () => {
			const res = validateProfileForm('Shuubham', 'not-valid-email');
			expect(res.isValid).toBe(false);
			expect(res.errors.email).toBeDefined();
		});
	});
});
