import { describe, it, expect } from 'vitest';
import {
	getSocietyNavItems,
	getGroupedSocietyNavItems,
	getEffectiveSocietyRole,
	normalizeActiveSection,
	isSocietyModuleActive,
	type SocietyRole
} from './navigation';

describe('Role-Based Society Navigation (ADR 033)', () => {
	const testSocId = 'soc-greenview-202';

	describe('Role Resolution (getEffectiveSocietyRole)', () => {
		it('resolves SUPER_ADMIN regardless of society assignments', () => {
			const role = getEffectiveSocietyRole(['SUPER_ADMIN'], [], testSocId);
			expect(role).toBe('SUPER_ADMIN');
		});

		it('resolves explicit SOC_ADMIN assignment for Platform ADMIN', () => {
			const role = getEffectiveSocietyRole(
				['ADMIN'],
				[{ society_id: testSocId, role: 'SOC_ADMIN' }],
				testSocId
			);
			expect(role).toBe('SOC_ADMIN');
		});

		it('resolves ADMIN if platform admin has no explicit society assignment', () => {
			const role = getEffectiveSocietyRole(['ADMIN'], [], testSocId);
			expect(role).toBe('ADMIN');
		});

		it('resolves SOC_ADMIN for society administrator', () => {
			const role = getEffectiveSocietyRole(
				[],
				[{ society_id: testSocId, role: 'SOC_ADMIN' }],
				testSocId
			);
			expect(role).toBe('SOC_ADMIN');
		});

		it('resolves GATEKEEPER for on-duty guard', () => {
			const role = getEffectiveSocietyRole(
				[],
				[{ society_id: testSocId, role: 'GATEKEEPER' }],
				testSocId
			);
			expect(role).toBe('GATEKEEPER');
		});

		it('resolves RESIDENT for linked resident or member', () => {
			const role = getEffectiveSocietyRole(
				[],
				[{ society_id: testSocId, role: 'RESIDENT' }],
				testSocId
			);
			expect(role).toBe('RESIDENT');
		});

		it('defaults to RESIDENT when unassigned', () => {
			const role = getEffectiveSocietyRole([], [], testSocId);
			expect(role).toBe('RESIDENT');
		});
	});

	describe('Resident Navigation Experience', () => {
		const residentItems = getSocietyNavItems('RESIDENT', testSocId);

		it('presents exactly 5 focused, resident-centric destinations', () => {
			expect(residentItems).toHaveLength(5);
		});

		it('presents destinations in the required logical order', () => {
			const ids = residentItems.map((item) => item.id);
			expect(ids).toEqual([
				'overview',
				'announcements',
				'maintenance',
				'documents',
				'resident-hub'
			]);
		});

		it('points resident maintenance to #resident-maintenance rather than admin console', () => {
			const maint = residentItems.find((item) => item.id === 'maintenance');
			expect(maint).toBeDefined();
			expect(maint?.href).toBe(`/app/societies/${testSocId}#resident-maintenance`);
			expect(maint?.label).toBe('Maintenance & Dues');
		});

		it('points resident documents to #resident-documents rather than admin console', () => {
			const docs = residentItems.find((item) => item.id === 'documents');
			expect(docs).toBeDefined();
			expect(docs?.href).toBe(`/app/societies/${testSocId}#resident-documents`);
			expect(docs?.label).toBe('Society Documents');
		});

		it('never exposes administrative modules to residents', () => {
			const ids = residentItems.map((item) => item.id);
			expect(ids).not.toContain('members');
			expect(ids).not.toContain('committee');
			expect(ids).not.toContain('structure');
			expect(ids).not.toContain('gatekeeping');
			expect(ids).not.toContain('audit');
			expect(ids).not.toContain('reports');
		});

		it('never shows upcoming or planned settings placeholders to residents', () => {
			const ids = residentItems.map((item) => item.id);
			expect(ids).not.toContain('settings');
		});
	});

	describe('Gatekeeper Navigation Experience', () => {
		const gkItems = getSocietyNavItems('GATEKEEPER', testSocId);

		it('presents exactly 3 perimeter-focused destinations', () => {
			expect(gkItems).toHaveLength(3);
			const ids = gkItems.map((item) => item.id);
			expect(ids).toEqual(['overview', 'gatekeeping', 'announcements']);
		});

		it('points gatekeeping to perimeter console (#gatekeeping)', () => {
			const gk = gkItems.find((item) => item.id === 'gatekeeping');
			expect(gk?.href).toBe(`/app/societies/${testSocId}#gatekeeping`);
			expect(gk?.label).toBe('Perimeter & Gatekeeping');
		});

		it('never exposes resident dues, private documents, or member rosters to gatekeepers', () => {
			const ids = gkItems.map((item) => item.id);
			expect(ids).not.toContain('maintenance');
			expect(ids).not.toContain('documents');
			expect(ids).not.toContain('members');
			expect(ids).not.toContain('audit');
		});
	});

	describe('Society Administrator Experience', () => {
		const adminRoles: SocietyRole[] = ['SOC_ADMIN', 'SUPER_ADMIN', 'ADMIN'];

		adminRoles.forEach((role) => {
			it(`provides complete operational suite for role: ${role}`, () => {
				const items = getSocietyNavItems(role, testSocId);
				const ids = items.map((item) => item.id);

				expect(ids).toContain('overview');
				expect(ids).toContain('reports');
				expect(ids).toContain('members');
				expect(ids).toContain('committee');
				expect(ids).toContain('structure');
				expect(ids).toContain('gatekeeping');
				expect(ids).toContain('announcements');
				expect(ids).toContain('maintenance');
				expect(ids).toContain('documents');
				expect(ids).toContain('audit');
				expect(ids).toContain('resident-hub');

				// Never contains fake Settings
				expect(ids).not.toContain('settings');
			});

			it(`points maintenance to admin management console for role: ${role}`, () => {
				const items = getSocietyNavItems(role, testSocId);
				const maint = items.find((item) => item.id === 'maintenance');
				expect(maint?.href).toBe(`/app/societies/${testSocId}#maintenance`);
				expect(maint?.label).toBe('Maintenance & Billing');
			});
		});
	});

	describe('URL Hash Normalization (normalizeActiveSection)', () => {
		it('normalizes empty hash to overview', () => {
			expect(normalizeActiveSection('')).toBe('overview');
			expect(normalizeActiveSection(undefined)).toBe('overview');
		});

		it('maps resident-maintenance hash to maintenance', () => {
			expect(normalizeActiveSection('#resident-maintenance')).toBe('maintenance');
		});

		it('maps resident-documents hash to documents', () => {
			expect(normalizeActiveSection('#resident-documents')).toBe('documents');
		});

		it('maps announcements-admin hash to announcements', () => {
			expect(normalizeActiveSection('#announcements-admin')).toBe('announcements');
		});

		it('preserves clean section ids', () => {
			expect(normalizeActiveSection('#members')).toBe('members');
			expect(normalizeActiveSection('#audit')).toBe('audit');
			expect(normalizeActiveSection('#resident-hub')).toBe('resident-hub');
		});
	});

	describe('Empty/Invalid Society Safety', () => {
		it('returns empty list if societyId is missing', () => {
			expect(getSocietyNavItems('RESIDENT', '')).toEqual([]);
			expect(getGroupedSocietyNavItems('RESIDENT', '')).toEqual([]);
		});
	});

	describe('Grouped Modular Society Navigation (ADR 040)', () => {
		it('groups resident destinations into OVERVIEW and OPERATIONS with modular routes', () => {
			const sections = getGroupedSocietyNavItems('RESIDENT', testSocId);
			const groupNames = sections.map((s) => s.group);

			expect(groupNames).toEqual(['OVERVIEW', 'OPERATIONS']);
			expect(groupNames).not.toContain('PEOPLE');
			expect(groupNames).not.toContain('INSIGHTS');

			const overview = sections.find((s) => s.group === 'OVERVIEW');
			expect(overview?.items).toHaveLength(1);
			expect(overview?.items[0].href).toBe(`/app/societies/${testSocId}`);

			const ops = sections.find((s) => s.group === 'OPERATIONS');
			expect(ops?.items.map((i) => i.id)).toEqual([
				'announcements',
				'maintenance',
				'resident-hub',
				'documents'
			]);

			expect(ops?.items.find((i) => i.id === 'announcements')?.href).toBe(
				`/app/societies/${testSocId}/announcements`
			);
			expect(ops?.items.find((i) => i.id === 'maintenance')?.href).toBe(
				`/app/societies/${testSocId}/maintenance`
			);
			expect(ops?.items.find((i) => i.id === 'resident-hub')?.href).toBe(
				`/app/societies/${testSocId}/resident-hub`
			);
			expect(ops?.items.find((i) => i.id === 'documents')?.href).toBe(
				`/app/societies/${testSocId}/documents`
			);
		});

		it('groups gatekeeper destinations into focused OPERATIONS without admin OVERVIEW', () => {
			const sections = getGroupedSocietyNavItems('GATEKEEPER', testSocId);
			const groupNames = sections.map((s) => s.group);

			expect(groupNames).toEqual(['OPERATIONS']);
			expect(groupNames).not.toContain('OVERVIEW');
			expect(groupNames).not.toContain('PEOPLE');
			expect(groupNames).not.toContain('INSIGHTS');

			const ops = sections.find((s) => s.group === 'OPERATIONS');
			expect(ops?.items.map((i) => i.id)).toEqual(['gate', 'announcements']);
			expect(ops?.items.find((i) => i.id === 'gate')?.href).toBe(
				`/app/societies/${testSocId}/gate`
			);
			expect(ops?.items.find((i) => i.id === 'gate')?.label).toBe('Gate Console');
		});

		it('provides all 4 groups with modular routes for administrators', () => {
			const adminRoles: SocietyRole[] = ['SOC_ADMIN', 'SUPER_ADMIN', 'ADMIN'];

			adminRoles.forEach((role) => {
				const sections = getGroupedSocietyNavItems(role, testSocId);
				const groupNames = sections.map((s) => s.group);

				expect(groupNames).toEqual(['OVERVIEW', 'PEOPLE', 'OPERATIONS', 'INSIGHTS']);

				const people = sections.find((s) => s.group === 'PEOPLE');
				expect(people?.items.map((i) => i.id)).toEqual(['members', 'committee']);
				expect(people?.items.find((i) => i.id === 'members')?.href).toBe(
					`/app/societies/${testSocId}/members`
				);
				expect(people?.items.find((i) => i.id === 'committee')?.href).toBe(
					`/app/societies/${testSocId}/committee`
				);

				const insights = sections.find((s) => s.group === 'INSIGHTS');
				expect(insights?.items.map((i) => i.id)).toEqual(['reports', 'audit']);
				expect(insights?.items.find((i) => i.id === 'reports')?.href).toBe(
					`/app/societies/${testSocId}/reports`
				);
				expect(insights?.items.find((i) => i.id === 'audit')?.href).toBe(
					`/app/societies/${testSocId}/audit`
				);
			});
		});
	});

	describe('Route-Aware Active Module Matching (isSocietyModuleActive)', () => {
		it('activates overview ONLY when at exact society root path', () => {
			const overviewHref = `/app/societies/${testSocId}`;

			expect(isSocietyModuleActive(overviewHref, `/app/societies/${testSocId}`, testSocId)).toBe(
				true
			);
			expect(isSocietyModuleActive(overviewHref, `/app/societies/${testSocId}/`, testSocId)).toBe(
				true
			);
			expect(
				isSocietyModuleActive(overviewHref, `/app/societies/${testSocId}/announcements`, testSocId)
			).toBe(false);
			expect(
				isSocietyModuleActive(overviewHref, `/app/societies/${testSocId}/maintenance`, testSocId)
			).toBe(false);
		});

		it('activates announcements exclusively for announcements paths', () => {
			const annHref = `/app/societies/${testSocId}/announcements`;

			expect(
				isSocietyModuleActive(annHref, `/app/societies/${testSocId}/announcements`, testSocId)
			).toBe(true);
			expect(
				isSocietyModuleActive(annHref, `/app/societies/${testSocId}/announcements/new`, testSocId)
			).toBe(true);
			expect(isSocietyModuleActive(annHref, `/app/societies/${testSocId}`, testSocId)).toBe(false);
			expect(
				isSocietyModuleActive(annHref, `/app/societies/${testSocId}/maintenance`, testSocId)
			).toBe(false);
			expect(
				isSocietyModuleActive(annHref, `/app/societies/${testSocId}/resident-hub`, testSocId)
			).toBe(false);
		});

		it('activates resident-hub exclusively for resident-hub paths', () => {
			const rhHref = `/app/societies/${testSocId}/resident-hub`;

			expect(
				isSocietyModuleActive(rhHref, `/app/societies/${testSocId}/resident-hub`, testSocId)
			).toBe(true);
			expect(
				isSocietyModuleActive(rhHref, `/app/societies/${testSocId}/announcements`, testSocId)
			).toBe(false);
			expect(isSocietyModuleActive(rhHref, `/app/societies/${testSocId}`, testSocId)).toBe(false);
		});
	});
});
