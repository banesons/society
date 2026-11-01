<script lang="ts">
	import ResponsiveModal from '$lib/components/ui/responsive-modal.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import { societyApi, type SocietyItem, type SuspensionReason } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { AlertTriangle, CheckCircle, ShieldAlert, Loader2, RefreshCw } from '@lucide/svelte';

	let {
		isOpen = false,
		society = null,
		targetStatus = 'ACTIVE',
		onClose,
		onSuccess
	}: {
		isOpen: boolean;
		society: SocietyItem | null;
		targetStatus: 'ACTIVE' | 'SUSPENDED';
		onClose: () => void;
		onSuccess?: () => void;
	} = $props();

	let reason = $state('');
	let selectedReasonId = $state<number | null>(null);
	let reasonDetails = $state('');
	let suspensionReasons = $state<SuspensionReason[]>([]);
	let isCatalogueLoading = $state(false);
	let catalogueError = $state<string | null>(null);
	let isLoading = $state(false);
	let error = $state<string | null>(null);

	const isSuspending = $derived(targetStatus === 'SUSPENDED');
	const isReactivating = $derived(
		targetStatus === 'ACTIVE' && society && society.status === 'SUSPENDED'
	);

	const selectedReason = $derived(suspensionReasons.find((r) => r.id === selectedReasonId));

	const isSubmitDisabled = $derived(
		isLoading ||
			(isSuspending &&
				(!selectedReasonId ||
					isCatalogueLoading ||
					!!catalogueError ||
					(selectedReason?.code === 'OTHER_REVIEW_REQUIRED' && !reasonDetails.trim())))
	);

	async function loadCatalogue() {
		isCatalogueLoading = true;
		catalogueError = null;
		try {
			suspensionReasons = await societyApi.getSuspensionReasons();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				catalogueError = err.message || 'Failed to load suspension reason catalogue.';
			} else {
				catalogueError = 'Failed to load suspension reason catalogue. Please retry.';
			}
		} finally {
			isCatalogueLoading = false;
		}
	}

	$effect(() => {
		if (isOpen && isSuspending && suspensionReasons.length === 0 && !isCatalogueLoading) {
			loadCatalogue();
		}
		if (!isOpen) {
			selectedReasonId = null;
			reasonDetails = '';
			reason = '';
			error = null;
		}
	});

	async function handleConfirm() {
		if (!society) return;
		error = null;

		if (isSuspending) {
			if (!selectedReasonId) {
				error = 'Please select a suspension reason.';
				return;
			}
			const trimmedDetails = reasonDetails.trim();
			if (selectedReason?.code === 'OTHER_REVIEW_REQUIRED' && !trimmedDetails) {
				error = 'Additional details are required when selecting "Other — requires review".';
				return;
			}
			if (trimmedDetails.length > 1000) {
				error = 'Additional details must not exceed 1000 characters.';
				return;
			}

			isLoading = true;
			try {
				await societyApi.suspendSociety(society.id, {
					reason_id: selectedReasonId,
					reason_details: trimmedDetails || undefined
				});
				selectedReasonId = null;
				reasonDetails = '';
				if (onSuccess) onSuccess();
				onClose();
			} catch (err: unknown) {
				if (err instanceof ApiClientError) {
					error = err.message || 'Failed to suspend society.';
				} else {
					error = 'An unexpected error occurred. Please try again.';
				}
			} finally {
				isLoading = false;
			}
			return;
		}

		const trimmedReason = reason.trim();
		isLoading = true;
		try {
			if (isReactivating) {
				await societyApi.reactivateSociety(society.id, trimmedReason || undefined);
			} else {
				await societyApi.activateSociety(society.id, trimmedReason || undefined);
			}

			reason = '';
			if (onSuccess) onSuccess();
			onClose();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message || 'Failed to update society status.';
			} else {
				error = 'An unexpected error occurred. Please try again.';
			}
		} finally {
			isLoading = false;
		}
	}

	const modalTitle = $derived.by(() => {
		if (isSuspending) return 'Suspend Society';
		if (isReactivating) return 'Reactivate Society';
		return 'Activate Society';
	});

	const modalDescription = $derived(
		society ? `Target: ${society.name} (${society.society_code})` : undefined
	);
</script>

<ResponsiveModal
	isOpen={isOpen && !!society}
	{onClose}
	title={modalTitle}
	description={modalDescription}
	maxWidth="md"
	isDismissible={!isLoading}
	closeOnBackdrop={!isLoading}
	closeOnEscape={!isLoading}
	showCloseButton={!isLoading}
	destructive={isSuspending}
>
	{#snippet icon()}
		<div
			class="flex h-9 w-9 items-center justify-center rounded-full {isSuspending
				? 'bg-rose-50 text-rose-600'
				: 'bg-emerald-50 text-emerald-600'}"
		>
			{#if isSuspending}
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

		{#if isSuspending}
			<p class="text-xs leading-relaxed text-neutral-700">
				Suspending this society will put tenant operations on administrative hold. The society
				profile remains intact and historical data is preserved.
			</p>
		{:else if isReactivating}
			<p class="text-xs leading-relaxed text-neutral-700">
				Reactivating this society will lift the administrative hold and restore active platform
				services.
			</p>
		{:else}
			<p class="text-xs leading-relaxed text-neutral-700">
				Activating this society approves its onboarding review and transitions status from <strong
					class="text-amber-700">PENDING</strong
				>
				to <strong class="text-emerald-700">ACTIVE</strong>.
			</p>
		{/if}

		{#if isSuspending}
			<div class="space-y-3.5">
				{#if isCatalogueLoading}
					<div
						class="flex items-center gap-2 rounded-xl border border-neutral-200 bg-neutral-50 p-3 text-xs text-neutral-500"
					>
						<Loader2 class="h-4 w-4 animate-spin text-neutral-900" />
						<span>Loading suspension reason catalogue...</span>
					</div>
				{:else if catalogueError}
					<div
						class="flex items-center justify-between gap-3 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
					>
						<span>{catalogueError}</span>
						<Button
							type="button"
							variant="outline"
							size="sm"
							onclick={loadCatalogue}
							class="border-rose-300 text-rose-800 hover:bg-rose-100"
						>
							<RefreshCw class="mr-1.5 h-3 w-3" />
							Retry
						</Button>
					</div>
				{:else}
					<div>
						<label for="suspension-reason" class="block text-xs font-medium text-neutral-700">
							Suspension reason *
						</label>
						<select
							id="suspension-reason"
							bind:value={selectedReasonId}
							disabled={isLoading}
							class="mt-1.5 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
						>
							<option value={null} disabled selected>Select a reason...</option>
							{#each suspensionReasons as r (r.id)}
								<option value={r.id}>{r.label}</option>
							{/each}
						</select>
						{#if selectedReason?.description}
							<p class="mt-1 text-[11px] text-neutral-500">{selectedReason.description}</p>
						{/if}
					</div>

					<div>
						<label for="suspension-details" class="block text-xs font-medium text-neutral-700">
							{selectedReason?.code === 'OTHER_REVIEW_REQUIRED'
								? 'Additional details *'
								: 'Additional details (optional)'}
						</label>
						<textarea
							id="suspension-details"
							bind:value={reasonDetails}
							rows="3"
							maxlength="1000"
							placeholder={selectedReason?.code === 'OTHER_REVIEW_REQUIRED'
								? 'Provide detailed justification for the suspension review...'
								: 'Optional context or reference (e.g., ticket ID or notice ref)...'}
							disabled={isLoading}
							class="mt-1.5 block w-full resize-none rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
						></textarea>
						{#if selectedReason?.code === 'OTHER_REVIEW_REQUIRED' && !reasonDetails.trim()}
							<p class="mt-1 text-[11px] text-amber-700">
								Additional details are required for this reason.
							</p>
						{/if}
					</div>
				{/if}
			</div>
		{:else}
			<div>
				<label for="status-reason" class="block text-xs font-medium text-neutral-700">
					Operational Reason (Optional, recorded in audit trail)
				</label>
				<input
					id="status-reason"
					type="text"
					bind:value={reason}
					placeholder="e.g., Onboarding review verified"
					disabled={isLoading}
					class="mt-1.5 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
				/>
			</div>
		{/if}
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
				variant={isSuspending ? 'destructive' : 'primary'}
				size="sm"
				onclick={handleConfirm}
				disabled={isSubmitDisabled}
			>
				{#if isLoading}
					<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					<span>Updating...</span>
				{:else}
					<span>
						{#if isSuspending}
							Confirm Suspension
						{:else if isReactivating}
							Confirm Reactivation
						{:else}
							Confirm Activation
						{/if}
					</span>
				{/if}
			</Button>
		</div>
	{/snippet}
</ResponsiveModal>
