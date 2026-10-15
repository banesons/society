<script lang="ts">
	import './layout.css';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	import type { Snippet } from 'svelte';
	import { auth } from '$lib/auth.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import Logo from '$lib/components/ui/logo.svelte';
	import LogoutAllDialog from '$lib/components/auth/logout-all-dialog.svelte';
	import SocietySwitcher from '$lib/components/nav/society-switcher.svelte';
	import { Shield, User, Terminal, Info, X, Menu } from '@lucide/svelte';

	import { initPerformance } from '$lib/firebase';

	let { children }: { children: Snippet } = $props();

	let isLogoutAllOpen = $state(false);
	let isMobileMenuOpen = $state(false);
	let isLoginPage = $derived(page.url.pathname === '/login');
	let isSocietyRoute = $derived(
		page.url.pathname.startsWith('/society/') || page.url.pathname.startsWith('/societies/')
	);

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
		<!-- Editorial Header (Public Marketing & Platform Admin only) -->
		{#if !isLoginPage && !isSocietyRoute}
			<header
				class="sticky top-0 z-40 border-b border-neutral-200/70 bg-[#f3f3f1]/90 backdrop-blur-md"
			>
				<div
					class="mx-auto flex h-20 max-w-(--breakpoint-2xl) items-center justify-between px-6 sm:px-10 lg:px-16"
				>
					<!-- Brand Logo -->
					<Logo size="md" />

					<!-- Desktop Random Menus in Navbar -->
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

					<!-- Right Side: Login & User State -->
					<div class="flex items-center gap-4 sm:gap-6">
						<a
							href="/developer"
							class="hidden items-center gap-1 rounded-full px-2.5 py-1 text-xs text-neutral-500 hover:bg-neutral-200/70 hover:text-neutral-900 lg:inline-flex"
							title="Developer Diagnostics"
						>
							<Terminal class="h-3 w-3" />
							<span>dev</span>
						</a>

						{#if auth.isAuthenticated && auth.user}
							<div class="hidden items-center gap-2.5 sm:flex">
								{#if auth.isSuperAdmin}
									<a href="/admin">
										<Button
											variant="outline"
											size="sm"
											class="border-neutral-300 bg-white text-xs hover:bg-neutral-100"
										>
											<Shield class="mr-1 h-3.5 w-3.5 text-neutral-800" />
											<span>Platform Admin</span>
										</Button>
									</a>
								{/if}
								<SocietySwitcher />
								<Badge variant="outline" class="flex items-center gap-1.5 py-1 text-xs">
									<User class="h-3 w-3 text-neutral-600" />
									<span>{auth.user.phone}</span>
								</Badge>
								<button
									type="button"
									onclick={() => auth.logout()}
									class="cursor-pointer text-xs text-neutral-600 underline underline-offset-4 transition-colors hover:text-neutral-950"
								>
									logout
								</button>
								<button
									type="button"
									onclick={() => (isLogoutAllOpen = true)}
									class="cursor-pointer text-xs text-rose-600 underline underline-offset-4 transition-colors hover:text-rose-800"
									title="Log out of all devices"
								>
									logout all
								</button>
							</div>
						{:else}
							<a
								href="/login"
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

				<!-- Mobile Adaptive Dropdown / Drawer -->
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
									href="/developer"
									onclick={closeMobileMenu}
									class="flex items-center gap-2 text-sm text-neutral-600 hover:text-neutral-950"
								>
									<Terminal class="h-4 w-4" />
									<span>Developer Diagnostics</span>
								</a>

								{#if auth.isAuthenticated && auth.user}
									<div class="space-y-3 pt-2">
										{#if auth.isSuperAdmin}
											<a
												href="/admin"
												onclick={closeMobileMenu}
												class="flex items-center gap-2 text-sm font-medium text-neutral-900"
											>
												<Shield class="h-4 w-4" />
												<span>Super Admin Console</span>
											</a>
										{/if}
										<div class="flex items-center justify-between text-xs text-neutral-600">
											<span>Signed in as {auth.user.phone}</span>
											<button
												type="button"
												onclick={() => {
													auth.logout();
													closeMobileMenu();
												}}
												class="text-neutral-950 underline underline-offset-4"
											>
												Logout
											</button>
										</div>
									</div>
								{:else}
									<div class="pt-2">
										<a
											href="/login"
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

		<!-- Session Replaced Alert -->
		{#if auth.sessionReplaced}
			<div
				role="status"
				class="border-b border-amber-200 bg-amber-50 px-4 py-2.5 text-xs text-amber-900"
			>
				<div class="mx-auto flex max-w-7xl items-center justify-between gap-3">
					<div class="flex items-center gap-2">
						<Info class="h-4 w-4 shrink-0 text-amber-600" />
						<span>
							Signed in successfully. Your oldest session was signed out to keep your account
							limited to two active devices.
						</span>
					</div>
					<button
						type="button"
						onclick={() => auth.dismissSessionReplacedNotice()}
						class="rounded p-1 text-amber-700 transition-colors hover:bg-amber-100"
						aria-label="Dismiss notice"
					>
						<X class="h-3.5 w-3.5" />
					</button>
				</div>
			</div>
		{/if}

		<!-- Main Workspace Area -->
		<main
			class={isLoginPage || isSocietyRoute
				? 'flex flex-1 flex-col'
				: 'mx-auto w-full max-w-(--breakpoint-2xl) flex-1 px-6 py-10 sm:px-10 lg:px-16'}
		>
			{@render children()}
		</main>

		<!-- Minimalist Editorial Footer -->
		{#if !isLoginPage && !isSocietyRoute}
			<footer
				class="border-t border-neutral-200 bg-[#ecece9] py-12 text-xs text-neutral-600 transition-colors"
			>
				<div class="mx-auto max-w-(--breakpoint-2xl) px-6 sm:px-10 lg:px-16">
					<div class="flex flex-col gap-8 md:flex-row md:items-center md:justify-between">
						<div>
							<Logo size="sm" />
							<p class="mt-2 max-w-sm text-xs leading-relaxed text-neutral-500">
								Next-generation platform governance for Cooperative Housing Societies. Built with
								cryptographic trust, immutable audits, and digital gatekeeping.
							</p>
						</div>

						<div class="flex flex-wrap items-center gap-6 text-xs text-neutral-600">
							<a href="/#governance" class="hover:text-neutral-950">governance</a>
							<a href="/#societies" class="hover:text-neutral-950">societies</a>
							<a href="/#compliance" class="hover:text-neutral-950">compliance</a>
							<a href="/developer" class="underline underline-offset-4 hover:text-neutral-950">
								diagnostics
							</a>
							{#if !auth.isAuthenticated}
								<a href="/login" class="underline underline-offset-4 hover:text-neutral-950">
									sign in
								</a>
							{/if}
						</div>
					</div>

					<div
						class="mt-8 flex flex-col justify-between border-t border-neutral-300/60 pt-6 text-[11px] text-neutral-500 sm:flex-row"
					>
						<div>
							&copy; {new Date().getFullYear()} CHS Platform. Documentation-first engineering.
						</div>
						<div class="mt-2 sm:mt-0">Two-device strict session limit enabled.</div>
					</div>
				</div>
			</footer>
		{/if}

		<LogoutAllDialog isOpen={isLogoutAllOpen} onClose={() => (isLogoutAllOpen = false)} />
	</div>
</QueryClientProvider>
