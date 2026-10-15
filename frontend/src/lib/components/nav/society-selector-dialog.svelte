<script lang="ts">
	import { goto } from '$app/navigation';
	import type { UserSocietyAssignment } from '$lib/api/society';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { Building2, MapPin, ArrowRight, X, ShieldCheck } from '@lucide/svelte';

	interface Props {
		isOpen: boolean;
		societies: UserSocietyAssignment[];
		currentSocietyId?: string;
		isDismissible?: boolean;
		onClose: () => void;
		onSelect?: (society: UserSocietyAssignment) => void;
	}

	let {
		isOpen = false,
		societies = [],
		currentSocietyId = '',
		isDismissible = true,
		onClose,
		onSelect
	}: Props = $props();

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && isDismissible) {
			onClose();
		}
	}

	function selectSociety(society: UserSocietyAssignment) {
		if (onSelect) {
			onSelect(society);
		} else {
			goto(`/society/${society.society_id}`);
			onClose();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-end justify-center bg-neutral-950/40 p-0 backdrop-blur-xs sm:items-center sm:p-4"
		role="dialog"
		aria-modal="true"
		aria-labelledby="society-selector-title"
	>
		<!-- Modal / Mobile Bottom Sheet -->
		<div
			class="relative flex max-h-[90vh] w-full max-w-lg flex-col rounded-t-[2rem] border border-neutral-200 bg-white p-6 text-neutral-900 shadow-2xl sm:rounded-3xl"
		>
			<!-- Header -->
			<div class="flex items-center justify-between border-b border-neutral-200 pb-4">
				<div class="flex items-center gap-3">
					<div
						class="flex h-10 w-10 items-center justify-center rounded-2xl bg-neutral-100 text-neutral-900"
					>
						<Building2 class="h-5 w-5" />
					</div>
					<div>
						<h3
							id="society-selector-title"
							class="font-serif text-lg font-medium tracking-tight text-neutral-950"
						>
							Select Housing Society
						</h3>
						<p class="text-xs text-neutral-500">Choose an authorized society tenant to manage</p>
					</div>
				</div>

				{#if isDismissible}
					<button
						type="button"
						onclick={onClose}
						class="rounded-full p-2 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
						aria-label="Close dialog"
					>
						<X class="h-5 w-5" />
					</button>
				{/if}
			</div>

			<!-- Society Stack Cards List -->
			<div class="mt-4 flex-1 space-y-3 overflow-y-auto py-1">
				{#if societies.length === 0}
					<div class="py-12 text-center text-xs text-neutral-500">
						<Building2 class="mx-auto h-8 w-8 text-neutral-300" />
						<p class="mt-2 font-medium text-neutral-700">No accessible societies</p>
						<p class="mt-1">You are not currently assigned as an administrator to any society.</p>
					</div>
				{:else}
					{#each societies as item (item.society_id)}
						{@const isCurrent = item.society_id === currentSocietyId}
						<button
							type="button"
							onclick={() => selectSociety(item)}
							class="group flex w-full flex-col gap-2 rounded-2xl border p-4 text-left transition-all {isCurrent
								? 'border-neutral-950 bg-neutral-50/80 shadow-xs'
								: 'border-neutral-200 bg-white hover:border-neutral-400 hover:bg-neutral-50/50'}"
						>
							<div class="flex items-start justify-between gap-3">
								<div>
									<div class="flex items-center gap-2">
										<h4 class="text-sm font-semibold text-neutral-950">
											{item.society_name}
										</h4>
										{#if isCurrent}
											<Badge
												variant="outline"
												class="border-neutral-950 bg-neutral-950 text-[10px] text-white"
											>
												Active Context
											</Badge>
										{/if}
									</div>
									<div class="mt-1 flex items-center gap-2 font-mono text-xs text-neutral-600">
										<span class="rounded bg-neutral-100 px-1.5 py-0.5 font-bold text-neutral-900">
											{item.society_code}
										</span>
										{#if item.city}
											<span class="text-neutral-400">·</span>
											<span class="inline-flex items-center gap-1 font-sans text-neutral-500">
												<MapPin class="h-3 w-3" />
												{item.city}{item.state ? `, ${item.state}` : ''}
											</span>
										{/if}
									</div>
								</div>

								<div
									class="flex h-8 w-8 items-center justify-center rounded-full border border-neutral-200 bg-white text-neutral-400 transition-colors group-hover:border-neutral-950 group-hover:bg-neutral-950 group-hover:text-white"
								>
									<ArrowRight class="h-4 w-4" />
								</div>
							</div>

							<div
								class="mt-1 flex items-center justify-between border-t border-neutral-100 pt-2 text-xs"
							>
								<span class="inline-flex items-center gap-1 font-medium text-neutral-700">
									<ShieldCheck class="h-3.5 w-3.5 text-neutral-500" />
									Role: <span class="font-mono text-neutral-900">{item.role}</span>
								</span>
								{#if item.status}
									<Badge variant="success" class="text-[10px]">
										{item.status}
									</Badge>
								{/if}
							</div>
						</button>
					{/each}
				{/if}
			</div>

			<!-- Footer -->
			{#if isDismissible}
				<div class="mt-4 flex items-center justify-end border-t border-neutral-200 pt-4">
					<Button variant="ghost" size="sm" onclick={onClose}>Close</Button>
				</div>
			{/if}
		</div>
	</div>
{/if}
