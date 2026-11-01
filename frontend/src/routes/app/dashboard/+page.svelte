<script lang="ts">
	import { onMount } from 'svelte';
	import { auth } from '$lib/auth.svelte';
	import { formatFullPhone } from '$lib/phone';
	import { appRoutes } from '$lib/routing';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import PlatformDashboard from '$lib/components/dashboard/platform-dashboard.svelte';
	import LogoutAllDialog from '$lib/components/auth/logout-all-dialog.svelte';
	import Breadcrumbs from '$lib/components/nav/breadcrumbs.svelte';
	import { Building2, Shield, ArrowRight, Smartphone } from '@lucide/svelte';

	let isLogoutAllOpen = $state(false);

	onMount(() => {
		if (auth.isAuthenticated) {
			auth.fetchCurrentUser();
		}
	});

	let singleSociety = $derived(
		auth.user?.society_assignments && auth.user.society_assignments.length === 1
			? auth.user.society_assignments[0]
			: null
	);

	let multipleSocieties = $derived(
		auth.user?.society_assignments && auth.user.society_assignments.length > 1
			? auth.user.society_assignments
			: []
	);
</script>

<svelte:head>
	<title>Workspace Dashboard — CHS Platform</title>
	<meta name="description" content="CHS authenticated application workspace dashboard" />
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div class="space-y-6 min-w-0 w-full">
	<!-- Top Breadcrumbs -->
	<div>
		<Breadcrumbs />
	</div>

	<!-- Header Greeting & Context Banner -->
	<div
		class="flex flex-col gap-4 rounded-3xl border border-[#e2e2de] bg-white p-6 shadow-xs sm:flex-row sm:items-center sm:justify-between sm:p-8"
	>
		<div class="min-w-0 flex-1">
			<div class="flex flex-wrap items-center gap-2.5">
				<h1 class="font-serif text-2xl font-bold tracking-tight text-[#121212] sm:text-3xl">
					Workspace Dashboard
				</h1>
				{#if auth.isSuperAdmin}
					<Badge
						variant="outline"
						class="border-neutral-300 bg-neutral-100 font-mono text-[10px] text-neutral-800"
					>
						SUPER_ADMIN
					</Badge>
				{:else if auth.isPlatformAdmin}
					<Badge
						variant="outline"
						class="border-neutral-300 bg-neutral-100 font-mono text-[10px] text-neutral-800"
					>
						ADMIN
					</Badge>
				{/if}
			</div>
			<p class="mt-1.5 text-xs text-neutral-500 sm:text-sm">
				Signed in as
				<span class="font-mono font-medium text-neutral-800">
					{auth.user?.phone ? formatFullPhone(auth.user.phone) : 'Verified User'}
				</span>
				· Two active device sessions limit enabled
			</p>
		</div>

		<div class="flex flex-wrap items-center gap-3">
			{#if auth.isPlatformAdmin}
				<a href={appRoutes.admin}>
					<Button variant="primary" size="sm">
						<Shield class="mr-1.5 h-3.5 w-3.5" />
						<span>Platform Administration</span>
						<ArrowRight class="ml-1 h-3.5 w-3.5" />
					</Button>
				</a>
			{/if}
			<Button
				variant="outline"
				size="sm"
				onclick={() => (isLogoutAllOpen = true)}
				class="border-neutral-200 text-xs text-neutral-600 hover:text-neutral-900"
			>
				<Smartphone class="mr-1.5 h-3.5 w-3.5 text-neutral-500" />
				<span>Manage Sessions</span>
			</Button>
		</div>
	</div>

	<!-- Platform Overview (For Super Admin / Platform Admin) -->
	{#if auth.isPlatformAdmin}
		<div class="space-y-4 min-w-0">
			<div class="flex items-center justify-between">
				<div>
					<h2 class="font-serif text-base font-bold tracking-tight text-[#121212]">Platform Ecosystem</h2>
					<p class="text-xs text-neutral-500">
						High-level operational metrics across all registered cooperative housing societies
					</p>
				</div>
				<a
					href={appRoutes.admin}
					class="text-xs font-semibold text-neutral-800 underline underline-offset-4 hover:text-neutral-950"
				>
					View complete administration console &rarr;
				</a>
			</div>

			<PlatformDashboard />
		</div>
	{/if}

	<!-- Prominent Single Society View -->
	{#if singleSociety}
		<div class="space-y-4 min-w-0">
			<h2 class="font-serif text-base font-bold tracking-tight text-[#121212]">Active Society Workspace</h2>
			<div class="rounded-3xl border border-[#e2e2de] bg-white p-6 shadow-xs sm:p-8">
				<div class="flex flex-col gap-6 sm:flex-row sm:items-center sm:justify-between">
					<div class="flex items-start gap-4 min-w-0">
						<div
							class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-neutral-100 text-neutral-900 sm:h-14 sm:w-14"
						>
							<Building2 class="h-6 w-6 sm:h-7 sm:w-7" />
						</div>
						<div class="min-w-0">
							<div class="flex flex-wrap items-center gap-2">
								<h3 class="font-serif text-xl font-bold text-[#121212] truncate">
									{singleSociety.society_name}
								</h3>
								<Badge
									variant={singleSociety.status === 'ACTIVE' ? 'success' : 'warning'}
									class="text-[10px]"
								>
									{singleSociety.status}
								</Badge>
							</div>
							<div class="mt-1 flex items-center gap-2 font-mono text-xs text-neutral-500">
								<span class="rounded bg-neutral-100 px-1.5 py-0.5 font-bold text-neutral-900">
									{singleSociety.society_code}
								</span>
								<span>·</span>
								<span>Role: <strong class="text-neutral-900">{singleSociety.role}</strong></span>
							</div>
						</div>
					</div>

					<div class="flex items-center gap-3">
						{#if singleSociety.role === 'GATEKEEPER'}
							<a href={appRoutes.gate(singleSociety.society_id)}>
								<Button variant="primary" size="md">
									<span>Enter Gate Operations</span>
									<ArrowRight class="ml-1.5 h-4 w-4" />
								</Button>
							</a>
						{:else}
							<a href={appRoutes.society(singleSociety.society_id)}>
								<Button variant="primary" size="md">
									<span>Enter Society Workspace</span>
									<ArrowRight class="ml-1.5 h-4 w-4" />
								</Button>
							</a>
						{/if}
					</div>
				</div>

				<div class="mt-6 grid grid-cols-1 gap-3 border-t border-neutral-100 pt-6 sm:grid-cols-3">
					{#if singleSociety.role === 'GATEKEEPER'}
						<a
							href={`/app/societies/${singleSociety.society_id}/gate`}
							class="flex items-center justify-between rounded-xl border border-neutral-200/80 p-3 text-xs font-medium text-neutral-700 transition-colors hover:border-neutral-950 hover:bg-neutral-50"
						>
							<span>Gate Console & Duty</span>
							<ArrowRight class="h-3.5 w-3.5 text-neutral-400" />
						</a>
						<a
							href={`/app/societies/${singleSociety.society_id}/gate`}
							class="flex items-center justify-between rounded-xl border border-neutral-200/80 p-3 text-xs font-medium text-neutral-700 transition-colors hover:border-neutral-950 hover:bg-neutral-50"
						>
							<span>Active Visits & Check-out</span>
							<ArrowRight class="h-3.5 w-3.5 text-neutral-400" />
						</a>
						<a
							href={`/app/societies/${singleSociety.society_id}/announcements`}
							class="flex items-center justify-between rounded-xl border border-neutral-200/80 p-3 text-xs font-medium text-neutral-700 transition-colors hover:border-neutral-950 hover:bg-neutral-50"
						>
							<span>Society Announcements</span>
							<ArrowRight class="h-3.5 w-3.5 text-neutral-400" />
						</a>
					{:else}
						<a
							href={`/app/societies/${singleSociety.society_id}/announcements`}
							class="flex items-center justify-between rounded-xl border border-neutral-200/80 p-3 text-xs font-medium text-neutral-700 transition-colors hover:border-neutral-950 hover:bg-neutral-50"
						>
							<span>View Announcements</span>
							<ArrowRight class="h-3.5 w-3.5 text-neutral-400" />
						</a>
						<a
							href={`/app/societies/${singleSociety.society_id}/maintenance`}
							class="flex items-center justify-between rounded-xl border border-neutral-200/80 p-3 text-xs font-medium text-neutral-700 transition-colors hover:border-neutral-950 hover:bg-neutral-50"
						>
							<span>Check Maintenance & Bills</span>
							<ArrowRight class="h-3.5 w-3.5 text-neutral-400" />
						</a>
						<a
							href={`/app/societies/${singleSociety.society_id}/resident-hub`}
							class="flex items-center justify-between rounded-xl border border-neutral-200/80 p-3 text-xs font-medium text-neutral-700 transition-colors hover:border-neutral-950 hover:bg-neutral-50"
						>
							<span>Visitor Passes & Approvals</span>
							<ArrowRight class="h-3.5 w-3.5 text-neutral-400" />
						</a>
					{/if}
				</div>
			</div>
		</div>
	{/if}

	<!-- Accessible Societies Section (if multiple or list view) -->
	{#if multipleSocieties.length > 0}
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<div>
					<h2 class="text-base font-bold tracking-tight text-neutral-950">Accessible Societies</h2>
					<p class="text-xs text-neutral-500">
						You have access to multiple cooperative housing societies
					</p>
				</div>
				<a
					href={appRoutes.societies}
					class="text-xs font-semibold text-neutral-800 underline underline-offset-4 hover:text-neutral-950"
				>
					View all societies &rarr;
				</a>
			</div>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each multipleSocieties as assignment (assignment.society_id)}
					<div
						class="group flex flex-col justify-between rounded-2xl border border-[#e2e2de] bg-white p-5 shadow-xs transition-all hover:border-neutral-900 hover:shadow-md min-w-0"
					>
						<div class="min-w-0">
							<div class="flex items-center justify-between">
								<Badge
									variant="outline"
									class="border-neutral-200 bg-neutral-50 font-mono text-[10px] text-neutral-700"
								>
									{assignment.society_code}
								</Badge>
								<Badge
									variant={assignment.status === 'ACTIVE' ? 'success' : 'warning'}
									class="text-[10px]"
								>
									{assignment.status}
								</Badge>
							</div>

							<h3 class="mt-3 font-serif text-base font-bold text-[#121212] truncate group-hover:text-neutral-900">
								{assignment.society_name}
							</h3>
							<p class="mt-0.5 text-xs text-neutral-500">
								Role: <span class="font-medium text-neutral-800">{assignment.role}</span>
							</p>
						</div>

						<div class="mt-6 border-t border-neutral-100 pt-3">
							<a
								href={assignment.role === 'GATEKEEPER'
									? appRoutes.gate(assignment.society_id)
									: appRoutes.society(assignment.society_id)}
								class="inline-flex w-full items-center justify-between text-xs font-semibold text-neutral-950 transition-colors group-hover:underline"
							>
								<span>{assignment.role === 'GATEKEEPER' ? 'Open Gate Console' : 'Open Society Console'}</span>
								<ArrowRight class="h-3.5 w-3.5 transition-transform group-hover:translate-x-1" />
							</a>
						</div>
					</div>
				{/each}
			</div>
		</div>
	{:else if !singleSociety && !auth.isPlatformAdmin}
		<div class="rounded-2xl border border-dashed border-neutral-300 bg-white/60 p-8 text-center">
			<Building2 class="mx-auto h-8 w-8 text-neutral-400" />
			<h3 class="mt-3 text-sm font-semibold text-neutral-900">No Society Assignments Found</h3>
			<p class="mt-1 text-xs text-neutral-500">
				Your account is verified, but has not yet been assigned to a residential society unit or
				managing committee.
			</p>
		</div>
	{/if}
</div>

<LogoutAllDialog isOpen={isLogoutAllOpen} onClose={() => (isLogoutAllOpen = false)} />
