<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Card from '$lib/components/ui/card.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import {
		Shield,
		KeyRound,
		ScrollText,
		Building2,
		ArrowRight,
		CheckCircle2,
		LogIn,
		Terminal,
		Lock
	} from '@lucide/svelte';
</script>

<svelte:head>
	<title>CHS Platform — Cooperative Housing Society Management</title>
	<meta
		name="description"
		content="Modern, secure, and transparent governance platform for Cooperative Housing Societies"
	/>
</svelte:head>

<div class="space-y-16 py-6">
	<!-- Hero Section -->
	<section
		class="relative overflow-hidden rounded-3xl border border-slate-800 bg-gradient-to-b from-slate-900 via-slate-900/90 to-slate-950 p-8 shadow-2xl sm:p-14"
	>
		<div
			class="pointer-events-none absolute -top-24 -right-24 h-96 w-96 rounded-full bg-indigo-500/10 blur-3xl"
		></div>
		<div
			class="pointer-events-none absolute -bottom-24 -left-24 h-96 w-96 rounded-full bg-violet-500/10 blur-3xl"
		></div>

		<div class="relative z-10 max-w-3xl">
			<div
				class="mb-4 inline-flex items-center gap-2 rounded-full border border-indigo-500/30 bg-indigo-500/10 px-3 py-1 text-xs font-medium text-indigo-300"
			>
				<Shield class="h-3.5 w-3.5 text-indigo-400" />
				<span>Enterprise-Grade Housing Society Governance</span>
			</div>

			<h1 class="text-4xl font-extrabold tracking-tight text-white sm:text-5xl lg:text-6xl">
				Modern Governance for <span
					class="bg-gradient-to-r from-indigo-400 to-violet-400 bg-clip-text text-transparent"
					>Residential Societies</span
				>
			</h1>

			<p class="mt-4 text-base leading-relaxed text-slate-300 sm:text-lg">
				A unified platform providing phone-based authentication, role-based society administration,
				immutable audit logging, and digital gatekeeping for Cooperative Housing Societies.
			</p>

			<div class="mt-8 flex flex-wrap items-center gap-4">
				{#if auth.isAuthenticated && auth.user}
					{#if auth.isSuperAdmin}
						<a href="/admin">
							<Button variant="primary" size="lg" class="shadow-lg shadow-indigo-600/30">
								<span>Open Super Admin Console</span>
								<ArrowRight class="ml-1 h-4 w-4" />
							</Button>
						</a>
					{/if}

					{#if auth.user?.society_assignments && auth.user.society_assignments.length > 0}
						<a href={`/society/${auth.user.society_assignments[0].society_id}`}>
							<Button
								variant={auth.isSuperAdmin ? 'outline' : 'primary'}
								size="lg"
								class={auth.isSuperAdmin
									? 'border-indigo-500/40 text-indigo-300 hover:bg-indigo-500/10'
									: 'shadow-lg shadow-indigo-600/30'}
							>
								<Building2 class="mr-2 h-4 w-4 text-indigo-400" />
								<span>Society Console ({auth.user.society_assignments.length})</span>
								<ArrowRight class="ml-1 h-4 w-4" />
							</Button>
						</a>
					{/if}

					{#if !auth.isSuperAdmin && (!auth.user?.society_assignments || auth.user.society_assignments.length === 0)}
						<div class="flex items-center gap-3">
							<Badge variant="success" class="px-3 py-1 text-xs">
								Signed in as {auth.user.phone}
							</Badge>
							<a href="/login">
								<Button variant="outline" size="md">Manage Account</Button>
							</a>
						</div>
					{/if}
				{:else}
					<a href="/login">
						<Button variant="primary" size="lg" class="shadow-lg shadow-indigo-600/30">
							<LogIn class="mr-2 h-4 w-4" />
							<span>Sign In to Platform</span>
							<ArrowRight class="ml-1 h-4 w-4" />
						</Button>
					</a>
				{/if}

				<a href="/developer">
					<Button variant="outline" size="lg">
						<Terminal class="mr-2 h-4 w-4 text-slate-400" />
						<span>Developer Diagnostics</span>
					</Button>
				</a>
			</div>
		</div>
	</section>

	<!-- Platform Core Capabilities -->
	<section class="space-y-6">
		<div class="mx-auto max-w-xl text-center">
			<h2 class="text-2xl font-bold tracking-tight text-slate-100 sm:text-3xl">
				Engineered for Trust & Compliance
			</h2>
			<p class="mt-2 text-xs leading-relaxed text-slate-400">
				Built with strict documentation-first discipline, cryptographic authorization, and
				transactional consistency.
			</p>
		</div>

		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
			<!-- Feature 1 -->
			<Card
				class="flex flex-col justify-between transition-all hover:border-slate-700 hover:bg-slate-900"
			>
				<div>
					<div
						class="mb-4 flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-400"
					>
						<KeyRound class="h-5 w-5" />
					</div>
					<h3 class="text-base font-semibold text-slate-100">Universal Identity (S01)</h3>
					<p class="mt-2 text-xs leading-relaxed text-slate-400">
						Phone-first OTP authentication with canonical E.164 normalization, Ed25519 JWT
						authority, and strict multi-device session concurrency limits.
					</p>
				</div>
				<div
					class="mt-4 flex items-center gap-1.5 border-t border-slate-800/80 pt-4 text-[11px] font-medium text-emerald-400"
				>
					<CheckCircle2 class="h-3.5 w-3.5" />
					<span>Production Verified</span>
				</div>
			</Card>

			<!-- Feature 2 -->
			<Card
				class="flex flex-col justify-between transition-all hover:border-slate-700 hover:bg-slate-900"
			>
				<div>
					<div
						class="mb-4 flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-400"
					>
						<Shield class="h-5 w-5" />
					</div>
					<h3 class="text-base font-semibold text-slate-100">Platform Administration (S02)</h3>
					<p class="mt-2 text-xs leading-relaxed text-slate-400">
						Isolated Super Admin bootstrapping, role-based platform authorization middleware,
						administrator lifecycle control, and immediate session termination on suspension.
					</p>
				</div>
				<div
					class="mt-4 flex items-center gap-1.5 border-t border-slate-800/80 pt-4 text-[11px] font-medium text-emerald-400"
				>
					<CheckCircle2 class="h-3.5 w-3.5" />
					<span>Production Verified</span>
				</div>
			</Card>

			<!-- Feature 3 -->
			<Card
				class="flex flex-col justify-between transition-all hover:border-slate-700 hover:bg-slate-900"
			>
				<div>
					<div
						class="mb-4 flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-400"
					>
						<ScrollText class="h-5 w-5" />
					</div>
					<h3 class="text-base font-semibold text-slate-100">Immutable Audit Trail</h3>
					<p class="mt-2 text-xs leading-relaxed text-slate-400">
						Append-only administrative event ledger backed by database triggers and atomic
						transactional boundaries ensuring zero untracked mutations.
					</p>
				</div>
				<div
					class="mt-4 flex items-center gap-1.5 border-t border-slate-800/80 pt-4 text-[11px] font-medium text-emerald-400"
				>
					<CheckCircle2 class="h-3.5 w-3.5" />
					<span>Production Verified</span>
				</div>
			</Card>

			<!-- Feature 4 -->
			<Card class="flex flex-col justify-between opacity-80">
				<div>
					<div
						class="mb-4 flex h-10 w-10 items-center justify-center rounded-xl bg-slate-800 text-slate-400"
					>
						<Building2 class="h-5 w-5" />
					</div>
					<h3 class="text-base font-semibold text-slate-200">Society Onboarding (S03)</h3>
					<p class="mt-2 text-xs leading-relaxed text-slate-400">
						Multi-society registration, committee verification workflows, state transitions (PENDING
						&rarr; ACTIVE), and society-level administrative boundaries.
					</p>
				</div>
				<div
					class="mt-4 flex items-center gap-1.5 border-t border-slate-800/80 pt-4 text-[11px] text-slate-500"
				>
					<span>Upcoming Sprint</span>
				</div>
			</Card>

			<!-- Feature 5 -->
			<Card class="flex flex-col justify-between opacity-80">
				<div>
					<div
						class="mb-4 flex h-10 w-10 items-center justify-center rounded-xl bg-slate-800 text-slate-400"
					>
						<Lock class="h-5 w-5" />
					</div>
					<h3 class="text-base font-semibold text-slate-200">Members & Gate Operations</h3>
					<p class="mt-2 text-xs leading-relaxed text-slate-400">
						Granular unit relationships (owners, tenants, family), designated society roles
						(Secretary, Treasurer), and real-time digital visitor verification.
					</p>
				</div>
				<div
					class="mt-4 flex items-center gap-1.5 border-t border-slate-800/80 pt-4 text-[11px] text-slate-500"
				>
					<span>Upcoming Sprint</span>
				</div>
			</Card>

			<!-- Feature 6 -->
			<Card class="flex flex-col justify-between opacity-80">
				<div>
					<div
						class="mb-4 flex h-10 w-10 items-center justify-center rounded-xl bg-slate-800 text-slate-400"
					>
						<Terminal class="h-5 w-5" />
					</div>
					<h3 class="text-base font-semibold text-slate-200">Financial Maintenance & Restro</h3>
					<p class="mt-2 text-xs leading-relaxed text-slate-400">
						Automated maintenance billing, member balance ledgers, amenity reservations, and society
						club/restaurant operations.
					</p>
				</div>
				<div
					class="mt-4 flex items-center gap-1.5 border-t border-slate-800/80 pt-4 text-[11px] text-slate-500"
				>
					<span>Upcoming Sprint</span>
				</div>
			</Card>
		</div>
	</section>

	<!-- Footer Callout -->
	<section class="rounded-2xl border border-slate-800 bg-slate-900/60 p-8 text-center sm:p-10">
		<h2 class="text-lg font-semibold text-slate-100 sm:text-xl">
			Ready to access platform administration?
		</h2>
		<p class="mx-auto mt-2 max-w-md text-xs text-slate-400">
			Sign in with your configured Super Admin credentials to access the administrative dashboard.
		</p>
		<div class="mt-6 flex justify-center gap-3">
			<a href="/login">
				<Button variant="primary" size="md">
					<span>Go to Login</span>
					<ArrowRight class="ml-1 h-4 w-4" />
				</Button>
			</a>
		</div>
	</section>
</div>
