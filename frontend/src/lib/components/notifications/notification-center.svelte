<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		Bell,
		X,
		Loader2,
		AlertCircle,
		Volume2,
		Megaphone,
		Receipt,
		CreditCard,
		UserCheck
	} from '@lucide/svelte';
	import {
		notificationsApi,
		type NotificationItem,
		type NotificationListResponse
	} from '$lib/api/notifications';
	import { isPushSupported, getNotificationPermission, registerPushDevice } from '$lib/push';

	interface Props {
		societyId?: string;
	}

	let { societyId }: Props = $props();

	let isOpen = $state(false);
	let isLoading = $state(false);
	let isRegisteringPush = $state(false);
	let pushStatus = $state<'default' | 'granted' | 'denied' | 'unsupported'>('default');
	let notifications = $state<NotificationItem[]>([]);
	let unreadCount = $state(0);
	let totalCount = $state(0);
	let error = $state<string | null>(null);

	async function loadUnreadCount() {
		try {
			const count = await notificationsApi.getUnreadCount(societyId);
			unreadCount = count;
		} catch {
			// Ignore polling error
		}
	}

	async function fetchNotifications() {
		isLoading = true;
		error = null;
		try {
			const res: NotificationListResponse = await notificationsApi.list({
				society_id: societyId,
				limit: 30,
				page: 1
			});
			notifications = res.items;
			totalCount = res.total;
			unreadCount = res.unread_count;
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load notifications';
		} finally {
			isLoading = false;
		}
	}

	function syncPushStatus() {
		if (isPushSupported()) {
			const perm = getNotificationPermission();
			if (perm === 'granted') {
				pushStatus = 'granted';
				registerPushDevice().catch(() => {});
			} else if (perm === 'denied') {
				pushStatus = 'denied';
			} else {
				pushStatus = 'default';
			}
		} else {
			pushStatus = 'unsupported';
		}
	}

	function toggleOpen() {
		isOpen = !isOpen;
		if (isOpen) {
			syncPushStatus();
			fetchNotifications();
		}
	}

	function close() {
		isOpen = false;
	}

	async function handleMarkRead(notif: NotificationItem, event?: MouseEvent) {
		if (event) event.stopPropagation();
		if (notif.read_at) return;

		try {
			const res = await notificationsApi.markAsRead(notif.id);
			notifications = notifications.map((n) =>
				n.id === notif.id ? { ...n, read_at: res.read_at } : n
			);
			unreadCount = Math.max(0, unreadCount - 1);
		} catch (err: unknown) {
			console.error('Failed to mark read:', err);
		}
	}

	async function handleMarkAllRead() {
		try {
			await notificationsApi.markAllAsRead(societyId);
			const now = new Date().toISOString();
			notifications = notifications.map((n) => ({ ...n, read_at: n.read_at || now }));
			unreadCount = 0;
		} catch (err: unknown) {
			console.error('Failed to mark all read:', err);
		}
	}

	async function handleNavigate(notif: NotificationItem) {
		if (!notif.read_at) {
			await handleMarkRead(notif);
		}
		if (notif.action_url) {
			close();
			goto(notif.action_url);
		}
	}

	async function handleEnablePush() {
		isRegisteringPush = true;
		try {
			const result = await registerPushDevice();
			if (result.status === 'granted') {
				pushStatus = 'granted';
			} else if (result.status === 'denied') {
				pushStatus = 'denied';
			} else if (result.status === 'unsupported') {
				pushStatus = 'unsupported';
			}
		} catch {
			pushStatus = 'denied';
		} finally {
			isRegisteringPush = false;
		}
	}

	function formatTimestamp(isoStr: string): string {
		try {
			const date = new Date(isoStr);
			const now = new Date();
			const diffMs = now.getTime() - date.getTime();
			const diffSec = Math.floor(diffMs / 1000);
			const diffMin = Math.floor(diffSec / 60);
			const diffHours = Math.floor(diffMin / 60);
			const diffDays = Math.floor(diffHours / 24);

			if (diffSec < 60) return 'Just now';
			if (diffMin < 60) return `${diffMin}m ago`;
			if (diffHours < 24) return `${diffHours}h ago`;
			if (diffDays < 7) return `${diffDays}d ago`;
			return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
		} catch {
			return isoStr;
		}
	}

	function getNotificationIcon(type: string) {
		switch (type) {
			case 'ANNOUNCEMENT_PUBLISHED':
				return Megaphone;
			case 'VISITOR_APPROVAL_REQUESTED':
			case 'VISIT_APPROVED':
			case 'VISIT_REJECTED':
				return UserCheck;
			case 'BILL_ISSUED':
				return Receipt;
			case 'PAYMENT_RECORDED':
			case 'PAYMENT_REVERSED':
				return CreditCard;
			default:
				return Volume2;
		}
	}

	onMount(() => {
		loadUnreadCount();
		syncPushStatus();

		if (typeof navigator !== 'undefined' && 'permissions' in navigator) {
			navigator.permissions
				.query({ name: 'notifications' as PermissionName })
				.then((status) => {
					status.onchange = () => {
						syncPushStatus();
					};
				})
				.catch(() => {});
		}

		// Polling unread count every 30s
		const interval = setInterval(loadUnreadCount, 30000);
		return () => clearInterval(interval);
	});
</script>

<div class="relative">
	<!-- Trigger Button with Unread Badge -->
	<button
		type="button"
		onclick={toggleOpen}
		class="relative flex h-10 w-10 items-center justify-center rounded-full border border-neutral-200/80 bg-white/90 text-neutral-800 shadow-2xs transition-all hover:border-neutral-300 hover:bg-neutral-100 hover:text-neutral-950 active:scale-95"
		aria-label="Open notifications"
		aria-expanded={isOpen}
	>
		<Bell class="h-4.5 w-4.5" />
		{#if unreadCount > 0}
			<span
				class="absolute -top-1 -right-1 flex h-5 min-w-5 items-center justify-center rounded-full bg-red-600 px-1 text-[11px] font-bold text-white shadow-xs ring-2 ring-white"
			>
				{unreadCount > 99 ? '99+' : unreadCount}
			</span>
		{/if}
	</button>

	<!-- Backdrop for mobile/desktop dismiss -->
	{#if isOpen}
		<button
			type="button"
			class="backdrop-blur-2xs fixed inset-0 z-40 bg-black/20 transition-opacity sm:bg-transparent"
			onclick={close}
			aria-label="Close notification center"
		></button>

		<!-- Notification Panel Popover -->
		<div
			class="fixed top-18 right-3 left-3 z-50 flex max-h-[85vh] flex-col overflow-hidden rounded-2xl border border-neutral-200/90 bg-white/98 shadow-2xl backdrop-blur-md sm:absolute sm:top-12 sm:right-0 sm:left-auto sm:w-[420px]"
			role="dialog"
			aria-label="Notifications Panel"
		>
			<!-- Header -->
			<div class="flex items-center justify-between border-b border-neutral-100 px-5 py-4">
				<div class="flex items-center gap-2.5">
					<h3 class="text-sm font-bold tracking-tight text-neutral-900">Notifications</h3>
					{#if unreadCount > 0}
						<span
							class="rounded-full border border-red-200/60 bg-red-50 px-2 py-0.5 text-xs font-semibold text-red-700"
						>
							{unreadCount} new
						</span>
					{/if}
				</div>

				<div class="flex items-center gap-2">
					{#if unreadCount > 0}
						<button
							type="button"
							onclick={handleMarkAllRead}
							class="text-xs font-medium text-neutral-600 transition-colors hover:text-neutral-950"
							title="Mark all as read"
						>
							Mark all read
						</button>
					{/if}
					<button
						type="button"
						onclick={close}
						class="flex h-7 w-7 items-center justify-center rounded-full text-neutral-400 hover:bg-neutral-100 hover:text-neutral-800"
						aria-label="Close"
					>
						<X class="h-4 w-4" />
					</button>
				</div>
			</div>

			<!-- Web Push Opt-in Prompt Banner -->
			{#if pushStatus === 'default'}
				<div class="border-b border-neutral-100 bg-amber-50/70 px-5 py-3 text-xs">
					<div class="flex items-start justify-between gap-3">
						<div class="space-y-0.5">
							<p class="font-semibold text-amber-900">Enable Desktop Push</p>
							<p class="leading-relaxed text-amber-700/90">
								Receive real-time visitor arrivals, bills, and emergency notices on this browser.
							</p>
						</div>
						<button
							type="button"
							onclick={handleEnablePush}
							disabled={isRegisteringPush}
							class="shrink-0 rounded-lg bg-amber-900 px-3 py-1.5 font-semibold text-white shadow-xs transition-all hover:bg-amber-950 disabled:opacity-50"
						>
							{#if isRegisteringPush}
								<Loader2 class="h-3 w-3 animate-spin" />
							{:else}
								Enable
							{/if}
						</button>
					</div>
				</div>
			{:else if pushStatus === 'denied'}
				<div
					class="border-b border-neutral-100 bg-neutral-50 px-5 py-2.5 text-[11px] text-neutral-500"
				>
					Push notifications are blocked in your browser settings.
				</div>
			{/if}

			<!-- Notification List Body -->
			<div class="flex-1 divide-y divide-neutral-100 overflow-y-auto overscroll-contain">
				{#if isLoading && notifications.length === 0}
					<div class="flex flex-col items-center justify-center py-12 text-neutral-400">
						<Loader2 class="h-6 w-6 animate-spin text-neutral-500" />
						<p class="mt-2 text-xs font-medium">Loading notifications...</p>
					</div>
				{:else if error}
					<div class="flex flex-col items-center justify-center px-4 py-10 text-center">
						<AlertCircle class="h-6 w-6 text-red-500" />
						<p class="mt-2 text-xs font-semibold text-neutral-800">{error}</p>
						<button
							type="button"
							onclick={fetchNotifications}
							class="mt-3 text-xs font-semibold text-neutral-900 underline hover:text-neutral-700"
						>
							Try again
						</button>
					</div>
				{:else if notifications.length === 0}
					<div class="flex flex-col items-center justify-center px-6 py-14 text-center">
						<div
							class="flex h-12 w-12 items-center justify-center rounded-full bg-neutral-100 text-neutral-400"
						>
							<Bell class="h-6 w-6" />
						</div>
						<h4 class="mt-3 text-sm font-semibold text-neutral-800">You're all caught up</h4>
						<p class="mt-1 max-w-[240px] text-xs text-neutral-500">
							New announcements, visitor requests, and billing notices will appear here.
						</p>
					</div>
				{:else}
					{#each notifications as notif (notif.id)}
						{@const Icon = getNotificationIcon(notif.type)}
						<div
							role="button"
							tabindex="0"
							onclick={() => handleNavigate(notif)}
							onkeydown={(e) => {
								if (e.key === 'Enter' || e.key === ' ') handleNavigate(notif);
							}}
							class="group flex w-full cursor-pointer items-start gap-3.5 px-5 py-3.5 text-left transition-colors hover:bg-neutral-50/80 {notif.read_at
								? 'opacity-70'
								: 'bg-white'}"
						>
							<!-- Left Icon -->
							<div
								class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-neutral-200/60 {notif.read_at
									? 'bg-neutral-100 text-neutral-500'
									: 'bg-neutral-900 text-white shadow-2xs'}"
							>
								<Icon class="h-4 w-4" />
							</div>

							<!-- Body -->
							<div class="min-w-0 flex-1">
								<div class="flex items-center justify-between gap-2">
									<h4
										class="truncate text-xs {notif.read_at
											? 'font-medium text-neutral-700'
											: 'font-bold text-neutral-950'}"
									>
										{notif.title}
									</h4>
									<span class="font-mono text-[10px] whitespace-nowrap text-neutral-400">
										{formatTimestamp(notif.created_at)}
									</span>
								</div>

								<p class="mt-0.5 line-clamp-2 text-xs leading-relaxed text-neutral-600">
									{notif.body}
								</p>
							</div>

							<!-- Unread Dot / Action -->
							{#if !notif.read_at}
								<button
									type="button"
									onclick={(e) => handleMarkRead(notif, e)}
									class="mt-1 flex h-4 w-4 shrink-0 items-center justify-center rounded-full text-transparent hover:bg-neutral-200/50 hover:text-neutral-500"
									title="Mark as read"
								>
									<span class="h-2 w-2 rounded-full bg-blue-600"></span>
								</button>
							{/if}
						</div>
					{/each}
				{/if}
			</div>

			<!-- Footer -->
			{#if totalCount > 0}
				<div
					class="flex items-center justify-between border-t border-neutral-100 bg-neutral-50/60 px-5 py-2.5 text-[11px] text-neutral-500"
				>
					<span>Total {totalCount} records</span>
					<button
						type="button"
						onclick={fetchNotifications}
						class="font-medium text-neutral-700 hover:text-neutral-950"
					>
						Refresh
					</button>
				</div>
			{/if}
		</div>
	{/if}
</div>
