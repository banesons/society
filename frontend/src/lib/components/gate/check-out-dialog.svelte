<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import { societyApi, type GateDutySession } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { LogOut, X, AlertCircle, Loader2 } from '@lucide/svelte';

	let {
		isOpen = false,
		societyId,
		session = null,
		onClose,
		onSuccess
	}: {
		isOpen: boolean;
		societyId: string;
		session: GateDutySession | null;
		onClose: () => void;
		onSuccess?: (session: GateDutySession) => void;
	} = $props();

	let handoverNotes = $state('');
	let isSubmitting = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		if (isOpen) {
			handoverNotes = '';
			error = null;
		}
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!session) return;

		error = null;
		isSubmitting = true;

		try {
			const res = await societyApi.checkOutDuty(societyId, session.gate_id, {
				handover_notes: handoverNotes.trim() || undefined
			});

			onSuccess?.(res.session);
			onClose();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message;
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Failed to conclude duty session';
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

{#if isOpen && session}
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
						<LogOut class="h-4 w-4" />
					</div>
					<div>
						<h3 class="text-sm font-semibold text-neutral-900">Check Out of Duty</h3>
						<p class="font-mono text-xs text-neutral-500">
							{session.gate_name} ({session.gate_code})
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

				<div
					class="rounded-xl border border-neutral-200 bg-neutral-50/70 p-3 text-xs text-neutral-600"
				>
					<div class="flex justify-between">
						<span class="text-neutral-500">Stationed Since:</span>
						<span class="font-mono text-neutral-900">
							{new Date(session.started_at).toLocaleTimeString([], {
								hour: '2-digit',
								minute: '2-digit'
							})}
						</span>
					</div>
					{#if session.shift_name}
						<div class="mt-1 flex justify-between">
							<span class="text-neutral-500">Shift Template:</span>
							<span class="font-medium text-neutral-900">{session.shift_name}</span>
						</div>
					{/if}
				</div>

				<div>
					<label for="checkout-notes" class="block text-xs font-medium text-neutral-700"
						>Handover / Shift Notes</label
					>
					<textarea
						id="checkout-notes"
						bind:value={handoverNotes}
						rows="3"
						placeholder="Report perimeter conditions, key handovers, or incidents..."
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
					<Button type="submit" variant="destructive" size="sm" disabled={isSubmitting}>
						{#if isSubmitting}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Checking out...</span>
						{:else}
							<span>Check Out & Hand Over</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
