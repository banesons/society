<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { societyApi, type Visit } from '$lib/api/society';
	import { formatPhoneDisplay } from '$lib/phone';
	import { Check, X, Loader2, AlertCircle, RefreshCw, Home, Clock } from '@lucide/svelte';

	let {
		societyId,
		onDecisionMade
	}: {
		societyId: string;
		onDecisionMade?: () => void;
	} = $props();

	let pendingVisits = $state<Visit[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	let processingId = $state<string | null>(null);
	let rejectingVisit = $state<Visit | null>(null);
	let rejectReason = $state('');

	async function loadPending() {
		if (!societyId) return;
		isLoading = true;
		error = null;
		try {
			const res = await societyApi.listVisits(societyId, {
				status: 'PENDING_APPROVAL',
				limit: 100
			});
			pendingVisits = res.visits;
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load approval requests';
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (societyId) {
			loadPending();
		}
	});

	async function handleApprove(visit: Visit) {
		processingId = visit.id;
		try {
			await societyApi.approveVisit(societyId, visit.id, { notes: 'Approved via resident queue' });
			await loadPending();
			if (onDecisionMade) onDecisionMade();
		} catch (err: unknown) {
			alert(err instanceof Error ? err.message : 'Approval failed');
		} finally {
			processingId = null;
		}
	}

	async function handleConfirmReject() {
		if (!rejectingVisit) return;
		processingId = rejectingVisit.id;
		try {
			await societyApi.rejectVisit(societyId, rejectingVisit.id, {
				reason: rejectReason.trim() || undefined
			});
			rejectingVisit = null;
			rejectReason = '';
			await loadPending();
			if (onDecisionMade) onDecisionMade();
		} catch (err: unknown) {
			alert(err instanceof Error ? err.message : 'Rejection failed');
		} finally {
			processingId = null;
		}
	}
</script>

<div class="space-y-4">
	<div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<h3 class="text-sm font-semibold tracking-tight text-neutral-900">Resident Approval Queue</h3>
			<p class="text-xs text-neutral-500">
				Pending visitor entries requiring resident or administrator clearance
			</p>
		</div>

		<Button variant="outline" size="sm" onclick={loadPending} disabled={isLoading}>
			<RefreshCw class="mr-1.5 h-3.5 w-3.5 {isLoading ? 'animate-spin' : ''}" />
			<span>Refresh Queue</span>
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

	{#if isLoading}
		<div class="flex min-h-[160px] flex-col items-center justify-center text-neutral-500">
			<Loader2 class="h-6 w-6 animate-spin text-neutral-900" />
			<p class="mt-2 text-xs">Checking approval queue...</p>
		</div>
	{:else if pendingVisits.length === 0}
		<div
			class="flex min-h-[160px] flex-col items-center justify-center rounded-2xl border border-dashed border-neutral-200 p-8 text-center"
		>
			<div
				class="flex h-10 w-10 items-center justify-center rounded-full bg-neutral-100 text-neutral-500"
			>
				<Check class="h-5 w-5 text-emerald-600" />
			</div>
			<p class="mt-3 text-xs font-medium text-neutral-900">No Pending Approvals</p>
			<p class="mt-1 text-[11px] text-neutral-500">
				All incoming visitors are cleared or processed
			</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-3 md:grid-cols-2">
			{#each pendingVisits as visit (visit.id)}
				<div
					class="rounded-2xl border border-neutral-200 bg-white/90 p-5 shadow-xs transition-shadow hover:shadow-sm"
				>
					<div class="flex items-start justify-between">
						<div>
							<div class="flex items-center gap-2">
								<h4 class="text-sm font-semibold text-neutral-950">{visit.visitor_name}</h4>
								<Badge variant="warning" class="text-[10px]">Pending Approval</Badge>
							</div>
							<p class="mt-1 font-mono text-xs text-neutral-600">
								{formatPhoneDisplay(visit.visitor_phone_masked)}
							</p>
						</div>

						<Badge variant="outline" class="font-mono text-[10px]">{visit.visit_type}</Badge>
					</div>

					<div
						class="mt-3.5 space-y-1.5 rounded-xl border border-neutral-100 bg-neutral-50/60 p-3 text-xs"
					>
						<div class="flex items-center justify-between text-neutral-600">
							<span class="flex items-center gap-1.5 text-neutral-500">
								<Home class="h-3.5 w-3.5" /> Destination
							</span>
							<span class="font-mono font-medium text-neutral-900"
								>{visit.building_name} — {visit.unit_identifier}</span
							>
						</div>

						<div class="flex items-center justify-between text-neutral-600">
							<span class="text-neutral-500">Purpose</span>
							<span class="font-medium text-neutral-900">{visit.purpose}</span>
						</div>

						{#if visit.vehicle_number}
							<div class="flex items-center justify-between text-neutral-600">
								<span class="text-neutral-500">Vehicle</span>
								<span class="font-mono text-neutral-900">{visit.vehicle_number}</span>
							</div>
						{/if}

						<div class="flex items-center justify-between text-neutral-600">
							<span class="flex items-center gap-1.5 text-neutral-500">
								<Clock class="h-3.5 w-3.5" /> Registered
							</span>
							<span class="text-neutral-700"
								>{new Date(visit.created_at).toLocaleTimeString([], {
									hour: '2-digit',
									minute: '2-digit'
								})}</span
							>
						</div>
					</div>

					<div class="mt-4 flex items-center justify-end gap-2 border-t border-neutral-100 pt-3">
						<button
							type="button"
							onclick={() => {
								rejectingVisit = visit;
								rejectReason = '';
							}}
							disabled={processingId === visit.id}
							class="inline-flex items-center gap-1 rounded-xl border border-rose-200 bg-rose-50 px-3 py-1.5 text-xs font-medium text-rose-700 hover:bg-rose-100 disabled:opacity-50"
						>
							<X class="h-3.5 w-3.5" />
							<span>Reject</span>
						</button>

						<Button
							variant="primary"
							size="sm"
							onclick={() => handleApprove(visit)}
							disabled={processingId === visit.id}
						>
							{#if processingId === visit.id}
								<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							{:else}
								<Check class="mr-1.5 h-3.5 w-3.5 text-emerald-400" />
							{/if}
							<span>Approve Entry</span>
						</Button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Reject Reason Modal -->
{#if rejectingVisit}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-xs"
		role="dialog"
		aria-modal="true"
	>
		<div class="w-full max-w-sm rounded-2xl border border-neutral-200 bg-white p-5 shadow-xl">
			<div class="flex items-center justify-between border-b border-neutral-100 pb-3">
				<h3 class="text-sm font-semibold text-neutral-900">Reject Visitor Entry</h3>
				<button
					type="button"
					onclick={() => (rejectingVisit = null)}
					class="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<p class="mt-3 text-xs text-neutral-600">
				Rejecting entry for <strong class="text-neutral-900">{rejectingVisit.visitor_name}</strong>.
				Provide an optional reason for the gatekeeper and audit log.
			</p>

			<div class="mt-3">
				<label for="rej-reason" class="mb-1 block text-xs font-medium text-neutral-700"
					>Rejection Reason</label
				>
				<input
					id="rej-reason"
					type="text"
					bind:value={rejectReason}
					placeholder="e.g. Resident unavailable, wrong flat"
					class="w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 outline-none focus:border-neutral-900"
				/>
			</div>

			<div class="mt-5 flex justify-end gap-2">
				<Button variant="outline" size="sm" onclick={() => (rejectingVisit = null)}>Cancel</Button>
				<button
					type="button"
					onclick={handleConfirmReject}
					class="inline-flex items-center gap-1 rounded-xl bg-rose-600 px-3.5 py-1.5 text-xs font-medium text-white hover:bg-rose-700"
				>
					Confirm Rejection
				</button>
			</div>
		</div>
	</div>
{/if}
