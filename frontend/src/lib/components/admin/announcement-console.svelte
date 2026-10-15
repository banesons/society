<script lang="ts">
	import {
		societyApi,
		type Announcement,
		type AnnouncementCategory,
		type AnnouncementPriority,
		type AnnouncementAudienceType,
		type CreateAnnouncementPayload,
		type UpdateAnnouncementPayload,
		type Building,
		type Wing,
		type Unit
	} from '$lib/api/society';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import {
		Megaphone,
		Plus,
		Search,
		Edit3,
		Send,
		Archive,
		AlertCircle,
		Loader2,
		RefreshCw,
		X
	} from '@lucide/svelte';

	interface Props {
		societyId: string;
	}

	let { societyId }: Props = $props();

	// Announcements data
	let announcements = $state<Announcement[]>([]);
	let totalCount = $state<number>(0);
	let isLoading = $state<boolean>(true);
	let error = $state<string | null>(null);

	// Filters
	let activeStatus = $state<string>('ALL');
	let selectedCategory = $state<string>('ALL');
	let searchQuery = $state<string>('');

	// Structural target options
	let buildings = $state<Building[]>([]);
	let availableWings = $state<Wing[]>([]);
	let units = $state<Unit[]>([]);

	// Composer Dialog State
	let isComposerOpen = $state<boolean>(false);
	let isEditing = $state<boolean>(false);
	let editingId = $state<string | null>(null);
	let isSubmitting = $state<boolean>(false);
	let composerError = $state<string | null>(null);

	// Form values
	let formTitle = $state<string>('');
	let formBody = $state<string>('');
	let formCategory = $state<AnnouncementCategory>('GENERAL');
	let formPriority = $state<AnnouncementPriority>('NORMAL');
	let formAudienceType = $state<AnnouncementAudienceType>('SOCIETY');
	let formTargetBuildingId = $state<string>('');
	let formTargetWingId = $state<string>('');
	let formTargetUnitId = $state<string>('');

	// Confirmation Dialogs
	let publishTarget = $state<Announcement | null>(null);
	let archiveTarget = $state<Announcement | null>(null);
	let isActionPending = $state<boolean>(false);

	const categories: { label: string; value: AnnouncementCategory }[] = [
		{ label: 'General Notice', value: 'GENERAL' },
		{ label: 'Maintenance Advisory', value: 'MAINTENANCE' },
		{ label: 'Security Alert', value: 'SECURITY' },
		{ label: 'Community Event', value: 'EVENT' },
		{ label: 'Emergency Notice', value: 'EMERGENCY' },
		{ label: 'Administrative Order', value: 'ADMINISTRATIVE' }
	];

	const priorities: { label: string; value: AnnouncementPriority }[] = [
		{ label: 'Normal Priority', value: 'NORMAL' },
		{ label: 'Urgent', value: 'URGENT' },
		{ label: 'Critical Alert', value: 'CRITICAL' }
	];

	async function loadStructure() {
		try {
			const [bldRes, untRes] = await Promise.all([
				societyApi.listBuildings(societyId).catch(() => ({ buildings: [] })),
				societyApi.listUnits(societyId, { limit: 100 }).catch(() => ({ units: [] }))
			]);
			buildings = bldRes.buildings || [];
			units = untRes.units || [];
		} catch (e) {
			console.error('Failed to load structural elements:', e);
		}
	}

	async function loadAnnouncements() {
		if (!societyId) return;
		isLoading = true;
		error = null;

		try {
			const res = await societyApi.getAdminAnnouncements(societyId, {
				status: activeStatus === 'ALL' ? undefined : activeStatus,
				category: selectedCategory === 'ALL' ? undefined : selectedCategory,
				search: searchQuery.trim() || undefined,
				limit: 50
			});
			announcements = res.announcements;
			totalCount = res.total;
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load announcements';
		} finally {
			isLoading = false;
		}
	}

	function resetForm() {
		formTitle = '';
		formBody = '';
		formCategory = 'GENERAL';
		formPriority = 'NORMAL';
		formAudienceType = 'SOCIETY';
		formTargetBuildingId = '';
		formTargetWingId = '';
		formTargetUnitId = '';
		editingId = null;
		isEditing = false;
		composerError = null;
	}

	function openCreateModal() {
		resetForm();
		isComposerOpen = true;
	}

	function openEditModal(ann: Announcement) {
		resetForm();
		isEditing = true;
		editingId = ann.id;
		formTitle = ann.title;
		formBody = ann.body;
		formCategory = ann.category;
		formPriority = ann.priority;
		formAudienceType = ann.audience_type;
		formTargetBuildingId = ann.target_building_id || '';
		formTargetWingId = ann.target_wing_id || '';
		formTargetUnitId = ann.target_unit_id || '';
		isComposerOpen = true;
	}

	async function saveAnnouncement() {
		if (!formTitle.trim()) {
			composerError = 'Title is required';
			return;
		}
		if (!formBody.trim()) {
			composerError = 'Body content is required';
			return;
		}

		isSubmitting = true;
		composerError = null;

		try {
			if (isEditing && editingId) {
				const payload: UpdateAnnouncementPayload = {
					title: formTitle.trim(),
					body: formBody.trim(),
					category: formCategory,
					priority: formPriority
				};
				// For drafts, targeting can also be updated
				const current = announcements.find((a) => a.id === editingId);
				if (current && current.status === 'DRAFT') {
					payload.audience_type = formAudienceType;
					payload.target_building_id =
						formAudienceType === 'BUILDING' ? formTargetBuildingId || undefined : undefined;
					payload.target_wing_id =
						formAudienceType === 'WING' ? formTargetWingId || undefined : undefined;
					payload.target_unit_id =
						formAudienceType === 'UNIT' ? formTargetUnitId || undefined : undefined;
				}
				await societyApi.updateAnnouncement(societyId, editingId, payload);
			} else {
				const payload: CreateAnnouncementPayload = {
					title: formTitle.trim(),
					body: formBody.trim(),
					category: formCategory,
					priority: formPriority,
					audience_type: formAudienceType,
					target_building_id:
						formAudienceType === 'BUILDING' ? formTargetBuildingId || undefined : undefined,
					target_wing_id: formAudienceType === 'WING' ? formTargetWingId || undefined : undefined,
					target_unit_id: formAudienceType === 'UNIT' ? formTargetUnitId || undefined : undefined
				};
				await societyApi.createAnnouncement(societyId, payload);
			}
			isComposerOpen = false;
			await loadAnnouncements();
		} catch (err: unknown) {
			composerError = err instanceof Error ? err.message : 'Operation failed';
		} finally {
			isSubmitting = false;
		}
	}

	async function confirmPublish() {
		if (!publishTarget) return;
		isActionPending = true;
		try {
			await societyApi.publishAnnouncement(societyId, publishTarget.id);
			publishTarget = null;
			await loadAnnouncements();
		} catch (err) {
			console.error('Publish error:', err);
		} finally {
			isActionPending = false;
		}
	}

	async function confirmArchive() {
		if (!archiveTarget) return;
		isActionPending = true;
		try {
			await societyApi.archiveAnnouncement(societyId, archiveTarget.id);
			archiveTarget = null;
			await loadAnnouncements();
		} catch (err) {
			console.error('Archive error:', err);
		} finally {
			isActionPending = false;
		}
	}

	function formatDate(dateStr?: string | null): string {
		if (!dateStr) return '—';
		const d = new Date(dateStr);
		return d.toLocaleDateString('en-IN', {
			day: 'numeric',
			month: 'short',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function getStatusBadge(status: string) {
		switch (status) {
			case 'PUBLISHED':
				return { variant: 'default' as const, label: 'Published' };
			case 'ARCHIVED':
				return { variant: 'outline' as const, label: 'Archived' };
			default:
				return { variant: 'warning' as const, label: 'Draft' };
		}
	}

	function getPriorityBadge(priority: string) {
		switch (priority) {
			case 'CRITICAL':
				return { variant: 'destructive' as const, label: 'Critical' };
			case 'URGENT':
				return { variant: 'warning' as const, label: 'Urgent' };
			default:
				return { variant: 'outline' as const, label: 'Normal' };
		}
	}

	$effect(() => {
		if (societyId) {
			loadAnnouncements();
			loadStructure();
		}
	});

	// Reactively compute available wings when building is chosen
	$effect(() => {
		if (formTargetBuildingId) {
			const bld = buildings.find((b) => b.id === formTargetBuildingId);
			availableWings = bld?.wings || [];
		} else {
			availableWings = [];
		}
	});
</script>

<div class="rounded-2xl border border-neutral-200 bg-white/90 p-6 shadow-xs">
	<!-- Header -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2">
				<div
					class="flex h-8 w-8 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
				>
					<Megaphone class="h-4 w-4" />
				</div>
				<h2 class="text-base font-semibold tracking-tight text-neutral-950">
					Announcement Management
				</h2>
				<Badge variant="outline" class="font-mono text-[10px] text-neutral-700">
					{totalCount} Total
				</Badge>
			</div>
			<p class="mt-1 text-xs text-neutral-500">
				Compose notices, manage publication lifecycle, target specific audiences, and inspect
				delivery state.
			</p>
		</div>

		<div class="flex items-center gap-2">
			<Button variant="primary" size="sm" onclick={openCreateModal}>
				<Plus class="mr-1.5 h-3.5 w-3.5" />
				<span>Compose Announcement</span>
			</Button>

			<Button variant="outline" size="sm" onclick={loadAnnouncements} title="Refresh">
				<RefreshCw class="h-3.5 w-3.5 {isLoading ? 'animate-spin' : ''}" />
			</Button>
		</div>
	</div>

	<!-- Status Tabs & Filter Controls -->
	<div class="mt-5 space-y-3">
		<div class="flex flex-wrap items-center justify-between gap-3 border-b border-neutral-100 pb-3">
			<!-- Status Tabs -->
			<div class="flex items-center gap-1">
				{#each ['ALL', 'DRAFT', 'PUBLISHED', 'ARCHIVED'] as st (st)}
					<button
						type="button"
						onclick={() => {
							activeStatus = st;
							loadAnnouncements();
						}}
						class="rounded-lg px-3 py-1.5 text-xs font-medium transition-colors {activeStatus === st
							? 'bg-neutral-900 text-white'
							: 'text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900'}"
					>
						{st === 'ALL' ? 'All Notices' : st.charAt(0) + st.slice(1).toLowerCase()}
					</button>
				{/each}
			</div>

			<!-- Category dropdown -->
			<div class="flex items-center gap-2">
				<select
					bind:value={selectedCategory}
					onchange={() => loadAnnouncements()}
					class="h-8 rounded-lg border border-neutral-200 bg-white px-2.5 text-xs text-neutral-800 focus:border-neutral-900 focus:outline-hidden"
				>
					<option value="ALL">All Categories</option>
					{#each categories as cat (cat.value)}
						<option value={cat.value}>{cat.label}</option>
					{/each}
				</select>
			</div>
		</div>

		<!-- Search Bar -->
		<div class="relative">
			<Search class="absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-neutral-400" />
			<input
				type="text"
				placeholder="Search by announcement title or keywords..."
				bind:value={searchQuery}
				onkeydown={(e) => e.key === 'Enter' && loadAnnouncements()}
				class="h-9 w-full rounded-xl border border-neutral-200 bg-neutral-50/50 pr-24 pl-9 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-900 focus:bg-white focus:outline-hidden"
			/>
			{#if searchQuery}
				<button
					type="button"
					onclick={() => {
						searchQuery = '';
						loadAnnouncements();
					}}
					class="absolute top-1/2 right-2 -translate-y-1/2 rounded-md px-2 py-0.5 text-[11px] font-medium text-neutral-500 hover:text-neutral-900"
				>
					Clear
				</button>
			{/if}
		</div>
	</div>

	<!-- Announcements Table / List -->
	<div class="mt-4">
		{#if isLoading}
			<div class="flex flex-col items-center justify-center py-12 text-center">
				<Loader2 class="h-6 w-6 animate-spin text-neutral-400" />
				<p class="mt-2 text-xs text-neutral-500">Loading notices...</p>
			</div>
		{:else if error}
			<div class="rounded-xl border border-rose-200 bg-rose-50/50 p-4 text-center">
				<AlertCircle class="mx-auto h-5 w-5 text-rose-600" />
				<p class="mt-1 text-xs font-medium text-rose-900">{error}</p>
				<Button variant="outline" size="sm" class="mt-3 text-xs" onclick={loadAnnouncements}>
					Try Again
				</Button>
			</div>
		{:else if announcements.length === 0}
			<div
				class="flex flex-col items-center justify-center rounded-xl border border-dashed border-neutral-200 py-12 text-center"
			>
				<Megaphone class="h-8 w-8 text-neutral-300" />
				<p class="mt-2 text-xs font-medium text-neutral-800">No announcements found</p>
				<p class="mt-0.5 text-[11px] text-neutral-400">
					Create your first announcement draft using the button above.
				</p>
			</div>
		{:else}
			<div class="overflow-x-auto rounded-xl border border-neutral-200">
				<table class="w-full text-left text-xs">
					<thead class="border-b border-neutral-200 bg-neutral-50 text-neutral-600">
						<tr>
							<th class="px-4 py-3 font-semibold">Title & Details</th>
							<th class="px-4 py-3 font-semibold">Audience</th>
							<th class="px-4 py-3 font-semibold">Status</th>
							<th class="px-4 py-3 font-semibold">Timeline</th>
							<th class="px-4 py-3 text-right font-semibold">Actions</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-100 bg-white">
						{#each announcements as item (item.id)}
							{@const status = getStatusBadge(item.status)}
							{@const priority = getPriorityBadge(item.priority)}
							<tr class="hover:bg-neutral-50/60">
								<!-- Title & Category -->
								<td class="px-4 py-3.5">
									<div class="font-medium text-neutral-900">{item.title}</div>
									<div class="mt-1 flex items-center gap-1.5">
										<Badge variant={priority.variant} class="font-mono text-[9px] uppercase">
											{priority.label}
										</Badge>
										<span class="text-[11px] text-neutral-500">{item.category}</span>
									</div>
								</td>

								<!-- Audience -->
								<td class="px-4 py-3.5 text-neutral-700">
									<div class="font-mono text-[11px] font-medium text-neutral-900">
										{item.audience_type}
									</div>
									{#if item.target_label}
										<div class="text-[11px] text-neutral-500">{item.target_label}</div>
									{:else}
										<div class="text-[11px] text-neutral-400">Society-wide</div>
									{/if}
								</td>

								<!-- Status -->
								<td class="px-4 py-3.5">
									<Badge variant={status.variant} class="font-mono text-[10px] uppercase">
										{status.label}
									</Badge>
								</td>

								<!-- Timeline -->
								<td class="px-4 py-3.5 text-[11px] text-neutral-500">
									{#if item.status === 'PUBLISHED' && item.published_at}
										<div>Pub: {formatDate(item.published_at)}</div>
									{:else if item.status === 'ARCHIVED' && item.archived_at}
										<div>Arch: {formatDate(item.archived_at)}</div>
									{:else}
										<div>Created: {formatDate(item.created_at)}</div>
									{/if}
								</td>

								<!-- Actions -->
								<td class="px-4 py-3.5 text-right">
									<div class="flex items-center justify-end gap-1.5">
										{#if item.status !== 'ARCHIVED'}
											<Button
												variant="outline"
												size="sm"
												onclick={() => openEditModal(item)}
												class="h-7 px-2 text-[11px]"
												title="Edit content"
											>
												<Edit3 class="mr-1 h-3 w-3" />
												<span>Edit</span>
											</Button>
										{/if}

										{#if item.status === 'DRAFT'}
											<Button
												variant="primary"
												size="sm"
												onclick={() => (publishTarget = item)}
												class="h-7 px-2 text-[11px]"
												title="Publish to Residents"
											>
												<Send class="mr-1 h-3 w-3" />
												<span>Publish</span>
											</Button>
										{/if}

										{#if item.status === 'PUBLISHED'}
											<Button
												variant="outline"
												size="sm"
												onclick={() => (archiveTarget = item)}
												class="h-7 px-2 text-[11px] text-neutral-600 hover:text-neutral-950"
												title="Archive Notice"
											>
												<Archive class="mr-1 h-3 w-3" />
												<span>Archive</span>
											</Button>
										{/if}
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</div>

<!-- Composer Modal -->
{#if isComposerOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div
			class="max-h-[90vh] w-full max-w-xl space-y-4 overflow-y-auto rounded-2xl border border-neutral-200 bg-white p-6 text-xs shadow-xl"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-3">
				<h3 class="text-base font-semibold text-neutral-950">
					{isEditing ? 'Edit Announcement' : 'Compose New Announcement'}
				</h3>
				<button
					type="button"
					onclick={() => (isComposerOpen = false)}
					class="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-900"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if composerError}
				<div class="rounded-lg border border-rose-200 bg-rose-50 p-2.5 text-rose-800">
					{composerError}
				</div>
			{/if}

			<!-- Title -->
			<div>
				<label class="block font-medium text-neutral-800" for="ann-title"
					>Announcement Title *</label
				>
				<input
					id="ann-title"
					type="text"
					placeholder="e.g., Annual Society General Meeting Notice"
					bind:value={formTitle}
					maxlength={200}
					class="mt-1 h-9 w-full rounded-xl border border-neutral-200 bg-white px-3 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
				/>
				<span class="mt-0.5 block text-right text-[10px] text-neutral-400"
					>{formTitle.length}/200</span
				>
			</div>

			<!-- Category & Priority Row -->
			<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
				<div>
					<label class="block font-medium text-neutral-800" for="ann-category">Category *</label>
					<select
						id="ann-category"
						bind:value={formCategory}
						class="mt-1 h-9 w-full rounded-xl border border-neutral-200 bg-white px-3 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
					>
						{#each categories as cat (cat.value)}
							<option value={cat.value}>{cat.label}</option>
						{/each}
					</select>
				</div>

				<div>
					<label class="block font-medium text-neutral-800" for="ann-priority">Priority *</label>
					<select
						id="ann-priority"
						bind:value={formPriority}
						class="mt-1 h-9 w-full rounded-xl border border-neutral-200 bg-white px-3 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
					>
						{#each priorities as pri (pri.value)}
							<option value={pri.value}>{pri.label}</option>
						{/each}
					</select>
				</div>
			</div>

			<!-- Audience Selection -->
			<div class="space-y-3 rounded-xl border border-neutral-200 bg-neutral-50/60 p-3.5">
				<label class="block font-semibold text-neutral-900" for="ann-audience"
					>Audience Targeting</label
				>
				<div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
					{#each ['SOCIETY', 'BUILDING', 'WING', 'UNIT'] as aud (aud)}
						<button
							type="button"
							disabled={isEditing &&
								announcements.find((a) => a.id === editingId)?.status === 'PUBLISHED'}
							onclick={() => {
								formAudienceType = aud as AnnouncementAudienceType;
							}}
							class="rounded-lg border px-2.5 py-1.5 text-center text-xs font-medium transition-all {formAudienceType ===
							aud
								? 'border-neutral-950 bg-neutral-950 text-white'
								: 'border-neutral-200 bg-white text-neutral-700 hover:border-neutral-400'} {isEditing &&
							announcements.find((a) => a.id === editingId)?.status === 'PUBLISHED'
								? 'cursor-not-allowed opacity-60'
								: ''}"
						>
							{aud}
						</button>
					{/each}
				</div>

				{#if isEditing && announcements.find((a) => a.id === editingId)?.status === 'PUBLISHED'}
					<p class="text-[11px] text-amber-700">
						Audience targeting is immutable once an announcement is published.
					</p>
				{/if}

				<!-- Target Selector based on Audience Type -->
				{#if formAudienceType === 'BUILDING'}
					<div>
						<label class="block text-[11px] font-medium text-neutral-700" for="target-building"
							>Select Target Building *</label
						>
						<select
							id="target-building"
							bind:value={formTargetBuildingId}
							class="mt-1 h-8 w-full rounded-lg border border-neutral-200 bg-white px-2.5 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
						>
							<option value="">— Select Building —</option>
							{#each buildings as bld (bld.id)}
								<option value={bld.id}>{bld.name} ({bld.building_code})</option>
							{/each}
						</select>
					</div>
				{:else if formAudienceType === 'WING'}
					<div class="space-y-2">
						<div>
							<label class="block text-[11px] font-medium text-neutral-700" for="wing-building"
								>Select Building *</label
							>
							<select
								id="wing-building"
								bind:value={formTargetBuildingId}
								class="mt-1 h-8 w-full rounded-lg border border-neutral-200 bg-white px-2.5 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
							>
								<option value="">— Select Building —</option>
								{#each buildings as bld (bld.id)}
									<option value={bld.id}>{bld.name}</option>
								{/each}
							</select>
						</div>

						{#if formTargetBuildingId}
							<div>
								<label class="block text-[11px] font-medium text-neutral-700" for="target-wing"
									>Select Wing *</label
								>
								<select
									id="target-wing"
									bind:value={formTargetWingId}
									class="mt-1 h-8 w-full rounded-lg border border-neutral-200 bg-white px-2.5 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
								>
									<option value="">— Select Wing —</option>
									{#each availableWings as w (w.id)}
										<option value={w.id}>{w.wing_code} ({w.name})</option>
									{/each}
								</select>
							</div>
						{/if}
					</div>
				{:else if formAudienceType === 'UNIT'}
					<div>
						<label class="block text-[11px] font-medium text-neutral-700" for="target-unit"
							>Select Target Unit *</label
						>
						<select
							id="target-unit"
							bind:value={formTargetUnitId}
							class="mt-1 h-8 w-full rounded-lg border border-neutral-200 bg-white px-2.5 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
						>
							<option value="">— Select Unit / Flat —</option>
							{#each units as u (u.id)}
								<option value={u.id}>{u.unit_identifier} (Floor {u.floor_number})</option>
							{/each}
						</select>
					</div>
				{/if}
			</div>

			<!-- Body -->
			<div>
				<label class="block font-medium text-neutral-800" for="ann-body">Announcement Body *</label>
				<textarea
					id="ann-body"
					rows={5}
					placeholder="Write the full announcement details, schedules, instructions, or agenda..."
					bind:value={formBody}
					class="mt-1 w-full rounded-xl border border-neutral-200 bg-white p-3 text-xs leading-relaxed text-neutral-900 focus:border-neutral-900 focus:outline-hidden"
				></textarea>
			</div>

			<!-- Form Actions -->
			<div class="flex items-center justify-end gap-2 border-t border-neutral-100 pt-3">
				<Button variant="outline" size="sm" onclick={() => (isComposerOpen = false)}>Cancel</Button>
				<Button variant="primary" size="sm" onclick={saveAnnouncement} disabled={isSubmitting}>
					{#if isSubmitting}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					{/if}
					<span>{isEditing ? 'Save Changes' : 'Save as Draft'}</span>
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Publish Confirmation Modal -->
{#if publishTarget}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div
			class="w-full max-w-md space-y-4 rounded-2xl border border-neutral-200 bg-white p-6 text-xs shadow-xl"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-2">
				<h3 class="text-sm font-semibold text-neutral-950">Publish Announcement</h3>
				<button
					type="button"
					onclick={() => (publishTarget = null)}
					class="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<p class="leading-relaxed text-neutral-700">
				Are you sure you want to publish <strong class="font-semibold text-neutral-950"
					>{publishTarget.title}</strong
				>?
			</p>
			<p class="text-[11px] text-neutral-500">
				Once published, this announcement will become immediately visible on the resident feed for
				all members in the selected audience ({publishTarget.audience_type}).
			</p>

			<div class="flex items-center justify-end gap-2 border-t border-neutral-100 pt-3">
				<Button
					id="cancel-publish-btn"
					variant="outline"
					size="sm"
					onclick={() => (publishTarget = null)}>Cancel</Button
				>
				<Button
					id="confirm-publish-btn"
					variant="primary"
					size="sm"
					onclick={confirmPublish}
					disabled={isActionPending}
				>
					{#if isActionPending}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					{/if}
					<span>Confirm Publication</span>
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Archive Confirmation Modal -->
{#if archiveTarget}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div
			class="w-full max-w-md space-y-4 rounded-2xl border border-neutral-200 bg-white p-6 text-xs shadow-xl"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-2">
				<h3 class="text-sm font-semibold text-neutral-950">Archive Announcement</h3>
				<button
					type="button"
					onclick={() => (archiveTarget = null)}
					class="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<p class="leading-relaxed text-neutral-700">
				Are you sure you want to archive <strong class="font-semibold text-neutral-950"
					>{archiveTarget.title}</strong
				>?
			</p>
			<p class="text-[11px] text-neutral-500">
				Archiving removes the notice from active resident feeds. The notice record is preserved
				permanently and will remain accessible to society administrators for audit and historical
				review.
			</p>

			<div class="flex items-center justify-end gap-2 border-t border-neutral-100 pt-3">
				<Button
					id="cancel-archive-btn"
					variant="outline"
					size="sm"
					onclick={() => (archiveTarget = null)}>Cancel</Button
				>
				<Button
					id="confirm-archive-btn"
					variant="destructive"
					size="sm"
					onclick={confirmArchive}
					disabled={isActionPending}
				>
					{#if isActionPending}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					{/if}
					<span>Archive Notice</span>
				</Button>
			</div>
		</div>
	</div>
{/if}
