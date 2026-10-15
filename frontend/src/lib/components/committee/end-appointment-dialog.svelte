<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { societyApi, type CommitteeAppointment } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { UserMinus, X, AlertCircle, Loader2 } from '@lucide/svelte';

	let {
		isOpen = false,
		societyId,
		appointment,
		onClose,
		onSuccess
	}: {
		isOpen: boolean;
		societyId: string;
		appointment: CommitteeAppointment | null;
		onClose: () => void;
		onSuccess?: (appointment: CommitteeAppointment) => void;
	} = $props();

	let endDate = $state(new Date().toISOString().split('T')[0]);
	let endStatus = $state<'ENDED' | 'REVOKED'>('ENDED');
	let notes = $state('');

	let isSubmitting = $state(false);
	let error = $state<string | null>(null);
	let validationErrors = $state<{
		endDate?: string;
	}>({});

	$effect(() => {
		if (isOpen && appointment) {
			endDate = new Date().toISOString().split('T')[0];
			endStatus = 'ENDED';
			notes = '';
			error = null;
			validationErrors = {};
		}
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!appointment) return;
		error = null;
		validationErrors = {};

		if (!endDate) {
			validationErrors.endDate = 'Effective end date is required.';
			return;
		}

		if (appointment.start_date && endDate < appointment.start_date) {
			validationErrors.endDate = `End date cannot precede the appointment start date (${appointment.start_date}).`;
			return;
		}

		isSubmitting = true;

		try {
			const res = await societyApi.endAppointment(societyId, appointment.id, {
				end_date: endDate,
				status: endStatus,
				notes: notes.trim() ? notes.trim() : undefined
			});

			onSuccess?.(res.appointment);
			onClose();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.code === 'INVALID_DATE_RANGE') {
					error = 'End date cannot be earlier than appointment start date.';
				} else if (err.code === 'APPOINTMENT_ALREADY_ENDED') {
					error = 'This appointment has already ended or been revoked.';
				} else {
					error = err.message || 'Failed to end appointment.';
				}
			} else {
				error = 'Failed to end appointment.';
			}
		} finally {
			isSubmitting = false;
		}
	}
</script>

{#if isOpen && appointment}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div
			class="relative w-full max-w-lg rounded-3xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-neutral-200 pb-4">
				<div class="flex items-center gap-3">
					<div
						class="flex h-10 w-10 items-center justify-center rounded-2xl bg-amber-50 text-amber-600"
					>
						<UserMinus class="h-5 w-5" />
					</div>
					<div>
						<h3 class="font-serif text-lg font-medium tracking-tight text-neutral-950">
							Conclude Committee Appointment
						</h3>
						<p class="text-xs text-neutral-500">Record completion of term or official revocation</p>
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

			<!-- Appointment Summary Card -->
			<div class="mt-4 rounded-2xl border border-neutral-200 bg-neutral-50/70 p-4">
				<div class="flex items-center justify-between">
					<div>
						<h4 class="text-xs font-semibold text-neutral-950">
							{appointment.member_name || 'Member'}
						</h4>
						<p class="font-mono text-[11px] text-neutral-500">
							{appointment.member_phone_masked || ''}
						</p>
					</div>
					<Badge variant="outline" class="font-mono text-xs text-neutral-800">
						{appointment.designation_name || appointment.designation_code}
					</Badge>
				</div>
				<div class="mt-3 flex items-center gap-4 text-[11px] text-neutral-500">
					<span
						>Effective Since: <strong class="text-neutral-900">{appointment.start_date}</strong
						></span
					>
					<span>Status: <Badge variant="success" class="text-[10px]">ACTIVE</Badge></span>
				</div>
			</div>

			<form onsubmit={handleSubmit} class="mt-5 space-y-4">
				{#if error}
					<div
						class="flex items-start gap-2.5 rounded-xl border border-rose-200 bg-rose-50/80 p-3.5 text-xs text-rose-800"
					>
						<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
						<span>{error}</span>
					</div>
				{/if}

				<!-- Action Type -->
				<div>
					<span class="block text-xs font-medium text-neutral-700">
						Termination Type <span class="text-rose-500">*</span>
					</span>
					<div class="mt-1.5 grid grid-cols-2 gap-3">
						<button
							type="button"
							onclick={() => (endStatus = 'ENDED')}
							class="flex flex-col rounded-2xl border p-3.5 text-left transition-all {endStatus ===
							'ENDED'
								? 'border-neutral-950 bg-neutral-900 text-white shadow-xs'
								: 'border-neutral-200 bg-white hover:border-neutral-300'}"
						>
							<span
								class="text-xs font-medium {endStatus === 'ENDED'
									? 'text-white'
									: 'text-neutral-900'}"
							>
								Regular Conclusion (ENDED)
							</span>
							<span
								class="mt-1 text-[11px] {endStatus === 'ENDED'
									? 'text-neutral-300'
									: 'text-neutral-500'}"
							>
								Term expired, resignation accepted, or routine handover
							</span>
						</button>

						<button
							type="button"
							onclick={() => (endStatus = 'REVOKED')}
							class="flex flex-col rounded-2xl border p-3.5 text-left transition-all {endStatus ===
							'REVOKED'
								? 'border-rose-600 bg-rose-600 text-white shadow-xs'
								: 'border-neutral-200 bg-white hover:border-neutral-300'}"
						>
							<span
								class="text-xs font-medium {endStatus === 'REVOKED'
									? 'text-white'
									: 'text-neutral-900'}"
							>
								Official Revocation (REVOKED)
							</span>
							<span
								class="mt-1 text-[11px] {endStatus === 'REVOKED'
									? 'text-rose-100'
									: 'text-neutral-500'}"
							>
								Removed by committee vote, disqualification, or dispute
							</span>
						</button>
					</div>
				</div>

				<!-- End Date -->
				<div>
					<label for="end-date-input" class="block text-xs font-medium text-neutral-700">
						Effective End Date <span class="text-rose-500">*</span>
					</label>
					<input
						id="end-date-input"
						type="date"
						bind:value={endDate}
						class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2.5 text-xs text-neutral-900 transition-colors focus:border-neutral-950 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden"
					/>
					{#if validationErrors.endDate}
						<p class="mt-1 text-[11px] text-rose-600">{validationErrors.endDate}</p>
					{/if}
				</div>

				<!-- Notes -->
				<div>
					<label for="end-notes" class="block text-xs font-medium text-neutral-700">
						Reason / Minutes Reference <span class="text-neutral-400">(Optional)</span>
					</label>
					<textarea
						id="end-notes"
						rows="2"
						bind:value={notes}
						placeholder="e.g. Term completed per MC Meeting Minutes 12/2026"
						class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2.5 text-xs text-neutral-900 placeholder-neutral-400 transition-colors focus:border-neutral-950 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden"
					></textarea>
				</div>

				<div class="mt-6 flex items-center justify-end gap-3 border-t border-neutral-200 pt-4">
					<Button type="button" variant="ghost" size="sm" onclick={onClose}>Cancel</Button>
					<Button
						type="submit"
						variant="primary"
						class={endStatus === 'REVOKED' ? 'bg-rose-600 text-white hover:bg-rose-700' : ''}
						size="sm"
						disabled={isSubmitting}
					>
						{#if isSubmitting}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Saving...</span>
						{:else}
							<UserMinus class="mr-1.5 h-3.5 w-3.5" />
							<span>{endStatus === 'REVOKED' ? 'Revoke Appointment' : 'End Appointment'}</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
