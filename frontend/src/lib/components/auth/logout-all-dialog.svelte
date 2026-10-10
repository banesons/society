<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import { ShieldAlert, AlertTriangle, X, Loader2 } from '@lucide/svelte';

	let {
		isOpen = false,
		onClose
	}: {
		isOpen: boolean;
		onClose: () => void;
	} = $props();

	let isLoading = $state(false);
	let error = $state<string | null>(null);

	async function handleConfirmLogoutAll() {
		isLoading = true;
		error = null;
		try {
			await auth.logoutAll();
			onClose();
			goto('/login');
		} catch (err: unknown) {
			error =
				err instanceof Error ? err.message : 'Failed to revoke all sessions. Please try again.';
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

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6"
		role="dialog"
		aria-labelledby="logout-all-title"
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
			class="relative w-full max-w-md rounded-2xl border border-rose-500/30 bg-slate-900 p-6 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-slate-800/80 pb-4">
				<div class="flex items-center gap-2.5">
					<div
						class="flex h-9 w-9 items-center justify-center rounded-lg bg-rose-500/10 text-rose-400"
					>
						<ShieldAlert class="h-5 w-5" />
					</div>
					<div>
						<h2 id="logout-all-title" class="text-base font-semibold text-slate-100">
							Log Out From All Devices?
						</h2>
						<p class="text-xs text-slate-400">Global session termination</p>
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

				<p class="text-xs leading-relaxed text-slate-300">
					This will <strong class="font-semibold text-rose-400">revoke all active sessions</strong> across
					every device registered to your mobile number. You will be signed out on this device and must
					authenticate again with a new OTP.
				</p>

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
						variant="outline"
						size="sm"
						onclick={handleConfirmLogoutAll}
						disabled={isLoading}
						class="border-rose-500/40 text-rose-400 hover:bg-rose-500/20"
					>
						{#if isLoading}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Revoking All Sessions...</span>
						{:else}
							<span>Confirm Logout All Devices</span>
						{/if}
					</Button>
				</div>
			</div>
		</div>
	</div>
{/if}
