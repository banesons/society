import type { UserProfile } from '$lib/auth.svelte';
import type { UserSocietyAssignment } from '$lib/api/society';

/**
 * Centralized canonical route paths for public and authenticated surfaces.
 */
export const ROUTES = {
	HOME: '/',
	LOGIN: '/app/login',
	LEGACY_LOGIN: '/login',
	APP: '/app',
	DASHBOARD: '/app/dashboard',
	NOTIFICATIONS: '/app/notifications',
	PROFILE: '/app/profile',
	SOCIETIES: '/app/societies',
	ADMIN: '/app/admin',
	LEGACY_ADMIN: '/admin',
	DEVELOPER: '/developer',
	UNAUTHORIZED: '/unauthorized'
} as const;

/**
 * Authoritative, strongly-typed route builders for CHS application.
 */
export const appRoutes = {
	root: ROUTES.APP,
	dashboard: ROUTES.DASHBOARD,
	notifications: ROUTES.NOTIFICATIONS,
	profile: ROUTES.PROFILE,
	societies: ROUTES.SOCIETIES,
	society: (id: string, hash?: string) =>
		hash
			? `/app/societies/${encodeURIComponent(id)}${hash.startsWith('#') ? hash : `#${hash}`}`
			: `/app/societies/${encodeURIComponent(id)}`,
	societySection: (id: string, section: string) =>
		`/app/societies/${encodeURIComponent(id)}#${section.replace(/^#/, '')}`,
	gate: (societyId: string) => `/app/societies/${encodeURIComponent(societyId)}/gate`,
	login: (returnTo?: string) =>
		returnTo ? `${ROUTES.LOGIN}?returnTo=${encodeURIComponent(returnTo)}` : ROUTES.LOGIN,
	admin: ROUTES.ADMIN,
	developer: ROUTES.DEVELOPER,
	unauthorized: ROUTES.UNAUTHORIZED
};

export type PostLoginResolution =
	| { type: 'navigate'; path: string }
	| { type: 'select_society'; assignments: UserSocietyAssignment[] }
	| { type: 'unauthorized'; reason: string };

/**
 * Safely sanitizes and validates a candidate redirect path to prevent open redirect vulnerabilities.
 * Only relative, internal application paths starting with '/' are permitted. Protocol-relative '//' is forbidden.
 */
export function sanitizeReturnTo(rawReturnTo: string | null | undefined): string | null {
	if (!rawReturnTo) return null;
	const trimmed = rawReturnTo.trim();
	// Must start with a single slash, not double slashes (protocol-relative), and not contain protocol like 'http:' or 'https:' or 'javascript:'
	if (
		!trimmed.startsWith('/') ||
		trimmed.startsWith('//') ||
		/^[a-zA-Z][a-zA-Z\d+\-.]*:/.test(trimmed)
	) {
		return null;
	}
	// Do not allow looping back to login
	if (trimmed.startsWith('/app/login') || trimmed.startsWith('/login')) {
		return null;
	}
	return trimmed;
}

/**
 * Resolves the authenticated user's landing destination based on platform roles
 * and verified society assignments per CHS routing policy.
 */
export function resolvePostLoginRoute(
	user: UserProfile | null,
	preferredReturnTo?: string | null
): PostLoginResolution {
	if (!user) {
		return { type: 'unauthorized', reason: 'UNAUTHENTICATED' };
	}

	// If a safe internal return destination was explicitly provided, honor it
	const safeReturn = sanitizeReturnTo(preferredReturnTo);
	if (safeReturn) {
		return { type: 'navigate', path: safeReturn };
	}

	// 1. Platform Super Admin: canonical platform destination /app/admin
	if (user.platform_roles?.includes('SUPER_ADMIN')) {
		return { type: 'navigate', path: appRoutes.admin };
	}

	// 2. Platform Admin: canonical /app/admin; no implicit society access
	if (user.platform_roles?.includes('ADMIN')) {
		return { type: 'navigate', path: appRoutes.admin };
	}

	// 3. Society-level resolution based on authenticated assignments
	const assignments = user.society_assignments ?? [];

	if (assignments.length === 0) {
		return { type: 'unauthorized', reason: 'NO_ACCESSIBLE_SOCIETIES' };
	}

	if (assignments.length === 1) {
		// If assignment is GATEKEEPER, directly open the Gate Console
		if (assignments[0].role === 'GATEKEEPER') {
			return { type: 'navigate', path: appRoutes.gate(assignments[0].society_id) };
		}
		return { type: 'navigate', path: appRoutes.society(assignments[0].society_id) };
	}

	// If user only has gatekeeper assignments across multiple societies, prioritize or show selector
	const gatekeeperOnly = assignments.every((a) => a.role === 'GATEKEEPER');
	if (gatekeeperOnly && assignments.length === 1) {
		return { type: 'navigate', path: appRoutes.gate(assignments[0].society_id) };
	}

	// Multiple accessible societies: require explicit selection
	return { type: 'select_society', assignments };
}

export interface BackDestination {
	href: string;
	label: string;
}

/**
 * Determines contextual back-navigation destination from a society or admin route.
 */
export function getContextualBackDestination(
	user: UserProfile | null,
	referrer?: string
): BackDestination {
	// If the user is on /admin or /app/admin, back goes to the canonical application dashboard
	if (referrer && (referrer.includes('/admin') || referrer.includes('/app/admin'))) {
		return {
			href: appRoutes.dashboard,
			label: 'Back to Dashboard'
		};
	}

	// If the user has platform administration privileges navigating from society
	if (user?.platform_roles?.includes('SUPER_ADMIN') || user?.platform_roles?.includes('ADMIN')) {
		return {
			href: appRoutes.admin,
			label: 'Return to Admin Console'
		};
	}

	// Canonical authenticated application dashboard fallback (never marketing home '/')
	return {
		href: appRoutes.dashboard,
		label: 'Back to Dashboard'
	};
}
