/**
 * Phone formatting and normalization utility functions.
 * Stored in DB and transferred over API strictly as 12-digit Indian integers (91XXXXXXXXXX).
 * Input fields display a non-editable visual {+} adornment.
 */

/**
 * Strips all spaces, +, dashes, and non-digits.
 * Converts 10-digit Indian numbers (6-9...) to canonical 12-digit format (91XXXXXXXXXX).
 */
export function normalizePhoneInput(input: string): string {
	if (!input) return '';
	// Strip all non-digit characters (including spaces, +, parentheses, dashes)
	const digits = input.replace(/\D/g, '');

	if (digits.length === 10 && /^[6-9]/.test(digits)) {
		return '91' + digits;
	}
	if (digits.length === 11 && digits.startsWith('0') && /^[6-9]/.test(digits.slice(1))) {
		return '91' + digits.slice(1);
	}
	if (digits.length === 12 && digits.startsWith('91') && /^[6-9]/.test(digits.slice(2))) {
		return digits;
	}
	return digits;
}

/**
 * Validates whether the input produces a valid 12-digit Indian mobile number (91[6-9]XXXXXXXXX).
 */
export function isValidIndianPhone(input: string): boolean {
	const normalized = normalizePhoneInput(input);
	return /^91[6-9]\d{9}$/.test(normalized);
}

export function formatFullPhone(phone?: string | null): string {
	if (!phone) return '—';
	const digits = phone.replace(/\D/g, '');
	if (digits.length === 12 && digits.startsWith('91')) {
		const national = digits.slice(2);
		return `+91 ${national.slice(0, 5)} ${national.slice(5)}`;
	}
	if (digits.length === 10 && /^[6-9]/.test(digits)) {
		return `+91 ${digits.slice(0, 5)} ${digits.slice(5)}`;
	}
	return phone;
}

/**
 * Normalizes input to 12-digit Indian integer format (91XXXXXXXXXX).
 * Throws an error if invalid.
 */
export function normalizePhone(input: string): string {
	const normalized = normalizePhoneInput(input);
	if (!isValidIndianPhone(normalized)) {
		throw new Error('Invalid Indian mobile phone number');
	}
	return normalized;
}

/**
 * Formats phone for clean UI display.
 */
export function formatPhoneDisplay(phone?: string | null): string {
	return formatFullPhone(phone);
}
