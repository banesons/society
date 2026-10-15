<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import {
		Shield,
		KeyRound,
		ScrollText,
		Building2,
		ArrowRight,
		CheckCircle2,
		Terminal,
		Lock,
		Users,
		Sliders
	} from '@lucide/svelte';
</script>

<svelte:head>
	<title>chs. — Residential Governance with Accountability</title>
	<meta
		name="description"
		content="Modern, secure, and transparent governance platform for Cooperative Housing Societies"
	/>
</svelte:head>

<div class="space-y-24 py-4 sm:py-8 lg:space-y-36">
	<!-- Hero Section: Editorial Grotesque Architecture -->
	<section class="space-y-12 sm:space-y-16">
		<!-- Massive 3-Line Statement Title -->
		<div class="max-w-6xl">
			<h1
				class="text-[3.25rem] leading-[0.93] font-bold tracking-[-0.04em] text-neutral-950 select-none sm:text-7xl md:text-8xl lg:text-[7.5rem] xl:text-[8.5rem]"
			>
				Residential<br />
				governance<br />
				with accountability.
			</h1>
		</div>

		<!-- Asymmetric Editorial Row (3 Columns: Prompt Link / Photo / Editorial Blurb) -->
		<div class="grid grid-cols-1 items-start gap-8 pt-4 md:grid-cols-12 md:gap-8 lg:gap-12">
			<!-- Left Column: Underlined Topic Link -->
			<div class="pt-2 md:col-span-4 lg:col-span-3">
				<a
					href="#governance"
					class="editorial-link text-sm font-normal tracking-tight text-neutral-950 transition-opacity hover:opacity-70 sm:text-base"
				>
					What is residential governance
				</a>
				<div class="mt-4 hidden md:block">
					<p class="text-xs leading-relaxed text-neutral-500">
						Cooperative Housing Societies (CHS) demand mathematical integrity, strict role
						separation, and zero tolerance for untracked mutations.
					</p>
				</div>
			</div>

			<!-- Center Column: Striking Architectural Photo -->
			<div class="md:col-span-5 lg:col-span-5">
				<div class="group relative overflow-hidden bg-neutral-200">
					<img
						src="/hero-architecture.jpg"
						alt="Architectural residential structure with geometric infrastructure"
						class="h-auto w-full object-cover transition-transform duration-700 ease-out group-hover:scale-[1.03]"
					/>
					<div
						class="absolute bottom-3 left-3 bg-neutral-950/70 px-2.5 py-1 font-mono text-[10px] tracking-wider text-white uppercase backdrop-blur-xs"
					>
						Architecture & Infrastructure
					</div>
				</div>
			</div>

			<!-- Right Column: Editorial Summary Blurb & Actions -->
			<div class="space-y-6 pt-2 md:col-span-3 lg:col-span-4">
				<p class="text-sm leading-relaxed text-neutral-800 sm:text-base">
					At CHS we engineer verifiable, transparent governance for residential communities. Built
					with phone-first cryptographic trust, immutable audit ledgers, and digital gatekeeping.
				</p>

				<!-- Dynamic Action State based on Authentication -->
				<div class="pt-2">
					{#if auth.isAuthenticated && auth.user}
						<div class="space-y-3">
							{#if auth.isSuperAdmin}
								<a href="/admin" class="inline-block">
									<Button variant="primary" size="md">
										<span>Super Admin Console</span>
										<ArrowRight class="h-3.5 w-3.5" />
									</Button>
								</a>
							{/if}

							{#if auth.user?.society_assignments && auth.user.society_assignments.length > 0}
								<a
									href={`/society/${auth.user.society_assignments[0].society_id}`}
									class="inline-block"
								>
									<Button variant={auth.isSuperAdmin ? 'outline' : 'primary'} size="md">
										<Building2 class="h-3.5 w-3.5" />
										<span>Society Console ({auth.user.society_assignments.length})</span>
									</Button>
								</a>
							{/if}

							{#if !auth.isSuperAdmin && (!auth.user?.society_assignments || auth.user.society_assignments.length === 0)}
								<div class="flex items-center gap-3">
									<Badge variant="outline" class="px-3 py-1 text-xs">
										Signed in as {auth.user.phone}
									</Badge>
									<a href="/developer" class="editorial-link text-xs font-medium text-neutral-900">
										Diagnostics &rarr;
									</a>
								</div>
							{/if}
						</div>
					{:else}
						<div class="flex flex-wrap items-center gap-4">
							<a href="/login">
								<Button variant="primary" size="md">
									<span>Sign In to Platform</span>
									<ArrowRight class="h-3.5 w-3.5" />
								</Button>
							</a>
							<a
								href="/developer"
								class="editorial-link text-xs font-medium text-neutral-700 hover:text-neutral-950"
							>
								dev diagnostics
							</a>
						</div>
					{/if}
				</div>
			</div>
		</div>
	</section>

	<!-- Editorial Statement & Metrics -->
	<section id="governance" class="border-t border-neutral-300 pt-16 lg:pt-24">
		<div class="grid grid-cols-1 gap-12 lg:grid-cols-12 lg:gap-16">
			<div class="lg:col-span-5">
				<span class="text-xs font-semibold tracking-wider text-neutral-400 uppercase">
					01 / Foundation
				</span>
				<h2 class="mt-3 text-3xl font-bold tracking-tight text-neutral-950 sm:text-4xl lg:text-5xl">
					Engineered for cryptographic trust.
				</h2>
			</div>
			<div class="space-y-6 text-sm leading-relaxed text-neutral-700 sm:text-base lg:col-span-7">
				<p>
					Traditional housing society administration relies on fragmented messaging apps, informal
					spreadsheets, and unverified paper trails. CHS replaces ambiguity with cryptographic
					verification, role isolation, and deterministic audit trails.
				</p>
				<p>
					Every unit assignment, administrative promotion, and gate entry is anchored by Ed25519
					signatures and governed by strict two-device session limits.
				</p>

				<!-- Editorial Stats Grid -->
				<div class="grid grid-cols-2 gap-6 pt-6 sm:grid-cols-4">
					<div class="border-l border-neutral-300 pl-4">
						<div class="font-mono text-2xl font-bold text-neutral-950 sm:text-3xl">0</div>
						<div class="mt-1 text-xs text-neutral-500">Untracked mutations</div>
					</div>
					<div class="border-l border-neutral-300 pl-4">
						<div class="font-mono text-2xl font-bold text-neutral-950 sm:text-3xl">2</div>
						<div class="mt-1 text-xs text-neutral-500">Max active devices</div>
					</div>
					<div class="border-l border-neutral-300 pl-4">
						<div class="font-mono text-2xl font-bold text-neutral-950 sm:text-3xl">E.164</div>
						<div class="mt-1 text-xs text-neutral-500">Canonical identity</div>
					</div>
					<div class="border-l border-neutral-300 pl-4">
						<div class="font-mono text-2xl font-bold text-neutral-950 sm:text-3xl">100%</div>
						<div class="mt-1 text-xs text-neutral-500">Documentation sync</div>
					</div>
				</div>
			</div>
		</div>
	</section>

	<!-- Capabilities Grid: Minimalist Swiss Layout -->
	<section id="societies" class="border-t border-neutral-300 pt-16 lg:pt-24">
		<div class="flex flex-col justify-between gap-4 md:flex-row md:items-end">
			<div>
				<span class="text-xs font-semibold tracking-wider text-neutral-400 uppercase">
					02 / Platform Disciplines
				</span>
				<h2 class="mt-3 text-3xl font-bold tracking-tight text-neutral-950 sm:text-4xl lg:text-5xl">
					Core Architectural Modules
				</h2>
			</div>
			<p class="max-w-md text-xs text-neutral-600 sm:text-sm">
				Each module operates within modular bounded contexts designed to minimize data exposure.
			</p>
		</div>

		<div class="mt-12 grid grid-cols-1 gap-px bg-neutral-300 md:grid-cols-2 lg:grid-cols-3">
			<!-- Discipline 1: Universal Identity -->
			<div
				class="flex flex-col justify-between bg-[#f3f3f1] p-8 transition-colors hover:bg-white sm:p-10"
			>
				<div>
					<div class="flex items-center justify-between font-mono text-xs text-neutral-400">
						<span>MOD-01</span>
						<KeyRound class="h-4 w-4 text-neutral-800" />
					</div>
					<h3 class="mt-6 text-xl font-bold tracking-tight text-neutral-950">Universal Identity</h3>
					<p class="mt-3 text-xs leading-relaxed text-neutral-600">
						Phone-first authentication utilizing SMS OTP verification, strict E.164 phone
						normalization, and automated FIFO 2-device session concurrency eviction.
					</p>
				</div>
				<div class="mt-8 flex items-center gap-1.5 text-xs font-medium text-neutral-900">
					<CheckCircle2 class="h-3.5 w-3.5 text-emerald-600" />
					<span>Production Verified (S01)</span>
				</div>
			</div>

			<!-- Discipline 2: Platform Administration -->
			<div
				class="flex flex-col justify-between bg-[#f3f3f1] p-8 transition-colors hover:bg-white sm:p-10"
			>
				<div>
					<div class="flex items-center justify-between font-mono text-xs text-neutral-400">
						<span>MOD-02</span>
						<Shield class="h-4 w-4 text-neutral-800" />
					</div>
					<h3 class="mt-6 text-xl font-bold tracking-tight text-neutral-950">
						Platform Governance
					</h3>
					<p class="mt-3 text-xs leading-relaxed text-neutral-600">
						Super Admin bootstrapping, role-based platform authorization middleware, administrator
						lifecycle management, and immediate session termination.
					</p>
				</div>
				<div class="mt-8 flex items-center gap-1.5 text-xs font-medium text-neutral-900">
					<CheckCircle2 class="h-3.5 w-3.5 text-emerald-600" />
					<span>Production Verified (S02)</span>
				</div>
			</div>

			<!-- Discipline 3: Immutable Audit Trail -->
			<div
				class="flex flex-col justify-between bg-[#f3f3f1] p-8 transition-colors hover:bg-white sm:p-10"
			>
				<div>
					<div class="flex items-center justify-between font-mono text-xs text-neutral-400">
						<span>MOD-03</span>
						<ScrollText class="h-4 w-4 text-neutral-800" />
					</div>
					<h3 class="mt-6 text-xl font-bold tracking-tight text-neutral-950">
						Immutable Audit Ledger
					</h3>
					<p class="mt-3 text-xs leading-relaxed text-neutral-600">
						Cryptographically verifiable administrative event trail backed by atomic transactional
						triggers ensuring every status shift is permanently recorded.
					</p>
				</div>
				<div class="mt-8 flex items-center gap-1.5 text-xs font-medium text-neutral-900">
					<CheckCircle2 class="h-3.5 w-3.5 text-emerald-600" />
					<span>Production Verified</span>
				</div>
			</div>

			<!-- Discipline 4: Society Onboarding -->
			<div
				id="members"
				class="flex flex-col justify-between bg-[#f3f3f1] p-8 transition-colors hover:bg-white sm:p-10"
			>
				<div>
					<div class="flex items-center justify-between font-mono text-xs text-neutral-400">
						<span>MOD-04</span>
						<Building2 class="h-4 w-4 text-neutral-800" />
					</div>
					<h3 class="mt-6 text-xl font-bold tracking-tight text-neutral-950">Society Onboarding</h3>
					<p class="mt-3 text-xs leading-relaxed text-neutral-600">
						Multi-society registration, committee verification workflows, state transitions (PENDING
						&rarr; ACTIVE), and society-level administrative boundaries.
					</p>
				</div>
				<div class="mt-8 flex items-center gap-1.5 text-xs font-medium text-neutral-900">
					<CheckCircle2 class="h-3.5 w-3.5 text-emerald-600" />
					<span>Production Verified (S03)</span>
				</div>
			</div>

			<!-- Discipline 5: Members & Gatekeeping -->
			<div
				id="gatekeeping"
				class="flex flex-col justify-between bg-[#f3f3f1] p-8 transition-colors hover:bg-white sm:p-10"
			>
				<div>
					<div class="flex items-center justify-between font-mono text-xs text-neutral-400">
						<span>MOD-05</span>
						<Users class="h-4 w-4 text-neutral-800" />
					</div>
					<h3 class="mt-6 text-xl font-bold tracking-tight text-neutral-950">
						Members & Gatekeeping
					</h3>
					<p class="mt-3 text-xs leading-relaxed text-neutral-600">
						Granular unit relationships (owners, tenants, family), designated committee roles
						(Secretary, Treasurer), and real-time digital visitor verification.
					</p>
				</div>
				<div class="mt-8 flex items-center gap-1.5 text-xs font-medium text-neutral-500">
					<Sliders class="h-3.5 w-3.5" />
					<span>Active Module</span>
				</div>
			</div>

			<!-- Discipline 6: Financial Compliance -->
			<div
				id="compliance"
				class="flex flex-col justify-between bg-[#f3f3f1] p-8 transition-colors hover:bg-white sm:p-10"
			>
				<div>
					<div class="flex items-center justify-between font-mono text-xs text-neutral-400">
						<span>MOD-06</span>
						<Lock class="h-4 w-4 text-neutral-800" />
					</div>
					<h3 class="mt-6 text-xl font-bold tracking-tight text-neutral-950">
						Financial Ledgers & Maintenance
					</h3>
					<p class="mt-3 text-xs leading-relaxed text-neutral-600">
						Automated maintenance billing schedules, member balance ledgers, amenity reservations,
						and club restaurant charging compliance.
					</p>
				</div>
				<div class="mt-8 flex items-center gap-1.5 text-xs font-medium text-neutral-500">
					<Sliders class="h-3.5 w-3.5" />
					<span>Active Module</span>
				</div>
			</div>
		</div>
	</section>

	<!-- Editorial Callout Section -->
	<section id="about" class="border-t border-neutral-300 pt-16 lg:pt-24">
		<div
			class="flex flex-col justify-between gap-8 rounded-3xl bg-neutral-950 p-10 text-white sm:p-16 lg:flex-row lg:items-center"
		>
			<div class="max-w-2xl space-y-4">
				<div class="font-mono text-xs tracking-widest text-neutral-400 uppercase">
					Documentation-First Standard
				</div>
				<h2 class="text-3xl font-bold tracking-tight sm:text-4xl lg:text-5xl">
					Experience residential governance designed for modern communities.
				</h2>
				<p class="text-xs leading-relaxed text-neutral-300 sm:text-sm">
					Explore live diagnostics, verify session tokens, or sign in to your society console.
				</p>
			</div>

			<div class="flex flex-wrap items-center gap-4">
				{#if !auth.isAuthenticated}
					<a href="/login">
						<Button
							variant="secondary"
							size="lg"
							class="bg-white text-neutral-950 hover:bg-neutral-100"
						>
							<span>Sign In Now</span>
							<ArrowRight class="h-4 w-4" />
						</Button>
					</a>
				{:else}
					<a href="/admin">
						<Button
							variant="secondary"
							size="lg"
							class="bg-white text-neutral-950 hover:bg-neutral-100"
						>
							<span>Admin Console</span>
							<ArrowRight class="h-4 w-4" />
						</Button>
					</a>
				{/if}

				<a href="/developer">
					<Button
						variant="outline"
						size="lg"
						class="border-neutral-700 bg-transparent text-white hover:bg-neutral-800"
					>
						<Terminal class="h-4 w-4 text-neutral-400" />
						<span>Diagnostics</span>
					</Button>
				</a>
			</div>
		</div>
	</section>
</div>
