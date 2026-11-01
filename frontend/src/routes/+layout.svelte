<script lang="ts">
	import './layout.css';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	import type { Snippet } from 'svelte';
	import { auth } from '$lib/auth.svelte';
	import { ROUTES } from '$lib/routing';
	import Button from '$lib/components/ui/button.svelte';
	import Logo from '$lib/components/ui/logo.svelte';
	import ToastContainer from '$lib/components/ui/toast-container.svelte';
	import AppShellHeader from '$lib/components/nav/app-shell-header.svelte';
	import ProfileMenu from '$lib/components/nav/profile-menu.svelte';
	import { Terminal, Menu, X, ArrowRight } from '@lucide/svelte';

	import { initPerformance } from '$lib/firebase';
	import { pwa } from '$lib/pwa.svelte';

	let { children }: { children: Snippet } = $props();

	let isMobileMenuOpen = $state(false);

	let isLoginPage = $derived(page.url.pathname === '/app/login' || page.url.pathname === '/login');
	let isSocietyRoute = $derived(
		page.url.pathname.startsWith('/app/societies/') ||
			page.url.pathname.startsWith('/society/') ||
			page.url.pathname.startsWith('/societies/')
	);
	let isAppRoute = $derived(
		page.url.pathname.startsWith('/app') || page.url.pathname.startsWith('/admin')
	);
	let isPublicMarketingPage = $derived(!isLoginPage && !isSocietyRoute && !isAppRoute);

	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				retry: 1,
				refetchOnWindowFocus: false,
				staleTime: 5000
			}
		}
	});

	onMount(() => {
		auth.init();
		pwa.init();
		initPerformance();
	});

	const navMenus = [
		{ name: 'societies', href: '/#societies' },
		{ name: 'governance', href: '/#governance' },
		{ name: 'members', href: '/#members' },
		{ name: 'gatekeeping', href: '/#gatekeeping' },
		{ name: 'compliance', href: '/#compliance' },
		{ name: 'about', href: '/#about' }
	];

	function closeMobileMenu() {
		isMobileMenuOpen = false;
	}
</script>

<svelte:head>
	<title>chs. — Cooperative Housing Society Governance</title>
	<meta
		name="description"
		content="Cooperative Housing Society platform universal identity, administration, and gatekeeping"
	/>
</svelte:head>

<QueryClientProvider client={queryClient}>
	<div
		class="flex min-h-screen flex-col bg-[#f3f3f1] font-sans text-neutral-900 selection:bg-neutral-900 selection:text-white"
	>
		<!-- 1. Dedicated Authenticated Application Shell Header (only for legacy /admin) -->
		{#if page.url.pathname === '/admin'}
			<AppShellHeader />

			<!-- 2. Public Marketing Header (only on public marketing pages like /) -->
		{:else if isPublicMarketingPage}
			<header
				class="sticky top-0 z-40 border-b border-neutral-200/70 bg-[#f3f3f1]/90 backdrop-blur-md"
			>
				<div
					class="mx-auto flex h-20 max-w-(--breakpoint-2xl) items-center justify-between px-6 sm:px-10 lg:px-16"
				>
					<!-- Brand Logo -->
					<Logo size="md" />

					<!-- Desktop Marketing Domain Menus -->
					<nav
						class="hidden items-center gap-8 text-sm font-normal text-neutral-800 md:flex lg:gap-10"
					>
						{#each navMenus as item (item.name)}
							<a
								href={item.href}
								class="text-neutral-700 lowercase transition-colors hover:text-neutral-950 hover:opacity-100"
							>
								{item.name}
							</a>
						{/each}
					</nav>

					<!-- Right Side: Diagnostics, Dashboard Link or Login -->
					<div class="flex items-center gap-4 sm:gap-6">
						<a
							href={ROUTES.DEVELOPER}
							class="hidden items-center gap-1 rounded-full px-2.5 py-1 text-xs text-neutral-500 hover:bg-neutral-200/70 hover:text-neutral-900 lg:inline-flex"
							title="Developer Diagnostics"
						>
							<Terminal class="h-3 w-3" />
							<span>dev</span>
						</a>

						{#if auth.isAuthenticated && auth.user}
							<div class="flex items-center gap-3">
								<a href={ROUTES.APP} class="hidden sm:inline-block">
									<Button variant="primary" size="sm">
										<span>Dashboard</span>
										<ArrowRight class="ml-1 h-3.5 w-3.5" />
									</Button>
								</a>
								<ProfileMenu />
							</div>
						{:else}
							<a
								href={ROUTES.LOGIN}
								class="editorial-link text-sm font-medium tracking-tight text-neutral-950 transition-opacity hover:opacity-75 sm:text-base"
							>
								Login
							</a>
						{/if}

						<!-- Mobile Menu Hamburger Button -->
						<button
							type="button"
							onclick={() => (isMobileMenuOpen = !isMobileMenuOpen)}
							class="flex h-10 w-10 items-center justify-center rounded-full text-neutral-900 transition-colors hover:bg-neutral-200/60 md:hidden"
							aria-label="Toggle navigation menu"
							aria-expanded={isMobileMenuOpen}
						>
							{#if isMobileMenuOpen}
								<X class="h-5 w-5" />
							{:else}
								<Menu class="h-5 w-5" />
							{/if}
						</button>
					</div>
				</div>

				<!-- Mobile Adaptive Dropdown / Drawer for Public Website -->
				{#if isMobileMenuOpen}
					<div
						class="animate-in fade-in slide-in-from-top-2 border-b border-neutral-200 bg-[#f3f3f1] px-6 py-6 duration-150 md:hidden"
					>
						<nav class="flex flex-col space-y-4 text-base font-medium">
							{#each navMenus as item (item.name)}
								<a
									href={item.href}
									onclick={closeMobileMenu}
									class="text-lg text-neutral-800 lowercase transition-colors hover:text-neutral-950"
								>
									{item.name}
								</a>
							{/each}

							<div class="space-y-3 border-t border-neutral-200 pt-4">
								<a
									href={ROUTES.DEVELOPER}
									onclick={closeMobileMenu}
									class="flex items-center gap-2 text-sm text-neutral-600 hover:text-neutral-950"
								>
									<Terminal class="h-4 w-4" />
									<span>Developer Diagnostics</span>
								</a>

								{#if auth.isAuthenticated && auth.user}
									<div class="pt-2">
										<a
											href={ROUTES.APP}
											onclick={closeMobileMenu}
											class="inline-block w-full rounded-full bg-neutral-950 px-5 py-2.5 text-center text-sm font-medium text-white shadow-xs"
										>
											Go to Dashboard
										</a>
									</div>
								{:else}
									<div class="pt-2">
										<a
											href={ROUTES.LOGIN}
											onclick={closeMobileMenu}
											class="inline-block w-full rounded-full bg-neutral-950 px-5 py-2.5 text-center text-sm font-medium text-white shadow-xs"
										>
											Sign In
										</a>
									</div>
								{/if}
							</div>
						</nav>
					</div>
				{/if}
			</header>
		{/if}

		<!-- Main Workspace Area -->
		<main
			class={isLoginPage || isSocietyRoute || isAppRoute
				? 'flex flex-1 flex-col'
				: 'mx-auto w-full max-w-(--breakpoint-2xl) flex-1 px-6 py-8 sm:px-10 lg:px-16'}
		>
			{@render children()}
		</main>

		<!-- Public Marketing Footer (Rendered ONLY on public marketing pages) -->
		{#if isPublicMarketingPage}
			<footer
				class="border-t border-neutral-200 bg-[#ecece9] py-12 text-xs text-neutral-600 transition-colors"
			>
				<div class="mx-auto max-w-(--breakpoint-2xl) px-6 sm:px-10 lg:px-16">
					<div class="flex flex-col gap-8 md:flex-row md:items-center md:justify-between">
						<div>
							<Logo size="sm" />
							<p class="mt-2 max-w-sm text-xs leading-relaxed text-neutral-500">
								Verifiable, phone-first residential governance for Cooperative Housing Societies.
							</p>
						</div>

						<div class="flex flex-wrap items-center gap-6 text-xs text-neutral-600">
							<a href="/#governance" class="hover:text-neutral-950">governance</a>
							<a href="/#societies" class="hover:text-neutral-950">societies</a>
							<a href="/#compliance" class="hover:text-neutral-950">compliance</a>
							<a
								href={ROUTES.DEVELOPER}
								class="underline underline-offset-4 hover:text-neutral-950"
							>
								diagnostics
							</a>
							{#if !auth.isAuthenticated}
								<a href={ROUTES.LOGIN} class="underline underline-offset-4 hover:text-neutral-950">
									sign in
								</a>
							{/if}
						</div>
					</div>

					<div
						class="mt-8 flex flex-col justify-between border-t border-neutral-300/60 pt-6 text-[11px] text-neutral-500 sm:flex-row"
					>
						<div>
							&copy; {new Date().getFullYear()} CHS Platform. All rights reserved.
						</div>
					</div>
				</div>
			</footer>
		{/if}

		<!-- Central Toast Notification Viewport Overlay -->
		<ToastContainer />
	</div>
</QueryClientProvider>
