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
			class="fixed inset-0 bg-slate-950/80 backdrop-blur-sm transition-opacity"
			onclick={() => !isLoading && onClose()}
			aria-label="Close dialog overlay"
		></button>

		<!-- Modal Container -->
		<div
			class="relative w-full max-w-md rounded-2xl border border-slate-800 bg-slate-900 p-6 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-slate-800/80 pb-4">
				<div class="flex items-center gap-2.5">
					<div
						class="flex h-9 w-9 items-center justify-center rounded-lg {isDisabling
							? 'bg-rose-500/10 text-rose-400'
							: 'bg-emerald-500/10 text-emerald-400'}"
					>
						{#if isDisabling}
							<ShieldAlert class="h-5 w-5" />
						{:else}
							<CheckCircle class="h-5 w-5" />
						{/if}
					</div>
					<div>
						<h2 id="status-dialog-title" class="text-base font-semibold text-slate-100">
							{isDisabling ? 'Disable Administrator' : 'Reactivate Administrator'}
						</h2>
						<p class="text-xs text-slate-400">
							Target identity: <span class="font-mono text-slate-200">{admin.phone}</span>
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={() => !isLoading && onClose()}
					disabled={isLoading}
					class="rounded-lg p-1.5 text-slate-400 hover:bg-slate-800 hover:text-slate-200 disabled:opacity-50"
					aria-label="Close dialog"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<div class="mt-4 space-y-4">
				{#if error}
					<div
						role="alert"
						class="flex items-start gap-2.5 rounded-lg border border-rose-500/30 bg-rose-500/10 p-3 text-xs text-rose-300"
					>
						<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-rose-400" />
						<span>{error}</span>
					</div>
				{/if}

				{#if isDisabling}
					<p class="text-xs leading-relaxed text-slate-300">
						Disabling this administrator will <strong class="font-semibold text-rose-400"
							>immediately revoke all active sessions</strong
						> and revoke administrative privileges. The underlying phone identity remains active.
					</p>
				{:else}
					<p class="text-xs leading-relaxed text-slate-300">
						Reactivating this administrator will restore their ability to sign in and perform
						administrative tasks.
					</p>
				{/if}

				<div>
					<label for="status-reason" class="block text-xs font-medium text-slate-300">
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
						class="mt-1.5 block w-full rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 focus:outline-none disabled:opacity-50"
					/>
				</div>

				<div class="flex items-center justify-end gap-3 border-t border-slate-800/80 pt-3">
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
						variant={isDisabling ? 'outline' : 'primary'}
						size="sm"
						onclick={handleConfirm}
						disabled={isLoading}
						class={isDisabling ? 'border-rose-500/50 text-rose-400 hover:bg-rose-500/20' : ''}
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
