import type { UserProfile } from '$lib/auth.svelte';
import type { UserSocietyAssignment } from '$lib/api/society';

export type PostLoginResolution =
	| { type: 'navigate'; path: string }
	| { type: 'select_society'; assignments: UserSocietyAssignment[] }
	| { type: 'unauthorized'; reason: string };

/**
 * Resolves the authenticated user's landing destination based on platform roles
 * and verified society assignments per CHS routing policy.
 */
export function resolvePostLoginRoute(user: UserProfile | null): PostLoginResolution {
	if (!user) {
		return { type: 'unauthorized', reason: 'UNAUTHENTICATED' };
	}

	// 1. Platform Super Admin: preserve /admin as default platform destination
	if (user.platform_roles?.includes('SUPER_ADMIN')) {
		return { type: 'navigate', path: '/admin' };
	}

	// 2. Platform Admin: preserve /admin as default; no implicit society access
	if (user.platform_roles?.includes('ADMIN')) {
		return { type: 'navigate', path: '/admin' };
	}

	// 3. Society-level resolution based on authenticated assignments
	const assignments = user.society_assignments ?? [];

	if (assignments.length === 0) {
		return { type: 'unauthorized', reason: 'NO_ACCESSIBLE_SOCIETIES' };
	}

	if (assignments.length === 1) {
		return { type: 'navigate', path: `/society/${assignments[0].society_id}` };
	}

	// Multiple accessible societies: require explicit selection
	return { type: 'select_society', assignments };
}

export interface BackDestination {
	href: string;
	label: string;
}

/**
 * Determines contextual back-navigation destination from a society route.
 */
export function getContextualBackDestination(
	user: UserProfile | null,
	referrer?: string
): BackDestination {
	// If the user has platform administration privileges (Super Admin or Admin)
	if (user?.platform_roles?.includes('SUPER_ADMIN') || user?.platform_roles?.includes('ADMIN')) {
		return {
			href: '/admin',
			label: 'Return to Admin Console'
		};
	}

	// If referrer was /admin and user has platform role
	if (referrer && referrer.includes('/admin')) {
		return {
			href: '/admin',
			label: 'Return to Admin Console'
		};
	}

	// Standard safe fallback for ordinary residents / society admins
	return {
		href: '/',
		label: 'Return to Home'
	};
}
