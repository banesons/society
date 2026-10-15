<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import {
		societyApi,
		type SocietyMember,
		type SocietyDesignation,
		type CommitteeAppointment
	} from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { UserCheck, X, AlertCircle, Loader2 } from '@lucide/svelte';

	let {
		isOpen = false,
		societyId,
		onClose,
		onSuccess
	}: {
		isOpen: boolean;
		societyId: string;
		onClose: () => void;
		onSuccess?: (appointment: CommitteeAppointment) => void;
	} = $props();

	let members = $state<SocietyMember[]>([]);
	let designations = $state<SocietyDesignation[]>([]);
	let isLoadingOptions = $state(false);

	let selectedMemberId = $state('');
	let selectedDesignationId = $state('');
	let startDate = $state(new Date().toISOString().split('T')[0]);
	let endDate = $state('');
	let notes = $state('');

	let isSubmitting = $state(false);
	let error = $state<string | null>(null);
	let validationErrors = $state<{
		memberId?: string;
		designationId?: string;
		startDate?: string;
		endDate?: string;
	}>({});

	let selectedDesignation = $derived(
		designations.find((d) => d.id === selectedDesignationId) || null
	);

	async function loadOptions() {
		if (!societyId) return;
		isLoadingOptions = true;
		error = null;

		try {
			const [membersRes, desigRes] = await Promise.all([
				societyApi.listMembers(societyId, { status: 'ACTIVE', limit: 200 }),
				societyApi.listDesignations(societyId, false)
			]);
			members = membersRes.members;
			designations = desigRes.designations;
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message || 'Failed to load member and designation catalogues.';
			} else {
				error = 'Failed to load options.';
			}
		} finally {
			isLoadingOptions = false;
		}
	}

	$effect(() => {
		if (isOpen) {
			selectedMemberId = '';
			selectedDesignationId = '';
			startDate = new Date().toISOString().split('T')[0];
			endDate = '';
			notes = '';
			error = null;
			validationErrors = {};
			loadOptions();
		}
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		validationErrors = {};

		if (!selectedMemberId) {
			validationErrors.memberId = 'Please select an active society member.';
		}
		if (!selectedDesignationId) {
			validationErrors.designationId = 'Please select a designation.';
		}
		if (!startDate) {
			validationErrors.startDate = 'Effective start date is required.';
		}
		if (endDate && startDate && endDate < startDate) {
			validationErrors.endDate = 'End date cannot be prior to the start date.';
		}

		if (Object.keys(validationErrors).length > 0) {
			return;
		}

		isSubmitting = true;

		try {
			const res = await societyApi.createAppointment(societyId, {
				member_id: selectedMemberId,
				designation_id: selectedDesignationId,
				start_date: startDate,
				end_date: endDate.trim() ? endDate.trim() : undefined,
				notes: notes.trim() ? notes.trim() : undefined
			});

			onSuccess?.(res.appointment);
			onClose();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.code === 'DESIGNATION_SLOT_OCCUPIED') {
					error =
						'This designation is a single-holder position and already has an active occupant. Please end or revoke the current appointment first.';
				} else if (err.code === 'OVERLAPPING_APPOINTMENT') {
					error = 'This member already holds an active appointment for this designation.';
				} else if (err.code === 'MULTIPLE_DESIGNATIONS_DISALLOWED') {
					error =
						'This designation cannot be held simultaneously with other active committee roles.';
				} else if (err.code === 'INVALID_MEMBER_STATUS') {
					error = 'Appointments can only be conferred upon ACTIVE society members.';
				} else if (err.code === 'DESIGNATION_INACTIVE') {
					error = 'The chosen designation is inactive and cannot accept new appointments.';
				} else {
					error = err.message || 'Failed to create appointment.';
				}
			} else {
				error = 'Failed to create appointment.';
			}
		} finally {
			isSubmitting = false;
		}
	}
</script>

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div
			class="relative w-full max-w-lg rounded-3xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-neutral-200 pb-4">
				<div class="flex items-center gap-3">
					<div
						class="flex h-10 w-10 items-center justify-center rounded-2xl bg-neutral-100 text-neutral-900"
					>
						<UserCheck class="h-5 w-5" />
					</div>
					<div>
						<h3 class="font-serif text-lg font-medium tracking-tight text-neutral-950">
							Appoint Committee Member
						</h3>
						<p class="text-xs text-neutral-500">
							Assign a governance designation to an active resident
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={onClose}
					class="rounded-full p-2 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			{#if isLoadingOptions}
				<div class="flex min-h-[220px] flex-col items-center justify-center text-neutral-500">
					<Loader2 class="h-7 w-7 animate-spin text-neutral-900" />
					<p class="mt-3 text-xs">Loading member and designation catalogues...</p>
				</div>
			{:else}
				<form onsubmit={handleSubmit} class="mt-5 space-y-4">
					{#if error}
						<div
							class="flex items-start gap-2.5 rounded-xl border border-rose-200 bg-rose-50/80 p-3.5 text-xs text-rose-800"
						>
							<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
							<span>{error}</span>
						</div>
					{/if}

					<!-- Member Select -->
					<div>
						<label for="member-select" class="block text-xs font-medium text-neutral-700">
							Society Member <span class="text-rose-500">*</span>
						</label>
						{#if members.length === 0}
							<p
								class="mt-1.5 rounded-xl border border-amber-200 bg-amber-50 p-2.5 text-xs text-amber-700"
							>
								No active society members found. Please register members in the directory first.
							</p>
						{:else}
							<select
								id="member-select"
								bind:value={selectedMemberId}
								class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2.5 text-xs text-neutral-900 transition-colors focus:border-neutral-950 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden"
							>
								<option value="">Select an active member...</option>
								{#each members as m (m.id)}
									<option value={m.id}>
										{m.first_name}
										{m.last_name || ''} ({m.phone})
									</option>
								{/each}
							</select>
						{/if}
						{#if validationErrors.memberId}
							<p class="mt-1 text-[11px] text-rose-600">{validationErrors.memberId}</p>
						{/if}
					</div>

					<!-- Designation Select -->
					<div>
						<label for="designation-select" class="block text-xs font-medium text-neutral-700">
							Designation / Position <span class="text-rose-500">*</span>
						</label>
						{#if designations.length === 0}
							<p
								class="mt-1.5 rounded-xl border border-amber-200 bg-amber-50 p-2.5 text-xs text-amber-700"
							>
								No active designations available.
							</p>
						{:else}
							<select
								id="designation-select"
								bind:value={selectedDesignationId}
								class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2.5 text-xs text-neutral-900 transition-colors focus:border-neutral-950 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden"
							>
								<option value="">Select a designation...</option>
								{#each designations as d (d.id)}
									<option value={d.id}>
										{d.name} ({d.code}) {d.allows_multiple_members ? '' : '— [Singleton]'}
									</option>
								{/each}
							</select>
						{/if}
						{#if selectedDesignation}
							<div class="mt-2 flex items-center gap-2 text-[11px] text-neutral-600">
								<Badge variant="outline" class="text-[10px]">
									{selectedDesignation.allows_multiple_members
										? 'Multi-occupant allowed'
										: 'Sole occupant only'}
								</Badge>
								{#if selectedDesignation.description}
									<span>{selectedDesignation.description}</span>
								{/if}
							</div>
						{/if}
						{#if validationErrors.designationId}
							<p class="mt-1 text-[11px] text-rose-600">{validationErrors.designationId}</p>
						{/if}
					</div>

					<!-- Date Inputs -->
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<label for="start-date" class="block text-xs font-medium text-neutral-700">
								Start Date <span class="text-rose-500">*</span>
							</label>
							<div class="relative mt-1.5">
								<input
									id="start-date"
									type="date"
									bind:value={startDate}
									class="w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2.5 text-xs text-neutral-900 transition-colors focus:border-neutral-950 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden"
								/>
							</div>
							{#if validationErrors.startDate}
								<p class="mt-1 text-[11px] text-rose-600">{validationErrors.startDate}</p>
							{/if}
						</div>

						<div>
							<label for="end-date" class="block text-xs font-medium text-neutral-700">
								Expected End Date <span class="text-neutral-400">(Optional)</span>
							</label>
							<div class="relative mt-1.5">
								<input
									id="end-date"
									type="date"
									bind:value={endDate}
									class="w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2.5 text-xs text-neutral-900 transition-colors focus:border-neutral-950 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden"
								/>
							</div>
							{#if validationErrors.endDate}
								<p class="mt-1 text-[11px] text-rose-600">{validationErrors.endDate}</p>
							{/if}
						</div>
					</div>

					<!-- Notes -->
					<div>
						<label for="appointment-notes" class="block text-xs font-medium text-neutral-700">
							Appointment Notes / Resolution Ref <span class="text-neutral-400">(Optional)</span>
						</label>
						<textarea
							id="appointment-notes"
							rows="2"
							bind:value={notes}
							placeholder="e.g. Appointed per AGM Resolution 4/2026"
							class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2.5 text-xs text-neutral-900 placeholder-neutral-400 transition-colors focus:border-neutral-950 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden"
						></textarea>
					</div>

					<div class="mt-6 flex items-center justify-end gap-3 border-t border-neutral-200 pt-4">
						<Button type="button" variant="ghost" size="sm" onclick={onClose}>Cancel</Button>
						<Button
							type="submit"
							variant="primary"
							size="sm"
							disabled={isSubmitting || members.length === 0 || designations.length === 0}
						>
							{#if isSubmitting}
								<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
								<span>Appointing...</span>
							{:else}
								<UserCheck class="mr-1.5 h-3.5 w-3.5" />
								<span>Confirm Appointment</span>
							{/if}
						</Button>
					</div>
				</form>
			{/if}
		</div>
	</div>
{/if}
