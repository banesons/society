<script lang="ts">
	import { page } from '$app/state';
	import { auth } from '$lib/auth.svelte';
	import { getEffectiveSocietyRole } from '$lib/navigation';
	import ResidentBillsView from '$lib/components/resident/resident-bills-view.svelte';
	import MaintenanceConsole from '$lib/components/admin/maintenance-console.svelte';

	let societyId = $derived(page.params.id ?? '');
	let effectiveRole = $derived(
		getEffectiveSocietyRole(auth.user?.platform_roles, auth.user?.society_assignments, societyId)
	);
	let isSocAdmin = $derived(
		effectiveRole === 'SOC_ADMIN' || effectiveRole === 'SUPER_ADMIN' || effectiveRole === 'ADMIN'
	);
</script>

<svelte:head>
	<title>Maintenance & Billing — CHS Neu</title>
</svelte:head>

<div class="mx-auto w-full max-w-7xl space-y-6 px-4 py-6 sm:px-6 lg:px-8">
	{#if societyId}
		<!-- Resident Bills & Dues -->
		<ResidentBillsView {societyId} />

		<!-- Admin Billing & Maintenance Console -->
		{#if isSocAdmin}
			<MaintenanceConsole {societyId} />
		{/if}
	{/if}
</div>
