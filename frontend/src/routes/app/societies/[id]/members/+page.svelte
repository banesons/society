<script lang="ts">
	import { page } from '$app/state';
	import MemberRoster from '$lib/components/members/member-roster.svelte';

	let societyId = $derived(page.params.id ?? '');
</script>

<svelte:head>
	<title>Members & Residents — CHS Neu</title>
</svelte:head>

<div class="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
	{#if societyId}
		<MemberRoster {societyId} />
	{/if}
</div>
