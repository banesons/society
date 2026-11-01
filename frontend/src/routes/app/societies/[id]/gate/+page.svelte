<script lang="ts">
	import { page } from '$app/state';
	import GateConsole from '$lib/components/gate/gate-console.svelte';

	let societyId = $derived(page.params.id ?? '');
</script>

<svelte:head>
	<title>Gatekeeping Console — CHS Neu</title>
</svelte:head>

<div class="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
	{#if societyId}
		<GateConsole {societyId} />
	{/if}
</div>
