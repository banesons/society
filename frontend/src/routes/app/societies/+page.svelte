<script lang="ts">
	import { onMount } from 'svelte';
	import { auth } from '$lib/auth.svelte';
	import { appRoutes } from '$lib/routing';
	import { societyApi, type UserSocietyAssignment } from '$lib/api/society';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Card from '$lib/components/ui/card.svelte';
	import Breadcrumbs from '$lib/components/nav/breadcrumbs.svelte';
	import { Building2, ArrowRight, MapPin, Loader2 } from '@lucide/svelte';

	let accessibleSocieties = $state<UserSocietyAssignment[]>([]);
	let isLoading = $state(true);

	onMount(async () => {
		if (auth.isSuperAdmin) {
			try {
				const res = await societyApi.listSocieties({ limit: 100 });
				accessibleSocieties = res.items
					.filter((s) => s.status === 'ACTIVE')
					.map((s) => ({
						society_id: s.id,
						society_code: s.society_code,
						society_name: s.name,
						role: 'SUPER_ADMIN',
						status: s.status as 'ACTIVE',
						city: s.city,
						state: s.state
					}));
			} catch {
				accessibleSocieties = auth.user?.society_assignments ?? [];
			}
		} else {
			accessibleSocieties = auth.user?.society_assignments ?? [];
		}
		isLoading = false;
	});
</script>

<svelte:head>
	<title>Accessible Societies — CHS Platform</title>
	<meta
		name="description"
		content="Select a cooperative housing society from your verified assignments"
	/>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div class="space-y-6">
	<!-- Top Breadcrumb -->
	<div class="flex items-center justify-between">
		<Breadcrumbs />
	</div>

	<!-- Page Header -->
	<div class="rounded-3xl border border-neutral-200 bg-white p-6 shadow-xs sm:p-8">
		<div class="flex items-center gap-3">
			<div
				class="flex h-12 w-12 items-center justify-center rounded-2xl bg-neutral-100 text-neutral-900"
			>
				<Building2 class="h-6 w-6" />
			</div>
			<div>
				<h1 class="text-2xl font-bold tracking-tight text-neutral-950">Accessible Societies</h1>
				<p class="mt-1 text-xs text-neutral-500 sm:text-sm">
					Select a society workspace to view governance, announcements, maintenance, and perimeter
					operations.
				</p>
			</div>
		</div>
	</div>

	<!-- Society List Grid -->
	{#if isLoading}
		<Card
			class="flex flex-col items-center justify-center border-neutral-200 bg-white/90 p-12 text-center shadow-xs"
		>
			<Loader2 class="h-8 w-8 animate-spin text-neutral-900" />
			<p class="mt-4 font-serif text-sm font-medium text-neutral-950">
				Loading society directory...
			</p>
		</Card>
	{:else if accessibleSocieties.length > 0}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each accessibleSocieties as society (society.society_id)}
				<a
					href={society.role === 'GATEKEEPER'
						? appRoutes.gate(society.society_id)
						: appRoutes.society(society.society_id)}
					class="group flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white p-6 shadow-xs transition-all hover:border-neutral-950 hover:shadow-md"
				>
					<div>
						<div class="flex items-center justify-between">
							<Badge
								variant="outline"
								class="border-neutral-200 bg-neutral-50 font-mono text-[10px] text-neutral-700"
							>
								{society.society_code}
							</Badge>
							<Badge
								variant={society.status === 'ACTIVE' ? 'success' : 'warning'}
								class="text-[10px]"
							>
								{society.status}
							</Badge>
						</div>

						<h2 class="mt-4 text-lg font-bold text-neutral-950 group-hover:text-neutral-900">
							{society.society_name}
						</h2>

						{#if society.city}
							<p class="mt-1 flex items-center gap-1 text-xs text-neutral-500">
								<MapPin class="h-3 w-3" />
								<span>{society.city}{society.state ? `, ${society.state}` : ''}</span>
							</p>
						{/if}

						<div class="mt-4 border-t border-neutral-100 pt-3 text-xs text-neutral-600">
							Assigned Role: <strong class="text-neutral-900">{society.role}</strong>
						</div>
					</div>

					<div
						class="mt-6 flex items-center justify-between border-t border-neutral-100 pt-3 text-xs font-semibold text-neutral-950"
					>
						<span>{society.role === 'GATEKEEPER' ? 'Open Gate Console' : 'Open Workspace'}</span>
						<ArrowRight class="h-4 w-4 transition-transform group-hover:translate-x-1" />
					</div>
				</a>
			{/each}
		</div>
	{:else}
		<Card class="rounded-2xl border border-dashed border-neutral-300 bg-white/60 p-12 text-center">
			<Building2 class="mx-auto h-10 w-10 text-neutral-400" />
			<h3 class="mt-3 text-base font-semibold text-neutral-900">No Societies Assigned</h3>
			<p class="mx-auto mt-1 max-w-sm text-xs text-neutral-500">
				Your verified phone identity is not currently assigned to any cooperative housing society
				units or administration teams.
			</p>
			<div class="mt-6">
				<a href={appRoutes.dashboard}>
					<Button variant="outline" size="sm">
						<span>Return to Dashboard</span>
					</Button>
				</a>
			</div>
		</Card>
	{/if}
</div>
