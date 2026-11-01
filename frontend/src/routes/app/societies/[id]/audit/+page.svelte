<script lang="ts">
	import { page } from '$app/state';
	import SocietyAuditConsole from '$lib/components/admin/society-audit-console.svelte';

	let societyId = $derived(page.params.id ?? '');
</script>

<svelte:head>
	<title>Audit & Activity — CHS Neu</title>
</svelte:head>

<div class="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
	{#if societyId}
		<SocietyAuditConsole {societyId} />
	{/if}
</div>
