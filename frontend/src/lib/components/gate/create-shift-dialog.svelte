<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import { societyApi, type GateShift } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { Clock, X, AlertCircle, Loader2 } from '@lucide/svelte';

	let {
		isOpen = false,
		societyId,
		onClose,
		onSuccess
	}: {
		isOpen: boolean;
		societyId: string;
		onClose: () => void;
		onSuccess?: (shift: GateShift) => void;
	} = $props();

	let name = $state('');
	let startTime = $state('06:00');
	let endTime = $state('14:00');
	let isSubmitting = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		if (isOpen) {
			name = '';
			startTime = '06:00';
			endTime = '14:00';
			error = null;
		}
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!name.trim()) {
			error = 'Shift name is required';
			return;
		}

		error = null;
		isSubmitting = true;

		try {
			const res = await societyApi.createShift(societyId, {
				name: name.trim(),
				start_time: startTime + ':00',
				end_time: endTime + ':00'
			});

			onSuccess?.(res.shift);
			onClose();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.code === 'SHIFT_NAME_EXISTS') {
					error = 'A shift with this name already exists in this society';
				} else {
					error = err.message;
				}
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Failed to create shift';
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
						<Clock class="h-4 w-4" />
					</div>
					<div>
						<h3 class="text-sm font-semibold text-neutral-900">Define Shift Schedule</h3>
						<p class="text-xs text-neutral-500">Configure standard operating shift hours</p>
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
					<label for="shift-name" class="block text-xs font-medium text-neutral-700"
						>Shift Name</label
					>
					<input
						id="shift-name"
						type="text"
						bind:value={name}
						placeholder="e.g. Morning Shift, Night Shift"
						required
						class="mt-1.5 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-900 focus:outline-hidden"
					/>
				</div>

				<div class="grid grid-cols-2 gap-3">
					<div>
						<label for="shift-start" class="block text-xs font-medium text-neutral-700"
							>Start Time</label
						>
						<input
							id="shift-start"
							type="time"
							bind:value={startTime}
							required
							class="mt-1.5 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 font-mono text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
						/>
					</div>

					<div>
						<label for="shift-end" class="block text-xs font-medium text-neutral-700"
							>End Time</label
						>
						<input
							id="shift-end"
							type="time"
							bind:value={endTime}
							required
							class="mt-1.5 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 font-mono text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
						/>
					</div>
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
							<span>Create Shift</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
