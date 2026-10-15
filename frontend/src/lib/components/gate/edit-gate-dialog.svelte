<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import { societyApi, type Gate } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { Sliders, X, AlertCircle, Loader2 } from '@lucide/svelte';

	let {
		isOpen = false,
		societyId,
		gate = null,
		onClose,
		onSuccess
	}: {
		isOpen: boolean;
		societyId: string;
		gate: Gate | null;
		onClose: () => void;
		onSuccess?: (updated: Gate) => void;
	} = $props();

	let name = $state('');
	let directionality = $state<'BIDIRECTIONAL' | 'INGRESS_ONLY' | 'EGRESS_ONLY'>('BIDIRECTIONAL');
	let operationalStatus = $state<'ACTIVE' | 'INACTIVE' | 'MAINTENANCE' | 'EMERGENCY_OPEN'>(
		'ACTIVE'
	);
	let requiredStaff = $state(1);
	let description = $state('');

	let isSubmitting = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		if (isOpen && gate) {
			name = gate.name;
			directionality = gate.directionality || 'BIDIRECTIONAL';
			operationalStatus = gate.operational_status || 'ACTIVE';
			requiredStaff = gate.required_staff_per_shift;
			description = gate.description || '';
			error = null;
		}
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!gate) return;

		error = null;
		isSubmitting = true;

		try {
			const res = await societyApi.updateGate(societyId, gate.id, {
				name: name.trim(),
				directionality,
				operational_status: operationalStatus,
				required_staff_per_shift: Number(requiredStaff),
				description: description.trim() || undefined
			});

			onSuccess?.(res.gate);
			onClose();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message;
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Failed to update gate settings';
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

{#if isOpen && gate}
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
						<Sliders class="h-4 w-4" />
					</div>
					<div>
						<h3 class="text-sm font-semibold text-neutral-900">Configure Gate</h3>
						<p class="font-mono text-xs text-neutral-500">{gate.gate_code}</p>
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
					<label for="gate-name" class="block text-xs font-medium text-neutral-700">Gate Name</label
					>
					<input
						id="gate-name"
						type="text"
						bind:value={name}
						required
						class="mt-1.5 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
					/>
				</div>

				<div class="grid grid-cols-2 gap-3">
					<div>
						<label for="gate-direction" class="block text-xs font-medium text-neutral-700"
							>Directionality</label
						>
						<select
							id="gate-direction"
							bind:value={directionality}
							class="mt-1.5 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
						>
							<option value="BIDIRECTIONAL">Bidirectional</option>
							<option value="INGRESS_ONLY">Ingress Only (Entry)</option>
							<option value="EGRESS_ONLY">Egress Only (Exit)</option>
						</select>
					</div>

					<div>
						<label for="gate-op-status" class="block text-xs font-medium text-neutral-700"
							>Operational Status</label
						>
						<select
							id="gate-op-status"
							bind:value={operationalStatus}
							class="mt-1.5 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
						>
							<option value="ACTIVE">Active (Open)</option>
							<option value="INACTIVE">Inactive (Closed)</option>
							<option value="MAINTENANCE">Maintenance</option>
							<option value="EMERGENCY_OPEN">Emergency Open</option>
						</select>
					</div>
				</div>

				<div>
					<label for="gate-staff" class="block text-xs font-medium text-neutral-700"
						>Staff Headcount Per Shift</label
					>
					<input
						id="gate-staff"
						type="number"
						min="0"
						max="10"
						bind:value={requiredStaff}
						class="mt-1.5 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
					/>
				</div>

				<div>
					<label for="gate-desc" class="block text-xs font-medium text-neutral-700"
						>Description</label
					>
					<textarea
						id="gate-desc"
						bind:value={description}
						rows="2"
						placeholder="Optional notes or instructions..."
						class="mt-1.5 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-900 focus:outline-hidden"
					></textarea>
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
					<Button type="submit" variant="primary" size="sm" disabled={isSubmitting}>
						{#if isSubmitting}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Saving...</span>
						{:else}
							<span>Save Configuration</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
