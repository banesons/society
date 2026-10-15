<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Card from '$lib/components/ui/card.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import AdminRoster from '$lib/components/admin/admin-roster.svelte';
	import AuditViewer from '$lib/components/admin/audit-viewer.svelte';
	import CreateAdminDialog from '$lib/components/admin/create-admin-dialog.svelte';
	import LogoutAllDialog from '$lib/components/auth/logout-all-dialog.svelte';
	import SocietyRoster from '$lib/components/admin/society-roster.svelte';
	import RegisterSocietyDialog from '$lib/components/admin/register-society-dialog.svelte';
	import PlatformDashboard from '$lib/components/dashboard/platform-dashboard.svelte';
	import {
		Shield,
		ShieldAlert,
		Users,
		ScrollText,
		LogIn,
		Lock,
		Loader2,
		ArrowLeft,
		Settings,
		KeyRound,
		Smartphone,
		LogOut,
		CheckCircle2,
		Building2,
		TrendingUp
	} from '@lucide/svelte';

	let activeTab = $state<'overview' | 'societies' | 'roster' | 'audit' | 'sessions'>('overview');
	let isCreateDialogOpen = $state(false);
	let isLogoutAllDialogOpen = $state(false);
	let isRegisterSocietyOpen = $state(false);

	let rosterRef = $state<AdminRoster | null>(null);
	let societyRosterRef = $state<SocietyRoster | null>(null);
	let auditRef = $state<AuditViewer | null>(null);

	function handleAdminCreated() {
		if (rosterRef) {
			rosterRef.refresh();
		}
		if (auditRef) {
			auditRef.refresh();
		}
	}
</script>

<svelte:head>
	<title>Platform Administration — CHS Platform</title>
	<meta
		name="description"
		content="Super Admin platform identity and administrative management console"
	/>
</svelte:head>

<div class="space-y-6">
	<!-- Top Navigation Back Link -->
	<div class="flex items-center justify-between">
		<a
			href="/"
			class="inline-flex items-center gap-1.5 text-xs font-medium text-slate-400 transition-colors hover:text-slate-200"
		>
			<ArrowLeft class="h-3.5 w-3.5" />
			<span>Return to Main Dashboard</span>
		</a>

		<div class="flex items-center gap-2 text-xs">
			<span class="text-neutral-500">Security Domain:</span>
			<Badge variant="outline" class="font-mono text-[11px] text-neutral-700">
				Platform Layer (S02)
			</Badge>
		</div>
	</div>

	<!-- State 1: Authentication / Profile Loading -->
	{#if auth.isLoading}
		<Card
			class="flex flex-col items-center justify-center border-neutral-200 bg-white/90 p-16 text-center shadow-xs"
		>
			<Loader2 class="h-8 w-8 animate-spin text-neutral-900" />
			<p class="mt-4 font-serif text-sm font-medium text-neutral-950">
				Verifying administrative authorization...
			</p>
			<p class="mt-1 text-xs text-neutral-500">
				Checking Ed25519 session credentials against server context
			</p>
		</Card>

		<!-- State 2: Unauthenticated -->
	{:else if !auth.isAuthenticated}
		<Card
			class="mx-auto flex max-w-lg flex-col items-center justify-center border-neutral-200 bg-white/90 p-12 text-center shadow-xs"
		>
			<div
				class="flex h-14 w-14 items-center justify-center rounded-2xl bg-neutral-100 text-neutral-900 shadow-xs"
			>
				<Lock class="h-7 w-7" />
			</div>
			<h1 class="mt-4 font-serif text-xl font-medium tracking-tight text-neutral-950">
				Authentication Required
			</h1>
			<p class="mt-2 text-xs leading-relaxed text-neutral-600">
				The Platform Administration console is restricted. You must sign in with a verified Super
				Admin mobile identity to access administrative rosters and audit ledgers.
			</p>
			<Button variant="primary" size="md" class="mt-6" onclick={() => auth.openLogin()}>
				<LogIn class="mr-2 h-4 w-4" />
				<span>Sign In to Platform</span>
			</Button>
		</Card>

		<!-- State 3: Authenticated but NOT Super Admin (Ordinary User or Regular Admin) -->
	{:else if !auth.isSuperAdmin}
		<Card
			class="mx-auto flex max-w-lg flex-col items-center justify-center border-rose-200 bg-white/90 p-12 text-center shadow-xs"
		>
			<div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-rose-50 text-rose-600">
				<ShieldAlert class="h-7 w-7" />
			</div>
			<h1 class="mt-4 font-serif text-xl font-medium tracking-tight text-neutral-950">
				Super Admin Privileges Required
			</h1>
			<p class="mt-2 text-xs leading-relaxed text-neutral-600">
				Authenticated identity <span class="font-mono font-medium text-neutral-900"
					>{auth.user?.phone}</span
				>
				does not hold an active
				<code class="rounded bg-neutral-100 px-1.5 py-0.5 text-neutral-900">SUPER_ADMIN</code> platform
				role. Administrative provisioning and audit logging are restricted.
			</p>
			<div class="mt-6 flex items-center gap-3">
				<a href="/">
					<Button variant="outline" size="sm">Back to Dashboard</Button>
				</a>
				<Button variant="ghost" size="sm" onclick={() => auth.logout()}>Switch Account</Button>
			</div>
		</Card>

		<!-- State 4: Active SUPER_ADMIN -->
	{:else}
		<!-- Header Banner -->
		<div class="rounded-3xl border border-neutral-200 bg-white p-6 shadow-sm sm:p-8">
			<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
				<div>
					<div class="flex items-center gap-2.5">
						<div
							class="flex h-10 w-10 items-center justify-center rounded-full bg-neutral-950 text-white shadow-xs"
						>
							<Shield class="h-5 w-5" />
						</div>
						<div>
							<h1 class="text-xl font-bold tracking-tight text-neutral-950 sm:text-2xl">
								Platform Administration
							</h1>
							<p class="mt-0.5 text-xs text-neutral-500">
								Super Admin management console · Identity bootstrap, roles, and immutable audit logs
							</p>
						</div>
					</div>
				</div>

				<div class="flex items-center gap-2.5">
					<div
						class="flex items-center gap-2 rounded-full border border-neutral-200 bg-neutral-50 px-3 py-1.5 font-mono text-xs text-neutral-800"
					>
						<span class="inline-block h-2 w-2 animate-pulse rounded-full bg-emerald-500"></span>
						<span>{auth.user?.phone}</span>
						<Badge variant="default" class="ml-1 px-2 py-0 text-[10px]">SUPER_ADMIN</Badge>
					</div>
				</div>
			</div>

			<!-- Tab Navigation -->
			<div class="mt-6 flex gap-6 border-b border-neutral-200 text-sm">
				<button
					type="button"
					onclick={() => (activeTab = 'overview')}
					class="flex cursor-pointer items-center gap-2 border-b-2 pb-3 font-medium transition-colors {activeTab ===
					'overview'
						? 'border-neutral-950 text-neutral-950'
						: 'border-transparent text-neutral-500 hover:text-neutral-900'}"
				>
					<TrendingUp class="h-4 w-4" />
					<span>Ecosystem Overview</span>
				</button>

				<button
					type="button"
					onclick={() => (activeTab = 'societies')}
					class="flex cursor-pointer items-center gap-2 border-b-2 pb-3 font-medium transition-colors {activeTab ===
					'societies'
						? 'border-neutral-950 text-neutral-950'
						: 'border-transparent text-neutral-500 hover:text-neutral-900'}"
				>
					<Building2 class="h-4 w-4" />
					<span>Societies</span>
				</button>

				<button
					type="button"
					onclick={() => (activeTab = 'roster')}
					class="flex cursor-pointer items-center gap-2 border-b-2 pb-3 font-medium transition-colors {activeTab ===
					'roster'
						? 'border-neutral-950 text-neutral-950'
						: 'border-transparent text-neutral-500 hover:text-neutral-900'}"
				>
					<Users class="h-4 w-4" />
					<span>Administrators</span>
				</button>

				<button
					type="button"
					onclick={() => (activeTab = 'audit')}
					class="flex cursor-pointer items-center gap-2 border-b-2 pb-3 font-medium transition-colors {activeTab ===
					'audit'
						? 'border-neutral-950 text-neutral-950'
						: 'border-transparent text-neutral-500 hover:text-neutral-900'}"
				>
					<ScrollText class="h-4 w-4" />
					<span>Audit Trail</span>
				</button>

				<button
					type="button"
					onclick={() => (activeTab = 'sessions')}
					class="flex cursor-pointer items-center gap-2 border-b-2 pb-3 font-medium transition-colors {activeTab ===
					'sessions'
						? 'border-neutral-950 text-neutral-950'
						: 'border-transparent text-neutral-500 hover:text-neutral-900'}"
				>
					<Settings class="h-4 w-4" />
					<span>Account & Sessions</span>
				</button>
			</div>
		</div>

		<!-- Tab Contents -->
		<Card class="p-6">
			{#if activeTab === 'overview'}
				<PlatformDashboard />
			{:else if activeTab === 'societies'}
				<SocietyRoster
					bind:this={societyRosterRef}
					onOpenRegister={() => (isRegisterSocietyOpen = true)}
				/>
			{:else if activeTab === 'roster'}
				<AdminRoster bind:this={rosterRef} onOpenCreate={() => (isCreateDialogOpen = true)} />
			{:else if activeTab === 'audit'}
				<AuditViewer bind:this={auditRef} />
			{:else}
				<!-- Sessions & Account Settings -->
				<div class="space-y-6">
					<div>
						<h2 class="text-base font-semibold text-neutral-950">Super Admin Account & Sessions</h2>
						<p class="mt-1 text-xs text-neutral-500">
							Manage platform credentials and active multi-device sessions
						</p>
					</div>

					<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
						<!-- Identity Card -->
						<div class="space-y-3 rounded-2xl border border-neutral-200 bg-neutral-50 p-4">
							<div class="flex items-center gap-2 text-xs font-semibold text-neutral-800">
								<KeyRound class="h-4 w-4 text-neutral-950" />
								<span>Identity Information</span>
							</div>
							<div class="space-y-2 text-xs">
								<div class="flex justify-between border-b border-neutral-200 py-1">
									<span class="text-neutral-500">Mobile Number:</span>
									<span class="font-mono font-medium text-neutral-950">{auth.user?.phone}</span>
								</div>
								<div class="flex justify-between border-b border-neutral-200 py-1">
									<span class="text-neutral-500">Platform Role:</span>
									<span class="font-semibold text-neutral-950">SUPER_ADMIN</span>
								</div>
								<div class="flex justify-between py-1">
									<span class="text-neutral-500">Status:</span>
									<span class="font-medium text-emerald-700">ACTIVE</span>
								</div>
							</div>
						</div>

						<!-- Session Concurrency Card -->
						<div class="space-y-3 rounded-2xl border border-neutral-200 bg-neutral-50 p-4">
							<div class="flex items-center gap-2 text-xs font-semibold text-neutral-800">
								<Smartphone class="h-4 w-4 text-emerald-700" />
								<span>Session Limits (S01 Policy)</span>
							</div>
							<div class="space-y-2 text-xs leading-relaxed text-neutral-600">
								<p>
									The CHS platform strictly enforces a maximum of <strong class="text-neutral-950"
										>2 concurrent active devices</strong
									> per mobile identity.
								</p>
								<div class="flex items-center gap-2 pt-1 text-[11px] font-medium text-emerald-700">
									<CheckCircle2 class="h-3.5 w-3.5" />
									<span>This Device: Authenticated via Ed25519 JWT</span>
								</div>
							</div>
						</div>
					</div>

					<!-- Session Termination Actions -->
					<div class="space-y-4 rounded-2xl border border-neutral-200 bg-neutral-50 p-5">
						<h3 class="text-sm font-semibold text-neutral-950">Session Termination Controls</h3>
						<div class="flex flex-col gap-3 sm:flex-row">
							<Button variant="secondary" size="sm" onclick={() => auth.logout()}>
								<LogOut class="mr-1.5 h-3.5 w-3.5" />
								<span>Log Out This Device Only</span>
							</Button>

							<Button
								variant="outline"
								size="sm"
								class="border-rose-200 text-rose-700 hover:bg-rose-50"
								onclick={() => (isLogoutAllDialogOpen = true)}
							>
								<ShieldAlert class="mr-1.5 h-3.5 w-3.5" />
								<span>Log Out From All Devices</span>
							</Button>
						</div>
						<p class="text-[11px] text-neutral-500">
							Logging out all devices invalidates all refresh tokens in the database, terminating
							any secondary session.
						</p>
					</div>
				</div>
			{/if}
		</Card>

		<!-- Create Administrator Dialog -->
		<CreateAdminDialog
			isOpen={isCreateDialogOpen}
			onClose={() => (isCreateDialogOpen = false)}
			onSuccess={handleAdminCreated}
		/>

		<!-- Register Housing Society Dialog -->
		<RegisterSocietyDialog
			isOpen={isRegisterSocietyOpen}
			onClose={() => (isRegisterSocietyOpen = false)}
			onSuccess={() => {
				if (societyRosterRef) societyRosterRef.refresh();
				if (auditRef) auditRef.refresh();
			}}
		/>

		<!-- Logout All Devices Dialog -->
		<LogoutAllDialog
			isOpen={isLogoutAllDialogOpen}
			onClose={() => (isLogoutAllDialogOpen = false)}
		/>
	{/if}
</div>
