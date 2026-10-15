<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import { societyApi, type Gate, type GateShift, type GateDutySession } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { LogIn, X, AlertCircle, Loader2 } from '@lucide/svelte';

	let {
		isOpen = false,
		societyId,
		gates = [],
		shifts = [],
		onClose,
		onSuccess
	}: {
		isOpen: boolean;
		societyId: string;
		gates: Gate[];
		shifts: GateShift[];
		onClose: () => void;
		onSuccess?: (session: GateDutySession) => void;
	} = $props();

	let selectedGateId = $state('');
	let selectedShiftId = $state('');
	let isSubmitting = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		if (isOpen) {
			const activeGates = gates.filter(
				(g) => g.status === 'ACTIVE' && g.operational_status !== 'INACTIVE'
			);
			selectedGateId = activeGates.length > 0 ? activeGates[0].id : '';
			selectedShiftId = shifts.length > 0 ? shifts[0].id : '';
			error = null;
		}
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!selectedGateId) {
			error = 'Please select a gate';
			return;
		}

		error = null;
		isSubmitting = true;

		try {
			const res = await societyApi.checkInDuty(societyId, selectedGateId, {
				shift_id: selectedShiftId || undefined
			});

			onSuccess?.(res.session);
			onClose();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.code === 'SESSION_ALREADY_ACTIVE') {
					error = 'You are already checked into an active duty session. Please check out first.';
				} else if (err.code === 'GATE_INACTIVE') {
					error = 'This gate is currently marked inactive or closed for operations.';
				} else {
					error = err.message;
				}
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Failed to check into gate duty';
			}
		} finally {
			isSubmitting = false;
		}
	}
</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape' && isOpen) onClose();
	}}
/>

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
		role="dialog"
		aria-modal="true"
		onclick={(e) => {
			if (e.target === e.currentTarget) onClose();
		}}
	>
		<div
			class="relative w-full max-w-md rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div class="flex items-center gap-2.5">
					<div
						class="flex h-9 w-9 items-center justify-center rounded-xl bg-neutral-100 text-neutral-900"
					>
						<LogIn class="h-4 w-4" />
					</div>
					<div>
						<h3 class="text-sm font-semibold text-neutral-900">Check In to Gate Duty</h3>
						<p class="text-xs text-neutral-500">
							Record active duty attendance at your assigned gate
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={onClose}
					class="rounded-lg p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<form onsubmit={handleSubmit} class="mt-4 space-y-4">
				{#if error}
					<div
						class="flex items-start gap-2 rounded-xl border border-rose-200 bg-rose-50/80 p-3 text-xs text-rose-800"
					>
						<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
						<span>{error}</span>
					</div>
				{/if}

				<div>
					<label for="checkin-gate" class="block text-xs font-medium text-neutral-700"
						>Station / Gate</label
					>
					<select
						id="checkin-gate"
						bind:value={selectedGateId}
						required
						class="mt-1.5 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
					>
						<option value="" disabled>Select gate...</option>
						{#each gates as g (g.id)}
							<option
								value={g.id}
								disabled={g.status !== 'ACTIVE' || g.operational_status === 'INACTIVE'}
							>
								{g.name} ({g.gate_code}) — {g.operational_status}
							</option>
						{/each}
					</select>
				</div>

				<div>
					<label for="checkin-shift" class="block text-xs font-medium text-neutral-700"
						>Operating Shift (Optional)</label
					>
					<select
						id="checkin-shift"
						bind:value={selectedShiftId}
						class="mt-1.5 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
					>
						<option value="">No specific shift / General duty</option>
						{#each shifts as s (s.id)}
							<option value={s.id}>
								{s.name} ({s.start_time} – {s.end_time})
							</option>
						{/each}
					</select>
				</div>

				<div class="flex items-center justify-end gap-2 border-t border-neutral-100 pt-4">
					<Button
						type="button"
						variant="outline"
						size="sm"
						onclick={onClose}
						disabled={isSubmitting}
					>
						Cancel
					</Button>
					<Button
						type="submit"
						variant="primary"
						size="sm"
						disabled={isSubmitting || !selectedGateId}
					>
						{#if isSubmitting}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Checking in...</span>
						{:else}
							<span>Check In Now</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
