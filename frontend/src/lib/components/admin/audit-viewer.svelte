<script lang="ts">
	import { onMount } from 'svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { platformApi, type AuditLogItem } from '$lib/api/platform';
	import {
		ScrollText,
		AlertCircle,
		Loader2,
		ChevronLeft,
		ChevronRight,
		RefreshCw,
		Filter
	} from '@lucide/svelte';

	let logs = $state<AuditLogItem[]>([]);
	let total = $state(0);
	let limit = $state(20);
	let offset = $state(0);
	let actionFilter = $state('');
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	const currentPage = $derived(Math.floor(offset / limit) + 1);
	const totalPages = $derived(Math.max(1, Math.ceil(total / limit)));

	async function loadLogs() {
		isLoading = true;
		error = null;
		try {
			const res = await platformApi.listAuditLogs({
				limit,
				offset,
				action: actionFilter || undefined
			});
			logs = res.items;
			total = res.total;
		} catch {
			error = 'Failed to load audit logs. Please check network connection.';
		} finally {
			isLoading = false;
		}
	}

	function handleNextPage() {
		if (offset + limit < total) {
			offset += limit;
			loadLogs();
		}
	}

	function handlePrevPage() {
		if (offset - limit >= 0) {
			offset -= limit;
			loadLogs();
		}
	}

	function handleFilterChange() {
		offset = 0;
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

	function getActionVariant(
		action: string
	): 'default' | 'success' | 'warning' | 'destructive' | 'outline' {
		switch (action) {
			case 'SUPER_ADMIN_BOOTSTRAPPED':
				return 'default';
			case 'ADMIN_CREATED':
			case 'ADMIN_REACTIVATED':
				return 'success';
			case 'ADMIN_DISABLED':
				return 'destructive';
			default:
				return 'outline';
		}
	}

	onMount(() => {
		loadLogs();
	});

	export function refresh() {
		loadLogs();
	}
</script>

<div class="space-y-4">
	<!-- Filter & Refresh Bar -->
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<div class="flex flex-wrap items-center gap-2">
			<div class="flex items-center gap-1.5 text-xs text-slate-400">
				<Filter class="h-3.5 w-3.5" />
				<span>Filter by Action:</span>
			</div>
			<select
				bind:value={actionFilter}
				onchange={handleFilterChange}
				aria-label="Filter audit logs by action"
				class="rounded-lg border border-slate-700 bg-slate-900 px-2.5 py-1.5 text-xs text-slate-200 focus:border-indigo-500 focus:outline-none"
			>
				<option value="">All Actions</option>
				<option value="SUPER_ADMIN_BOOTSTRAPPED">SUPER_ADMIN_BOOTSTRAPPED</option>
				<option value="ADMIN_CREATED">ADMIN_CREATED</option>
				<option value="ADMIN_DISABLED">ADMIN_DISABLED</option>
				<option value="ADMIN_REACTIVATED">ADMIN_REACTIVATED</option>
			</select>

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

		<div class="text-xs text-slate-400">Append-only immutable security ledger</div>
	</div>

	<!-- Content State -->
	{#if isLoading && logs.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-xl border border-slate-800 bg-slate-900/50 p-12 text-center"
		>
			<Loader2 class="h-8 w-8 animate-spin text-indigo-400" />
			<p class="mt-3 text-xs text-slate-400">Loading audit trail...</p>
		</div>
	{:else if error}
		<div
			role="alert"
			class="flex items-center justify-between rounded-xl border border-rose-500/30 bg-rose-500/10 p-4 text-xs text-rose-300"
		>
			<div class="flex items-center gap-2">
				<AlertCircle class="h-4 w-4 shrink-0 text-rose-400" />
				<span>{error}</span>
			</div>
			<Button variant="outline" size="sm" onclick={() => loadLogs()}>Retry</Button>
		</div>
	{:else if logs.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-xl border border-slate-800 bg-slate-900/50 p-12 text-center"
		>
			<div
				class="flex h-12 w-12 items-center justify-center rounded-full bg-slate-800 text-slate-400"
			>
				<ScrollText class="h-6 w-6" />
			</div>
			<h3 class="mt-3 text-sm font-semibold text-slate-200">No Audit Events Found</h3>
			<p class="mt-1 max-w-sm text-xs text-slate-400">
				No platform administration actions match the selected filter.
			</p>
		</div>
	{:else}
		<!-- Audit Table -->
		<div class="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/80 shadow-sm">
			<table class="w-full text-left text-xs text-slate-300">
				<thead
					class="border-b border-slate-800 bg-slate-950/60 text-[11px] font-semibold tracking-wider text-slate-400 uppercase"
				>
					<tr>
						<th scope="col" class="px-4 py-3">Timestamp</th>
						<th scope="col" class="px-4 py-3">Action</th>
						<th scope="col" class="px-4 py-3">Actor</th>
						<th scope="col" class="px-4 py-3">Details</th>
						<th scope="col" class="px-4 py-3">Client IP</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-800/60 font-sans">
					{#each logs as item (item.id)}
						<tr class="transition-colors hover:bg-slate-800/30">
							<td class="px-4 py-3 font-mono text-[11px] whitespace-nowrap text-slate-300">
								{formatDate(item.created_at)}
							</td>
							<td class="px-4 py-3 whitespace-nowrap">
								<Badge variant={getActionVariant(item.action)}>
									{item.action}
								</Badge>
							</td>
							<td class="px-4 py-3 font-mono text-[11px] whitespace-nowrap text-slate-400">
								{item.actor_id ? item.actor_id.slice(0, 8) + '...' : 'System'}
							</td>
							<td class="max-w-xs truncate px-4 py-3 text-slate-300">
								{#if item.details}
									<div class="flex flex-col gap-0.5">
										{#if item.details.phone}
											<span class="font-mono text-slate-200">{String(item.details.phone)}</span>
										{/if}
										{#if item.details.reason}
											<span class="text-[11px] text-slate-400 italic"
												>Reason: {String(item.details.reason)}</span
											>
										{/if}
										{#if item.details.previous_status && item.details.new_status}
											<span class="text-[11px] text-slate-400">
												{String(item.details.previous_status)} &rarr; {String(
													item.details.new_status
												)}
											</span>
										{/if}
									</div>
								{:else}
									<span class="text-slate-400">—</span>
								{/if}
							</td>
							<td class="px-4 py-3 font-mono text-[11px] whitespace-nowrap text-slate-400">
								{item.client_ip || '—'}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<!-- Pagination Footer -->
		<div
			class="flex flex-col gap-3 pt-2 text-xs text-slate-400 sm:flex-row sm:items-center sm:justify-between"
		>
			<div>
				Showing <span class="font-medium text-slate-200">{offset + 1}</span> to
				<span class="font-medium text-slate-200">{Math.min(offset + limit, total)}</span> of
				<span class="font-medium text-slate-200">{total}</span> audit records
			</div>
			<div class="flex items-center gap-2">
				<Button
					variant="outline"
					size="sm"
					disabled={offset === 0 || isLoading}
					onclick={handlePrevPage}
					aria-label="Previous audit page"
				>
					<ChevronLeft class="mr-1 h-3.5 w-3.5" />
					<span>Previous</span>
				</Button>
				<span class="px-2 font-mono text-xs">
					{currentPage} / {totalPages}
				</span>
				<Button
					variant="outline"
					size="sm"
					disabled={offset + limit >= total || isLoading}
					onclick={handleNextPage}
					aria-label="Next audit page"
				>
					<span>Next</span>
					<ChevronRight class="ml-1 h-3.5 w-3.5" />
				</Button>
			</div>
		</div>
	{/if}
</div>
