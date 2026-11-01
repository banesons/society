<script lang="ts">
	import { onMount } from 'svelte';
	import { societyApi, type Visit } from '$lib/api/society';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import {
		QrCode,
		Building2,
		AlertCircle,
		Loader2,
		ExternalLink,
		RefreshCw
	} from '@lucide/svelte';

	let visits = $state<Visit[]>([]);
	let total = $state(0);
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	async function loadHistory() {
		isLoading = true;
		error = null;
		try {
			const res = await societyApi.getMyVisitorHistory({ limit: 50, offset: 0 });
			visits = res.visits ?? [];
			total = res.total ?? 0;
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load visitor passes';
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		loadHistory();
	});

	function getStatusBadge(status: string): { label: string; class: string } {
		switch (status) {
			case 'REGISTERED_PENDING_GATE_VERIFICATION':
				return {
					label: 'Awaiting Gatekeeper Admission',
					class: 'border-amber-200 bg-amber-50 text-amber-900'
				};
			case 'CHECKED_IN':
				return {
					label: 'Inside Society (Active)',
					class: 'border-emerald-200 bg-emerald-50 text-emerald-900'
				};
			case 'CHECKED_OUT':
				return {
					label: 'Completed Visit',
					class: 'border-neutral-200 bg-neutral-100 text-neutral-800'
				};
			case 'REJECTED':
				return {
					label: 'Entry Denied',
					class: 'border-rose-200 bg-rose-50 text-rose-900'
				};
			default:
				return {
					label: status,
					class: 'border-neutral-200 bg-neutral-100 text-neutral-700'
				};
		}
	}
</script>

<div class="space-y-4">
	<div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2">
				<div class="flex h-7 w-7 items-center justify-center rounded-lg bg-neutral-900 text-white">
					<QrCode class="h-3.5 w-3.5" />
				</div>
				<h2 class="font-serif text-lg font-medium tracking-tight text-neutral-950 sm:text-xl">
					Visitor Passes & Society Visits
					{#if total > 0}
						<span class="text-xs text-neutral-400 font-sans font-normal ml-1">({total})</span>
					{/if}
				</h2>
			</div>
			<p class="mt-1 text-xs text-neutral-500">
				Authenticated movement records and digital passes linked to your verified phone number.
			</p>
		</div>

		<Button
			variant="outline"
			size="sm"
			onclick={loadHistory}
			disabled={isLoading}
			class="self-start text-xs sm:self-auto"
		>
			<RefreshCw class="mr-1.5 h-3 w-3 {isLoading ? 'animate-spin' : ''}" />
			<span>Refresh Passes</span>
		</Button>
	</div>

	{#if isLoading && visits.length === 0}
		<div
			class="flex items-center justify-center rounded-2xl border border-neutral-200/80 bg-white py-12 text-xs text-neutral-500"
		>
			<Loader2 class="mr-2 h-4 w-4 animate-spin text-neutral-900" />
			<span>Retrieving visitor records...</span>
		</div>
	{:else if error}
		<div
			class="flex items-start gap-2.5 rounded-2xl border border-rose-200 bg-rose-50/80 p-4 text-xs text-rose-800"
		>
			<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
			<div>
				<p class="font-medium text-rose-900">Unable to load visitor history</p>
				<p class="mt-0.5">{error}</p>
			</div>
		</div>
	{:else if visits.length === 0}
		<div
			class="rounded-2xl border border-dashed border-neutral-300 bg-neutral-50/50 p-8 text-center text-xs text-neutral-500"
		>
			<QrCode class="mx-auto h-8 w-8 text-neutral-300" />
			<p class="mt-2 font-medium text-neutral-800">No Visitor Passes Found</p>
			<p class="mx-auto mt-1 max-w-sm text-neutral-500">
				When you register as a visitor at a society gate using your phone number, your digital
				passes and check-in timeline will appear here.
			</p>
		</div>
	{:else}
		<div class="space-y-3">
			{#each visits as v (v.id)}
				{@const badge = getStatusBadge(v.status)}
				<div
					class="flex flex-col gap-3 rounded-2xl border border-neutral-200/90 bg-white p-4 shadow-2xs transition-colors hover:border-neutral-300 sm:flex-row sm:items-center sm:justify-between"
				>
					<div class="flex items-start gap-3.5">
						{#if v.visitor_photo_url}
							<img
								src={v.visitor_photo_url}
								alt="Visitor"
								class="h-11 w-11 shrink-0 rounded-xl border border-neutral-200 object-cover"
							/>
						{:else}
							<div
								class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-neutral-200 bg-neutral-100 font-serif text-sm font-bold text-neutral-500"
							>
								{v.visitor_name.charAt(0).toUpperCase()}
							</div>
						{/if}

						<div class="min-w-0">
							<div class="flex flex-wrap items-center gap-2">
								<span class="text-sm font-semibold text-neutral-900">
									{v.visitor_name}
								</span>
								{#if v.pass_reference}
									<span
										class="rounded bg-neutral-100 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-neutral-700"
									>
										{v.pass_reference}
									</span>
								{/if}
								<Badge variant="outline" class="text-[10px] {badge.class}">
									{badge.label}
								</Badge>
							</div>

							<div
								class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-neutral-600"
							>
								{#if v.society_name}
									<span class="inline-flex items-center gap-1 font-medium text-neutral-900">
										<Building2 class="h-3 w-3 text-neutral-400" />
										<span>{v.society_name}</span>
									</span>
								{/if}
								{#if v.entry_gate_name}
									<span class="inline-flex items-center gap-1 text-neutral-500">
										<span>Entry Gate:</span>
										<span class="font-medium text-neutral-700">{v.entry_gate_name}</span>
									</span>
								{/if}
								{#if v.exit_gate_name}
									<span class="inline-flex items-center gap-1 text-neutral-500">
										<span>Exit Gate:</span>
										<span class="font-medium text-neutral-700">{v.exit_gate_name}</span>
									</span>
								{/if}
								{#if v.purpose}
									<span class="rounded bg-neutral-50 px-1.5 py-0.5 text-[11px] text-neutral-600">
										{v.purpose}
									</span>
								{/if}
							</div>

							{#if v.rejection_reason}
								<div
									class="mt-2 rounded-xl border border-rose-100 bg-rose-50/70 px-3 py-1.5 text-xs text-rose-800"
								>
									<span class="font-medium">Denial reason:</span>
									{v.rejection_reason}
								</div>
							{/if}

							<div
								class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-[11px] text-neutral-400"
							>
								<span
									>Registered: {new Date(v.created_at).toLocaleString([], {
										month: 'short',
										day: 'numeric',
										hour: '2-digit',
										minute: '2-digit'
									})}</span
								>
								{#if v.check_in_at}
									<span
										>· Admitted: {new Date(v.check_in_at).toLocaleTimeString([], {
											hour: '2-digit',
											minute: '2-digit'
										})}</span
									>
								{/if}
								{#if v.check_out_at}
									<span
										>· Departed: {new Date(v.check_out_at).toLocaleTimeString([], {
											hour: '2-digit',
											minute: '2-digit'
										})}</span
									>
								{/if}
							</div>
						</div>
					</div>

					<div class="flex items-center gap-2 self-end sm:self-center">
						{#if v.pass_reference && v.status === 'REGISTERED_PENDING_GATE_VERIFICATION'}
							<a
								href="/visit/pass/{v.pass_reference}"
								target="_blank"
								class="inline-flex items-center gap-1.5 rounded-full border border-neutral-300 bg-white px-3 py-1.5 text-xs font-medium text-neutral-800 shadow-2xs transition-colors hover:border-neutral-900 hover:text-neutral-950"
							>
								<ExternalLink class="h-3 w-3" />
								<span>Open Pass</span>
							</a>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
