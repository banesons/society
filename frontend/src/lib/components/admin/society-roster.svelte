<script lang="ts">
	import { onMount } from 'svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import SocietyDetailsDialog from '$lib/components/admin/society-details-dialog.svelte';
	import EditSocietyDialog from '$lib/components/admin/edit-society-dialog.svelte';
	import SocietyStatusDialog from '$lib/components/admin/society-status-dialog.svelte';
	import FormationWizard from '$lib/components/formation/formation-wizard.svelte';
	import { societyApi, type SocietyItem } from '$lib/api/society';
	import {
		Building2,
		Search,
		Filter,
		RefreshCw,
		Loader2,
		AlertCircle,
		ChevronLeft,
		ChevronRight,
		Copy,
		Check,
		Edit3,
		Eye,
		ShieldAlert,
		CheckCircle2,
		Layers
	} from '@lucide/svelte';

	let { onOpenRegister }: { onOpenRegister: () => void } = $props();

	let societies = $state<SocietyItem[]>([]);
	let total = $state(0);
	let limit = $state(10);
	let offset = $state(0);
	let statusFilter = $state('');
	let searchInput = $state('');
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	// Dialog states
	let selectedSociety = $state<SocietyItem | null>(null);
	let isDetailsOpen = $state(false);
	let isEditOpen = $state(false);
	let isStatusOpen = $state(false);
	let isFormationOpen = $state(false);
	let targetStatus = $state<'ACTIVE' | 'SUSPENDED'>('ACTIVE');
	let copiedCode = $state<string | null>(null);

	const currentPage = $derived(Math.floor(offset / limit) + 1);
	const totalPages = $derived(Math.max(1, Math.ceil(total / limit)));

	async function loadSocieties() {
		isLoading = true;
		error = null;
		try {
			const res = await societyApi.listSocieties({
				limit,
				offset,
				status: statusFilter || undefined,
				search: searchInput.trim() || undefined
			});
			societies = res.items;
			total = res.total;
		} catch {
			error = 'Failed to load societies. Please check network connection.';
		} finally {
			isLoading = false;
		}
	}

	function handleSearch(e: SubmitEvent) {
		e.preventDefault();
		offset = 0;
		loadSocieties();
	}

	function handleFilterChange() {
		offset = 0;
		loadSocieties();
	}

	function handleNextPage() {
		if (offset + limit < total) {
			offset += limit;
			loadSocieties();
		}
	}

	function handlePrevPage() {
		if (offset - limit >= 0) {
			offset -= limit;
			loadSocieties();
		}
	}

	function openDetails(soc: SocietyItem) {
		selectedSociety = soc;
		isDetailsOpen = true;
	}

	function openEdit(soc: SocietyItem) {
		selectedSociety = soc;
		isEditOpen = true;
	}

	function openFormation(soc: SocietyItem) {
		selectedSociety = soc;
		isFormationOpen = true;
	}

	function openStatus(soc: SocietyItem, target: 'ACTIVE' | 'SUSPENDED') {
		selectedSociety = soc;
		targetStatus = target;
		isStatusOpen = true;
	}

	async function copyCode(code: string) {
		try {
			await navigator.clipboard.writeText(code);
			copiedCode = code;
			setTimeout(() => {
				copiedCode = null;
			}, 2000);
		} catch {
			// fallback
		}
	}

	function formatDate(iso: string) {
		try {
			return new Date(iso).toLocaleDateString('en-IN', {
				year: 'numeric',
				month: 'short',
				day: 'numeric'
			});
		} catch {
			return iso;
		}
	}

	function getStatusVariant(status: string): 'default' | 'success' | 'warning' | 'destructive' {
		switch (status) {
			case 'ACTIVE':
				return 'success';
			case 'SUSPENDED':
				return 'destructive';
			default:
				return 'warning';
		}
	}

	onMount(() => {
		loadSocieties();
	});

	export function refresh() {
		loadSocieties();
	}
</script>

<div class="space-y-4">
	<!-- Filter & Action Bar -->
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<div class="flex flex-wrap items-center gap-2">
			<!-- Search Form -->
			<form onsubmit={handleSearch} class="relative flex items-center">
				<div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
					<Search class="h-3.5 w-3.5 text-neutral-400" />
				</div>
				<input
					type="text"
					bind:value={searchInput}
					placeholder="Search by name or code..."
					class="w-48 rounded-full border border-neutral-300 bg-white py-1.5 pr-3 pl-9 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-none sm:w-60"
				/>
			</form>

			<div class="flex items-center gap-1.5 text-xs text-neutral-400">
				<Filter class="h-3.5 w-3.5" />
			</div>
			<select
				bind:value={statusFilter}
				onchange={handleFilterChange}
				aria-label="Filter societies by status"
				class="rounded-full border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
			>
				<option value="">All Statuses</option>
				<option value="PENDING">PENDING</option>
				<option value="ACTIVE">ACTIVE</option>
				<option value="SUSPENDED">SUSPENDED</option>
			</select>

			<Button
				variant="ghost"
				size="sm"
				onclick={() => loadSocieties()}
				disabled={isLoading}
				title="Refresh societies"
				aria-label="Refresh society roster"
			>
				<RefreshCw class="h-3.5 w-3.5 {isLoading ? 'animate-spin' : ''}" />
			</Button>
		</div>

		<div>
			<Button variant="primary" size="sm" onclick={onOpenRegister}>
				<Building2 class="mr-1.5 h-3.5 w-3.5" />
				<span>Register Society</span>
			</Button>
		</div>
	</div>

	<!-- Content State -->
	{#if isLoading && societies.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-2xl border border-neutral-200 bg-neutral-50/50 p-12 text-center"
		>
			<Loader2 class="h-8 w-8 animate-spin text-neutral-800" />
			<p class="mt-3 text-xs text-neutral-500">Loading societies...</p>
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
			<Button variant="outline" size="sm" onclick={() => loadSocieties()}>Retry</Button>
		</div>
	{:else if societies.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-2xl border border-neutral-200 bg-neutral-50/50 p-12 text-center"
		>
			<div
				class="flex h-12 w-12 items-center justify-center rounded-full bg-neutral-200 text-neutral-600"
			>
				<Building2 class="h-6 w-6" />
			</div>
			<h3 class="mt-3 text-sm font-semibold text-neutral-900">No Societies Found</h3>
			<p class="mt-1 max-w-sm text-xs text-neutral-500">
				No housing societies match the filter criteria. Register a new society or adjust your search
				filters.
			</p>
			<Button variant="primary" size="sm" class="mt-4" onclick={onOpenRegister}>
				Register Society
			</Button>
		</div>
	{:else}
		<!-- Table Layout -->
		<div class="overflow-x-auto rounded-2xl border border-neutral-200 bg-white shadow-xs">
			<table class="w-full text-left text-xs text-neutral-700">
				<thead
					class="border-b border-neutral-200 bg-neutral-50/80 text-[11px] font-semibold tracking-wider text-neutral-500 uppercase"
				>
					<tr>
						<th scope="col" class="px-4 py-3.5">Society Code</th>
						<th scope="col" class="px-4 py-3.5">Society Name</th>
						<th scope="col" class="px-4 py-3.5">Location</th>
						<th scope="col" class="px-4 py-3.5">Status</th>
						<th scope="col" class="px-4 py-3.5">Created</th>
						<th scope="col" class="px-4 py-3.5 text-right">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-neutral-100 font-sans">
					{#each societies as item (item.id)}
						<tr class="transition-colors hover:bg-neutral-50/60">
							<!-- Society Code (Prominent & Copyable) -->
							<td class="px-4 py-3 font-mono font-bold whitespace-nowrap text-neutral-950">
								<div class="flex items-center gap-1.5">
									<span class="rounded bg-neutral-100 px-1.5 py-0.5 text-neutral-900"
										>{item.society_code}</span
									>
									<button
										type="button"
										onclick={() => copyCode(item.society_code)}
										title="Copy code"
										aria-label="Copy code {item.society_code}"
										class="cursor-pointer rounded p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-800"
									>
										{#if copiedCode === item.society_code}
											<Check class="h-3 w-3 text-emerald-600" />
										{:else}
											<Copy class="h-3 w-3" />
										{/if}
									</button>
								</div>
							</td>

							<!-- Name -->
							<td class="px-4 py-3 font-medium text-neutral-900">
								<a
									href={`/society/${item.id}`}
									class="inline-block max-w-xs truncate transition-colors hover:text-neutral-950 hover:underline"
									title="Open {item.name} Console"
								>
									{item.name}
								</a>
							</td>

							<!-- Location -->
							<td class="px-4 py-3 whitespace-nowrap text-neutral-600">
								{item.city}, {item.state}
							</td>

							<!-- Status Badge -->
							<td class="px-4 py-3 whitespace-nowrap">
								<Badge variant={getStatusVariant(item.status)}>
									{item.status}
								</Badge>
							</td>

							<!-- Created Date -->
							<td class="px-4 py-3 whitespace-nowrap text-neutral-500">
								{formatDate(item.created_at)}
							</td>

							<!-- Actions -->
							<td class="px-4 py-3 text-right whitespace-nowrap">
								<div class="flex items-center justify-end gap-1.5">
									{#if item.status === 'ACTIVE'}
										<Button
											variant="ghost"
											size="sm"
											class="p-1.5 text-neutral-700 hover:bg-neutral-100 hover:text-neutral-950"
											title="Formation & Structure"
											aria-label="Formation & Structure for {item.name}"
											onclick={() => openFormation(item)}
										>
											<Layers class="h-3.5 w-3.5" />
										</Button>
									{/if}

									<Button
										variant="ghost"
										size="sm"
										class="p-1.5 text-neutral-600 hover:bg-neutral-100 hover:text-neutral-950"
										title="View Details"
										aria-label="View Details for {item.name}"
										onclick={() => openDetails(item)}
									>
										<Eye class="h-3.5 w-3.5" />
									</Button>

									<Button
										variant="ghost"
										size="sm"
										class="p-1.5 text-neutral-600 hover:bg-neutral-100 hover:text-neutral-950"
										title="Edit Profile"
										aria-label="Edit Profile for {item.name}"
										onclick={() => openEdit(item)}
									>
										<Edit3 class="h-3.5 w-3.5" />
									</Button>

									{#if item.status === 'PENDING'}
										<Button
											variant="outline"
											size="sm"
											class="border-emerald-200 py-1 text-xs text-emerald-800 hover:bg-emerald-50"
											onclick={() => openStatus(item, 'ACTIVE')}
										>
											<CheckCircle2 class="mr-1 h-3 w-3" />
											<span>Activate</span>
										</Button>
									{:else if item.status === 'ACTIVE'}
										<Button
											variant="outline"
											size="sm"
											class="border-rose-200 py-1 text-xs text-rose-700 hover:bg-rose-50"
											onclick={() => openStatus(item, 'SUSPENDED')}
										>
											<ShieldAlert class="mr-1 h-3 w-3" />
											<span>Suspend</span>
										</Button>
									{:else if item.status === 'SUSPENDED'}
										<Button
											variant="outline"
											size="sm"
											class="border-neutral-300 py-1 text-xs text-neutral-800 hover:bg-neutral-100"
											onclick={() => openStatus(item, 'ACTIVE')}
										>
											<CheckCircle2 class="mr-1 h-3 w-3" />
											<span>Reactivate</span>
										</Button>
									{/if}
								</div>
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
				<span class="font-medium text-neutral-900">{total}</span> societies
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

<!-- Details Modal -->
<SocietyDetailsDialog
	isOpen={isDetailsOpen}
	society={selectedSociety}
	onClose={() => {
		isDetailsOpen = false;
		selectedSociety = null;
	}}
	onEdit={(soc) => {
		selectedSociety = soc;
		isEditOpen = true;
	}}
/>

<!-- Edit Modal -->
<EditSocietyDialog
	isOpen={isEditOpen}
	society={selectedSociety}
	onClose={() => {
		isEditOpen = false;
		selectedSociety = null;
	}}
	onOpenFormation={(soc) => {
		isEditOpen = false;
		openFormation(soc);
	}}
	onSuccess={() => {
		loadSocieties();
	}}
/>

<!-- Status Transition Modal -->
<SocietyStatusDialog
	isOpen={isStatusOpen}
	society={selectedSociety}
	{targetStatus}
	onClose={() => {
		isStatusOpen = false;
		selectedSociety = null;
	}}
	onSuccess={() => {
		loadSocieties();
	}}
/>

<!-- Formation Wizard Modal -->
{#if selectedSociety}
	<FormationWizard
		society={selectedSociety}
		isOpen={isFormationOpen}
		onClose={() => {
			isFormationOpen = false;
			selectedSociety = null;
		}}
		onComplete={() => {
			isFormationOpen = false;
			selectedSociety = null;
			loadSocieties();
		}}
	/>
{/if}
