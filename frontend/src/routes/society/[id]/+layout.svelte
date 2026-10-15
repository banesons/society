<script lang="ts">
	import { setContext, type Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { auth } from '$lib/auth.svelte';
	import { societyApi, type SocietyProfile, type UserSocietyAssignment } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { getContextualBackDestination, type BackDestination } from '$lib/routing';
	import SocietyShellHeader from '$lib/components/nav/society-shell-header.svelte';
	import SocietyDrawer from '$lib/components/nav/society-drawer.svelte';
	import SocietySelectorDialog from '$lib/components/nav/society-selector-dialog.svelte';
	import LogoutAllDialog from '$lib/components/auth/logout-all-dialog.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { ArrowLeft } from '@lucide/svelte';

	let { children }: { children: Snippet } = $props();

	const queryClient = useQueryClient();

	let societyId = $derived(page.params.id);
	let profile = $state<SocietyProfile | null>(null);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let errorCode = $state<string | null>(null);

	let isMenuOpen = $state(false);
	let isSelectorOpen = $state(false);
	let isLogoutAllOpen = $state(false);

	let backDestination = $derived<BackDestination>(getContextualBackDestination(auth.user));
	let accessibleSocieties = $state<UserSocietyAssignment[]>([]);

	$effect(() => {
		if (auth.isSuperAdmin) {
			societyApi
				.listSocieties({ limit: 100 })
				.then((res) => {
					accessibleSocieties = res.items
						.filter((s) => s.status === 'ACTIVE')
						.map((s) => ({
							society_id: s.id,
							society_code: s.society_code,
							society_name: s.name,
							role: 'SUPER_ADMIN',
							status: s.status as 'ACTIVE',
							city: s.city,
							state: s.state
						}));
				})
				.catch(() => {
					accessibleSocieties = auth.user?.society_assignments ?? [];
				});
		} else {
			accessibleSocieties = auth.user?.society_assignments ?? [];
		}
	});

	let hasMultipleSocieties = $derived(accessibleSocieties.length > 1);

	async function loadProfile() {
		if (!societyId) return;
		profile = null;
		isLoading = true;
		error = null;
		errorCode = null;

		try {
			const res = await societyApi.getSocietyProfile(societyId);
			profile = res;
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				errorCode = err.code;
				if (err.code === 'SOCIETY_PENDING') {
					error =
						'This society has not been activated yet. Operations will be available once onboarding review completes.';
				} else if (err.code === 'SOCIETY_SUSPENDED') {
					error =
						'This society is currently suspended. Operational access is blocked until platform reactivation.';
				} else if (err.code === 'SOCIETY_ACCESS_DENIED') {
					error =
						'Access denied. You do not hold an active administrator assignment for this society.';
				} else {
					error = err.message;
				}
			} else {
				error = 'Failed to load society profile.';
			}
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (societyId && !auth.isLoading) {
			loadProfile();
		}
	});

	// Provide society context to child routes
	setContext('societyShell', {
		get profile() {
			return profile;
		},
		get isLoading() {
			return isLoading;
		},
		get error() {
			return error;
		},
		get errorCode() {
			return errorCode;
		},
		reloadProfile: loadProfile
	});

	async function handleSwitchSociety(target: UserSocietyAssignment) {
		if (target.society_id === societyId) {
			isSelectorOpen = false;
			return;
		}

		// Tenant Isolation: Invalidate and purge tenant-specific query cache
		queryClient.removeQueries({ queryKey: ['society', societyId] });
		queryClient.invalidateQueries({ queryKey: ['society'] });

		isSelectorOpen = false;
		await goto(`/society/${target.society_id}`);
	}
	let currentRole = $derived(() => {
		if (auth.isSuperAdmin) return 'SUPER_ADMIN';
		if (auth.isPlatformAdmin) return 'ADMIN';
		const match = auth.user?.society_assignments?.find((a) => a.society_id === societyId);
		return match?.role || 'SOC_ADMIN';
	});
</script>

<div class="flex min-h-screen flex-col bg-[#f3f3f1] font-sans text-neutral-900">
	<!-- Dedicated Society Application Header -->
	<SocietyShellHeader
		{profile}
		{backDestination}
		{isMenuOpen}
		onToggleMenu={() => (isMenuOpen = !isMenuOpen)}
	/>

	<!-- Desktop Back-Navigation Row (hidden on mobile since mobile header contains leading back arrow) -->
	<div class="mx-auto w-full max-w-7xl px-4 pt-6 sm:px-6 lg:px-8">
		<div class="mb-6 hidden items-center justify-between sm:flex">
			<a
				href={backDestination.href}
				class="inline-flex items-center gap-1.5 text-xs font-medium text-neutral-500 transition-colors hover:text-neutral-900"
			>
				<ArrowLeft class="h-3.5 w-3.5" />
				<span>{backDestination.label}</span>
			</a>
			<div class="flex items-center gap-2">
				<Badge variant="outline" class="bg-white/80 font-mono text-xs text-neutral-700">
					Tenant Boundary: {currentRole()}
				</Badge>
			</div>
		</div>
	</div>

	<!-- Main Society Application Content -->
	<main class="flex-1">
		{@render children()}
	</main>

	<!-- Slide-in Society Navigation & Account Drawer -->
	<SocietyDrawer
		isOpen={isMenuOpen}
		{profile}
		societyId={societyId ?? ''}
		{hasMultipleSocieties}
		onClose={() => (isMenuOpen = false)}
		onSwitchSociety={() => (isSelectorOpen = true)}
		onLogoutAll={() => (isLogoutAllOpen = true)}
	/>

	<!-- Society Switcher / Selector Dialog -->
	<SocietySelectorDialog
		isOpen={isSelectorOpen}
		societies={accessibleSocieties}
		currentSocietyId={societyId ?? ''}
		isDismissible={true}
		onClose={() => (isSelectorOpen = false)}
		onSelect={handleSwitchSociety}
	/>

	<!-- Global Logout All Dialog -->
	<LogoutAllDialog isOpen={isLogoutAllOpen} onClose={() => (isLogoutAllOpen = false)} />
</div>
