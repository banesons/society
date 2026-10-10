<script lang="ts">
	import './layout.css';
	import { onMount } from 'svelte';
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	import type { Snippet } from 'svelte';
	import { auth } from '$lib/auth.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import LoginDialog from '$lib/components/auth/login-dialog.svelte';
	import { LogIn, LogOut, Shield, User } from '@lucide/svelte';

	let { children }: { children: Snippet } = $props();



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
	<title>CHS Platform — S01 Identity & Authentication</title>
	<meta name="description" content="Cooperative Housing Society platform universal identity and authentication" />
</svelte:head>

<QueryClientProvider client={queryClient}>
	<div
		class="flex min-h-screen flex-col bg-slate-950 font-sans text-slate-50 selection:bg-indigo-500/30 selection:text-indigo-200"
	>
		<header class="sticky top-0 z-40 border-b border-slate-800/80 bg-slate-900/60 backdrop-blur-md">
			<div class="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
				<div class="flex items-center gap-3">
					<div
						class="flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-tr from-indigo-600 to-violet-500 text-sm font-bold tracking-wider text-white shadow-lg shadow-indigo-500/20"
					>
						CHS
					</div>
					<div>
						<div class="text-sm font-semibold tracking-tight text-slate-100">CHS Platform</div>
						<div class="font-mono text-[10px] text-slate-400">S01 · Identity & Auth</div>
					</div>
				</div>

				<div class="flex items-center gap-4 text-xs font-medium">
					{#if auth.isAuthenticated && auth.user}
						<div class="flex items-center gap-2">
							<Badge variant="success" class="flex items-center gap-1.5 py-1">
								<User class="h-3 w-3" />
								<span>{auth.user.phone}</span>
							</Badge>
							<Button variant="ghost" size="sm" onclick={() => auth.logout()}>
								<LogOut class="h-3.5 w-3.5 mr-1" />
								<span>Logout</span>
							</Button>
							<Button variant="outline" size="sm" onclick={() => auth.logoutAll()}>
								<Shield class="h-3.5 w-3.5 mr-1 text-rose-400" />
								<span class="text-rose-400">Logout All</span>
							</Button>
						</div>
					{:else}
						<Button variant="primary" size="sm" onclick={() => auth.openLogin()}>
							<LogIn class="h-3.5 w-3.5 mr-1" />
							<span>Sign In</span>
						</Button>
					{/if}
				</div>
			</div>
		</header>

		<main class="mx-auto w-full max-w-7xl flex-1 px-4 py-8 sm:px-6 lg:px-8">
			{@render children()}
		</main>

		<footer
			class="border-t border-slate-800/80 bg-slate-950 py-6 text-center text-xs text-slate-500"
		>
			Cooperative Housing Society (CHS) · Identity & Authentication Layer · Documentation-First Development
		</footer>

		<LoginDialog isOpen={auth.isLoginOpen} onClose={() => auth.closeLogin()} />
	</div>
</QueryClientProvider>
