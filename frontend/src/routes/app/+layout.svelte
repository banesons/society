<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import { ROUTES, appRoutes } from '$lib/routing';
	import AppShellHeader from '$lib/components/nav/app-shell-header.svelte';
	import NavigationRail from '$lib/components/nav/navigation-rail.svelte';
	import Card from '$lib/components/ui/card.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import { Loader2, AlertTriangle, RefreshCw } from '@lucide/svelte';

	let { children }: { children: Snippet } = $props();

	let isLoginPage = $derived(page.url.pathname === '/app/login' || page.url.pathname === '/login');
	let isSocietyRoute = $derived(page.url.pathname.startsWith('/app/societies/'));

	$effect(() => {
		// Public login flow does not require auth guard redirection
		if (isLoginPage) return;

		// When authentication check finishes and user is not authenticated, redirect to /app/login with returnTo
		// Do not redirect on temporary network drop; allow user to retry
		if (!auth.isLoading && !auth.isAuthenticated && !auth.networkError) {
			const destination = page.url.pathname + page.url.search + page.url.hash;
			const targetUrl = appRoutes.login(
				destination !== ROUTES.DASHBOARD && destination !== ROUTES.APP ? destination : undefined
			);
			goto(targetUrl, { replaceState: true });
		}
	});
</script>

<svelte:head>
	<!-- Authenticated application pages must not be indexed by public search engines -->
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div
	class="flex min-h-screen flex-col bg-[#f3f3f1] font-sans text-neutral-900 selection:bg-neutral-900 selection:text-white"
>
	{#if isLoginPage}
		<!-- Standalone Auth Flow (Login Page) -->
		{@render children()}
	{:else if auth.isLoading}
		<!-- Authenticated Shell Loading Skeleton (prevents flash of unauthenticated or unauthorized content) -->
		<div class="flex min-h-screen flex-col">
			<AppShellHeader />
			<main
				class="mx-auto flex w-full max-w-(--breakpoint-2xl) flex-1 items-center justify-center p-6 sm:p-12"
			>
				<Card
					class="flex flex-col items-center justify-center border-neutral-200 bg-white/90 p-12 text-center shadow-xs"
				>
					<Loader2 class="h-8 w-8 animate-spin text-neutral-900" />
					<p class="mt-4 font-serif text-sm font-medium text-neutral-950">
						Restoring application session...
					</p>
					<p class="mt-1 text-xs text-neutral-500">
						Verifying identity credentials and active tenant context
					</p>
				</Card>
			</main>
		</div>
	{:else if auth.networkError && !auth.isAuthenticated}
		<!-- Resilient Offline / Network Unavailable Fallback (Preserves Session) -->
		<div class="flex min-h-screen flex-col">
			<AppShellHeader />
			<main
				class="mx-auto flex w-full max-w-(--breakpoint-2xl) flex-1 items-center justify-center p-6 sm:p-12"
			>
				<Card
					class="flex max-w-md flex-col items-center justify-center border-neutral-200 bg-white/90 p-8 text-center shadow-xs"
				>
					<div
						class="mb-3 flex h-12 w-12 items-center justify-center rounded-2xl bg-amber-50 text-amber-600"
					>
						<AlertTriangle class="h-6 w-6" />
					</div>
					<h2 class="font-serif text-base font-semibold text-neutral-950">
						Network Unavailable
					</h2>
					<p class="mt-1.5 text-xs leading-relaxed text-neutral-500">
						Unable to verify your session due to a network connection issue. Your saved session remains intact.
					</p>
					<Button
						variant="primary"
						size="sm"
						class="mt-5"
						onclick={() => auth.init()}
					>
						<RefreshCw class="mr-1.5 h-3.5 w-3.5" />
						<span>Retry Connection</span>
					</Button>
				</Card>
			</main>
		</div>
	{:else if !auth.isAuthenticated}
		<!-- Redirecting state fallback -->
		<div class="flex min-h-screen flex-col">
			<AppShellHeader />
			<main
				class="mx-auto flex w-full max-w-(--breakpoint-2xl) flex-1 items-center justify-center p-6 sm:p-12"
			>
				<Card
					class="flex flex-col items-center justify-center border-neutral-200 bg-white/90 p-12 text-center shadow-xs"
				>
					<Loader2 class="h-8 w-8 animate-spin text-neutral-900" />
					<p class="mt-4 font-serif text-sm font-medium text-neutral-950">
						Redirecting to secure login...
					</p>
				</Card>
			</main>
		</div>
	{:else}
		<!-- Fully Authenticated Application Shell with Navigation Rail -->
		<div class="flex min-h-screen min-w-0 flex-1 overflow-x-hidden">
			<!-- Dual-mode Navigation Rail (Desktop Side Rail & Mobile Bottom Rail) -->
			<NavigationRail />

			<div class="flex min-h-screen min-w-0 flex-1 flex-col overflow-x-hidden">
				{#if !isSocietyRoute}
					<AppShellHeader />
				{/if}

				<main
					class={isSocietyRoute
						? 'flex min-w-0 w-full flex-1 flex-col overflow-x-hidden pb-20 md:pb-0'
						: 'mx-auto min-w-0 w-full max-w-(--breakpoint-2xl) flex-1 overflow-x-hidden px-4 py-8 pb-20 sm:px-10 md:pb-8 lg:px-16'}
				>
					{@render children()}
				</main>
			</div>
		</div>
	{/if}
</div>
