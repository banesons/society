<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { societyApi, type SocietyMember } from '$lib/api/society';
	import { auth } from '$lib/auth.svelte';
	import { ApiClientError } from '$lib/api/client';
	import { formatFullPhone } from '$lib/phone';
	import RegisterMemberDialog from './register-member-dialog.svelte';
	import MemberDetailDialog from './member-detail-dialog.svelte';
	import DepartureDialog from './departure-dialog.svelte';
	import {
		Users,
		UserPlus,
		Search,
		Loader2,
		AlertCircle,
		UserMinus,
		Eye,
		ShieldCheck,
		Clock
	} from '@lucide/svelte';

	let { societyId }: { societyId: string } = $props();

	let members = $state<SocietyMember[]>([]);
	let total = $state(0);
	let isLoading = $state(false);
	let error = $state<string | null>(null);

	// Filters
	let searchQuery = $state('');
	const statusOptions = ['ALL', 'ACTIVE', 'INACTIVE', 'LEFT'] as const;
	let statusFilter = $state<(typeof statusOptions)[number]>('ALL');

	// Dialog states
	let isRegisterOpen = $state(false);
	let selectedMemberId = $state<string | null>(null);
	let isDetailOpen = $state(false);
	let departingMember = $state<SocietyMember | null>(null);
	let isDepartureOpen = $state(false);

	async function loadMembers() {
		if (!societyId) return;
		members = [];
		total = 0;
		isLoading = true;
		error = null;

		try {
			const res = await societyApi.listMembers(societyId, {
				search: searchQuery.trim() || undefined,
				status: statusFilter === 'ALL' ? undefined : statusFilter,
				limit: 100
			});
			members = res.members;
			total = res.total;
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message || 'Failed to load society members.';
			} else {
				error = 'Failed to load members.';
			}
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (societyId && !auth.isLoading) {
			// Trigger reload when search or status filter changes
			const timer = setTimeout(() => {
				loadMembers();
			}, 250);
			return () => clearTimeout(timer);
		}
	});

	function getStatusVariant(status: string): 'default' | 'success' | 'warning' | 'destructive' {
		switch (status) {
			case 'ACTIVE':
				return 'success';
			case 'INACTIVE':
				return 'warning';
			case 'LEFT':
				return 'destructive';
			default:
				return 'default';
		}
	}
</script>

<div class="mt-6 rounded-2xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-xs">
	<!-- Top Bar -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2">
				<div
					class="flex h-8 w-8 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
				>
					<Users class="h-4 w-4" />
				</div>
				<h2 class="text-base font-semibold tracking-tight text-neutral-950">
					Society Members & Residents
				</h2>
				<Badge
					variant="outline"
					class="border-emerald-200 bg-emerald-50 font-mono text-[10px] text-emerald-800"
					>S06 Active</Badge
				>
			</div>
			<p class="mt-1 text-xs text-neutral-500">
				Manage owners, tenants, and co-occupants ({total} registered). Tracks property associations and
				departure lifecycles.
			</p>
		</div>

		<Button variant="primary" size="sm" onclick={() => (isRegisterOpen = true)}>
			<UserPlus class="mr-1.5 h-3.5 w-3.5" />
			<span>Register Member</span>
		</Button>
	</div>

	<!-- Controls & Filters -->
	<div class="mt-5 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<!-- Search -->
		<div class="relative flex-1 sm:max-w-xs">
			<Search class="absolute top-2.5 left-3 h-4 w-4 text-neutral-400" />
			<input
				type="text"
				placeholder="Search by name or phone..."
				bind:value={searchQuery}
				class="w-full rounded-full border border-neutral-300 bg-white py-2 pr-3 pl-9 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
			/>
		</div>

		<!-- Status Filter Pills -->
		<div class="flex items-center gap-1 rounded-full border border-neutral-200 bg-neutral-100 p-1">
			{#each statusOptions as st (st)}
				<button
					onclick={() => (statusFilter = st)}
					class="rounded-full px-3 py-1 text-xs font-medium transition-all {statusFilter === st
						? 'bg-neutral-950 text-white shadow-xs'
						: 'text-neutral-600 hover:text-neutral-900'}"
				>
					{st}
				</button>
			{/each}
		</div>
	</div>

	<!-- Content -->
	{#if isLoading && members.length === 0}
		<div class="flex min-h-[220px] flex-col items-center justify-center text-neutral-500">
			<Loader2 class="h-7 w-7 animate-spin text-neutral-900" />
			<p class="mt-3 text-xs">Loading society member registry...</p>
		</div>
	{:else if error}
		<div
			class="mt-4 flex items-center gap-2 rounded-2xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800"
		>
			<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
			<span>{error}</span>
		</div>
	{:else if members.length === 0}
		<div
			class="mt-6 rounded-2xl border border-dashed border-neutral-200 bg-neutral-50/50 p-8 text-center"
		>
			<Users class="mx-auto h-8 w-8 text-neutral-400" />
			<h4 class="mt-2 text-xs font-semibold text-neutral-800">No members found</h4>
			<p class="mt-1 text-xs text-neutral-500">
				{searchQuery || statusFilter !== 'ALL'
					? 'No members match the current search or status filter.'
					: 'Get started by registering the first owner or resident for this society.'}
			</p>
			{#if !searchQuery && statusFilter === 'ALL'}
				<div class="mt-4">
					<Button variant="outline" size="sm" onclick={() => (isRegisterOpen = true)}>
						<UserPlus class="mr-1.5 h-3.5 w-3.5" />
						<span>Register Member</span>
					</Button>
				</div>
			{/if}
		</div>
	{:else}
		<div class="mt-4 overflow-x-auto rounded-2xl border border-neutral-200 bg-white shadow-xs">
			<table class="w-full text-left text-xs">
				<thead
					class="border-b border-neutral-200 bg-neutral-50/80 text-[11px] font-semibold tracking-wider text-neutral-500 uppercase"
				>
					<tr>
						<th class="px-4 py-3">Member</th>
						<th class="px-4 py-3">Flat / Unit</th>
						<th class="px-4 py-3">Identity Status</th>
						<th class="px-4 py-3">Member Status</th>
						<th class="px-4 py-3 text-right">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-neutral-100 text-neutral-800">
					{#each members as m (m.id)}
						<tr class="transition-colors hover:bg-neutral-50/60">
							<td class="px-4 py-3">
								<div class="flex items-center gap-2 font-medium text-neutral-950">
									<span>{m.first_name} {m.last_name || ''}</span>
									{#if m.email}
										<span class="text-[11px] text-neutral-500">({m.email})</span>
									{/if}
								</div>
								<div class="font-mono text-[11px] text-neutral-500">
									{formatFullPhone(m.phone || m.phone_masked)}
								</div>
							</td>
							<td class="px-4 py-3 font-mono font-medium text-neutral-900">
								{#if m.unit_numbers}
									<span>{m.unit_numbers}</span>
								{:else}
									<span class="font-sans text-neutral-400">—</span>
								{/if}
							</td>
							<td class="px-4 py-3">
								{#if m.user_id}
									<span class="inline-flex items-center gap-1 text-[11px] text-emerald-700">
										<ShieldCheck class="h-3.5 w-3.5 text-emerald-600" />
										<span>Verified Linked</span>
									</span>
								{:else}
									<span class="inline-flex items-center gap-1 text-[11px] text-amber-700">
										<Clock class="h-3.5 w-3.5 text-amber-600" />
										<span>Pre-registered</span>
									</span>
								{/if}
							</td>
							<td class="px-4 py-3">
								<Badge variant={getStatusVariant(m.status)} class="text-[10px]">
									{m.status}
								</Badge>
							</td>
							<td class="px-4 py-3 text-right">
								<div class="inline-flex items-center gap-1">
									<button
										onclick={() => {
											selectedMemberId = m.id;
											isDetailOpen = true;
										}}
										class="inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-[11px] font-medium text-neutral-700 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
									>
										<Eye class="h-3 w-3" />
										<span>Profile & Units</span>
									</button>

									{#if m.status !== 'LEFT'}
										<button
											onclick={() => {
												departingMember = m;
												isDepartureOpen = true;
											}}
											class="inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-[11px] font-medium text-amber-700 transition-colors hover:bg-amber-50 hover:text-amber-800"
										>
											<UserMinus class="h-3 w-3" />
											<span>Depart</span>
										</button>
									{/if}
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<!-- Modals -->
<RegisterMemberDialog
	isOpen={isRegisterOpen}
	{societyId}
	onClose={() => (isRegisterOpen = false)}
	onSuccess={() => loadMembers()}
/>

<MemberDetailDialog
	isOpen={isDetailOpen}
	{societyId}
	memberId={selectedMemberId}
	onClose={() => {
		isDetailOpen = false;
		selectedMemberId = null;
	}}
	onDepartClick={(mem) => {
		isDetailOpen = false;
		departingMember = mem;
		isDepartureOpen = true;
	}}
	onRefresh={() => loadMembers()}
/>

<DepartureDialog
	isOpen={isDepartureOpen}
	{societyId}
	member={departingMember}
	onClose={() => {
		isDepartureOpen = false;
		departingMember = null;
	}}
	onSuccess={() => loadMembers()}
/>
