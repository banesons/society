<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { societyApi, type Visit } from '$lib/api/society';
	import { formatPhoneDisplay } from '$lib/phone';
	import { Search, Loader2, AlertCircle, ChevronLeft, ChevronRight, Eye } from '@lucide/svelte';

	let {
		societyId,
		onSelectVisit
	}: {
		societyId: string;
		onSelectVisit?: (visit: Visit) => void;
	} = $props();

	let visits = $state<Visit[]>([]);
	let totalCount = $state(0);
	let limit = $state(20);
	let offset = $state(0);

	let statusFilter = $state('ALL');
	let typeFilter = $state('ALL');
	let searchInput = $state('');
	let activeSearch = $state('');

	let isLoading = $state(true);
	let error = $state<string | null>(null);

	async function loadVisits() {
		if (!societyId) return;
		isLoading = true;
		error = null;

		try {
			const res = await societyApi.listVisits(societyId, {
				status: statusFilter,
				visit_type: typeFilter,
				search: activeSearch,
				limit,
				offset
			});
			visits = res.visits;
			totalCount = res.total;
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load visitor history';
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (societyId) {
			loadVisits();
		}
	});

	function handleSearch(e: Event) {
		e.preventDefault();
		activeSearch = searchInput.trim();
		offset = 0;
		loadVisits();
	}

	function handleFilterChange() {
		offset = 0;
		loadVisits();
	}

	function getStatusVariant(status: string): 'default' | 'success' | 'warning' | 'destructive' {
		switch (status) {
			case 'CHECKED_IN':
			case 'APPROVED':
				return 'success';
			case 'PENDING_APPROVAL':
				return 'warning';
			case 'REJECTED':
			case 'CANCELLED':
			case 'EXPIRED':
				return 'destructive';
			default:
				return 'default';
		}
	}
</script>

<div class="space-y-4">
	<!-- Filter & Search Toolbar -->
	<div
		class="flex flex-col gap-3 rounded-2xl border border-neutral-200 bg-white/90 p-4 shadow-xs sm:flex-row sm:items-center sm:justify-between"
	>
		<form onsubmit={handleSearch} class="relative flex-1 sm:max-w-sm">
			<Search class="absolute top-2.5 left-3 h-4 w-4 text-neutral-400" />
			<input
				type="text"
				bind:value={searchInput}
				placeholder="Search by visitor, phone, unit, or plate..."
				class="w-full rounded-xl border border-neutral-200 bg-neutral-50/60 py-2 pr-3 pl-9 text-xs text-neutral-900 outline-none focus:border-neutral-900 focus:bg-white"
			/>
		</form>

		<div class="flex flex-wrap items-center gap-2 text-xs">
			<select
				bind:value={statusFilter}
				onchange={handleFilterChange}
				class="rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-800 outline-none focus:border-neutral-900"
			>
				<option value="ALL">All Statuses</option>
				<option value="CHECKED_IN">Currently Inside (Checked In)</option>
				<option value="CHECKED_OUT">Completed (Checked Out)</option>
				<option value="PENDING_APPROVAL">Pending Approval</option>
				<option value="APPROVED">Approved</option>
				<option value="REJECTED">Rejected</option>
				<option value="CANCELLED">Cancelled</option>
			</select>

			<select
				bind:value={typeFilter}
				onchange={handleFilterChange}
				class="rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-800 outline-none focus:border-neutral-900"
			>
				<option value="ALL">All Types</option>
				<option value="EXPECTED">Expected</option>
				<option value="WALK_IN">Walk-In</option>
			</select>
		</div>
	</div>

	{#if error}
		<div
			class="flex items-center gap-2 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
		>
			<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
			<span>{error}</span>
		</div>
	{/if}

	<!-- Visitor History Table -->
	<div class="overflow-hidden rounded-2xl border border-neutral-200 bg-white/90 shadow-xs">
		{#if isLoading}
			<div class="flex min-h-[220px] flex-col items-center justify-center text-neutral-500">
				<Loader2 class="h-6 w-6 animate-spin text-neutral-900" />
				<p class="mt-2 text-xs">Loading visitor history...</p>
			</div>
		{:else if visits.length === 0}
			<div
				class="flex min-h-[220px] flex-col items-center justify-center p-8 text-center text-xs text-neutral-500"
			>
				<p class="font-medium text-neutral-900">No visitor records found</p>
				<p class="mt-1">Try adjusting the status or search filter.</p>
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-left text-xs">
					<thead class="border-b border-neutral-200 bg-neutral-50/80 font-medium text-neutral-600">
						<tr>
							<th class="px-4 py-3">Visitor Name</th>
							<th class="px-4 py-3">Destination Unit</th>
							<th class="px-4 py-3">Phone</th>
							<th class="px-4 py-3">Purpose</th>
							<th class="px-4 py-3">Type</th>
							<th class="px-4 py-3">Status</th>
							<th class="px-4 py-3">Check-In</th>
							<th class="px-4 py-3">Check-Out</th>
							<th class="px-4 py-3 text-right">Actions</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-100 text-neutral-800">
						{#each visits as v (v.id)}
							<tr class="hover:bg-neutral-50/50">
								<td class="px-4 py-3">
									<div class="font-semibold text-neutral-950">{v.visitor_name}</div>
									{#if v.vehicle_number}
										<div class="font-mono text-[10px] text-neutral-500 uppercase">
											{v.vehicle_number}
										</div>
									{/if}
								</td>
								<td class="px-4 py-3 font-mono">
									{v.building_name} — {v.unit_identifier}
								</td>
								<td class="px-4 py-3 font-mono text-neutral-600">
									{formatPhoneDisplay(v.visitor_phone_masked)}
								</td>
								<td class="px-4 py-3">
									<Badge variant="outline" class="font-mono text-[10px]">{v.purpose}</Badge>
								</td>
								<td class="px-4 py-3 text-[11px] text-neutral-500">
									{v.visit_type}
								</td>
								<td class="px-4 py-3">
									<Badge variant={getStatusVariant(v.status)} class="text-[10px]">
										{v.status}
									</Badge>
								</td>
								<td class="px-4 py-3 font-mono text-[11px] text-neutral-600">
									{v.check_in_at
										? new Date(v.check_in_at).toLocaleTimeString([], {
												hour: '2-digit',
												minute: '2-digit'
											})
										: '—'}
								</td>
								<td class="px-4 py-3 font-mono text-[11px] text-neutral-600">
									{v.check_out_at
										? new Date(v.check_out_at).toLocaleTimeString([], {
												hour: '2-digit',
												minute: '2-digit'
											})
										: '—'}
								</td>
								<td class="px-4 py-3 text-right">
									{#if onSelectVisit}
										<button
											type="button"
											onclick={() => onSelectVisit(v)}
											class="inline-flex items-center gap-1 rounded-lg border border-neutral-200 px-2.5 py-1 text-[11px] font-medium text-neutral-700 hover:bg-neutral-100"
										>
											<Eye class="h-3 w-3" />
											<span>Details</span>
										</button>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<!-- Pagination Footer -->
			<div
				class="flex items-center justify-between border-t border-neutral-100 px-4 py-3 text-xs text-neutral-600"
			>
				<span
					>Showing {offset + 1}–{Math.min(offset + limit, totalCount)} of {totalCount} visits</span
				>
				<div class="flex items-center gap-2">
					<Button
						variant="outline"
						size="sm"
						disabled={offset === 0}
						onclick={() => {
							offset = Math.max(0, offset - limit);
							loadVisits();
						}}
					>
						<ChevronLeft class="h-3.5 w-3.5" />
					</Button>
					<Button
						variant="outline"
						size="sm"
						disabled={offset + limit >= totalCount}
						onclick={() => {
							offset += limit;
							loadVisits();
						}}
					>
						<ChevronRight class="h-3.5 w-3.5" />
					</Button>
				</div>
			</div>
		{/if}
	</div>
</div>
