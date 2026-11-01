<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import ResponsiveModal from '$lib/components/ui/responsive-modal.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import { LogOut, AlertTriangle, Loader2 } from '@lucide/svelte';

	interface Props {
		isOpen: boolean;
		onClose: () => void;
	}

	let { isOpen = false, onClose }: Props = $props();

	let isLoading = $state(false);
	let error = $state<string | null>(null);

	async function handleConfirmSignOut() {
		if (isLoading) return;
		isLoading = true;
		error = null;
		try {
			await auth.logout();
			onClose();
			await goto('/login', { replaceState: true });
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to sign out. Please try again.';
			isLoading = false;
		}
	}
</script>

<ResponsiveModal
	{isOpen}
	{onClose}
	title="Sign out?"
	description="You will be signed out of this device and will need to authenticate again to access your account."
	maxWidth="sm"
	isDismissible={!isLoading}
	closeOnBackdrop={!isLoading}
	closeOnEscape={!isLoading}
	showCloseButton={!isLoading}
	destructive={false}
>
	{#snippet icon()}
		<div
			class="flex h-9 w-9 items-center justify-center rounded-full bg-neutral-100 text-neutral-800"
		>
			<LogOut class="h-4 w-4" />
		</div>
	{/snippet}

	<div class="space-y-3">
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
			Are you sure you want to sign out of your session on this browser? Your active session
			credentials will be cleared.
		</p>
	</div>

	{#snippet footer()}
		<div class="flex items-center justify-end gap-2.5">
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
				variant="primary"
				size="sm"
				onclick={handleConfirmSignOut}
				disabled={isLoading}
			>
				{#if isLoading}
					<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					<span>Signing Out...</span>
				{:else}
					<span>Sign Out</span>
				{/if}
			</Button>
		</div>
	{/snippet}
</ResponsiveModal>
