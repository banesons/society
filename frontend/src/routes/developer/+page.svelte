<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { api } from '$lib/api/client';
	import { auth } from '$lib/auth.svelte';
	import Card from '$lib/components/ui/card.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import {
		Server,
		Database,
		Layout,
		ShieldCheck,
		RefreshCw,
		CheckCircle2,
		UserCheck,
		Smartphone,
		KeyRound,
		RotateCw,
		Terminal
	} from '@lucide/svelte';

	import { browser, dev } from '$app/environment';

	interface HealthData {
		status: string;
		version: string;
		database: {
			connected: boolean;
			latency_ms?: number;
			message?: string;
		};
	}

	const healthQuery = createQuery({
		queryKey: ['system-health'],
		queryFn: () => api.get<HealthData>('/health'),
		refetchInterval: 20000,
		enabled: browser
	});

	function handleRefresh() {
		$healthQuery.refetch();
	}

	// Interactive Test State
	let testPhone = $state('+919967089101');
	let testOtp = $state('123456');
	let testLoading = $state(false);
	let testCooldown = $state(0);
	let testTimer: ReturnType<typeof setInterval> | null = null;
	let lastResponse = $state<{
		action: string;
		status: number;
		timestamp: string;
		data: unknown;
	} | null>(null);

	function startCooldown(sec: number) {
		testCooldown = sec;
		if (testTimer) clearInterval(testTimer);
		testTimer = setInterval(() => {
			if (testCooldown > 0) {
				testCooldown -= 1;
			} else if (testTimer) {
				clearInterval(testTimer);
			}
		}, 1000);
	}

	async function runTestSendOTP() {
		testLoading = true;
		try {
			const res = await api.post<{ resend_after_seconds: number }>('/auth/otp/send', {
				phone: testPhone
			});
			startCooldown(res.resend_after_seconds);
			lastResponse = {
				action: 'POST /api/v1/auth/otp/send',
				status: 200,
				timestamp: new Date().toLocaleTimeString(),
				data: res
			};
		} catch (err: unknown) {
			const e = err as { statusCode?: number; status?: number; message?: string; code?: string };
			lastResponse = {
				action: 'POST /api/v1/auth/otp/send',
				status: e.statusCode || e.status || 400,
				timestamp: new Date().toLocaleTimeString(),
				data: { error: e.message, code: e.code || 'UNKNOWN_ERROR' }
			};
		} finally {
			testLoading = false;
		}
	}

	async function runTestVerifyOTP() {
		testLoading = true;
		try {
			await auth.verifyOTP(testPhone, testOtp);
			lastResponse = {
				action: 'POST /api/v1/auth/otp/verify',
				status: 200,
				timestamp: new Date().toLocaleTimeString(),
				data: {
					message: 'Authenticated successfully',
					user: auth.user,
					token_type: 'Bearer',
					session_limit: 'Max 2 Active Devices'
				}
			};
		} catch (err: unknown) {
			const e = err as { statusCode?: number; status?: number; message?: string; code?: string };
			lastResponse = {
				action: 'POST /api/v1/auth/otp/verify',
				status: e.statusCode || e.status || 400,
				timestamp: new Date().toLocaleTimeString(),
				data: { error: e.message, code: e.code || 'VERIFICATION_FAILED' }
			};
		} finally {
			testLoading = false;
		}
	}

	async function runFetchMe() {
		testLoading = true;
		try {
			await auth.fetchCurrentUser();
			lastResponse = {
				action: 'GET /api/v1/auth/me',
				status: 200,
				timestamp: new Date().toLocaleTimeString(),
				data: auth.user
			};
		} catch (err: unknown) {
			const e = err as { status?: number; message?: string };
			lastResponse = {
				action: 'GET /api/v1/auth/me',
				status: e.status || 401,
				timestamp: new Date().toLocaleTimeString(),
				data: { error: e.message }
			};
		} finally {
			testLoading = false;
		}
	}

	async function runRefreshSession() {
		testLoading = true;
		try {
			const newToken = await auth.refreshSession();
			lastResponse = {
				action: 'POST /api/v1/auth/refresh',
				status: 200,
				timestamp: new Date().toLocaleTimeString(),
				data: {
					message: 'Refresh token rotated successfully',
					token_truncated: newToken ? `${newToken.substring(0, 24)}...` : null
				}
			};
		} catch (err: unknown) {
			const e = err as { status?: number; message?: string };
			lastResponse = {
				action: 'POST /api/v1/auth/refresh',
				status: e.status || 401,
				timestamp: new Date().toLocaleTimeString(),
				data: { error: e.message }
			};
		} finally {
			testLoading = false;
		}
	}

	async function runLogout() {
		testLoading = true;
		try {
			await auth.logout();
			lastResponse = {
				action: 'POST /api/v1/auth/logout',
				status: 200,
				timestamp: new Date().toLocaleTimeString(),
				data: { message: 'Current device session revoked successfully' }
			};
		} catch (err: unknown) {
			const e = err as { message?: string };
			lastResponse = {
				action: 'POST /api/v1/auth/logout',
				status: 500,
				timestamp: new Date().toLocaleTimeString(),
				data: { error: e.message }
			};
		} finally {
			testLoading = false;
		}
	}

	async function runLogoutAll() {
		testLoading = true;
		try {
			await auth.logoutAll();
			lastResponse = {
				action: 'POST /api/v1/auth/logout-all',
				status: 200,
				timestamp: new Date().toLocaleTimeString(),
				data: { message: 'All active user sessions revoked successfully' }
			};
		} catch (err: unknown) {
			const e = err as { message?: string };
			lastResponse = {
				action: 'POST /api/v1/auth/logout-all',
				status: 500,
				timestamp: new Date().toLocaleTimeString(),
				data: { error: e.message }
			};
		} finally {
			testLoading = false;
		}
	}
</script>

<svelte:head>
	<title>Developer Diagnostics — CHS Platform</title>
	<meta
		name="description"
		content="Development, API diagnostics, and authentication testing console"
	/>
</svelte:head>

<div class="space-y-8">
	{#if !dev}
		<div class="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 text-xs text-amber-300">
			<strong>Notice:</strong> Developer diagnostics and interactive test shortcuts are restricted in
			production environments.
		</div>
	{/if}

	<!-- Hero / Overview Banner -->
	<div
		class="relative overflow-hidden rounded-2xl border border-slate-800 bg-gradient-to-b from-slate-900 to-slate-950 p-8 shadow-2xl"
	>
		<div class="absolute -top-16 -right-16 h-64 w-64 rounded-full bg-indigo-500/10 blur-3xl"></div>
		<div class="relative z-10 flex flex-col justify-between gap-6 md:flex-row md:items-center">
			<div>
				<div class="mb-3 inline-flex items-center gap-2">
					<Badge variant="default">Sprint S01/S02</Badge>
					<span class="text-xs text-slate-400">Developer Diagnostics & API Testing</span>
					{#if dev}
						<Badge variant="success" class="text-[10px]">LOCAL DEV</Badge>
					{/if}
				</div>
				<h1 class="text-3xl font-bold tracking-tight text-white sm:text-4xl">
					CHS Platform Diagnostics
				</h1>
				<p class="mt-2 max-w-2xl text-sm leading-relaxed text-slate-400">
					Go-managed OTP generation, canonical E.164 normalization, Ed25519 JWT authority, salted
					SHA-256 session state, and strict 2-device concurrency enforcement.
				</p>
			</div>
			<div class="flex items-center gap-3">
				{#if auth.isAuthenticated}
					<Button variant="outline" size="md" onclick={() => auth.logout()}>
						<span>Sign Out</span>
					</Button>
				{:else}
					<Button variant="primary" size="md" onclick={() => auth.openLogin()}>
						<Smartphone class="mr-1.5 h-4 w-4" />
						<span>Sign In with Phone</span>
					</Button>
				{/if}
				<Button
					variant="outline"
					size="md"
					onclick={handleRefresh}
					disabled={$healthQuery.isFetching}
				>
					<RefreshCw class={`h-4 w-4 ${$healthQuery.isFetching ? 'animate-spin' : ''}`} />
					<span>{$healthQuery.isFetching ? 'Pinging...' : 'Verify Foundation'}</span>
				</Button>
			</div>
		</div>
	</div>

	<!-- Identity & Authentication Live Test Console -->
	<div class="grid grid-cols-1 gap-6 lg:grid-cols-12">
		<!-- Left: Session State & Actions -->
		<div class="space-y-6 lg:col-span-7">
			<Card class="space-y-6">
				<div class="flex items-center justify-between border-b border-slate-800/80 pb-4">
					<div class="flex items-center gap-3">
						<div
							class="flex h-10 w-10 items-center justify-center rounded-lg border border-indigo-500/20 bg-indigo-500/10 text-indigo-400"
						>
							{#if auth.isAuthenticated}
								<UserCheck class="h-5 w-5 text-emerald-400" />
							{:else}
								<Smartphone class="h-5 w-5 text-indigo-400" />
							{/if}
						</div>
						<div>
							<h3 class="text-sm font-semibold text-slate-100">Authentication & Session State</h3>
							<p class="font-mono text-xs text-slate-400">
								{auth.isAuthenticated ? 'Active User Session' : 'Unauthenticated Client'}
							</p>
						</div>
					</div>
					<div>
						{#if auth.isAuthenticated}
							<Badge variant="success">Authenticated</Badge>
						{:else}
							<Badge variant="outline">Unauthenticated</Badge>
						{/if}
					</div>
				</div>

				{#if auth.isAuthenticated && auth.user}
					<!-- Authenticated Profile Card -->
					<div class="space-y-4 rounded-xl border border-slate-800 bg-slate-950/60 p-5">
						<div class="grid grid-cols-1 gap-4 text-xs sm:grid-cols-3">
							<div>
								<span class="mb-1 block text-slate-400">Masked Phone</span>
								<span class="font-mono text-sm font-medium text-slate-100">{auth.user.phone}</span>
							</div>
							<div>
								<span class="mb-1 block text-slate-400">User UUID</span>
								<span
									class="block truncate font-mono text-[11px] text-slate-300"
									title={auth.user.id}
								>
									{auth.user.id}
								</span>
							</div>
							<div>
								<span class="mb-1 block text-slate-400">Account Status</span>
								<Badge variant="success">{auth.user.status}</Badge>
							</div>
						</div>

						<div
							class="flex flex-wrap gap-4 border-t border-slate-800/80 pt-3 text-[11px] text-slate-400"
						>
							<span>Token Type: <strong class="text-slate-200">Ed25519 Bearer (15m)</strong></span>
							<span
								>Refresh Policy: <strong class="text-slate-200">Single-Use Rotation (30d)</strong
								></span
							>
							<span>Device Cap: <strong class="text-indigo-400">Max 2 Active</strong></span>
						</div>
					</div>

					<!-- Authenticated Action Triggers -->
					<div class="space-y-2">
						<span class="block text-xs font-medium text-slate-300">Session Operations</span>
						<div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
							<Button
								variant="outline"
								size="sm"
								onclick={runFetchMe}
								disabled={testLoading}
								class="w-full text-xs"
							>
								<span>GET /auth/me</span>
							</Button>
							<Button
								variant="outline"
								size="sm"
								onclick={runRefreshSession}
								disabled={testLoading}
								class="w-full text-xs"
							>
								<span>Rotate Token</span>
							</Button>
							<Button
								variant="outline"
								size="sm"
								onclick={runLogout}
								disabled={testLoading}
								class="w-full text-xs text-amber-300 hover:text-amber-200"
							>
								<span>Logout Device</span>
							</Button>
							<Button
								variant="outline"
								size="sm"
								onclick={runLogoutAll}
								disabled={testLoading}
								class="w-full text-xs text-rose-300 hover:text-rose-200"
							>
								<span>Logout All</span>
							</Button>
						</div>
					</div>
				{:else}
					<!-- Unauthenticated Direct Test Form -->
					<div class="space-y-4">
						<p class="text-xs leading-relaxed text-slate-400">
							Test the S01 identity pipeline directly with configured development/test numbers:
						</p>

						<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
							<div>
								<label for="test-phone" class="mb-1 block text-xs font-medium text-slate-300">
									Mobile Number (E.164)
								</label>
								<input
									id="test-phone"
									type="text"
									bind:value={testPhone}
									class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 font-mono text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none"
								/>
								<div class="mt-1 flex gap-1.5">
									<button
										type="button"
										onclick={() => {
											testPhone = '+919967089101';
										}}
										class="font-mono text-[10px] text-indigo-400 hover:underline"
									>
										+919967089101
									</button>
									<span class="text-[10px] text-slate-600">·</span>
									<button
										type="button"
										onclick={() => {
											testPhone = '+918422862001';
										}}
										class="font-mono text-[10px] text-indigo-400 hover:underline"
									>
										+918422862001
									</button>
								</div>
							</div>

							<div>
								<label for="test-otp" class="mb-1 block text-xs font-medium text-slate-300">
									OTP Passcode
								</label>
								<input
									id="test-otp"
									type="text"
									bind:value={testOtp}
									maxlength={6}
									class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-2 font-mono text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none"
								/>
								<span class="mt-1 block text-[10px] text-slate-500">Dev OTP: 123456</span>
							</div>
						</div>

						<div class="flex gap-2 pt-2">
							<Button
								variant="primary"
								size="sm"
								onclick={runTestSendOTP}
								disabled={testLoading || testCooldown > 0}
								class="flex-1"
							>
								{#if testLoading}
									<RotateCw class="mr-1 h-3.5 w-3.5 animate-spin" />
									<span>Processing...</span>
								{:else if testCooldown > 0}
									<span>Cooldown ({testCooldown}s)</span>
								{:else}
									<Smartphone class="mr-1 h-3.5 w-3.5" />
									<span>Send OTP</span>
								{/if}
							</Button>

							<Button
								variant="outline"
								size="sm"
								onclick={runTestVerifyOTP}
								disabled={testLoading}
								class="flex-1"
							>
								<KeyRound class="mr-1 h-3.5 w-3.5 text-emerald-400" />
								<span>Verify & Authenticate</span>
							</Button>
						</div>
					</div>
				{/if}
			</Card>
		</div>

		<!-- Right: Live API Inspection Terminal -->
		<div class="lg:col-span-5">
			<Card class="flex h-full flex-col justify-between border-slate-800 bg-slate-950 font-mono">
				<div>
					<div class="mb-3 flex items-center justify-between border-b border-slate-800/80 pb-3">
						<div class="flex items-center gap-2 text-xs text-slate-300">
							<Terminal class="h-4 w-4 text-indigo-400" />
							<span>Live Response Inspector</span>
						</div>
						{#if lastResponse}
							<Badge
								variant={lastResponse.status < 300 ? 'success' : 'destructive'}
								class="text-[10px]"
							>
								HTTP {lastResponse.status}
							</Badge>
						{:else}
							<span class="text-[10px] text-slate-500">Awaiting Action</span>
						{/if}
					</div>

					{#if lastResponse}
						<div class="space-y-2">
							<div
								class="flex justify-between border-b border-slate-900 pb-1 text-[11px] text-slate-400"
							>
								<span class="text-indigo-400">{lastResponse.action}</span>
								<span>{lastResponse.timestamp}</span>
							</div>
							<pre
								class="mt-2 max-h-60 overflow-auto rounded border border-slate-800/60 bg-slate-900/90 p-3 text-[11px] leading-relaxed text-emerald-400">{JSON.stringify(
									lastResponse.data,
									null,
									2
								)}</pre>
						</div>
					{:else}
						<div class="py-12 text-center text-xs text-slate-500">
							<p>Execute an authentication action to inspect the unified API response payload.</p>
							<p class="mt-2 text-[10px] text-slate-600">
								All responses adhere to S00 unified contract &#123; success, data, error &#125;
							</p>
						</div>
					{/if}
				</div>

				<div
					class="mt-4 flex justify-between border-t border-slate-900 pt-3 text-[10px] text-slate-500"
				>
					<span>Authority: Go Modular Monolith</span>
					<span>Tracing: X-Request-ID</span>
				</div>
			</Card>
		</div>
	</div>

	<!-- Status Grid (S00 Connectivity Verification) -->
	<div class="grid grid-cols-1 gap-6 md:grid-cols-3">
		<!-- Go Backend Status -->
		<Card class="flex flex-col justify-between">
			<div>
				<div class="mb-4 flex items-center justify-between">
					<div class="flex items-center gap-3">
						<div
							class="flex h-10 w-10 items-center justify-center rounded-lg border border-indigo-500/20 bg-indigo-500/10 text-indigo-400"
						>
							<Server class="h-5 w-5" />
						</div>
						<div>
							<h3 class="text-sm font-semibold text-slate-100">Go Backend Service</h3>
							<p class="font-mono text-xs text-slate-400">Chi Router · /api/v1</p>
						</div>
					</div>
					{#if $healthQuery.isLoading}
						<Badge variant="outline">Checking...</Badge>
					{:else if $healthQuery.isSuccess}
						<Badge variant="success">Operational</Badge>
					{:else}
						<Badge variant="destructive">Offline</Badge>
					{/if}
				</div>

				<div class="space-y-2.5 pt-2 text-xs text-slate-300">
					<div class="flex justify-between border-b border-slate-800/60 py-1">
						<span class="text-slate-400">Endpoint</span>
						<span class="font-mono text-slate-200">GET /api/v1/health</span>
					</div>
					<div class="flex justify-between border-b border-slate-800/60 py-1">
						<span class="text-slate-400">Auth Authority</span>
						<span class="font-medium text-emerald-400">Go Ed25519 JWT</span>
					</div>
					<div class="flex justify-between py-1">
						<span class="text-slate-400">Tracing Header</span>
						<span class="font-mono text-[11px] text-slate-200">X-Request-ID Verified</span>
					</div>
				</div>
			</div>

			<div
				class="mt-6 flex items-center justify-between border-t border-slate-800/60 pt-4 text-[11px] text-slate-400"
			>
				<span>Architecture: Modular Monolith</span>
				<span class="font-mono text-indigo-400">Port 8080</span>
			</div>
		</Card>

		<!-- Supabase PostgreSQL Status -->
		<Card class="flex flex-col justify-between">
			<div>
				<div class="mb-4 flex items-center justify-between">
					<div class="flex items-center gap-3">
						<div
							class="flex h-10 w-10 items-center justify-center rounded-lg border border-emerald-500/20 bg-emerald-500/10 text-emerald-400"
						>
							<Database class="h-5 w-5" />
						</div>
						<div>
							<h3 class="text-sm font-semibold text-slate-100">Supabase Database</h3>
							<p class="font-mono text-xs text-slate-400">PostgreSQL 17+ · Cloud</p>
						</div>
					</div>
					{#if $healthQuery.isLoading}
						<Badge variant="outline">Connecting...</Badge>
					{:else if $healthQuery.isSuccess && $healthQuery.data.database.connected}
						<Badge variant="success">Connected</Badge>
					{:else}
						<Badge variant="warning">Config Standby</Badge>
					{/if}
				</div>

				<div class="space-y-2.5 pt-2 text-xs text-slate-300">
					<div class="flex justify-between border-b border-slate-800/60 py-1">
						<span class="text-slate-400">Auth Schema</span>
						<span class="font-mono text-emerald-400">users, auth_otps, user_sessions</span>
					</div>
					<div class="flex justify-between border-b border-slate-800/60 py-1">
						<span class="text-slate-400">Ping Status</span>
						<span class="font-medium text-slate-200">
							{#if $healthQuery.isSuccess}
								{$healthQuery.data.database.message || 'Operational'}
							{:else}
								Awaiting Backend
							{/if}
						</span>
					</div>
					<div class="flex justify-between py-1">
						<span class="text-slate-400">Supabase Auth</span>
						<span class="font-mono text-slate-400">Bypassed (Go Authority)</span>
					</div>
				</div>
			</div>

			<div
				class="mt-6 flex items-center justify-between border-t border-slate-800/60 pt-4 text-[11px] text-slate-400"
			>
				<span>Authority: supabase/migrations</span>
				<span class="text-emerald-400">MCP Linked</span>
			</div>
		</Card>

		<!-- Frontend Client Shell -->
		<Card class="flex flex-col justify-between">
			<div>
				<div class="mb-4 flex items-center justify-between">
					<div class="flex items-center gap-3">
						<div
							class="flex h-10 w-10 items-center justify-center rounded-lg border border-violet-500/20 bg-violet-500/10 text-violet-400"
						>
							<Layout class="h-5 w-5" />
						</div>
						<div>
							<h3 class="text-sm font-semibold text-slate-100">SvelteKit Frontend</h3>
							<p class="font-mono text-xs text-slate-400">Tailwind v4 · TanStack</p>
						</div>
					</div>
					<Badge variant="success">Active</Badge>
				</div>

				<div class="space-y-2.5 pt-2 text-xs text-slate-300">
					<div class="flex justify-between border-b border-slate-800/60 py-1">
						<span class="text-slate-400">Auth Store</span>
						<span class="font-mono text-slate-200">src/lib/auth.svelte.ts</span>
					</div>
					<div class="flex justify-between border-b border-slate-800/60 py-1">
						<span class="text-slate-400">Token Handling</span>
						<span class="font-mono text-slate-200">Auto-Refresh Interceptor</span>
					</div>
					<div class="flex justify-between py-1">
						<span class="text-slate-400">Design Framework</span>
						<span class="text-slate-200">Rhea / shadcn-svelte style</span>
					</div>
				</div>
			</div>

			<div
				class="mt-6 flex items-center justify-between border-t border-slate-800/60 pt-4 text-[11px] text-slate-400"
			>
				<span>Single Source of Truth: /docs</span>
				<span class="text-violet-400">Port 5173</span>
			</div>
		</Card>
	</div>

	<!-- Boundary & Rules Panel -->
	<Card class="bg-slate-900/40">
		<div class="flex items-start gap-4">
			<div
				class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-emerald-500/20 bg-emerald-500/10 text-emerald-400"
			>
				<ShieldCheck class="h-5 w-5" />
			</div>
			<div>
				<h4 class="text-sm font-semibold text-slate-100">
					Sprint S01 Security & Boundary Controls
				</h4>
				<p class="mt-1 text-xs leading-relaxed text-slate-400">
					Sprint S01 establishes identity verification and session tokens only. In accordance with
					documentation-first standards, business roles, society memberships, unit ownership, gate
					operations, and maintenance billing remain strictly deferred.
				</p>
				<div class="mt-4 flex flex-wrap gap-2 text-[11px]">
					<span
						class="inline-flex items-center gap-1 rounded-md border border-slate-700/60 bg-slate-800/60 px-2.5 py-1 text-slate-300"
					>
						<CheckCircle2 class="h-3 w-3 text-emerald-400" /> Max 2 Active Devices Limit
					</span>
					<span
						class="inline-flex items-center gap-1 rounded-md border border-slate-700/60 bg-slate-800/60 px-2.5 py-1 text-slate-300"
					>
						<CheckCircle2 class="h-3 w-3 text-emerald-400" /> Refresh-Token Rotation & Reuse Revocation
					</span>
					<span
						class="inline-flex items-center gap-1 rounded-md border border-slate-700/60 bg-slate-800/60 px-2.5 py-1 text-slate-300"
					>
						<CheckCircle2 class="h-3 w-3 text-emerald-400" /> E.164 Phone Normalization & Masking
					</span>
					<span
						class="inline-flex items-center gap-1 rounded-md border border-slate-700/60 bg-slate-800/60 px-2.5 py-1 text-slate-300"
					>
						<CheckCircle2 class="h-3 w-3 text-emerald-400" /> 60s Resend Cooldown & IP Throttling
					</span>
					<span
						class="inline-flex items-center gap-1 rounded-md border border-slate-700/60 bg-slate-800/60 px-2.5 py-1 text-slate-300"
					>
						<CheckCircle2 class="h-3 w-3 text-emerald-400" /> Zero Business Roles in S01
					</span>
				</div>
			</div>
		</div>
	</Card>
</div>
