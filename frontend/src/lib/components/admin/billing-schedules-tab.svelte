<script lang="ts">
	import { onMount } from 'svelte';
	import {
		societyApi,
		type BillingSchedule,
		type ScheduleExecution,
		type CreateBillingSchedulePayload
	} from '$lib/api/society';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import ResponsiveModal from '$lib/components/ui/responsive-modal.svelte';
	import {
		Calendar,
		Clock,
		Play,
		Pause,
		RotateCw,
		Plus,
		History,
		AlertCircle,
		CheckCircle2,
		XCircle,
		Loader2,
		PowerOff
	} from '@lucide/svelte';

	interface Props {
		societyId: string;
	}

	let { societyId }: Props = $props();

	let schedules = $state<BillingSchedule[]>([]);
	let isLoading = $state<boolean>(true);
	let error = $state<string | null>(null);

	// Action states
	let actionLoadingId = $state<string | null>(null);
	let actionSuccessMsg = $state<string | null>(null);

	// Create/Edit Modal state
	let isCreateModalOpen = $state<boolean>(false);
	let isSubmitting = $state<boolean>(false);
	let formError = $state<string | null>(null);

	let formName = $state<string>('');
	let formRecurrence = $state<'MONTHLY' | 'QUARTERLY' | 'ANNUAL'>('MONTHLY');
	let formTemplateId = $state<string>('');
	let formDayOfMonth = $state<number>(1);
	let formDueDays = $state<number>(15);
	let formTimezone = $state<string>('Asia/Kolkata');
	let formAutoFinalize = $state<boolean>(false);

	// Executions Modal state
	let isExecutionsModalOpen = $state<boolean>(false);
	let selectedSchedule = $state<BillingSchedule | null>(null);
	let executions = $state<ScheduleExecution[]>([]);
	let isLoadingExecutions = $state<boolean>(false);

	onMount(() => {
		loadSchedules();
	});

	async function loadSchedules() {
		isLoading = true;
		error = null;
		try {
			const res = await societyApi.listBillingSchedules(societyId);
			schedules = res.schedules || [];
		} catch (err: any) {
			error = err.message || 'Failed to load billing schedules';
		} finally {
			isLoading = false;
		}
	}

	function openCreateModal() {
		formName = '';
		formRecurrence = 'MONTHLY';
		formTemplateId = '';
		formDayOfMonth = 1;
		formDueDays = 15;
		formTimezone = 'Asia/Kolkata';
		formAutoFinalize = false;
		formError = null;
		isCreateModalOpen = true;
	}

	async function handleCreateSchedule() {
		if (!formName.trim()) {
			formError = 'Schedule name is required';
			return;
		}
		if (!formTemplateId.trim()) {
			formError = 'Template ID is required';
			return;
		}

		isSubmitting = true;
		formError = null;
		try {
			const payload: CreateBillingSchedulePayload = {
				name: formName.trim(),
				recurrence_type: formRecurrence,
				template_id: formTemplateId.trim(),
				day_of_month: formDayOfMonth,
				due_days: formDueDays,
				timezone: formTimezone.trim() || 'Asia/Kolkata',
				auto_finalize: formAutoFinalize
			};
			await societyApi.createBillingSchedule(societyId, payload);
			isCreateModalOpen = false;
			await loadSchedules();
			actionSuccessMsg = 'Schedule created successfully';
			setTimeout(() => (actionSuccessMsg = null), 4000);
		} catch (err: any) {
			formError = err.message || 'Failed to create billing schedule';
		} finally {
			isSubmitting = false;
		}
	}

	async function handleRunNow(sched: BillingSchedule) {
		if (actionLoadingId) return;
		actionLoadingId = sched.id;
		error = null;
		try {
			const exec = await societyApi.runBillingScheduleNow(societyId, sched.id);
			actionSuccessMsg = `Manual run triggered: Status ${exec.status}`;
			setTimeout(() => (actionSuccessMsg = null), 4000);
			await loadSchedules();
		} catch (err: any) {
			error = err.message || 'Failed to execute schedule run';
		} finally {
			actionLoadingId = null;
		}
	}

	async function handleToggleStatus(sched: BillingSchedule) {
		if (actionLoadingId) return;
		actionLoadingId = sched.id;
		error = null;
		try {
			if (sched.status === 'ACTIVE') {
				await societyApi.pauseBillingSchedule(societyId, sched.id);
				actionSuccessMsg = `Schedule "${sched.name}" paused`;
			} else {
				await societyApi.resumeBillingSchedule(societyId, sched.id);
				actionSuccessMsg = `Schedule "${sched.name}" resumed`;
			}
			setTimeout(() => (actionSuccessMsg = null), 4000);
			await loadSchedules();
		} catch (err: any) {
			error = err.message || 'Failed to update schedule status';
		} finally {
			actionLoadingId = null;
		}
	}

	async function handleDisable(sched: BillingSchedule) {
		if (actionLoadingId) return;
		actionLoadingId = sched.id;
		error = null;
		try {
			await societyApi.disableBillingSchedule(societyId, sched.id);
			actionSuccessMsg = `Schedule "${sched.name}" disabled`;
			setTimeout(() => (actionSuccessMsg = null), 4000);
			await loadSchedules();
		} catch (err: any) {
			error = err.message || 'Failed to disable schedule';
		} finally {
			actionLoadingId = null;
		}
	}

	async function openExecutionsModal(sched: BillingSchedule) {
		selectedSchedule = sched;
		isExecutionsModalOpen = true;
		isLoadingExecutions = true;
		executions = [];
		try {
			const res = await societyApi.listScheduleExecutions(societyId, sched.id, 20, 0);
			executions = res.executions || [];
		} catch (err: any) {
			error = err.message || 'Failed to load execution history';
		} finally {
			isLoadingExecutions = false;
		}
	}

	function formatDate(isoStr?: string | null): string {
		if (!isoStr) return '—';
		return new Date(isoStr).toLocaleString('en-IN', {
			day: 'numeric',
			month: 'short',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}
</script>

<div class="space-y-6">
	<!-- Header Bar -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-border/40 pb-4">
		<div>
			<h2 class="text-xl font-bold tracking-tight">Recurring Billing Automation</h2>
			<p class="text-xs text-muted-foreground uppercase tracking-widest font-mono mt-0.5">
				Autonomous Period Assessment & Batch Generation
			</p>
		</div>
		<div class="flex items-center gap-2">
			<Button variant="outline" size="sm" onclick={loadSchedules} disabled={isLoading}>
				<RotateCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
				Refresh
			</Button>
			<Button variant="default" size="sm" onclick={openCreateModal}>
				<Plus class="w-3.5 h-3.5 mr-1.5" />
				New Schedule
			</Button>
		</div>
	</div>

	<!-- Alert Messages -->
	{#if error}
		<div class="p-3 bg-destructive/10 border border-destructive/20 text-destructive text-xs rounded-sm flex items-center gap-2">
			<AlertCircle class="w-4 h-4 shrink-0" />
			<span>{error}</span>
		</div>
	{/if}

	{#if actionSuccessMsg}
		<div class="p-3 bg-emerald-500/10 border border-emerald-500/20 text-emerald-700 dark:text-emerald-300 text-xs rounded-sm flex items-center gap-2">
			<CheckCircle2 class="w-4 h-4 shrink-0" />
			<span>{actionSuccessMsg}</span>
		</div>
	{/if}

	<!-- Schedules Table / List -->
	{#if isLoading && schedules.length === 0}
		<div class="py-16 text-center text-muted-foreground">
			<Loader2 class="w-6 h-6 animate-spin mx-auto mb-2 text-primary" />
			<p class="text-xs font-mono">Loading billing schedules...</p>
		</div>
	{:else if schedules.length === 0}
		<div class="p-8 border border-dashed border-border/80 text-center rounded-sm">
			<Calendar class="w-8 h-8 text-muted-foreground mx-auto mb-2 opacity-50" />
			<h3 class="text-sm font-semibold">No Recurring Schedules Configured</h3>
			<p class="text-xs text-muted-foreground mt-1 max-w-sm mx-auto">
				Set up recurring schedules to automatically generate monthly, quarterly, or annual maintenance bills.
			</p>
			<Button variant="outline" size="sm" class="mt-4" onclick={openCreateModal}>
				<Plus class="w-3.5 h-3.5 mr-1.5" />
				Create First Schedule
			</Button>
		</div>
	{:else}
		<div class="border border-border/60 rounded-sm overflow-hidden bg-card">
			<div class="overflow-x-auto">
				<table class="w-full text-left text-xs">
					<thead class="bg-muted/40 uppercase tracking-widest text-[10px] font-mono border-b border-border/40">
						<tr>
							<th class="py-3 px-4">Schedule Name</th>
							<th class="py-3 px-4">Recurrence</th>
							<th class="py-3 px-4">Next Run</th>
							<th class="py-3 px-4">Last Run</th>
							<th class="py-3 px-4">Status</th>
							<th class="py-3 px-4 text-right">Actions</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border/20 font-sans">
						{#each schedules as s (s.id)}
							<tr class="hover:bg-muted/30 transition-colors">
								<td class="py-3.5 px-4">
									<div class="font-medium text-foreground">{s.name}</div>
									<div class="text-[11px] text-muted-foreground font-mono mt-0.5">
										Template: {s.template_name || s.template_id.slice(0, 8)}
										{#if s.auto_finalize}
											• <span class="text-emerald-600 font-semibold">Auto-Finalize</span>
										{/if}
									</div>
								</td>
								<td class="py-3.5 px-4 font-mono">
									<span class="inline-block px-2 py-0.5 bg-muted rounded text-[11px] font-medium">
										{s.recurrence_type}
									</span>
									<span class="text-muted-foreground text-[10px] block mt-0.5">
										Day {s.day_of_month} (Due: +{s.due_days}d)
									</span>
								</td>
								<td class="py-3.5 px-4 font-mono text-[11px]">
									<div class="flex items-center gap-1.5">
										<Clock class="w-3 h-3 text-muted-foreground" />
										<span>{formatDate(s.next_run_at)}</span>
									</div>
									<span class="text-[10px] text-muted-foreground block pl-4.5">
										{s.timezone}
									</span>
								</td>
								<td class="py-3.5 px-4 font-mono text-[11px] text-muted-foreground">
									{formatDate(s.last_run_at)}
								</td>
								<td class="py-3.5 px-4">
									{#if s.status === 'ACTIVE'}
										<Badge variant="outline" class="border-emerald-500/40 text-emerald-700 bg-emerald-500/10 text-[10px]">
											ACTIVE
										</Badge>
									{:else if s.status === 'PAUSED'}
										<Badge variant="outline" class="border-amber-500/40 text-amber-700 bg-amber-500/10 text-[10px]">
											PAUSED
										</Badge>
									{:else}
										<Badge variant="outline" class="border-muted text-muted-foreground text-[10px]">
											DISABLED
										</Badge>
									{/if}
								</td>
								<td class="py-3.5 px-4 text-right">
									<div class="flex items-center justify-end gap-1.5">
										<Button
											variant="ghost"
											size="icon"
											class="h-7 w-7"
											title="Run Now (Manual Trigger)"
											disabled={actionLoadingId === s.id || s.status === 'DISABLED'}
											onclick={() => handleRunNow(s)}
										>
											{#if actionLoadingId === s.id}
												<Loader2 class="w-3.5 h-3.5 animate-spin" />
											{:else}
												<Play class="w-3.5 h-3.5 text-primary" />
											{/if}
										</Button>

										<Button
											variant="ghost"
											size="icon"
											class="h-7 w-7"
											title={s.status === 'ACTIVE' ? 'Pause' : 'Resume'}
											disabled={actionLoadingId === s.id || s.status === 'DISABLED'}
											onclick={() => handleToggleStatus(s)}
										>
											{#if s.status === 'ACTIVE'}
												<Pause class="w-3.5 h-3.5 text-amber-600" />
											{:else}
												<Play class="w-3.5 h-3.5 text-emerald-600" />
											{/if}
										</Button>

										{#if s.status !== 'DISABLED'}
											<Button
												variant="ghost"
												size="icon"
												class="h-7 w-7 text-muted-foreground hover:text-destructive"
												title="Disable Schedule"
												disabled={actionLoadingId === s.id}
												onclick={() => handleDisable(s)}
											>
												<PowerOff class="w-3.5 h-3.5" />
											</Button>
										{/if}

										<Button
											variant="ghost"
											size="icon"
											class="h-7 w-7 text-muted-foreground"
											title="Execution History"
											onclick={() => openExecutionsModal(s)}
										>
											<History class="w-3.5 h-3.5" />
										</Button>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>

<!-- Create Schedule Modal -->
<ResponsiveModal
	bind:isOpen={isCreateModalOpen}
	title="Create Recurring Billing Schedule"
	description="Configure automated periodic billing assessments for this society."
>
	<form onsubmit={(e) => { e.preventDefault(); handleCreateSchedule(); }} class="space-y-4">
		{#if formError}
			<div class="p-2.5 bg-destructive/10 border border-destructive/20 text-destructive text-xs rounded-sm">
				{formError}
			</div>
		{/if}

		<div class="space-y-1.5">
			<label class="text-xs font-mono uppercase tracking-wider block text-muted-foreground" for="sched-name">
				Schedule Name *
			</label>
			<input
				id="sched-name"
				type="text"
				bind:value={formName}
				placeholder="e.g. Monthly Standard Maintenance"
				class="w-full px-3 py-2 text-xs border border-border/80 rounded-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary font-sans"
				required
			/>
		</div>

		<div class="grid grid-cols-2 gap-3">
			<div class="space-y-1.5">
				<label class="text-xs font-mono uppercase tracking-wider block text-muted-foreground" for="sched-recurrence">
					Recurrence *
				</label>
				<select
					id="sched-recurrence"
					bind:value={formRecurrence}
					class="w-full px-3 py-2 text-xs border border-border/80 rounded-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary font-sans"
				>
					<option value="MONTHLY">Monthly</option>
					<option value="QUARTERLY">Quarterly</option>
					<option value="ANNUAL">Annual</option>
				</select>
			</div>

			<div class="space-y-1.5">
				<label class="text-xs font-mono uppercase tracking-wider block text-muted-foreground" for="sched-template-id">
					Template ID *
				</label>
				<input
					id="sched-template-id"
					type="text"
					bind:value={formTemplateId}
					placeholder="Billing Template UUID"
					class="w-full px-3 py-2 text-xs border border-border/80 rounded-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary font-mono"
					required
				/>
			</div>
		</div>

		<div class="grid grid-cols-3 gap-3">
			<div class="space-y-1.5">
				<label class="text-xs font-mono uppercase tracking-wider block text-muted-foreground" for="sched-day">
					Day of Month
				</label>
				<input
					id="sched-day"
					type="number"
					min="1"
					max="31"
					bind:value={formDayOfMonth}
					class="w-full px-3 py-2 text-xs border border-border/80 rounded-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary font-mono"
				/>
			</div>

			<div class="space-y-1.5">
				<label class="text-xs font-mono uppercase tracking-wider block text-muted-foreground" for="sched-due">
					Due Days Offset
				</label>
				<input
					id="sched-due"
					type="number"
					min="1"
					max="90"
					bind:value={formDueDays}
					class="w-full px-3 py-2 text-xs border border-border/80 rounded-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary font-mono"
				/>
			</div>

			<div class="space-y-1.5">
				<label class="text-xs font-mono uppercase tracking-wider block text-muted-foreground" for="sched-tz">
					Timezone
				</label>
				<input
					id="sched-tz"
					type="text"
					bind:value={formTimezone}
					class="w-full px-3 py-2 text-xs border border-border/80 rounded-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary font-mono"
				/>
			</div>
		</div>

		<div class="flex items-center gap-2 pt-1">
			<input
				id="sched-autofinalize"
				type="checkbox"
				bind:checked={formAutoFinalize}
				class="h-4 w-4 rounded border-border text-primary focus:ring-primary"
			/>
			<label class="text-xs cursor-pointer select-none text-foreground" for="sched-autofinalize">
				Auto-finalize successfully generated drafts (convert to active dues automatically)
			</label>
		</div>

		<div class="flex justify-end gap-2 pt-4 border-t border-border/40">
			<Button variant="ghost" size="sm" onclick={() => (isCreateModalOpen = false)} disabled={isSubmitting}>
				Cancel
			</Button>
			<Button variant="default" size="sm" type="submit" disabled={isSubmitting}>
				{#if isSubmitting}
					<Loader2 class="w-3.5 h-3.5 animate-spin mr-1.5" />
				{/if}
				Create Schedule
			</Button>
		</div>
	</form>
</ResponsiveModal>

<!-- Execution History Modal -->
<ResponsiveModal
	bind:isOpen={isExecutionsModalOpen}
	title={`Executions — ${selectedSchedule?.name || ''}`}
	description="Immutable audit history of all scheduled and manual batch runs."
>
	<div class="space-y-4">
		{#if isLoadingExecutions}
			<div class="py-8 text-center text-muted-foreground">
				<Loader2 class="w-5 h-5 animate-spin mx-auto mb-2 text-primary" />
				<p class="text-xs font-mono">Loading execution history...</p>
			</div>
		{:else if executions.length === 0}
			<p class="text-xs text-muted-foreground text-center py-6">
				No execution history found for this schedule.
			</p>
		{:else}
			<div class="border border-border/60 rounded-sm overflow-hidden bg-card">
				<div class="max-h-80 overflow-y-auto">
					<table class="w-full text-left text-xs">
						<thead class="bg-muted/40 uppercase tracking-widest text-[9px] font-mono border-b border-border/40 sticky top-0">
							<tr>
								<th class="py-2.5 px-3">Trigger / Period</th>
								<th class="py-2.5 px-3">Batch</th>
								<th class="py-2.5 px-3">Started</th>
								<th class="py-2.5 px-3">Status</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border/20 font-sans">
							{#each executions as e (e.id)}
								<tr class="hover:bg-muted/20">
									<td class="py-2.5 px-3">
										<div class="font-medium text-foreground">{e.period_name || '—'}</div>
										<span class="text-[10px] font-mono text-muted-foreground">
											Source: {e.trigger_source}
										</span>
									</td>
									<td class="py-2.5 px-3 font-mono text-[11px] text-muted-foreground">
										{e.batch_number || '—'}
									</td>
									<td class="py-2.5 px-3 font-mono text-[10px] text-muted-foreground">
										{formatDate(e.started_at)}
									</td>
									<td class="py-2.5 px-3">
										{#if e.status === 'SUCCESS'}
											<Badge variant="outline" class="border-emerald-500/40 text-emerald-700 bg-emerald-500/10 text-[9px]">
												SUCCESS
											</Badge>
										{:else if e.status === 'FAILED'}
											<Badge variant="outline" class="border-destructive/40 text-destructive bg-destructive/10 text-[9px]" title={e.error_message || ''}>
												FAILED
											</Badge>
										{:else}
											<Badge variant="outline" class="border-muted text-muted-foreground text-[9px]">
												{e.status}
											</Badge>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}

		<div class="flex justify-end pt-2">
			<Button variant="ghost" size="sm" onclick={() => (isExecutionsModalOpen = false)}>
				Close
			</Button>
		</div>
	</div>
</ResponsiveModal>
