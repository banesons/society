<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { societyApi, type CommitteeAppointment, type CommitteeOverview } from '$lib/api/society';
	import { auth } from '$lib/auth.svelte';
	import { ApiClientError } from '$lib/api/client';
	import AppointMemberDialog from './appoint-member-dialog.svelte';
	import EndAppointmentDialog from './end-appointment-dialog.svelte';
	import DesignationsDialog from './designations-dialog.svelte';
	import {
		Award,
		UserPlus,
		Layers,
		Loader2,
		AlertCircle,
		UserMinus,
		ShieldCheck
	} from '@lucide/svelte';

	let { societyId }: { societyId: string } = $props();

	let overview = $state<CommitteeOverview | null>(null);
	let appointments = $state<CommitteeAppointment[]>([]);
	let isLoading = $state(false);
	let error = $state<string | null>(null);

	// Status filter: 'ALL' | 'ACTIVE' | 'ENDED' | 'REVOKED'
	const statusOptions = ['ALL', 'ACTIVE', 'ENDED', 'REVOKED'] as const;
	let statusFilter = $state<(typeof statusOptions)[number]>('ACTIVE');

	// Dialog controls
	let isAppointOpen = $state(false);
	let isDesignationsOpen = $state(false);
	let endingAppointment = $state<CommitteeAppointment | null>(null);
	let isEndOpen = $state(false);

	let displayOfficeBearers = $derived.by(() => {
		if (overview?.office_bearers && overview.office_bearers.length > 0) {
			return overview.office_bearers;
		}
		// Fallback derivation if overview endpoint failed or returned empty office_bearers list
		const canonical = [
			{ code: 'CHAIRMAN', name: 'Chairman' },
			{ code: 'SECRETARY', name: 'Secretary' },
			{ code: 'TREASURER', name: 'Treasurer' }
		];
		return canonical.map((c) => {
			const apt = appointments.find((a) => a.designation_code === c.code && a.status === 'ACTIVE');
			return {
				designation_id: apt?.designation_id || c.code.toLowerCase(),
				designation_code: c.code,
				designation_name: c.name,
				appointment_id: apt?.id,
				member_id: apt?.member_id,
				member_name: apt?.member_name,
				member_phone_masked: apt?.member_phone || apt?.member_phone_masked,
				start_date: apt?.start_date,
				expected_end_date: apt?.expected_end_date,
				status: apt?.status
			};
		});
	});

	async function loadCommitteeData() {
		if (!societyId) return;
		overview = null;
		appointments = [];
		isLoading = true;
		error = null;

		const [overviewRes, appointmentsRes] = await Promise.allSettled([
			societyApi.getCommitteeOverview(societyId),
			societyApi.listAppointments(societyId, {
				status: statusFilter === 'ALL' ? undefined : statusFilter
			})
		]);

		if (overviewRes.status === 'fulfilled') {
			overview = overviewRes.value;
		}

		if (appointmentsRes.status === 'fulfilled') {
			appointments = appointmentsRes.value.appointments || [];
		}

		// Error handling: only show error alert if appointments request failed
		// or if both failed. Overview failures degrade gracefully into derived cards.
		if (appointmentsRes.status === 'rejected') {
			const err = appointmentsRes.reason;
			if (err instanceof ApiClientError) {
				error = err.message || 'Failed to load committee appointments.';
			} else {
				error = 'Failed to load committee appointments.';
			}
		} else if (overviewRes.status === 'rejected') {
			const err = overviewRes.reason;
			if (err instanceof ApiClientError && err.statusCode !== 404) {
				error = err.message || 'Failed to load committee overview.';
			}
		}

		isLoading = false;
	}

	$effect(() => {
		if (societyId && !auth.isLoading) {
			loadCommitteeData();
		}
	});

	function handleFilterChange(newStatus: (typeof statusOptions)[number]) {
		statusFilter = newStatus;
		loadCommitteeData();
	}

	function getStatusBadgeVariant(
		status: string
	): 'default' | 'success' | 'warning' | 'destructive' {
		switch (status) {
			case 'ACTIVE':
				return 'success';
			case 'ENDED':
				return 'default';
			case 'REVOKED':
				return 'destructive';
			default:
				return 'warning';
		}
	}
</script>

<div class="mt-6 rounded-2xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-xs">
	<!-- Top Bar -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2">
				<div
					class="flex h-8 w-8 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
				>
					<Award class="h-4 w-4" />
				</div>
				<h2 class="text-base font-semibold tracking-tight text-neutral-950">
					Managing Committee & Governance
				</h2>
				<Badge
					variant="outline"
					class="border-emerald-200 bg-emerald-50 font-mono text-[10px] text-emerald-800"
					>S07 Active</Badge
				>
			</div>
			<p class="mt-1 text-xs text-neutral-500">
				Society-specific office-bearers, committee appointments, and historical governance records.
			</p>
		</div>

		<div class="flex items-center gap-2.5">
			<Button variant="outline" size="sm" onclick={() => (isDesignationsOpen = true)}>
				<Layers class="mr-1.5 h-3.5 w-3.5" />
				<span>Designations Catalogue</span>
			</Button>

			<Button variant="primary" size="sm" onclick={() => (isAppointOpen = true)}>
				<UserPlus class="mr-1.5 h-3.5 w-3.5" />
				<span>Appoint Member</span>
			</Button>
		</div>
	</div>

	<!-- Error Alert -->
	{#if error}
		<div
			class="mt-4 flex items-start gap-2.5 rounded-2xl border border-rose-200 bg-rose-50 p-3.5 text-xs text-rose-800"
		>
			<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
			<span>{error}</span>
		</div>
	{/if}

	<!-- Office-Bearers Quick Overview Cards -->
	<div class="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
		{#if displayOfficeBearers.length > 0}
			{#each displayOfficeBearers as bearer (bearer.designation_id)}
				<div
					class="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-neutral-50/70 p-4"
				>
					<div>
						<div class="flex items-center justify-between">
							<span class="text-xs font-semibold text-neutral-900">
								{bearer.designation_name}
							</span>
							{#if bearer.member_id}
								<Badge
									variant={getStatusBadgeVariant(bearer.status || 'ACTIVE')}
									class="text-[10px]"
								>
									{bearer.status || 'ACTIVE'}
								</Badge>
							{:else}
								<Badge variant="outline" class="bg-white text-[10px] text-neutral-500">VACANT</Badge
								>
							{/if}
						</div>

						<div class="mt-3">
							{#if bearer.member_id}
								<p class="text-sm font-bold text-neutral-950">{bearer.member_name}</p>
								<p class="mt-0.5 font-mono text-xs text-neutral-600">
									{bearer.member_phone_masked}
								</p>
								<p class="mt-2 text-[11px] text-neutral-500">
									Appointed: <strong class="text-neutral-800">{bearer.start_date}</strong>
								</p>
								{#if bearer.expected_end_date}
									<p class="mt-0.5 text-[11px] text-neutral-500">
										Expected End: <strong class="text-neutral-800"
											>{bearer.expected_end_date}</strong
										>
									</p>
								{/if}
							{:else}
								<p class="text-xs text-neutral-500 italic">No active appointment recorded</p>
								<button
									type="button"
									onclick={() => (isAppointOpen = true)}
									class="mt-3 inline-flex items-center gap-1 text-[11px] font-medium text-neutral-900 underline underline-offset-2 transition-colors hover:text-black"
								>
									<UserPlus class="h-3 w-3" />
									<span>Appoint to {bearer.designation_name}</span>
								</button>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		{:else if isLoading}
			<div class="col-span-3 flex min-h-[100px] items-center justify-center text-neutral-500">
				<Loader2 class="h-5 w-5 animate-spin text-neutral-900" />
			</div>
		{/if}
	</div>

	<!-- Status Filter Tabs -->
	<div class="mt-6 flex items-center justify-between border-b border-neutral-100 pb-3">
		<div class="flex items-center gap-1 rounded-full border border-neutral-200 bg-neutral-100 p-1">
			{#each statusOptions as opt (opt)}
				<button
					type="button"
					onclick={() => handleFilterChange(opt)}
					class="rounded-full px-3 py-1 text-xs font-medium transition-colors {statusFilter === opt
						? 'bg-neutral-950 text-white shadow-xs'
						: 'text-neutral-600 hover:text-neutral-900'}"
				>
					{opt === 'ALL' ? 'All Appointments' : opt}
				</button>
			{/each}
		</div>

		<span class="text-xs text-neutral-500">
			Showing <strong class="text-neutral-900">{appointments.length}</strong>
			record{appointments.length === 1 ? '' : 's'}
		</span>
	</div>

	<!-- Appointments Table -->
	{#if isLoading}
		<div class="flex min-h-[200px] flex-col items-center justify-center text-neutral-500">
			<Loader2 class="h-6 w-6 animate-spin text-neutral-900" />
			<p class="mt-2 text-xs">Loading committee appointments...</p>
		</div>
	{:else if appointments.length === 0}
		<div
			class="mt-4 rounded-2xl border border-dashed border-neutral-200 bg-neutral-50/50 p-8 text-center text-neutral-500"
		>
			<ShieldCheck class="mx-auto h-8 w-8 text-neutral-400" />
			<p class="mt-2 text-xs font-semibold text-neutral-800">
				No {statusFilter === 'ALL' ? '' : statusFilter.toLowerCase()} appointments found
			</p>
			<p class="mt-1 text-[11px] text-neutral-500">
				Appoint an active member to assign them a committee designation.
			</p>
		</div>
	{:else}
		<div class="mt-4 overflow-x-auto rounded-2xl border border-neutral-200 bg-white shadow-xs">
			<table class="w-full text-left text-xs">
				<thead
					class="border-b border-neutral-200 bg-neutral-50/80 text-[11px] font-semibold tracking-wider text-neutral-500 uppercase"
				>
					<tr>
						<th class="px-4 py-3">Designation</th>
						<th class="px-4 py-3">Member</th>
						<th class="px-4 py-3">Term Duration</th>
						<th class="px-4 py-3">Status</th>
						<th class="px-4 py-3">Resolution Notes</th>
						<th class="px-4 py-3 text-right">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-neutral-100 text-neutral-800">
					{#each appointments as appt (appt.id)}
						<tr class="transition-colors hover:bg-neutral-50/60">
							<!-- Designation -->
							<td class="px-4 py-3 font-medium text-neutral-950">
								<div class="flex items-center gap-2">
									<span>{appt.designation_name || appt.designation_code}</span>
									<Badge
										variant="outline"
										class="bg-neutral-100 font-mono text-[9px] text-neutral-600"
									>
										{appt.designation_code}
									</Badge>
								</div>
							</td>

							<!-- Member -->
							<td class="px-4 py-3">
								<div class="font-medium text-neutral-900">{appt.member_name}</div>
								<div class="font-mono text-[11px] text-neutral-500">
									{appt.member_phone || appt.member_phone_masked || ''}
								</div>
							</td>

							<!-- Term Duration -->
							<td class="px-4 py-3 font-mono text-[11px] text-neutral-700">
								<span>{appt.start_date}</span>
								<span class="text-neutral-400"> → </span>
								<span
									>{appt.end_date
										? appt.end_date
										: appt.expected_end_date
											? appt.expected_end_date
											: 'Present'}</span
								>
							</td>

							<!-- Status -->
							<td class="px-4 py-3">
								<Badge variant={getStatusBadgeVariant(appt.status)} class="text-[10px]">
									{appt.status}
								</Badge>
							</td>

							<!-- Resolution Notes -->
							<td class="max-w-[200px] truncate px-4 py-3 text-[11px] text-neutral-600">
								{appt.notes || '—'}
							</td>

							<!-- Actions -->
							<td class="px-4 py-3 text-right">
								{#if appt.status === 'ACTIVE'}
									<Button
										variant="ghost"
										size="sm"
										class="text-xs text-amber-700 hover:bg-amber-50 hover:text-amber-800"
										onclick={() => {
											endingAppointment = appt;
											isEndOpen = true;
										}}
									>
										<UserMinus class="mr-1 h-3.5 w-3.5" />
										<span>End / Revoke</span>
									</Button>
								{:else}
									<span class="text-[11px] text-neutral-400 italic">Term concluded</span>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<!-- Dialogs -->
<AppointMemberDialog
	{societyId}
	isOpen={isAppointOpen}
	onClose={() => (isAppointOpen = false)}
	onSuccess={() => {
		loadCommitteeData();
	}}
/>

<EndAppointmentDialog
	{societyId}
	appointment={endingAppointment}
	isOpen={isEndOpen}
	onClose={() => {
		isEndOpen = false;
		endingAppointment = null;
	}}
	onSuccess={() => {
		loadCommitteeData();
	}}
/>

<DesignationsDialog
	{societyId}
	isOpen={isDesignationsOpen}
	onClose={() => (isDesignationsOpen = false)}
	onUpdated={() => {
		loadCommitteeData();
	}}
/>
