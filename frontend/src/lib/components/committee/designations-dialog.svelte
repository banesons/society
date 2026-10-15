<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import {
		societyApi,
		type SocietyDesignation,
		type CreateDesignationPayload
	} from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import {
		Layers,
		Plus,
		X,
		AlertCircle,
		CheckCircle2,
		Loader2,
		ToggleLeft,
		ToggleRight
	} from '@lucide/svelte';

	let {
		isOpen = false,
		societyId,
		onClose,
		onUpdated
	}: {
		isOpen: boolean;
		societyId: string;
		onClose: () => void;
		onUpdated?: () => void;
	} = $props();

	let designations = $state<SocietyDesignation[]>([]);
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let statusMessage = $state<string | null>(null);

	// New designation form state
	let showCreateForm = $state(false);
	let isCreating = $state(false);
	let newCode = $state('');
	let newName = $state('');
	let newDescription = $state('');
	let allowsMultiple = $state(false);
	let displayOrder = $state(10);
	let formErrors = $state<{
		code?: string;
		name?: string;
	}>({});

	// Toggling status state
	let togglingId = $state<string | null>(null);

	async function loadDesignations() {
		if (!societyId) return;
		isLoading = true;
		error = null;

		try {
			const res = await societyApi.listDesignations(societyId, true);
			designations = res.designations;
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message || 'Failed to load designations catalogue.';
			} else {
				error = 'Failed to load designations.';
			}
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (isOpen) {
			showCreateForm = false;
			newCode = '';
			newName = '';
			newDescription = '';
			allowsMultiple = false;
			displayOrder = 10;
			error = null;
			statusMessage = null;
			formErrors = {};
			loadDesignations();
		}
	});

	async function handleToggleStatus(d: SocietyDesignation) {
		togglingId = d.id;
		error = null;
		statusMessage = null;

		try {
			const updated = await societyApi.setDesignationStatus(societyId, d.id, !d.is_active);
			designations = designations.map((item) => (item.id === d.id ? updated.designation : item));
			statusMessage = `Designation "${d.name}" is now ${updated.designation.is_active ? 'ACTIVE' : 'INACTIVE'}.`;
			onUpdated?.();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message || 'Failed to update designation status.';
			} else {
				error = 'Failed to toggle status.';
			}
		} finally {
			togglingId = null;
		}
	}

	async function handleCreateDesignation(e: SubmitEvent) {
		e.preventDefault();
		formErrors = {};
		error = null;
		statusMessage = null;

		const cleanCode = newCode.trim().toUpperCase().replace(/\s+/g, '_');
		const cleanName = newName.trim();

		if (!cleanCode) {
			formErrors.code = 'Designation code is required.';
		}
		if (!cleanName) {
			formErrors.name = 'Display name is required.';
		}

		if (Object.keys(formErrors).length > 0) return;

		isCreating = true;

		try {
			const payload: CreateDesignationPayload = {
				code: cleanCode,
				name: cleanName,
				description: newDescription.trim() || undefined,
				allows_multiple_members: allowsMultiple,
				allows_multiple_designations: true,
				display_order: Number(displayOrder) || 10
			};

			const res = await societyApi.createDesignation(societyId, payload);
			designations = [...designations, res.designation];
			statusMessage = `Successfully created designation "${res.designation.name}".`;
			showCreateForm = false;
			newCode = '';
			newName = '';
			newDescription = '';
			allowsMultiple = false;
			displayOrder = 10;
			onUpdated?.();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.code === 'DESIGNATION_CODE_EXISTS' || err.statusCode === 409) {
					formErrors.code = 'A designation with this code already exists in this society.';
				} else {
					error = err.message || 'Failed to create designation.';
				}
			} else {
				error = 'Failed to create designation.';
			}
		} finally {
			isCreating = false;
		}
	}
</script>

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div
			class="relative flex max-h-[90vh] w-full max-w-2xl flex-col rounded-3xl border border-neutral-200 bg-white text-neutral-900 shadow-2xl transition-all"
		>
			<!-- Header -->
			<div class="flex items-center justify-between border-b border-neutral-200 p-6 pb-4">
				<div class="flex items-center gap-3">
					<div
						class="flex h-10 w-10 items-center justify-center rounded-2xl bg-neutral-100 text-neutral-900"
					>
						<Layers class="h-5 w-5" />
					</div>
					<div>
						<h3 class="font-serif text-lg font-medium tracking-tight text-neutral-950">
							Society Designation Catalogue
						</h3>
						<p class="text-xs text-neutral-500">
							Manage civic governance designations, occupancy rules, and availability
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={onClose}
					class="rounded-full p-2 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			<!-- Messages -->
			{#if error}
				<div
					class="mx-6 mt-4 flex items-start gap-2.5 rounded-xl border border-rose-200 bg-rose-50/80 p-3 text-xs text-rose-800"
				>
					<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
					<span>{error}</span>
				</div>
			{/if}

			{#if statusMessage}
				<div
					class="mx-6 mt-4 flex items-start gap-2.5 rounded-xl border border-emerald-200 bg-emerald-50/80 p-3 text-xs text-emerald-800"
				>
					<CheckCircle2 class="mt-0.5 h-4 w-4 shrink-0 text-emerald-600" />
					<span>{statusMessage}</span>
				</div>
			{/if}

			<!-- Scrollable Content -->
			<div class="flex-1 space-y-6 overflow-y-auto p-6">
				<!-- Top Action Bar -->
				<div class="flex items-center justify-between">
					<span class="text-xs font-medium text-neutral-700">
						Configured Designations ({designations.length})
					</span>
					<Button
						variant={showCreateForm ? 'ghost' : 'outline'}
						size="sm"
						onclick={() => (showCreateForm = !showCreateForm)}
					>
						{#if showCreateForm}
							<X class="mr-1.5 h-3.5 w-3.5" />
							<span>Cancel</span>
						{:else}
							<Plus class="mr-1.5 h-3.5 w-3.5" />
							<span>Add Custom Designation</span>
						{/if}
					</Button>
				</div>

				<!-- Inline Create Form -->
				{#if showCreateForm}
					<form
						onsubmit={handleCreateDesignation}
						class="space-y-3 rounded-2xl border border-neutral-200 bg-neutral-50/80 p-4"
					>
						<h4 class="font-serif text-xs font-semibold text-neutral-900">
							New Society Designation
						</h4>

						<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
							<div>
								<label for="new-code" class="block text-[11px] font-medium text-neutral-700">
									Designation Code <span class="text-rose-500">*</span>
								</label>
								<input
									id="new-code"
									type="text"
									placeholder="e.g. VICE_CHAIRMAN"
									bind:value={newCode}
									class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 font-mono text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
								/>
								{#if formErrors.code}
									<p class="mt-0.5 text-[10px] text-rose-600">{formErrors.code}</p>
								{/if}
							</div>

							<div>
								<label for="new-name" class="block text-[11px] font-medium text-neutral-700">
									Display Name <span class="text-rose-500">*</span>
								</label>
								<input
									id="new-name"
									type="text"
									placeholder="e.g. Vice Chairman"
									bind:value={newName}
									class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
								/>
								{#if formErrors.name}
									<p class="mt-0.5 text-[10px] text-rose-600">{formErrors.name}</p>
								{/if}
							</div>
						</div>

						<div>
							<label for="new-desc" class="block text-[11px] font-medium text-neutral-700">
								Description <span class="text-neutral-400">(Optional)</span>
							</label>
							<input
								id="new-desc"
								type="text"
								placeholder="e.g. Deputises for Chairman during committee sessions"
								bind:value={newDescription}
								class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
							/>
						</div>

						<div class="flex flex-col gap-3 pt-1 sm:flex-row sm:items-center sm:justify-between">
							<label class="flex cursor-pointer items-center gap-2 text-xs text-neutral-700">
								<input
									type="checkbox"
									bind:checked={allowsMultiple}
									class="h-4 w-4 rounded border-neutral-300 text-neutral-950 focus:ring-neutral-950"
								/>
								<span>Permit multiple active members simultaneously</span>
							</label>

							<Button type="submit" variant="primary" size="sm" disabled={isCreating}>
								{#if isCreating}
									<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
									<span>Saving...</span>
								{:else}
									<Plus class="mr-1.5 h-3.5 w-3.5" />
									<span>Save Designation</span>
								{/if}
							</Button>
						</div>
					</form>
				{/if}

				<!-- List of Designations -->
				{#if isLoading}
					<div class="flex min-h-[160px] flex-col items-center justify-center text-neutral-500">
						<Loader2 class="h-6 w-6 animate-spin text-neutral-900" />
						<p class="mt-2 text-xs">Loading catalogue...</p>
					</div>
				{:else if designations.length === 0}
					<p class="py-6 text-center text-xs text-neutral-400">No designations configured.</p>
				{:else}
					<div
						class="divide-y divide-neutral-200 rounded-2xl border border-neutral-200 bg-white shadow-xs"
					>
						{#each designations as d (d.id)}
							<div
								class="flex items-center justify-between p-4 transition-colors hover:bg-neutral-50/50"
							>
								<div class="space-y-1">
									<div class="flex items-center gap-2">
										<span class="text-xs font-semibold text-neutral-950">{d.name}</span>
										<Badge variant="outline" class="font-mono text-[10px] text-neutral-600">
											{d.code}
										</Badge>
										{#if d.is_system_default}
											<Badge
												variant="outline"
												class="border-neutral-300 text-[9px] text-neutral-600"
											>
												System Default
											</Badge>
										{/if}
										<Badge variant={d.is_active ? 'success' : 'default'} class="text-[10px]">
											{d.is_active ? 'Active' : 'Inactive'}
										</Badge>
									</div>

									<p class="text-[11px] text-neutral-600">
										{d.description || 'No description provided.'}
									</p>

									<div class="flex items-center gap-3 text-[10px] text-neutral-500">
										<span>
											Occupancy: <strong class="text-neutral-700"
												>{d.allows_multiple_members
													? 'Multiple active members'
													: 'Sole holder (Singleton)'}</strong
											>
										</span>
									</div>
								</div>

								<div class="flex items-center gap-2">
									<Button
										variant="ghost"
										size="sm"
										disabled={togglingId === d.id}
										onclick={() => handleToggleStatus(d)}
										class="text-xs {d.is_active
											? 'text-amber-700 hover:bg-amber-50 hover:text-amber-800'
											: 'text-emerald-700 hover:bg-emerald-50 hover:text-emerald-800'}"
									>
										{#if togglingId === d.id}
											<Loader2 class="h-3.5 w-3.5 animate-spin" />
										{:else if d.is_active}
											<ToggleRight class="mr-1 h-4 w-4" />
											<span>Deactivate</span>
										{:else}
											<ToggleLeft class="mr-1 h-4 w-4" />
											<span>Activate</span>
										{/if}
									</Button>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="flex justify-end border-t border-neutral-200 p-4">
				<Button variant="ghost" size="sm" onclick={onClose}>Close</Button>
			</div>
		</div>
	</div>
{/if}
