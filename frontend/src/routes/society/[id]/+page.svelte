<script lang="ts">
	import { page } from '$app/state';
	import { societyApi, type SocietyProfile, type FormationSummary } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import FormationWizard from '$lib/components/formation/formation-wizard.svelte';
	import MemberRoster from '$lib/components/members/member-roster.svelte';
	import CommitteeRoster from '$lib/components/committee/committee-roster.svelte';
	import GateConsole from '$lib/components/gate/gate-console.svelte';
	import ResidentVisitorConsole from '$lib/components/resident/resident-visitor-console.svelte';
	import ResidentAnnouncementFeed from '$lib/components/resident/resident-announcement-feed.svelte';
	import AnnouncementConsole from '$lib/components/admin/announcement-console.svelte';
	import MaintenanceConsole from '$lib/components/admin/maintenance-console.svelte';
	import DocumentConsole from '$lib/components/admin/document-console.svelte';
	import ResidentBillsView from '$lib/components/resident/resident-bills-view.svelte';
	import ResidentDocumentsView from '$lib/components/resident/resident-documents-view.svelte';
	import SocietyAuditConsole from '$lib/components/admin/society-audit-console.svelte';
	import SocietyDashboard from '$lib/components/dashboard/society-dashboard.svelte';
	import { auth } from '$lib/auth.svelte';
	import {
		Building2,
		ShieldCheck,
		Loader2,
		MapPin,
		Mail,
		Lock,
		Home,
		Layers,
		Sparkles,
		LogIn
	} from '@lucide/svelte';

	let societyId = $derived(page.params.id ?? '');
	let userAssignment = $derived(
		auth.user?.society_assignments?.find((a) => a.society_id === societyId)
	);
	let isSocAdmin = $derived(auth.isSuperAdmin || userAssignment?.role === 'SOC_ADMIN');
	let isGatekeeper = $derived(userAssignment?.role === 'GATEKEEPER');
	let isResident = $derived(userAssignment?.role === 'RESIDENT');

	let profile = $state<SocietyProfile | null>(null);
	let formationSummary = $state<FormationSummary | null>(null);
	let isFormationWizardOpen = $state(false);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let errorCode = $state<string | null>(null);

	async function loadProfile() {
		if (!societyId) return;
		profile = null;
		formationSummary = null;
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
		if (societyId && !auth.isLoading) {
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

<div class="mx-auto max-w-7xl px-4 pb-12 text-neutral-900 sm:px-6 lg:px-8">
	{#if auth.isLoading || isLoading}
		<div class="flex min-h-[300px] flex-col items-center justify-center text-neutral-500">
			<Loader2 class="h-8 w-8 animate-spin text-neutral-900" />
			<p class="mt-3 text-xs">Loading society console...</p>
		</div>
	{:else if !auth.isAuthenticated}
		<div class="rounded-3xl border border-neutral-200 bg-white/90 p-8 text-center shadow-xs">
			<div
				class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
			>
				<Lock class="h-5 w-5" />
			</div>
			<h2 class="mt-3 text-base font-semibold text-neutral-900">Authentication Required</h2>
			<p class="mx-auto mt-2 max-w-md text-xs leading-relaxed text-neutral-500">
				Please sign in with your administrator phone number to access the society management
				console.
			</p>
			<div class="mt-6">
				<Button variant="primary" size="sm" onclick={() => auth.openLogin()}>
					<LogIn class="mr-1.5 h-3.5 w-3.5" />
					<span>Sign In</span>
				</Button>
			</div>
		</div>
	{:else if error}
		<div class="rounded-3xl border border-rose-200 bg-rose-50/80 p-8 text-center shadow-xs">
			<div
				class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-rose-100 text-rose-600"
			>
				<Lock class="h-5 w-5" />
			</div>
			<h2 class="mt-3 text-base font-semibold text-rose-900">
				{errorCode ? `Society Authorization (${errorCode})` : 'Access Denied'}
			</h2>
			<p class="mx-auto mt-2 max-w-md text-xs leading-relaxed text-rose-700">
				{error}
			</p>
			<div class="mt-6">
				<a href="/admin">
					<Button
						variant="outline"
						size="sm"
						class="border-rose-300 text-rose-800 hover:bg-rose-100"
					>
						<Home class="mr-1.5 h-3.5 w-3.5" />
						<span>Return to Admin Console</span>
					</Button>
				</a>
			</div>
		</div>
	{:else if profile}
		<!-- Header Banner -->
		<div
			id="overview"
			class="rounded-3xl border border-neutral-200 bg-white/90 p-6 shadow-xs sm:p-8"
		>
			<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
				<div class="flex items-center gap-4">
					<div
						class="flex h-14 w-14 items-center justify-center rounded-2xl bg-neutral-100 text-neutral-900"
					>
						<Building2 class="h-7 w-7" />
					</div>
					<div>
						<div class="flex items-center gap-2.5">
							<h1 class="text-xl font-bold tracking-tight text-neutral-950 sm:text-2xl">
								{profile.name}
							</h1>
							<Badge variant={getStatusVariant(profile.status)}>
								{profile.status}
							</Badge>
						</div>
						<div class="mt-1 flex items-center gap-2 font-mono text-xs text-neutral-600">
							<span class="rounded bg-neutral-100 px-1.5 py-0.5 font-bold text-neutral-900"
								>{profile.society_code}</span
							>
							<span class="text-neutral-400">·</span>
							<span class="text-neutral-500">{profile.city}, {profile.state}</span>
						</div>
					</div>
				</div>

				<div class="flex items-center gap-2">
					<Badge
						variant="outline"
						class="flex items-center gap-1.5 border-emerald-200 bg-emerald-50 py-1 text-emerald-800"
					>
						<ShieldCheck class="h-3.5 w-3.5 text-emerald-600" />
						<span>
							{#if isSocAdmin}
								Verified SOC_ADMIN Context
							{:else if isGatekeeper}
								Verified GATEKEEPER Duty Context
							{:else if isResident}
								Verified Resident Context
							{:else}
								Society Member Context
							{/if}
						</span>
					</Badge>
				</div>
			</div>
		</div>

		<!-- Overview Cards -->
		<div class="mt-6 grid grid-cols-1 gap-6 md:grid-cols-2">
			<!-- Location -->
			<div class="rounded-2xl border border-neutral-200 bg-white/90 p-6 shadow-xs">
				<div class="flex items-center gap-2.5 text-sm font-semibold text-neutral-900">
					<MapPin class="h-4 w-4 text-neutral-700" />
					<span>Society Address</span>
				</div>
				<div class="mt-4 space-y-1 text-xs leading-relaxed text-neutral-700">
					<p class="font-medium text-neutral-950">{profile.address_line1}</p>
					{#if profile.address_line2}
						<p>{profile.address_line2}</p>
					{/if}
					<p>{profile.locality}</p>
					<p class="font-mono text-neutral-900">
						{profile.city}, {profile.state} — {profile.pincode}
					</p>
				</div>
			</div>

			<!-- Contact Info -->
			<div class="rounded-2xl border border-neutral-200 bg-white/90 p-6 shadow-xs">
				<div class="flex items-center gap-2.5 text-sm font-semibold text-neutral-900">
					<Mail class="h-4 w-4 text-neutral-700" />
					<span>Society Contacts</span>
				</div>
				<div class="mt-4 space-y-2 text-xs text-neutral-700">
					<div class="flex items-center gap-2">
						<span class="text-neutral-500">Email:</span>
						<span class="font-mono text-neutral-900">{profile.contact_email || '—'}</span>
					</div>
					<div class="flex items-center gap-2">
						<span class="text-neutral-500">Phone:</span>
						<span class="font-mono text-neutral-900">{profile.contact_phone || '—'}</span>
					</div>
				</div>
			</div>
		</div>

		<!-- S16 Society Dashboard & Performance Overview (SOC_ADMIN only) -->
		{#if isSocAdmin && societyId}
			<div id="reports" class="mt-6">
				<SocietyDashboard {societyId} societyName={profile.name} />
			</div>
		{/if}

		<!-- Society Announcements Feed (Visible to all authorized society members and admins) -->
		{#if isResident || isSocAdmin || isGatekeeper}
			<div id="announcements" class="mt-6">
				<ResidentAnnouncementFeed {societyId} />
			</div>
		{/if}

		<!-- Admin Announcement Management Console (SOC_ADMIN only) -->
		{#if isSocAdmin && societyId}
			<div id="announcements-admin" class="mt-6">
				<AnnouncementConsole {societyId} />
			</div>
		{/if}

		<!-- Admin Maintenance & Billing Foundation Console (SOC_ADMIN only) -->
		{#if isSocAdmin && societyId}
			<div id="maintenance" class="mt-6">
				<MaintenanceConsole {societyId} />
			</div>
		{/if}

		<!-- Admin Documents & Records Console (SOC_ADMIN only) -->
		{#if isSocAdmin && societyId}
			<div id="documents" class="mt-6">
				<DocumentConsole {societyId} />
			</div>
		{/if}

		<!-- Admin Society Audit & Activity Console (SOC_ADMIN only) -->
		{#if isSocAdmin && societyId}
			<div id="audit" class="mt-6">
				<SocietyAuditConsole {societyId} societyName={profile?.name} />
			</div>
		{/if}

		<!-- Resident Maintenance Dues & Statement Section -->
		{#if (isResident || isSocAdmin) && societyId}
			<div id="resident-maintenance" class="mt-6">
				<ResidentBillsView {societyId} />
			</div>
		{/if}

		<!-- Resident Documents & Official Records Section -->
		{#if (isResident || isSocAdmin) && societyId}
			<div id="resident-documents" class="mt-6">
				<ResidentDocumentsView {societyId} />
			</div>
		{/if}

		<!-- Resident Visitor Hub Section (Occupant Visitor Approvals & Unit Policy) -->
		{#if isResident || isSocAdmin}
			<div id="resident-hub" class="mt-6">
				<ResidentVisitorConsole {societyId} />
			</div>
		{/if}

		<!-- Physical Structure & Gates Section (SOC_ADMIN only) -->
		{#if isSocAdmin}
			<div
				id="structure"
				class="mt-6 rounded-2xl border border-neutral-200 bg-white/90 p-6 shadow-xs"
			>
				<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
					<div>
						<div class="flex items-center gap-2">
							<div
								class="flex h-8 w-8 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
							>
								<Layers class="h-4 w-4" />
							</div>
							<h2 class="text-base font-semibold tracking-tight text-neutral-950">
								Physical Structure & Gates
							</h2>
							<Badge
								variant="outline"
								class="border-emerald-200 bg-emerald-50 font-mono text-[10px] text-emerald-800"
								>Configured</Badge
							>
						</div>
						<p class="mt-1 text-xs text-neutral-500">
							Buildings, wings, units, gates, and gatekeeper staffing requirements configured for
							this society.
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
					<span
						>Staffing Status: <strong class="text-neutral-900"
							>{formationSummary?.required_staff_per_shift ?? 0} staff / shift</strong
						></span
					>
					<span>Registered Gatekeepers: <strong class="text-neutral-900">0</strong></span>
				</div>
			</div>
		{/if}

		<!-- Members & Relationships Section (SOC_ADMIN only) -->
		{#if isSocAdmin && societyId}
			<div id="members" class="mt-6">
				<MemberRoster {societyId} />
			</div>
		{/if}

		<!-- Society Roles & Committee Designations Section (SOC_ADMIN only) -->
		{#if isSocAdmin && societyId}
			<div id="committee" class="mt-6">
				<CommitteeRoster {societyId} />
			</div>
		{/if}

		<!-- Gate Foundation & Perimeter Operations Section (Gatekeepers & SOC_ADMIN) -->
		{#if (isGatekeeper || isSocAdmin) && societyId}
			<div id="gatekeeping" class="mt-6">
				<GateConsole {societyId} />
			</div>
		{/if}

		<!-- Formation Wizard Dialog -->
		{#if isSocAdmin}
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
	{/if}
</div>
