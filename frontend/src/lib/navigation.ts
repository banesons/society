import type { Component } from 'svelte';
import {
	Building2,
	Users,
	Shield,
	Layers,
	DoorClosed,
	Wrench,
	FileText,
	MessageSquare,
	BarChart3,
	ScrollText,
	UserCheck
} from '@lucide/svelte';

export type SocietyRole = 'SUPER_ADMIN' | 'ADMIN' | 'SOC_ADMIN' | 'GATEKEEPER' | 'RESIDENT';

export interface SocietyNavItem {
	id: string;
	label: string;
	href: string;
	icon: Component;
	sectionId: string;
}

/**
 * Resolves the effective operational role for the current society.
 */
export function getEffectiveSocietyRole(
	platformRoles?: string[],
	societyAssignments?: Array<{ society_id: string; role: string }>,
	currentSocietyId?: string
): SocietyRole {
	if (platformRoles?.includes('SUPER_ADMIN')) {
		return 'SUPER_ADMIN';
	}
	if (platformRoles?.includes('ADMIN')) {
		const assignment = societyAssignments?.find((a) => a.society_id === currentSocietyId);
		return (assignment?.role as SocietyRole) || 'ADMIN';
	}
	const assignment = societyAssignments?.find((a) => a.society_id === currentSocietyId);
	if (assignment?.role === 'SOC_ADMIN') {
		return 'SOC_ADMIN';
	}
	if (assignment?.role === 'GATEKEEPER') {
		return 'GATEKEEPER';
	}
	return 'RESIDENT';
}

/**
 * Returns role-specific, verified operational navigation items.
 *
 * Guarantees:
 * 1. Residents see ONLY their 5 verified functional destinations.
 * 2. Administrative items are strictly hidden from non-admin roles.
 * 3. Gatekeepers see only perimeter operations and broadcast notices.
 * 4. Planned or unimplemented features (e.g. Settings) are NEVER returned.
 * 5. Every link targets a verified DOM section rendered for that role.
 */
export function getSocietyNavItems(role: SocietyRole, societyId: string): SocietyNavItem[] {
	if (!societyId) return [];

	if (role === 'RESIDENT') {
		return [
			{
				id: 'overview',
				label: 'Home',
				href: `/app/societies/${societyId}#overview`,
				icon: Building2,
				sectionId: 'overview'
			},
			{
				id: 'announcements',
				label: 'Announcements',
				href: `/app/societies/${societyId}#announcements`,
				icon: MessageSquare,
				sectionId: 'announcements'
			},
			{
				id: 'maintenance',
				label: 'Maintenance & Dues',
				href: `/app/societies/${societyId}#resident-maintenance`,
				icon: Wrench,
				sectionId: 'resident-maintenance'
			},
			{
				id: 'documents',
				label: 'Society Documents',
				href: `/app/societies/${societyId}#resident-documents`,
				icon: FileText,
				sectionId: 'resident-documents'
			},
			{
				id: 'resident-hub',
				label: 'Visitor Approvals & Passes',
				href: `/app/societies/${societyId}#resident-hub`,
				icon: UserCheck,
				sectionId: 'resident-hub'
			}
		];
	}

	if (role === 'GATEKEEPER') {
		return [
			{
				id: 'overview',
				label: 'Home',
				href: `/app/societies/${societyId}#overview`,
				icon: Building2,
				sectionId: 'overview'
			},
			{
				id: 'gatekeeping',
				label: 'Perimeter & Gatekeeping',
				href: `/app/societies/${societyId}#gatekeeping`,
				icon: DoorClosed,
				sectionId: 'gatekeeping'
			},
			{
				id: 'announcements',
				label: 'Announcements',
				href: `/app/societies/${societyId}#announcements`,
				icon: MessageSquare,
				sectionId: 'announcements'
			}
		];
	}

	// SOC_ADMIN, SUPER_ADMIN, ADMIN (Platform / Society Administrators)
	return [
		{
			id: 'overview',
			label: 'Overview',
			href: `/app/societies/${societyId}#overview`,
			icon: Building2,
			sectionId: 'overview'
		},
		{
			id: 'reports',
			label: 'Dashboard & Reports',
			href: `/app/societies/${societyId}#reports`,
			icon: BarChart3,
			sectionId: 'reports'
		},
		{
			id: 'members',
			label: 'Members & Residents',
			href: `/app/societies/${societyId}#members`,
			icon: Users,
			sectionId: 'members'
		},
		{
			id: 'committee',
			label: 'Managing Committee',
			href: `/app/societies/${societyId}#committee`,
			icon: Shield,
			sectionId: 'committee'
		},
		{
			id: 'structure',
			label: 'Physical Structure',
			href: `/app/societies/${societyId}#structure`,
			icon: Layers,
			sectionId: 'structure'
		},
		{
			id: 'gatekeeping',
			label: 'Gatekeeping Console',
			href: `/app/societies/${societyId}#gatekeeping`,
			icon: DoorClosed,
			sectionId: 'gatekeeping'
		},
		{
			id: 'announcements',
			label: 'Announcements',
			href: `/app/societies/${societyId}#announcements`,
			icon: MessageSquare,
			sectionId: 'announcements'
		},
		{
			id: 'maintenance',
			label: 'Maintenance & Billing',
			href: `/app/societies/${societyId}#maintenance`,
			icon: Wrench,
			sectionId: 'maintenance'
		},
		{
			id: 'documents',
			label: 'Documents & Records',
			href: `/app/societies/${societyId}#documents`,
			icon: FileText,
			sectionId: 'documents'
		},
		{
			id: 'audit',
			label: 'Audit & Activity',
			href: `/app/societies/${societyId}#audit`,
			icon: ScrollText,
			sectionId: 'audit'
		},
		{
			id: 'resident-hub',
			label: 'Resident Visitor Hub',
			href: `/app/societies/${societyId}#resident-hub`,
			icon: UserCheck,
			sectionId: 'resident-hub'
		}
	];
}

/**
 * Maps any raw URL hash or target section identifier to its canonical active nav item id.
 */
export function normalizeActiveSection(hashOrId?: string): string {
	if (!hashOrId) return 'overview';
	const clean = hashOrId.replace(/^#/, '');
	if (clean === 'resident-maintenance') return 'maintenance';
	if (clean === 'resident-documents') return 'documents';
	if (clean === 'announcements-admin') return 'announcements';
	return clean;
}

export type NavGroup = 'OVERVIEW' | 'PEOPLE' | 'OPERATIONS' | 'INSIGHTS';

export interface GroupedNavSection {
	group: NavGroup;
	label: string;
	items: SocietyNavItem[];
}

/**
 * Returns modular, role-gated society navigation organized into logical groups (ADR 040).
 */
export function getGroupedSocietyNavItems(
	role: SocietyRole,
	societyId: string
): GroupedNavSection[] {
	if (!societyId) return [];

	const sections: GroupedNavSection[] = [];

	// 1. OVERVIEW (Admins and Residents only; Gatekeepers focus strictly on Gate Operations)
	if (role !== 'GATEKEEPER') {
		sections.push({
			group: 'OVERVIEW',
			label: 'Overview',
			items: [
				{
					id: 'overview',
					label: 'Society Overview',
					href: `/app/societies/${societyId}`,
					icon: Building2,
					sectionId: 'overview'
				}
			]
		});
	}

	// 2. PEOPLE (Admins only)
	if (role === 'SUPER_ADMIN' || role === 'ADMIN' || role === 'SOC_ADMIN') {
		sections.push({
			group: 'PEOPLE',
			label: 'People',
			items: [
				{
					id: 'members',
					label: 'Members & Residents',
					href: `/app/societies/${societyId}/members`,
					icon: Users,
					sectionId: 'members'
				},
				{
					id: 'committee',
					label: 'Managing Committee',
					href: `/app/societies/${societyId}/committee`,
					icon: Shield,
					sectionId: 'committee'
				}
			]
		});
	}

	// 3. OPERATIONS
	const operationsItems: SocietyNavItem[] = [];

	if (role === 'SUPER_ADMIN' || role === 'ADMIN' || role === 'SOC_ADMIN') {
		operationsItems.push(
			{
				id: 'structure',
				label: 'Physical Structure',
				href: `/app/societies/${societyId}/structure`,
				icon: Layers,
				sectionId: 'structure'
			},
			{
				id: 'gate',
				label: 'Gate Management',
				href: `/app/societies/${societyId}/gate`,
				icon: DoorClosed,
				sectionId: 'gatekeeping'
			},
			{
				id: 'announcements',
				label: 'Announcements',
				href: `/app/societies/${societyId}/announcements`,
				icon: MessageSquare,
				sectionId: 'announcements'
			},
			{
				id: 'maintenance',
				label: 'Maintenance & Billing',
				href: `/app/societies/${societyId}/maintenance`,
				icon: Wrench,
				sectionId: 'maintenance'
			},
			{
				id: 'resident-hub',
				label: 'Resident Visitor Hub',
				href: `/app/societies/${societyId}/resident-hub`,
				icon: UserCheck,
				sectionId: 'resident-hub'
			},
			{
				id: 'documents',
				label: 'Documents & Records',
				href: `/app/societies/${societyId}/documents`,
				icon: FileText,
				sectionId: 'documents'
			}
		);
	} else if (role === 'GATEKEEPER') {
		operationsItems.push(
			{
				id: 'gate',
				label: 'Gate Console',
				href: `/app/societies/${societyId}/gate`,
				icon: DoorClosed,
				sectionId: 'gatekeeping'
			},
			{
				id: 'announcements',
				label: 'Announcements',
				href: `/app/societies/${societyId}/announcements`,
				icon: MessageSquare,
				sectionId: 'announcements'
			}
		);
	} else {
		// RESIDENT
		operationsItems.push(
			{
				id: 'announcements',
				label: 'Announcements',
				href: `/app/societies/${societyId}/announcements`,
				icon: MessageSquare,
				sectionId: 'announcements'
			},
			{
				id: 'maintenance',
				label: 'Maintenance & Dues',
				href: `/app/societies/${societyId}/maintenance`,
				icon: Wrench,
				sectionId: 'resident-maintenance'
			},
			{
				id: 'resident-hub',
				label: 'Visitor Passes & Approvals',
				href: `/app/societies/${societyId}/resident-hub`,
				icon: UserCheck,
				sectionId: 'resident-hub'
			},
			{
				id: 'documents',
				label: 'Society Documents',
				href: `/app/societies/${societyId}/documents`,
				icon: FileText,
				sectionId: 'resident-documents'
			}
		);
	}

	if (operationsItems.length > 0) {
		sections.push({
			group: 'OPERATIONS',
			label: 'Operations',
			items: operationsItems
		});
	}

	// 4. INSIGHTS (Admins only)
	if (role === 'SUPER_ADMIN' || role === 'ADMIN' || role === 'SOC_ADMIN') {
		sections.push({
			group: 'INSIGHTS',
			label: 'Insights',
			items: [
				{
					id: 'reports',
					label: 'Reports & Analytics',
					href: `/app/societies/${societyId}/reports`,
					icon: BarChart3,
					sectionId: 'reports'
				},
				{
					id: 'audit',
					label: 'Audit Logs',
					href: `/app/societies/${societyId}/audit`,
					icon: ScrollText,
					sectionId: 'audit'
				}
			]
		});
	}

	return sections;
}

/**
 * Route-aware matching that distinguishes exact routes from their descendants.
 */
export function isSocietyModuleActive(
	moduleHref: string,
	currentPath: string,
	societyId: string
): boolean {
	const baseSocietyPath = `/app/societies/${societyId}`;
	if (moduleHref === baseSocietyPath) {
		return currentPath === baseSocietyPath || currentPath === `${baseSocietyPath}/`;
	}
	return currentPath === moduleHref || currentPath.startsWith(`${moduleHref}/`);
}
