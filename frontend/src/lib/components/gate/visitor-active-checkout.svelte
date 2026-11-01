<script lang="ts">
	import { onMount } from 'svelte';
	import { societyApi, type Visit, type Gate } from '$lib/api/society';
	import Button from '$lib/components/ui/button.svelte';
	import ResponsiveModal from '$lib/components/ui/responsive-modal.svelte';
	import { toast } from '$lib/toast.svelte';
	import { Search, LogOut, Clock, Building2, Car, CheckCircle2, Loader2 } from '@lucide/svelte';

	let {
		societyId,
		gates,
		onCheckedOut
	}: { societyId: string; gates: Gate[]; onCheckedOut?: () => void } = $props();

	let activeVisits = $state<Visit[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let searchQuery = $state('');
	let searchTimeout: ReturnType<typeof setTimeout> | null = null;

	// Checkout dialog state
	let isCheckoutModalOpen = $state(false);
	let selectedVisit = $state<Visit | null>(null);
	let selectedGateId = $state<string>('');
	let checkoutNotes = $state('');
	let isSubmittingCheckout = $state(false);

	let egressGates = $derived(
		gates.filter(
			(g) =>
				g.status === 'ACTIVE' &&
				(g.operational_status === 'ACTIVE' || g.operational_status === 'EMERGENCY_OPEN') &&
				g.directionality !== 'INGRESS_ONLY'
		)
	);

	async function loadActiveVisits(search = '') {
		isLoading = true;
		error = null;
		try {
			const res = await societyApi.listActiveVisits(societyId, search);
			activeVisits = res.visits ?? [];
		} catch (err: unknown) {
			if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Failed to load active visits';
			}
		} finally {
			isLoading = false;
		}
	}

	function handleSearchInput(e: Event) {
		const val = (e.target as HTMLInputElement).value;
		searchQuery = val;
		if (searchTimeout) clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => {
			loadActiveVisits(searchQuery);
		}, 300);
	}

	function openCheckoutModal(visit: Visit) {
		selectedVisit = visit;
		// Default to first egress gate, or if visit had an entry gate that is bidirectional, or first available
		if (egressGates.length > 0) {
			selectedGateId = egressGates[0].id;
		}
		checkoutNotes = '';
		isCheckoutModalOpen = true;
	}

	async function confirmCheckout() {
		if (!selectedVisit || !selectedGateId || isSubmittingCheckout) return;

		isSubmittingCheckout = true;
		try {
			await societyApi.checkOutVisitor(societyId, selectedVisit.id, {
				gate_id: selectedGateId,
				notes: checkoutNotes.trim() || undefined
			});
			toast.success(`Visitor ${selectedVisit.visitor_name} checked out successfully`);
			isCheckoutModalOpen = false;
			selectedVisit = null;
			loadActiveVisits(searchQuery);
			onCheckedOut?.();
		} catch (err: unknown) {
			if (err instanceof Error) {
				toast.error(err.message);
			} else {
				toast.error('Failed to complete visitor checkout');
			}
		} finally {
			isSubmittingCheckout = false;
		}
	}

	function formatTime(iso?: string): string {
		if (!iso) return '—';
		return new Date(iso).toLocaleTimeString('en-IN', {
			hour: '2-digit',
			minute: '2-digit',
			hour12: true
		});
	}

	onMount(() => {
		loadActiveVisits();
	});
</script>

<div class="space-y-4">
	<!-- Search & Actions Bar -->
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<div class="relative max-w-md flex-1">
			<Search class="absolute top-1/2 left-3.5 h-4 w-4 -translate-y-1/2 text-[#8a8a82]" />
			<input
				type="text"
				placeholder="Search by visitor name, flat, vehicle, or phone..."
				value={searchQuery}
				oninput={handleSearchInput}
				class="w-full rounded-xl border border-[#e2e2de] bg-white py-2.5 pr-4 pl-10 text-xs font-medium text-[#121212] placeholder:text-[#8a8a82] focus:border-[#121212] focus:ring-1 focus:ring-[#121212] focus:outline-hidden"
			/>
		</div>

		<div class="flex items-center gap-2">
			<span
				class="rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-medium text-emerald-800"
			>
				{activeVisits.length} Currently Inside
			</span>
			<Button
				variant="outline"
				size="sm"
				onclick={() => loadActiveVisits(searchQuery)}
				class="border-[#e2e2de] text-xs text-[#121212]"
			>
				Refresh
			</Button>
		</div>
	</div>

	<!-- Active Visits Cards Grid -->
	{#if isLoading}
		<div
			class="flex min-h-[200px] items-center justify-center rounded-2xl border border-[#e2e2de] bg-white p-8"
		>
			<div class="flex items-center gap-2.5 text-xs text-[#666660]">
				<Loader2 class="h-4 w-4 animate-spin text-[#121212]" />
				<span>Loading active visits inside perimeter...</span>
			</div>
		</div>
	{:else if error}
		<div class="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800">
			<p class="font-medium">Failed to load active visits</p>
			<p class="mt-0.5 text-rose-600">{error}</p>
			<button
				type="button"
				onclick={() => loadActiveVisits(searchQuery)}
				class="mt-2 text-xs font-medium underline hover:text-rose-950"
			>
				Try again
			</button>
		</div>
	{:else if activeVisits.length === 0}
		<div class="rounded-2xl border border-dashed border-[#e2e2de] bg-white p-12 text-center">
			<CheckCircle2 class="mx-auto h-7 w-7 text-emerald-600" />
			<p class="mt-2 text-sm font-medium text-[#121212]">No Active Visits Inside</p>
			<p class="mt-0.5 text-xs text-[#666660]">
				All checked-in visitors have departed the society grounds, or no active visits match your
				search.
			</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
			{#each activeVisits as visit (visit.id)}
				<div
					class="flex flex-col justify-between rounded-xl border border-[#e2e2de] bg-white p-4 shadow-2xs transition-shadow hover:shadow-xs"
				>
					<div class="space-y-2.5">
						<!-- Top row: visitor name + badge -->
						<div class="flex items-start justify-between gap-2">
							<div class="min-w-0">
								<h3 class="truncate text-sm font-semibold text-[#121212]">
									{visit.visitor_name}
								</h3>
								<p class="text-xs text-[#666660]">
									{visit.visitor_phone_masked || 'Phone masked'}
								</p>
							</div>
							<span
								class="inline-flex shrink-0 items-center rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[10px] font-medium text-emerald-800"
							>
								Inside
							</span>
						</div>

						<!-- Details -->
						<div class="space-y-1 text-xs text-[#666660]">
							<div class="flex items-center gap-1.5">
								<Building2 class="h-3.5 w-3.5 shrink-0 text-[#8a8a82]" />
								<span>Flat {visit.unit_identifier} ({visit.building_name})</span>
							</div>

							{#if visit.entry_gate_name || visit.check_in_at}
								<div class="flex items-center gap-1.5">
									<Clock class="h-3.5 w-3.5 shrink-0 text-[#8a8a82]" />
									<span
										>Entered {formatTime(visit.check_in_at)} via {visit.entry_gate_name ||
											'Gate'}</span
									>
								</div>
							{/if}

							{#if visit.vehicle_number}
								<div class="flex items-center gap-1.5">
									<Car class="h-3.5 w-3.5 shrink-0 text-[#8a8a82]" />
									<span class="font-mono font-medium text-[#121212]">{visit.vehicle_number}</span>
								</div>
							{/if}
						</div>
					</div>

					<!-- Bottom Checkout Action -->
					<div class="mt-4 border-t border-[#f3f3f1] pt-3">
						<Button
							onclick={() => openCheckoutModal(visit)}
							class="h-9 w-full rounded-lg bg-[#121212] text-xs font-medium text-white hover:bg-[#262626]"
						>
							<LogOut class="mr-1.5 h-3.5 w-3.5" />
							<span>Check Out Visitor</span>
						</Button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Checkout Confirmation Modal -->
{#if isCheckoutModalOpen && selectedVisit}
	<ResponsiveModal
		isOpen={isCheckoutModalOpen}
		onClose={() => {
			if (!isSubmittingCheckout) {
				isCheckoutModalOpen = false;
				selectedVisit = null;
			}
		}}
		title="Confirm Visitor Departure"
	>
		<div class="space-y-4">
			<div class="rounded-xl border border-[#e2e2de] bg-[#f3f3f1] p-3 text-xs">
				<p class="font-semibold text-[#121212]">{selectedVisit.visitor_name}</p>
				<p class="text-[#666660]">
					Flat {selectedVisit.unit_identifier} · Entered at {formatTime(selectedVisit.check_in_at)}
				</p>
				{#if selectedVisit.entry_gate_name}
					<p class="mt-0.5 text-[11px] text-[#8a8a82]">
						Entry Gate: {selectedVisit.entry_gate_name}
					</p>
				{/if}
			</div>

			<!-- Egress Gate Selector (Allows cross-gate departure) -->
			<div>
				<label for="egress-gate-select" class="mb-1 block text-xs font-medium text-[#121212]">
					Departure / Exit Gate
				</label>
				<select
					id="egress-gate-select"
					bind:value={selectedGateId}
					class="w-full rounded-xl border border-[#e2e2de] bg-white p-2.5 text-xs font-medium text-[#121212] focus:border-[#121212] focus:outline-hidden"
				>
					{#each egressGates as gate (gate.id)}
						<option value={gate.id}>
							{gate.name} ({gate.directionality})
						</option>
					{/each}
				</select>
				<p class="mt-1 text-[11px] text-[#8a8a82]">
					Visitors may depart through any authorized egress gate regardless of where they entered.
				</p>
			</div>

			<div>
				<label for="checkout-notes" class="mb-1 block text-xs font-medium text-[#121212]">
					Departure Notes (Optional)
				</label>
				<input
					id="checkout-notes"
					type="text"
					placeholder="Optional remarks or observations"
					bind:value={checkoutNotes}
					class="w-full rounded-xl border border-[#e2e2de] bg-white p-2.5 text-xs text-[#121212] focus:border-[#121212] focus:outline-hidden"
				/>
			</div>

			<div class="flex items-center justify-end gap-2 border-t border-[#e2e2de] pt-2">
				<Button
					variant="outline"
					size="sm"
					disabled={isSubmittingCheckout}
					onclick={() => {
						isCheckoutModalOpen = false;
						selectedVisit = null;
					}}
					class="text-xs"
				>
					Cancel
				</Button>
				<Button
					size="sm"
					disabled={isSubmittingCheckout || !selectedGateId}
					onclick={confirmCheckout}
					class="bg-[#121212] text-xs text-white hover:bg-[#262626]"
				>
					{#if isSubmittingCheckout}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
						<span>Confirming Exit...</span>
					{:else}
						<LogOut class="mr-1.5 h-3.5 w-3.5" />
						<span>Record Exit</span>
					{/if}
				</Button>
			</div>
		</div>
	</ResponsiveModal>
{/if}
