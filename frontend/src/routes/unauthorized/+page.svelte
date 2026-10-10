<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import Card from '$lib/components/ui/card.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { ShieldAlert, ArrowLeft, LogOut, Terminal, User } from '@lucide/svelte';

	async function handleLogout() {
		await auth.logout();
		goto('/login');
	}

	$effect(() => {
		if (auth.isAuthenticated && auth.isSuperAdmin) {
			goto('/admin');
		}
	});
</script>

<svelte:head>
	<title>Access Denied — CHS Platform</title>
	<meta name="description" content="Account access restrictions and unauthorized role state" />
</svelte:head>

<div class="flex min-h-[75vh] items-center justify-center px-4 py-10">
	<div class="w-full max-w-lg space-y-6">
		<Card class="border-amber-500/30 bg-slate-900/90 p-8 text-center shadow-2xl sm:p-10">
			<div
				class="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-amber-500/10 text-amber-400 shadow-inner"
			>
				<ShieldAlert class="h-7 w-7" />
			</div>

			<div class="mt-4 inline-flex items-center gap-1.5">
				<Badge variant="warning">Access Denied</Badge>
				<span class="text-xs text-slate-400">Platform Layer (S02)</span>
			</div>

			<h1 class="mt-3 text-xl font-bold tracking-tight text-white sm:text-2xl">
				No Platform Administrative Role
			</h1>

			{#if auth.isAuthenticated && auth.user}
				<div
					class="mt-4 rounded-xl border border-slate-800 bg-slate-950/70 p-3.5 text-xs text-slate-300"
				>
					<div class="flex items-center justify-center gap-2 font-mono text-slate-200">
						<User class="h-3.5 w-3.5 text-slate-400" />
						<span>Verified Identity: <strong>{auth.user.phone}</strong></span>
					</div>
					<div class="mt-1 text-[11px] text-slate-400">
						Account status: <span class="font-medium text-emerald-400">ACTIVE</span> · Platform
						roles:
						<span class="italic"
							>{auth.user.platform_roles?.length
								? auth.user.platform_roles.join(', ')
								: 'None'}</span
						>
					</div>
				</div>

				<p class="mt-4 text-xs leading-relaxed text-slate-400">
					Your mobile identity has been authenticated, but you do not hold an active <code
						class="text-indigo-400">SUPER_ADMIN</code
					> platform role. Platform management rosters and audit logs are restricted.
				</p>
			{:else}
				<p class="mt-4 text-xs leading-relaxed text-slate-400">
					You are not signed in. Please sign in with an authorized Super Admin account to access
					administrative features.
				</p>
			{/if}

			<div class="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
				{#if auth.isSuperAdmin}
					<a href="/admin" class="w-full sm:w-auto">
						<Button variant="primary" size="md" class="w-full">
							<span>Go to Super Admin Console</span>
						</Button>
					</a>
				{/if}

				<a href="/" class="w-full sm:w-auto">
					<Button variant="outline" size="md" class="w-full">
						<ArrowLeft class="mr-1.5 h-4 w-4" />
						<span>Return to Home</span>
					</Button>
				</a>

				{#if auth.isAuthenticated}
					<Button variant="secondary" size="md" onclick={handleLogout} class="w-full sm:w-auto">
						<LogOut class="mr-1.5 h-4 w-4" />
						<span>Switch Account</span>
					</Button>
				{:else}
					<a href="/login" class="w-full sm:w-auto">
						<Button variant="primary" size="md" class="w-full">
							<span>Go to Sign In</span>
						</Button>
					</a>
				{/if}
			</div>

			<div
				class="mt-6 flex items-center justify-center gap-2 border-t border-slate-800/80 pt-6 text-[11px] text-slate-500"
			>
				<span>Need to test with dev numbers?</span>
				<a
					href="/developer"
					class="inline-flex items-center gap-1 font-medium text-indigo-400 hover:text-indigo-300"
				>
					<Terminal class="h-3 w-3" />
					<span>Developer Diagnostics</span>
				</a>
			</div>
		</Card>
	</div>
</div>
