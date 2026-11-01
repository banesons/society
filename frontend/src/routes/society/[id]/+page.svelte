<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { appRoutes } from '$lib/routing';
	import Card from '$lib/components/ui/card.svelte';
	import { Loader2 } from '@lucide/svelte';

	onMount(() => {
		const id = page.params.id;
		if (id) {
			const hash = window.location.hash;
			const target = appRoutes.society(id, hash);
			goto(target, { replaceState: true });
		}
	});
</script>

<svelte:head>
	<title>Redirecting to Society Workspace...</title>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-[#f3f3f1] p-8">
	<Card
		class="flex flex-col items-center justify-center border-neutral-200 bg-white p-12 text-center shadow-xs"
	>
		<Loader2 class="h-8 w-8 animate-spin text-neutral-900" />
		<p class="mt-4 font-serif text-sm font-medium text-neutral-950">
			Redirecting to canonical society workspace...
		</p>
	</Card>
</div>
