<script lang="ts">
	import './layout.css';
	import { onMount } from 'svelte';
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	import type { Snippet } from 'svelte';
	import { auth } from '$lib/auth.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import LogoutAllDialog from '$lib/components/auth/logout-all-dialog.svelte';
	import SocietySwitcher from '$lib/components/nav/society-switcher.svelte';
	import { LogIn, LogOut, Shield, User, Terminal, Info, X } from '@lucide/svelte';

	let { children }: { children: Snippet } = $props();

	let isLogoutAllOpen = $state(false);

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
	});
</script>

<svelte:head>
	<title>CHS Platform — Cooperative Housing Society Management</title>
	<meta
		name="description"
		content="Cooperative Housing Society platform universal identity and authentication"
	/>
</svelte:head>

<QueryClientProvider client={queryClient}>
	<div
		class="flex min-h-screen flex-col bg-slate-950 font-sans text-slate-50 selection:bg-indigo-500/30 selection:text-indigo-200"
	>
		<header class="sticky top-0 z-40 border-b border-slate-800/80 bg-slate-900/60 backdrop-blur-md">
			<div class="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
				<a href="/" class="flex items-center gap-3 transition-opacity hover:opacity-90">
					<div
						class="flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-tr from-indigo-600 to-violet-500 text-sm font-bold tracking-wider text-white shadow-lg shadow-indigo-500/20"
					>
						CHS
					</div>
					<div>
						<div class="text-sm font-semibold tracking-tight text-slate-100">CHS Platform</div>
						<div class="font-mono text-[10px] text-slate-400">Housing Society Governance</div>
					</div>
				</a>

				<div class="flex items-center gap-3 text-xs font-medium">
					<a
						href="/developer"
						class="hidden items-center gap-1 rounded-lg px-2 py-1 text-slate-400 hover:bg-slate-800/50 hover:text-slate-200 sm:inline-flex"
					>
						<Terminal class="h-3.5 w-3.5" />
						<span>Dev</span>
					</a>

					{#if auth.isAuthenticated && auth.user}
						<div class="flex items-center gap-2">
							{#if auth.isSuperAdmin}
								<a href="/admin">
									<Button
										variant="outline"
										size="sm"
										class="border-indigo-500/40 text-indigo-300 hover:bg-indigo-500/10"
									>
										<Shield class="mr-1 h-3.5 w-3.5 text-indigo-400" />
										<span>Platform Admin</span>
									</Button>
								</a>
							{/if}
							<SocietySwitcher />
							<Badge variant="success" class="flex items-center gap-1.5 py-1">
								<User class="h-3 w-3" />
								<span>{auth.user.phone}</span>
							</Badge>
							<Button variant="ghost" size="sm" onclick={() => auth.logout()}>
								<LogOut class="mr-1 h-3.5 w-3.5" />
								<span>Logout</span>
							</Button>
							<Button variant="outline" size="sm" onclick={() => (isLogoutAllOpen = true)}>
								<Shield class="mr-1 h-3.5 w-3.5 text-rose-400" />
								<span class="text-rose-400">Logout All</span>
							</Button>
						</div>
					{:else}
						<a href="/login">
							<Button variant="primary" size="sm">
								<LogIn class="mr-1 h-3.5 w-3.5" />
								<span>Sign In</span>
							</Button>
						</a>
					{/if}
				</div>
			</div>
		</header>

		{#if auth.sessionReplaced}
			<div
				role="status"
				class="border-b border-amber-500/30 bg-amber-500/10 px-4 py-2 text-xs text-amber-200"
			>
				<div class="mx-auto flex max-w-7xl items-center justify-between gap-3">
					<div class="flex items-center gap-2">
						<Info class="h-4 w-4 shrink-0 text-amber-400" />
						<span>
							Signed in successfully. Your oldest session was signed out to keep your account
							limited to two active devices.
						</span>
					</div>
					<button
						type="button"
						onclick={() => auth.dismissSessionReplacedNotice()}
						class="rounded p-1 text-amber-400 transition-colors hover:bg-amber-500/20 hover:text-amber-200"
						aria-label="Dismiss notice"
					>
						<X class="h-3.5 w-3.5" />
					</button>
				</div>
			</div>
		{/if}

		<main class="mx-auto w-full max-w-7xl flex-1 px-4 py-8 sm:px-6 lg:px-8">
			{@render children()}
		</main>

		<footer
			class="border-t border-slate-800/80 bg-slate-950 py-6 text-center text-xs text-slate-500"
		>
			<div class="flex items-center justify-center gap-4">
				<span>Cooperative Housing Society (CHS) · Documentation-First Development</span>
				<span>·</span>
				<a
					href="/developer"
					class="text-slate-400 underline underline-offset-2 hover:text-slate-300">Diagnostics</a
				>
			</div>
		</footer>

		<LogoutAllDialog isOpen={isLogoutAllOpen} onClose={() => (isLogoutAllOpen = false)} />
	</div>
</QueryClientProvider>
