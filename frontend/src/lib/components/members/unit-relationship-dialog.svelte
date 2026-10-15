<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import {
		societyApi,
		type UnitRelationship,
		type SocietyMember,
		type CreateRelationshipPayload
	} from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { formatFullPhone } from '$lib/phone';
	import { Home, KeyRound, Plus, X, AlertCircle, Loader2, UserCheck } from '@lucide/svelte';

	let {
		isOpen = false,
		societyId,
		unitId,
		unitIdentifier = 'Unit',
		onClose,
		onRefresh
	}: {
		isOpen: boolean;
		societyId: string;
		unitId: string | null;
		unitIdentifier?: string;
		onClose: () => void;
		onRefresh?: () => void;
	} = $props();

	let relationships = $state<UnitRelationship[]>([]);
	let totalOwnership = $state(0);
	let isVacant = $state(true);
	let members = $state<SocietyMember[]>([]);

	let isLoading = $state(false);
	let error = $state<string | null>(null);

	// Add Form State
	let showAddForm = $state(false);
	let selectedMemberId = $state('');
	let selectedRelType = $state('PRIMARY_OWNER');
	let isOccupant = $state(true);
	let ownershipShare = $state<number | undefined>(undefined);
	let startDate = $state(new Date().toISOString().split('T')[0]);
	let notes = $state('');
	let isSubmitting = $state(false);

	// End Rel State
	let endingRelId = $state<string | null>(null);
	let endingDate = $state(new Date().toISOString().split('T')[0]);
	let endingReason = $state('');
	let isEnding = $state(false);

	const isOwnerType = $derived(
		selectedRelType === 'PRIMARY_OWNER' || selectedRelType === 'SECONDARY_OWNER'
	);
	const isOccupancyMandatory = $derived(selectedRelType === 'AUTHORIZED_OCCUPANT');

	$effect(() => {
		if (isOccupancyMandatory) {
			isOccupant = true;
		}
	});

	async function loadUnitData() {
		if (!unitId) return;
		isLoading = true;
		error = null;

		try {
			const [unitRes, memRes] = await Promise.all([
				societyApi.listUnitRelationships(societyId, unitId),
				societyApi.listMembers(societyId, { limit: 100, status: 'ACTIVE' })
			]);

			relationships = unitRes.relationships;
			totalOwnership = unitRes.total_ownership_share;
			isVacant = unitRes.is_vacant;
			members = memRes.members;
			if (members.length > 0 && !selectedMemberId) {
				selectedMemberId = members[0].id;
			}
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message || 'Failed to load unit relationships.';
			} else {
				error = 'Failed to load relationships.';
			}
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (isOpen && unitId) {
			showAddForm = false;
			endingRelId = null;
			loadUnitData();
		}
	});

	async function handleAddRelationship(e: SubmitEvent) {
		e.preventDefault();
		if (!unitId) return;

		if (!selectedMemberId) {
			error = 'Please select a member.';
			return;
		}

		if (isOwnerType && ownershipShare != null && ownershipShare > 0) {
			if (totalOwnership + ownershipShare > 100) {
				error = `Ownership share cannot exceed 100%. Current total is ${totalOwnership}%. Available: ${100 - totalOwnership}%.`;
				return;
			}
		}

		isSubmitting = true;
		error = null;

		try {
			const payload: CreateRelationshipPayload = {
				member_id: selectedMemberId,
				relationship_type: selectedRelType,
				is_occupant: isOccupancyMandatory ? true : isOccupant,
				ownership_share: isOwnerType ? ownershipShare : undefined,
				start_date: startDate || undefined,
				notes: notes.trim() || undefined
			};

			await societyApi.createUnitRelationship(societyId, unitId, payload);
			showAddForm = false;
			ownershipShare = undefined;
			notes = '';
			await loadUnitData();
			onRefresh?.();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.code === 'TOTAL_OWNERSHIP_EXCEEDED') {
					error = 'Total ownership share for this unit cannot exceed 100%.';
				} else if (err.code === 'RELATIONSHIP_ALREADY_EXISTS') {
					error = 'An active relationship already exists between this member and unit.';
				} else {
					error = err.message || 'Failed to create association.';
				}
			} else {
				error = 'Failed to create association.';
			}
		} finally {
			isSubmitting = false;
		}
	}

	async function handleEndRelationship(relId: string) {
		if (!unitId) return;
		isEnding = true;
		error = null;

		try {
			await societyApi.endUnitRelationship(societyId, unitId, relId, {
				end_date: endingDate,
				reason: endingReason.trim() || undefined
			});

			endingRelId = null;
			endingReason = '';
			await loadUnitData();
			onRefresh?.();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message || 'Failed to end relationship.';
			} else {
				error = 'Failed to end relationship.';
			}
		} finally {
			isEnding = false;
		}
	}
</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape' && isOpen) onClose();
	}}
/>

{#if isOpen && unitId}
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
						class="flex h-10 w-10 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
					>
						<Home class="h-5 w-5" />
					</div>
					<div>
						<h3 class="text-base font-semibold tracking-tight text-neutral-950">
							Unit {unitIdentifier}
						</h3>
						<p class="text-xs text-neutral-500">Ownership & Tenancy Association Console</p>
					</div>
				</div>
				<button
					onclick={onClose}
					class="rounded-full p-1.5 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			{#if isLoading}
				<div class="flex min-h-[250px] flex-col items-center justify-center text-neutral-500">
					<Loader2 class="h-8 w-8 animate-spin text-neutral-900" />
					<p class="mt-3 text-xs">Loading unit occupancy & relationships...</p>
				</div>
			{:else}
				{#if error}
					<div
						class="mt-4 flex items-start gap-2.5 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
					>
						<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
						<span>{error}</span>
					</div>
				{/if}

				<!-- Status Banner -->
				<div
					class="mt-4 grid grid-cols-2 gap-3 rounded-2xl border border-neutral-200 bg-neutral-50/70 p-3.5"
				>
					<div>
						<span class="text-[10px] text-neutral-500">Occupancy Status</span>
						<div class="mt-1 flex items-center gap-2">
							<Badge variant={isVacant ? 'warning' : 'success'} class="text-xs">
								{isVacant ? 'Vacant (No Resident)' : 'Occupied'}
							</Badge>
						</div>
					</div>
					<div>
						<span class="text-[10px] text-neutral-500">Allocated Ownership Share</span>
						<div class="mt-1 flex items-center gap-2">
							<span
								class="font-mono text-sm font-bold {totalOwnership > 100
									? 'text-rose-600'
									: 'text-emerald-700'}"
							>
								{totalOwnership}% / 100%
							</span>
							{#if totalOwnership < 100}
								<span class="text-[10px] text-neutral-500"
									>({100 - totalOwnership}% unallocated)</span
								>
							{/if}
						</div>
					</div>
				</div>

				<!-- Relationships List -->
				<div class="mt-6">
					<div class="flex items-center justify-between">
						<h4 class="text-xs font-semibold tracking-wider text-neutral-500 uppercase">
							Active & Historical Associations ({relationships.length})
						</h4>
						{#if !showAddForm}
							<Button variant="outline" size="sm" onclick={() => (showAddForm = true)}>
								<Plus class="mr-1.5 h-3.5 w-3.5" />
								<span>Add Association</span>
							</Button>
						{/if}
					</div>

					{#if showAddForm}
						<div class="mt-4 rounded-2xl border border-neutral-300 bg-neutral-50 p-4">
							<div class="flex items-center justify-between border-b border-neutral-200 pb-2">
								<h5 class="text-xs font-semibold text-neutral-900">New Unit Association</h5>
								<button
									onclick={() => (showAddForm = false)}
									class="text-xs text-neutral-500 transition-colors hover:text-neutral-900"
								>
									Cancel
								</button>
							</div>

							<form onsubmit={handleAddRelationship} class="mt-3 space-y-3">
								<div class="grid grid-cols-2 gap-3">
									<div>
										<label for="rel-member" class="block text-[11px] font-medium text-neutral-700"
											>Member *</label
										>
										<select
											id="rel-member"
											required
											bind:value={selectedMemberId}
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-hidden"
										>
											{#each members as m (m.id)}
												<option value={m.id}
													>{m.first_name}
													{m.last_name || ''} ({formatFullPhone(m.phone || m.phone_masked)})</option
												>
											{/each}
										</select>
									</div>
									<div>
										<label for="rel-type" class="block text-[11px] font-medium text-neutral-700"
											>Relationship Type *</label
										>
										<select
											id="rel-type"
											required
											bind:value={selectedRelType}
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-hidden"
										>
											<option value="PRIMARY_OWNER">PRIMARY_OWNER</option>
											<option value="SECONDARY_OWNER">SECONDARY_OWNER</option>
											<option value="OWNER_FAMILY">OWNER_FAMILY</option>
											<option value="PRIMARY_TENANT">PRIMARY_TENANT</option>
											<option value="SECONDARY_TENANT">SECONDARY_TENANT</option>
											<option value="TENANT_FAMILY">TENANT_FAMILY</option>
											<option value="AUTHORIZED_OCCUPANT">AUTHORIZED_OCCUPANT</option>
										</select>
									</div>
								</div>

								<div class="grid grid-cols-3 gap-3">
									<div class="flex items-center gap-2 pt-5">
										<input
											type="checkbox"
											id="is-occupant"
											disabled={isOccupancyMandatory}
											bind:checked={isOccupant}
											class="h-4 w-4 rounded border-neutral-300 text-neutral-950 focus:ring-neutral-950 disabled:opacity-60"
										/>
										<label for="is-occupant" class="text-xs text-neutral-700"
											>Resident Occupant</label
										>
									</div>

									<div>
										<label for="share-pct" class="block text-[11px] font-medium text-neutral-700">
											Ownership % {isOwnerType ? '(Optional)' : '(Disabled)'}
										</label>
										<input
											id="share-pct"
											type="number"
											step="0.01"
											min="0"
											max="100"
											disabled={!isOwnerType}
											bind:value={ownershipShare}
											placeholder={isOwnerType ? 'e.g. 50' : 'N/A'}
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-1.5 font-mono text-xs text-neutral-900 focus:border-neutral-950 focus:outline-hidden disabled:opacity-40"
										/>
									</div>

									<div>
										<label for="start-date" class="block text-[11px] font-medium text-neutral-700"
											>Start Date</label
										>
										<input
											id="start-date"
											type="date"
											bind:value={startDate}
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-hidden"
										/>
									</div>
								</div>

								<div>
									<label for="rel-notes" class="block text-[11px] font-medium text-neutral-700"
										>Notes</label
									>
									<input
										id="rel-notes"
										type="text"
										bind:value={notes}
										placeholder="e.g. Lease agreement reference or registry number"
										class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
									/>
								</div>

								<div class="flex justify-end gap-2 border-t border-neutral-200 pt-2">
									<Button
										variant="ghost"
										size="sm"
										onclick={() => (showAddForm = false)}
										disabled={isSubmitting}
									>
										Cancel
									</Button>
									<Button variant="primary" size="sm" type="submit" disabled={isSubmitting}>
										{#if isSubmitting}
											<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
											<span>Associating...</span>
										{:else}
											<UserCheck class="mr-1.5 h-3.5 w-3.5" />
											<span>Confirm Association</span>
										{/if}
									</Button>
								</div>
							</form>
						</div>
					{/if}

					<!-- Existing Relationships Table -->
					<div class="mt-3 space-y-2">
						{#if relationships.length === 0}
							<div
								class="rounded-2xl border border-dashed border-neutral-200 bg-neutral-50/50 p-6 text-center text-xs text-neutral-500"
							>
								No relationships registered for this unit.
							</div>
						{:else}
							{#each relationships as rel (rel.id)}
								<div class="rounded-2xl border border-neutral-200 bg-neutral-50/60 p-3.5">
									<div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
										<div class="flex items-center gap-3">
											<div
												class="flex h-9 w-9 items-center justify-center rounded-full {rel.relationship_type.includes(
													'OWNER'
												)
													? 'bg-emerald-100 text-emerald-800'
													: 'bg-neutral-100 text-neutral-800'}"
											>
												{#if rel.relationship_type.includes('OWNER')}
													<Home class="h-4 w-4" />
												{:else}
													<KeyRound class="h-4 w-4" />
												{/if}
											</div>
											<div>
												<div class="flex items-center gap-2">
													<span class="text-xs font-semibold text-neutral-950">
														{rel.first_name}
														{rel.last_name || ''}
													</span>
													<Badge variant="outline" class="bg-white text-[10px]">
														{rel.relationship_type}
													</Badge>
													{#if rel.ownership_share != null}
														<span class="font-mono text-[10px] font-semibold text-emerald-700">
															{rel.ownership_share}% Share
														</span>
													{/if}
												</div>
												<div class="mt-0.5 flex items-center gap-2 text-[11px] text-neutral-500">
													<span class="font-mono"
														>{formatFullPhone(rel.phone || rel.phone_masked)}</span
													>
													<span class="text-neutral-300">·</span>
													<span>{rel.is_occupant ? 'Occupant' : 'Non-occupant'}</span>
													<span class="text-neutral-300">·</span>
													<span>From {rel.start_date || '—'}</span>
													{#if rel.end_date}
														<span class="text-neutral-300">·</span>
														<span class="text-rose-600">Ended {rel.end_date}</span>
													{/if}
												</div>
											</div>
										</div>

										<div class="flex items-center gap-2">
											<Badge
												variant={rel.status === 'ACTIVE' ? 'success' : 'destructive'}
												class="text-[10px]"
											>
												{rel.status}
											</Badge>
											{#if rel.status === 'ACTIVE' && endingRelId !== rel.id}
												<button
													onclick={() => {
														endingRelId = rel.id;
														endingDate = new Date().toISOString().split('T')[0];
														endingReason = '';
													}}
													class="rounded-full px-2.5 py-1 text-[11px] font-medium text-rose-700 transition-colors hover:bg-rose-50"
												>
													End
												</button>
											{/if}
										</div>
									</div>

									{#if endingRelId === rel.id}
										<div class="mt-3 border-t border-neutral-200 pt-3">
											<p class="text-[11px] font-semibold text-rose-700">
												Terminate Relationship Record
											</p>
											<div class="mt-2 grid grid-cols-2 gap-2">
												<div>
													<label for="term-date" class="block text-[10px] text-neutral-500"
														>Effective End Date</label
													>
													<input
														id="term-date"
														type="date"
														bind:value={endingDate}
														class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-2 py-1 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-hidden"
													/>
												</div>
												<div>
													<label for="term-reason" class="block text-[10px] text-neutral-500"
														>Reason</label
													>
													<input
														id="term-reason"
														type="text"
														bind:value={endingReason}
														placeholder="e.g. Sale of flat, Lease termination"
														class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-2 py-1 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-hidden"
													/>
												</div>
											</div>
											<div class="mt-2 flex justify-end gap-2">
												<button
													onclick={() => (endingRelId = null)}
													class="rounded-full px-2.5 py-1 text-[11px] text-neutral-500 transition-colors hover:text-neutral-900"
												>
													Cancel
												</button>
												<button
													onclick={() => handleEndRelationship(rel.id)}
													disabled={isEnding}
													class="rounded-full bg-rose-600 px-3 py-1 text-[11px] font-semibold text-white transition-colors hover:bg-rose-700 disabled:opacity-50"
												>
													{isEnding ? 'Terminating...' : 'Confirm Termination'}
												</button>
											</div>
										</div>
									{/if}
								</div>
							{/each}
						{/if}
					</div>
				</div>
			{/if}

			<div class="mt-6 flex items-center justify-end border-t border-neutral-100 pt-4">
				<Button variant="ghost" size="sm" onclick={onClose}>Close</Button>
			</div>
		</div>
	</div>
{/if}
