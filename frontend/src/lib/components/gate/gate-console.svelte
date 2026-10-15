<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import {
		societyApi,
		type Gate,
		type SocietyAdminAssignment,
		type GateShift,
		type GateDutySession,
		type VisitorDashboardMetrics,
		type Visit,
		type VisitorPass
	} from '$lib/api/society';
	import { formatPhoneDisplay } from '$lib/phone';
	import EditGateDialog from './edit-gate-dialog.svelte';
	import AssignGatekeeperDialog from './assign-gatekeeper-dialog.svelte';
	import CreateShiftDialog from './create-shift-dialog.svelte';
	import CheckInDialog from './check-in-dialog.svelte';
	import CheckOutDialog from './check-out-dialog.svelte';
	import VisitorTerminal from './visitor-terminal.svelte';
	import VisitorApprovalsQueue from './visitor-approvals-queue.svelte';
	import VisitorHistory from './visitor-history.svelte';
	import VisitorRegisterDialog from './visitor-register-dialog.svelte';
	import VisitorPassDialog from './visitor-pass-dialog.svelte';
	import VisitDetailsDialog from './visit-details-dialog.svelte';
	import {
		Shield,
		Sliders,
		Users,
		Clock,
		Radio,
		UserPlus,
		LogIn,
		LogOut,
		AlertCircle,
		Loader2,
		QrCode,
		CheckSquare,
		History
	} from '@lucide/svelte';

	let { societyId }: { societyId: string } = $props();

	type Tab = 'terminal' | 'approvals' | 'history' | 'duty' | 'gates' | 'roster';
	let activeTab = $state<Tab>('terminal');

	let gates = $state<Gate[]>([]);
	let gatekeepers = $state<SocietyAdminAssignment[]>([]);
	let shifts = $state<GateShift[]>([]);
	let sessions = $state<GateDutySession[]>([]);
	let metrics = $state<VisitorDashboardMetrics | null>(null);

	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let statusBanner = $state<{ type: 'success' | 'info'; message: string } | null>(null);

	// Dialog states
	let selectedGateForEdit = $state<Gate | null>(null);
	let isEditGateOpen = $state(false);

	let isAssignGatekeeperOpen = $state(false);
	let isCreateShiftOpen = $state(false);

	let isCheckInOpen = $state(false);
	let selectedSessionForCheckOut = $state<GateDutySession | null>(null);
	let isCheckOutOpen = $state(false);

	let isRegisterOpen = $state(false);
	let registerMode = $state<'walk_in' | 'expected'>('walk_in');

	let isPassDialogOpen = $state(false);
	let activePassVisit = $state<Visit | null>(null);
	let activePass = $state<VisitorPass | null>(null);

	let isDetailsDialogOpen = $state(false);
	let selectedVisitForDetails = $state<Visit | null>(null);

	let revokingId = $state<string | null>(null);

	async function loadAllData() {
		if (!societyId) return;
		gates = [];
		gatekeepers = [];
		shifts = [];
		sessions = [];
		metrics = null;
		isLoading = true;
		error = null;

		try {
			const [gatesRes, gkRes, shiftsRes, sessRes, metricsRes] = await Promise.all([
				societyApi.listGates(societyId),
				societyApi.listGatekeepers(societyId, 'ALL'),
				societyApi.listShifts(societyId),
				societyApi.listDutySessions(societyId, { status: 'ALL' }),
				societyApi.getVisitorDashboard(societyId).catch(() => null)
			]);

			gates = gatesRes.gates;
			gatekeepers = gkRes.gatekeepers;
			shifts = shiftsRes.shifts;
			sessions = sessRes.sessions;
			metrics = metricsRes;
		} catch (err: unknown) {
			if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Failed to load gate management data';
			}
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (societyId) {
			loadAllData();
		}
	});

	async function handleRevokeGatekeeper(assignment: SocietyAdminAssignment) {
		if (
			!confirm(
				`Revoke gatekeeper access for ${formatPhoneDisplay(assignment.user_phone_masked)}? Active duty sessions will be ended immediately.`
			)
		) {
			return;
		}

		revokingId = assignment.id;
		try {
			await societyApi.revokeGatekeeper(societyId, assignment.id);
			await loadAllData();
		} catch (err: unknown) {
			alert(err instanceof Error ? err.message : 'Failed to revoke gatekeeper');
		} finally {
			revokingId = null;
		}
	}

	function getDirectionBadge(dir: string): { label: string; class: string } {
		switch (dir) {
			case 'INGRESS_ONLY':
				return { label: 'Inbound Only', class: 'border-blue-200 bg-blue-50 text-blue-800' };
			case 'EGRESS_ONLY':
				return { label: 'Outbound Only', class: 'border-amber-200 bg-amber-50 text-amber-800' };
			default:
				return {
					label: 'Bidirectional',
					class: 'border-neutral-200 bg-neutral-100 text-neutral-800'
				};
		}
	}

	function getOpStatusBadge(status: string): { label: string; class: string } {
		switch (status) {
			case 'ACTIVE':
				return { label: 'Open', class: 'border-emerald-200 bg-emerald-50 text-emerald-800' };
			case 'INACTIVE':
				return { label: 'Closed', class: 'border-neutral-200 bg-neutral-100 text-neutral-700' };
			case 'MAINTENANCE':
				return { label: 'Maintenance', class: 'border-amber-200 bg-amber-50 text-amber-800' };
			case 'EMERGENCY_OPEN':
				return { label: 'Emergency Open', class: 'border-rose-200 bg-rose-50 text-rose-800' };
			default:
				return { label: status, class: 'border-neutral-200 bg-neutral-100 text-neutral-700' };
		}
	}
</script>

<div class="mt-8 rounded-3xl border border-neutral-200 bg-white/90 p-6 shadow-xs sm:p-8">
	<!-- Section Header -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2">
				<div
					class="flex h-8 w-8 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
				>
					<Shield class="h-4 w-4" />
				</div>
				<h2 class="text-base font-semibold tracking-tight text-neutral-950">
					Gate & Visitor Operations
				</h2>
				<Badge
					variant="outline"
					class="border-emerald-200 bg-emerald-50 font-mono text-[10px] text-emerald-800"
				>
					S09 Operational
				</Badge>
			</div>
			<p class="mt-1 text-xs text-neutral-500">
				Live gatekeeper terminal, visitor intake, resident approvals, and security staffing.
			</p>
		</div>

		<!-- Action Buttons -->
		<div class="flex flex-wrap items-center gap-2">
			{#if activeTab === 'gates'}
				<Badge variant="outline" class="border-neutral-200 text-xs text-neutral-600">
					{gates.length} Gates Configured
				</Badge>
			{:else if activeTab === 'roster'}
				<Button variant="primary" size="sm" onclick={() => (isAssignGatekeeperOpen = true)}>
					<UserPlus class="mr-1.5 h-3.5 w-3.5" />
					<span>Assign Gatekeeper</span>
				</Button>
			{:else if activeTab === 'duty'}
				<Button variant="outline" size="sm" onclick={() => (isCreateShiftOpen = true)}>
					<Clock class="mr-1.5 h-3.5 w-3.5" />
					<span>Add Shift Schedule</span>
				</Button>
				<Button variant="primary" size="sm" onclick={() => (isCheckInOpen = true)}>
					<LogIn class="mr-1.5 h-3.5 w-3.5" />
					<span>Check In to Duty</span>
				</Button>
			{:else if activeTab === 'terminal'}
				<Button
					variant="outline"
					size="sm"
					onclick={() => {
						registerMode = 'expected';
						isRegisterOpen = true;
					}}
				>
					<Clock class="mr-1.5 h-3.5 w-3.5" />
					<span>Expected Visitor</span>
				</Button>
				<Button
					variant="primary"
					size="sm"
					onclick={() => {
						registerMode = 'walk_in';
						isRegisterOpen = true;
					}}
				>
					<UserPlus class="mr-1.5 h-3.5 w-3.5" />
					<span>Walk-In Intake</span>
				</Button>
			{/if}
		</div>
	</div>

	{#if statusBanner}
		<div
			class="mt-4 flex items-center justify-between rounded-2xl border p-4 text-xs {statusBanner.type ===
			'info'
				? 'border-amber-200 bg-amber-50/80 text-amber-900'
				: 'border-emerald-200 bg-emerald-50/80 text-emerald-900'}"
		>
			<div class="flex items-center gap-2">
				<AlertCircle class="h-4 w-4 shrink-0" />
				<span class="font-medium">{statusBanner.message}</span>
			</div>
			<button
				type="button"
				onclick={() => (statusBanner = null)}
				class="rounded-lg p-1 hover:bg-black/5"
			>
				&times;
			</button>
		</div>
	{/if}

	<!-- Operational Overview Stat Cards (Derived from Authoritative Backend Metrics) -->
	<div class="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-5">
		<div class="rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5">
			<span class="text-[11px] font-medium text-neutral-500">Approved Arrivals</span>
			<p class="mt-1 text-xl font-bold tracking-tight text-neutral-950">
				{metrics?.expected_today ?? 0}
			</p>
			<span class="text-[10px] text-neutral-400">Ready for gate check-in</span>
		</div>

		<div class="rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5">
			<span class="text-[11px] font-medium text-neutral-500">Awaiting Approval</span>
			<p class="mt-1 text-xl font-bold tracking-tight text-amber-700">
				{metrics?.awaiting_approval ?? 0}
			</p>
			<span class="text-[10px] text-neutral-400">Pending flat resident</span>
		</div>

		<div class="rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5">
			<span class="text-[11px] font-medium text-neutral-500">Currently Inside</span>
			<p class="mt-1 text-xl font-bold tracking-tight text-emerald-800">
				{metrics?.currently_inside ?? 0}
			</p>
			<span class="text-[10px] text-neutral-400">Active perimeter passes</span>
		</div>

		<div class="rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5">
			<span class="text-[11px] font-medium text-neutral-500">Completed Today</span>
			<p class="mt-1 text-xl font-bold tracking-tight text-neutral-950">
				{metrics?.completed_today ?? 0}
			</p>
			<span class="text-[10px] text-neutral-400">Departed through gates</span>
		</div>

		<div class="rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5">
			<span class="text-[11px] font-medium text-neutral-500">Guards on Duty</span>
			<p class="mt-1 text-xl font-bold tracking-tight text-neutral-950">
				{sessions.filter((s) => s.status === 'ACTIVE').length}
			</p>
			<span class="text-[10px] text-neutral-400">Across {gates.length} gates</span>
		</div>
	</div>

	<!-- Tab Switcher -->
	<div class="mt-6 flex overflow-x-auto border-b border-neutral-200 text-xs font-medium">
		<button
			type="button"
			class="flex items-center gap-2 border-b-2 px-4 py-2.5 whitespace-nowrap transition-colors {activeTab ===
			'terminal'
				? 'border-neutral-900 font-semibold text-neutral-900'
				: 'border-transparent text-neutral-500 hover:text-neutral-800'}"
			onclick={() => (activeTab = 'terminal')}
		>
			<QrCode class="h-3.5 w-3.5" />
			<span>Live Gate Terminal</span>
		</button>

		<button
			type="button"
			class="flex items-center gap-2 border-b-2 px-4 py-2.5 whitespace-nowrap transition-colors {activeTab ===
			'approvals'
				? 'border-neutral-900 font-semibold text-neutral-900'
				: 'border-transparent text-neutral-500 hover:text-neutral-800'}"
			onclick={() => (activeTab = 'approvals')}
		>
			<CheckSquare class="h-3.5 w-3.5" />
			<span>Approval Queue ({metrics?.awaiting_approval ?? 0})</span>
		</button>

		<button
			type="button"
			class="flex items-center gap-2 border-b-2 px-4 py-2.5 whitespace-nowrap transition-colors {activeTab ===
			'history'
				? 'border-neutral-900 font-semibold text-neutral-900'
				: 'border-transparent text-neutral-500 hover:text-neutral-800'}"
			onclick={() => (activeTab = 'history')}
		>
			<History class="h-3.5 w-3.5" />
			<span>Visitor Log & History</span>
		</button>

		<button
			type="button"
			class="flex items-center gap-2 border-b-2 px-4 py-2.5 whitespace-nowrap transition-colors {activeTab ===
			'duty'
				? 'border-neutral-900 font-semibold text-neutral-900'
				: 'border-transparent text-neutral-500 hover:text-neutral-800'}"
			onclick={() => (activeTab = 'duty')}
		>
			<Radio class="h-3.5 w-3.5" />
			<span
				>Security Staff Duty ({sessions.filter((s) => s.status === 'ACTIVE').length} On Duty)</span
			>
		</button>

		<button
			type="button"
			class="flex items-center gap-2 border-b-2 px-4 py-2.5 whitespace-nowrap transition-colors {activeTab ===
			'gates'
				? 'border-neutral-900 font-semibold text-neutral-900'
				: 'border-transparent text-neutral-500 hover:text-neutral-800'}"
			onclick={() => (activeTab = 'gates')}
		>
			<Sliders class="h-3.5 w-3.5" />
			<span>Physical Gates ({gates.length})</span>
		</button>

		<button
			type="button"
			class="flex items-center gap-2 border-b-2 px-4 py-2.5 whitespace-nowrap transition-colors {activeTab ===
			'roster'
				? 'border-neutral-900 font-semibold text-neutral-900'
				: 'border-transparent text-neutral-500 hover:text-neutral-800'}"
			onclick={() => (activeTab = 'roster')}
		>
			<Users class="h-3.5 w-3.5" />
			<span>Guard Staff Roster ({gatekeepers.filter((g) => g.status === 'ACTIVE').length})</span>
		</button>
	</div>

	<!-- Content Area -->
	<div class="mt-6">
		{#if isLoading}
			<div class="flex items-center justify-center py-12 text-xs text-neutral-500">
				<Loader2 class="mr-2 h-4 w-4 animate-spin text-neutral-900" />
				<span>Loading gate operations data...</span>
			</div>
		{:else if error}
			<div
				class="flex items-start gap-2 rounded-2xl border border-rose-200 bg-rose-50/80 p-4 text-xs text-rose-800"
			>
				<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
				<div>
					<p class="font-semibold text-rose-900">Perimeter Access Notice</p>
					<p class="mt-0.5">{error}</p>
				</div>
			</div>
		{:else if activeTab === 'terminal'}
			<!-- Tab 1: Live Gate Terminal -->
			<VisitorTerminal
				{societyId}
				{gates}
				onRegisterWalkIn={() => {
					registerMode = 'walk_in';
					isRegisterOpen = true;
				}}
				onRegisterExpected={() => {
					registerMode = 'expected';
					isRegisterOpen = true;
				}}
			/>
		{:else if activeTab === 'approvals'}
			<!-- Tab 2: Resident Approvals Queue -->
			<VisitorApprovalsQueue
				{societyId}
				onDecisionMade={() => {
					loadAllData();
				}}
			/>
		{:else if activeTab === 'history'}
			<!-- Tab 3: Visitor Log & History -->
			<VisitorHistory
				{societyId}
				onSelectVisit={(v) => {
					selectedVisitForDetails = v;
					isDetailsDialogOpen = true;
				}}
			/>
		{:else if activeTab === 'duty'}
			<!-- Tab 4: Duty Sessions & Shift Terminal (S08) -->
			<div class="space-y-6">
				<!-- Live Posts Summary -->
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
					<div class="rounded-2xl border border-neutral-200 bg-neutral-50/70 p-4">
						<span class="text-[11px] font-medium text-neutral-500">Active Guards on Duty</span>
						<p class="mt-1 text-2xl font-bold tracking-tight text-neutral-900">
							{sessions.filter((s) => s.status === 'ACTIVE').length}
						</p>
						<p class="mt-0.5 text-[11px] text-neutral-500">Across {gates.length} perimeter gates</p>
					</div>

					<div class="rounded-2xl border border-neutral-200 bg-neutral-50/70 p-4">
						<span class="text-[11px] font-medium text-neutral-500">Configured Shifts</span>
						<p class="mt-1 text-2xl font-bold tracking-tight text-neutral-900">
							{shifts.length}
						</p>
						<p class="mt-0.5 text-[11px] text-neutral-500">Daily template schedules</p>
					</div>

					<div class="rounded-2xl border border-neutral-200 bg-neutral-50/70 p-4">
						<span class="text-[11px] font-medium text-neutral-500">Staff Headcount Need</span>
						<p class="mt-1 text-2xl font-bold tracking-tight text-neutral-900">
							{gates.reduce((acc, g) => acc + (g.required_staff_per_shift || 0), 0)}
						</p>
						<p class="mt-0.5 text-[11px] text-neutral-500">Required guards / shift</p>
					</div>
				</div>

				<!-- Duty Sessions Table -->
				<div class="overflow-hidden rounded-2xl border border-neutral-200">
					<div class="border-b border-neutral-200 bg-neutral-50/80 px-4 py-3 sm:px-6">
						<div class="flex items-center justify-between">
							<span class="text-xs font-semibold text-neutral-900"
								>Duty Attendance & Session History</span
							>
							<Button variant="outline" size="sm" onclick={loadAllData}>
								<Clock class="mr-1.5 h-3.5 w-3.5" />
								<span>Refresh Attendance</span>
							</Button>
						</div>
					</div>

					{#if sessions.length === 0}
						<div class="py-12 text-center text-xs text-neutral-500">
							<p>No duty sessions recorded yet.</p>
							<p class="mt-1">Security staff can check in when starting their shift.</p>
						</div>
					{:else}
						<div class="overflow-x-auto">
							<table class="w-full text-left text-xs">
								<thead class="border-b border-neutral-200 bg-neutral-50/40 text-neutral-500">
									<tr>
										<th class="px-4 py-3 font-medium">Gate Post</th>
										<th class="px-4 py-3 font-medium">Stationed Guard</th>
										<th class="px-4 py-3 font-medium">Shift Schedule</th>
										<th class="px-4 py-3 font-medium">Check-In Time</th>
										<th class="px-4 py-3 font-medium">Status</th>
										<th class="px-4 py-3 text-right font-medium">Actions</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-neutral-100 text-neutral-800">
									{#each sessions as s (s.id)}
										<tr class="hover:bg-neutral-50/50">
											<td class="px-4 py-3">
												<span class="font-medium text-neutral-900">{s.gate_name}</span>
											</td>
											<td class="px-4 py-3 font-mono">
												{formatPhoneDisplay(s.user_phone)}
											</td>
											<td class="px-4 py-3">
												{s.shift_name || '—'}
											</td>
											<td class="px-4 py-3 font-mono text-[11px] text-neutral-600">
												{new Date(s.started_at).toLocaleString([], {
													month: 'short',
													day: 'numeric',
													hour: '2-digit',
													minute: '2-digit'
												})}
											</td>
											<td class="px-4 py-3">
												{#if s.status === 'ACTIVE'}
													<Badge
														variant="outline"
														class="border-emerald-200 bg-emerald-50 text-[10px] text-emerald-800"
													>
														On Duty
													</Badge>
												{:else}
													<Badge
														variant="outline"
														class="border-neutral-200 bg-neutral-100 text-[10px] text-neutral-600"
													>
														Ended
													</Badge>
												{/if}
											</td>
											<td class="px-4 py-3 text-right">
												{#if s.status === 'ACTIVE'}
													<Button
														variant="outline"
														size="sm"
														onclick={() => {
															selectedSessionForCheckOut = s;
															isCheckOutOpen = true;
														}}
													>
														<LogOut class="mr-1 h-3 w-3" />
														<span>Check Out</span>
													</Button>
												{:else}
													<span class="text-neutral-400">—</span>
												{/if}
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{/if}
				</div>
			</div>
		{:else if activeTab === 'gates'}
			<!-- Tab 5: Gates Configuration (S08) -->
			<div class="space-y-4">
				<div class="overflow-hidden rounded-2xl border border-neutral-200">
					<div class="overflow-x-auto">
						<table class="w-full text-left text-xs">
							<thead class="border-b border-neutral-200 bg-neutral-50/80 text-neutral-500">
								<tr>
									<th class="px-4 py-3 font-medium">Gate Name & Code</th>
									<th class="px-4 py-3 font-medium">Directionality</th>
									<th class="px-4 py-3 font-medium">Operational Status</th>
									<th class="px-4 py-3 font-medium">Staff Required</th>
									<th class="px-4 py-3 text-right font-medium">Actions</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-neutral-100 text-neutral-800">
								{#each gates as g (g.id)}
									{@const dir = getDirectionBadge(g.directionality)}
									{@const op = getOpStatusBadge(g.operational_status)}
									<tr class="hover:bg-neutral-50/50">
										<td class="px-4 py-3">
											<div class="font-medium text-neutral-900">{g.name}</div>
											<div class="font-mono text-[11px] text-neutral-400">{g.gate_code}</div>
										</td>
										<td class="px-4 py-3">
											<Badge variant="outline" class="text-[10px] {dir.class}">
												{dir.label}
											</Badge>
										</td>
										<td class="px-4 py-3">
											<Badge variant="outline" class="text-[10px] {op.class}">
												{op.label}
											</Badge>
										</td>
										<td class="px-4 py-3 font-mono">
											{g.required_staff_per_shift} guard / shift
										</td>
										<td class="px-4 py-3 text-right">
											<Button
												variant="outline"
												size="sm"
												onclick={() => {
													selectedGateForEdit = g;
													isEditGateOpen = true;
												}}
											>
												<Sliders class="mr-1 h-3 w-3" />
												<span>Configure</span>
											</Button>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			</div>
		{:else if activeTab === 'roster'}
			<!-- Tab 6: Security Staff Roster (S08) -->
			<div class="space-y-4">
				<div class="overflow-hidden rounded-2xl border border-neutral-200">
					<div class="overflow-x-auto">
						<table class="w-full text-left text-xs">
							<thead class="border-b border-neutral-200 bg-neutral-50/80 text-neutral-500">
								<tr>
									<th class="px-4 py-3 font-medium">Security Guard Phone</th>
									<th class="px-4 py-3 font-medium">Role Assignment</th>
									<th class="px-4 py-3 font-medium">Status</th>
									<th class="px-4 py-3 font-medium">Assigned Date</th>
									<th class="px-4 py-3 text-right font-medium">Actions</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-neutral-100 text-neutral-800">
								{#each gatekeepers as gk (gk.id)}
									<tr class="hover:bg-neutral-50/50">
										<td class="px-4 py-3 font-mono font-medium text-neutral-900">
											{formatPhoneDisplay(gk.user_phone_masked)}
										</td>
										<td class="px-4 py-3">
											<Badge variant="outline" class="text-[10px]">
												{gk.role}
											</Badge>
										</td>
										<td class="px-4 py-3">
											<Badge
												variant="outline"
												class="text-[10px] {gk.status === 'ACTIVE'
													? 'border-emerald-200 bg-emerald-50 text-emerald-800'
													: 'border-neutral-200 bg-neutral-100 text-neutral-600'}"
											>
												{gk.status}
											</Badge>
										</td>
										<td class="px-4 py-3 font-mono text-[11px] text-neutral-600">
											{new Date(gk.created_at).toLocaleDateString()}
										</td>
										<td class="px-4 py-3 text-right">
											{#if gk.status === 'ACTIVE'}
												<Button
													variant="outline"
													size="sm"
													onclick={() => handleRevokeGatekeeper(gk)}
													disabled={revokingId === gk.id}
												>
													{#if revokingId === gk.id}
														<Loader2 class="mr-1 h-3 w-3 animate-spin" />
														<span>Revoking...</span>
													{:else}
														<span>Revoke Access</span>
													{/if}
												</Button>
											{:else}
												<span class="text-neutral-400">—</span>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			</div>
		{/if}
	</div>
</div>

<!-- S08 & S09 Dialogs -->
<EditGateDialog
	isOpen={isEditGateOpen}
	{societyId}
	gate={selectedGateForEdit}
	onClose={() => {
		isEditGateOpen = false;
		selectedGateForEdit = null;
	}}
	onSuccess={() => {
		loadAllData();
	}}
/>

<AssignGatekeeperDialog
	isOpen={isAssignGatekeeperOpen}
	{societyId}
	onClose={() => (isAssignGatekeeperOpen = false)}
	onSuccess={() => {
		loadAllData();
	}}
/>

<CreateShiftDialog
	isOpen={isCreateShiftOpen}
	{societyId}
	onClose={() => (isCreateShiftOpen = false)}
	onSuccess={() => {
		loadAllData();
	}}
/>

<CheckInDialog
	isOpen={isCheckInOpen}
	{societyId}
	{gates}
	{shifts}
	onClose={() => (isCheckInOpen = false)}
	onSuccess={() => {
		loadAllData();
	}}
/>

<CheckOutDialog
	isOpen={isCheckOutOpen}
	{societyId}
	session={selectedSessionForCheckOut}
	onClose={() => {
		isCheckOutOpen = false;
		selectedSessionForCheckOut = null;
	}}
	onSuccess={() => {
		loadAllData();
	}}
/>

<!-- S09 Visitor Dialogs -->
<VisitorRegisterDialog
	{societyId}
	isOpen={isRegisterOpen}
	mode={registerMode}
	onClose={() => (isRegisterOpen = false)}
	onSuccess={(visit, pass) => {
		loadAllData();
		if (visit.status === 'PENDING_APPROVAL') {
			statusBanner = {
				type: 'info',
				message: `Visitor "${visit.visitor_name}" registered. Unit policy requires resident approval (Status: PENDING_APPROVAL). Resident must approve before entry.`
			};
		} else {
			statusBanner = {
				type: 'success',
				message: `Visitor "${visit.visitor_name}" registered successfully (Status: ${visit.status}). Ready for gate check-in.`
			};
		}
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

<VisitDetailsDialog
	{societyId}
	visitId={selectedVisitForDetails?.id || null}
	isOpen={isDetailsDialogOpen}
	onClose={() => {
		isDetailsDialogOpen = false;
		selectedVisitForDetails = null;
	}}
/>
