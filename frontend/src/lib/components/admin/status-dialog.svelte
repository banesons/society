<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import { platformApi, type AdminItem } from '$lib/api/platform';
	import { ApiClientError } from '$lib/api/client';
	import { AlertTriangle, CheckCircle, X, Loader2, ShieldAlert } from '@lucide/svelte';

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

	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Escape' && isOpen && !isLoading) {
			onClose();
		}
	}
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if isOpen && admin}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6"
		role="dialog"
		aria-labelledby="status-dialog-title"
		aria-modal="true"
	>
		<!-- Backdrop -->
		<button
			type="button"
			tabindex="-1"
			class="fixed inset-0 bg-neutral-950/40 backdrop-blur-xs transition-opacity"
			onclick={() => !isLoading && onClose()}
			aria-label="Close dialog overlay"
		></button>

		<!-- Modal Container -->
		<div
			class="relative w-full max-w-md rounded-3xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div class="flex items-center gap-2.5">
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
					<div>
						<h2
							id="status-dialog-title"
							class="text-base font-semibold tracking-tight text-neutral-900"
						>
							{isDisabling ? 'Disable Administrator' : 'Reactivate Administrator'}
						</h2>
						<p class="text-xs text-neutral-500">
							Target identity: <span class="font-mono font-bold text-neutral-900"
								>{admin.phone}</span
							>
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={() => !isLoading && onClose()}
					disabled={isLoading}
					class="rounded-full p-1.5 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900 disabled:opacity-50"
					aria-label="Close dialog"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<div class="mt-4 space-y-4">
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

				<div class="flex items-center justify-end gap-3 border-t border-neutral-100 pt-3">
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
			</div>
		</div>
	</div>
{/if}
