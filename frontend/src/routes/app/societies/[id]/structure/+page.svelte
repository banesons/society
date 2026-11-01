<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { societyApi, type FormationSummary, type SocietyProfile } from '$lib/api/society';
	import FormationWizard from '$lib/components/formation/formation-wizard.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { Layers, Sparkles, Loader2 } from '@lucide/svelte';

	let societyId = $derived(page.params.id ?? '');
	let formationSummary = $state<FormationSummary | null>(null);
	let profile = $state<SocietyProfile | null>(null);
	let isFormationWizardOpen = $state(false);
	let isLoading = $state(true);

	async function loadStructure() {
		if (!societyId) return;
		isLoading = true;
		try {
			const [summaryRes, profileRes] = await Promise.all([
				societyApi.getFormationSummary(societyId),
				societyApi.getSocietyProfile(societyId)
			]);
			formationSummary = summaryRes;
			profile = profileRes;
		} catch {
			// handled gracefully
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		loadStructure();
	});
</script>

<svelte:head>
	<title>Physical Structure & Gates — CHS Neu</title>
</svelte:head>

<div class="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
	{#if isLoading}
		<div class="flex flex-col items-center justify-center py-20 text-center">
			<Loader2 class="h-8 w-8 animate-spin text-neutral-900" />
			<p class="mt-4 text-xs text-neutral-500">Loading structure configuration...</p>
		</div>
	{:else}
		<div class="rounded-2xl border border-neutral-200 bg-white/90 p-6 shadow-xs">
			<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
				<div>
					<div class="flex items-center gap-2">
						<div
							class="flex h-8 w-8 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
						>
							<Layers class="h-4 w-4" />
						</div>
						<h1 class="text-base font-semibold tracking-tight text-neutral-950">
							Physical Structure & Gates
						</h1>
						<Badge
							variant="outline"
							class="border-emerald-200 bg-emerald-50 font-mono text-[10px] text-emerald-800"
						>
							Configured
						</Badge>
					</div>
					<p class="mt-1 text-xs text-neutral-500">
						Buildings, wings, units, gates, and gatekeeper staffing requirements configured for this
						society.
					</p>
				</div>

				<Button variant="primary" size="sm" onclick={() => (isFormationWizardOpen = true)}>
					<Sparkles class="mr-1.5 h-3.5 w-3.5" />
					<span>Society Formation Wizard</span>
				</Button>
			</div>

			<!-- Dynamic Formation Summary Metrics -->
			<div class="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-6">
				<div class="rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5">
					<span class="text-[11px] text-neutral-500">Buildings</span>
					<p class="mt-1 text-lg font-bold text-neutral-950">
						{formationSummary?.total_buildings ?? 0}
					</p>
				</div>
				<div class="rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5">
					<span class="text-[11px] text-neutral-500">Wings</span>
					<p class="mt-1 text-lg font-bold text-neutral-950">
						{formationSummary?.total_wings ?? 0}
					</p>
				</div>
				<div class="rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5">
					<span class="text-[11px] text-neutral-500">Total Floors</span>
					<p class="mt-1 text-lg font-bold text-neutral-950">
						{formationSummary?.total_floors ?? 0}
					</p>
				</div>
				<div class="rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5">
					<span class="text-[11px] text-neutral-500">Units / Flats</span>
					<p class="mt-1 text-lg font-bold text-neutral-950">
						{formationSummary?.total_units ?? 0}
					</p>
				</div>
				<div class="rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5">
					<span class="text-[11px] text-neutral-500">Gates</span>
					<p class="mt-1 text-lg font-bold text-neutral-950">
						{formationSummary?.total_gates ?? 0}
					</p>
				</div>
				<div class="rounded-xl border border-neutral-200 bg-neutral-50/80 p-3.5">
					<span class="text-[11px] text-neutral-500">Req. Headcount</span>
					<p class="mt-1 text-lg font-bold text-emerald-800">
						{formationSummary?.total_required_headcount ?? 0}
					</p>
				</div>
			</div>

			<div
				class="mt-4 flex items-center justify-between rounded-xl border border-neutral-200 bg-neutral-50/60 px-4 py-2.5 text-xs text-neutral-600"
			>
				<span>
					Staffing Status:
					<strong class="text-neutral-900">
						{formationSummary?.required_staff_per_shift ?? 0} staff / shift
					</strong>
				</span>
				<span>Registered Gatekeepers: <strong class="text-neutral-900">0</strong></span>
			</div>
		</div>

		{#if profile}
			<FormationWizard
				society={profile}
				isOpen={isFormationWizardOpen}
				onClose={() => (isFormationWizardOpen = false)}
				onComplete={() => {
					isFormationWizardOpen = false;
					loadStructure();
				}}
			/>
		{/if}
	{/if}
</div>
