<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import { societyApi, type SocietyItem, type SuspensionReason } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { AlertTriangle, CheckCircle, ShieldAlert, X, Loader2, RefreshCw } from '@lucide/svelte';

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
</script>

{#if isOpen && society}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6"
		role="dialog"
		aria-labelledby="status-society-title"
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
						class="flex h-9 w-9 items-center justify-center rounded-lg {isSuspending
							? 'bg-rose-500/10 text-rose-400'
							: 'bg-emerald-500/10 text-emerald-400'}"
					>
						{#if isSuspending}
							<ShieldAlert class="h-5 w-5" />
						{:else}
							<CheckCircle class="h-5 w-5" />
						{/if}
					</div>
					<div>
						<h2 id="status-society-title" class="text-base font-semibold text-slate-100">
							{#if isSuspending}
								Suspend Society
							{:else if isReactivating}
								Reactivate Society
							{:else}
								Activate Society
							{/if}
						</h2>
						<p class="text-xs text-slate-400">
							Target: <span class="font-medium text-slate-200">{society.name}</span> (<span
								class="font-mono text-indigo-300">{society.society_code}</span
							>)
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

				{#if isSuspending}
					<p class="text-xs leading-relaxed text-slate-300">
						Suspending this society will put tenant operations on administrative hold. The society
						profile remains intact and historical data is preserved.
					</p>
				{:else if isReactivating}
					<p class="text-xs leading-relaxed text-slate-300">
						Reactivating this society will lift the administrative hold and restore active platform
						services.
					</p>
				{:else}
					<p class="text-xs leading-relaxed text-slate-300">
						Activating this society approves its onboarding review and transitions status from <strong
							class="text-amber-400">PENDING</strong
						>
						to <strong class="text-emerald-400">ACTIVE</strong>.
					</p>
				{/if}

				{#if isSuspending}
					<div class="space-y-3.5">
						{#if isCatalogueLoading}
							<div
								class="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-950/50 p-3 text-xs text-slate-400"
							>
								<Loader2 class="h-4 w-4 animate-spin text-indigo-400" />
								<span>Loading suspension reason catalogue...</span>
							</div>
						{:else if catalogueError}
							<div
								class="flex items-center justify-between gap-3 rounded-lg border border-rose-500/30 bg-rose-500/10 p-3 text-xs text-rose-300"
							>
								<span>{catalogueError}</span>
								<Button
									type="button"
									variant="outline"
									size="sm"
									onclick={loadCatalogue}
									class="border-rose-500/40 text-rose-300 hover:bg-rose-500/20"
								>
									<RefreshCw class="mr-1.5 h-3 w-3" />
									Retry
								</Button>
							</div>
						{:else}
							<div>
								<label for="suspension-reason" class="block text-xs font-medium text-slate-300">
									Suspension reason *
								</label>
								<select
									id="suspension-reason"
									bind:value={selectedReasonId}
									disabled={isLoading}
									class="mt-1.5 block w-full rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 focus:border-indigo-500 focus:outline-none disabled:opacity-50"
								>
									<option value={null} disabled selected>Select a reason...</option>
									{#each suspensionReasons as r (r.id)}
										<option value={r.id}>{r.label}</option>
									{/each}
								</select>
								{#if selectedReason?.description}
									<p class="mt-1 text-[11px] text-slate-400">{selectedReason.description}</p>
								{/if}
							</div>

							<div>
								<label for="suspension-details" class="block text-xs font-medium text-slate-300">
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
									class="mt-1.5 block w-full resize-none rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none disabled:opacity-50"
								></textarea>
								{#if selectedReason?.code === 'OTHER_REVIEW_REQUIRED' && !reasonDetails.trim()}
									<p class="mt-1 text-[11px] text-amber-400">
										Additional details are required for this reason.
									</p>
								{/if}
							</div>
						{/if}
					</div>
				{:else}
					<div>
						<label for="status-reason" class="block text-xs font-medium text-slate-300">
							Operational Reason (Optional, recorded in audit trail)
						</label>
						<input
							id="status-reason"
							type="text"
							bind:value={reason}
							placeholder="e.g., Onboarding review verified"
							disabled={isLoading}
							class="mt-1.5 block w-full rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none disabled:opacity-50"
						/>
					</div>
				{/if}

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
						variant={isSuspending ? 'outline' : 'primary'}
						size="sm"
						onclick={handleConfirm}
						disabled={isSubmitDisabled}
						class={isSuspending ? 'border-rose-500/50 text-rose-400 hover:bg-rose-500/20' : ''}
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
			</div>
		</div>
	</div>
{/if}
