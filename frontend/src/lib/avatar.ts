/**
 * Utility functions for user avatars and initials generation.
 */

/**
 * Extracts clean, accessible initials from a full name or display name.
 * Examples:
 * - "Meenal Jain" -> "MJ"
 * - "Shubham Bane" -> "SB"
 * - "Shubham" -> "S"
 * - "" -> "U"
 */
export function getInitials(name?: string | null): string {
	if (!name) return 'U';
	const trimmed = name.trim();
	if (!trimmed) return 'U';

	const alphaOnly = trimmed.replace(/[^a-zA-Z\s]/g, '').trim();
	if (!alphaOnly) return 'CH';

	const parts = alphaOnly.split(/\s+/).filter(Boolean);
	if (parts.length === 0) return 'CH';

	if (parts.length === 1) {
		const single = parts[0];
		return single.slice(0, Math.min(2, single.length)).toUpperCase();
	}

	const first = parts[0][0];
	const last = parts[parts.length - 1][0];
	return (first + last).toUpperCase();
}
