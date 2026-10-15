<script lang="ts">
	import { societyApi, type Announcement } from '$lib/api/society';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import {
		Bell,
		Search,
		CheckCircle2,
		AlertCircle,
		Clock,
		Eye,
		Loader2,
		Megaphone,
		Inbox,
		RefreshCw,
		X
	} from '@lucide/svelte';

	interface Props {
		societyId: string;
	}

	let { societyId }: Props = $props();

	let announcements = $state<Announcement[]>([]);
	let unreadCount = $state<number>(0);
	let isLoading = $state<boolean>(true);
	let error = $state<string | null>(null);

	// Filters
	let selectedCategory = $state<string>('ALL');
	let unreadOnly = $state<boolean>(false);
	let searchQuery = $state<string>('');

	// Active viewing dialog
	let activeNotice = $state<Announcement | null>(null);
	let isDetailOpen = $state<boolean>(false);

	const categories: { label: string; value: string }[] = [
		{ label: 'All Notices', value: 'ALL' },
		{ label: 'General', value: 'GENERAL' },
		{ label: 'Maintenance', value: 'MAINTENANCE' },
		{ label: 'Security', value: 'SECURITY' },
		{ label: 'Event', value: 'EVENT' },
		{ label: 'Emergency', value: 'EMERGENCY' },
		{ label: 'Administrative', value: 'ADMINISTRATIVE' }
	];

	async function loadFeed() {
		if (!societyId) return;
		isLoading = true;
		error = null;

		try {
			const res = await societyApi.getAnnouncementFeed(societyId, {
				category: selectedCategory === 'ALL' ? undefined : selectedCategory,
				unread_only: unreadOnly ? true : undefined,
				search: searchQuery.trim() || undefined,
				limit: 50
			});
			announcements = res.announcements;
			unreadCount = res.unread_count;
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load announcement feed';
		} finally {
			isLoading = false;
		}
	}

	async function openDetail(notice: Announcement) {
		activeNotice = notice;
		isDetailOpen = true;

		// If unread, mark read immediately
		if (!notice.is_read) {
			try {
				await societyApi.markAnnouncementRead(societyId, notice.id);
				// Locally update notice state
				notice.is_read = true;
				notice.read_at = new Date().toISOString();
				if (unreadCount > 0) {
					unreadCount -= 1;
				}
			} catch (err) {
				console.error('Failed to mark announcement read:', err);
			}
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

	function getAudienceBadge(notice: Announcement) {
		if (notice.audience_type === 'SOCIETY') {
			return 'Society-wide';
		}
		if (notice.target_label) {
			return `${notice.audience_type}: ${notice.target_label}`;
		}
		return notice.audience_type;
	}

	$effect(() => {
		// Reload when societyId, category, or unreadOnly changes
		if (societyId) {
			loadFeed();
		}
	});

	function handleSearchKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			loadFeed();
		}
	}
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
					Society Announcements
				</h2>
				{#if unreadCount > 0}
					<Badge variant="destructive" class="px-2 py-0.5 text-[10px] font-bold">
						{unreadCount} Unread
					</Badge>
				{:else}
					<Badge
						variant="outline"
						class="border-emerald-200 bg-emerald-50 text-[10px] text-emerald-800"
					>
						Up to date
					</Badge>
				{/if}
			</div>
			<p class="mt-1 text-xs text-neutral-500">
				Official notices, maintenance advisories, and administrative updates for your society.
			</p>
		</div>

		<div class="flex items-center gap-2">
			<Button
				variant="outline"
				size="sm"
				onclick={() => {
					unreadOnly = !unreadOnly;
				}}
				class={unreadOnly
					? 'border-neutral-950 bg-neutral-950 text-white hover:bg-neutral-800'
					: ''}
			>
				<Bell class="mr-1.5 h-3.5 w-3.5" />
				<span>{unreadOnly ? 'Showing Unread' : 'Show Unread Only'}</span>
			</Button>

			<Button variant="outline" size="sm" onclick={loadFeed} title="Refresh Feed">
				<RefreshCw class="h-3.5 w-3.5 {isLoading ? 'animate-spin' : ''}" />
			</Button>
		</div>
	</div>

	<!-- Controls & Category Filters -->
	<div class="mt-5 space-y-3">
		<!-- Search input -->
		<div class="relative">
			<Search class="absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-neutral-400" />
			<input
				type="text"
				placeholder="Search announcements by title or content..."
				bind:value={searchQuery}
				onkeydown={handleSearchKeydown}
				class="h-9 w-full rounded-xl border border-neutral-200 bg-neutral-50/50 pr-24 pl-9 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-900 focus:bg-white focus:outline-hidden"
			/>
			{#if searchQuery}
				<button
					type="button"
					onclick={() => {
						searchQuery = '';
						loadFeed();
					}}
					class="absolute top-1/2 right-2 -translate-y-1/2 rounded-md px-2 py-0.5 text-[11px] font-medium text-neutral-500 hover:text-neutral-900"
				>
					Clear
				</button>
			{/if}
		</div>

		<!-- Category Chips -->
		<div class="flex flex-wrap items-center gap-1.5 overflow-x-auto pb-1 text-xs">
			{#each categories as cat (cat.value)}
				<button
					type="button"
					onclick={() => {
						selectedCategory = cat.value;
					}}
					class="rounded-lg px-2.5 py-1 text-xs font-medium transition-colors {selectedCategory ===
					cat.value
						? 'bg-neutral-900 text-white'
						: 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200 hover:text-neutral-900'}"
				>
					{cat.label}
				</button>
			{/each}
		</div>
	</div>

	<!-- Feed Content -->
	<div class="mt-5">
		{#if isLoading}
			<div class="flex flex-col items-center justify-center py-12 text-center">
				<Loader2 class="h-6 w-6 animate-spin text-neutral-400" />
				<p class="mt-2 text-xs text-neutral-500">Loading notices...</p>
			</div>
		{:else if error}
			<div class="rounded-xl border border-rose-200 bg-rose-50/50 p-4 text-center">
				<AlertCircle class="mx-auto h-5 w-5 text-rose-600" />
				<p class="mt-1 text-xs font-medium text-rose-900">{error}</p>
				<Button variant="outline" size="sm" class="mt-3 text-xs" onclick={loadFeed}>
					Try Again
				</Button>
			</div>
		{:else if announcements.length === 0}
			<div
				class="flex flex-col items-center justify-center rounded-xl border border-dashed border-neutral-200 py-12 text-center"
			>
				<Inbox class="h-8 w-8 text-neutral-300" />
				<p class="mt-2 text-xs font-medium text-neutral-800">No announcements found</p>
				<p class="mt-0.5 text-[11px] text-neutral-400">
					{unreadOnly
						? 'All notices in this category have been marked read.'
						: 'There are no published announcements matching your filter.'}
				</p>
			</div>
		{:else}
			<div class="space-y-3">
				{#each announcements as item (item.id)}
					{@const priority = getPriorityBadge(item.priority)}
					{@const audience = getAudienceBadge(item)}
					<div
						class="group relative flex flex-col justify-between gap-3 rounded-xl border p-4 transition-all hover:border-neutral-300 hover:shadow-xs sm:flex-row sm:items-start {item.is_read
							? 'border-neutral-200 bg-white'
							: 'border-amber-200/80 bg-amber-50/30'}"
					>
						<!-- Left notice body -->
						<div class="flex-1 space-y-1.5">
							<div class="flex flex-wrap items-center gap-2">
								{#if !item.is_read}
									<span class="inline-flex h-2 w-2 rounded-full bg-amber-500" title="Unread Notice"
									></span>
								{/if}
								<h3 class="text-sm font-semibold text-neutral-900 group-hover:text-neutral-950">
									{item.title}
								</h3>
								<Badge variant={priority.variant} class="text-[10px] tracking-wider uppercase">
									{priority.label}
								</Badge>
								<Badge variant="outline" class="text-[10px] text-neutral-600">
									{item.category}
								</Badge>
								<Badge
									variant="outline"
									class="border-neutral-200 bg-neutral-100 text-[10px] text-neutral-700"
								>
									{audience}
								</Badge>
							</div>

							<p class="line-clamp-2 text-xs leading-relaxed text-neutral-600">
								{item.body}
							</p>

							<div class="flex items-center gap-3 pt-1 text-[11px] text-neutral-400">
								<span class="flex items-center gap-1">
									<Clock class="h-3 w-3" />
									Published {formatDate(item.published_at)}
								</span>
								{#if item.is_read}
									<span class="flex items-center gap-1 text-emerald-600">
										<CheckCircle2 class="h-3 w-3" />
										Read
									</span>
								{/if}
							</div>
						</div>

						<!-- Action -->
						<div class="flex sm:flex-col sm:items-end">
							<Button variant="outline" size="sm" onclick={() => openDetail(item)} class="text-xs">
								<Eye class="mr-1 h-3.5 w-3.5" />
								<span>View Details</span>
							</Button>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

<!-- Notice Detail Modal -->
{#if isDetailOpen && activeNotice}
	{@const priority = getPriorityBadge(activeNotice.priority)}
	{@const audience = getAudienceBadge(activeNotice)}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div
			class="w-full max-w-lg space-y-4 rounded-2xl border border-neutral-200 bg-white p-6 text-xs shadow-xl"
		>
			<div class="flex items-start justify-between gap-4">
				<h3 class="text-base font-semibold text-neutral-950">
					{activeNotice.title}
				</h3>
				<button
					type="button"
					onclick={() => (isDetailOpen = false)}
					class="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-900"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<!-- Meta Pills -->
			<div class="flex flex-wrap items-center gap-2 border-b border-neutral-100 pb-3">
				<Badge variant={priority.variant} class="text-[10px] font-bold">
					{priority.label}
				</Badge>
				<Badge variant="outline" class="text-[10px]">
					{activeNotice.category}
				</Badge>
				<Badge
					variant="outline"
					class="border-neutral-200 bg-neutral-50 text-[10px] text-neutral-700"
				>
					{audience}
				</Badge>
				<span class="text-[11px] text-neutral-400">
					Published {formatDate(activeNotice.published_at)}
				</span>
			</div>

			<!-- Notice Body Text -->
			<div
				class="max-h-80 overflow-y-auto rounded-xl border border-neutral-100 bg-neutral-50/50 p-4 font-sans text-xs leading-relaxed whitespace-pre-wrap text-neutral-800"
			>
				{activeNotice.body}
			</div>

			<!-- Read State Status -->
			<div
				class="flex items-center justify-between border-t border-neutral-100 pt-3 text-[11px] text-neutral-500"
			>
				<span class="flex items-center gap-1.5 font-medium text-emerald-600">
					<CheckCircle2 class="h-3.5 w-3.5" />
					Marked as read for your account
				</span>
				<Button variant="outline" size="sm" onclick={() => (isDetailOpen = false)}>Close</Button>
			</div>
		</div>
	</div>
{/if}
