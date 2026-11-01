<script lang="ts">
	import { page } from '$app/state';
	import { auth } from '$lib/auth.svelte';
	import { societyApi, type SocietyProfile } from '$lib/api/society';

	interface Crumb {
		label: string;
		href?: string;
	}

	interface Props {
		societyId?: string | null;
		societyName?: string | null;
		crumbs?: Crumb[];
	}

	let { societyId, societyName, crumbs }: Props = $props();

	let currentPath = $derived(page.url.pathname);

	// Society name resolution: props > auth.user assignments > async fetched profile
	let fetchedSocietyName = $state<string | null>(null);

	$effect(() => {
		const targetId = societyId || page.params.id;
		if (!societyName && targetId && !auth.isLoading) {
			const matched = auth.user?.society_assignments?.find((a) => a.society_id === targetId);
			if (matched?.society_name) {
				fetchedSocietyName = matched.society_name;
			} else {
				societyApi
					.getSocietyProfile(targetId)
					.then((res: SocietyProfile) => {
						fetchedSocietyName = res.name;
					})
					.catch(() => {
						fetchedSocietyName = 'Society Workspace';
					});
			}
		}
	});

	let resolvedSocietyName = $derived(
		societyName ||
			fetchedSocietyName ||
			auth.user?.society_assignments?.find((a) => a.society_id === (societyId || page.params.id))
				?.society_name ||
			'Society'
	);

	const MODULE_LABELS: Record<string, string> = {
		announcements: 'Announcements',
		maintenance: 'Maintenance',
		'resident-hub': 'Resident Visitor Hub',
		documents: 'Documents',
		members: 'Members & Residents',
		committee: 'Managing Committee',
		structure: 'Physical Structure',
		gate: 'Gate Management',
		reports: 'Reports & Analytics',
		audit: 'Audit Logs'
	};

	let resolvedCrumbs = $derived.by<Crumb[]>(() => {
		if (crumbs && crumbs.length > 0) return crumbs;

		// 1. Platform Administration
		if (currentPath.startsWith('/app/admin') || currentPath.startsWith('/admin')) {
			const items: Crumb[] = [{ label: 'Platform Administration', href: '/app/admin' }];
			if (currentPath.includes('/societies')) {
				items.push({ label: 'Societies' });
			} else if (currentPath !== '/app/admin' && currentPath !== '/admin') {
				items.push({ label: 'Console' });
			}
			return items;
		}

		// 2. Personal Profile
		if (currentPath.startsWith('/app/profile')) {
			return [{ label: 'Workspace', href: '/app/dashboard' }, { label: 'Personal Profile' }];
		}

		// 3. Notifications
		if (currentPath.startsWith('/app/notifications')) {
			return [{ label: 'Workspace', href: '/app/dashboard' }, { label: 'Notifications' }];
		}

		// 4. Societies Directory
		if (currentPath === '/app/societies' || currentPath === '/app/societies/') {
			return [{ label: 'Workspace', href: '/app/dashboard' }, { label: 'Societies' }];
		}

		// 5. Society Modules (/app/societies/{id}/*)
		const societyMatch = currentPath.match(/^\/app\/societies\/([^/?#]+)(?:\/([^/?#]+))?/);
		if (societyMatch) {
			const sid = societyMatch[1];
			const subModule = societyMatch[2];

			const items: Crumb[] = [
				{ label: 'Workspace', href: '/app/dashboard' },
				{
					label: resolvedSocietyName,
					href: subModule ? `/app/societies/${sid}` : undefined
				}
			];

			if (subModule) {
				const label = MODULE_LABELS[subModule] || subModule.replace(/-/g, ' ');
				items.push({ label });
			}

			return items;
		}

		// Default Workspace fallback
		return [{ label: 'Workspace', href: '/app/dashboard' }, { label: 'Dashboard' }];
	});
</script>

<nav aria-label="Breadcrumb" class="flex items-center">
	<ol class="flex flex-wrap items-center gap-1.5 font-sans text-xs text-neutral-500">
		{#each resolvedCrumbs as crumb, index (index)}
			{@const isLast = index === resolvedCrumbs.length - 1}
			<li class="flex min-w-0 items-center gap-1.5">
				{#if index > 0}
					<span class="text-neutral-300 select-none" aria-hidden="true">/</span>
				{/if}

				{#if isLast}
					<span
						class="max-w-[180px] truncate font-semibold text-neutral-950 sm:max-w-xs md:max-w-md"
						aria-current="page"
					>
						{crumb.label}
					</span>
				{:else if crumb.href}
					<a
						href={crumb.href}
						class="max-w-[120px] truncate transition-colors hover:text-neutral-900 sm:max-w-[200px]"
					>
						{crumb.label}
					</a>
				{:else}
					<span class="max-w-[120px] truncate sm:max-w-[200px]">
						{crumb.label}
					</span>
				{/if}
			</li>
		{/each}
	</ol>
</nav>
