<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import { screen } from '$lib/screen.svelte';
	import { ROUTES, appRoutes } from '$lib/routing';
	import { notificationsApi } from '$lib/api/notifications';
	import { societyApi, type SocietyProfile, type UserSocietyAssignment } from '$lib/api/society';
	import {
		getGroupedSocietyNavItems,
		getEffectiveSocietyRole,
		isSocietyModuleActive,
		type SocietyRole
	} from '$lib/navigation';
	import { onMount } from 'svelte';
	import Logo from '$lib/components/ui/logo.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import SocietySelectorDialog from '$lib/components/nav/society-selector-dialog.svelte';
	import { getInitials } from '$lib/avatar';
	import { formatFullPhone } from '$lib/phone';
	import { sidebar } from '$lib/sidebar.svelte';
	import {
		LayoutDashboard,
		Building2,
		Bell,
		Shield,
		User,
		ArrowLeft,
		RefreshCw,
		PanelLeftClose,
		PanelLeftOpen
	} from '@lucide/svelte';

	let unreadCount = $state(0);
	let isSelectorOpen = $state(false);

	async function checkUnread() {
		try {
			const count = await notificationsApi.getUnreadCount();
			unreadCount = count;
		} catch {
			// ignore polling errors
		}
	}

	onMount(() => {
		checkUnread();
		const interval = setInterval(checkUnread, 30000);
		return () => clearInterval(interval);
	});

	let currentPath = $derived(page.url.pathname);

	// Context Resolution
	let societyMatch = $derived(currentPath.match(/^\/app\/societies\/([^/?#]+)/));
	let currentSocietyId = $derived(societyMatch ? societyMatch[1] : null);
	let isSocietyWorkspace = $derived(
		!!currentSocietyId &&
			currentPath.startsWith('/app/societies/') &&
			currentPath !== ROUTES.SOCIETIES &&
			currentPath !== `${ROUTES.SOCIETIES}/`
	);

	let isDashboardActive = $derived(currentPath === ROUTES.DASHBOARD || currentPath === ROUTES.APP);
	let isSocietiesActive = $derived(
		currentPath === ROUTES.SOCIETIES || currentPath === `${ROUTES.SOCIETIES}/`
	);
	let isNotificationsActive = $derived(currentPath.startsWith(ROUTES.NOTIFICATIONS));
	let isProfileActive = $derived(currentPath.startsWith(ROUTES.PROFILE));
	let isAdminActive = $derived(
		currentPath.startsWith(ROUTES.ADMIN) || currentPath.startsWith(ROUTES.LEGACY_ADMIN)
	);

	// Society Data for Contextual Navigation
	let societyProfile = $state<SocietyProfile | null>(null);

	$effect(() => {
		if (currentSocietyId && !auth.isLoading) {
			societyApi
				.getSocietyProfile(currentSocietyId)
				.then((res) => (societyProfile = res))
				.catch(() => {
					// Fallback from assignments if profile API is forbidden or failing
					const assignment = auth.user?.society_assignments?.find(
						(a) => a.society_id === currentSocietyId
					);
					if (assignment) {
						societyProfile = {
							id: assignment.society_id,
							name: assignment.society_name,
							society_code: assignment.society_code,
							status: assignment.status,
							address_line1: '',
							locality: '',
							city: assignment.city || '',
							state: assignment.state || '',
							pincode: ''
						};
					}
				});
		} else {
			societyProfile = null;
		}
	});

	let currentRole = $derived<SocietyRole>(
		getEffectiveSocietyRole(
			auth.user?.platform_roles,
			auth.user?.society_assignments,
			currentSocietyId ?? undefined
		)
	);

	let groupedNav = $derived(
		currentSocietyId ? getGroupedSocietyNavItems(currentRole, currentSocietyId) : []
	);

	// Accessible societies for tenant switcher
	let accessibleSocieties = $state<UserSocietyAssignment[]>([]);

	$effect(() => {
		if (auth.isSuperAdmin) {
			societyApi
				.listSocieties({ limit: 100 })
				.then((res) => {
					accessibleSocieties = res.items
						.filter((s) => s.status === 'ACTIVE')
						.map((s) => ({
							society_id: s.id,
							society_code: s.society_code,
							society_name: s.name,
							role: 'SUPER_ADMIN',
							status: s.status as 'ACTIVE',
							city: s.city,
							state: s.state
						}));
				})
				.catch(() => {
					accessibleSocieties = auth.user?.society_assignments ?? [];
				});
		} else {
			accessibleSocieties = auth.user?.society_assignments ?? [];
		}
	});

	let hasMultipleSocieties = $derived(accessibleSocieties.length > 1);

	function handleSwitchSociety(target: UserSocietyAssignment) {
		isSelectorOpen = false;
		if (target.society_id !== currentSocietyId) {
			if (target.role === 'GATEKEEPER') {
				goto(appRoutes.gate(target.society_id));
			} else {
				goto(appRoutes.society(target.society_id));
			}
		}
	}

	let userDisplayName = $derived.by(() => {
		if (auth.user?.display_name) return auth.user.display_name;
		if (auth.user?.first_name || auth.user?.last_name) {
			return `${auth.user.first_name || ''} ${auth.user.last_name || ''}`.trim();
		}
		if (auth.user?.phone) return formatFullPhone(auth.user.phone);
		return 'Personal Profile';
	});

	let userInitials = $derived.by(() => {
		return getInitials(userDisplayName);
	});
</script>

<!-- ========================================================================= -->
<!-- MOBILE BOTTOM NAVIGATION RAIL (< 768px / md)                              -->
<!-- ========================================================================= -->
<nav
	class="fixed inset-x-0 bottom-0 z-40 flex md:hidden items-center justify-around border-t border-neutral-200/90 bg-white/95 px-2 pt-2 pb-[max(0.75rem,env(safe-area-inset-bottom))] shadow-lg backdrop-blur-md"
	aria-label="Mobile Application Navigation"
>
	<!-- 1. Dashboard -->
	<a
		href={ROUTES.DASHBOARD}
		class="flex flex-1 flex-col items-center justify-center py-1 transition-colors {isDashboardActive
			? 'font-semibold text-neutral-950'
			: 'text-neutral-400 hover:text-neutral-700'}"
		aria-current={isDashboardActive ? 'page' : undefined}
	>
		<div class="relative">
			<LayoutDashboard class="h-5 w-5" />
		</div>
		<span class="mt-1 text-[10px] tracking-tight">Dashboard</span>
	</a>

	<!-- 2. Societies -->
	<a
		href={ROUTES.SOCIETIES}
		class="flex flex-1 flex-col items-center justify-center py-1 transition-colors {isSocietiesActive
			? 'font-semibold text-neutral-950'
			: 'text-neutral-400 hover:text-neutral-700'}"
		aria-current={isSocietiesActive ? 'page' : undefined}
	>
		<div class="relative">
			<Building2 class="h-5 w-5" />
		</div>
		<span class="mt-1 text-[10px] tracking-tight">Societies</span>
	</a>

	<!-- 3. Notifications (with unread badge) -->
	<a
		href={ROUTES.NOTIFICATIONS}
		class="flex flex-1 flex-col items-center justify-center py-1 transition-colors {isNotificationsActive
			? 'font-semibold text-neutral-950'
			: 'text-neutral-400 hover:text-neutral-700'}"
		aria-current={isNotificationsActive ? 'page' : undefined}
	>
		<div class="relative">
			<Bell class="h-5 w-5" />
			{#if unreadCount > 0}
				<span
					class="absolute -top-1 -right-1 flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-rose-600 px-1 font-mono text-[9px] font-bold text-white shadow-xs"
				>
					{unreadCount > 9 ? '9+' : unreadCount}
				</span>
			{/if}
		</div>
		<span class="mt-1 text-[10px] tracking-tight">Alerts</span>
	</a>

	<!-- 4. Profile -->
	<a
		href={ROUTES.PROFILE}
		class="flex flex-1 flex-col items-center justify-center py-1 transition-colors {isProfileActive
			? 'font-semibold text-neutral-950'
			: 'text-neutral-400 hover:text-neutral-700'}"
		aria-current={isProfileActive ? 'page' : undefined}
	>
		<div class="relative">
			<User class="h-5 w-5" />
		</div>
		<span class="mt-1 text-[10px] tracking-tight">Profile</span>
	</a>
</nav>

<!-- ========================================================================= -->
<!-- DESKTOP VERTICAL SIDE NAVIGATION RAIL (>= 768px / md)                     -->
<!-- ========================================================================= -->
<aside
	class="sticky top-0 z-30 hidden h-screen {sidebar.isCollapsed
		? 'w-18'
		: isSocietyWorkspace
			? 'w-64'
			: 'w-56'} shrink-0 flex-col justify-between border-r border-neutral-200/90 bg-[#f8f8f7] py-5 transition-[width] duration-200 md:flex"
	aria-label="Desktop Navigation Rail"
>
		<!-- Top: Platform Brand Logo & Collapse Toggle -->
		{#if sidebar.isCollapsed}
			<div class="flex flex-col items-center gap-2.5 px-2">
				<Logo size="sm" />
				<button
					type="button"
					onclick={() => sidebar.toggle()}
					class="flex h-8 w-8 items-center justify-center rounded-lg text-neutral-400 transition-colors hover:bg-neutral-200/60 hover:text-neutral-900"
					title="Expand sidebar"
					aria-label="Expand sidebar"
				>
					<PanelLeftOpen class="h-4 w-4" />
				</button>
			</div>
		{:else}
			<div class="flex items-center justify-between px-4 lg:px-5">
				<Logo size="md" />
				<button
					type="button"
					onclick={() => sidebar.toggle()}
					class="flex h-8 w-8 items-center justify-center rounded-lg text-neutral-400 transition-colors hover:bg-neutral-200/60 hover:text-neutral-900"
					title="Collapse sidebar"
					aria-label="Collapse sidebar"
				>
					<PanelLeftClose class="h-4 w-4" />
				</button>
			</div>
		{/if}

		{#if isSocietyWorkspace}
			<!-- STATE B — CONTEXTUAL SOCIETY WORKSPACE SUBMENU -->
			<div class="mt-3 flex flex-1 flex-col overflow-hidden">
				<!-- Exit to Workspace Directory -->
				<div class={sidebar.isCollapsed ? 'flex justify-center px-2' : 'px-4'}>
					<a
						href={ROUTES.SOCIETIES}
						class="inline-flex items-center gap-1.5 text-xs font-medium text-neutral-500 transition-colors hover:text-neutral-900 {sidebar.isCollapsed
							? 'h-9 w-9 justify-center rounded-xl hover:bg-neutral-200/60'
							: ''}"
						title="Return to Societies Directory"
						aria-label="Return to Societies Directory"
					>
						<ArrowLeft class="h-3.5 w-3.5" />
						{#if !sidebar.isCollapsed}
							<span>All Societies</span>
						{/if}
					</a>
				</div>

				<!-- Selected Society Identity Banner -->
				{#if sidebar.isCollapsed}
					<div class="mx-2 mt-2 flex justify-center">
						<button
							type="button"
							onclick={() => (isSelectorOpen = true)}
							class="flex h-10 w-10 items-center justify-center rounded-xl border border-neutral-200/90 bg-white shadow-2xs transition-colors hover:bg-neutral-100"
							title={`${societyProfile?.name || 'Society'} (${currentRole}) — Switch Tenant`}
							aria-label="Switch Tenant"
						>
							<Building2 class="h-4 w-4 text-neutral-700" />
						</button>
					</div>
				{:else}
					<div
						class="mx-3 mt-3 rounded-2xl border border-neutral-200/90 bg-white/90 p-3 shadow-2xs"
					>
						<div class="flex items-start justify-between gap-2">
							<div class="min-w-0 flex-1">
								<h3
									class="truncate font-serif text-xs font-semibold text-neutral-950"
									title={societyProfile?.name || 'Society'}
								>
									{societyProfile?.name || 'Loading Society...'}
								</h3>
								<div class="mt-0.5 flex items-center gap-1.5 font-mono text-[10px] text-neutral-500">
									<span class="font-semibold text-neutral-700">
										{societyProfile?.society_code || ''}
									</span>
									{#if societyProfile?.city}
										<span>·</span>
										<span class="truncate">{societyProfile.city}</span>
									{/if}
								</div>
							</div>
							<Badge variant="outline" class="px-1.5 py-0 font-mono text-[9px]">
								{currentRole}
							</Badge>
						</div>

						{#if hasMultipleSocieties || auth.isSuperAdmin}
							<button
								type="button"
								onclick={() => (isSelectorOpen = true)}
								class="mt-2.5 flex w-full items-center justify-between rounded-lg border border-neutral-200 bg-neutral-50/80 px-2 py-1 text-[10px] font-medium text-neutral-700 transition-colors hover:bg-neutral-100"
							>
								<span class="flex items-center gap-1.5">
									<RefreshCw class="h-3 w-3 text-neutral-500" />
									<span>Switch Tenant</span>
								</span>
								<span class="text-neutral-400">→</span>
							</button>
						{/if}
					</div>
				{/if}

				<!-- Contextual Submenu Groups (OPERATIONS, PEOPLE, INSIGHTS) -->
				<nav
					class="mt-3 flex-1 space-y-3 overflow-y-auto {sidebar.isCollapsed ? 'px-2' : 'px-3'}"
					aria-label="Society Modules"
				>
					{#each groupedNav as section (section.group)}
						<div>
							{#if !sidebar.isCollapsed}
								<p class="px-3 text-[10px] font-semibold tracking-wider text-neutral-400 uppercase">
									{section.label}
								</p>
							{:else}
								<div class="mx-auto my-1.5 h-px w-6 bg-neutral-200/90" aria-hidden="true"></div>
							{/if}
							<div class="mt-1 space-y-0.5">
								{#each section.items as item (item.id)}
									{@const Icon = item.icon}
									{@const isActive = isSocietyModuleActive(
										item.href,
										currentPath,
										currentSocietyId ?? ''
									)}
									<a
										href={item.href}
										class="group flex items-center {sidebar.isCollapsed
											? 'h-9 w-9 justify-center rounded-xl'
											: 'gap-2.5 rounded-xl px-3 py-2 text-xs'} font-medium transition-all {isActive
											? 'bg-neutral-950 font-semibold text-white shadow-xs'
											: 'text-neutral-600 hover:bg-neutral-200/60 hover:text-neutral-950'}"
										title={item.label}
										aria-label={item.label}
										aria-current={isActive ? 'page' : undefined}
									>
										<Icon
											class="h-4 w-4 shrink-0 {isActive
												? 'text-white'
												: 'text-neutral-500 group-hover:text-neutral-900'}"
										/>
										{#if !sidebar.isCollapsed}
											<span class="truncate">{item.label}</span>
										{/if}
									</a>
								{/each}
							</div>
						</div>
					{/each}
				</nav>
			</div>
		{:else}
			<!-- STATE A — PRIMARY GENERAL WORKSPACE NAVIGATION -->
			<nav
				class="mt-6 flex flex-1 flex-col gap-1.5 {sidebar.isCollapsed
					? 'items-center px-2'
					: 'px-3'}"
			>
				<!-- Dashboard -->
				<a
					href={ROUTES.DASHBOARD}
					class="group flex items-center {sidebar.isCollapsed
						? 'h-10 w-10 justify-center rounded-xl'
						: 'gap-3 rounded-2xl px-3.5 py-2.5 text-xs'} font-medium transition-all {isDashboardActive
						? 'bg-neutral-950 text-white shadow-xs'
						: 'text-neutral-600 hover:bg-neutral-200/60 hover:text-neutral-950'}"
					title="Workspace Dashboard"
					aria-label="Workspace Dashboard"
					aria-current={isDashboardActive ? 'page' : undefined}
				>
					<LayoutDashboard class="h-4 w-4 shrink-0" />
					{#if !sidebar.isCollapsed}
						<span class="inline">Dashboard</span>
					{/if}
				</a>

				<!-- Societies Workspace -->
				<a
					href={ROUTES.SOCIETIES}
					class="group flex items-center {sidebar.isCollapsed
						? 'h-10 w-10 justify-center rounded-xl'
						: 'gap-3 rounded-2xl px-3.5 py-2.5 text-xs'} font-medium transition-all {isSocietiesActive
						? 'bg-neutral-950 text-white shadow-xs'
						: 'text-neutral-600 hover:bg-neutral-200/60 hover:text-neutral-950'}"
					title="Societies"
					aria-label="Societies"
					aria-current={isSocietiesActive ? 'page' : undefined}
				>
					<Building2 class="h-4 w-4 shrink-0" />
					{#if !sidebar.isCollapsed}
						<span class="inline">Societies</span>
					{/if}
				</a>

				<!-- Platform Administration (if authorized) -->
				{#if auth.isPlatformAdmin}
					<a
						href={ROUTES.ADMIN}
						class="group flex items-center {sidebar.isCollapsed
							? 'h-10 w-10 justify-center rounded-xl'
							: 'gap-3 rounded-2xl px-3.5 py-2.5 text-xs'} font-medium transition-all {isAdminActive
							? 'bg-neutral-950 text-white shadow-xs'
							: 'text-neutral-600 hover:bg-neutral-200/60 hover:text-neutral-950'}"
						title="Platform Administration"
						aria-label="Platform Administration"
						aria-current={isAdminActive ? 'page' : undefined}
					>
						<Shield class="h-4 w-4 shrink-0" />
						{#if !sidebar.isCollapsed}
							<span class="inline">Platform Admin</span>
						{/if}
					</a>
				{/if}
			</nav>
		{/if}

		<!-- Bottom: Personal Identity & Profile Link -->
		<div
			class="border-t border-neutral-200/80 {sidebar.isCollapsed
				? 'flex justify-center px-2'
				: 'px-3'} pt-3"
		>
			<a
				href={ROUTES.PROFILE}
				class="flex items-center {sidebar.isCollapsed
					? 'h-10 w-10 justify-center rounded-xl'
					: 'gap-3 rounded-2xl p-2'} transition-colors {isProfileActive
					? 'bg-neutral-200/80 text-neutral-950'
					: 'text-neutral-700 hover:bg-neutral-200/50 hover:text-neutral-950'}"
				title={userDisplayName}
				aria-label={`Personal Profile: ${userDisplayName}`}
				aria-current={isProfileActive ? 'page' : undefined}
			>
				<div
					class="flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded-full bg-neutral-950 font-serif text-xs font-semibold text-white shadow-xs"
				>
					{#if auth.user?.profile_photo_url}
						<img
							src={auth.user.profile_photo_url}
							alt={userDisplayName}
							class="h-full w-full object-cover"
						/>
					{:else}
						<span>{userInitials}</span>
					{/if}
				</div>
				{#if !sidebar.isCollapsed}
					<div class="min-w-0 flex-1">
						<p class="truncate text-xs font-semibold text-neutral-900">
							{userDisplayName}
						</p>
						<p class="text-[10px] text-neutral-400 capitalize">Personal Identity</p>
					</div>
				{/if}
			</a>
		</div>
	</aside>

<SocietySelectorDialog
	isOpen={isSelectorOpen}
	societies={accessibleSocieties}
	currentSocietyId={currentSocietyId ?? ''}
	isDismissible={true}
	onClose={() => (isSelectorOpen = false)}
	onSelect={handleSwitchSociety}
/>
