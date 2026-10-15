import { api } from './client';

export interface PlatformDashboardSummary {
	societies: {
		total: number;
		active: number;
		pending: number;
		suspended: number;
	};
	new_societies_30d: number;
	total_registered_units: number;
	members: {
		total_active_accounts: number;
	};
	financial_velocity_30d: {
		amount_paise: number;
		formatted_inr: string;
	};
	platform_audit_events_24h: number;
}

export interface SocietyDashboardSummary {
	society_id: string;
	occupancy: {
		total_units: number;
		occupied_units: number;
		vacant_units: number;
		occupancy_rate_percent: number;
		owner_occupied_units: number;
		tenant_occupied_units: number;
	};
	members: {
		total_active_members: number;
	};
	gate: {
		visitors_on_premises: number;
		today_entries: number;
		today_exits: number;
		pending_approvals: number;
	};
	finance: {
		current_period_id?: string;
		current_period_name: string;
		billed_paise: number;
		collected_paise: number;
		collection_efficiency_percent: number;
		overdue_arrears_paise: number;
		payments_collected_mtd_paise: number;
	};
	communication: {
		active_announcements_count: number;
	};
}

export interface GateDashboardSummary {
	society_id: string;
	visitors_on_premises: number;
	today_entries: number;
	today_exits: number;
	pending_approvals: number;
	active_guards_on_duty: number;
}

export const dashboardApi = {
	getPlatformDashboard(): Promise<PlatformDashboardSummary> {
		return api.get<PlatformDashboardSummary>('/platform/dashboard/summary');
	},

	getSocietyDashboard(societyId: string): Promise<SocietyDashboardSummary> {
		return api.get<SocietyDashboardSummary>(`/societies/${societyId}/dashboard/summary`);
	},

	getGateDashboard(societyId: string): Promise<GateDashboardSummary> {
		return api.get<GateDashboardSummary>(`/societies/${societyId}/gate/dashboard`);
	}
};
