<script lang="ts">
	import ResponsiveModal from '$lib/components/ui/responsive-modal.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import { platformApi, type AdminItem } from '$lib/api/platform';
	import { ApiClientError } from '$lib/api/client';
	import { AlertTriangle, CheckCircle, Loader2, ShieldAlert } from '@lucide/svelte';

	let {
		isOpen = false,
		admin = null,
		targetStatus = 'DISABLED',
		onClose,
		onSuccess
	}: {
		isOpen: boolean;
		admin: AdminItem | null;
		targetStatus: 'ACTIVE' | 'DISABLED';
		onClose: () => void;
		onSuccess?: () => void;
	} = $props();

	let reason = $state('');
	let isLoading = $state(false);
	let error = $state<string | null>(null);

	const isDisabling = $derived(targetStatus === 'DISABLED');

	async function handleConfirm() {
		if (!admin) return;
		error = null;
		isLoading = true;

		try {
			await platformApi.updateAdminStatus(admin.id, targetStatus, reason.trim());
			reason = '';
			if (onSuccess) {
				onSuccess();
			}
			onClose();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.code === 'CANNOT_DISABLE_SELF') {
					error = 'Super Admin account cannot be disabled.';
				} else {
					error = err.message || 'Failed to update administrator status.';
				}
			} else {
				error = 'An unexpected error occurred. Please try again.';
			}
		} finally {
			isLoading = false;
		}
	}
</script>

<ResponsiveModal
	isOpen={isOpen && !!admin}
	{onClose}
	title={isDisabling ? 'Disable Administrator' : 'Reactivate Administrator'}
	description={admin ? `Target identity: ${admin.phone}` : undefined}
	maxWidth="md"
	isDismissible={!isLoading}
	closeOnBackdrop={!isLoading}
	closeOnEscape={!isLoading}
	showCloseButton={!isLoading}
	destructive={isDisabling}
>
	{#snippet icon()}
		<div
			class="flex h-9 w-9 items-center justify-center rounded-full {isDisabling
				? 'bg-rose-50 text-rose-600'
				: 'bg-emerald-50 text-emerald-600'}"
		>
			{#if isDisabling}
				<ShieldAlert class="h-4 w-4" />
			{:else}
				<CheckCircle class="h-4 w-4" />
			{/if}
		</div>
	{/snippet}

	<div class="space-y-4">
		{#if error}
			<div
				role="alert"
				class="flex items-start gap-2.5 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
			>
				<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
				<span>{error}</span>
			</div>
		{/if}

		{#if isDisabling}
			<p class="text-xs leading-relaxed text-neutral-700">
				Disabling this administrator will <strong class="font-semibold text-rose-700"
					>immediately revoke all active sessions</strong
				> and revoke administrative privileges. The underlying phone identity remains active.
			</p>
		{:else}
			<p class="text-xs leading-relaxed text-neutral-700">
				Reactivating this administrator will restore their ability to sign in and perform
				administrative tasks.
			</p>
		{/if}

		<div>
			<label for="status-reason" class="block text-xs font-medium text-neutral-700">
				Reason (Optional, recorded in immutable audit log)
			</label>
			<input
				id="status-reason"
				type="text"
				bind:value={reason}
				placeholder={isDisabling
					? 'e.g., Security review, role rotation'
					: 'e.g., Completed verification'}
				disabled={isLoading}
				class="mt-1.5 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
			/>
		</div>
	</div>

	{#snippet footer()}
		<div class="flex items-center justify-end gap-3">
			<Button
				type="button"
				variant="ghost"
				size="sm"
				onclick={() => onClose()}
				disabled={isLoading}
			>
				Cancel
			</Button>
			<Button
				type="button"
				variant={isDisabling ? 'destructive' : 'primary'}
				size="sm"
				onclick={handleConfirm}
				disabled={isLoading}
			>
				{#if isLoading}
					<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					<span>Updating...</span>
				{:else}
					<span>{isDisabling ? 'Confirm Disable' : 'Confirm Reactivate'}</span>
				{/if}
			</Button>
		</div>
	{/snippet}
</ResponsiveModal>
