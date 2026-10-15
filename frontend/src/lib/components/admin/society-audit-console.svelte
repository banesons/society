<script lang="ts">
	import { onMount } from 'svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { societyApi } from '$lib/api/society';
	import type { AuditLogItem } from '$lib/api/platform';
	import {
		ScrollText,
		AlertCircle,
		Loader2,
		ChevronLeft,
		ChevronRight,
		RefreshCw,
		Search,
		Eye,
		ShieldCheck,
		X,
		Copy,
		Check
	} from '@lucide/svelte';

	let {
		societyId,
		societyName = 'Society'
	}: {
		societyId: string;
		societyName?: string;
	} = $props();

	let logs = $state<AuditLogItem[]>([]);
	let total = $state(0);
	let limit = $state(20);
	let page = $state(1);
	let actionFilter = $state('');
	let searchQuery = $state('');
	let startDate = $state('');
	let endDate = $state('');
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	// Event detail modal state
	let selectedEvent = $state<AuditLogItem | null>(null);
	let isDetailOpen = $state(false);
	let copiedId = $state(false);

	// Cache isolation tracker: track previous society ID to clear on change
	let currentSocietyId = $state(societyId);

	$effect(() => {
		if (societyId !== currentSocietyId) {
			currentSocietyId = societyId;
			logs = [];
			page = 1;
			actionFilter = '';
			searchQuery = '';
			startDate = '';
			endDate = '';
			loadLogs();
		}
	});

	const totalPages = $derived(Math.max(1, Math.ceil(total / limit)));

	const societyActionOptions = [
		{ value: '', label: 'All Society Operations' },
		{ value: 'MEMBER_REGISTERED', label: 'Member Registered' },
		{ value: 'MEMBER_DEPARTED', label: 'Member Departed' },
		{ value: 'UNIT_RELATIONSHIP_CREATED', label: 'Unit Relationship Created' },
		{ value: 'UNIT_RELATIONSHIP_ENDED', label: 'Unit Relationship Ended' },
		{ value: 'COMMITTEE_APPOINTMENT_CREATED', label: 'Committee Appointment Created' },
		{ value: 'COMMITTEE_APPOINTMENT_ENDED', label: 'Committee Appointment Ended' },
		{ value: 'GATE_DUTY_STARTED', label: 'Gate Duty Started' },
		{ value: 'GATE_DUTY_ENDED', label: 'Gate Duty Ended' },
		{ value: 'VISIT_REGISTERED', label: 'Visit Registered' },
		{ value: 'WALK_IN_REGISTERED', label: 'Walk-in Visit Registered' },
		{ value: 'VISITOR_CHECKED_IN', label: 'Visitor Checked In' },
		{ value: 'VISITOR_CHECKED_OUT', label: 'Visitor Checked Out' },
		{ value: 'ANNOUNCEMENT_CREATED', label: 'Announcement Created' },
		{ value: 'ANNOUNCEMENT_PUBLISHED', label: 'Announcement Published' },
		{ value: 'ANNOUNCEMENT_ARCHIVED', label: 'Announcement Archived' },
		{ value: 'BILLING_PERIOD_CREATED', label: 'Billing Period Created' },
		{ value: 'BILLS_GENERATED', label: 'Bills Generated' },
		{ value: 'MAINTENANCE_PAYMENT_RECORDED', label: 'Payment Recorded' },
		{ value: 'MAINTENANCE_PAYMENT_REVERSED', label: 'Payment Reversed' },
		{ value: 'DOCUMENT_CREATED', label: 'Document Created' },
		{ value: 'DOCUMENT_VERSION_UPLOADED', label: 'Document Version Uploaded' },
		{ value: 'DOCUMENT_ARCHIVED', label: 'Document Archived' },
		{ value: 'DOCUMENT_RESTORED', label: 'Document Restored' }
	];

	async function loadLogs() {
		if (!currentSocietyId) return;
		isLoading = true;
		error = null;
		try {
			const res = await societyApi.listSocietyAuditLogs(currentSocietyId, {
				page,
				limit,
				action: actionFilter || undefined,
				search: searchQuery.trim() || undefined,
				start_date: startDate || undefined,
				end_date: endDate || undefined
			});
			logs = res.items;
			total = res.total;
		} catch {
			error =
				'Failed to load society audit history. Please check permissions or network connection.';
		} finally {
			isLoading = false;
		}
	}

	function handleNextPage() {
		if (page < totalPages) {
			page += 1;
			loadLogs();
		}
	}

	function handlePrevPage() {
		if (page > 1) {
			page -= 1;
			loadLogs();
		}
	}

	function handleFilterChange() {
		page = 1;
		loadLogs();
	}

	function resetFilters() {
		actionFilter = '';
		searchQuery = '';
		startDate = '';
		endDate = '';
		page = 1;
		loadLogs();
	}

	function formatDate(iso: string) {
		try {
			return new Date(iso).toLocaleDateString('en-IN', {
				year: 'numeric',
				month: 'short',
				day: 'numeric',
				hour: '2-digit',
				minute: '2-digit',
				second: '2-digit'
			});
		} catch {
			return iso;
		}
	}

	function formatUTC(iso: string) {
		try {
			return new Date(iso).toISOString();
		} catch {
			return iso;
		}
	}

	function getActionVariant(
		action: string
	): 'default' | 'success' | 'warning' | 'destructive' | 'outline' {
		if (
			action.includes('DEPARTED') ||
			action.includes('ENDED') ||
			action.includes('ARCHIVED') ||
			action.includes('REVERSED') ||
			action.includes('VOID')
		) {
			return 'destructive';
		}
		if (
			action.includes('CREATED') ||
			action.includes('REGISTERED') ||
			action.includes('PUBLISHED') ||
			action.includes('RECORDED') ||
			action.includes('RESTORED')
		) {
			return 'success';
		}
		return 'outline';
	}

	function inspectEvent(item: AuditLogItem) {
		selectedEvent = item;
		isDetailOpen = true;
	}

	async function copyEventId() {
		if (!selectedEvent) return;
		try {
			await navigator.clipboard.writeText(selectedEvent.id);
			copiedId = true;
			setTimeout(() => {
				copiedId = false;
			}, 2000);
		} catch {
			// fallback
		}
	}

	onMount(() => {
		loadLogs();
	});
</script>

<div class="space-y-4">
	<!-- Header Banner -->
	<div class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<h2 class="text-base font-semibold tracking-tight text-neutral-900">
				Audit & Activity History
			</h2>
			<p class="text-xs text-neutral-500">
				Tenant-isolated, immutable chronological activity trail for {societyName}.
			</p>
		</div>
		<div class="flex items-center gap-2">
			<span
				class="inline-flex items-center gap-1 rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-1 text-[11px] font-medium text-emerald-700"
			>
				<ShieldCheck class="h-3 w-3" />
				Append-Only Ledger
			</span>
		</div>
	</div>

	<!-- Filter & Search Toolbar -->
	<div class="rounded-2xl border border-neutral-200 bg-white p-4 shadow-xs">
		<div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
			<!-- Left controls -->
			<div class="flex flex-wrap items-center gap-2.5">
				<!-- Search -->
				<div class="relative w-full sm:w-60">
					<Search
						class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-neutral-400"
					/>
					<input
						type="text"
						bind:value={searchQuery}
						onkeydown={(e) => e.key === 'Enter' && handleFilterChange()}
						placeholder="Search activity, actor, target..."
						class="w-full rounded-full border border-neutral-300 bg-white py-1.5 pr-3 pl-9 text-xs text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-950 focus:outline-none"
					/>
				</div>

				<!-- Action dropdown -->
				<select
					bind:value={actionFilter}
					onchange={handleFilterChange}
					aria-label="Filter society activity by action"
					class="rounded-full border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
				>
					{#each societyActionOptions as opt (opt.value)}
						<option value={opt.value}>{opt.label}</option>
					{/each}
				</select>

				<!-- Date Range -->
				<div class="flex items-center gap-1.5">
					<input
						type="date"
						bind:value={startDate}
						onchange={handleFilterChange}
						aria-label="From date"
						class="rounded-full border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-700 focus:border-neutral-950 focus:outline-none"
					/>
					<span class="text-xs text-neutral-400">to</span>
					<input
						type="date"
						bind:value={endDate}
						onchange={handleFilterChange}
						aria-label="To date"
						class="rounded-full border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-700 focus:border-neutral-950 focus:outline-none"
					/>
				</div>

				<!-- Clear -->
				{#if actionFilter || searchQuery || startDate || endDate}
					<Button variant="ghost" size="sm" onclick={resetFilters}>Reset</Button>
				{/if}
			</div>

			<!-- Right controls -->
			<div class="flex items-center gap-3">
				<div class="flex items-center gap-1.5 text-xs text-neutral-500">
					<span>Rows:</span>
					<select
						bind:value={limit}
						onchange={handleFilterChange}
						aria-label="Rows per page"
						class="rounded-full border border-neutral-300 bg-white px-2.5 py-1 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
					>
						<option value={10}>10</option>
						<option value={20}>20</option>
						<option value={50}>50</option>
					</select>
				</div>

				<Button
					variant="ghost"
					size="sm"
					onclick={() => loadLogs()}
					disabled={isLoading}
					title="Refresh audit logs"
					aria-label="Refresh audit logs"
				>
					<RefreshCw class="h-3.5 w-3.5 {isLoading ? 'animate-spin' : ''}" />
				</Button>
			</div>
		</div>
	</div>

	<!-- Content State -->
	{#if isLoading && logs.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-2xl border border-neutral-200 bg-neutral-50/50 p-12 text-center"
		>
			<Loader2 class="h-8 w-8 animate-spin text-neutral-800" />
			<p class="mt-3 text-xs text-neutral-500">Loading society activity history...</p>
		</div>
	{:else if error}
		<div
			role="alert"
			class="flex items-center justify-between rounded-2xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-700"
		>
			<div class="flex items-center gap-2">
				<AlertCircle class="h-4 w-4 shrink-0 text-rose-500" />
				<span>{error}</span>
			</div>
			<Button variant="outline" size="sm" onclick={() => loadLogs()}>Retry</Button>
		</div>
	{:else if logs.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-2xl border border-neutral-200 bg-neutral-50/50 p-12 text-center"
		>
			<div
				class="flex h-12 w-12 items-center justify-center rounded-full bg-neutral-200 text-neutral-600"
			>
				<ScrollText class="h-6 w-6" />
			</div>
			<h3 class="mt-3 text-sm font-semibold text-neutral-900">No Society Activity Found</h3>
			<p class="mt-1 max-w-sm text-xs text-neutral-500">
				No operations match the selected criteria for this society.
			</p>
			{#if actionFilter || searchQuery || startDate || endDate}
				<div class="mt-4">
					<Button variant="outline" size="sm" onclick={resetFilters}>Clear Filters</Button>
				</div>
			{/if}
		</div>
	{:else}
		<!-- Activity Table -->
		<div class="overflow-x-auto rounded-2xl border border-neutral-200 bg-white shadow-xs">
			<table class="w-full text-left text-xs text-neutral-700">
				<thead
					class="border-b border-neutral-200 bg-neutral-50/80 text-[11px] font-semibold tracking-wider text-neutral-500 uppercase"
				>
					<tr>
						<th scope="col" class="px-4 py-3.5">Timestamp</th>
						<th scope="col" class="px-4 py-3.5">Action</th>
						<th scope="col" class="px-4 py-3.5">Actor</th>
						<th scope="col" class="px-4 py-3.5">Subject / Details</th>
						<th scope="col" class="px-4 py-3.5">Client IP</th>
						<th scope="col" class="px-4 py-3.5 text-right">Inspection</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-neutral-100 font-sans">
					{#each logs as item (item.id)}
						<tr class="transition-colors hover:bg-neutral-50/60">
							<td class="px-4 py-3 font-mono text-[11px] whitespace-nowrap text-neutral-600">
								{formatDate(item.created_at)}
							</td>
							<td class="px-4 py-3 whitespace-nowrap">
								<Badge variant={getActionVariant(item.action)}>
									{item.action_label || item.action}
								</Badge>
							</td>
							<td class="px-4 py-3 whitespace-nowrap">
								<div class="font-medium text-neutral-900">{item.actor_name || 'System'}</div>
								{#if item.actor_phone}
									<div class="font-mono text-[10px] text-neutral-400">{item.actor_phone}</div>
								{/if}
							</td>
							<td class="px-4 py-3 whitespace-nowrap">
								{#if item.target_display}
									<span class="font-medium text-neutral-800">{item.target_display}</span>
								{:else if item.target_user_id}
									<span class="font-mono text-[10px] text-neutral-500"
										>{item.target_user_id.slice(0, 8)}...</span
									>
								{:else}
									<span class="text-neutral-400">—</span>
								{/if}
							</td>
							<td class="px-4 py-3 font-mono text-[11px] whitespace-nowrap text-neutral-500">
								{item.client_ip || '—'}
							</td>
							<td class="px-4 py-3 text-right whitespace-nowrap">
								<Button
									variant="ghost"
									size="sm"
									onclick={() => inspectEvent(item)}
									title="Inspect event details"
								>
									<Eye class="h-3.5 w-3.5 text-neutral-600" />
									<span class="ml-1 text-xs">Details</span>
								</Button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>

			<!-- Pagination Footer -->
			<div
				class="flex flex-col gap-3 border-t border-neutral-200 px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
			>
				<div class="text-xs text-neutral-500">
					Showing <span class="font-semibold text-neutral-900">{(page - 1) * limit + 1}</span>
					to <span class="font-semibold text-neutral-900">{Math.min(page * limit, total)}</span>
					of <span class="font-semibold text-neutral-900">{total}</span> society events
				</div>

				<div class="flex items-center gap-2">
					<Button
						variant="outline"
						size="sm"
						onclick={handlePrevPage}
						disabled={page <= 1 || isLoading}
					>
						<ChevronLeft class="mr-1 h-3.5 w-3.5" />
						Previous
					</Button>
					<span class="text-xs font-medium text-neutral-600">
						Page {page} of {totalPages}
					</span>
					<Button
						variant="outline"
						size="sm"
						onclick={handleNextPage}
						disabled={page >= totalPages || isLoading}
					>
						Next
						<ChevronRight class="ml-1 h-3.5 w-3.5" />
					</Button>
				</div>
			</div>
		</div>
	{/if}
</div>

<!-- Event Detail Modal -->
{#if isDetailOpen && selectedEvent}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-xs"
		role="dialog"
		aria-modal="true"
		aria-labelledby="soc-audit-detail-title"
	>
		<div
			class="animate-in fade-in zoom-in-95 relative flex max-h-[90vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-xl duration-150"
		>
			<!-- Header -->
			<div
				class="flex items-center justify-between border-b border-neutral-200 bg-neutral-50/50 px-6 py-4"
			>
				<div class="flex items-center gap-2.5">
					<div
						class="flex h-9 w-9 items-center justify-center rounded-full bg-neutral-100 text-neutral-800"
					>
						<ShieldCheck class="h-5 w-5" />
					</div>
					<div>
						<h2 id="soc-audit-detail-title" class="text-sm font-semibold text-neutral-900">
							{selectedEvent.action_label || selectedEvent.action}
						</h2>
						<div class="mt-0.5 flex items-center gap-2">
							<span class="font-mono text-[10px] text-neutral-500">{selectedEvent.id}</span>
							<button
								type="button"
								onclick={copyEventId}
								class="text-neutral-400 hover:text-neutral-700"
								title="Copy event UUID"
							>
								{#if copiedId}
									<Check class="h-3 w-3 text-emerald-600" />
								{:else}
									<Copy class="h-3 w-3" />
								{/if}
							</button>
						</div>
					</div>
				</div>
				<button
					type="button"
					onclick={() => (isDetailOpen = false)}
					class="rounded-full p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700"
					aria-label="Close modal"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<!-- Body -->
			<div class="space-y-5 overflow-y-auto px-6 py-5 text-xs text-neutral-700">
				<!-- Context Summary Cards -->
				<div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
					<div class="rounded-xl border border-neutral-200 bg-neutral-50/60 p-3">
						<div class="text-[10px] font-semibold tracking-wider text-neutral-400 uppercase">
							Timestamp
						</div>
						<div class="mt-1 font-mono text-[11px] text-neutral-900">
							{formatDate(selectedEvent.created_at)}
						</div>
						<div class="font-mono text-[9px] text-neutral-400">
							{formatUTC(selectedEvent.created_at)}
						</div>
					</div>

					<div class="rounded-xl border border-neutral-200 bg-neutral-50/60 p-3">
						<div class="text-[10px] font-semibold tracking-wider text-neutral-400 uppercase">
							Actor
						</div>
						<div class="mt-1 font-medium text-neutral-900">
							{selectedEvent.actor_name || 'System'}
						</div>
						{#if selectedEvent.actor_phone}
							<div class="font-mono text-[10px] text-neutral-500">{selectedEvent.actor_phone}</div>
						{/if}
					</div>

					<div class="rounded-xl border border-neutral-200 bg-neutral-50/60 p-3">
						<div class="text-[10px] font-semibold tracking-wider text-neutral-400 uppercase">
							Society
						</div>
						<div class="mt-1 font-medium text-neutral-900">
							{selectedEvent.society_name || societyName}
						</div>
						{#if selectedEvent.society_code}
							<div class="font-mono text-[10px] text-neutral-500">{selectedEvent.society_code}</div>
						{/if}
					</div>
				</div>

				<!-- Target Reference -->
				{#if selectedEvent.target_display || selectedEvent.target_user_id}
					<div class="rounded-xl border border-neutral-200 p-3">
						<div class="text-[10px] font-semibold tracking-wider text-neutral-400 uppercase">
							Subject / Target
						</div>
						<div class="mt-1 flex items-center justify-between">
							<span class="font-medium text-neutral-900"
								>{selectedEvent.target_display || 'User Record'}</span
							>
							{#if selectedEvent.target_user_id}
								<span class="font-mono text-[10px] text-neutral-500"
									>{selectedEvent.target_user_id}</span
								>
							{/if}
						</div>
					</div>
				{/if}

				<!-- Network Context -->
				<div class="space-y-1 rounded-xl border border-neutral-200 p-3">
					<div class="text-[10px] font-semibold tracking-wider text-neutral-400 uppercase">
						Audit Origin
					</div>
					<div class="flex items-center justify-between text-neutral-600">
						<span>Client IP:</span>
						<span class="font-mono text-[11px] text-neutral-900"
							>{selectedEvent.client_ip || 'Unavailable'}</span
						>
					</div>
				</div>

				<!-- Sanitized Metadata -->
				<div class="space-y-2">
					<div class="flex items-center justify-between">
						<span class="text-[10px] font-semibold tracking-wider text-neutral-400 uppercase"
							>Sanitized Event Metadata</span
						>
						<span class="rounded bg-neutral-100 px-1.5 py-0.5 font-mono text-[9px] text-neutral-500"
							>JSONB Details</span
						>
					</div>
					<div
						class="max-h-64 overflow-x-auto rounded-xl border border-neutral-200 bg-neutral-950 p-4 font-mono text-xs text-neutral-200"
					>
						<pre class="text-[11px] leading-relaxed">{JSON.stringify(
								selectedEvent.details,
								null,
								2
							)}</pre>
					</div>
				</div>
			</div>

			<!-- Footer -->
			<div class="flex justify-end border-t border-neutral-200 bg-neutral-50 px-6 py-3">
				<Button variant="outline" size="sm" onclick={() => (isDetailOpen = false)}>Close</Button>
			</div>
		</div>
	</div>
{/if}
