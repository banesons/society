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
			class="flex items-center gap-2 rounded-lg border border-slate-700/80 bg-slate-800/60 px-2.5 py-1.5 text-xs font-medium text-slate-200 transition-colors hover:bg-slate-800 hover:text-white"
			aria-expanded={isOpen}
			aria-haspopup="true"
		>
			<Building2 class="h-3.5 w-3.5 text-indigo-400" />
			<span class="max-w-[120px] truncate sm:max-w-[160px]">
				Societies ({assignments.length})
			</span>
			<ChevronDown class="h-3 w-3 text-slate-400" />
		</button>

		{#if isOpen}
			<div
				class="animate-in fade-in zoom-in-95 absolute right-0 z-50 mt-1.5 w-64 origin-top-right rounded-xl border border-slate-800 bg-slate-900 p-1.5 shadow-2xl duration-100"
			>
				<div class="px-2.5 py-1 text-[10px] font-semibold tracking-wider text-slate-400 uppercase">
					Assigned Societies
				</div>
				<div class="mt-1 space-y-0.5">
					{#each assignments as assignment (assignment.society_id)}
						<a
							href="/society/{assignment.society_id}"
							onclick={handleSelect}
							class="flex flex-col rounded-lg px-2.5 py-2 text-xs text-slate-300 transition-colors hover:bg-slate-800/80 hover:text-white"
						>
							<div class="truncate font-medium text-slate-100">
								{assignment.society_name}
							</div>
							<div class="flex items-center gap-1.5 font-mono text-[10px] text-slate-400">
								<span>{assignment.society_code}</span>
								<span>·</span>
								<span class="text-indigo-400">{assignment.role}</span>
							</div>
						</a>
					{/each}
				</div>
			</div>
		{/if}
	</div>
{/if}
