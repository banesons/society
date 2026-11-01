<script lang="ts">
	import { page } from '$app/state';
	import CommitteeRoster from '$lib/components/committee/committee-roster.svelte';

	let societyId = $derived(page.params.id ?? '');
</script>

<svelte:head>
	<title>Managing Committee — CHS Neu</title>
</svelte:head>

<div class="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
	{#if societyId}
		<CommitteeRoster {societyId} />
	{/if}
</div>
