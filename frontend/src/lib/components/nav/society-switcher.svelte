<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import { Building2, ChevronDown } from '@lucide/svelte';
	import type { UserSocietyAssignment } from '$lib/api/society';

	let isOpen = $state(false);

	let assignments = $derived<UserSocietyAssignment[]>(auth.user?.society_assignments || []);

	function handleSelect() {
		isOpen = false;
	}
</script>

{#if assignments.length > 0}
	<div class="relative">
		<button
			type="button"
			onclick={() => (isOpen = !isOpen)}
			class="flex items-center gap-2 rounded-full border border-neutral-300 bg-white/80 px-3 py-1.5 text-xs font-medium text-neutral-800 transition-colors hover:border-neutral-400 hover:bg-white"
			aria-expanded={isOpen}
			aria-haspopup="true"
		>
			<Building2 class="h-3.5 w-3.5 text-neutral-700" />
			<span class="max-w-[120px] truncate sm:max-w-[160px]">
				Societies ({assignments.length})
			</span>
			<ChevronDown class="h-3 w-3 text-neutral-500" />
		</button>

		{#if isOpen}
			<div
				class="animate-in fade-in zoom-in-95 absolute right-0 z-50 mt-1.5 w-64 origin-top-right rounded-2xl border border-neutral-200 bg-white p-2 shadow-xl duration-100"
			>
				<div class="px-3 py-1 text-[10px] font-semibold tracking-wider text-neutral-400 uppercase">
					Assigned Societies
				</div>
				<div class="mt-1 space-y-0.5">
					{#each assignments as assignment (assignment.society_id)}
						<a
							href="/society/{assignment.society_id}"
							onclick={handleSelect}
							class="flex flex-col rounded-xl px-3 py-2 text-xs text-neutral-700 transition-colors hover:bg-neutral-100 hover:text-neutral-950"
						>
							<div class="truncate font-medium text-neutral-900">
								{assignment.society_name}
							</div>
							<div class="flex items-center gap-1.5 font-mono text-[10px] text-neutral-500">
								<span>{assignment.society_code}</span>
								<span>·</span>
								<span class="font-medium text-neutral-900">{assignment.role}</span>
							</div>
						</a>
					{/each}
				</div>
			</div>
		{/if}
	</div>
{/if}
