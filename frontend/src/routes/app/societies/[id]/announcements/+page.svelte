<script lang="ts">
	import { page } from '$app/state';
	import { auth } from '$lib/auth.svelte';
	import { getEffectiveSocietyRole } from '$lib/navigation';
	import ResidentAnnouncementFeed from '$lib/components/resident/resident-announcement-feed.svelte';
	import AnnouncementConsole from '$lib/components/admin/announcement-console.svelte';

	let societyId = $derived(page.params.id ?? '');
	let effectiveRole = $derived(
		getEffectiveSocietyRole(auth.user?.platform_roles, auth.user?.society_assignments, societyId)
	);
	let isSocAdmin = $derived(
		effectiveRole === 'SOC_ADMIN' || effectiveRole === 'SUPER_ADMIN' || effectiveRole === 'ADMIN'
	);
</script>

<svelte:head>
	<title>Announcements & Broadcasts — CHS Neu</title>
</svelte:head>

<div class="mx-auto w-full max-w-7xl space-y-6 px-4 py-6 sm:px-6 lg:px-8">
	{#if societyId}
		<ResidentAnnouncementFeed {societyId} />

		{#if isSocAdmin}
			<AnnouncementConsole {societyId} />
		{/if}
	{/if}
</div>
