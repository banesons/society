<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		societyApi,
		type SocietyProfile,
		type Announcement,
		type PendingQRVisit
	} from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import SocietyDashboard from '$lib/components/dashboard/society-dashboard.svelte';
	import ResidentAnnouncementFeed from '$lib/components/resident/resident-announcement-feed.svelte';
	import { auth } from '$lib/auth.svelte';
	import { appRoutes } from '$lib/routing';
	import {
		Building2,
		ShieldCheck,
		Loader2,
		MapPin,
		Mail,
		Phone,
		Lock,
		Home,
		DoorClosed,
		Users,
		Wrench,
		FileText,
		MessageSquare,
		BarChart3,
		Shield,
		ArrowRight,
		LogIn,
		QrCode,
		UserCheck
	} from '@lucide/svelte';
	import { getEffectiveSocietyRole } from '$lib/navigation';

	let societyId = $derived(page.params.id ?? '');
	let effectiveRole = $derived(
		getEffectiveSocietyRole(auth.user?.platform_roles, auth.user?.society_assignments, societyId)
	);
	let isSocAdmin = $derived(
		effectiveRole === 'SOC_ADMIN' || effectiveRole === 'SUPER_ADMIN' || effectiveRole === 'ADMIN'
	);
	let isGatekeeper = $derived(effectiveRole === 'GATEKEEPER');
	let isResident = $derived(effectiveRole === 'RESIDENT');

	let profile = $state<SocietyProfile | null>(null);
	let recentAnnouncements = $state<Announcement[]>([]);
	let pendingQRRegistrations = $state<PendingQRVisit[]>([]);
	let pendingQRTotal = $state(0);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let errorCode = $state<string | null>(null);

	async function loadOverview() {
		if (!societyId) return;
		isLoading = true;
		error = null;
		errorCode = null;

		try {
			const [profRes, feedRes, qrRes] = await Promise.all([
				societyApi.getSocietyProfile(societyId),
				societyApi
					.getAnnouncementFeed(societyId, { limit: 3 })
					.then((res) => res.announcements)
					.catch(() => [] as Announcement[]),
				isSocAdmin
					? societyApi
							.listPendingQRRegistrations(societyId, { limit: 3 })
							.then((res) => {
								pendingQRTotal = res.total;
								return res.pending_registrations;
							})
							.catch(() => [] as PendingQRVisit[])
					: Promise.resolve([] as PendingQRVisit[])
			]);

			profile = profRes;
			recentAnnouncements = feedRes;
			pendingQRRegistrations = qrRes;
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
			if (isGatekeeper) {
				goto(`/app/societies/${societyId}/gate`, { replaceState: true });
				return;
			}
			loadOverview();
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
	<title>{profile ? `${profile.name} — Operations Hub` : 'Society Operations'} — CHS</title>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div class="mx-auto w-full max-w-7xl min-w-0 px-4 pb-16 text-neutral-900 sm:px-6 lg:px-8">
	{#if auth.isLoading || isLoading || isGatekeeper}
		<div class="flex min-h-[300px] flex-col items-center justify-center text-neutral-500">
			<Loader2 class="h-8 w-8 animate-spin text-neutral-900" />
			<p class="mt-3 text-xs">
				{#if isGatekeeper}
					Directing to Gate Operations...
				{:else}
					Loading society operations hub...
				{/if}
			</p>
		</div>
	{:else if !auth.isAuthenticated}
		<div class="rounded-3xl border border-[#e2e2de] bg-white p-8 text-center shadow-xs">
			<div
				class="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-neutral-100 text-neutral-900"
			>
				<Lock class="h-5 w-5" />
			</div>
			<h2 class="mt-3 text-base font-semibold text-neutral-900">Authentication Required</h2>
			<p class="mx-auto mt-2 max-w-md text-xs leading-relaxed text-neutral-500">
				Please sign in with your verified phone identity to access the society management console.
			</p>
			<div class="mt-6">
				<a href={appRoutes.login()}>
					<Button variant="primary" size="sm">
						<LogIn class="mr-1.5 h-3.5 w-3.5" />
						<span>Sign In</span>
					</Button>
				</a>
			</div>
		</div>
	{:else if error}
		<div class="rounded-3xl border border-rose-200 bg-rose-50/80 p-8 text-center shadow-xs">
			<div
				class="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-rose-100 text-rose-600"
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
				<a href={appRoutes.dashboard}>
					<Button
						variant="outline"
						size="sm"
						class="border-rose-300 text-rose-800 hover:bg-rose-100"
					>
						<Home class="mr-1.5 h-3.5 w-3.5" />
						<span>Return to Dashboard</span>
					</Button>
				</a>
			</div>
		</div>
	{:else if profile}
		<!-- 1. Editorial Society Header Banner -->
		<section
			class="rounded-3xl border border-[#e2e2de] bg-white p-6 shadow-xs sm:p-8"
			aria-labelledby="society-overview-heading"
		>
			<div class="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
				<div class="flex items-start gap-4 min-w-0">
					<div
						class="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-neutral-100 text-neutral-900"
					>
						<Building2 class="h-7 w-7" />
					</div>
					<div class="min-w-0 flex-1">
						<div class="flex flex-wrap items-center gap-2.5">
							<h1
								id="society-overview-heading"
								class="truncate font-serif text-2xl font-bold tracking-tight text-[#121212] sm:text-3xl"
							>
								{profile.name}
							</h1>
							<Badge variant={getStatusVariant(profile.status)}>
								{profile.status}
							</Badge>
						</div>
						<div class="mt-1.5 flex flex-wrap items-center gap-2 text-xs text-neutral-600">
							<span class="rounded bg-neutral-100 px-2 py-0.5 font-mono font-semibold text-neutral-900">
								{profile.society_code}
							</span>
							<span class="text-neutral-300">·</span>
							<span class="truncate">{profile.locality ? `${profile.locality}, ` : ''}{profile.city}, {profile.state}</span>
						</div>
					</div>
				</div>

				<div class="flex flex-wrap items-center gap-2.5">
					<Badge
						variant="outline"
						class="flex items-center gap-1.5 border-emerald-200 bg-emerald-50/80 px-2.5 py-1 text-emerald-800"
					>
						<ShieldCheck class="h-3.5 w-3.5 text-emerald-600" />
						<span class="text-xs font-medium">
							{#if isSocAdmin}
								Verified Administrator
							{:else if isResident}
								Verified Resident
							{:else}
								Society Member
							{/if}
						</span>
					</Badge>

					{#if isSocAdmin}
						<a href={`/app/societies/${societyId}/gate`}>
							<Button variant="primary" size="sm">
								<DoorClosed class="mr-1.5 h-3.5 w-3.5" />
								<span>Gate Console</span>
							</Button>
						</a>
					{:else if isResident}
						<a href={`/app/societies/${societyId}/resident-hub`}>
							<Button variant="primary" size="sm">
								<UserCheck class="mr-1.5 h-3.5 w-3.5" />
								<span>Visitor Passes</span>
							</Button>
						</a>
					{/if}
				</div>
			</div>

			<!-- Society Metadata Strip: Location & Contact -->
			<div class="mt-6 grid grid-cols-1 gap-4 border-t border-[#e2e2de] pt-5 sm:grid-cols-2 lg:grid-cols-3">
				<div class="flex items-start gap-2.5 min-w-0">
					<MapPin class="mt-0.5 h-4 w-4 shrink-0 text-neutral-400" />
					<div class="min-w-0 text-xs leading-relaxed text-neutral-600">
						<p class="font-medium text-neutral-900 truncate">{profile.address_line1}</p>
						<p class="font-mono text-[11px] text-neutral-500">{profile.city}, {profile.state} — {profile.pincode}</p>
					</div>
				</div>

				<div class="flex items-start gap-2.5 min-w-0">
					<Mail class="mt-0.5 h-4 w-4 shrink-0 text-neutral-400" />
					<div class="min-w-0 text-xs leading-relaxed text-neutral-600">
						<p class="text-neutral-500">Contact Email</p>
						<p class="font-mono font-medium text-neutral-900 truncate">{profile.contact_email || 'Not specified'}</p>
					</div>
				</div>

				<div class="flex items-start gap-2.5 min-w-0">
					<Phone class="mt-0.5 h-4 w-4 shrink-0 text-neutral-400" />
					<div class="min-w-0 text-xs leading-relaxed text-neutral-600">
						<p class="text-neutral-500">Contact Phone</p>
						<p class="font-mono font-medium text-neutral-900 truncate">{profile.contact_phone || 'Not specified'}</p>
					</div>
				</div>
			</div>
		</section>

		<!-- 2. Role-Aware Operational Intelligence & Scorecard -->
		{#if isSocAdmin}
			<section class="mt-8 space-y-4" aria-label="Operational Intelligence">
				<SocietyDashboard {societyId} societyName={profile.name} />
			</section>
		{/if}

		<!-- 3. Actionable Items: Pending QR Approvals (Only when real data exists) -->
		{#if isSocAdmin && pendingQRTotal > 0}
			<section
				class="mt-8 rounded-3xl border border-amber-200 bg-amber-50/70 p-6 shadow-xs"
				aria-labelledby="pending-qr-heading"
			>
				<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
					<div class="flex items-center gap-3">
						<div
							class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-amber-500 text-white"
						>
							<QrCode class="h-5 w-5" />
						</div>
						<div>
							<h2 id="pending-qr-heading" class="text-sm font-bold text-neutral-950">
								{pendingQRTotal} Visitor{pendingQRTotal === 1 ? '' : 's'} Awaiting Gate Admission
							</h2>
							<p class="text-xs text-neutral-600">
								Self-registered via QR codes at entry gates and pending gatekeeper decision
							</p>
						</div>
					</div>

					<a href={`/app/societies/${societyId}/gate`}>
						<Button variant="primary" size="sm" class="bg-amber-600 hover:bg-amber-700 text-white border-transparent">
							<span>Review in Gate Console</span>
							<ArrowRight class="ml-1.5 h-3.5 w-3.5" />
						</Button>
					</a>
				</div>
			</section>
		{/if}

		<!-- 4. Modular Operations & Quick Workspaces Directory -->
		<section class="mt-8 space-y-4" aria-labelledby="modules-directory-heading">
			<div class="flex items-center justify-between">
				<h2 id="modules-directory-heading" class="font-serif text-lg font-bold tracking-tight text-[#121212]">
					Operations Directory
				</h2>
				<span class="text-xs text-neutral-500">Dedicated operational modules</span>
			</div>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#if isSocAdmin}
					<!-- Gatekeeping & Security -->
					<a
						href={`/app/societies/${societyId}/gate`}
						class="group flex flex-col justify-between rounded-2xl border border-[#e2e2de] bg-white p-5 shadow-xs transition-all hover:border-neutral-900 hover:shadow-md"
					>
						<div class="flex items-start gap-3.5">
							<div
								class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-neutral-900 transition-colors group-hover:bg-[#121212] group-hover:text-white"
							>
								<DoorClosed class="h-5 w-5" />
							</div>
							<div class="min-w-0">
								<h3 class="text-sm font-bold text-neutral-950 group-hover:text-neutral-900">
									Gate Operations
								</h3>
								<p class="mt-0.5 text-xs text-neutral-500 leading-relaxed">
									Duty shifts, visitor pass checks, QR approvals, and turnstile logs.
								</p>
							</div>
						</div>
						<div class="mt-4 flex items-center justify-end border-t border-neutral-100 pt-3 text-xs font-semibold text-neutral-950">
							<span class="group-hover:underline">Open Console</span>
							<ArrowRight class="ml-1 h-3.5 w-3.5 transition-transform group-hover:translate-x-1" />
						</div>
					</a>

					<!-- Members & Residents -->
					<a
						href={`/app/societies/${societyId}/members`}
						class="group flex flex-col justify-between rounded-2xl border border-[#e2e2de] bg-white p-5 shadow-xs transition-all hover:border-neutral-900 hover:shadow-md"
					>
						<div class="flex items-start gap-3.5">
							<div
								class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-neutral-900 transition-colors group-hover:bg-[#121212] group-hover:text-white"
							>
								<Users class="h-5 w-5" />
							</div>
							<div class="min-w-0">
								<h3 class="text-sm font-bold text-neutral-950 group-hover:text-neutral-900">
									Members & Occupancies
								</h3>
								<p class="mt-0.5 text-xs text-neutral-500 leading-relaxed">
									Owner and tenant rosters, unit allocations, and occupancy history.
								</p>
							</div>
						</div>
						<div class="mt-4 flex items-center justify-end border-t border-neutral-100 pt-3 text-xs font-semibold text-neutral-950">
							<span class="group-hover:underline">Manage Roster</span>
							<ArrowRight class="ml-1 h-3.5 w-3.5 transition-transform group-hover:translate-x-1" />
						</div>
					</a>

					<!-- Maintenance & Billing -->
					<a
						href={`/app/societies/${societyId}/maintenance`}
						class="group flex flex-col justify-between rounded-2xl border border-[#e2e2de] bg-white p-5 shadow-xs transition-all hover:border-neutral-900 hover:shadow-md"
					>
						<div class="flex items-start gap-3.5">
							<div
								class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-neutral-900 transition-colors group-hover:bg-[#121212] group-hover:text-white"
							>
								<Wrench class="h-5 w-5" />
							</div>
							<div class="min-w-0">
								<h3 class="text-sm font-bold text-neutral-950 group-hover:text-neutral-900">
									Maintenance & Billing
								</h3>
								<p class="mt-0.5 text-xs text-neutral-500 leading-relaxed">
									Recurring charge cycles, offline and digital payment receipts, and arrears.
								</p>
							</div>
						</div>
						<div class="mt-4 flex items-center justify-end border-t border-neutral-100 pt-3 text-xs font-semibold text-neutral-950">
							<span class="group-hover:underline">Billing Operations</span>
							<ArrowRight class="ml-1 h-3.5 w-3.5 transition-transform group-hover:translate-x-1" />
						</div>
					</a>

					<!-- Announcements -->
					<a
						href={`/app/societies/${societyId}/announcements`}
						class="group flex flex-col justify-between rounded-2xl border border-[#e2e2de] bg-white p-5 shadow-xs transition-all hover:border-neutral-900 hover:shadow-md"
					>
						<div class="flex items-start gap-3.5">
							<div
								class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-neutral-900 transition-colors group-hover:bg-[#121212] group-hover:text-white"
							>
								<MessageSquare class="h-5 w-5" />
							</div>
							<div class="min-w-0">
								<h3 class="text-sm font-bold text-neutral-950 group-hover:text-neutral-900">
									Announcements
								</h3>
								<p class="mt-0.5 text-xs text-neutral-500 leading-relaxed">
									Publish broadcast notices, emergency alerts, and circulars to residents.
								</p>
							</div>
						</div>
						<div class="mt-4 flex items-center justify-end border-t border-neutral-100 pt-3 text-xs font-semibold text-neutral-950">
							<span class="group-hover:underline">Broadcast Hub</span>
							<ArrowRight class="ml-1 h-3.5 w-3.5 transition-transform group-hover:translate-x-1" />
						</div>
					</a>

					<!-- Managing Committee -->
					<a
						href={`/app/societies/${societyId}/committee`}
						class="group flex flex-col justify-between rounded-2xl border border-[#e2e2de] bg-white p-5 shadow-xs transition-all hover:border-neutral-900 hover:shadow-md"
					>
						<div class="flex items-start gap-3.5">
							<div
								class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-neutral-900 transition-colors group-hover:bg-[#121212] group-hover:text-white"
							>
								<Shield class="h-5 w-5" />
							</div>
							<div class="min-w-0">
								<h3 class="text-sm font-bold text-neutral-950 group-hover:text-neutral-900">
									Managing Committee
								</h3>
								<p class="mt-0.5 text-xs text-neutral-500 leading-relaxed">
									Designations, terms, appointed office-bearers, and governance quorum.
								</p>
							</div>
						</div>
						<div class="mt-4 flex items-center justify-end border-t border-neutral-100 pt-3 text-xs font-semibold text-neutral-950">
							<span class="group-hover:underline">View Committee</span>
							<ArrowRight class="ml-1 h-3.5 w-3.5 transition-transform group-hover:translate-x-1" />
						</div>
					</a>

					<!-- Audit Ledger -->
					<a
						href={`/app/societies/${societyId}/audit`}
						class="group flex flex-col justify-between rounded-2xl border border-[#e2e2de] bg-white p-5 shadow-xs transition-all hover:border-neutral-900 hover:shadow-md"
					>
						<div class="flex items-start gap-3.5">
							<div
								class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-neutral-900 transition-colors group-hover:bg-[#121212] group-hover:text-white"
							>
								<BarChart3 class="h-5 w-5" />
							</div>
							<div class="min-w-0">
								<h3 class="text-sm font-bold text-neutral-950 group-hover:text-neutral-900">
									Audit Ledger
								</h3>
								<p class="mt-0.5 text-xs text-neutral-500 leading-relaxed">
									Immutable operational log recording all administrative actions.
								</p>
							</div>
						</div>
						<div class="mt-4 flex items-center justify-end border-t border-neutral-100 pt-3 text-xs font-semibold text-neutral-950">
							<span class="group-hover:underline">Review Audit Log</span>
							<ArrowRight class="ml-1 h-3.5 w-3.5 transition-transform group-hover:translate-x-1" />
						</div>
					</a>
				{:else if isResident}
					<!-- Resident Hub -->
					<a
						href={`/app/societies/${societyId}/resident-hub`}
						class="group flex flex-col justify-between rounded-2xl border border-[#e2e2de] bg-white p-5 shadow-xs transition-all hover:border-neutral-900 hover:shadow-md"
					>
						<div class="flex items-start gap-3.5">
							<div
								class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-neutral-900 transition-colors group-hover:bg-[#121212] group-hover:text-white"
							>
								<UserCheck class="h-5 w-5" />
							</div>
							<div class="min-w-0">
								<h3 class="text-sm font-bold text-neutral-950 group-hover:text-neutral-900">
									Visitor Hub
								</h3>
								<p class="mt-0.5 text-xs text-neutral-500 leading-relaxed">
									Pre-approve guests, issue digital passes, and approve entries at the gate.
								</p>
							</div>
						</div>
						<div class="mt-4 flex items-center justify-end border-t border-neutral-100 pt-3 text-xs font-semibold text-neutral-950">
							<span class="group-hover:underline">Open Passes</span>
							<ArrowRight class="ml-1 h-3.5 w-3.5 transition-transform group-hover:translate-x-1" />
						</div>
					</a>

					<!-- Resident Maintenance -->
					<a
						href={`/app/societies/${societyId}/maintenance`}
						class="group flex flex-col justify-between rounded-2xl border border-[#e2e2de] bg-white p-5 shadow-xs transition-all hover:border-neutral-900 hover:shadow-md"
					>
						<div class="flex items-start gap-3.5">
							<div
								class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-neutral-900 transition-colors group-hover:bg-[#121212] group-hover:text-white"
							>
								<Wrench class="h-5 w-5" />
							</div>
							<div class="min-w-0">
								<h3 class="text-sm font-bold text-neutral-950 group-hover:text-neutral-900">
									Maintenance & Dues
								</h3>
								<p class="mt-0.5 text-xs text-neutral-500 leading-relaxed">
									View statement of dues, itemized maintenance bills, and receipts.
								</p>
							</div>
						</div>
						<div class="mt-4 flex items-center justify-end border-t border-neutral-100 pt-3 text-xs font-semibold text-neutral-950">
							<span class="group-hover:underline">View Dues</span>
							<ArrowRight class="ml-1 h-3.5 w-3.5 transition-transform group-hover:translate-x-1" />
						</div>
					</a>

					<!-- Documents -->
					<a
						href={`/app/societies/${societyId}/documents`}
						class="group flex flex-col justify-between rounded-2xl border border-[#e2e2de] bg-white p-5 shadow-xs transition-all hover:border-neutral-900 hover:shadow-md"
					>
						<div class="flex items-start gap-3.5">
							<div
								class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-neutral-900 transition-colors group-hover:bg-[#121212] group-hover:text-white"
							>
								<FileText class="h-5 w-5" />
							</div>
							<div class="min-w-0">
								<h3 class="text-sm font-bold text-neutral-950 group-hover:text-neutral-900">
									Official Records
								</h3>
								<p class="mt-0.5 text-xs text-neutral-500 leading-relaxed">
									Access registered society bylaws, AGM minutes, and circular records.
								</p>
							</div>
						</div>
						<div class="mt-4 flex items-center justify-end border-t border-neutral-100 pt-3 text-xs font-semibold text-neutral-950">
							<span class="group-hover:underline">Browse Records</span>
							<ArrowRight class="ml-1 h-3.5 w-3.5 transition-transform group-hover:translate-x-1" />
						</div>
					</a>
				{/if}
			</div>
		</section>

		<!-- 5. Recent Announcements (Authentic real data, intentional empty state) -->
		<section class="mt-8 space-y-4" aria-labelledby="recent-announcements-heading">
			<div class="flex items-center justify-between">
				<h2 id="recent-announcements-heading" class="font-serif text-lg font-bold tracking-tight text-[#121212]">
					Recent Announcements
				</h2>
				<a
					href={`/app/societies/${societyId}/announcements`}
					class="text-xs font-semibold text-neutral-900 hover:underline"
				>
					View all notices →
				</a>
			</div>

			{#if recentAnnouncements.length > 0}
				<div class="space-y-3">
					{#each recentAnnouncements as notice (notice.id)}
						<article
							class="rounded-2xl border border-[#e2e2de] bg-white p-5 shadow-xs transition-all hover:border-neutral-300"
						>
							<div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
								<div class="flex items-center gap-2">
									<Badge
										variant={notice.priority === 'URGENT' ? 'destructive' : 'outline'}
										class="text-[10px]"
									>
										{notice.category}
									</Badge>
									<h3 class="text-sm font-semibold text-neutral-950">{notice.title}</h3>
								</div>
								<time class="font-mono text-xs text-neutral-500">
									{new Date(notice.published_at || notice.created_at).toLocaleDateString('en-IN', {
										day: 'numeric',
										month: 'short',
										year: 'numeric'
									})}
								</time>
							</div>
							<p class="mt-2 line-clamp-2 text-xs leading-relaxed text-neutral-600">
								{notice.body}
							</p>
						</article>
					{/each}
				</div>
			{:else}
				<div
					class="rounded-2xl border border-dashed border-[#e2e2de] bg-white/60 p-8 text-center text-xs text-neutral-500"
				>
					<MessageSquare class="mx-auto h-8 w-8 text-neutral-300" />
					<p class="mt-2 font-medium text-neutral-700">No broadcast announcements published yet.</p>
					<p class="mt-0.5 text-neutral-500">
						{#if isSocAdmin}
							Use the Announcements console to broadcast notices or AGM updates to residents.
						{:else}
							Official society circulars and notices will appear here once published.
						{/if}
					</p>
				</div>
			{/if}
		</section>
	{/if}
</div>
