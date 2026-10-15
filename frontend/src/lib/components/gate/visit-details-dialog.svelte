<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import {
		societyApi,
		type Visit,
		type VisitorPass,
		type VisitApproval,
		type GateEvent
	} from '$lib/api/society';
	import { formatPhoneDisplay } from '$lib/phone';
	import { X, Loader2, QrCode, ShieldCheck, LogIn, LogOut, AlertCircle } from '@lucide/svelte';

	let {
		societyId,
		visitId,
		isOpen,
		onClose
	}: {
		societyId: string;
		visitId: string | null;
		isOpen: boolean;
		onClose: () => void;
	} = $props();

	let visit = $state<Visit | null>(null);
	let passes = $state<VisitorPass[]>([]);
	let approvals = $state<VisitApproval[]>([]);
	let events = $state<GateEvent[]>([]);

	let isLoading = $state(false);
	let error = $state<string | null>(null);

	async function loadDetails() {
		if (!societyId || !visitId) return;
		isLoading = true;
		error = null;

		try {
			const res = await societyApi.getVisitDetails(societyId, visitId);
			visit = res.visit;
			passes = res.passes;
			approvals = res.approvals;
			events = res.events;
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load visit details';
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (isOpen && visitId) {
			loadDetails();
		}
	});
</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape' && isOpen) onClose();
	}}
/>

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-xs"
		role="dialog"
		aria-modal="true"
		onclick={(e) => {
			if (e.target === e.currentTarget) onClose();
		}}
	>
		<div
			class="max-h-[90vh] w-full max-w-xl overflow-y-auto rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div>
					<h3 class="text-sm font-semibold tracking-tight text-neutral-900">
						Visit Lifecycle Details
					</h3>
					<p class="text-[11px] text-neutral-500">
						Authoritative audit ledger and movement timeline
					</p>
				</div>
				<button
					type="button"
					onclick={onClose}
					class="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if isLoading}
				<div class="flex min-h-[200px] flex-col items-center justify-center text-neutral-500">
					<Loader2 class="h-6 w-6 animate-spin text-neutral-900" />
					<p class="mt-2 text-xs">Loading visit record...</p>
				</div>
			{:else if error}
				<div
					class="mt-4 flex items-center gap-2 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
				>
					<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
					<span>{error}</span>
				</div>
			{:else if visit}
				<div class="mt-4 space-y-5 text-xs">
					<!-- Overview Card -->
					<div class="rounded-xl border border-neutral-100 bg-neutral-50/80 p-4">
						<div class="flex items-start justify-between">
							<div>
								<h4 class="text-base font-bold text-neutral-950">{visit.visitor_name}</h4>
								<p class="mt-0.5 font-mono text-neutral-600">
									{formatPhoneDisplay(visit.visitor_phone_masked)}
								</p>
							</div>
							<Badge
								variant={visit.status === 'CHECKED_IN' || visit.status === 'APPROVED'
									? 'success'
									: 'default'}
							>
								{visit.status}
							</Badge>
						</div>

						<div
							class="mt-3 grid grid-cols-2 gap-3 border-t border-neutral-200/60 pt-3 text-neutral-700"
						>
							<div>
								<span class="text-neutral-500">Destination:</span>
								<p class="font-mono font-medium text-neutral-900">
									{visit.building_name} — {visit.unit_identifier}
								</p>
							</div>
							<div>
								<span class="text-neutral-500">Purpose:</span>
								<p class="font-medium text-neutral-900">{visit.purpose}</p>
							</div>
							{#if visit.vehicle_number}
								<div>
									<span class="text-neutral-500">Vehicle:</span>
									<p class="font-mono text-neutral-900 uppercase">{visit.vehicle_number}</p>
								</div>
							{/if}
							<div>
								<span class="text-neutral-500">Visit Type:</span>
								<p class="font-medium text-neutral-900">{visit.visit_type}</p>
							</div>
						</div>
					</div>

					<!-- Pass History -->
					<div>
						<h5 class="text-xs font-semibold tracking-wider text-neutral-500 uppercase">
							Visitor Passes
						</h5>
						{#if passes.length === 0}
							<p class="mt-2 text-neutral-400">No passes issued for this visit.</p>
						{:else}
							<div class="mt-2 space-y-2">
								{#each passes as p (p.id)}
									<div
										class="flex items-center justify-between rounded-xl border border-neutral-200 bg-white p-3"
									>
										<div class="flex items-center gap-2">
											<QrCode class="h-4 w-4 text-neutral-700" />
											<span class="font-mono font-bold text-neutral-900">{p.pass_code}</span>
											<Badge variant="outline" class="text-[10px]">{p.status}</Badge>
										</div>
										<span class="text-[11px] text-neutral-500">
											Expires: {new Date(p.valid_until).toLocaleTimeString([], {
												hour: '2-digit',
												minute: '2-digit'
											})}
										</span>
									</div>
								{/each}
							</div>
						{/if}
					</div>

					<!-- Resident Approval Decisions -->
					<div>
						<h5 class="text-xs font-semibold tracking-wider text-neutral-500 uppercase">
							Resident Approval Decisions
						</h5>
						{#if approvals.length === 0}
							<p class="mt-2 text-neutral-400">No explicit approval actions recorded.</p>
						{:else}
							<div class="mt-2 space-y-2">
								{#each approvals as a (a.id)}
									<div class="rounded-xl border border-neutral-200 bg-white p-3">
										<div class="flex items-center justify-between">
											<div class="flex items-center gap-2">
												<ShieldCheck
													class="h-4 w-4 {a.decision === 'APPROVED'
														? 'text-emerald-600'
														: 'text-rose-600'}"
												/>
												<span class="font-semibold text-neutral-900">{a.decision}</span>
											</div>
											<span class="text-[11px] text-neutral-500"
												>{new Date(a.created_at).toLocaleTimeString()}</span
											>
										</div>
										{#if a.notes}
											<p class="mt-1 text-neutral-600 italic">"{a.notes}"</p>
										{/if}
									</div>
								{/each}
							</div>
						{/if}
					</div>

					<!-- Physical Movement Events -->
					<div>
						<h5 class="text-xs font-semibold tracking-wider text-neutral-500 uppercase">
							Perimeter Gate Events
						</h5>
						{#if events.length === 0}
							<p class="mt-2 text-neutral-400">No physical entry or exit logged yet.</p>
						{:else}
							<div class="mt-2 space-y-2">
								{#each events as e (e.id)}
									<div
										class="flex items-center justify-between rounded-xl border border-neutral-200 bg-white p-3"
									>
										<div class="flex items-center gap-2">
											{#if e.event_type === 'ENTRY'}
												<LogIn class="h-4 w-4 text-emerald-600" />
											{:else}
												<LogOut class="h-4 w-4 text-amber-600" />
											{/if}
											<div>
												<span class="font-semibold text-neutral-900"
													>{e.event_type} ({e.direction})</span
												>
												<span class="text-neutral-500">at {e.gate_name}</span>
											</div>
										</div>
										<span class="font-mono text-[11px] text-neutral-500"
											>{new Date(e.created_at).toLocaleTimeString()}</span
										>
									</div>
								{/each}
							</div>
						{/if}
					</div>
				</div>
			{/if}

			<div class="mt-6 flex justify-end border-t border-neutral-100 pt-4">
				<Button variant="outline" size="sm" onclick={onClose}>Close</Button>
			</div>
		</div>
	</div>
{/if}
