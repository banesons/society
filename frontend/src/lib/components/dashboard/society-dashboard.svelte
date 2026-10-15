<script lang="ts">
	import { onMount } from 'svelte';
	import { dashboardApi, type SocietyDashboardSummary } from '$lib/api/dashboard';
	import { ApiClientError } from '$lib/api/client';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import {
		Building2,
		Users,
		DoorClosed,
		Receipt,
		RefreshCw,
		TrendingUp,
		AlertCircle,
		ArrowUpRight,
		ArrowDownRight
	} from '@lucide/svelte';

	interface Props {
		societyId: string;
		societyName?: string;
	}

	let { societyId, societyName }: Props = $props();

	let summary = $state<SocietyDashboardSummary | null>(null);
	let isLoading = $state<boolean>(true);
	let isRefreshing = $state<boolean>(false);
	let error = $state<string | null>(null);

	export function formatPaise(paise: number): string {
		const rupees = paise / 100;
		return new Intl.NumberFormat('en-IN', {
			style: 'currency',
			currency: 'INR',
			minimumFractionDigits: 2,
			maximumFractionDigits: 2
		}).format(rupees);
	}

	export async function loadDashboard(silent = false) {
		if (!societyId) return;
		if (silent) {
			isRefreshing = true;
		} else {
			isLoading = true;
		}
		error = null;

		try {
			const res = await dashboardApi.getSocietyDashboard(societyId);
			summary = res;
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message;
			} else {
				error = 'Failed to load society dashboard metrics.';
			}
		} finally {
			isLoading = false;
			isRefreshing = false;
		}
	}

	onMount(() => {
		loadDashboard();
	});
</script>

<div class="space-y-6">
	<!-- Section Header -->
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2">
				<div class="flex h-8 w-8 items-center justify-center rounded-xl bg-neutral-900 text-white">
					<TrendingUp class="h-4 w-4" />
				</div>
				<h2 class="text-lg font-bold tracking-tight text-neutral-950">
					Society Operations & Performance Hub
				</h2>
				<Badge variant="outline" class="border-neutral-300 font-mono text-[10px] text-neutral-700">
					S16 Realtime
				</Badge>
			</div>
			<p class="mt-1 text-xs text-neutral-500">
				{#if societyName}
					<span class="font-medium text-neutral-800">{societyName}</span> ·
				{/if}
				Read-only operational intelligence, unit occupancy, perimeter flow, and financial recovery scorecards.
			</p>
		</div>

		<Button
			variant="outline"
			size="sm"
			disabled={isLoading || isRefreshing}
			onclick={() => loadDashboard(true)}
			class="self-start border-neutral-300 text-neutral-700 hover:bg-neutral-100 sm:self-auto"
		>
			<RefreshCw class="mr-1.5 h-3.5 w-3.5 {isRefreshing ? 'animate-spin' : ''}" />
			<span>{isRefreshing ? 'Refreshing…' : 'Refresh Metrics'}</span>
		</Button>
	</div>

	<!-- Loading Skeleton -->
	{#if isLoading && !summary}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
			{#each [0, 1, 2, 3] as idx (idx)}
				<div
					class="h-32 animate-pulse rounded-2xl border border-neutral-200 bg-neutral-100/70 p-5"
				></div>
			{/each}
		</div>
		<div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
			{#each [0, 1, 2] as idx (idx)}
				<div
					class="h-56 animate-pulse rounded-2xl border border-neutral-200 bg-neutral-100/70 p-6"
				></div>
			{/each}
		</div>
	{:else if error}
		<!-- Error State -->
		<div class="rounded-2xl border border-rose-200 bg-rose-50/60 p-6">
			<div class="flex items-start gap-3">
				<AlertCircle class="mt-0.5 h-5 w-5 shrink-0 text-rose-600" />
				<div>
					<h3 class="text-sm font-semibold text-rose-950">Unable to load dashboard intelligence</h3>
					<p class="mt-1 text-xs text-rose-800">{error}</p>
					<Button
						variant="outline"
						size="sm"
						class="mt-3 border-rose-300 text-rose-900 hover:bg-rose-100"
						onclick={() => loadDashboard()}
					>
						Try Again
					</Button>
				</div>
			</div>
		</div>
	{:else if summary}
		<!-- 1. Top KPI Summary Strip -->
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<!-- Occupancy Rate -->
			<div class="rounded-2xl border border-neutral-200 bg-white/90 p-5 shadow-xs">
				<div class="flex items-center justify-between">
					<span class="text-xs font-semibold tracking-wider text-neutral-500 uppercase"
						>Occupancy Rate</span
					>
					<Building2 class="h-4 w-4 text-neutral-400" />
				</div>
				<div class="mt-3 flex items-baseline gap-2">
					<span class="font-mono text-3xl font-extrabold tracking-tight text-neutral-950">
						{summary.occupancy.occupancy_rate_percent.toFixed(1)}%
					</span>
				</div>
				<p class="mt-1 text-[11px] text-neutral-500">
					<span class="font-mono font-medium text-neutral-900"
						>{summary.occupancy.occupied_units}</span
					>
					of
					<span class="font-mono font-medium text-neutral-900">{summary.occupancy.total_units}</span
					> units occupied
				</p>
			</div>

			<!-- Active Visitors on Premises -->
			<div class="rounded-2xl border border-neutral-200 bg-white/90 p-5 shadow-xs">
				<div class="flex items-center justify-between">
					<span class="text-xs font-semibold tracking-wider text-neutral-500 uppercase"
						>Inside Premises</span
					>
					<div class="flex items-center gap-1.5">
						<span class="relative flex h-2 w-2">
							<span
								class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"
							></span>
							<span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
						</span>
						<DoorClosed class="h-4 w-4 text-neutral-400" />
					</div>
				</div>
				<div class="mt-3 flex items-baseline gap-2">
					<span class="font-mono text-3xl font-extrabold tracking-tight text-neutral-950">
						{summary.gate.visitors_on_premises}
					</span>
					<span class="text-xs text-neutral-500">active guests</span>
				</div>
				<p class="mt-1 text-[11px] text-neutral-500">
					<span class="font-mono font-medium text-neutral-900">{summary.gate.today_entries}</span> check-ins
					today
				</p>
			</div>

			<!-- Collection Efficiency -->
			<div class="rounded-2xl border border-neutral-200 bg-white/90 p-5 shadow-xs">
				<div class="flex items-center justify-between">
					<span class="text-xs font-semibold tracking-wider text-neutral-500 uppercase"
						>Collections Rate</span
					>
					<Receipt class="h-4 w-4 text-neutral-400" />
				</div>
				<div class="mt-3 flex items-baseline gap-2">
					<span class="font-mono text-3xl font-extrabold tracking-tight text-neutral-950">
						{summary.finance.collection_efficiency_percent.toFixed(1)}%
					</span>
				</div>
				<p class="mt-1 text-[11px] text-neutral-500">
					{summary.finance.current_period_name || 'Current Billing Cycle'}
				</p>
			</div>

			<!-- Total Active Members -->
			<div class="rounded-2xl border border-neutral-200 bg-white/90 p-5 shadow-xs">
				<div class="flex items-center justify-between">
					<span class="text-xs font-semibold tracking-wider text-neutral-500 uppercase"
						>Community Size</span
					>
					<Users class="h-4 w-4 text-neutral-400" />
				</div>
				<div class="mt-3 flex items-baseline gap-2">
					<span class="font-mono text-3xl font-extrabold tracking-tight text-neutral-950">
						{summary.members.total_active_members}
					</span>
					<span class="text-xs text-neutral-500">members</span>
				</div>
				<p class="mt-1 text-[11px] text-neutral-500">
					<span class="font-mono font-medium text-neutral-900"
						>{summary.communication.active_announcements_count}</span
					> active broadcasts
				</p>
			</div>
		</div>

		<!-- 2. Detailed Performance Scorecards -->
		<div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
			<!-- Occupancy & Distribution -->
			<div
				class="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white/90 p-6 shadow-xs"
			>
				<div>
					<div class="flex items-center justify-between border-b border-neutral-100 pb-3">
						<div class="flex items-center gap-2">
							<Building2 class="h-4 w-4 text-neutral-700" />
							<h3 class="text-sm font-bold text-neutral-950">Occupancy Distribution</h3>
						</div>
						<Badge variant="outline" class="font-mono text-[10px]">Flat Registry</Badge>
					</div>

					{#if summary.occupancy.total_units === 0}
						<div class="py-8 text-center text-xs text-neutral-500">
							No units configured yet. Use the Society Formation Wizard to generate units.
						</div>
					{:else}
						<!-- Progress distribution bar -->
						<div class="mt-5 space-y-2">
							<div class="flex justify-between text-xs">
								<span class="text-neutral-500">Occupancy Progress</span>
								<span class="font-mono font-semibold text-neutral-900"
									>{summary.occupancy.occupancy_rate_percent}%</span
								>
							</div>
							<div class="h-2.5 w-full overflow-hidden rounded-full bg-neutral-100">
								<div
									class="h-full rounded-full bg-neutral-900 transition-all duration-500"
									style="width: {Math.min(
										100,
										Math.max(0, summary.occupancy.occupancy_rate_percent)
									)}%"
								></div>
							</div>
						</div>

						<div class="mt-6 space-y-3 font-mono text-xs">
							<div class="flex items-center justify-between border-b border-neutral-100 py-1.5">
								<span class="font-sans text-neutral-500">Owner-Occupied</span>
								<span class="font-bold text-neutral-950"
									>{summary.occupancy.owner_occupied_units} units</span
								>
							</div>
							<div class="flex items-center justify-between border-b border-neutral-100 py-1.5">
								<span class="font-sans text-neutral-500">Tenant-Occupied</span>
								<span class="font-bold text-neutral-950"
									>{summary.occupancy.tenant_occupied_units} units</span
								>
							</div>
							<div class="flex items-center justify-between border-b border-neutral-100 py-1.5">
								<span class="font-sans text-neutral-500">Vacant / Unassigned</span>
								<span class="font-bold text-neutral-600"
									>{summary.occupancy.vacant_units} units</span
								>
							</div>
						</div>
					{/if}
				</div>

				<div class="mt-6 border-t border-neutral-100 pt-3 text-[11px] text-neutral-500">
					Occupancy is derived strictly from verified active member resident relationships.
				</div>
			</div>

			<!-- Gate & Footfall Realtime -->
			<div
				class="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white/90 p-6 shadow-xs"
			>
				<div>
					<div class="flex items-center justify-between border-b border-neutral-100 pb-3">
						<div class="flex items-center gap-2">
							<DoorClosed class="h-4 w-4 text-neutral-700" />
							<h3 class="text-sm font-bold text-neutral-950">Gate & Perimeter Activity</h3>
						</div>
						<Badge variant="outline" class="font-mono text-[10px]">Today (IST)</Badge>
					</div>

					<div class="mt-5 grid grid-cols-2 gap-3">
						<div class="rounded-xl border border-neutral-100 bg-neutral-50/70 p-3.5">
							<div class="flex items-center gap-1.5 text-[11px] text-neutral-500">
								<ArrowDownRight class="h-3.5 w-3.5 text-emerald-600" />
								<span>Entries</span>
							</div>
							<div class="mt-1 font-mono text-2xl font-bold text-neutral-950">
								{summary.gate.today_entries}
							</div>
						</div>
						<div class="rounded-xl border border-neutral-100 bg-neutral-50/70 p-3.5">
							<div class="flex items-center gap-1.5 text-[11px] text-neutral-500">
								<ArrowUpRight class="h-3.5 w-3.5 text-blue-600" />
								<span>Exits</span>
							</div>
							<div class="mt-1 font-mono text-2xl font-bold text-neutral-950">
								{summary.gate.today_exits}
							</div>
						</div>
					</div>

					<div class="mt-4 space-y-2.5 font-mono text-xs">
						<div class="flex items-center justify-between border-b border-neutral-100 py-1.5">
							<span class="font-sans text-neutral-500">Pending Resident Approvals</span>
							<span
								class="font-bold {summary.gate.pending_approvals > 0
									? 'text-amber-600'
									: 'text-neutral-900'}"
							>
								{summary.gate.pending_approvals}
							</span>
						</div>
						<div class="flex items-center justify-between border-b border-neutral-100 py-1.5">
							<span class="font-sans text-neutral-500">Currently Inside Perimeter</span>
							<span class="font-bold text-emerald-700">
								{summary.gate.visitors_on_premises} visitors
							</span>
						</div>
					</div>
				</div>

				<div class="mt-6 border-t border-neutral-100 pt-3 text-[11px] text-neutral-500">
					Synchronized live across entry/exit turnstiles and security tablets.
				</div>
			</div>

			<!-- Financial Health Scorecard -->
			<div
				class="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white/90 p-6 shadow-xs"
			>
				<div>
					<div class="flex items-center justify-between border-b border-neutral-100 pb-3">
						<div class="flex items-center gap-2">
							<Receipt class="h-4 w-4 text-neutral-700" />
							<h3 class="text-sm font-bold text-neutral-950">Maintenance & Dues Scorecard</h3>
						</div>
						<Badge variant="outline" class="font-mono text-[10px]">Paise-Exact</Badge>
					</div>

					<div class="mt-4 space-y-3 font-mono text-xs">
						<div class="flex items-center justify-between border-b border-neutral-100 py-1">
							<span class="font-sans text-neutral-500">Current Cycle Billed</span>
							<span class="font-bold text-neutral-950"
								>{formatPaise(summary.finance.billed_paise)}</span
							>
						</div>
						<div class="flex items-center justify-between border-b border-neutral-100 py-1">
							<span class="font-sans text-neutral-500">Current Cycle Collected</span>
							<span class="font-bold text-emerald-700"
								>{formatPaise(summary.finance.collected_paise)}</span
							>
						</div>
						<div class="flex items-center justify-between border-b border-neutral-100 py-1">
							<span class="font-sans text-neutral-500">Overdue Arrears</span>
							<span
								class="font-bold {summary.finance.overdue_arrears_paise > 0
									? 'text-rose-600'
									: 'text-neutral-900'}"
							>
								{formatPaise(summary.finance.overdue_arrears_paise)}
							</span>
						</div>
						<div class="flex items-center justify-between border-b border-neutral-100 py-1">
							<span class="font-sans text-neutral-500">Collections (Month-to-Date)</span>
							<span class="font-bold text-neutral-900"
								>{formatPaise(summary.finance.payments_collected_mtd_paise)}</span
							>
						</div>
					</div>
				</div>

				<div class="mt-6 border-t border-neutral-100 pt-3 text-[11px] text-neutral-500">
					All monetary sums computed in exact integer paise; reversals excluded.
				</div>
			</div>
		</div>
	{/if}
</div>
