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
			<div class="flex items-center gap-1.5 text-xs text-neutral-400">
				<Filter class="h-3.5 w-3.5" />
				<span>Filters:</span>
			</div>
			<select
				bind:value={roleFilter}
				onchange={handleFilterChange}
				aria-label="Filter administrators by role"
				class="rounded-full border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
			>
				<option value="">All Roles</option>
				<option value="SUPER_ADMIN">SUPER_ADMIN</option>
				<option value="ADMIN">ADMIN</option>
			</select>

			<select
				bind:value={statusFilter}
				onchange={handleFilterChange}
				aria-label="Filter administrators by status"
				class="rounded-full border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
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
			class="flex flex-col items-center justify-center rounded-2xl border border-neutral-200 bg-neutral-50/50 p-12 text-center"
		>
			<Loader2 class="h-8 w-8 animate-spin text-neutral-800" />
			<p class="mt-3 text-xs text-neutral-500">Loading administrators...</p>
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
			<Button variant="outline" size="sm" onclick={() => loadAdmins()}>Retry</Button>
		</div>
	{:else if admins.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-2xl border border-neutral-200 bg-neutral-50/50 p-12 text-center"
		>
			<div
				class="flex h-12 w-12 items-center justify-center rounded-full bg-neutral-200 text-neutral-600"
			>
				<Users class="h-6 w-6" />
			</div>
			<h3 class="mt-3 text-sm font-semibold text-neutral-900">No Administrators Found</h3>
			<p class="mt-1 max-w-sm text-xs text-neutral-500">
				No administrators match the selected filter criteria. Try adjusting your filters or
				provision a new administrator.
			</p>
			<Button variant="primary" size="sm" class="mt-4" onclick={onOpenCreate}>
				Add Administrator
			</Button>
		</div>
	{:else}
		<!-- Table Layout (Desktop & Tablet) -->
		<div class="overflow-x-auto rounded-2xl border border-neutral-200 bg-white shadow-xs">
			<table class="w-full text-left text-xs text-neutral-700">
				<thead
					class="border-b border-neutral-200 bg-neutral-50/80 text-[11px] font-semibold tracking-wider text-neutral-500 uppercase"
				>
					<tr>
						<th scope="col" class="px-4 py-3.5">Administrator</th>
						<th scope="col" class="px-4 py-3.5">Role</th>
						<th scope="col" class="px-4 py-3.5">Status</th>
						<th scope="col" class="px-4 py-3.5">Created</th>
						<th scope="col" class="px-4 py-3.5 text-right">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-neutral-100 font-sans">
					{#each admins as item (item.id)}
						{@const isSelf = auth.user?.id === item.user_id}
						{@const isSuperAdmin = item.role === 'SUPER_ADMIN'}
						<tr class="transition-colors hover:bg-neutral-50/60">
							<td class="px-4 py-3 font-mono font-medium text-neutral-900">
								<div class="flex items-center gap-2">
									{#if isSuperAdmin}
										<ShieldCheck class="h-4 w-4 text-neutral-950" />
									{:else}
										<Shield class="h-4 w-4 text-neutral-400" />
									{/if}
									<span>{item.phone}</span>
									{#if isSelf}
										<span
											class="rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] font-semibold text-neutral-800"
										>
											You
										</span>
									{/if}
								</div>
							</td>
							<td class="px-4 py-3">
								{#if item.role === 'SUPER_ADMIN'}
									<Badge variant="default" class="px-2 py-0 text-[10px]">SUPER_ADMIN</Badge>
								{:else}
									<Badge variant="outline" class="text-neutral-700">ADMIN</Badge>
								{/if}
							</td>
							<td class="px-4 py-3">
								{#if item.status === 'ACTIVE'}
									<Badge variant="success">ACTIVE</Badge>
								{:else}
									<Badge variant="destructive">DISABLED</Badge>
								{/if}
							</td>
							<td class="px-4 py-3 text-neutral-500">
								{formatDate(item.created_at)}
							</td>
							<td class="px-4 py-3 text-right">
								{#if isSuperAdmin}
									<span class="text-[11px] text-neutral-400 italic">Protected</span>
								{:else if isSelf}
									<span class="text-[11px] text-neutral-400 italic">Self</span>
								{:else if item.status === 'ACTIVE'}
									<Button
										variant="outline"
										size="sm"
										class="border-rose-200 py-1 text-xs text-rose-700 hover:bg-rose-50"
										onclick={() => openStatusDialog(item, 'DISABLED')}
									>
										Disable
									</Button>
								{:else}
									<Button
										variant="outline"
										size="sm"
										class="border-emerald-200 py-1 text-xs text-emerald-800 hover:bg-emerald-50"
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
			class="flex flex-col gap-3 pt-2 text-xs text-neutral-500 sm:flex-row sm:items-center sm:justify-between"
		>
			<div>
				Showing <span class="font-medium text-neutral-900">{offset + 1}</span> to
				<span class="font-medium text-neutral-900">{Math.min(offset + limit, total)}</span> of
				<span class="font-medium text-neutral-900">{total}</span> administrators
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
				<span class="px-2 font-mono text-xs text-neutral-700">
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
