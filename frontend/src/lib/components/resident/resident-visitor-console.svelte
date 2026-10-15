<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { societyApi, type ResidentUnit, type Visit, type VisitorPass } from '$lib/api/society';
	import { formatPhoneDisplay } from '$lib/phone';
	import VisitorRegisterDialog from '$lib/components/gate/visitor-register-dialog.svelte';
	import VisitorPassDialog from '$lib/components/gate/visitor-pass-dialog.svelte';
	import {
		Home,
		Bell,
		Check,
		RefreshCw,
		AlertCircle,
		Loader2,
		Calendar,
		UserCheck,
		UserX
	} from '@lucide/svelte';

	let { societyId }: { societyId: string } = $props();

	let units = $state<ResidentUnit[]>([]);
	let pendingVisits = $state<Visit[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let successMessage = $state<string | null>(null);

	let togglingUnitId = $state<string | null>(null);
	let actingVisitId = $state<string | null>(null);

	// Pre-register expected visitor dialog
	let isExpectedDialogOpen = $state(false);
	let isPassDialogOpen = $state(false);
	let activePassVisit = $state<Visit | null>(null);
	let activePass = $state<VisitorPass | null>(null);

	// Reject dialog
	let rejectTargetVisit = $state<Visit | null>(null);
	let rejectReason = $state('Declined by resident');

	async function loadResidentData() {
		if (!societyId) return;
		isLoading = true;
		error = null;

		try {
			const [unitsRes, pendingRes] = await Promise.all([
				societyApi.getMyUnits(societyId),
				societyApi.getPendingVisits(societyId)
			]);
			units = unitsRes.units;
			pendingVisits = pendingRes.visits;
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load resident dashboard';
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (societyId) {
			loadResidentData();
		}
	});

	async function handleTogglePolicy(unit: ResidentUnit) {
		togglingUnitId = unit.id;
		error = null;
		successMessage = null;

		const targetSetting = !unit.requires_visitor_approval;
		try {
			const res = await societyApi.updateUnitVisitorPolicy(societyId, unit.id, targetSetting);
			units = units.map((u) =>
				u.id === unit.id
					? { ...u, requires_visitor_approval: res.unit.requires_visitor_approval }
					: u
			);
			successMessage = `Visitor approval requirement for Flat ${unit.unit_identifier} turned ${
				res.unit.requires_visitor_approval ? 'ON' : 'OFF'
			}.`;
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to update approval preference';
		} finally {
			togglingUnitId = null;
		}
	}

	async function handleApprove(visit: Visit) {
		actingVisitId = visit.id;
		error = null;
		successMessage = null;

		try {
			await societyApi.approveVisit(societyId, visit.id, {
				notes: 'Approved via resident portal'
			});
			successMessage = `Visitor "${visit.visitor_name}" approved for entry. Gatekeeper notified.`;
			await loadResidentData();
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to approve visitor';
		} finally {
			actingVisitId = null;
		}
	}

	async function handleConfirmReject() {
		if (!rejectTargetVisit) return;
		const visit = rejectTargetVisit;
		actingVisitId = visit.id;
		error = null;
		successMessage = null;

		try {
			await societyApi.rejectVisit(societyId, visit.id, {
				reason: rejectReason.trim() || 'Declined by resident'
			});
			successMessage = `Visitor "${visit.visitor_name}" rejected. Gatekeeper notified.`;
			rejectTargetVisit = null;
			await loadResidentData();
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to reject visitor';
		} finally {
			actingVisitId = null;
		}
	}
</script>

<div class="space-y-6">
	<!-- Resident Hub Header -->
	<div
		class="flex flex-col gap-4 rounded-3xl border border-neutral-200 bg-white/90 p-6 shadow-xs sm:flex-row sm:items-center sm:justify-between sm:p-8"
	>
		<div class="flex items-center gap-3">
			<div
				class="flex h-11 w-11 items-center justify-center rounded-2xl bg-neutral-900 text-white shadow-2xs"
			>
				<Home class="h-5 w-5" />
			</div>
			<div>
				<div class="flex items-center gap-2">
					<h2 class="text-base font-semibold tracking-tight text-neutral-950">
						Resident Visitor Hub
					</h2>
					<Badge
						variant="outline"
						class="border-emerald-200 bg-emerald-50 font-mono text-[10px] text-emerald-800"
					>
						Flat Occupant Control
					</Badge>
				</div>
				<p class="text-xs text-neutral-500">
					Control your flat's gate entry policy, approve arriving guests, and pre-register visitors.
				</p>
			</div>
		</div>

		<div class="flex items-center gap-2">
			<Button
				variant="outline"
				size="sm"
				onclick={loadResidentData}
				disabled={isLoading}
				class="h-9 px-3"
			>
				<RefreshCw class="mr-1.5 h-3.5 w-3.5 {isLoading ? 'animate-spin' : ''}" />
				<span>Refresh</span>
			</Button>

			<Button
				variant="primary"
				size="sm"
				onclick={() => (isExpectedDialogOpen = true)}
				class="h-9 px-4"
			>
				<Calendar class="mr-1.5 h-3.5 w-3.5" />
				<span>Pre-Register Visitor</span>
			</Button>
		</div>
	</div>

	<!-- Alert Messages -->
	{#if error}
		<div
			class="flex items-center gap-2 rounded-2xl border border-rose-200 bg-rose-50/80 p-4 text-xs text-rose-800"
		>
			<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
			<span>{error}</span>
		</div>
	{/if}

	{#if successMessage}
		<div
			class="flex items-center justify-between rounded-2xl border border-emerald-200 bg-emerald-50/80 p-4 text-xs text-emerald-800"
		>
			<div class="flex items-center gap-2">
				<Check class="h-4 w-4 shrink-0 text-emerald-600" />
				<span class="font-medium">{successMessage}</span>
			</div>
			<button
				type="button"
				onclick={() => (successMessage = null)}
				class="rounded-lg p-1 hover:bg-black/5"
			>
				&times;
			</button>
		</div>
	{/if}

	<!-- Section 1: My Units & Approval Policy Control -->
	<div class="rounded-3xl border border-neutral-200 bg-white/90 p-6 shadow-xs sm:p-8">
		<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
			<div>
				<h3 class="text-sm font-semibold tracking-tight text-neutral-900">
					My Authorized Flats & Visitor Policies
				</h3>
				<p class="text-xs text-neutral-500">
					Configure whether gatekeepers must obtain your explicit approval before admitting walk-in
					visitors.
				</p>
			</div>
			<Badge variant="outline" class="border-neutral-200 font-mono text-[11px] text-neutral-700">
				{units.length} Authorized {units.length === 1 ? 'Unit' : 'Units'}
			</Badge>
		</div>

		{#if isLoading && units.length === 0}
			<div class="flex items-center justify-center py-12 text-neutral-400">
				<Loader2 class="h-6 w-6 animate-spin" />
			</div>
		{:else if units.length === 0}
			<div class="py-10 text-center text-xs text-neutral-500">
				No authorized flat relationships found for this society. Contact society administration.
			</div>
		{:else}
			<div class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
				{#each units as unit (unit.id)}
					<div
						class="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-neutral-50/60 p-4 transition-all hover:border-neutral-300"
					>
						<div>
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-2">
									<div
										class="flex h-7 w-7 items-center justify-center rounded-lg bg-white font-mono text-xs font-semibold text-neutral-900 shadow-2xs"
									>
										{unit.unit_identifier}
									</div>
									<div>
										<h4 class="text-xs font-semibold text-neutral-950">
											Flat {unit.unit_identifier}
										</h4>
										<p class="text-[11px] text-neutral-500">
											Floor {unit.floor_number} &bull; {unit.relationship_type}
										</p>
									</div>
								</div>

								<Badge
									variant="outline"
									class={unit.requires_visitor_approval
										? 'border-amber-200 bg-amber-50 text-amber-800'
										: 'border-neutral-200 bg-white text-neutral-600'}
								>
									{unit.requires_visitor_approval ? 'Approval Required' : 'Direct Entry (Default)'}
								</Badge>
							</div>

							<div class="mt-4 rounded-xl border border-neutral-200 bg-white p-3 text-xs">
								<p class="font-medium text-neutral-900">
									{unit.requires_visitor_approval
										? 'Strict Security Mode Active'
										: 'Direct Entry Mode Active (Default)'}
								</p>
								<p class="mt-0.5 text-[11px] leading-relaxed text-neutral-500">
									{unit.requires_visitor_approval
										? 'Walk-in visitors will be stopped at the gate in PENDING state until you confirm arrival.'
										: 'Walk-in visitors are checked in directly at the gate by security without holding for app approval.'}
								</p>
							</div>
						</div>

						<div class="mt-4 flex items-center justify-between border-t border-neutral-100 pt-3">
							<span class="text-[11px] font-medium text-neutral-600">Require My Approval:</span>
							<button
								type="button"
								onclick={() => handleTogglePolicy(unit)}
								disabled={togglingUnitId === unit.id}
								class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-hidden disabled:opacity-50 {unit.requires_visitor_approval
									? 'bg-neutral-900'
									: 'bg-neutral-200'}"
								role="switch"
								aria-checked={unit.requires_visitor_approval}
							>
								{#if togglingUnitId === unit.id}
									<span class="absolute inset-0 flex items-center justify-center">
										<Loader2 class="h-3 w-3 animate-spin text-white" />
									</span>
								{:else}
									<span
										aria-hidden="true"
										class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow-sm ring-0 transition duration-200 ease-in-out {unit.requires_visitor_approval
											? 'translate-x-5'
											: 'translate-x-0'}"
									></span>
								{/if}
							</button>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Section 2: Incoming Visitor Approvals Queue -->
	<div class="rounded-3xl border border-neutral-200 bg-white/90 p-6 shadow-xs sm:p-8">
		<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
			<div class="flex items-center gap-2">
				<div
					class="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-100 text-amber-900"
				>
					<Bell class="h-4 w-4" />
				</div>
				<div>
					<h3 class="text-sm font-semibold tracking-tight text-neutral-900">
						Pending Visitor Approvals
					</h3>
					<p class="text-xs text-neutral-500">
						Visitors currently at the perimeter gate awaiting your admission confirmation.
					</p>
				</div>
			</div>
			<Badge
				variant="outline"
				class={pendingVisits.length > 0
					? 'border-amber-300 bg-amber-50 font-mono text-[11px] text-amber-900'
					: 'border-neutral-200 font-mono text-[11px] text-neutral-600'}
			>
				{pendingVisits.length} Pending
			</Badge>
		</div>

		{#if isLoading && pendingVisits.length === 0}
			<div class="flex items-center justify-center py-12 text-neutral-400">
				<Loader2 class="h-6 w-6 animate-spin" />
			</div>
		{:else if pendingVisits.length === 0}
			<div class="py-12 text-center text-xs text-neutral-400">
				No visitors waiting for approval right now.
			</div>
		{:else}
			<div class="mt-4 divide-y divide-neutral-100">
				{#each pendingVisits as visit (visit.id)}
					<div
						class="flex flex-col gap-3 py-4 first:pt-0 last:pb-0 sm:flex-row sm:items-center sm:justify-between"
					>
						<div>
							<div class="flex items-center gap-2">
								<h4 class="text-xs font-semibold text-neutral-950">{visit.visitor_name}</h4>
								<Badge variant="outline" class="border-neutral-200 bg-neutral-50 text-[10px]">
									{visit.purpose}
								</Badge>
								{#if visit.vehicle_number}
									<span class="font-mono text-[10px] text-neutral-700 uppercase">
										[{visit.vehicle_number}]
									</span>
								{/if}
							</div>
							<div
								class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-[11px] text-neutral-500"
							>
								<span class="font-medium text-neutral-700"
									>Destination: Flat {visit.unit_identifier || 'Your Flat'}</span
								>
								<span class="font-mono">{formatPhoneDisplay(visit.visitor_phone_masked)}</span>
								<span
									>Arrived: {new Date(visit.created_at).toLocaleTimeString([], {
										hour: '2-digit',
										minute: '2-digit'
									})}</span
								>
							</div>
						</div>

						<div class="flex items-center gap-2">
							<Button
								variant="outline"
								size="sm"
								onclick={() => (rejectTargetVisit = visit)}
								disabled={actingVisitId === visit.id}
								class="border-rose-200 text-rose-700 hover:bg-rose-50"
							>
								<UserX class="mr-1 h-3.5 w-3.5" />
								<span>Deny</span>
							</Button>
							<Button
								variant="primary"
								size="sm"
								onclick={() => handleApprove(visit)}
								disabled={actingVisitId === visit.id}
								class="bg-emerald-700 text-white hover:bg-emerald-800"
							>
								{#if actingVisitId === visit.id}
									<Loader2 class="mr-1 h-3.5 w-3.5 animate-spin" />
									<span>Processing...</span>
								{:else}
									<UserCheck class="mr-1 h-3.5 w-3.5" />
									<span>Approve Entry</span>
								{/if}
							</Button>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

<!-- Reject Confirmation Modal -->
{#if rejectTargetVisit}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-xs"
		role="dialog"
		aria-modal="true"
	>
		<div class="w-full max-w-sm rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl">
			<h4 class="text-sm font-semibold text-neutral-900">
				Deny Visitor Entry: {rejectTargetVisit.visitor_name}
			</h4>
			<p class="mt-1 text-xs text-neutral-500">
				Provide an optional reason for the gate security record.
			</p>

			<div class="mt-4">
				<label for="rej-reason" class="mb-1 block text-xs font-medium text-neutral-700">
					Reason
				</label>
				<input
					id="rej-reason"
					type="text"
					bind:value={rejectReason}
					class="w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 outline-none focus:border-neutral-900"
				/>
			</div>

			<div class="mt-6 flex justify-end gap-2">
				<Button
					variant="outline"
					size="sm"
					onclick={() => (rejectTargetVisit = null)}
					disabled={actingVisitId === rejectTargetVisit.id}
				>
					Cancel
				</Button>
				<Button
					variant="primary"
					size="sm"
					onclick={handleConfirmReject}
					disabled={actingVisitId === rejectTargetVisit.id}
					class="bg-rose-600 text-white hover:bg-rose-700"
				>
					{#if actingVisitId === rejectTargetVisit.id}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
						<span>Denying...</span>
					{:else}
						<span>Confirm Denial</span>
					{/if}
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- S09 Pre-Register Expected Visitor Dialog -->
<VisitorRegisterDialog
	{societyId}
	isOpen={isExpectedDialogOpen}
	mode="expected"
	onClose={() => (isExpectedDialogOpen = false)}
	onSuccess={(visit, pass) => {
		loadResidentData();
		if (pass) {
			activePassVisit = visit;
			activePass = pass;
			isPassDialogOpen = true;
		}
	}}
/>

<VisitorPassDialog
	isOpen={isPassDialogOpen}
	visit={activePassVisit}
	pass={activePass}
	onClose={() => {
		isPassDialogOpen = false;
		activePassVisit = null;
		activePass = null;
	}}
/>
