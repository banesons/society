<script lang="ts">
	import { onMount } from 'svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import StatusDialog from '$lib/components/admin/status-dialog.svelte';
	import { platformApi, type AdminItem } from '$lib/api/platform';
	import { auth } from '$lib/auth.svelte';
	import {
		Shield,
		ShieldCheck,
		Users,
		AlertCircle,
		Loader2,
		ChevronLeft,
		ChevronRight,
		RefreshCw,
		Filter
	} from '@lucide/svelte';

	let { onOpenCreate }: { onOpenCreate: () => void } = $props();

	let admins = $state<AdminItem[]>([]);
	let total = $state(0);
	let limit = $state(10);
	let offset = $state(0);
	let roleFilter = $state('');
	let statusFilter = $state('');
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	// Status dialog state
	let statusDialogAdmin = $state<AdminItem | null>(null);
	let statusDialogTarget = $state<'ACTIVE' | 'DISABLED'>('DISABLED');
	let isStatusDialogOpen = $state(false);

	const currentPage = $derived(Math.floor(offset / limit) + 1);
	const totalPages = $derived(Math.max(1, Math.ceil(total / limit)));

	async function loadAdmins() {
		isLoading = true;
		error = null;
		try {
			const res = await platformApi.listAdmins({
				limit,
				offset,
				role: roleFilter || undefined,
				status: statusFilter || undefined
			});
			admins = res.items;
			total = res.total;
		} catch {
			error = 'Failed to load platform administrators. Please check network connection.';
		} finally {
			isLoading = false;
		}
	}

	function handleNextPage() {
		if (offset + limit < total) {
			offset += limit;
			loadAdmins();
		}
	}

	function handlePrevPage() {
		if (offset - limit >= 0) {
			offset -= limit;
			loadAdmins();
		}
	}

	function handleFilterChange() {
		offset = 0;
		loadAdmins();
	}

	function openStatusDialog(admin: AdminItem, newStatus: 'ACTIVE' | 'DISABLED') {
		statusDialogAdmin = admin;
		statusDialogTarget = newStatus;
		isStatusDialogOpen = true;
	}

	function formatDate(iso: string) {
		try {
			return new Date(iso).toLocaleDateString('en-IN', {
				year: 'numeric',
				month: 'short',
				day: 'numeric',
				hour: '2-digit',
				minute: '2-digit'
			});
		} catch {
			return iso;
		}
	}

	onMount(() => {
		loadAdmins();
	});

	export function refresh() {
		loadAdmins();
	}
</script>

<div class="space-y-4">
	<!-- Filter & Action Bar -->
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<div class="flex flex-wrap items-center gap-2">
			<div class="flex items-center gap-1.5 text-xs text-slate-400">
				<Filter class="h-3.5 w-3.5" />
				<span>Filters:</span>
			</div>
			<select
				bind:value={roleFilter}
				onchange={handleFilterChange}
				aria-label="Filter administrators by role"
				class="rounded-lg border border-slate-700 bg-slate-900 px-2.5 py-1.5 text-xs text-slate-200 focus:border-indigo-500 focus:outline-none"
			>
				<option value="">All Roles</option>
				<option value="SUPER_ADMIN">SUPER_ADMIN</option>
				<option value="ADMIN">ADMIN</option>
			</select>

			<select
				bind:value={statusFilter}
				onchange={handleFilterChange}
				aria-label="Filter administrators by status"
				class="rounded-lg border border-slate-700 bg-slate-900 px-2.5 py-1.5 text-xs text-slate-200 focus:border-indigo-500 focus:outline-none"
			>
				<option value="">All Statuses</option>
				<option value="ACTIVE">ACTIVE</option>
				<option value="DISABLED">DISABLED</option>
			</select>

			<Button
				variant="ghost"
				size="sm"
				onclick={() => loadAdmins()}
				disabled={isLoading}
				title="Refresh roster"
				aria-label="Refresh administrator roster"
			>
				<RefreshCw class="h-3.5 w-3.5 {isLoading ? 'animate-spin' : ''}" />
			</Button>
		</div>

		<div>
			<Button variant="primary" size="sm" onclick={onOpenCreate}>
				<Users class="mr-1.5 h-3.5 w-3.5" />
				<span>Add Administrator</span>
			</Button>
		</div>
	</div>

	<!-- Content State -->
	{#if isLoading && admins.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-xl border border-slate-800 bg-slate-900/50 p-12 text-center"
		>
			<Loader2 class="h-8 w-8 animate-spin text-indigo-400" />
			<p class="mt-3 text-xs text-slate-400">Loading administrators...</p>
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
			<Button variant="outline" size="sm" onclick={() => loadAdmins()}>Retry</Button>
		</div>
	{:else if admins.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-xl border border-slate-800 bg-slate-900/50 p-12 text-center"
		>
			<div
				class="flex h-12 w-12 items-center justify-center rounded-full bg-slate-800 text-slate-400"
			>
				<Users class="h-6 w-6" />
			</div>
			<h3 class="mt-3 text-sm font-semibold text-slate-200">No Administrators Found</h3>
			<p class="mt-1 max-w-sm text-xs text-slate-400">
				No administrators match the selected filter criteria. Try adjusting your filters or
				provision a new administrator.
			</p>
			<Button variant="primary" size="sm" class="mt-4" onclick={onOpenCreate}>
				Add Administrator
			</Button>
		</div>
	{:else}
		<!-- Table Layout (Desktop & Tablet) -->
		<div class="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/80 shadow-sm">
			<table class="w-full text-left text-xs text-slate-300">
				<thead
					class="border-b border-slate-800 bg-slate-950/60 text-[11px] font-semibold tracking-wider text-slate-400 uppercase"
				>
					<tr>
						<th scope="col" class="px-4 py-3">Administrator</th>
						<th scope="col" class="px-4 py-3">Role</th>
						<th scope="col" class="px-4 py-3">Status</th>
						<th scope="col" class="px-4 py-3">Created</th>
						<th scope="col" class="px-4 py-3 text-right">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-800/60 font-sans">
					{#each admins as item (item.id)}
						{@const isSelf = auth.user?.id === item.user_id}
						{@const isSuperAdmin = item.role === 'SUPER_ADMIN'}
						<tr class="transition-colors hover:bg-slate-800/30">
							<td class="px-4 py-3 font-mono font-medium text-slate-200">
								<div class="flex items-center gap-2">
									{#if isSuperAdmin}
										<ShieldCheck class="h-4 w-4 text-indigo-400" />
									{:else}
										<Shield class="h-4 w-4 text-slate-400" />
									{/if}
									<span>{item.phone}</span>
									{#if isSelf}
										<span
											class="rounded bg-indigo-500/20 px-1.5 py-0.5 text-[10px] font-semibold text-indigo-300"
										>
											You
										</span>
									{/if}
								</div>
							</td>
							<td class="px-4 py-3">
								{#if item.role === 'SUPER_ADMIN'}
									<Badge
										variant="default"
										class="border-indigo-500/40 bg-indigo-500/20 text-indigo-300"
									>
										SUPER_ADMIN
									</Badge>
								{:else}
									<Badge variant="outline" class="text-slate-300">ADMIN</Badge>
								{/if}
							</td>
							<td class="px-4 py-3">
								{#if item.status === 'ACTIVE'}
									<Badge variant="success">ACTIVE</Badge>
								{:else}
									<Badge variant="destructive">DISABLED</Badge>
								{/if}
							</td>
							<td class="px-4 py-3 text-slate-400">
								{formatDate(item.created_at)}
							</td>
							<td class="px-4 py-3 text-right">
								{#if isSuperAdmin}
									<span class="text-[11px] text-slate-400 italic">Protected</span>
								{:else if isSelf}
									<span class="text-[11px] text-slate-400 italic">Self</span>
								{:else if item.status === 'ACTIVE'}
									<Button
										variant="outline"
										size="sm"
										class="border-rose-500/30 py-1 text-xs text-rose-300 hover:bg-rose-500/10"
										onclick={() => openStatusDialog(item, 'DISABLED')}
									>
										Disable
									</Button>
								{:else}
									<Button
										variant="outline"
										size="sm"
										class="border-emerald-500/30 py-1 text-xs text-emerald-300 hover:bg-emerald-500/10"
										onclick={() => openStatusDialog(item, 'ACTIVE')}
									>
										Reactivate
									</Button>
								{/if}
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
				<span class="font-medium text-slate-200">{total}</span> administrators
			</div>
			<div class="flex items-center gap-2">
				<Button
					variant="outline"
					size="sm"
					disabled={offset === 0 || isLoading}
					onclick={handlePrevPage}
					aria-label="Previous page"
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
					aria-label="Next page"
				>
					<span>Next</span>
					<ChevronRight class="ml-1 h-3.5 w-3.5" />
				</Button>
			</div>
		</div>
	{/if}
</div>

<!-- Status Confirmation Dialog -->
<StatusDialog
	isOpen={isStatusDialogOpen}
	admin={statusDialogAdmin}
	targetStatus={statusDialogTarget}
	onClose={() => {
		isStatusDialogOpen = false;
		statusDialogAdmin = null;
	}}
	onSuccess={() => {
		loadAdmins();
	}}
/>
