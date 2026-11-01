<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import ResponsiveModal from '$lib/components/ui/responsive-modal.svelte';
	import { societyApi, type PendingQRVisit } from '$lib/api/society';
	import {
		Check,
		X,
		Loader2,
		AlertCircle,
		RefreshCw,
		Clock,
		User,
		QrCode,
		MapPin
	} from '@lucide/svelte';

	let {
		societyId,
		currentGateId,
		onDecisionMade
	}: {
		societyId: string;
		currentGateId?: string;
		onDecisionMade?: () => void;
	} = $props();

	let pendingRegistrations = $state<PendingQRVisit[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	let processingId = $state<string | null>(null);
	let rejectingVisit = $state<PendingQRVisit | null>(null);
	let rejectReason = $state('');
	let isRejectDialogOpen = $state(false);

	async function loadPending() {
		if (!societyId) return;
		isLoading = true;
		error = null;
		try {
			const res = await societyApi.listPendingQRRegistrations(societyId, {
				gate_id: currentGateId,
				limit: 100
			});
			pendingRegistrations = res.pending_registrations;
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load pending registrations';
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (societyId) {
			loadPending();
		}
	});

	async function handleAllowEntry(visit: PendingQRVisit) {
		if (!currentGateId && !visit.gate_id) {
			alert('Please select an active gate for admission');
			return;
		}
		const gateId = currentGateId || visit.gate_id!;
		processingId = visit.id;
		try {
			await societyApi.admitQRVisitor(societyId, visit.id, {
				decision: 'APPROVE',
				gate_id: gateId
			});
			await loadPending();
			if (onDecisionMade) onDecisionMade();
		} catch (err: unknown) {
			alert(err instanceof Error ? err.message : 'Failed to admit visitor');
		} finally {
			processingId = null;
		}
	}

	function openRejectDialog(visit: PendingQRVisit) {
		rejectingVisit = visit;
		rejectReason = '';
		isRejectDialogOpen = true;
	}

	async function handleConfirmReject() {
		if (!rejectingVisit) return;
		const gateId = currentGateId || rejectingVisit.gate_id || '';
		processingId = rejectingVisit.id;
		try {
			await societyApi.admitQRVisitor(societyId, rejectingVisit.id, {
				decision: 'REJECT',
				gate_id: gateId,
				rejection_reason: rejectReason.trim() || undefined
			});
			isRejectDialogOpen = false;
			rejectingVisit = null;
			rejectReason = '';
			await loadPending();
			if (onDecisionMade) onDecisionMade();
		} catch (err: unknown) {
			alert(err instanceof Error ? err.message : 'Failed to reject visitor');
		} finally {
			processingId = null;
		}
	}
</script>

<div class="space-y-4">
	<div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2">
				<h3 class="text-sm font-semibold tracking-tight text-neutral-900">QR Self-Registrations</h3>
				{#if pendingRegistrations.length > 0}
					<Badge
						variant="outline"
						class="border-amber-200 bg-amber-50 font-mono text-[10px] text-amber-800"
					>
						{pendingRegistrations.length} Pending
					</Badge>
				{/if}
			</div>
			<p class="text-xs text-neutral-500">
				Visitors awaiting security gate verification after scanning physical entrance QR
			</p>
		</div>

		<Button variant="outline" size="sm" onclick={loadPending} disabled={isLoading} class="text-xs">
			<RefreshCw class="mr-1.5 h-3.5 w-3.5 {isLoading ? 'animate-spin' : ''}" />
			<span>Refresh</span>
		</Button>
	</div>

	{#if error}
		<div
			class="flex items-center gap-2 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
		>
			<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
			<span>{error}</span>
		</div>
	{/if}

	{#if isLoading && pendingRegistrations.length === 0}
		<div
			class="rounded-2xl border border-[#e2e2de] bg-white p-8 text-center text-xs text-neutral-500"
		>
			<Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin text-neutral-400" />
			Loading pending registrations...
		</div>
	{:else if pendingRegistrations.length === 0}
		<div class="rounded-2xl border border-dashed border-[#e2e2de] bg-neutral-50 p-8 text-center">
			<QrCode class="mx-auto mb-2 h-8 w-8 text-neutral-400 opacity-60" />
			<p class="text-xs font-semibold text-neutral-800">No Pending QR Registrations</p>
			<p class="mt-0.5 text-[11px] text-neutral-500">
				Visitors who register by scanning gate QR codes will appear here for verification.
			</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			{#each pendingRegistrations as reg (reg.id)}
				<div
					class="flex flex-col justify-between space-y-4 rounded-2xl border border-[#e2e2de] bg-white p-5 shadow-sm"
				>
					<!-- Top Row: Photo & Details -->
					<div class="flex items-start gap-3.5">
						{#if reg.visitor_photo_url}
							<img
								src={reg.visitor_photo_url}
								alt={reg.visitor_name}
								class="h-16 w-16 shrink-0 rounded-xl border border-[#e2e2de] bg-neutral-100 object-cover shadow-sm"
							/>
						{:else}
							<div
								class="flex h-16 w-16 shrink-0 items-center justify-center rounded-xl border border-[#e2e2de] bg-neutral-100 text-neutral-400"
							>
								<User class="h-8 w-8" />
							</div>
						{/if}

						<div class="min-w-0 flex-1 space-y-1">
							<div class="flex items-center justify-between gap-2">
								<h4 class="truncate text-sm font-semibold text-neutral-900">
									{reg.visitor_name}
								</h4>
								<span
									class="shrink-0 rounded-md bg-neutral-100 px-2 py-0.5 font-mono text-xs font-bold text-neutral-800"
								>
									{reg.pass_reference}
								</span>
							</div>

							<p class="font-mono text-xs text-neutral-500">
								{reg.visitor_phone_masked}
							</p>

							<div class="flex flex-wrap items-center gap-1.5 pt-0.5">
								<Badge variant="outline" class="border-neutral-200 text-[10px] font-medium">
									{reg.purpose.replace(/_/g, ' ')}
								</Badge>
								{#if reg.gate_name}
									<span class="flex items-center gap-0.5 text-[10px] text-neutral-500">
										<MapPin class="h-2.5 w-2.5" />
										{reg.gate_name}
									</span>
								{/if}
							</div>
						</div>
					</div>

					<!-- Middle: Origin & Time -->
					<div
						class="flex items-center justify-between rounded-xl bg-neutral-50 p-2.5 text-[11px] text-neutral-600"
					>
						<div>
							{#if reg.coming_from}
								<span class="text-neutral-500">From:</span>
								<span class="ml-1 font-medium text-neutral-800">{reg.coming_from}</span>
							{:else}
								<span class="text-neutral-400">Self-Registration</span>
							{/if}
						</div>
						<div class="flex items-center gap-1 font-mono text-[10px] text-neutral-400">
							<Clock class="h-3 w-3" />
							{new Date(reg.created_at).toLocaleTimeString([], {
								hour: '2-digit',
								minute: '2-digit'
							})}
						</div>
					</div>

					<!-- Bottom: Action Buttons -->
					<div class="flex items-center gap-2 border-t border-[#f0f0ed] pt-1">
						<Button
							variant="outline"
							size="sm"
							class="flex-1 border-rose-200 text-xs text-rose-700 hover:bg-rose-50"
							onclick={() => openRejectDialog(reg)}
							disabled={processingId === reg.id}
						>
							<X class="mr-1 h-3.5 w-3.5" />
							Reject
						</Button>

						<Button
							size="sm"
							class="flex-1 bg-[#121212] text-xs font-medium text-white hover:bg-neutral-800"
							onclick={() => handleAllowEntry(reg)}
							disabled={processingId === reg.id}
						>
							{#if processingId === reg.id}
								<Loader2 class="mr-1 h-3.5 w-3.5 animate-spin" />
							{:else}
								<Check class="mr-1 h-3.5 w-3.5" />
							{/if}
							Allow Entry
						</Button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Rejection Reason Modal -->
<ResponsiveModal
	isOpen={isRejectDialogOpen}
	title="Reject Visitor Registration"
	description="State reason for denying entrance to {rejectingVisit?.visitor_name}"
	onClose={() => {
		isRejectDialogOpen = false;
		rejectingVisit = null;
	}}
>
	<div class="space-y-4">
		<div>
			<label class="mb-1 block text-xs font-semibold text-neutral-700">
				Reason for Rejection
			</label>
			<textarea
				bind:value={rejectReason}
				placeholder="e.g. Visitor identity did not match photograph, invalid destination"
				rows="3"
				class="w-full rounded-xl border border-[#e2e2de] px-3 py-2 text-xs focus:border-neutral-900 focus:outline-none"
			></textarea>
		</div>

		<div class="flex items-center justify-end gap-2 pt-2">
			<Button
				variant="outline"
				size="sm"
				onclick={() => {
					isRejectDialogOpen = false;
					rejectingVisit = null;
				}}
			>
				Cancel
			</Button>
			<Button
				variant="destructive"
				size="sm"
				onclick={handleConfirmReject}
				disabled={processingId !== null}
			>
				{#if processingId}
					<Loader2 class="mr-1 h-3.5 w-3.5 animate-spin" />
				{/if}
				Confirm Rejection
			</Button>
		</div>
	</div>
</ResponsiveModal>
