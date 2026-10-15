<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import {
		societyApi,
		type SocietyMember,
		type DeparturePreviewResponse,
		type DepartureReasonCode,
		DEPARTURE_REASON_OPTIONS
	} from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import {
		UserMinus,
		X,
		AlertCircle,
		CheckCircle2,
		Loader2,
		RefreshCw,
		Home,
		KeyRound
	} from '@lucide/svelte';

	let {
		isOpen = false,
		societyId,
		member,
		onClose,
		onSuccess
	}: {
		isOpen: boolean;
		societyId: string;
		member: SocietyMember | null;
		onClose: () => void;
		onSuccess?: () => void;
	} = $props();

	let preview = $state<DeparturePreviewResponse | null>(null);
	let isLoadingPreview = $state(false);
	let isSubmitting = $state(false);
	let error = $state<string | null>(null);
	let isStale = $state(false);

	let endDate = $state(new Date().toISOString().split('T')[0]);
	let reasonCode = $state<DepartureReasonCode | ''>('');
	let reasonDetails = $state('');
	let validationErrors = $state<{
		reasonCode?: string;
		reasonDetails?: string;
		endDate?: string;
	}>({});

	async function loadPreview() {
		if (!member) return;
		isLoadingPreview = true;
		error = null;
		isStale = false;

		try {
			const res = await societyApi.previewMemberDeparture(societyId, member.id);
			preview = res;
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message || 'Failed to generate departure preview.';
			} else {
				error = 'Failed to load departure preview.';
			}
		} finally {
			isLoadingPreview = false;
		}
	}

	$effect(() => {
		if (isOpen && member) {
			endDate = new Date().toISOString().split('T')[0];
			reasonCode = '';
			reasonDetails = '';
			validationErrors = {};
			error = null;
			isStale = false;
			loadPreview();
		}
	});

	async function handleConfirm(e: SubmitEvent) {
		e.preventDefault();
		if (!member || !preview) return;

		validationErrors = {};

		if (!reasonCode) {
			validationErrors.reasonCode = 'Please select a departure reason.';
		}

		const trimmedDetails = reasonDetails.trim();
		if (reasonCode === 'OTHER' && !trimmedDetails) {
			validationErrors.reasonDetails = 'An explanation is required when "Other" is selected.';
		} else if (trimmedDetails.length > 500) {
			validationErrors.reasonDetails = 'Explanation must not exceed 500 characters.';
		}

		if (!endDate) {
			validationErrors.endDate = 'Effective departure date is required.';
		}

		if (Object.keys(validationErrors).length > 0) {
			return;
		}

		isSubmitting = true;
		error = null;

		try {
			await societyApi.confirmMemberDeparture(societyId, member.id, {
				expected_version_hash: preview.version_hash,
				end_date: endDate,
				reason_code: reasonCode as DepartureReasonCode,
				reason_details: trimmedDetails || undefined
			});

			onSuccess?.();
			onClose();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.statusCode === 409 || err.code === 'PREVIEW_STALE') {
					isStale = true;
					error =
						'Member unit relationships changed while preview was open. Please refresh preview to confirm current changes.';
				} else {
					error = err.message || 'Failed to confirm departure.';
				}
			} else {
				error = 'Failed to confirm departure.';
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

{#if isOpen && member}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
		role="dialog"
		aria-modal="true"
		onclick={(e) => {
			if (e.target === e.currentTarget) onClose();
		}}
	>
		<div
			class="relative max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-3xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div class="flex items-center gap-3">
					<div
						class="flex h-10 w-10 items-center justify-center rounded-full bg-amber-50 text-amber-700"
					>
						<UserMinus class="h-5 w-5" />
					</div>
					<div>
						<h3 class="text-base font-semibold tracking-tight text-neutral-950">
							Member Departure Workflow
						</h3>
						<p class="text-xs text-neutral-500">
							Review and record departure for {member.first_name}
							{member.last_name || ''}
						</p>
					</div>
				</div>
				<button
					onclick={onClose}
					class="rounded-full p-1.5 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			{#if isLoadingPreview}
				<div class="flex min-h-[220px] flex-col items-center justify-center text-neutral-500">
					<Loader2 class="h-8 w-8 animate-spin text-neutral-900" />
					<p class="mt-3 text-xs">Computing departure impact and snapshot hash...</p>
				</div>
			{:else}
				{#if error}
					<div
						class="mt-4 flex items-start gap-2.5 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
					>
						<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
						<div class="flex-1">
							<span>{error}</span>
							{#if isStale}
								<div class="mt-2">
									<Button variant="outline" size="sm" onclick={loadPreview}>
										<RefreshCw class="mr-1.5 h-3.5 w-3.5" />
										<span>Refresh Preview</span>
									</Button>
								</div>
							{/if}
						</div>
					</div>
				{/if}

				{#if preview}
					<!-- Manifest Summary -->
					<div class="mt-4 rounded-2xl border border-neutral-200 bg-neutral-50/70 p-4">
						<div class="flex items-center justify-between">
							<span class="text-xs font-semibold text-neutral-900">Departure Impact Manifest</span>
							<Badge variant="outline" class="bg-white font-mono text-[10px] text-neutral-700">
								Hash: {preview.version_hash.slice(0, 10)}...
							</Badge>
						</div>

						<div class="mt-4 grid grid-cols-3 gap-3 text-center">
							<div class="rounded-xl border border-neutral-200 bg-white p-3">
								<span class="text-[10px] text-neutral-500">Total Associations</span>
								<p class="mt-0.5 text-base font-bold text-neutral-950">
									{(preview.preserved_ownership_relationships?.length ?? 0) +
										(preview.targeted_terminations?.length ?? 0)}
								</p>
							</div>
							<div class="rounded-xl border border-neutral-200 bg-white p-3">
								<span class="text-[10px] text-neutral-500">Terminated Leases</span>
								<p class="mt-0.5 text-base font-bold text-amber-700">
									{preview.targeted_terminations?.length ?? 0}
								</p>
							</div>
							<div class="rounded-xl border border-neutral-200 bg-white p-3">
								<span class="text-[10px] text-neutral-500">Preserved Records</span>
								<p class="mt-0.5 text-base font-bold text-emerald-700">
									{preview.preserved_ownership_relationships?.length ?? 0}
								</p>
							</div>
						</div>

						<!-- Terminations List -->
						{#if preview.targeted_terminations && preview.targeted_terminations.length > 0}
							<div class="mt-4">
								<span class="text-[11px] font-semibold tracking-wider text-amber-800 uppercase">
									Tenancy & Occupancy Terminations (Set to ENDED)
								</span>
								<div class="mt-1 space-y-1.5">
									{#each preview.targeted_terminations as term (term.relationship_id)}
										<div
											class="flex items-center justify-between rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-800"
										>
											<div class="flex items-center gap-2">
												<KeyRound class="h-3.5 w-3.5 text-amber-600" />
												<span class="font-semibold text-neutral-950">{term.unit_identifier}</span>
												<span class="text-neutral-400">·</span>
												<span>{term.relationship_type}</span>
											</div>
											<Badge
												variant="outline"
												class="border-amber-200 bg-amber-50 text-[10px] text-amber-800"
												>Will End</Badge
											>
										</div>
									{/each}
								</div>
							</div>
						{/if}

						<!-- Preserved Ownerships List -->
						{#if preview.preserved_ownership_relationships && preview.preserved_ownership_relationships.length > 0}
							<div class="mt-4">
								<span class="text-[11px] font-semibold tracking-wider text-emerald-800 uppercase">
									Ownership Rights Preserved (Title Retained, Occupancy Cleared)
								</span>
								<div class="mt-1 space-y-1.5">
									{#each preview.preserved_ownership_relationships as own (own.relationship_id)}
										<div
											class="flex items-center justify-between rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-800"
										>
											<div class="flex items-center gap-2">
												<Home class="h-3.5 w-3.5 text-emerald-600" />
												<span class="font-semibold text-neutral-950">{own.unit_identifier}</span>
												<span class="text-neutral-400">·</span>
												<span
													>{own.relationship_type}
													{own.ownership_share != null ? `(${own.ownership_share}%)` : ''}</span
												>
											</div>
											<Badge
												variant="outline"
												class="border-emerald-200 bg-emerald-50 text-[10px] text-emerald-800"
												>Retained</Badge
											>
										</div>
									{/each}
								</div>
							</div>
						{/if}
					</div>

					<!-- Confirmation Form -->
					<form onsubmit={handleConfirm} class="mt-4 space-y-4">
						<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
							<div>
								<label for="departure-date" class="block text-xs font-medium text-neutral-700"
									>Effective Departure Date *</label
								>
								<input
									id="departure-date"
									type="date"
									required
									bind:value={endDate}
									class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-neutral-950 focus:outline-hidden"
								/>
								{#if validationErrors.endDate}
									<p class="mt-1 text-xs text-rose-600">{validationErrors.endDate}</p>
								{/if}
							</div>
							<div>
								<label
									for="departure-reason-code"
									class="block text-xs font-medium text-neutral-700">Departure Reason *</label
								>
								<select
									id="departure-reason-code"
									required
									bind:value={reasonCode}
									class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-neutral-950 focus:outline-hidden"
								>
									<option value="" disabled>Select departure reason...</option>
									{#each DEPARTURE_REASON_OPTIONS as opt (opt.code)}
										<option value={opt.code}>{opt.label}</option>
									{/each}
								</select>
								{#if validationErrors.reasonCode}
									<p class="mt-1 text-xs text-rose-600">{validationErrors.reasonCode}</p>
								{/if}
							</div>
						</div>

						{#if reasonCode === 'OTHER'}
							<div class="space-y-1">
								<div class="flex items-center justify-between">
									<label
										for="departure-reason-details"
										class="block text-xs font-medium text-neutral-700"
									>
										Reason Explanation * <span class="text-neutral-500">(Required for 'Other')</span
										>
									</label>
									<span class="text-[11px] text-neutral-400">
										{reasonDetails.length}/500
									</span>
								</div>
								<textarea
									id="departure-reason-details"
									bind:value={reasonDetails}
									rows="3"
									maxlength="500"
									placeholder="Provide specific justification or departure context..."
									class="w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
								></textarea>
								{#if validationErrors.reasonDetails}
									<p class="text-xs text-rose-600">{validationErrors.reasonDetails}</p>
								{/if}
							</div>
						{/if}

						<div
							class="rounded-2xl border border-amber-200 bg-amber-50/70 p-3.5 text-[11px] leading-relaxed text-amber-900"
						>
							<strong>Two-Phase Protection:</strong> This confirmation validates that no
							relationships changed since the preview was calculated. Membership status will
							transition to <strong>LEFT</strong> and audit trails will record the departure reason code
							and metadata.
						</div>

						<div class="mt-6 flex items-center justify-end gap-3 border-t border-neutral-100 pt-4">
							<Button variant="ghost" size="sm" onclick={onClose} disabled={isSubmitting}>
								Cancel
							</Button>
							<Button variant="primary" size="sm" type="submit" disabled={isSubmitting || isStale}>
								{#if isSubmitting}
									<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
									<span>Processing Departure...</span>
								{:else}
									<CheckCircle2 class="mr-1.5 h-3.5 w-3.5" />
									<span>Confirm Member Departure</span>
								{/if}
							</Button>
						</div>
					</form>
				{/if}
			{/if}
		</div>
	</div>
{/if}
