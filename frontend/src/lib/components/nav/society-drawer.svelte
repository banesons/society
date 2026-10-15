<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import type { SocietyProfile } from '$lib/api/society';
	import { formatFullPhone } from '$lib/phone';
	import Badge from '$lib/components/ui/badge.svelte';
	import {
		Building2,
		Users,
		Shield,
		Layers,
		DoorClosed,
		Wrench,
		FileText,
		MessageSquare,
		BarChart3,
		Settings,
		RefreshCw,
		LogOut,
		ShieldAlert,
		X,
		MapPin,
		User,
		Home,
		ScrollText
	} from '@lucide/svelte';

	interface Props {
		isOpen: boolean;
		profile: SocietyProfile | null;
		societyId: string;
		activeSection?: string;
		hasMultipleSocieties?: boolean;
		onClose: () => void;
		onSwitchSociety: () => void;
		onLogoutAll: () => void;
	}

	let {
		isOpen = false,
		profile,
		societyId,
		activeSection = 'overview',
		hasMultipleSocieties = false,
		onClose,
		onSwitchSociety,
		onLogoutAll
	}: Props = $props();

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && isOpen) {
			onClose();
		}
	}

	let navItems = $derived([
		{
			id: 'overview',
			label: 'Overview',
			href: `/society/${societyId}#overview`,
			icon: Building2,
			implemented: true
		},
		{
			id: 'members',
			label: 'Members & Residents',
			href: `/society/${societyId}#members`,
			icon: Users,
			implemented: true
		},
		{
			id: 'committee',
			label: 'Managing Committee',
			href: `/society/${societyId}#committee`,
			icon: Shield,
			implemented: true
		},
		{
			id: 'structure',
			label: 'Physical Structure',
			href: `/society/${societyId}#structure`,
			icon: Layers,
			implemented: true
		},
		{
			id: 'resident-hub',
			label: 'Resident Hub',
			href: `/society/${societyId}#resident-hub`,
			icon: Home,
			implemented: true
		},
		{
			id: 'gatekeeping',
			label: 'Gatekeeping',
			href: `/society/${societyId}#gatekeeping`,
			icon: DoorClosed,
			implemented: true
		},
		{
			id: 'announcements',
			label: 'Announcements',
			href: `/society/${societyId}#announcements`,
			icon: MessageSquare,
			implemented: true
		},
		{
			id: 'maintenance',
			label: 'Maintenance',
			href: `/society/${societyId}#maintenance`,
			icon: Wrench,
			implemented: true
		},
		{
			id: 'documents',
			label: 'Documents',
			href: `/society/${societyId}#documents`,
			icon: FileText,
			implemented: true
		},
		{
			id: 'audit',
			label: 'Audit & Activity',
			href: `/society/${societyId}#audit`,
			icon: ScrollText,
			implemented: true
		},
		{
			id: 'reports',
			label: 'Dashboard & Reports',
			href: `/society/${societyId}#reports`,
			icon: BarChart3,
			implemented: true
		},
		{ id: 'settings', label: 'Settings', icon: Settings, implemented: false }
	]);

	let currentRole = $derived(() => {
		if (auth.isSuperAdmin) return 'SUPER_ADMIN';
		if (auth.isPlatformAdmin) return 'ADMIN';
		const match = auth.user?.society_assignments?.find((a) => a.society_id === societyId);
		return match?.role || 'SOC_ADMIN';
	});

	function getStatusVariant(status?: string): 'default' | 'success' | 'warning' | 'destructive' {
		switch (status) {
			case 'ACTIVE':
				return 'success';
			case 'SUSPENDED':
				return 'destructive';
			default:
				return 'warning';
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex justify-end bg-neutral-950/40 backdrop-blur-xs transition-opacity duration-200"
		role="dialog"
		aria-modal="true"
		aria-label="Society navigation and account menu"
	>
		<!-- Backdrop click to dismiss -->
		<button
			type="button"
			class="absolute inset-0 h-full w-full cursor-default"
			onclick={onClose}
			aria-label="Close menu backdrop"
			tabindex="-1"
		></button>

		<!-- Drawer Container (anchored right on desktop, full-width on mobile) -->
		<aside
			class="relative flex h-full w-full max-w-sm flex-col border-l border-neutral-200 bg-white text-neutral-900 shadow-2xl sm:max-w-md"
		>
			<!-- Top Drawer Header with Close Button -->
			<div class="flex items-center justify-between border-b border-neutral-200 px-6 py-5">
				<span class="text-xs font-semibold tracking-wider text-neutral-500 uppercase">
					Society Menu
				</span>
				<button
					type="button"
					onclick={onClose}
					class="flex h-9 w-9 items-center justify-center rounded-full text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-950"
					aria-label="Close menu"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			<!-- Drawer Scrollable Content -->
			<div class="flex-1 space-y-6 overflow-y-auto px-6 py-5">
				<!-- GROUP A — CURRENT SOCIETY -->
				<section class="rounded-2xl border border-neutral-200 bg-neutral-50/70 p-4">
					{#if profile}
						<div class="flex items-start justify-between gap-3">
							<div>
								<h3 class="font-serif text-base font-semibold text-neutral-950">
									{profile.name}
								</h3>
								<div class="mt-1 flex items-center gap-2 font-mono text-xs text-neutral-600">
									<span class="rounded bg-neutral-200/70 px-1.5 py-0.5 font-bold text-neutral-900">
										{profile.society_code}
									</span>
									<span class="text-neutral-400">·</span>
									<span class="inline-flex items-center gap-1 font-sans text-neutral-600">
										<MapPin class="h-3 w-3" />
										{profile.city}, {profile.state}
									</span>
								</div>
							</div>
							<Badge variant={getStatusVariant(profile.status)} class="text-[10px]">
								{profile.status}
							</Badge>
						</div>

						<div
							class="mt-3 flex items-center justify-between border-t border-neutral-200/80 pt-2.5 text-xs"
						>
							<span class="text-neutral-500">Verified Role</span>
							<Badge variant="outline" class="font-mono text-[11px] text-neutral-800">
								{currentRole()}
							</Badge>
						</div>
					{:else}
						<div class="text-xs text-neutral-500">Loading society identity...</div>
					{/if}
				</section>

				<!-- GROUP B — SOCIETY NAVIGATION -->
				<section>
					<h4 class="mb-2 text-xs font-semibold tracking-wider text-neutral-500 uppercase">
						Navigation
					</h4>
					<nav class="space-y-1">
						{#each navItems as item (item.id)}
							{@const Icon = item.icon}
							{#if item.implemented}
								{@const isActive = activeSection === item.id}
								<a
									href={item.href}
									onclick={onClose}
									class="group flex items-center justify-between rounded-xl px-3.5 py-2.5 text-xs font-medium transition-colors {isActive
										? 'bg-neutral-950 font-semibold text-white shadow-xs'
										: 'text-neutral-800 hover:bg-neutral-100 hover:text-neutral-950'}"
								>
									<div class="flex items-center gap-3">
										<Icon
											class="h-4 w-4 {isActive
												? 'text-white'
												: 'text-neutral-500 group-hover:text-neutral-900'}"
										/>
										<span>{item.label}</span>
									</div>
									{#if isActive}
										<span class="h-1.5 w-1.5 rounded-full bg-white"></span>
									{/if}
								</a>
							{:else}
								<!-- Unimplemented future module -->
								<div
									class="flex cursor-not-allowed items-center justify-between rounded-xl px-3.5 py-2.5 text-xs text-neutral-400 select-none"
									title="Module planned for future release"
								>
									<div class="flex items-center gap-3">
										<Icon class="h-4 w-4 text-neutral-300" />
										<span>{item.label}</span>
									</div>
									<span
										class="rounded bg-neutral-100 px-1.5 py-0.5 text-[9px] font-medium text-neutral-400"
									>
										Upcoming
									</span>
								</div>
							{/if}
						{/each}
					</nav>
				</section>

				<!-- GROUP C — SWITCH SOCIETY -->
				{#if hasMultipleSocieties || auth.isSuperAdmin}
					<section class="border-t border-neutral-200 pt-5">
						<button
							type="button"
							onclick={() => {
								onClose();
								onSwitchSociety();
							}}
							class="flex w-full items-center justify-between rounded-xl border border-neutral-300 bg-white px-3.5 py-3 text-xs font-semibold text-neutral-900 shadow-2xs transition-colors hover:border-neutral-950 hover:bg-neutral-50"
						>
							<div class="flex items-center gap-2.5">
								<RefreshCw class="h-4 w-4 text-neutral-700" />
								<span>Switch Society</span>
							</div>
							<span class="text-[11px] font-normal text-neutral-500"> Choose tenant → </span>
						</button>
					</section>
				{/if}

				<!-- GROUP D — ACCOUNT -->
				<section class="border-t border-neutral-200 pt-5">
					<h4 class="mb-3 text-xs font-semibold tracking-wider text-neutral-500 uppercase">
						Account & Session
					</h4>

					{#if auth.user}
						<div class="rounded-xl border border-neutral-200 bg-neutral-50/70 p-3.5">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-2.5">
									<div
										class="flex h-7 w-7 items-center justify-center rounded-full bg-neutral-200 text-neutral-700"
									>
										<User class="h-3.5 w-3.5" />
									</div>
									<div>
										<p class="font-mono text-xs font-semibold text-neutral-900">
											{formatFullPhone(auth.user.phone)}
										</p>
										<p class="text-[11px] text-neutral-500">
											{auth.isSuperAdmin
												? 'Platform Super Admin'
												: auth.isPlatformAdmin
													? 'Platform Admin'
													: 'Society Administrator'}
										</p>
									</div>
								</div>
							</div>
						</div>

						<div class="mt-3 space-y-2">
							<button
								type="button"
								onclick={() => {
									onClose();
									auth.logout();
								}}
								class="flex w-full items-center gap-2.5 rounded-xl px-3.5 py-2.5 text-xs font-medium text-neutral-700 transition-colors hover:bg-neutral-100 hover:text-neutral-950"
							>
								<LogOut class="h-4 w-4 text-neutral-500" />
								<span>Log Out This Device</span>
							</button>

							<button
								type="button"
								onclick={() => {
									onClose();
									onLogoutAll();
								}}
								class="flex w-full items-center gap-2.5 rounded-xl px-3.5 py-2.5 text-xs font-medium text-rose-600 transition-colors hover:bg-rose-50 hover:text-rose-800"
							>
								<ShieldAlert class="h-4 w-4 text-rose-500" />
								<span>Log Out All Devices</span>
							</button>
						</div>
					{/if}
				</section>
			</div>
		</aside>
	</div>
{/if}
