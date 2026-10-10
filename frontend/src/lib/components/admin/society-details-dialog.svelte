<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import type { SocietyItem } from '$lib/api/society';
	import SocietyAdminsTab from './society-admins-tab.svelte';
	import FormationWizard from '$lib/components/formation/formation-wizard.svelte';
	import {
		Building2,
		X,
		Copy,
		Check,
		MapPin,
		Calendar,
		Mail,
		FileText,
		Users,
		Layers
	} from '@lucide/svelte';

	let {
		isOpen = false,
		society = null,
		onClose,
		onEdit
	}: {
		isOpen: boolean;
		society: SocietyItem | null;
		onClose: () => void;
		onEdit?: (soc: SocietyItem) => void;
	} = $props();

	let copied = $state(false);
	let activeTab = $state<'overview' | 'admins'>('overview');
	let isFormationOpen = $state(false);

	async function copyCode() {
		if (!society) return;
		try {
			await navigator.clipboard.writeText(society.society_code);
			copied = true;
			setTimeout(() => {
				copied = false;
			}, 2000);
		} catch {
			// fallback
		}
	}

	function formatDate(iso: string) {
		try {
			return new Date(iso).toLocaleDateString('en-IN', {
				year: 'numeric',
				month: 'short',
				day: 'numeric',
				hour: '2-digit',
				minute: '2-digit'
			});
		} catch {
			return iso;
		}
	}

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

{#if isOpen && society}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6"
		role="dialog"
		aria-labelledby="society-details-title"
		aria-modal="true"
	>
		<!-- Backdrop -->
		<button
			type="button"
			tabindex="-1"
			class="fixed inset-0 bg-slate-950/80 backdrop-blur-sm transition-opacity"
			onclick={onClose}
			aria-label="Close dialog overlay"
		></button>

		<!-- Modal Container -->
		<div
			class="relative max-h-[90vh] w-full max-w-xl overflow-y-auto rounded-2xl border border-slate-800 bg-slate-900 p-6 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-slate-800/80 pb-4">
				<div class="flex items-center gap-2.5">
					<div
						class="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-500/10 text-indigo-400"
					>
						<Building2 class="h-5 w-5" />
					</div>
					<div>
						<h2 id="society-details-title" class="text-base font-semibold text-slate-100">
							{society.name}
						</h2>
						<div class="flex items-center gap-2 text-xs">
							<Badge variant={getStatusVariant(society.status)}>{society.status}</Badge>
							<span class="text-slate-400">· {society.city}, {society.state}</span>
						</div>
					</div>
				</div>
				<button
					type="button"
					onclick={onClose}
					class="rounded-lg p-1.5 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
					aria-label="Close dialog"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<!-- Tab Navigation -->
			<div class="mt-4 flex items-center gap-2 border-b border-slate-800/80 pb-2">
				<button
					type="button"
					onclick={() => (activeTab = 'overview')}
					class={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
						activeTab === 'overview'
							? 'bg-indigo-500/20 text-indigo-300'
							: 'text-slate-400 hover:bg-slate-800/60 hover:text-slate-200'
					}`}
				>
					<Building2 class="h-3.5 w-3.5" />
					<span>Overview</span>
				</button>
				<button
					type="button"
					onclick={() => (activeTab = 'admins')}
					class={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
						activeTab === 'admins'
							? 'bg-indigo-500/20 text-indigo-300'
							: 'text-slate-400 hover:bg-slate-800/60 hover:text-slate-200'
					}`}
				>
					<Users class="h-3.5 w-3.5" />
					<span>Administrators</span>
				</button>
			</div>

			{#if activeTab === 'admins'}
				<div class="mt-4">
					<SocietyAdminsTab societyId={society.id} />
				</div>
			{:else}
				<div class="mt-5 space-y-4 text-xs">
					<!-- Immutable Code Box -->
					<div
						class="flex items-center justify-between rounded-xl border border-indigo-500/30 bg-slate-950/70 p-4"
					>
						<div>
							<div class="text-[11px] font-medium tracking-wider text-slate-400 uppercase">
								Society Code (Immutable)
							</div>
							<div class="mt-0.5 font-mono text-base font-bold text-indigo-300">
								{society.society_code}
							</div>
						</div>
						<Button
							variant="outline"
							size="sm"
							onclick={copyCode}
							class="border-indigo-500/40 text-xs"
						>
							{#if copied}
								<Check class="mr-1 h-3.5 w-3.5 text-emerald-400" />
								<span class="text-emerald-400">Copied</span>
							{:else}
								<Copy class="mr-1 h-3.5 w-3.5 text-indigo-400" />
								<span>Copy</span>
							{/if}
						</Button>
					</div>

					<!-- Address Card -->
					<div class="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
						<div class="flex items-center gap-2 font-semibold text-slate-200">
							<MapPin class="h-4 w-4 text-indigo-400" />
							<span>Address & Location</span>
						</div>
						<div class="mt-2 space-y-1 text-slate-300">
							<p>{society.address_line1}</p>
							{#if society.address_line2}
								<p>{society.address_line2}</p>
							{/if}
							<p>{society.locality}</p>
							<p class="font-medium text-slate-200">
								{society.city}, {society.state} — {society.pincode}
							</p>
						</div>
					</div>

					<!-- Contact & Registration Info -->
					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<div class="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
							<div class="flex items-center gap-2 font-semibold text-slate-200">
								<FileText class="h-4 w-4 text-indigo-400" />
								<span>Legal Registration</span>
							</div>
							<div class="mt-2 text-slate-300">
								<span class="text-slate-400">Reg No:</span>
								<span class="font-mono">{society.registration_number || 'Not Registered / NA'}</span
								>
							</div>
						</div>

						<div class="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
							<div class="flex items-center gap-2 font-semibold text-slate-200">
								<Mail class="h-4 w-4 text-indigo-400" />
								<span>Contact Information</span>
							</div>
							<div class="mt-2 space-y-1 text-slate-300">
								<div class="truncate">
									<span class="text-slate-400">Email:</span>
									{society.contact_email || '—'}
								</div>
								<div>
									<span class="text-slate-400">Phone:</span>
									<span class="font-mono">{society.contact_phone || '—'}</span>
								</div>
							</div>
						</div>
					</div>

					<!-- Timestamps & Provenance -->
					<div class="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
						<div class="flex items-center gap-2 font-semibold text-slate-200">
							<Calendar class="h-4 w-4 text-indigo-400" />
							<span>Platform Audit Metadata</span>
						</div>
						<div class="mt-2 grid grid-cols-2 gap-2 text-slate-400">
							<div>
								<span>Created:</span>
								<strong class="text-slate-300">{formatDate(society.created_at)}</strong>
							</div>
							<div>
								<span>Last Updated:</span>
								<strong class="text-slate-300">{formatDate(society.updated_at)}</strong>
							</div>
							<div class="col-span-2 truncate">
								<span>Created By:</span>
								<span class="font-mono text-slate-300">{society.created_by}</span>
							</div>
						</div>
					</div>
				</div>
			{/if}

			<div class="mt-5 flex items-center justify-between border-t border-slate-800/80 pt-3">
				<div>
					{#if society.status === 'ACTIVE'}
						<Button
							variant="outline"
							size="sm"
							class="border-indigo-500/40 text-indigo-300 hover:bg-indigo-500/10"
							onclick={() => (isFormationOpen = true)}
						>
							<Layers class="mr-1.5 h-3.5 w-3.5 text-indigo-400" />
							<span>Formation & Structure</span>
						</Button>
					{/if}
				</div>
				<div class="flex items-center gap-2">
					{#if onEdit}
						<Button
							variant="outline"
							size="sm"
							onclick={() => {
								onClose();
								if (society) onEdit(society);
							}}
						>
							Edit Profile
						</Button>
					{/if}
					<Button variant="primary" size="sm" onclick={onClose}>Close</Button>
				</div>
			</div>
		</div>
	</div>

	<!-- Formation Wizard Dialog -->
	<FormationWizard
		{society}
		isOpen={isFormationOpen}
		onClose={() => (isFormationOpen = false)}
		onComplete={() => {
			isFormationOpen = false;
		}}
	/>
{/if}
