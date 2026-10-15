<script lang="ts">
	import { onMount } from 'svelte';
	import { dashboardApi, type PlatformDashboardSummary } from '$lib/api/dashboard';
	import { ApiClientError } from '$lib/api/client';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import {
		Building2,
		Users,
		Receipt,
		ScrollText,
		RefreshCw,
		TrendingUp,
		AlertCircle
	} from '@lucide/svelte';

	let summary = $state<PlatformDashboardSummary | null>(null);
	let isLoading = $state<boolean>(true);
	let isRefreshing = $state<boolean>(false);
	let error = $state<string | null>(null);

	export async function loadDashboard(silent = false) {
		if (silent) {
			isRefreshing = true;
		} else {
			isLoading = true;
		}
		error = null;

		try {
			const res = await dashboardApi.getPlatformDashboard();
			summary = res;
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message;
			} else {
				error = 'Failed to load platform dashboard metrics.';
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
	<!-- Top Bar -->
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2">
				<div class="flex h-8 w-8 items-center justify-center rounded-xl bg-neutral-900 text-white">
					<TrendingUp class="h-4 w-4" />
				</div>
				<h2 class="text-lg font-bold tracking-tight text-neutral-950">
					Platform Ecosystem Health & Intelligence
				</h2>
				<Badge variant="outline" class="border-neutral-300 font-mono text-[10px] text-neutral-700">
					Super Admin Console
				</Badge>
			</div>
			<p class="mt-1 text-xs text-neutral-500">
				Platform-wide society adoption, unit census, global member identities, and financial
				velocity.
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
			<span>{isRefreshing ? 'Refreshing…' : 'Refresh Ecosystem'}</span>
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
		<div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
			{#each [0, 1] as idx (idx)}
				<div
					class="h-48 animate-pulse rounded-2xl border border-neutral-200 bg-neutral-100/70 p-6"
				></div>
			{/each}
		</div>
	{:else if error}
		<!-- Error State -->
		<div class="rounded-2xl border border-rose-200 bg-rose-50/60 p-6">
			<div class="flex items-start gap-3">
				<AlertCircle class="mt-0.5 h-5 w-5 shrink-0 text-rose-600" />
				<div>
					<h3 class="text-sm font-semibold text-rose-950">Failed to aggregate platform metrics</h3>
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
		<!-- 1. Top KPI Strip -->
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<!-- Total Registered Societies -->
			<div class="rounded-2xl border border-neutral-200 bg-white/90 p-5 shadow-xs">
				<div class="flex items-center justify-between">
					<span class="text-xs font-semibold tracking-wider text-neutral-500 uppercase"
						>Societies</span
					>
					<Building2 class="h-4 w-4 text-neutral-400" />
				</div>
				<div class="mt-3 flex items-baseline gap-2">
					<span class="font-mono text-3xl font-extrabold tracking-tight text-neutral-950">
						{summary.societies.total}
					</span>
					<span class="text-xs font-medium text-emerald-600"
						>+{summary.new_societies_30d} new (30d)</span
					>
				</div>
				<p class="mt-1 text-[11px] text-neutral-500">
					<span class="font-mono font-medium text-neutral-900">{summary.societies.active}</span>
					active ·
					<span class="font-mono font-medium text-neutral-900">{summary.societies.pending}</span> pending
					onboarding
				</p>
			</div>

			<!-- Registered Flat Units -->
			<div class="rounded-2xl border border-neutral-200 bg-white/90 p-5 shadow-xs">
				<div class="flex items-center justify-between">
					<span class="text-xs font-semibold tracking-wider text-neutral-500 uppercase"
						>Registered Units</span
					>
					<Building2 class="h-4 w-4 text-neutral-400" />
				</div>
				<div class="mt-3 flex items-baseline gap-2">
					<span class="font-mono text-3xl font-extrabold tracking-tight text-neutral-950">
						{summary.total_registered_units}
					</span>
					<span class="text-xs text-neutral-500">flats</span>
				</div>
				<p class="mt-1 text-[11px] text-neutral-500">Ecosystem-wide unit census</p>
			</div>

			<!-- Financial Velocity 30d -->
			<div class="rounded-2xl border border-neutral-200 bg-white/90 p-5 shadow-xs">
				<div class="flex items-center justify-between">
					<span class="text-xs font-semibold tracking-wider text-neutral-500 uppercase"
						>Volume (30 Days)</span
					>
					<Receipt class="h-4 w-4 text-neutral-400" />
				</div>
				<div class="mt-3 flex items-baseline gap-2">
					<span class="font-mono text-2xl font-extrabold tracking-tight text-neutral-950">
						{summary.financial_velocity_30d.formatted_inr}
					</span>
				</div>
				<p class="mt-1 text-[11px] text-neutral-500">Recorded maintenance collections</p>
			</div>

			<!-- Audit Log Activity 24h -->
			<div class="rounded-2xl border border-neutral-200 bg-white/90 p-5 shadow-xs">
				<div class="flex items-center justify-between">
					<span class="text-xs font-semibold tracking-wider text-neutral-500 uppercase"
						>Audit Events (24h)</span
					>
					<ScrollText class="h-4 w-4 text-neutral-400" />
				</div>
				<div class="mt-3 flex items-baseline gap-2">
					<span class="font-mono text-3xl font-extrabold tracking-tight text-neutral-950">
						{summary.platform_audit_events_24h}
					</span>
					<span class="text-xs text-neutral-500">events</span>
				</div>
				<p class="mt-1 text-[11px] text-neutral-500">Immutable ledger throughput</p>
			</div>
		</div>

		<!-- 2. Society Lifecycle & Account Breakdown -->
		<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
			<!-- Society Lifecycle Distribution -->
			<div class="rounded-2xl border border-neutral-200 bg-white/90 p-6 shadow-xs">
				<div class="flex items-center justify-between border-b border-neutral-100 pb-3">
					<div class="flex items-center gap-2">
						<Building2 class="h-4 w-4 text-neutral-700" />
						<h3 class="text-sm font-bold text-neutral-950">Societies Lifecycle Pipeline</h3>
					</div>
					<Badge variant="outline" class="font-mono text-[10px]">Multi-Tenant</Badge>
				</div>

				<div class="mt-4 space-y-3 font-mono text-xs">
					<div class="flex items-center justify-between border-b border-neutral-100 py-1.5">
						<div class="flex items-center gap-2 font-sans">
							<span class="h-2 w-2 rounded-full bg-emerald-500"></span>
							<span class="text-neutral-700">Active Societies</span>
						</div>
						<span class="font-bold text-neutral-950">{summary.societies.active}</span>
					</div>
					<div class="flex items-center justify-between border-b border-neutral-100 py-1.5">
						<div class="flex items-center gap-2 font-sans">
							<span class="h-2 w-2 rounded-full bg-amber-500"></span>
							<span class="text-neutral-700">Pending Review / Activation</span>
						</div>
						<span class="font-bold text-amber-600">{summary.societies.pending}</span>
					</div>
					<div class="flex items-center justify-between border-b border-neutral-100 py-1.5">
						<div class="flex items-center gap-2 font-sans">
							<span class="h-2 w-2 rounded-full bg-rose-500"></span>
							<span class="text-neutral-700">Suspended / Administrative Hold</span>
						</div>
						<span class="font-bold text-rose-600">{summary.societies.suspended}</span>
					</div>
				</div>
			</div>

			<!-- User Accounts & Access -->
			<div class="rounded-2xl border border-neutral-200 bg-white/90 p-6 shadow-xs">
				<div class="flex items-center justify-between border-b border-neutral-100 pb-3">
					<div class="flex items-center gap-2">
						<Users class="h-4 w-4 text-neutral-700" />
						<h3 class="text-sm font-bold text-neutral-950">Identity & Global Accounts</h3>
					</div>
					<Badge variant="outline" class="font-mono text-[10px]">Active Directory</Badge>
				</div>

				<div class="mt-4 space-y-3 font-mono text-xs">
					<div class="flex items-center justify-between border-b border-neutral-100 py-1.5">
						<span class="font-sans text-neutral-700">Global Active Member Accounts</span>
						<span class="font-bold text-neutral-950">{summary.members.total_active_accounts}</span>
					</div>
					<div class="flex items-center justify-between border-b border-neutral-100 py-1.5">
						<span class="font-sans text-neutral-700">Onboarded Units Across Platform</span>
						<span class="font-bold text-neutral-950">{summary.total_registered_units}</span>
					</div>
					<div class="flex items-center justify-between border-b border-neutral-100 py-1.5">
						<span class="font-sans text-neutral-700">New Registrations (Last 30 Days)</span>
						<span class="font-bold text-emerald-700">+{summary.new_societies_30d}</span>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
