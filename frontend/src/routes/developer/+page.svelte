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
	import { goto } from '$app/navigation';

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
	let testPhone = $state('919967089101');
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
			const cleanPhone = testPhone.replace(/\D/g, '');
			const res = await api.post<{ resend_after_seconds: number; otp?: string }>('/auth/otp/send', {
				phone: cleanPhone
			});
			startCooldown(res.resend_after_seconds);
			if (res.otp) {
				testOtp = res.otp;
			}
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
	{#if !dev && (!auth.isAuthenticated || !auth.user?.platform_roles?.includes('SUPER_ADMIN'))}
		<div class="rounded-3xl border border-red-500/20 bg-white p-12 text-center shadow-sm">
			<div
				class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-red-50 text-red-600"
			>
				<ShieldCheck class="h-7 w-7" />
			</div>
			<h2 class="text-xl font-bold text-neutral-900">Developer Diagnostics Restricted</h2>
			<p class="mx-auto mt-2 max-w-md text-sm text-neutral-500">
				This diagnostic console is disabled in production environments for non-administrative
				accounts. Please log in with a platform Super Administrator credential.
			</p>
			<div class="mt-6">
				<Button variant="outline" onclick={() => goto('/admin')}>Return to Admin Dashboard</Button>
			</div>
		</div>
	{:else}
		{#if !dev}
			<div class="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 text-xs text-amber-300">
				<strong>Notice:</strong> Developer diagnostics and interactive test shortcuts are restricted in
				production environments.
			</div>
		{/if}

		<!-- Hero / Overview Banner -->
		<div
			class="relative overflow-hidden rounded-3xl border border-neutral-200 bg-white p-8 shadow-sm"
		>
			<div class="relative z-10 flex flex-col justify-between gap-6 md:flex-row md:items-center">
				<div>
					<div class="mb-3 inline-flex items-center gap-2">
						<Badge variant="default">Sprint S01/S02</Badge>
						<span class="text-xs text-neutral-500">Developer Diagnostics & API Testing</span>
						{#if dev}
							<Badge variant="success" class="text-[10px]">LOCAL DEV</Badge>
						{/if}
					</div>
					<h1 class="text-3xl font-bold tracking-tight text-neutral-950 sm:text-4xl">
						CHS Platform Diagnostics
					</h1>
					<p class="mt-2 max-w-2xl text-sm leading-relaxed text-neutral-600">
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
					<div class="flex items-center justify-between border-b border-neutral-200 pb-4">
						<div class="flex items-center gap-3">
							<div
								class="flex h-10 w-10 items-center justify-center rounded-full border border-neutral-200 bg-neutral-100 text-neutral-800"
							>
								{#if auth.isAuthenticated}
									<UserCheck class="h-5 w-5 text-emerald-600" />
								{:else}
									<Smartphone class="h-5 w-5 text-neutral-800" />
								{/if}
							</div>
							<div>
								<h3 class="text-sm font-semibold text-neutral-950">
									Authentication & Session State
								</h3>
								<p class="font-mono text-xs text-neutral-500">
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
						<div class="space-y-4 rounded-2xl border border-neutral-200 bg-neutral-50 p-5">
							<div class="grid grid-cols-1 gap-4 text-xs sm:grid-cols-3">
								<div>
									<span class="mb-1 block text-neutral-500">Masked Phone</span>
									<span class="font-mono text-sm font-medium text-neutral-950"
										>{auth.user.phone}</span
									>
								</div>
								<div>
									<span class="mb-1 block text-neutral-500">User UUID</span>
									<span
										class="block truncate font-mono text-[11px] text-neutral-700"
										title={auth.user.id}
									>
										{auth.user.id}
									</span>
								</div>
								<div>
									<span class="mb-1 block text-neutral-500">Account Status</span>
									<Badge variant="success">{auth.user.status}</Badge>
								</div>
							</div>

							<div
								class="flex flex-wrap gap-4 border-t border-neutral-200 pt-3 text-[11px] text-neutral-500"
							>
								<span
									>Token Type: <strong class="text-neutral-900">Ed25519 Bearer (15m)</strong></span
								>
								<span
									>Refresh Policy: <strong class="text-neutral-900"
										>Single-Use Rotation (30d)</strong
									></span
								>
								<span>Device Cap: <strong class="text-neutral-950">Max 2 Active</strong></span>
							</div>
						</div>

						<!-- Authenticated Action Triggers -->
						<div class="space-y-2">
							<span class="block text-xs font-medium text-neutral-700">Session Operations</span>
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
									class="w-full text-xs text-amber-700 hover:bg-amber-50"
								>
									<span>Logout Device</span>
								</Button>
								<Button
									variant="outline"
									size="sm"
									onclick={runLogoutAll}
									disabled={testLoading}
									class="w-full text-xs text-rose-700 hover:bg-rose-50"
								>
									<span>Logout All</span>
								</Button>
							</div>
						</div>
					{:else}
						<!-- Unauthenticated Direct Test Form -->
						<div class="space-y-4">
							<p class="text-xs leading-relaxed text-neutral-600">
								Test the S01 identity pipeline directly with configured development/test numbers:
							</p>

							<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
								<div>
									<label for="test-phone" class="mb-1 block text-xs font-medium text-neutral-700">
										Mobile Number (India)
									</label>
									<div
										class="flex rounded-xl border border-neutral-300 bg-white focus-within:border-neutral-950"
									>
										<span
											class="inline-flex items-center rounded-l-xl border-r border-neutral-200 bg-neutral-50 px-2.5 text-xs font-semibold text-neutral-600 select-none"
										>
											+
										</span>
										<input
											id="test-phone"
											type="tel"
											bind:value={testPhone}
											placeholder="919967089101"
											class="w-full bg-transparent px-3 py-2 font-mono text-xs text-neutral-900 placeholder-neutral-400 focus:outline-none"
										/>
									</div>
									<div class="mt-1 flex gap-1.5">
										<button
											type="button"
											onclick={() => {
												testPhone = '919967089101';
											}}
											class="cursor-pointer font-mono text-[10px] text-neutral-900 underline hover:opacity-75"
										>
											919967089101
										</button>
										<span class="text-[10px] text-neutral-400">·</span>
										<button
											type="button"
											onclick={() => {
												testPhone = '918422862001';
											}}
											class="cursor-pointer font-mono text-[10px] text-neutral-900 underline hover:opacity-75"
										>
											918422862001
										</button>
									</div>
								</div>

								<div>
									<label for="test-otp" class="mb-1 block text-xs font-medium text-neutral-700">
										OTP Passcode
									</label>
									<input
										id="test-otp"
										type="text"
										bind:value={testOtp}
										maxlength={6}
										class="w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 font-mono text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-none"
									/>
									<span class="mt-1 block text-[10px] text-neutral-500">Dev OTP: 123456</span>
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
									<KeyRound class="mr-1 h-3.5 w-3.5 text-neutral-800" />
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
								class="flex h-10 w-10 items-center justify-center rounded-full border border-neutral-200 bg-neutral-100 text-neutral-800"
							>
								<Server class="h-5 w-5" />
							</div>
							<div>
								<h3 class="text-sm font-semibold text-neutral-950">Go Backend Service</h3>
								<p class="font-mono text-xs text-neutral-500">Chi Router · /api/v1</p>
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

					<div class="space-y-2.5 pt-2 text-xs text-neutral-700">
						<div class="flex justify-between border-b border-neutral-100 py-1">
							<span class="text-neutral-500">Endpoint</span>
							<span class="font-mono text-neutral-950">GET /api/v1/health</span>
						</div>
						<div class="flex justify-between border-b border-neutral-100 py-1">
							<span class="text-neutral-500">Auth Authority</span>
							<span class="font-medium text-emerald-700">Go Ed25519 JWT</span>
						</div>
						<div class="flex justify-between py-1">
							<span class="text-neutral-500">Tracing Header</span>
							<span class="font-mono text-[11px] text-neutral-950">X-Request-ID Verified</span>
						</div>
					</div>
				</div>

				<div
					class="mt-6 flex items-center justify-between border-t border-neutral-100 pt-4 text-[11px] text-neutral-500"
				>
					<span>Architecture: Modular Monolith</span>
					<span class="font-mono font-semibold text-neutral-900">Port 8080</span>
				</div>
			</Card>

			<!-- Supabase PostgreSQL Status -->
			<Card class="flex flex-col justify-between">
				<div>
					<div class="mb-4 flex items-center justify-between">
						<div class="flex items-center gap-3">
							<div
								class="flex h-10 w-10 items-center justify-center rounded-full border border-emerald-200 bg-emerald-50 text-emerald-700"
							>
								<Database class="h-5 w-5" />
							</div>
							<div>
								<h3 class="text-sm font-semibold text-neutral-950">Supabase Database</h3>
								<p class="font-mono text-xs text-neutral-500">PostgreSQL 17+ · Cloud</p>
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

					<div class="space-y-2.5 pt-2 text-xs text-neutral-700">
						<div class="flex justify-between border-b border-neutral-100 py-1">
							<span class="text-neutral-500">Auth Schema</span>
							<span class="font-mono text-emerald-700">users, auth_otps, user_sessions</span>
						</div>
						<div class="flex justify-between border-b border-neutral-100 py-1">
							<span class="text-neutral-500">Ping Status</span>
							<span class="font-medium text-neutral-900">
								{#if $healthQuery.isSuccess}
									{$healthQuery.data.database.message || 'Operational'}
								{:else}
									Awaiting Backend
								{/if}
							</span>
						</div>
						<div class="flex justify-between py-1">
							<span class="text-neutral-500">Supabase Auth</span>
							<span class="font-mono text-neutral-500">Bypassed (Go Authority)</span>
						</div>
					</div>
				</div>

				<div
					class="mt-6 flex items-center justify-between border-t border-neutral-100 pt-4 text-[11px] text-neutral-500"
				>
					<span>Authority: supabase/migrations</span>
					<span class="font-medium text-emerald-700">MCP Linked</span>
				</div>
			</Card>

			<!-- Frontend Client Shell -->
			<Card class="flex flex-col justify-between">
				<div>
					<div class="mb-4 flex items-center justify-between">
						<div class="flex items-center gap-3">
							<div
								class="flex h-10 w-10 items-center justify-center rounded-full border border-neutral-200 bg-neutral-100 text-neutral-800"
							>
								<Layout class="h-5 w-5" />
							</div>
							<div>
								<h3 class="text-sm font-semibold text-neutral-950">SvelteKit Frontend</h3>
								<p class="font-mono text-xs text-neutral-500">Tailwind v4 · TanStack</p>
							</div>
						</div>
						<Badge variant="success">Active</Badge>
					</div>

					<div class="space-y-2.5 pt-2 text-xs text-neutral-700">
						<div class="flex justify-between border-b border-neutral-100 py-1">
							<span class="text-neutral-500">Auth Store</span>
							<span class="font-mono text-neutral-950">src/lib/auth.svelte.ts</span>
						</div>
						<div class="flex justify-between border-b border-neutral-100 py-1">
							<span class="text-neutral-500">Token Handling</span>
							<span class="font-mono text-neutral-950">Auto-Refresh Interceptor</span>
						</div>
						<div class="flex justify-between py-1">
							<span class="text-neutral-500">Design Framework</span>
							<span class="text-neutral-900">Editorial Swiss / Hedvig</span>
						</div>
					</div>
				</div>

				<div
					class="mt-6 flex items-center justify-between border-t border-neutral-100 pt-4 text-[11px] text-neutral-500"
				>
					<span>Single Source of Truth: /docs</span>
					<span class="font-mono font-medium text-neutral-900">Port 5173</span>
				</div>
			</Card>
		</div>

		<!-- Boundary & Rules Panel -->
		<Card class="border-neutral-200 bg-white">
			<div class="flex items-start gap-4">
				<div
					class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-emerald-200 bg-emerald-50 text-emerald-700"
				>
					<ShieldCheck class="h-5 w-5" />
				</div>
				<div>
					<h4 class="text-sm font-semibold text-neutral-950">
						Sprint S01 Security & Boundary Controls
					</h4>
					<p class="mt-1 text-xs leading-relaxed text-neutral-600">
						Sprint S01 establishes identity verification and session tokens only. In accordance with
						documentation-first standards, business roles, society memberships, unit ownership, gate
						operations, and maintenance billing remain strictly deferred.
					</p>
					<div class="mt-4 flex flex-wrap gap-2 text-[11px]">
						<span
							class="inline-flex items-center gap-1 rounded-full border border-neutral-200 bg-neutral-50 px-3 py-1 text-neutral-700"
						>
							<CheckCircle2 class="h-3 w-3 text-emerald-600" /> Max 2 Active Devices Limit
						</span>
						<span
							class="inline-flex items-center gap-1 rounded-full border border-neutral-200 bg-neutral-50 px-3 py-1 text-neutral-700"
						>
							<CheckCircle2 class="h-3 w-3 text-emerald-600" /> Refresh-Token Rotation & Reuse Revocation
						</span>
						<span
							class="inline-flex items-center gap-1 rounded-full border border-neutral-200 bg-neutral-50 px-3 py-1 text-neutral-700"
						>
							<CheckCircle2 class="h-3 w-3 text-emerald-600" /> E.164 Phone Normalization & Masking
						</span>
						<span
							class="inline-flex items-center gap-1 rounded-full border border-neutral-200 bg-neutral-50 px-3 py-1 text-neutral-700"
						>
							<CheckCircle2 class="h-3 w-3 text-emerald-600" /> 60s Resend Cooldown & IP Throttling
						</span>
						<span
							class="inline-flex items-center gap-1 rounded-full border border-neutral-200 bg-neutral-50 px-3 py-1 text-neutral-700"
						>
							<CheckCircle2 class="h-3 w-3 text-emerald-600" /> Zero Business Roles in S01
						</span>
					</div>
				</div>
			</div>
		</Card>
	{/if}
</div>
