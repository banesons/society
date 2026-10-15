<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { societyApi, type Gate, type Visit, type VisitorPass } from '$lib/api/society';
	import { formatPhoneDisplay } from '$lib/phone';
	import {
		Search,
		LogIn,
		LogOut,
		QrCode,
		UserPlus,
		Loader2,
		AlertCircle,
		CheckCircle2,
		Clock,
		RefreshCw
	} from '@lucide/svelte';

	let {
		societyId,
		gates,
		onRegisterWalkIn,
		onRegisterExpected
	}: {
		societyId: string;
		gates: Gate[];
		onRegisterWalkIn: () => void;
		onRegisterExpected: () => void;
	} = $props();

	// Selected active gate for terminal operations
	let selectedGateId = $state<string>('');

	// Verification search
	let passCodeSearch = $state('');
	let isVerifying = $state(false);
	let verifiedPass = $state<VisitorPass | null>(null);
	let verifiedVisit = $state<Visit | null>(null);
	let verifyError = $state<string | null>(null);

	// Currently inside visitors
	let insideVisits = $state<Visit[]>([]);
	let expectedVisits = $state<Visit[]>([]);
	let isLoadingLists = $state(true);

	// Action loading states
	let checkingInId = $state<string | null>(null);
	let checkingOutId = $state<string | null>(null);
	let actionError = $state<string | null>(null);

	let selectedGate = $derived(gates.find((g) => g.id === selectedGateId) || gates[0] || null);

	$effect(() => {
		if (gates.length > 0 && !selectedGateId) {
			const activeGate = gates.find(
				(g) => g.status === 'ACTIVE' && g.operational_status === 'ACTIVE'
			);
			selectedGateId = activeGate ? activeGate.id : gates[0].id;
		}
	});

	async function loadTerminalData() {
		if (!societyId) return;
		isLoadingLists = true;
		actionError = null;

		try {
			const [insideRes, expRes] = await Promise.all([
				societyApi.listVisits(societyId, { status: 'CHECKED_IN', limit: 100 }),
				societyApi.listVisits(societyId, { status: 'APPROVED', limit: 50 })
			]);
			insideVisits = insideRes.visits;
			expectedVisits = expRes.visits;
		} catch (err: unknown) {
			console.error('Failed to load terminal lists', err);
		} finally {
			isLoadingLists = false;
		}
	}

	$effect(() => {
		if (societyId) {
			loadTerminalData();
		}
	});

	async function handleVerifyPass(e: Event) {
		e.preventDefault();
		if (!passCodeSearch.trim()) return;

		isVerifying = true;
		verifyError = null;
		verifiedPass = null;
		verifiedVisit = null;

		try {
			const res = await societyApi.verifyVisitorPass(societyId, passCodeSearch.trim());
			verifiedPass = res.pass;
			verifiedVisit = res.visit;
		} catch (err: unknown) {
			verifyError = err instanceof Error ? err.message : 'Invalid or expired pass code';
		} finally {
			isVerifying = false;
		}
	}

	async function handleCheckIn(visit: Visit) {
		if (!selectedGate) {
			actionError = 'Please select an active gate for entry.';
			return;
		}

		checkingInId = visit.id;
		actionError = null;

		try {
			await societyApi.checkInVisitor(societyId, visit.id, {
				gate_id: selectedGate.id,
				vehicle_number: visit.vehicle_number
			});
			// Reset verification if it matches
			if (verifiedVisit?.id === visit.id) {
				verifiedVisit = null;
				verifiedPass = null;
				passCodeSearch = '';
			}
			await loadTerminalData();
		} catch (err: unknown) {
			actionError = err instanceof Error ? err.message : 'Check-in failed';
		} finally {
			checkingInId = null;
		}
	}

	async function handleCheckOut(visit: Visit) {
		if (!selectedGate) {
			actionError = 'Please select an active gate for exit.';
			return;
		}

		checkingOutId = visit.id;
		actionError = null;

		try {
			await societyApi.checkOutVisitor(societyId, visit.id, {
				gate_id: selectedGate.id
			});
			await loadTerminalData();
		} catch (err: unknown) {
			actionError = err instanceof Error ? err.message : 'Check-out failed';
		} finally {
			checkingOutId = null;
		}
	}
</script>

<div class="space-y-6">
	<!-- Gate Terminal Header & Selector Bar -->
	<div
		class="flex flex-col gap-4 rounded-2xl border border-neutral-200 bg-white/90 p-4 shadow-xs sm:flex-row sm:items-center sm:justify-between"
	>
		<div class="flex items-center gap-3">
			<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-neutral-900 text-white">
				<QrCode class="h-5 w-5" />
			</div>
			<div>
				<h3 class="text-sm font-semibold text-neutral-950">Live Perimeter Gate Terminal</h3>
				<p class="text-xs text-neutral-500">
					Rapid visitor entry, pass verification, and vehicle departure logging
				</p>
			</div>
		</div>

		<div class="flex flex-wrap items-center gap-2">
			<!-- Gate Selector Dropdown -->
			<div class="flex items-center gap-2">
				<span class="text-xs font-medium text-neutral-600">Active Gate:</span>
				<select
					bind:value={selectedGateId}
					class="rounded-xl border border-neutral-200 bg-white px-3 py-1.5 text-xs font-medium text-neutral-900 shadow-2xs outline-none focus:border-neutral-900"
				>
					{#each gates as g (g.id)}
						<option value={g.id}>
							{g.name} ({g.directionality === 'BIDIRECTIONAL'
								? 'In/Out'
								: g.directionality === 'INGRESS_ONLY'
									? 'Inbound'
									: 'Outbound'})
						</option>
					{/each}
				</select>
			</div>

			<Button variant="outline" size="sm" onclick={onRegisterExpected}>
				<Clock class="mr-1.5 h-3.5 w-3.5" />
				<span>Expected Visitor</span>
			</Button>

			<Button variant="primary" size="sm" onclick={onRegisterWalkIn}>
				<UserPlus class="mr-1.5 h-3.5 w-3.5" />
				<span>Walk-In Intake</span>
			</Button>
		</div>
	</div>

	{#if actionError}
		<div
			class="flex items-center gap-2 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
		>
			<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
			<span>{actionError}</span>
		</div>
	{/if}

	<!-- Pass Verification Box -->
	<div class="rounded-2xl border border-neutral-200 bg-white/90 p-5 shadow-xs">
		<h4 class="text-xs font-semibold tracking-wider text-neutral-500 uppercase">
			Verify Visitor Pass Code
		</h4>
		<form onsubmit={handleVerifyPass} class="mt-3 flex flex-col gap-2 sm:flex-row">
			<div class="relative flex-1">
				<Search class="absolute top-2.5 left-3 h-4 w-4 text-neutral-400" />
				<input
					type="text"
					bind:value={passCodeSearch}
					placeholder="Enter Pass Code (e.g. GP-8492) or scan token..."
					class="w-full rounded-xl border border-neutral-200 bg-neutral-50/60 py-2 pr-3 pl-9 font-mono text-xs text-neutral-900 uppercase outline-none placeholder:normal-case focus:border-neutral-900 focus:bg-white"
				/>
			</div>

			<Button
				variant="primary"
				size="sm"
				type="submit"
				disabled={isVerifying || !passCodeSearch.trim()}
			>
				{#if isVerifying}
					<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					<span>Verifying...</span>
				{:else}
					<span>Verify Pass</span>
				{/if}
			</Button>
		</form>

		{#if verifyError}
			<div
				class="mt-3 flex items-center gap-2 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
			>
				<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
				<span>{verifyError}</span>
			</div>
		{/if}

		{#if verifiedVisit}
			<div
				class="mt-4 flex flex-col gap-4 rounded-xl border border-emerald-200 bg-emerald-50/50 p-4 sm:flex-row sm:items-center sm:justify-between"
			>
				<div class="space-y-1">
					<div class="flex items-center gap-2">
						<CheckCircle2 class="h-4 w-4 text-emerald-600" />
						<span class="text-sm font-semibold text-neutral-950">{verifiedVisit.visitor_name}</span>
						{#if verifiedPass}
							<Badge variant="outline" class="font-mono text-[10px]">{verifiedPass.pass_code}</Badge
							>
						{/if}
						<Badge variant="success" class="text-[10px]">Pass Verified</Badge>
					</div>
					<div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-neutral-600">
						<span
							>Flat: <strong class="font-mono text-neutral-900"
								>{verifiedVisit.building_name} — {verifiedVisit.unit_identifier}</strong
							></span
						>
						<span
							>Phone: <strong class="font-mono text-neutral-900"
								>{formatPhoneDisplay(verifiedVisit.visitor_phone_masked)}</strong
							></span
						>
						<span>Purpose: <strong class="text-neutral-900">{verifiedVisit.purpose}</strong></span>
						{#if verifiedVisit.vehicle_number}
							<span
								>Vehicle: <strong class="font-mono text-neutral-900"
									>{verifiedVisit.vehicle_number}</strong
								></span
							>
						{/if}
					</div>
				</div>

				<Button
					variant="primary"
					size="sm"
					onclick={() => verifiedVisit && handleCheckIn(verifiedVisit)}
					disabled={checkingInId === verifiedVisit.id}
				>
					{#if checkingInId === verifiedVisit.id}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
						<span>Checking In...</span>
					{:else}
						<LogIn class="mr-1.5 h-3.5 w-3.5" />
						<span>Admit & Check In</span>
					{/if}
				</Button>
			</div>
		{/if}
	</div>

	<!-- Dual Columns: Currently Inside & Expected Arrivals -->
	<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
		<!-- Active Visitors Currently Inside -->
		<div class="rounded-2xl border border-neutral-200 bg-white/90 p-5 shadow-xs">
			<div class="flex items-center justify-between border-b border-neutral-100 pb-3">
				<div class="flex items-center gap-2">
					<div class="h-2 w-2 animate-pulse rounded-full bg-emerald-500"></div>
					<h4 class="text-sm font-semibold text-neutral-950">Currently Inside Society</h4>
					<Badge variant="outline" class="font-mono text-[10px]">{insideVisits.length}</Badge>
				</div>
				<button
					type="button"
					onclick={loadTerminalData}
					class="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700"
					title="Refresh"
				>
					<RefreshCw class="h-3.5 w-3.5" />
				</button>
			</div>

			<div class="mt-4 space-y-2.5">
				{#if isLoadingLists}
					<div class="flex h-32 items-center justify-center text-xs text-neutral-500">
						<Loader2 class="h-5 w-5 animate-spin text-neutral-900" />
					</div>
				{:else if insideVisits.length === 0}
					<div
						class="flex h-32 flex-col items-center justify-center rounded-xl border border-dashed border-neutral-200 text-center text-xs text-neutral-500"
					>
						<span>No visitors currently inside society perimeter</span>
					</div>
				{:else}
					{#each insideVisits as v (v.id)}
						<div
							class="flex flex-col gap-2 rounded-xl border border-neutral-100 bg-neutral-50/60 p-3 sm:flex-row sm:items-center sm:justify-between"
						>
							<div class="space-y-0.5">
								<div class="flex items-center gap-2">
									<span class="text-xs font-semibold text-neutral-900">{v.visitor_name}</span>
									<span class="font-mono text-[11px] text-neutral-500"
										>({v.building_name} — {v.unit_identifier})</span
									>
								</div>
								<div class="flex items-center gap-3 text-[11px] text-neutral-500">
									<span
										>In: {v.check_in_at
											? new Date(v.check_in_at).toLocaleTimeString([], {
													hour: '2-digit',
													minute: '2-digit'
												})
											: '—'}</span
									>
									{#if v.vehicle_number}
										<span class="font-mono uppercase">{v.vehicle_number}</span>
									{/if}
									<span>{v.purpose}</span>
								</div>
							</div>

							<Button
								variant="outline"
								size="sm"
								onclick={() => handleCheckOut(v)}
								disabled={checkingOutId === v.id}
								class="text-xs"
							>
								{#if checkingOutId === v.id}
									<Loader2 class="mr-1 h-3 w-3 animate-spin" />
									<span>Processing...</span>
								{:else}
									<LogOut class="mr-1 h-3 w-3 text-neutral-600" />
									<span>Check Out</span>
								{/if}
							</Button>
						</div>
					{/each}
				{/if}
			</div>
		</div>

		<!-- Approved & Expected Arrivals Today -->
		<div class="rounded-2xl border border-neutral-200 bg-white/90 p-5 shadow-xs">
			<div class="flex items-center justify-between border-b border-neutral-100 pb-3">
				<div class="flex items-center gap-2">
					<Clock class="h-4 w-4 text-neutral-600" />
					<h4 class="text-sm font-semibold text-neutral-950">Approved Arrivals</h4>
					<Badge variant="outline" class="font-mono text-[10px]">{expectedVisits.length}</Badge>
				</div>
				<button
					type="button"
					onclick={loadTerminalData}
					class="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700"
					title="Refresh"
				>
					<RefreshCw class="h-3.5 w-3.5" />
				</button>
			</div>

			<div class="mt-4 space-y-2.5">
				{#if isLoadingLists}
					<div class="flex h-32 items-center justify-center text-xs text-neutral-500">
						<Loader2 class="h-5 w-5 animate-spin text-neutral-900" />
					</div>
				{:else if expectedVisits.length === 0}
					<div
						class="flex h-32 flex-col items-center justify-center rounded-xl border border-dashed border-neutral-200 text-center text-xs text-neutral-500"
					>
						<span>No approved visitors awaiting check-in</span>
					</div>
				{:else}
					{#each expectedVisits as ev (ev.id)}
						<div
							class="flex flex-col gap-2 rounded-xl border border-neutral-100 bg-neutral-50/60 p-3 sm:flex-row sm:items-center sm:justify-between"
						>
							<div class="space-y-0.5">
								<div class="flex items-center gap-2">
									<span class="text-xs font-semibold text-neutral-900">{ev.visitor_name}</span>
									<span class="font-mono text-[11px] text-neutral-500"
										>({ev.building_name} — {ev.unit_identifier})</span
									>
									<Badge variant="outline" class="font-mono text-[9px] uppercase"
										>{ev.visit_type}</Badge
									>
								</div>
								<div
									class="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-[11px] text-neutral-500"
								>
									<span
										>{ev.expected_arrival
											? new Date(ev.expected_arrival).toLocaleTimeString([], {
													hour: '2-digit',
													minute: '2-digit'
												})
											: 'Approved'}</span
									>
									<span class="font-mono">{formatPhoneDisplay(ev.visitor_phone_masked)}</span>
									<span>{ev.purpose}</span>
									{#if ev.vehicle_number}
										<span class="font-mono text-[10px] text-neutral-700 uppercase"
											>[{ev.vehicle_number}]</span
										>
									{/if}
								</div>
							</div>

							<Button
								variant="primary"
								size="sm"
								onclick={() => handleCheckIn(ev)}
								disabled={checkingInId === ev.id}
								class="text-xs"
							>
								{#if checkingInId === ev.id}
									<Loader2 class="mr-1 h-3 w-3 animate-spin" />
									<span>Admitting...</span>
								{:else}
									<LogIn class="mr-1 h-3 w-3" />
									<span>Check In</span>
								{/if}
							</Button>
						</div>
					{/each}
				{/if}
			</div>
		</div>
	</div>
</div>
