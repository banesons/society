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
	<title>Access Denied — chs.</title>
	<meta name="description" content="Account access restrictions and unauthorized role state" />
</svelte:head>

<div class="flex min-h-[75vh] items-center justify-center px-4 py-12">
	<div class="w-full max-w-lg space-y-6">
		<Card class="border-neutral-200 bg-white p-8 text-center shadow-lg sm:p-10">
			<div
				class="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-amber-50 text-amber-700"
			>
				<ShieldAlert class="h-7 w-7" />
			</div>

			<div class="mt-4 inline-flex items-center gap-1.5">
				<Badge variant="warning">Access Denied</Badge>
				<span class="text-xs text-neutral-500">Platform Layer (S02)</span>
			</div>

			<h1 class="mt-3 text-2xl font-bold tracking-tight text-neutral-950">
				No Platform Administrative Role
			</h1>

			{#if auth.isAuthenticated && auth.user}
				<div
					class="mt-4 rounded-xl border border-neutral-200 bg-neutral-50 p-4 text-left text-xs text-neutral-700"
				>
					<div class="flex items-center gap-2 font-mono text-neutral-900">
						<User class="h-3.5 w-3.5 text-neutral-500" />
						<span>Verified Identity: <strong>{auth.user.phone}</strong></span>
					</div>
					<div class="mt-1 text-[11px] text-neutral-500">
						Account status: <span class="font-medium text-emerald-700">ACTIVE</span> · Platform
						roles:
						<span class="font-medium text-neutral-900 italic"
							>{auth.user.platform_roles?.length
								? auth.user.platform_roles.join(', ')
								: 'None'}</span
						>
					</div>
				</div>

				<p class="mt-4 text-xs leading-relaxed text-neutral-600">
					Your mobile identity has been authenticated, but you do not hold an active <code
						class="rounded bg-neutral-100 px-1.5 py-0.5 font-mono text-neutral-900"
						>SUPER_ADMIN</code
					> platform role. Platform management rosters and audit logs are restricted.
				</p>
			{:else}
				<p class="mt-4 text-xs leading-relaxed text-neutral-600">
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
				class="mt-6 flex items-center justify-center gap-2 border-t border-neutral-100 pt-6 text-[11px] text-neutral-500"
			>
				<span>Need to test with dev numbers?</span>
				<a
					href="/developer"
					class="inline-flex items-center gap-1 font-medium text-neutral-950 underline underline-offset-4 hover:opacity-75"
				>
					<Terminal class="h-3 w-3" />
					<span>Developer Diagnostics</span>
				</a>
			</div>
		</Card>
	</div>
</div>
