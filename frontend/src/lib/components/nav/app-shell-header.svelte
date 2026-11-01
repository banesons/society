<script lang="ts">
	import Logo from '$lib/components/ui/logo.svelte';
	import NotificationCenter from '$lib/components/notifications/notification-center.svelte';
	import ProfileMenu from '$lib/components/nav/profile-menu.svelte';
	import { auth } from '$lib/auth.svelte';
	import { sidebar } from '$lib/sidebar.svelte';
	import { PanelLeftClose, PanelLeftOpen } from '@lucide/svelte';

	interface Props {
		societyId?: string;
	}

	let { societyId }: Props = $props();
</script>

<!-- Dedicated Authenticated Application Shell Header -->
<header class="sticky top-0 z-40 border-b border-neutral-200/80 bg-[#f3f3f1] transition-colors">
	<div
		class="mx-auto flex h-16 max-w-(--breakpoint-2xl) items-center justify-between px-4 sm:h-20 sm:px-10 lg:px-16"
	>
		<!-- Left: Platform Branding on Mobile, Sidebar Toggle & Context on Desktop (No duplicate logo) -->
		<div class="flex items-center gap-3 sm:gap-4">
			<!-- Mobile Only: Platform Logo (< 768px where sidebar is hidden) -->
			<div class="md:hidden">
				<Logo size="md" />
			</div>

			<!-- Desktop Only: Sidebar Collapse/Expand Toggle Button (>= 768px) -->
			<button
				type="button"
				onclick={() => sidebar.toggle()}
				class="hidden h-9 w-9 items-center justify-center rounded-xl text-neutral-600 transition-colors hover:bg-neutral-200/70 hover:text-neutral-950 md:flex"
				title={sidebar.isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
				aria-label={sidebar.isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
			>
				{#if sidebar.isCollapsed}
					<PanelLeftOpen class="h-4 w-4" />
				{:else}
					<PanelLeftClose class="h-4 w-4" />
				{/if}
			</button>

			<span class="hidden h-4 w-px bg-neutral-300 md:inline-block" aria-hidden="true"></span>
			<span
				class="hidden text-xs font-medium tracking-tight text-neutral-500 lowercase md:inline-block"
			>
				{#if auth.isSuperAdmin}
					platform governance
				{:else if auth.isPlatformAdmin}
					administration
				{:else}
					workspace
				{/if}
			</span>
		</div>

		<!-- Right: Strictly Persistent Interactive Controls (Notifications & Profile Menu) -->
		<div class="flex items-center gap-2.5 sm:gap-4">
			<NotificationCenter {societyId} />
			<ProfileMenu />
		</div>
	</div>
</header>
