<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import { appRoutes } from '$lib/routing';
	import Card from '$lib/components/ui/card.svelte';
	import { Loader2 } from '@lucide/svelte';

	onMount(() => {
		if (!auth.isLoading) {
			if (auth.isAuthenticated) {
				goto(appRoutes.dashboard, { replaceState: true });
			} else {
				goto(appRoutes.login(), { replaceState: true });
			}
		}
	});

	$effect(() => {
		if (!auth.isLoading) {
			if (auth.isAuthenticated) {
				goto(appRoutes.dashboard, { replaceState: true });
			} else {
				goto(appRoutes.login(), { replaceState: true });
			}
		}
	});
</script>

<svelte:head>
	<title>CHS Application Entry</title>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div class="flex flex-1 items-center justify-center p-12">
	<Card
		class="flex flex-col items-center justify-center border-neutral-200 bg-white/90 p-12 text-center shadow-xs"
	>
		<Loader2 class="h-8 w-8 animate-spin text-neutral-900" />
		<p class="mt-4 font-serif text-sm font-medium text-neutral-950">
			Loading CHS application workspace...
		</p>
	</Card>
</div>
