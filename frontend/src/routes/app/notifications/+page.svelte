<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { toast } from '$lib/toast.svelte';
	import {
		notificationsApi,
		type NotificationItem,
		type NotificationListResponse
	} from '$lib/api/notifications';
	import Button from '$lib/components/ui/button.svelte';
	import {
		Bell,
		CheckCheck,
		Loader2,
		Megaphone,
		Receipt,
		UserCheck,
		ArrowRight
	} from '@lucide/svelte';

	let notifications = $state<NotificationItem[]>([]);
	let unreadCount = $state(0);
	let isLoading = $state(true);
	let isMarkingAll = $state(false);
	let activeFilter = $state<'all' | 'unread' | 'announcements' | 'billing' | 'visitors'>('all');

	async function loadNotifications() {
		isLoading = true;
		try {
			const res: NotificationListResponse = await notificationsApi.list({
				limit: 50,
				page: 1,
				unread_only: activeFilter === 'unread' ? true : undefined
			});
			notifications = res.items;
			unreadCount = res.unread_count;
		} catch (err: unknown) {
			const msg = err instanceof Error ? err.message : 'Failed to retrieve notifications';
			toast.error(msg);
		} finally {
			isLoading = false;
		}
	}

	async function handleMarkAllAsRead() {
		if (unreadCount === 0) return;
		isMarkingAll = true;
		try {
			await notificationsApi.markAllAsRead();
			const now = new Date().toISOString();
			notifications = notifications.map((n) => ({ ...n, read_at: n.read_at || now }));
			unreadCount = 0;
			toast.success('All notifications marked as read');
		} catch {
			toast.error('Failed to mark notifications as read');
		} finally {
			isMarkingAll = false;
		}
	}

	async function handleMarkSingleRead(item: NotificationItem) {
		if (item.read_at) return;
		try {
			const res = await notificationsApi.markAsRead(item.id);
			item.read_at = res.read_at;
			unreadCount = Math.max(0, unreadCount - 1);
		} catch {
			// Silently fail on optimistic mark
		}
	}

	function handleNotificationClick(item: NotificationItem) {
		handleMarkSingleRead(item);
		if (item.action_url) {
			goto(item.action_url);
		}
	}

	function getNotificationIcon(type: string) {
		switch (type) {
			case 'ANNOUNCEMENT':
				return Megaphone;
			case 'MAINTENANCE_BILL':
			case 'PAYMENT_RECEIPT':
				return Receipt;
			case 'VISIT_REQUEST':
			case 'GATE_PASS':
				return UserCheck;
			default:
				return Bell;
		}
	}

	function formatTimeAgo(dateStr: string): string {
		try {
			const date = new Date(dateStr);
			const now = new Date();
			const diffSeconds = Math.floor((now.getTime() - date.getTime()) / 1000);

			if (diffSeconds < 60) return 'Just now';
			if (diffSeconds < 3600) return `${Math.floor(diffSeconds / 60)}m ago`;
			if (diffSeconds < 86400) return `${Math.floor(diffSeconds / 3600)}h ago`;
			if (diffSeconds < 604800) return `${Math.floor(diffSeconds / 86400)}d ago`;

			return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
		} catch {
			return dateStr;
		}
	}

	let filteredNotifications = $derived.by(() => {
		if (activeFilter === 'unread') {
			return notifications.filter((n) => !n.read_at);
		}
		if (activeFilter === 'announcements') {
			return notifications.filter((n) => n.type === 'ANNOUNCEMENT');
		}
		if (activeFilter === 'billing') {
			return notifications.filter(
				(n) => n.type === 'MAINTENANCE_BILL' || n.type === 'PAYMENT_RECEIPT'
			);
		}
		if (activeFilter === 'visitors') {
			return notifications.filter((n) => n.type === 'VISIT_REQUEST' || n.type === 'GATE_PASS');
		}
		return notifications;
	});

	import Breadcrumbs from '$lib/components/nav/breadcrumbs.svelte';

	onMount(() => {
		loadNotifications();
	});
</script>

<svelte:head>
	<title>Notifications — CHS Neu</title>
	<meta
		name="description"
		content="Official society notifications, visitor alerts, and billing updates"
	/>
</svelte:head>

<div class="mx-auto w-full max-w-4xl px-3 pb-24 sm:px-6 sm:pb-16">
	<!-- Top Breadcrumbs -->
	<div class="mb-4">
		<Breadcrumbs />
	</div>

	<!-- Screen Header -->
	<div
		class="mb-6 flex flex-col gap-3 border-b border-neutral-200/80 pb-4 sm:flex-row sm:items-center sm:justify-between"
	>
		<div>
			<div class="flex items-center gap-2.5">
				<h1 class="font-serif text-2xl font-semibold tracking-tight text-neutral-950 sm:text-3xl">
					Notifications
				</h1>
				{#if unreadCount > 0}
					<span
						class="inline-flex items-center rounded-full bg-neutral-950 px-2 py-0.5 font-mono text-[11px] font-medium text-white"
					>
						{unreadCount} new
					</span>
				{/if}
			</div>
			<p class="mt-1 text-xs text-neutral-500 sm:text-sm">
				Broadcast announcements, visitor permissions, and society financial notices.
			</p>
		</div>

		{#if unreadCount > 0}
			<Button
				variant="outline"
				size="sm"
				onclick={handleMarkAllAsRead}
				disabled={isMarkingAll}
				class="self-start text-xs sm:self-auto"
			>
				{#if isMarkingAll}
					<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					<span>Marking read...</span>
				{:else}
					<CheckCheck class="mr-1.5 h-3.5 w-3.5 text-neutral-600" />
					<span>Mark all as read</span>
				{/if}
			</Button>
		{/if}
	</div>

	<!-- Category Filters -->
	<div class="mb-6 flex scrollbar-none overflow-x-auto pb-1">
		<div class="flex items-center gap-1.5 sm:gap-2">
			<button
				type="button"
				onclick={() => {
					activeFilter = 'all';
					loadNotifications();
				}}
				class="rounded-full px-3.5 py-1.5 text-xs font-medium transition-colors {activeFilter ===
				'all'
					? 'bg-neutral-950 text-white'
					: 'border border-neutral-200 bg-white text-neutral-700 hover:bg-neutral-100'}"
			>
				All
			</button>

			<button
				type="button"
				onclick={() => {
					activeFilter = 'unread';
					loadNotifications();
				}}
				class="rounded-full px-3.5 py-1.5 text-xs font-medium transition-colors {activeFilter ===
				'unread'
					? 'bg-neutral-950 text-white'
					: 'border border-neutral-200 bg-white text-neutral-700 hover:bg-neutral-100'}"
			>
				Unread {unreadCount > 0 ? `(${unreadCount})` : ''}
			</button>

			<button
				type="button"
				onclick={() => (activeFilter = 'announcements')}
				class="rounded-full px-3.5 py-1.5 text-xs font-medium transition-colors {activeFilter ===
				'announcements'
					? 'bg-neutral-950 text-white'
					: 'border border-neutral-200 bg-white text-neutral-700 hover:bg-neutral-100'}"
			>
				Announcements
			</button>

			<button
				type="button"
				onclick={() => (activeFilter = 'visitors')}
				class="rounded-full px-3.5 py-1.5 text-xs font-medium transition-colors {activeFilter ===
				'visitors'
					? 'bg-neutral-950 text-white'
					: 'border border-neutral-200 bg-white text-neutral-700 hover:bg-neutral-100'}"
			>
				Visitors
			</button>

			<button
				type="button"
				onclick={() => (activeFilter = 'billing')}
				class="rounded-full px-3.5 py-1.5 text-xs font-medium transition-colors {activeFilter ===
				'billing'
					? 'bg-neutral-950 text-white'
					: 'border border-neutral-200 bg-white text-neutral-700 hover:bg-neutral-100'}"
			>
				Billing
			</button>
		</div>
	</div>

	<!-- Notification Feed -->
	{#if isLoading}
		<div class="flex flex-col items-center justify-center py-20 text-center">
			<Loader2 class="h-7 w-7 animate-spin text-neutral-900" />
			<p class="mt-3 text-xs text-neutral-500">Checking for new notifications...</p>
		</div>
	{:else if filteredNotifications.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-3xl border border-dashed border-neutral-300 bg-white/70 px-4 py-16 text-center shadow-xs"
		>
			<div
				class="flex h-14 w-14 items-center justify-center rounded-full bg-neutral-100 text-neutral-400"
			>
				<Bell class="h-7 w-7" />
			</div>
			<h2 class="mt-4 font-serif text-base font-semibold text-neutral-950">You're all caught up</h2>
			<p class="mt-1 max-w-sm text-xs text-neutral-500">
				{#if activeFilter === 'unread'}
					No unread notifications. All active items have been reviewed.
				{:else}
					New society announcements, gate passes, and billing receipts will appear here.
				{/if}
			</p>
		</div>
	{:else}
		<div class="space-y-3">
			{#each filteredNotifications as item (item.id)}
				{@const IconComponent = getNotificationIcon(item.type)}
				<div
					onclick={() => handleNotificationClick(item)}
					onkeydown={(e) => {
						if (e.key === 'Enter' || e.key === ' ') {
							e.preventDefault();
							handleNotificationClick(item);
						}
					}}
					role="button"
					tabindex="0"
					class="group relative flex items-start gap-3.5 rounded-2xl border p-4 text-left transition-all {item.read_at
						? 'border-neutral-200/80 bg-white/70 hover:border-neutral-300 hover:bg-white'
						: 'border-neutral-950/20 bg-white shadow-xs hover:border-neutral-950/40'}"
				>
					<!-- Unread Indicator Dot -->
					{#if !item.read_at}
						<span
							class="absolute top-4 left-2 h-2 w-2 rounded-full bg-neutral-950"
							aria-label="Unread notification"
						></span>
					{/if}

					<!-- Icon Badge -->
					<div
						class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl transition-colors {item.read_at
							? 'bg-neutral-100 text-neutral-600'
							: 'bg-neutral-950 text-white'}"
					>
						<IconComponent class="h-4 w-4" />
					</div>

					<!-- Content Body -->
					<div class="min-w-0 flex-1 pr-2">
						<div class="flex items-center justify-between gap-2">
							<h3
								class="text-xs font-semibold sm:text-sm {item.read_at
									? 'text-neutral-800'
									: 'font-bold text-neutral-950'}"
							>
								{item.title}
							</h3>
							<span class="shrink-0 font-mono text-[10px] text-neutral-400 sm:text-xs">
								{formatTimeAgo(item.created_at)}
							</span>
						</div>

						<p class="mt-1 line-clamp-2 text-xs leading-relaxed text-neutral-600">
							{item.body}
						</p>

						{#if item.action_url}
							<div
								class="mt-2 inline-flex items-center gap-1 text-[11px] font-medium text-neutral-900 group-hover:underline"
							>
								<span>View Details</span>
								<ArrowRight class="h-3 w-3 transition-transform group-hover:translate-x-0.5" />
							</div>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
