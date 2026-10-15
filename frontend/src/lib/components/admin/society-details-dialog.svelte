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
			class="fixed inset-0 bg-neutral-950/40 backdrop-blur-xs transition-opacity"
			onclick={onClose}
			aria-label="Close dialog overlay"
		></button>

		<!-- Modal Container -->
		<div
			class="relative max-h-[90vh] w-full max-w-xl overflow-y-auto rounded-3xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div class="flex items-center gap-2.5">
					<div
						class="flex h-9 w-9 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
					>
						<Building2 class="h-4 w-4" />
					</div>
					<div>
						<h2
							id="society-details-title"
							class="text-base font-semibold tracking-tight text-neutral-900"
						>
							{society.name}
						</h2>
						<div class="mt-0.5 flex items-center gap-2 text-xs">
							<Badge variant={getStatusVariant(society.status)}>{society.status}</Badge>
							<span class="text-neutral-500">· {society.city}, {society.state}</span>
						</div>
					</div>
				</div>
				<button
					type="button"
					onclick={onClose}
					class="rounded-full p-1.5 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
					aria-label="Close dialog"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<!-- Tab Navigation -->
			<div class="mt-4 flex items-center gap-1 border-b border-neutral-100 pb-2">
				<button
					type="button"
					onclick={() => (activeTab = 'overview')}
					class={`flex items-center gap-1.5 rounded-full px-3.5 py-1.5 text-xs font-medium transition-colors ${
						activeTab === 'overview'
							? 'bg-neutral-950 text-white'
							: 'text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900'
					}`}
				>
					<Building2 class="h-3.5 w-3.5" />
					<span>Overview</span>
				</button>
				<button
					type="button"
					onclick={() => (activeTab = 'admins')}
					class={`flex items-center gap-1.5 rounded-full px-3.5 py-1.5 text-xs font-medium transition-colors ${
						activeTab === 'admins'
							? 'bg-neutral-950 text-white'
							: 'text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900'
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
						class="flex items-center justify-between rounded-2xl border border-neutral-200 bg-neutral-50 p-4"
					>
						<div>
							<div class="text-[11px] font-semibold tracking-wider text-neutral-500 uppercase">
								Society Code (Immutable)
							</div>
							<div class="mt-0.5 font-mono text-base font-bold text-neutral-950">
								{society.society_code}
							</div>
						</div>
						<Button
							variant="outline"
							size="sm"
							onclick={copyCode}
							class="border-neutral-300 text-xs hover:bg-neutral-100"
						>
							{#if copied}
								<Check class="mr-1 h-3.5 w-3.5 text-emerald-600" />
								<span class="text-emerald-700">Copied</span>
							{:else}
								<Copy class="mr-1 h-3.5 w-3.5 text-neutral-700" />
								<span>Copy</span>
							{/if}
						</Button>
					</div>

					<!-- Address Card -->
					<div class="rounded-2xl border border-neutral-200 bg-neutral-50/60 p-4">
						<div class="flex items-center gap-2 font-semibold text-neutral-900">
							<MapPin class="h-4 w-4 text-neutral-700" />
							<span>Address & Location</span>
						</div>
						<div class="mt-2 space-y-1 text-neutral-700">
							<p>{society.address_line1}</p>
							{#if society.address_line2}
								<p>{society.address_line2}</p>
							{/if}
							<p>{society.locality}</p>
							<p class="font-medium text-neutral-900">
								{society.city}, {society.state} — {society.pincode}
							</p>
						</div>
					</div>

					<!-- Contact & Registration Info -->
					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<div class="rounded-2xl border border-neutral-200 bg-neutral-50/60 p-4">
							<div class="flex items-center gap-2 font-semibold text-neutral-900">
								<FileText class="h-4 w-4 text-neutral-700" />
								<span>Legal Registration</span>
							</div>
							<div class="mt-2 text-neutral-700">
								<span class="text-neutral-500">Reg No:</span>
								<span class="ml-1 font-mono text-neutral-900"
									>{society.registration_number || 'Not Registered / NA'}</span
								>
							</div>
						</div>

						<div class="rounded-2xl border border-neutral-200 bg-neutral-50/60 p-4">
							<div class="flex items-center gap-2 font-semibold text-neutral-900">
								<Mail class="h-4 w-4 text-neutral-700" />
								<span>Contact Information</span>
							</div>
							<div class="mt-2 space-y-1 text-neutral-700">
								<div class="truncate">
									<span class="text-neutral-500">Email:</span>
									<span class="ml-1 text-neutral-900">{society.contact_email || '—'}</span>
								</div>
								<div>
									<span class="text-neutral-500">Phone:</span>
									<span class="ml-1 font-mono text-neutral-900">{society.contact_phone || '—'}</span
									>
								</div>
							</div>
						</div>
					</div>

					<!-- Timestamps & Provenance -->
					<div class="rounded-2xl border border-neutral-200 bg-neutral-50/60 p-4">
						<div class="flex items-center gap-2 font-semibold text-neutral-900">
							<Calendar class="h-4 w-4 text-neutral-700" />
							<span>Platform Audit Metadata</span>
						</div>
						<div class="mt-2 grid grid-cols-2 gap-2 text-neutral-600">
							<div>
								<span>Created:</span>
								<strong class="ml-1 text-neutral-900">{formatDate(society.created_at)}</strong>
							</div>
							<div>
								<span>Last Updated:</span>
								<strong class="ml-1 text-neutral-900">{formatDate(society.updated_at)}</strong>
							</div>
							<div class="col-span-2 truncate">
								<span>Created By:</span>
								<span class="ml-1 font-mono text-neutral-900">{society.created_by}</span>
							</div>
						</div>
					</div>
				</div>
			{/if}

			<div class="mt-5 flex items-center justify-between border-t border-neutral-100 pt-3">
				<div>
					{#if society.status === 'ACTIVE'}
						<Button
							variant="outline"
							size="sm"
							class="border-neutral-300 text-neutral-800 hover:bg-neutral-100"
							onclick={() => (isFormationOpen = true)}
						>
							<Layers class="mr-1.5 h-3.5 w-3.5 text-neutral-700" />
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
							class="border-neutral-300 hover:bg-neutral-100"
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
