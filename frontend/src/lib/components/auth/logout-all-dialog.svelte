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
			class="fixed inset-0 bg-neutral-950/40 backdrop-blur-xs transition-opacity"
			onclick={() => !isLoading && onClose()}
			aria-label="Close dialog overlay"
		></button>

		<!-- Modal Container -->
		<div
			class="relative w-full max-w-md rounded-3xl border border-neutral-200 bg-white p-6 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div class="flex items-center gap-2.5">
					<div
						class="flex h-9 w-9 items-center justify-center rounded-full bg-rose-50 text-rose-600"
					>
						<ShieldAlert class="h-5 w-5" />
					</div>
					<div>
						<h2 id="logout-all-title" class="text-base font-semibold text-neutral-900">
							Log Out From All Devices?
						</h2>
						<p class="text-xs text-neutral-500">Global session termination</p>
					</div>
				</div>
				<button
					type="button"
					onclick={() => !isLoading && onClose()}
					disabled={isLoading}
					class="rounded-full p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-900 disabled:opacity-50"
					aria-label="Close dialog"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<div class="mt-4 space-y-4">
				{#if error}
					<div
						role="alert"
						class="flex items-start gap-2.5 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-700"
					>
						<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-rose-500" />
						<span>{error}</span>
					</div>
				{/if}

				<p class="text-xs leading-relaxed text-neutral-600">
					This will <strong class="font-semibold text-neutral-950"
						>revoke all active sessions</strong
					> across every device registered to your mobile number. You will be signed out on this device
					and must authenticate again with a new OTP.
				</p>

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
						variant="outline"
						size="sm"
						onclick={handleConfirmLogoutAll}
						disabled={isLoading}
						class="border-rose-200 text-rose-700 hover:bg-rose-50"
					>
						{#if isLoading}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Revoking All Sessions...</span>
						{:else}
							<span>Confirm Logout All</span>
						{/if}
					</Button>
				</div>
			</div>
		</div>
	</div>
{/if}
