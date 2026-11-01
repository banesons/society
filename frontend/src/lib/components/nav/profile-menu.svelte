<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { screen } from '$lib/screen.svelte';
	import { formatFullPhone } from '$lib/phone';
	import { getInitials } from '$lib/avatar';
	import { ROUTES } from '$lib/routing';
	import Badge from '$lib/components/ui/badge.svelte';
	import { User, Shield, LayoutDashboard, Building2, LogOut, ChevronDown, Smartphone } from '@lucide/svelte';
	import { pwa } from '$lib/pwa.svelte';

	let isOpen = $state(false);
	let avatarImgFailed = $state(false);

	let containerRef = $state<HTMLElement | null>(null);

	const userDisplayName = $derived(
		auth.user?.display_name?.trim() ||
			(auth.user?.first_name || auth.user?.last_name
				? [auth.user?.first_name, auth.user?.last_name].filter(Boolean).join(' ')
				: '')
	);

	const userInitials = $derived(getInitials(userDisplayName || auth.user?.phone || 'Account'));

	function toggleMenu() {
		isOpen = !isOpen;
	}

	function closeMenu() {
		isOpen = false;
	}

	function handlePointerDown(event: PointerEvent) {
		if (!isOpen || !containerRef) return;
		const path = event.composedPath();
		if (!path.includes(containerRef)) {
			closeMenu();
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && isOpen) {
			closeMenu();
		}
	}
</script>

<svelte:window onpointerdown={handlePointerDown} onkeydown={handleKeydown} />

{#if screen.isMobile}
	<!-- MOBILE: Direct navigation to Personal Profile (NO POPUP) -->
	<a
		href={ROUTES.PROFILE}
		class="group flex items-center justify-center rounded-full border border-neutral-200/90 bg-white/90 p-1 text-xs text-neutral-800 shadow-2xs transition-all hover:bg-neutral-100 focus:ring-2 focus:ring-neutral-950 focus:outline-hidden active:scale-95"
		aria-label="View Personal Profile"
	>
		<div
			class="flex h-7 w-7 items-center justify-center overflow-hidden rounded-full bg-neutral-950 text-white shadow-xs"
		>
			{#if auth.user?.profile_photo_url && !avatarImgFailed}
				<img
					src={auth.user.profile_photo_url}
					alt={userDisplayName || 'Profile'}
					class="h-full w-full object-cover"
					onerror={() => (avatarImgFailed = true)}
				/>
			{:else if userDisplayName}
				<span class="font-sans text-[11px] font-semibold tracking-tight text-white">
					{userInitials}
				</span>
			{:else}
				<User class="h-3.5 w-3.5" />
			{/if}
		</div>
	</a>
{:else}
	<!-- DESKTOP: Interactive Popover Menu -->
	<div class="relative" bind:this={containerRef}>
		<!-- Trigger Button (Avatar, Name & Chevron) -->
		<button
			type="button"
			onclick={toggleMenu}
			class="group flex items-center gap-2.5 rounded-full border border-neutral-200/90 bg-white/90 p-1.5 pr-3 text-xs text-neutral-800 shadow-2xs transition-all hover:border-neutral-300 hover:bg-neutral-100 hover:text-neutral-950 focus:ring-2 focus:ring-neutral-950 focus:outline-hidden active:scale-95"
			aria-label="Account profile and session menu"
			aria-haspopup="menu"
			aria-expanded={isOpen}
		>
			<div
				class="flex h-7 w-7 items-center justify-center overflow-hidden rounded-full bg-neutral-950 text-white shadow-xs transition-transform group-hover:scale-105"
			>
				{#if auth.user?.profile_photo_url && !avatarImgFailed}
					<img
						src={auth.user.profile_photo_url}
						alt={userDisplayName || 'Profile'}
						class="h-full w-full object-cover"
						onerror={() => (avatarImgFailed = true)}
					/>
				{:else if userDisplayName}
					<span class="font-sans text-[11px] font-semibold tracking-tight text-white">
						{userInitials}
					</span>
				{:else}
					<User class="h-3.5 w-3.5" />
				{/if}
			</div>
			<span class="max-w-[140px] truncate font-sans text-xs font-medium text-neutral-800">
				{userDisplayName || (auth.user?.phone ? formatFullPhone(auth.user.phone) : 'Account')}
			</span>
			<ChevronDown
				class="h-3 w-3 text-neutral-400 transition-transform duration-200 group-hover:text-neutral-700 {isOpen
					? 'rotate-180'
					: ''}"
			/>
		</button>

		<!-- Popover Menu -->
		{#if isOpen}
			<div
				class="animate-in fade-in zoom-in-95 absolute top-12 right-0 z-50 w-72 origin-top-right overflow-hidden rounded-2xl border border-neutral-200/90 bg-white/98 shadow-xl backdrop-blur-md transition-all"
				role="menu"
				aria-label="User Account Menu"
			>
				<!-- Identity Section -->
				<div class="border-b border-neutral-100 bg-neutral-50/70 p-4">
					<div class="flex items-center justify-between">
						<span class="text-[11px] font-medium tracking-wider text-neutral-500 uppercase">
							Account Identity
						</span>
						{#if auth.isSuperAdmin}
							<Badge variant="outline" class="border-neutral-300 bg-white font-mono text-[10px]">
								SUPER_ADMIN
							</Badge>
						{:else if auth.isPlatformAdmin}
							<Badge variant="outline" class="border-neutral-300 bg-white font-mono text-[10px]">
								ADMIN
							</Badge>
						{/if}
					</div>

					<p class="mt-1.5 truncate font-sans text-sm font-semibold text-neutral-950">
						{userDisplayName || 'Account'}
					</p>
					{#if auth.user?.phone}
						<p class="font-mono text-[11px] text-neutral-500">
							{formatFullPhone(auth.user.phone)}
						</p>
					{/if}
					{#if auth.user?.email}
						<p class="truncate font-sans text-[11px] text-neutral-500">
							{auth.user.email}
						</p>
					{/if}

					{#if auth.user?.society_assignments && auth.user.society_assignments.length > 0}
						<div class="mt-2 flex flex-wrap gap-1">
							{#each auth.user.society_assignments as assignment (assignment.society_id)}
								<span
									class="inline-flex items-center gap-1 rounded-full bg-neutral-200/60 px-2 py-0.5 text-[10px] text-neutral-700"
								>
									<Building2 class="h-2.5 w-2.5" />
									<span>{assignment.society_name} ({assignment.role})</span>
								</span>
							{/each}
						</div>
					{/if}
				</div>

				<!-- Navigation Links -->
				<div class="p-1.5">
					<a
						href={ROUTES.PROFILE}
						onclick={closeMenu}
						class="flex items-center gap-2.5 rounded-xl px-3 py-2 text-xs font-medium text-neutral-800 transition-colors hover:bg-neutral-100 hover:text-neutral-950"
						role="menuitem"
					>
						<User class="h-4 w-4 text-neutral-500" />
						<span>Personal Profile</span>
					</a>

					<a
						href={ROUTES.DASHBOARD}
						onclick={closeMenu}
						class="flex items-center gap-2.5 rounded-xl px-3 py-2 text-xs font-medium text-neutral-800 transition-colors hover:bg-neutral-100 hover:text-neutral-950"
						role="menuitem"
					>
						<LayoutDashboard class="h-4 w-4 text-neutral-500" />
						<span>Dashboard</span>
					</a>

					{#if auth.isPlatformAdmin}
						<a
							href={ROUTES.ADMIN}
							onclick={closeMenu}
							class="flex items-center gap-2.5 rounded-xl px-3 py-2 text-xs font-medium text-neutral-800 transition-colors hover:bg-neutral-100 hover:text-neutral-950"
							role="menuitem"
						>
							<Shield class="h-4 w-4 text-neutral-500" />
							<span>Platform Administration</span>
						</a>
					{/if}

					{#if auth.user?.society_assignments && auth.user.society_assignments.length > 0}
						<a
							href={`/app/societies/${auth.user.society_assignments[0].society_id}`}
							onclick={closeMenu}
							class="flex items-center gap-2.5 rounded-xl px-3 py-2 text-xs font-medium text-neutral-800 transition-colors hover:bg-neutral-100 hover:text-neutral-950"
							role="menuitem"
						>
							<Building2 class="h-4 w-4 text-neutral-500" />
							<span>Society Management</span>
						</a>
					{/if}

					{#if pwa.canInstall}
						<button
							type="button"
							onclick={async () => {
								closeMenu();
								await pwa.promptInstall();
							}}
							class="flex w-full items-center justify-between rounded-xl px-3 py-2 text-left text-xs font-medium text-neutral-900 transition-colors hover:bg-neutral-100"
							role="menuitem"
						>
							<span class="flex items-center gap-2.5">
								<Smartphone class="h-4 w-4 text-neutral-600" />
								<span>Install Society App</span>
							</span>
							<span class="rounded-full bg-neutral-200/80 px-1.5 py-0.5 text-[10px] font-semibold text-neutral-700">PWA</span>
						</button>
					{/if}
				</div>

				<!-- Session & Sign Out Navigation Link -->
				<div class="border-t border-neutral-100 p-1.5">
					<a
						href={ROUTES.PROFILE}
						onclick={closeMenu}
						class="flex items-center justify-between rounded-xl px-3 py-2 text-xs font-medium text-neutral-600 transition-colors hover:bg-neutral-100 hover:text-neutral-950"
						role="menuitem"
					>
						<span class="flex items-center gap-2">
							<LogOut class="h-3.5 w-3.5 text-neutral-400" />
							<span>Manage Sessions & Sign Out</span>
						</span>
						<span class="text-[11px] text-neutral-400">&rarr;</span>
					</a>
				</div>
			</div>
		{/if}
	</div>
{/if}
