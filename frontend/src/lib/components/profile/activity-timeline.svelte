<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getUserActivityTimeline,
		type UserActivityEvent,
		type UserActivityTimelineResponse
	} from '$lib/api/profile';
	import {
		Clock,
		Shield,
		User,
		LogIn,
		LogOut,
		Building2,
		Key,
		CheckCircle2,
		XCircle,
		QrCode,
		Loader2,
		FileText
	} from '@lucide/svelte';

	type Category = 'ALL' | 'ACCOUNT' | 'VISITOR' | 'SOCIETY' | 'SECURITY';

	let activeCategory = $state<Category>('ALL');
	let events = $state<UserActivityEvent[]>([]);
	let total = $state(0);
	let offset = $state(0);
	const limit = 20;
	let isLoading = $state(true);
	let isLoadingMore = $state(false);
	let error = $state<string | null>(null);

	const categories: { id: Category; label: string }[] = [
		{ id: 'ALL', label: 'All Activity' },
		{ id: 'ACCOUNT', label: 'Account & Profile' },
		{ id: 'VISITOR', label: 'Visitor Movement' },
		{ id: 'SOCIETY', label: 'Society & Units' },
		{ id: 'SECURITY', label: 'Security & Sessions' }
	];

	async function fetchTimeline(reset = false) {
		if (reset) {
			offset = 0;
			isLoading = true;
			error = null;
		} else {
			isLoadingMore = true;
		}

		try {
			const res: UserActivityTimelineResponse = await getUserActivityTimeline(
				activeCategory,
				limit,
				offset
			);
			if (reset) {
				events = res.events ?? [];
			} else {
				events = [...events, ...(res.events ?? [])];
			}
			total = res.total ?? 0;
		} catch (err: unknown) {
			if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Failed to load activity timeline';
			}
		} finally {
			isLoading = false;
			isLoadingMore = false;
		}
	}

	function handleCategoryChange(cat: Category) {
		if (activeCategory === cat) return;
		activeCategory = cat;
		fetchTimeline(true);
	}

	function loadMore() {
		if (isLoadingMore || offset + limit >= total) return;
		offset += limit;
		fetchTimeline(false);
	}

	function formatTimestamp(iso: string): { time: string; date: string } {
		const d = new Date(iso);
		const now = new Date();
		const isToday =
			d.getDate() === now.getDate() &&
			d.getMonth() === now.getMonth() &&
			d.getFullYear() === now.getFullYear();

		const yesterday = new Date(now.getTime() - 86400000);
		const isYesterday =
			d.getDate() === yesterday.getDate() &&
			d.getMonth() === yesterday.getMonth() &&
			d.getFullYear() === yesterday.getFullYear();

		const time = d.toLocaleTimeString('en-IN', {
			hour: '2-digit',
			minute: '2-digit',
			hour12: true
		});

		let date = d.toLocaleDateString('en-IN', {
			day: 'numeric',
			month: 'short',
			year: d.getFullYear() !== now.getFullYear() ? 'numeric' : undefined
		});

		if (isToday) date = 'Today';
		else if (isYesterday) date = 'Yesterday';

		return { time, date };
	}

	function groupEventsByDate(
		items: UserActivityEvent[]
	): Array<{ date: string; items: UserActivityEvent[] }> {
		const groups: Record<string, UserActivityEvent[]> = {};
		for (const ev of items) {
			const { date } = formatTimestamp(ev.timestamp);
			if (!groups[date]) {
				groups[date] = [];
			}
			groups[date].push(ev);
		}

		return Object.keys(groups).map((date) => ({
			date,
			items: groups[date]
		}));
	}

	let groupedEvents = $derived(groupEventsByDate(events));

	function getActionIcon(action: string, category: string) {
		switch (action) {
			case 'VISITOR_CHECKED_IN':
				return LogIn;
			case 'VISITOR_CHECKED_OUT':
				return LogOut;
			case 'VISIT_APPROVED':
				return CheckCircle2;
			case 'VISIT_REJECTED':
			case 'VISIT_CANCELLED':
				return XCircle;
			case 'VISITOR_PASS_ISSUED':
				return QrCode;
			case 'PROFILE_UPDATED':
			case 'PROFILE_PHOTO_UPLOADED':
			case 'USER_REGISTERED':
				return User;
			case 'VISITOR_IDENTITY_LINKED':
				return Key;
			case 'USER_LOGIN':
			case 'USER_LOGOUT':
				return Shield;
			case 'UNIT_RELATIONSHIP_CREATED':
			case 'MEMBER_REGISTERED':
				return Building2;
			default:
				if (category === 'VISITOR') return LogIn;
				if (category === 'SECURITY') return Shield;
				if (category === 'SOCIETY') return Building2;
				return FileText;
		}
	}

	onMount(() => {
		fetchTimeline(true);
	});
</script>

<div class="space-y-6">
	<!-- Section Header & Filter Navigation -->
	<div
		class="flex flex-col gap-4 border-b border-[#e2e2de] pb-4 sm:flex-row sm:items-center sm:justify-between"
	>
		<div>
			<h2 class="text-lg font-medium text-[#121212]">Activity Timeline</h2>
			<p class="text-xs text-[#666660]">
				Authentic chronological record of your identity, visitor movements, and account events.
			</p>
		</div>

		<!-- Category Filter Pills -->
		<div class="flex flex-wrap gap-1.5" role="tablist" aria-label="Activity Categories">
			{#each categories as cat (cat.id)}
				<button
					type="button"
					role="tab"
					aria-selected={activeCategory === cat.id}
					onclick={() => handleCategoryChange(cat.id)}
					class="rounded-full px-3 py-1 text-xs font-medium transition-colors {activeCategory ===
					cat.id
						? 'bg-[#121212] text-white'
						: 'bg-[#f3f3f1] text-[#666660] hover:bg-[#e8e8e4] hover:text-[#121212]'}"
				>
					{cat.label}
				</button>
			{/each}
		</div>
	</div>

	<!-- Content Area -->
	{#if isLoading}
		<div
			class="flex min-h-[180px] items-center justify-center rounded-xl border border-[#e2e2de] bg-white p-8"
		>
			<div class="flex items-center gap-2.5 text-xs text-[#666660]">
				<Loader2 class="h-4 w-4 animate-spin text-[#121212]" />
				<span>Loading verified timeline...</span>
			</div>
		</div>
	{:else if error}
		<div class="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800">
			<p class="font-medium">Failed to load activity history</p>
			<p class="mt-0.5 text-rose-600">{error}</p>
			<button
				type="button"
				onclick={() => fetchTimeline(true)}
				class="mt-2 text-xs font-medium underline hover:text-rose-950"
			>
				Try again
			</button>
		</div>
	{:else if events.length === 0}
		<div class="rounded-xl border border-dashed border-[#e2e2de] bg-white p-10 text-center">
			<Clock class="mx-auto h-6 w-6 text-[#8a8a82]" />
			<p class="mt-2 text-sm font-medium text-[#121212]">No activity recorded yet</p>
			<p class="mt-0.5 text-xs text-[#666660]">
				Events associated with your verified phone and account will appear here automatically.
			</p>
		</div>
	{:else}
		<div class="space-y-8">
			{#each groupedEvents as group (group.date)}
				<div class="space-y-3">
					<div class="sticky top-0 z-10 flex items-center gap-2 bg-[#f3f3f1] py-1">
						<span class="text-xs font-semibold tracking-wider text-[#8a8a82] uppercase">
							{group.date}
						</span>
						<div class="h-px flex-1 bg-[#e2e2de]"></div>
					</div>

					<div
						class="divide-y divide-[#e2e2de] overflow-hidden rounded-xl border border-[#e2e2de] bg-white shadow-xs"
					>
						{#each group.items as ev (ev.id)}
							{@const IconComponent = getActionIcon(ev.action, ev.category)}
							{@const { time } = formatTimestamp(ev.timestamp)}
							<div class="flex items-start gap-3.5 p-4 transition-colors hover:bg-[#fafaf8]">
								<!-- Action Icon -->
								<div
									class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-[#f3f3f1] text-[#121212]"
								>
									<IconComponent class="h-4 w-4" />
								</div>

								<!-- Content -->
								<div class="min-w-0 flex-1">
									<div
										class="flex flex-col gap-0.5 sm:flex-row sm:items-baseline sm:justify-between"
									>
										<p class="text-sm font-medium text-[#121212]">
											{ev.display_title || ev.title || ev.action}
										</p>
										<time class="shrink-0 text-xs text-[#8a8a82]" datetime={ev.timestamp}>
											{time}
										</time>
									</div>

									{#if ev.display_summary || ev.summary}
										<p class="mt-0.5 text-xs text-[#666660]">
											{ev.display_summary || ev.summary}
										</p>
									{/if}

									<!-- Location / Society context pill tags -->
									{#if ev.society_name || ev.gate_name || ev.unit_identifier}
										<div
											class="mt-2 flex flex-wrap items-center gap-1.5 text-[11px] text-[#666660]"
										>
											{#if ev.society_name}
												<span
													class="inline-flex items-center gap-1 rounded bg-[#f3f3f1] px-1.5 py-0.5 font-medium text-[#121212]"
												>
													<Building2 class="h-3 w-3 text-[#8a8a82]" />
													{ev.society_name}
												</span>
											{/if}
											{#if ev.gate_name}
												<span class="rounded bg-[#f3f3f1] px-1.5 py-0.5 font-medium text-[#666660]">
													{ev.gate_name}
												</span>
											{/if}
											{#if ev.unit_identifier}
												<span class="rounded bg-[#f3f3f1] px-1.5 py-0.5 font-medium text-[#666660]">
													Flat {ev.unit_identifier}
												</span>
											{/if}
										</div>
									{/if}
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/each}

			<!-- Load More Button -->
			{#if offset + limit < total}
				<div class="pt-2 text-center">
					<button
						type="button"
						onclick={loadMore}
						disabled={isLoadingMore}
						class="inline-flex items-center gap-2 rounded-lg border border-[#e2e2de] bg-white px-4 py-2 text-xs font-medium text-[#121212] transition-colors hover:bg-[#f3f3f1] disabled:opacity-50"
					>
						{#if isLoadingMore}
							<Loader2 class="h-3.5 w-3.5 animate-spin" />
							<span>Loading more...</span>
						{:else}
							<span>Load earlier activity ({total - (offset + limit)} remaining)</span>
						{/if}
					</button>
				</div>
			{/if}
		</div>
	{/if}
</div>
