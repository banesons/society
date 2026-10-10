<script lang="ts">
	import { page } from '$app/state';
	import { societyApi, type SocietyProfile, type FormationSummary } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import FormationWizard from '$lib/components/formation/formation-wizard.svelte';
	import {
		Building2,
		ShieldCheck,
		Loader2,
		MapPin,
		Mail,
		Lock,
		Home,
		ArrowLeft,
		Layers,
		Sparkles
	} from '@lucide/svelte';

	let societyId = $derived(page.params.id);
	let profile = $state<SocietyProfile | null>(null);
	let formationSummary = $state<FormationSummary | null>(null);
	let isFormationWizardOpen = $state(false);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let errorCode = $state<string | null>(null);

	async function loadProfile() {
		if (!societyId) return;
		isLoading = true;
		error = null;
		errorCode = null;

		try {
			const [profRes, formRes] = await Promise.all([
				societyApi.getSocietyProfile(societyId),
				societyApi.getFormationSummary(societyId).catch(() => null)
			]);
			profile = profRes;
			formationSummary = formRes;
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				errorCode = err.code;
				if (err.code === 'SOCIETY_PENDING') {
					error =
						'This society has not been activated yet. Operations will be available once onboarding review completes.';
				} else if (err.code === 'SOCIETY_SUSPENDED') {
					error =
						'This society is currently suspended. Operational access is blocked until platform reactivation.';
				} else if (err.code === 'SOCIETY_ACCESS_DENIED') {
					error =
						'Access denied. You do not hold an active administrator assignment for this society.';
				} else {
					error = err.message;
				}
			} else {
				error = 'Failed to load society profile.';
			}
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (societyId) {
			loadProfile();
		}
	});

	function getStatusVariant(status: string): 'default' | 'success' | 'warning' | 'destructive' {
		switch (status) {
			case 'ACTIVE':
				return 'success';
			case 'SUSPENDED':
				return 'destructive';
			default:
				return 'warning';
		}
	}
</script>

<svelte:head>
	<title>{profile ? `${profile.name} — Society Console` : 'Society Console'} — CHS</title>
</svelte:head>

<div class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
	<!-- Navigation back -->
	<div class="mb-6 flex items-center justify-between">
		<a
			href="/"
			class="inline-flex items-center gap-1.5 text-xs font-medium text-slate-400 transition-colors hover:text-slate-200"
		>
			<ArrowLeft class="h-3.5 w-3.5" />
			<span>Return to Home</span>
		</a>
		<div class="flex items-center gap-2">
			<Badge variant="outline" class="font-mono text-xs text-indigo-400">
				Tenant Boundary: SOC_ADMIN
			</Badge>
		</div>
	</div>

	{#if isLoading}
		<div class="flex min-h-[300px] flex-col items-center justify-center text-slate-400">
			<Loader2 class="h-8 w-8 animate-spin text-indigo-500" />
			<p class="mt-3 text-xs">Loading society console...</p>
		</div>
	{:else if error}
		<div class="rounded-2xl border border-rose-500/30 bg-rose-500/10 p-6 text-center shadow-xl">
			<div
				class="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-rose-500/20 text-rose-400"
			>
				<Lock class="h-6 w-6" />
			</div>
			<h2 class="mt-3 text-base font-semibold text-rose-200">
				{errorCode ? `Society Authorization (${errorCode})` : 'Access Denied'}
			</h2>
			<p class="mx-auto mt-2 max-w-md text-xs leading-relaxed text-rose-300">
				{error}
			</p>
			<div class="mt-6">
				<a href="/">
					<Button variant="outline" size="sm" class="border-rose-500/40 text-rose-200">
						<Home class="mr-1.5 h-3.5 w-3.5" />
						<span>Return to Dashboard</span>
					</Button>
				</a>
			</div>
		</div>
	{:else if profile}
		<!-- Header Banner -->
		<div
			class="rounded-2xl border border-slate-800 bg-gradient-to-r from-slate-900 via-slate-900 to-indigo-950/40 p-6 shadow-xl sm:p-8"
		>
			<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
				<div class="flex items-center gap-4">
					<div
						class="flex h-14 w-14 items-center justify-center rounded-2xl border border-indigo-500/20 bg-indigo-500/10 text-indigo-400 shadow-inner"
					>
						<Building2 class="h-7 w-7" />
					</div>
					<div>
						<div class="flex items-center gap-2.5">
							<h1 class="text-xl font-bold tracking-tight text-white sm:text-2xl">
								{profile.name}
							</h1>
							<Badge variant={getStatusVariant(profile.status)}>
								{profile.status}
							</Badge>
						</div>
						<div class="mt-1 flex items-center gap-2 font-mono text-xs text-indigo-300">
							<span>{profile.society_code}</span>
							<span class="text-slate-500">·</span>
							<span class="text-slate-400">{profile.city}, {profile.state}</span>
						</div>
					</div>
				</div>

				<div class="flex items-center gap-2">
					<Badge
						variant="outline"
						class="flex items-center gap-1.5 border-emerald-500/30 py-1 text-emerald-400"
					>
						<ShieldCheck class="h-3.5 w-3.5" />
						<span>Verified SOC_ADMIN Context</span>
					</Badge>
				</div>
			</div>
		</div>

		<!-- Overview Cards -->
		<div class="mt-6 grid grid-cols-1 gap-6 md:grid-cols-2">
			<!-- Location -->
			<div class="rounded-2xl border border-slate-800 bg-slate-900/60 p-6 shadow-lg">
				<div class="flex items-center gap-2.5 text-sm font-semibold text-slate-200">
					<MapPin class="h-4 w-4 text-indigo-400" />
					<span>Society Address</span>
				</div>
				<div class="mt-4 space-y-1 text-xs leading-relaxed text-slate-300">
					<p class="font-medium text-slate-100">{profile.address_line1}</p>
					{#if profile.address_line2}
						<p>{profile.address_line2}</p>
					{/if}
					<p>{profile.locality}</p>
					<p class="font-mono text-indigo-300">
						{profile.city}, {profile.state} — {profile.pincode}
					</p>
				</div>
			</div>

			<!-- Contact Info -->
			<div class="rounded-2xl border border-slate-800 bg-slate-900/60 p-6 shadow-lg">
				<div class="flex items-center gap-2.5 text-sm font-semibold text-slate-200">
					<Mail class="h-4 w-4 text-indigo-400" />
					<span>Society Contacts</span>
				</div>
				<div class="mt-4 space-y-2 text-xs text-slate-300">
					<div class="flex items-center gap-2">
						<span class="text-slate-400">Email:</span>
						<span class="font-mono text-slate-200">{profile.contact_email || '—'}</span>
					</div>
					<div class="flex items-center gap-2">
						<span class="text-slate-400">Phone:</span>
						<span class="font-mono text-slate-200">{profile.contact_phone || '—'}</span>
					</div>
				</div>
			</div>
		</div>

		<!-- S05 Society Structure & Formation Section -->
		<div class="mt-6 rounded-2xl border border-slate-800 bg-slate-900/60 p-6 shadow-xl">
			<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
				<div>
					<div class="flex items-center gap-2">
						<div
							class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-500/10 text-indigo-400"
						>
							<Layers class="h-4 w-4" />
						</div>
						<h2 class="text-base font-semibold text-white">Physical Structure & Gates</h2>
						<Badge variant="outline" class="font-mono text-[10px] text-emerald-400"
							>S05 Active</Badge
						>
					</div>
					<p class="mt-1 text-xs text-slate-400">
						Buildings, wings, units, gates, and gatekeeper staffing requirements configured for this
						society.
					</p>
				</div>

				<Button
					variant="primary"
					size="sm"
					onclick={() => (isFormationWizardOpen = true)}
					class="shadow-lg shadow-indigo-500/20"
				>
					<Sparkles class="mr-1.5 h-3.5 w-3.5" />
					<span>Society Formation Wizard</span>
				</Button>
			</div>

			<!-- Dynamic Formation Summary Metrics -->
			<div class="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-6">
				<div class="rounded-xl border border-slate-800/80 bg-slate-950/60 p-3.5">
					<span class="text-[11px] text-slate-400">Buildings</span>
					<p class="mt-1 text-lg font-bold text-slate-100">
						{formationSummary?.total_buildings ?? 0}
					</p>
				</div>
				<div class="rounded-xl border border-slate-800/80 bg-slate-950/60 p-3.5">
					<span class="text-[11px] text-slate-400">Wings</span>
					<p class="mt-1 text-lg font-bold text-slate-100">{formationSummary?.total_wings ?? 0}</p>
				</div>
				<div class="rounded-xl border border-slate-800/80 bg-slate-950/60 p-3.5">
					<span class="text-[11px] text-slate-400">Total Floors</span>
					<p class="mt-1 text-lg font-bold text-slate-100">{formationSummary?.total_floors ?? 0}</p>
				</div>
				<div class="rounded-xl border border-slate-800/80 bg-slate-950/60 p-3.5">
					<span class="text-[11px] text-slate-400">Units / Flats</span>
					<p class="mt-1 text-lg font-bold text-indigo-400">{formationSummary?.total_units ?? 0}</p>
				</div>
				<div class="rounded-xl border border-slate-800/80 bg-slate-950/60 p-3.5">
					<span class="text-[11px] text-slate-400">Gates</span>
					<p class="mt-1 text-lg font-bold text-slate-100">{formationSummary?.total_gates ?? 0}</p>
				</div>
				<div class="rounded-xl border border-slate-800/80 bg-slate-950/60 p-3.5">
					<span class="text-[11px] text-slate-400">Req. Headcount</span>
					<p class="mt-1 text-lg font-bold text-emerald-400">
						{formationSummary?.total_required_headcount ?? 0}
					</p>
				</div>
			</div>

			<div
				class="mt-4 flex items-center justify-between rounded-xl border border-slate-800/60 bg-slate-950/40 px-4 py-2.5 text-xs text-slate-400"
			>
				<span
					>Staffing Status: <strong class="text-slate-300"
						>{formationSummary?.required_staff_per_shift ?? 0} staff / shift</strong
					></span
				>
				<span
					>Registered Gatekeepers: <strong class="text-slate-400">0</strong> (Identity deferred to S08)</span
				>
			</div>
		</div>

		<!-- Formation Wizard Dialog -->
		<FormationWizard
			society={profile}
			isOpen={isFormationWizardOpen}
			onClose={() => (isFormationWizardOpen = false)}
			onComplete={() => {
				isFormationWizardOpen = false;
				loadProfile();
			}}
		/>
	{/if}
</div>
