<script lang="ts">
	import Logo from '$lib/components/ui/logo.svelte';
	import NotificationCenter from '$lib/components/notifications/notification-center.svelte';
	import ProfileMenu from '$lib/components/nav/profile-menu.svelte';
	import type { SocietyProfile } from '$lib/api/society';
	import type { BackDestination } from '$lib/routing';
	import { ArrowLeft, Menu, Building2 } from '@lucide/svelte';

	interface Props {
		profile: SocietyProfile | null;
		backDestination: BackDestination;
		isMenuOpen: boolean;
		onToggleMenu: () => void;
	}

	let { profile, backDestination, isMenuOpen, onToggleMenu }: Props = $props();
</script>

<!-- Society Application Shell Header -->
<header class="sticky top-0 z-40 border-b border-neutral-200/70 bg-[#f3f3f1]/95 backdrop-blur-md">
	<!-- Mobile App Bar Layout (< 768px / md) -->
	<div class="flex h-16 items-center justify-between px-4 md:hidden">
		<!-- Left: Leading Back Arrow + Brand Logo -->
		<div class="flex items-center gap-2">
			<a
				href={backDestination.href}
				class="flex h-11 w-11 items-center justify-center rounded-full text-neutral-800 transition-colors hover:bg-neutral-200/60 active:scale-95"
				aria-label={backDestination.label}
				title={backDestination.label}
			>
				<ArrowLeft class="h-5 w-5" />
			</a>

			<div class="pl-1">
				<Logo size="sm" />
			</div>
		</div>

		<!-- Right: Notifications + Menu Button -->
		<div class="flex items-center gap-1.5">
			<NotificationCenter societyId={profile?.id} />
			<button
				type="button"
				onclick={onToggleMenu}
				class="flex h-11 w-11 items-center justify-center rounded-full text-neutral-900 transition-colors hover:bg-neutral-200/60 active:scale-95"
				aria-label="Open society navigation menu"
				aria-expanded={isMenuOpen}
			>
				<Menu class="h-5 w-5" />
			</button>
		</div>
	</div>

	<!-- Desktop Header Layout (>= 768px / md) -->
	<div
		class="mx-auto hidden h-20 max-w-(--breakpoint-2xl) items-center justify-between px-6 md:flex sm:px-10 lg:px-16"
	>
		<!-- Left: Contextual Society Indicator (no duplicate brand logo on desktop) -->
		<div class="flex items-center gap-3">
			{#if profile}
				<div
					class="flex items-center gap-2 rounded-full border border-neutral-200/90 bg-white/90 py-1.5 pr-3 pl-3 text-xs shadow-2xs"
				>
					<Building2 class="h-3.5 w-3.5 text-neutral-600" />
					<span class="font-medium text-neutral-900">{profile.name}</span>
					<span class="font-mono text-neutral-400">·</span>
					<span class="font-mono font-semibold text-neutral-700">{profile.society_code}</span>
				</div>
			{:else}
				<span class="text-xs font-medium text-neutral-500">Society Workspace</span>
			{/if}
		</div>

		<!-- Right: Notifications & Profile Menu -->
		<div class="flex items-center gap-3 sm:gap-4">
			<NotificationCenter societyId={profile?.id} />
			<ProfileMenu />
		</div>
	</div>
</header>
