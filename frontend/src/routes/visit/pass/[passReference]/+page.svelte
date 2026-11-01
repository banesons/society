<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { societyApi, type PublicVisitorPass } from '$lib/api/society';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import {
		MapPin,
		Clock,
		CheckCircle2,
		XCircle,
		AlertCircle,
		Loader2,
		User,
		RefreshCw
	} from '@lucide/svelte';

	let passReference = $derived(page.params.passReference);

	let pass = $state<PublicVisitorPass | null>(null);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let pollInterval: ReturnType<typeof setInterval> | null = null;

	async function loadPass() {
		if (!passReference) return;
		try {
			pass = await societyApi.getPublicVisitorPass(passReference);
			error = null;
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Visitor pass not found or expired';
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		loadPass();
		pollInterval = setInterval(loadPass, 5000);
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
	});
</script>

<svelte:head>
	<title
		>{pass
			? `${pass.visitor_name} — Visitor Pass | ${pass.society_name}`
			: 'Visitor Pass | CHS Neu'}</title
	>
</svelte:head>

<div
	class="min-h-screen bg-[#f3f3f1] font-sans text-[#121212] antialiased selection:bg-neutral-800 selection:text-white"
>
	<!-- Minimal Header -->
	<header
		class="sticky top-0 z-10 border-b border-[#e2e2de] bg-white/80 px-4 py-3 backdrop-blur-md"
	>
		<div class="mx-auto flex max-w-md items-center justify-between">
			<div class="flex items-center gap-2">
				<div
					class="flex h-6 w-6 items-center justify-center rounded bg-[#121212] font-serif text-xs font-bold text-white"
				>
					C
				</div>
				<span class="text-xs font-semibold tracking-tight text-neutral-900">CHS Neu Security</span>
			</div>
			<span class="font-mono text-[10px] tracking-wider text-neutral-400 uppercase"
				>Digital Pass</span
			>
		</div>
	</header>

	<main class="mx-auto max-w-md px-4 py-6">
		{#if isLoading && !pass}
			<div class="flex flex-col items-center justify-center py-20 text-center">
				<Loader2 class="mb-3 h-8 w-8 animate-spin text-neutral-900" />
				<p class="text-xs font-medium text-neutral-700">Loading Visitor Pass...</p>
			</div>
		{:else if error}
			<div class="space-y-4 rounded-2xl border border-rose-200 bg-white p-6 text-center shadow-sm">
				<div
					class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-rose-50 text-rose-600"
				>
					<AlertCircle class="h-6 w-6" />
				</div>
				<div>
					<h2 class="text-base font-semibold text-neutral-900">Pass Unavailable</h2>
					<p class="mt-1 text-xs text-neutral-500">{error}</p>
				</div>
				<Button variant="outline" size="sm" onclick={loadPass} class="text-xs">
					<RefreshCw class="mr-1.5 h-3 w-3" />
					Retry Lookup
				</Button>
			</div>
		{:else if pass}
			<div class="space-y-4">
				<!-- Status Banner -->
				{#if pass.status === 'CHECKED_IN'}
					<div
						class="flex items-center gap-3 rounded-2xl border border-emerald-200 bg-emerald-50/80 p-4"
					>
						<CheckCircle2 class="h-5 w-5 shrink-0 text-emerald-600" />
						<div>
							<p class="text-xs font-semibold text-emerald-950">Entry Approved & Admitted</p>
							<p class="text-[11px] text-emerald-800">You are checked into {pass.society_name}.</p>
						</div>
					</div>
				{:else if pass.status === 'CHECKED_OUT'}
					<div
						class="flex items-center gap-3 rounded-2xl border border-neutral-300 bg-neutral-100 p-4"
					>
						<Clock class="h-5 w-5 shrink-0 text-neutral-600" />
						<div>
							<p class="text-xs font-semibold text-neutral-950">Visit Completed</p>
							<p class="text-[11px] text-neutral-600">You have checked out of the premises.</p>
						</div>
					</div>
				{:else if pass.status === 'REJECTED'}
					<div class="flex items-center gap-3 rounded-2xl border border-rose-200 bg-rose-50/80 p-4">
						<XCircle class="h-5 w-5 shrink-0 text-rose-600" />
						<div>
							<p class="text-xs font-semibold text-rose-950">Entry Denied</p>
							<p class="text-[11px] text-rose-800">Security has declined entry for this visit.</p>
						</div>
					</div>
				{:else}
					<div
						class="flex items-center gap-3 rounded-2xl border border-amber-200 bg-amber-50/80 p-4"
					>
						<Clock class="h-5 w-5 shrink-0 animate-pulse text-amber-600" />
						<div>
							<p class="text-xs font-semibold text-amber-950">Awaiting Security Verification</p>
							<p class="text-[11px] text-amber-800">
								Please present this screen to the gatekeeper.
							</p>
						</div>
					</div>
				{/if}

				<!-- Digital Pass Card -->
				<div class="overflow-hidden rounded-2xl border border-[#e2e2de] bg-white shadow-sm">
					<!-- Pass Header -->
					<div class="flex items-center justify-between bg-neutral-900 p-5 text-white">
						<div>
							<p class="font-mono text-[10px] tracking-widest text-neutral-400 uppercase">
								Visitor Pass
							</p>
							<h2 class="mt-0.5 text-xl font-bold tracking-tight text-white">
								{pass.society_name}
							</h2>
							{#if pass.gate_name}
								<p class="mt-0.5 flex items-center gap-1 text-xs text-neutral-300">
									<MapPin class="h-3 w-3 text-neutral-400" />
									{pass.gate_name}
								</p>
							{/if}
						</div>

						<div class="text-right">
							<p class="font-mono text-[10px] tracking-widest text-neutral-400 uppercase">
								Pass Code
							</p>
							<div class="font-mono text-lg font-bold tracking-wider text-emerald-400">
								{pass.pass_reference}
							</div>
						</div>
					</div>

					<!-- Pass Body -->
					<div class="space-y-6 p-6">
						<!-- Photo & Primary Details -->
						<div class="flex items-center gap-4 border-b border-[#e2e2de] pb-6">
							{#if pass.visitor_photo_url}
								<img
									src={pass.visitor_photo_url}
									alt={pass.visitor_name}
									class="h-16 w-16 shrink-0 rounded-xl border border-[#e2e2de] bg-neutral-100 object-cover shadow-sm"
								/>
							{:else}
								<div
									class="flex h-16 w-16 shrink-0 items-center justify-center rounded-xl border border-[#e2e2de] bg-neutral-100 text-neutral-400"
								>
									<User class="h-8 w-8" />
								</div>
							{/if}

							<div class="space-y-1">
								<h3 class="text-base font-semibold text-neutral-900">{pass.visitor_name}</h3>
								<p class="font-mono text-xs text-neutral-500">{pass.visitor_phone_masked}</p>
								<Badge variant="outline" class="border-neutral-300 font-mono text-[10px]">
									{pass.purpose.replace(/_/g, ' ')}
								</Badge>
							</div>
						</div>

						<!-- Grid Info -->
						<div class="grid grid-cols-2 gap-4 text-xs">
							<div>
								<span class="block text-[10px] tracking-wider text-neutral-400 uppercase"
									>Status</span
								>
								<span class="font-medium text-neutral-800">
									{pass.status.replace(/_/g, ' ')}
								</span>
							</div>

							<div>
								<span class="block text-[10px] tracking-wider text-neutral-400 uppercase"
									>Registered At</span
								>
								<span class="font-medium text-neutral-800">
									{new Date(pass.registered_at).toLocaleTimeString([], {
										hour: '2-digit',
										minute: '2-digit'
									})}
								</span>
							</div>

							{#if pass.coming_from}
								<div class="col-span-2">
									<span class="block text-[10px] tracking-wider text-neutral-400 uppercase"
										>Origin</span
									>
									<span class="font-medium text-neutral-800">{pass.coming_from}</span>
								</div>
							{/if}

							{#if pass.check_in_at}
								<div>
									<span class="block text-[10px] tracking-wider text-neutral-400 uppercase"
										>Check In</span
									>
									<span class="font-medium text-emerald-700">
										{new Date(pass.check_in_at).toLocaleTimeString([], {
											hour: '2-digit',
											minute: '2-digit'
										})}
									</span>
								</div>
							{/if}
						</div>
					</div>

					<!-- Pass Footer -->
					<div class="border-t border-[#e2e2de] bg-neutral-50 p-4 text-center">
						<p class="text-[11px] text-neutral-500">
							Keep this screen open when approaching security at the gate. Status updates
							automatically.
						</p>
					</div>
				</div>
			</div>
		{/if}
	</main>
</div>
