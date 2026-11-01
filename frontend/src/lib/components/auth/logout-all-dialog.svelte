<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import ResponsiveModal from '$lib/components/ui/responsive-modal.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import { ShieldAlert, AlertTriangle, Loader2 } from '@lucide/svelte';

	interface Props {
		isOpen: boolean;
		onClose: () => void;
	}

	let { isOpen = false, onClose }: Props = $props();

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
</script>

<ResponsiveModal
	{isOpen}
	{onClose}
	title="Log Out From All Devices?"
	description="Global session termination"
	maxWidth="md"
	isDismissible={!isLoading}
	closeOnBackdrop={!isLoading}
	closeOnEscape={!isLoading}
	showCloseButton={!isLoading}
	destructive={true}
>
	{#snippet icon()}
		<div class="flex h-9 w-9 items-center justify-center rounded-full bg-rose-50 text-rose-600">
			<ShieldAlert class="h-5 w-5" />
		</div>
	{/snippet}

	<div class="space-y-4">
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
			This will <strong class="font-semibold text-neutral-950">revoke all active sessions</strong>
			across every device registered to your mobile number. You will be signed out on this device and
			must authenticate again with a new OTP.
		</p>
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
	{/snippet}
</ResponsiveModal>
