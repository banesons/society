import { describe, it, expect, vi, beforeEach } from 'vitest';
import { dashboardApi } from './dashboard';
import { api } from './client';

describe('Dashboard API Client', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it('calls getPlatformDashboard endpoint', async () => {
		const mockSummary = {
			societies: { total: 10, active: 8, pending: 1, suspended: 1 },
			new_societies_30d: 2,
			total_registered_units: 450,
			members: { total_active_accounts: 380 },
			financial_velocity_30d: { amount_paise: 50000000, formatted_inr: '₹5,00,000.00' },
			platform_audit_events_24h: 120
		};

		const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockSummary);

		const result = await dashboardApi.getPlatformDashboard();

		expect(getSpy).toHaveBeenCalledWith('/platform/dashboard/summary');
		expect(result).toEqual(mockSummary);
	});

	it('calls getSocietyDashboard endpoint with societyId', async () => {
		const societyId = '893c528f-7ee3-463d-82d2-8d76a5ebca57';
		const mockSocietySummary = {
			society_id: societyId,
			occupancy: {
				total_units: 100,
				occupied_units: 80,
				vacant_units: 20,
				occupancy_rate_percent: 80.0,
				owner_occupied_units: 60,
				tenant_occupied_units: 20
			},
			members: { total_active_members: 150 },
			gate: {
				visitors_on_premises: 3,
				today_entries: 25,
				today_exits: 22,
				pending_approvals: 1
			},
			finance: {
				current_period_name: 'October 2026',
				billed_paise: 25000000,
				collected_paise: 20000000,
				collection_efficiency_percent: 80.0,
				overdue_arrears_paise: 5000000,
				payments_collected_mtd_paise: 20000000
			},
			communication: {
				active_announcements_count: 3
			}
		};

		const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockSocietySummary);

		const result = await dashboardApi.getSocietyDashboard(societyId);

		expect(getSpy).toHaveBeenCalledWith(`/societies/${societyId}/dashboard/summary`);
		expect(result).toEqual(mockSocietySummary);
	});

	it('calls getGateDashboard endpoint with societyId', async () => {
		const societyId = '893c528f-7ee3-463d-82d2-8d76a5ebca57';
		const mockGateSummary = {
			society_id: societyId,
			visitors_on_premises: 5,
			today_entries: 40,
			today_exits: 35,
			pending_approvals: 2,
			active_guards_on_duty: 3
		};

		const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockGateSummary);

		const result = await dashboardApi.getGateDashboard(societyId);

		expect(getSpy).toHaveBeenCalledWith(`/societies/${societyId}/gate/dashboard`);
		expect(result).toEqual(mockGateSummary);
	});
});
