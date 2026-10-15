<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import { societyApi, type SocietyAdminAssignment } from '$lib/api/society';
	import { normalizePhone, formatPhoneDisplay } from '$lib/phone';
	import { ApiClientError } from '$lib/api/client';
	import { Shield, X, AlertCircle, Loader2 } from '@lucide/svelte';

	let {
		isOpen = false,
		societyId,
		onClose,
		onSuccess
	}: {
		isOpen: boolean;
		societyId: string;
		onClose: () => void;
		onSuccess?: (assignment: SocietyAdminAssignment) => void;
	} = $props();

	let phoneInput = $state('');
	let isSubmitting = $state(false);
	let error = $state<string | null>(null);

	let normalizedPhone = $derived.by(() => {
		try {
			return normalizePhone(phoneInput);
		} catch {
			return null;
		}
	});

	$effect(() => {
		if (isOpen) {
			phoneInput = '';
			error = null;
		}
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!normalizedPhone) {
			error = 'Please enter a valid 10-digit Indian mobile number';
			return;
		}

		error = null;
		isSubmitting = true;

		try {
			const res = await societyApi.assignGatekeeper(societyId, normalizedPhone);
			onSuccess?.(res.gatekeeper);
			onClose();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.code === 'GATEKEEPER_ALREADY_ASSIGNED') {
					error = 'This user already holds an active gatekeeper assignment in this society';
				} else {
					error = err.message;
				}
			} else if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Failed to assign gatekeeper';
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
						<Shield class="h-4 w-4" />
					</div>
					<div>
						<h3 class="text-sm font-semibold text-neutral-900">Assign Gatekeeper</h3>
						<p class="text-xs text-neutral-500">Add security personnel to society staff roster</p>
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
					<label for="guard-phone" class="block text-xs font-medium text-neutral-700"
						>Mobile Number</label
					>
					<input
						id="guard-phone"
						type="tel"
						bind:value={phoneInput}
						placeholder="e.g. 9820012345"
						required
						class="mt-1.5 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 font-mono text-xs text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-900 focus:outline-hidden"
					/>
					{#if normalizedPhone}
						<p class="mt-1.5 font-mono text-[11px] text-emerald-700">
							E.164: {formatPhoneDisplay(normalizedPhone)}
						</p>
					{/if}
				</div>

				<div
					class="rounded-xl border border-neutral-200 bg-neutral-50/70 p-3 text-xs text-neutral-600"
				>
					<p class="font-medium text-neutral-900">Operational Role Scoping</p>
					<p class="mt-1 text-[11px] leading-relaxed text-neutral-500">
						The appointed guard will be able to sign in via OTP, view society gates, and record
						active duty shifts. Gatekeepers do not have administrative access.
					</p>
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
						disabled={isSubmitting || !normalizedPhone}
					>
						{#if isSubmitting}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Assigning...</span>
						{:else}
							<span>Assign Gatekeeper</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
