<script lang="ts">
	import { onMount } from 'svelte';
	import { auth } from '$lib/auth.svelte';
	import { formatFullPhone } from '$lib/phone';
	import { toast } from '$lib/toast.svelte';
	import { getInitials } from '$lib/avatar';
	import {
		getPersonalProfile,
		updatePersonalProfile,
		getActiveSessions,
		revokeActiveSession,
		type UserPersonalProfile,
		type ActiveSession
	} from '$lib/api/profile';
	import { getDailyBackdrop, type DailyBackdrop } from '$lib/backdrop';
	import Breadcrumbs from '$lib/components/nav/breadcrumbs.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import SignOutDialog from '$lib/components/auth/sign-out-dialog.svelte';
	import LogoutAllDialog from '$lib/components/auth/logout-all-dialog.svelte';
	import ActivityTimeline from '$lib/components/profile/activity-timeline.svelte';
	import MyVisitorHistory from '$lib/components/profile/my-visitor-history.svelte';
	import { pwa } from '$lib/pwa.svelte';
	import {
		ShieldCheck,
		CheckCircle2,
		Building2,
		Edit3,
		Check,
		X,
		Loader2,
		Calendar,
		Mail,
		Phone,
		Lock,
		Upload,
		Trash2,
		LogOut,
		Smartphone,
		Laptop,
		Globe
	} from '@lucide/svelte';

	// State
	let profile = $state<UserPersonalProfile | null>(null);
	let activeSessions = $state<ActiveSession[]>([]);
	let isLoading = $state(true);
	let isLoadingSessions = $state(false);
	let revokingSessionId = $state<string | null>(null);
	let isEditing = $state(false);
	let isSaving = $state(false);
	let isSignOutOpen = $state(false);
	let isLogoutAllOpen = $state(false);
	let isUploadingPhoto = $state(false);
	let backdrop = $state<DailyBackdrop>(getDailyBackdrop());
	let backdropFailed = $state(false);

	// Editable form state
	let firstName = $state('');
	let lastName = $state('');
	let email = $state('');
	let fieldErrors = $state<{ firstName?: string; email?: string }>({});

	// Profile avatar state (supports local preview & persistent URL)
	let avatarUrl = $state<string | null>(null);
	let avatarLoadFailed = $state(false);
	let fileInputRef = $state<HTMLInputElement | null>(null);

	async function loadProfile() {
		isLoading = true;
		try {
			const res = await getPersonalProfile();
			profile = res;
			firstName = res.first_name || '';
			lastName = res.last_name || '';
			email = res.email || '';
			avatarUrl = res.profile_photo_url || null;
			avatarLoadFailed = false;

			// Sync with auth store
			auth.updateProfile({
				first_name: res.first_name,
				last_name: res.last_name,
				display_name: res.display_name,
				email: res.email,
				profile_photo_url: res.profile_photo_url
			});
		} catch {
			toast.error('Unable to retrieve personal profile information');
		} finally {
			isLoading = false;
		}
	}

	function startEditing() {
		if (!profile) return;
		firstName = profile.first_name || '';
		lastName = profile.last_name || '';
		email = profile.email || '';
		fieldErrors = {};
		isEditing = true;
	}

	function cancelEditing() {
		if (profile) {
			firstName = profile.first_name || '';
			lastName = profile.last_name || '';
			email = profile.email || '';
		}
		fieldErrors = {};
		isEditing = false;
	}

	function validate(): boolean {
		const errors: { firstName?: string; email?: string } = {};
		const cleanFirst = firstName.trim();
		const cleanEmail = email.trim();

		if (!cleanFirst) {
			errors.firstName = 'First name cannot be empty';
		} else if (cleanFirst.length > 100) {
			errors.firstName = 'First name must be under 100 characters';
		}

		if (cleanEmail) {
			const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
			if (!emailPattern.test(cleanEmail)) {
				errors.email = 'Please provide a valid email address';
			} else if (cleanEmail.length > 255) {
				errors.email = 'Email address exceeds maximum length';
			}
		}

		fieldErrors = errors;
		return Object.keys(errors).length === 0;
	}

	async function handleSave() {
		if (!validate() || isSaving) return;

		isSaving = true;
		try {
			const updated = await updatePersonalProfile({
				first_name: firstName.trim(),
				last_name: lastName.trim(),
				email: email.trim(),
				profile_photo_url: avatarUrl
			});

			profile = updated;
			firstName = updated.first_name || '';
			lastName = updated.last_name || '';
			email = updated.email || '';
			avatarUrl = updated.profile_photo_url || null;
			isEditing = false;
			toast.success('Personal profile updated successfully');

			// Sync auth session context immediately across entire shell
			auth.updateProfile({
				first_name: updated.first_name,
				last_name: updated.last_name,
				display_name: updated.display_name,
				email: updated.email,
				profile_photo_url: updated.profile_photo_url
			});
		} catch (err: unknown) {
			const msg =
				err instanceof Error && err.message
					? err.message
					: 'Failed to update profile. Please verify your details.';
			toast.error(msg);
		} finally {
			isSaving = false;
		}
	}

	function handlePhotoSelect(event: Event) {
		const target = event.target as HTMLInputElement;
		const files = target.files;
		if (!files || files.length === 0) return;

		const file = files[0];

		// Reuse established system validation rules: Max 5MB for images, jpg/png/webp
		const validTypes = ['image/jpeg', 'image/png', 'image/webp'];
		if (!validTypes.includes(file.type)) {
			toast.error('Please select a valid image file (JPEG, PNG, or WebP)');
			return;
		}

		const maxBytes = 5 * 1024 * 1024; // 5MB system limit
		if (file.size > maxBytes) {
			toast.error('Image file exceeds the 5MB size limit');
			return;
		}

		isUploadingPhoto = true;
		const reader = new FileReader();
		reader.onload = async () => {
			const dataUrl = reader.result as string;
			avatarUrl = dataUrl;
			avatarLoadFailed = false;

			try {
				const updated = await updatePersonalProfile({
					first_name: (profile?.first_name || firstName || 'User').trim(),
					last_name: (profile?.last_name || lastName || '').trim(),
					email: (profile?.email || email || '').trim(),
					profile_photo_url: dataUrl
				});
				profile = updated;
				avatarUrl = updated.profile_photo_url || dataUrl;
				auth.updateProfile({
					profile_photo_url: updated.profile_photo_url || dataUrl
				});
				toast.success('Profile photo updated successfully');
			} catch {
				toast.error('Failed to save profile photo');
			} finally {
				isUploadingPhoto = false;
			}
		};
		reader.onerror = () => {
			isUploadingPhoto = false;
			toast.error('Failed to read image file');
		};
		reader.readAsDataURL(file);
	}

	async function handleRemovePhoto() {
		avatarUrl = null;
		avatarLoadFailed = false;
		if (fileInputRef) {
			fileInputRef.value = '';
		}
		try {
			const updated = await updatePersonalProfile({
				first_name: (profile?.first_name || firstName || 'User').trim(),
				last_name: (profile?.last_name || lastName || '').trim(),
				email: (profile?.email || email || '').trim(),
				profile_photo_url: ''
			});
			profile = updated;
			avatarUrl = null;
			auth.updateProfile({
				profile_photo_url: null
			});
			toast.info('Profile portrait removed');
		} catch {
			toast.error('Failed to remove portrait');
		}
	}

	async function loadSessions() {
		isLoadingSessions = true;
		try {
			const res = await getActiveSessions();
			activeSessions = res.sessions || [];
		} catch {
			// Fail gracefully without breaking page
		} finally {
			isLoadingSessions = false;
		}
	}

	async function handleRevokeSession(s: ActiveSession) {
		if (s.is_current) {
			isSignOutOpen = true;
			return;
		}
		revokingSessionId = s.id;
		try {
			await revokeActiveSession(s.id);
			toast.success('Device access revoked successfully');
			await loadSessions();
		} catch (err: unknown) {
			const msg = err instanceof Error ? err.message : 'Failed to revoke device session';
			toast.error(msg);
		} finally {
			revokingSessionId = null;
		}
	}

	function getDeviceIcon(s: ActiveSession) {
		const os = s.os.toLowerCase();
		if (os.includes('ios') || os.includes('android')) return Smartphone;
		if (os.includes('mac') || os.includes('win') || os.includes('linux')) return Laptop;
		return Globe;
	}

	function formatRelativeActive(iso: string, isCurrent: boolean): string {
		if (isCurrent) return 'Active now';
		try {
			const date = new Date(iso);
			const diffMin = Math.round((Date.now() - date.getTime()) / 60000);
			if (diffMin < 2) return 'Active just now';
			if (diffMin < 60) return `Active ${diffMin}m ago`;
			const diffHours = Math.round(diffMin / 60);
			if (diffHours < 24) return `Active ${diffHours}h ago`;
			return `Active ${date.toLocaleDateString('en-IN', { month: 'short', day: 'numeric' })}`;
		} catch {
			return 'Recent session';
		}
	}

	onMount(() => {
		loadProfile();
		loadSessions();
	});

	let hasRealName = $derived.by(() => {
		if (isEditing) {
			return Boolean((firstName || '').trim());
		}
		return Boolean(profile?.first_name || profile?.last_name);
	});

	let computedDisplayName = $derived.by(() => {
		if (isEditing) {
			const full = `${firstName || ''} ${lastName || ''}`.trim();
			if (full) return full;
			return profile ? formatFullPhone(profile.phone_raw || profile.phone) : 'Account Identity';
		}
		if (!profile) return '';
		const full = `${profile.first_name || ''} ${profile.last_name || ''}`.trim();
		if (full) return full;
		if (profile.display_name && profile.display_name !== 'Verified Resident')
			return profile.display_name;
		return formatFullPhone(profile.phone_raw || profile.phone);
	});

	let computedInitials = $derived.by(() => {
		if (hasRealName) {
			return getInitials(computedDisplayName);
		}
		return 'CH';
	});

	let todayFormatted = $derived.by(() => {
		const now = new Date();
		return new Intl.DateTimeFormat('en-US', {
			weekday: 'long',
			month: 'long',
			day: 'numeric'
		}).format(now);
	});
</script>

<svelte:head>
	<title>Personal Identity & Profile — CHS Neu</title>
</svelte:head>

<div class="mx-auto w-full max-w-5xl px-3 pb-20 sm:px-6 lg:px-8">
	<!-- Hidden File Input for Accessible Avatar Upload -->
	<input
		type="file"
		accept="image/jpeg,image/png,image/webp"
		class="sr-only"
		bind:this={fileInputRef}
		onchange={handlePhotoSelect}
		id="profile-photo-input"
		aria-label="Upload profile photo"
	/>

	<!-- Top Editorial Breadcrumb / Section Header -->
	<div
		class="mb-4 flex flex-col gap-2 border-b border-neutral-200/80 pb-3 sm:mb-6 sm:flex-row sm:items-center sm:justify-between"
	>
		<Breadcrumbs />
		<div class="font-mono text-[11px] text-neutral-500 sm:text-xs">
			<span>{profile?.phone ? formatFullPhone(profile.phone) : 'Verified Identity'}</span>
		</div>
	</div>

	<!-- Cover Container with Overlapping Portrait -->
	<div class="relative mb-14 sm:mb-18 md:mb-20">
		<!-- Daily Photographic Backdrop Hero -->
		<div
			class="relative h-44 w-full overflow-hidden rounded-2xl border border-neutral-200/90 bg-neutral-900 sm:h-64 sm:rounded-3xl md:h-80 lg:h-96"
		>
			{#if !backdropFailed}
				<img
					src={backdrop.url}
					alt={backdrop.title}
					loading="eager"
					decoding="async"
					onerror={() => (backdropFailed = true)}
					class="h-full w-full object-cover transition-transform duration-700 ease-out hover:scale-102"
				/>
			{:else}
				<!-- Architectural fallback pattern when offline or image failure -->
				<div
					class="flex h-full w-full items-center justify-center bg-gradient-to-tr from-neutral-800 via-neutral-900 to-neutral-800"
				>
					<div
						class="h-full w-full bg-[radial-gradient(#525252_1px,transparent_1px)] [background-size:16px_16px] opacity-25"
					></div>
				</div>
			{/if}

			<!-- Restrained dark gradient overlay to ensure crisp contrast and legibility -->
			<div
				class="pointer-events-none absolute inset-0 bg-gradient-to-t from-black/75 via-black/30 to-black/20"
			></div>

			<!-- TOP-RIGHT: Current Date -->
			<div
				class="absolute top-3 right-3 flex items-center gap-2 rounded-full border border-white/20 bg-black/45 px-3 py-1.5 text-xs font-medium text-white/95 shadow-sm backdrop-blur-md select-none sm:top-4 sm:right-5 sm:gap-2.5 sm:px-4 sm:py-2 sm:text-sm md:text-base"
			>
				<Calendar class="h-3.5 w-3.5 text-white/90 sm:h-4 sm:w-4" />
				<span class="tracking-tight">{todayFormatted}</span>
			</div>

			<!-- BOTTOM-RIGHT: Photographer Attribution Credit -->
			{#if !backdropFailed && backdrop.photographer}
				<div
					class="absolute right-3 bottom-3 rounded-full border border-white/10 bg-black/30 px-2 py-0.5 text-[9px] text-white/75 backdrop-blur-xs select-none sm:right-5 sm:bottom-4 sm:px-2.5 sm:text-[10px]"
				>
					<span>Photo by {backdrop.photographer}</span>
				</div>
			{/if}
		</div>

		<!-- Overlapping Profile Portrait (Lower-Left Edge) -->
		<div class="absolute -bottom-8 left-4 sm:-bottom-12 sm:left-6 md:-bottom-14 md:left-8">
			<div class="group relative">
				<div
					class="flex h-20 w-20 items-center justify-center overflow-hidden rounded-full border-3 border-white bg-neutral-900 text-white shadow-md transition-transform group-hover:scale-102 sm:h-28 sm:w-28 sm:border-4 md:h-36 md:w-36"
				>
					{#if avatarUrl && !avatarLoadFailed}
						<img
							src={avatarUrl}
							alt="Profile portrait"
							onerror={() => (avatarLoadFailed = true)}
							class="h-full w-full object-cover"
						/>
					{:else}
						<!-- High-contrast Editorial Initials Fallback -->
						<div
							class="flex h-full w-full items-center justify-center bg-gradient-to-tr from-neutral-900 via-neutral-800 to-neutral-700 select-none"
						>
							<span
								class="font-serif text-xl font-semibold tracking-wider text-neutral-100 sm:text-3xl md:text-5xl"
							>
								{computedInitials}
							</span>
						</div>
					{/if}
				</div>

				<!-- Unobtrusive Accessible Photo Actions Overlay -->
				<div
					class="backdrop-blur-2xs absolute inset-0 flex items-center justify-center gap-1.5 rounded-full bg-black/50 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100"
				>
					<button
						type="button"
						onclick={() => fileInputRef?.click()}
						disabled={isUploadingPhoto}
						class="flex h-7 w-7 items-center justify-center rounded-full bg-white text-neutral-900 shadow-sm transition-transform hover:scale-110 active:scale-95 sm:h-8 sm:w-8"
						title={avatarUrl ? 'Replace portrait' : 'Upload portrait'}
						aria-label={avatarUrl ? 'Replace portrait' : 'Upload portrait'}
					>
						{#if isUploadingPhoto}
							<Loader2 class="h-3 w-3 animate-spin sm:h-3.5 sm:w-3.5" />
						{:else}
							<Upload class="h-3 w-3 sm:h-3.5 sm:w-3.5" />
						{/if}
					</button>

					{#if avatarUrl}
						<button
							type="button"
							onclick={handleRemovePhoto}
							class="flex h-7 w-7 items-center justify-center rounded-full bg-rose-600 text-white shadow-sm transition-transform hover:scale-110 active:scale-95 sm:h-8 sm:w-8"
							title="Remove portrait"
							aria-label="Remove portrait"
						>
							<Trash2 class="h-3 w-3 sm:h-3.5 sm:w-3.5" />
						</button>
					{/if}
				</div>
			</div>
		</div>
	</div>

	{#if isLoading}
		<div class="flex flex-col items-center justify-center py-20 text-center">
			<Loader2 class="h-8 w-8 animate-spin text-neutral-900" />
			<p class="mt-4 font-serif text-sm text-neutral-600">Retrieving identity details...</p>
		</div>
	{:else if profile}
		<!-- Main Editorial Identity Section -->
		<div class="mb-10 sm:mb-14">
			<div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between sm:gap-6">
				<div class="max-w-3xl">
					{#if hasRealName}
						<!-- Person's actual profile name as primary identity heading -->
						<h1
							class="font-serif text-3xl leading-[1.05] font-normal tracking-tight break-words text-neutral-950 sm:text-5xl sm:leading-[0.95] md:text-6xl lg:text-[4.75rem]"
						>
							{computedDisplayName}
						</h1>
					{:else}
						<!-- Restrained heading when profile name has not yet been set -->
						<div>
							<h1
								class="font-serif text-2xl font-normal tracking-tight text-neutral-900 sm:text-3xl md:text-4xl"
							>
								Personal Profile
							</h1>
							<p class="mt-1 text-xs text-neutral-500">
								Add your name to complete your personal resident profile.
							</p>
						</div>
					{/if}

					<!-- Identity Verification & Account Status Badges -->
					<div class="mt-3 flex flex-wrap items-center gap-2 sm:mt-4 sm:gap-2.5">
						<span
							class="inline-flex items-center gap-1.5 rounded-full border border-emerald-200 bg-emerald-50/90 px-2.5 py-0.5 text-[11px] font-medium text-emerald-800 sm:px-3 sm:py-1 sm:text-xs"
						>
							<CheckCircle2 class="h-3 w-3 text-emerald-600 sm:h-3.5 sm:w-3.5" />
							<span>Verified Account Identity</span>
						</span>

						{#if profile.status}
							<span
								class="inline-flex items-center gap-1.5 rounded-full border border-neutral-200 bg-white px-2.5 py-0.5 text-[11px] text-neutral-700 uppercase sm:px-3 sm:py-1 sm:text-xs"
							>
								<span>STATUS: {profile.status}</span>
							</span>
						{/if}

						{#if profile.platform_roles && profile.platform_roles.length > 0}
							{#each profile.platform_roles as role (role)}
								<span
									class="inline-flex items-center gap-1.5 rounded-full border border-neutral-900 bg-neutral-900 px-2.5 py-0.5 text-[11px] font-medium text-white sm:px-3 sm:py-1 sm:text-xs"
								>
									<ShieldCheck class="h-3 w-3 text-neutral-300 sm:h-3.5 sm:w-3.5" />
									<span>{role}</span>
								</span>
							{/each}
						{/if}
					</div>
				</div>

				<!-- Edit Profile Action Button -->
				{#if !isEditing}
					<button
						type="button"
						onclick={startEditing}
						class="inline-flex items-center gap-2 self-start rounded-full border border-neutral-300 bg-white px-3.5 py-1.5 text-xs font-medium text-neutral-900 shadow-2xs transition-all hover:border-neutral-950 hover:bg-neutral-950 hover:text-white focus:ring-2 focus:ring-neutral-950 focus:outline-hidden sm:px-4 sm:py-2"
						aria-label="Edit personal profile"
					>
						<Edit3 class="h-3.5 w-3.5" />
						<span>Edit Profile</span>
					</button>
				{:else}
					<div class="flex items-center gap-2 self-start">
						<button
							type="button"
							onclick={cancelEditing}
							disabled={isSaving}
							class="inline-flex items-center gap-1.5 rounded-full border border-neutral-200 bg-white px-3 py-1.5 text-xs font-medium text-neutral-700 transition-colors hover:bg-neutral-100 disabled:opacity-50 sm:px-3.5"
						>
							<X class="h-3.5 w-3.5" />
							<span>Cancel</span>
						</button>
						<button
							type="button"
							onclick={handleSave}
							disabled={isSaving}
							class="inline-flex items-center gap-1.5 rounded-full bg-neutral-950 px-3.5 py-1.5 text-xs font-medium text-white shadow-xs transition-transform hover:scale-102 disabled:opacity-50 sm:px-4"
						>
							{#if isSaving}
								<Loader2 class="h-3.5 w-3.5 animate-spin" />
								<span>Saving...</span>
							{:else}
								<Check class="h-3.5 w-3.5" />
								<span>Save Changes</span>
							{/if}
						</button>
					</div>
				{/if}
			</div>
		</div>

		<!-- Personal Information & Verification Sections (Editorial 2-Column Grid) -->
		<div class="grid grid-cols-1 gap-8 lg:grid-cols-12 lg:gap-12">
			<!-- Left / Primary: Personal Information Fields -->
			<div class="space-y-8 lg:col-span-7">
				<div>
					<h2 class="font-serif text-xl font-medium tracking-tight text-neutral-950 sm:text-2xl">
						Personal Information
					</h2>
					<p class="mt-1 text-xs text-neutral-500">
						Your authoritative name and contact preferences for society governance and records.
					</p>
				</div>

				<div class="space-y-6">
					<!-- First Name -->
					<div class="border-b border-neutral-200/90 pb-4">
						<div class="flex items-center justify-between">
							<label
								for="profile-first-name"
								class="text-[11px] font-medium tracking-wider text-neutral-500 uppercase"
							>
								First Name
							</label>
							{#if isEditing}
								<span class="text-[10px] text-neutral-400">Required</span>
							{/if}
						</div>
						{#if isEditing}
							<input
								id="profile-first-name"
								type="text"
								bind:value={firstName}
								disabled={isSaving}
								placeholder="Enter first name"
								class="mt-2 w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2 text-sm text-neutral-900 transition-colors focus:border-neutral-950 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden"
								class:border-rose-400={fieldErrors.firstName}
							/>
							{#if fieldErrors.firstName}
								<p class="mt-1 text-xs text-rose-600">{fieldErrors.firstName}</p>
							{/if}
						{:else}
							<p class="mt-1.5 text-base font-medium text-neutral-900">
								{profile.first_name || '—'}
							</p>
						{/if}
					</div>

					<!-- Last Name -->
					<div class="border-b border-neutral-200/90 pb-4">
						<div class="flex items-center justify-between">
							<label
								for="profile-last-name"
								class="text-[11px] font-medium tracking-wider text-neutral-500 uppercase"
							>
								Last Name
							</label>
							{#if isEditing}
								<span class="text-[10px] text-neutral-400">Optional</span>
							{/if}
						</div>
						{#if isEditing}
							<input
								id="profile-last-name"
								type="text"
								bind:value={lastName}
								disabled={isSaving}
								placeholder="Enter last name"
								class="mt-2 w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2 text-sm text-neutral-900 transition-colors focus:border-neutral-950 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden"
							/>
						{:else}
							<p class="mt-1.5 text-base font-medium text-neutral-900">
								{profile.last_name || '—'}
							</p>
						{/if}
					</div>

					<!-- Email Address -->
					<div class="border-b border-neutral-200/90 pb-4">
						<div class="flex items-center justify-between">
							<label
								for="profile-email"
								class="text-[11px] font-medium tracking-wider text-neutral-500 uppercase"
							>
								Email Address
							</label>
							{#if isEditing}
								<span class="text-[10px] text-neutral-400">Notifications & Summaries</span>
							{/if}
						</div>
						{#if isEditing}
							<div class="relative mt-2">
								<input
									id="profile-email"
									type="email"
									bind:value={email}
									disabled={isSaving}
									placeholder="e.g. name@domain.com"
									class="w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2 text-sm text-neutral-900 transition-colors focus:border-neutral-950 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden"
									class:border-rose-400={fieldErrors.email}
								/>
							</div>
							{#if fieldErrors.email}
								<p class="mt-1 text-xs text-rose-600">{fieldErrors.email}</p>
							{/if}
						{:else}
							<div class="mt-1.5 flex items-center gap-2">
								<Mail class="h-4 w-4 text-neutral-400" />
								<p class="text-base text-neutral-900">
									{profile.email || 'Not configured'}
								</p>
							</div>
						{/if}
					</div>

					<!-- Verified Phone Number (Read-Only Authentication Anchor without "Immutable Identity" label) -->
					<div class="border-b border-neutral-200/90 pb-4">
						<div class="flex items-center justify-between">
							<span class="text-[11px] font-medium tracking-wider text-neutral-500 uppercase">
								Verified Phone Number
							</span>
							<span
								class="inline-flex items-center gap-1 rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] font-medium text-neutral-600"
							>
								<Lock class="h-2.5 w-2.5 text-neutral-500" />
								<span>Verified Account</span>
							</span>
						</div>
						<div class="mt-2 flex items-center gap-2">
							<Phone class="h-4 w-4 text-emerald-600" />
							<span class="font-mono text-base font-semibold text-neutral-900">
								{profile.phone ? formatFullPhone(profile.phone) : '—'}
							</span>
						</div>
						<p class="mt-2 text-xs leading-relaxed text-neutral-500">
							Your mobile number is the primary cryptographic identity anchor for OTP sign-in. To
							transfer account ownership, contact society administration.
						</p>
					</div>
				</div>
			</div>

			<!-- Right / Secondary: Society Memberships & Account Context -->
			<div class="space-y-8 lg:col-span-5">
				<div>
					<h2 class="font-serif text-xl font-medium tracking-tight text-neutral-950 sm:text-2xl">
						Society Context
					</h2>
					<p class="mt-1 text-xs text-neutral-500">
						Cooperative housing societies linked to your verified identity.
					</p>
				</div>

				<div class="space-y-4">
					{#if profile.society_assignments && profile.society_assignments.length > 0}
						{#each profile.society_assignments as assignment (assignment.society_id)}
							<div
								class="flex flex-col gap-3 rounded-2xl border border-neutral-200/80 bg-white/70 p-4 shadow-2xs backdrop-blur-xs transition-colors hover:border-neutral-300 hover:bg-white sm:flex-row sm:items-center sm:justify-between"
							>
								<div class="flex items-start gap-3">
									<div
										class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-neutral-800"
									>
										<Building2 class="h-4 w-4" />
									</div>
									<div>
										<h3 class="text-sm font-semibold text-neutral-900">
											{assignment.society_name}
										</h3>
										<p class="font-serif text-xs text-neutral-500 capitalize">
											Assigned Role: {assignment.role.toLowerCase().replace('_', ' ')}
										</p>
									</div>
								</div>

								<a
									href={`/app/societies/${assignment.society_id}`}
									class="self-start rounded-lg border border-neutral-200 px-2.5 py-1 text-xs font-medium text-neutral-700 transition-colors hover:border-neutral-900 hover:text-neutral-950 sm:self-auto"
								>
									View Society
								</a>
							</div>
						{/each}
					{:else}
						<div
							class="rounded-2xl border border-dashed border-neutral-300 p-6 text-center text-xs text-neutral-500"
						>
							No active society tenancy assigned. Contact your building administrator to link your
							residency.
						</div>
					{/if}
				</div>

				<!-- Security & Session Information Card -->
				<div class="rounded-2xl border border-neutral-200/80 bg-neutral-50/60 p-5">
					<div class="flex items-center gap-2">
						<ShieldCheck class="h-4 w-4 text-neutral-700" />
						<h3 class="text-xs font-medium tracking-wide text-neutral-900 uppercase">
							Security & Session Health
						</h3>
					</div>
					<p class="mt-2 text-xs leading-relaxed text-neutral-600">
						Sessions are protected via strict device limit invariants. Signing in on a new device
						automatically rotates and retires the oldest active token.
					</p>
					<div class="mt-4 flex items-center justify-between border-t border-neutral-200/60 pt-3">
						<span class="text-[11px] text-neutral-500">Account status</span>
						<span class="text-xs font-medium text-emerald-700">Active & Verified</span>
					</div>
				</div>

				<!-- Active Sessions Card (Strictly Active Devices Only, Max 2) -->
				<div class="rounded-2xl border border-neutral-200/80 bg-white p-5 shadow-xs">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2">
							<ShieldCheck class="h-4 w-4 text-neutral-800" />
							<h3 class="text-xs font-semibold tracking-wide text-neutral-900 uppercase">
								Active Sessions
							</h3>
						</div>
						<span
							class="rounded-full bg-neutral-100 px-2.5 py-0.5 text-[10px] font-medium text-neutral-600"
						>
							{activeSessions.length} of 2 Devices Active
						</span>
					</div>

					<p class="mt-1.5 text-xs text-neutral-500">
						Your account is restricted to a maximum of 2 active devices. New logins automatically
						retire the oldest session (FIFO).
					</p>

					<!-- Active Sessions List -->
					<div class="mt-4 space-y-2.5">
						{#if isLoadingSessions && activeSessions.length === 0}
							<div class="flex items-center justify-center py-6 text-xs text-neutral-500">
								<Loader2 class="mr-2 h-3.5 w-3.5 animate-spin" />
								<span>Retrieving active sessions...</span>
							</div>
						{:else if activeSessions.length === 0}
							<!-- Fallback row for currently active browser session -->
							<div
								class="flex flex-col gap-2 rounded-xl border border-neutral-200/80 bg-neutral-50/50 p-3.5 sm:flex-row sm:items-center sm:justify-between"
							>
								<div class="flex items-start gap-3">
									<div
										class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-neutral-900 text-white"
									>
										<Laptop class="h-4 w-4" />
									</div>
									<div>
										<div class="flex items-center gap-2">
											<p class="text-xs font-semibold text-neutral-950">Current Device</p>
											<span
												class="inline-flex items-center rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[10px] font-medium text-emerald-800"
											>
												This Device
											</span>
										</div>
										<p class="mt-0.5 text-[11px] text-neutral-500">Active now</p>
									</div>
								</div>
								<Button
									variant="outline"
									size="sm"
									onclick={() => (isSignOutOpen = true)}
									class="self-start border-neutral-300 bg-white text-xs text-neutral-800 hover:bg-neutral-100 sm:self-auto"
								>
									<LogOut class="mr-1.5 h-3.5 w-3.5 text-neutral-600" />
									<span>Sign Out</span>
								</Button>
							</div>
						{:else}
							{#each activeSessions as session (session.id)}
								{@const DeviceIcon = getDeviceIcon(session)}
								<div
									class="flex flex-col gap-3 rounded-xl border p-3.5 sm:flex-row sm:items-center sm:justify-between {session.is_current
										? 'border-neutral-200/90 bg-neutral-50/50'
										: 'border-neutral-200 bg-white'}"
								>
									<div class="flex items-start gap-3">
										<div
											class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg {session.is_current
												? 'bg-neutral-900 text-white'
												: 'bg-neutral-100 text-neutral-800'}"
										>
											<DeviceIcon class="h-4 w-4" />
										</div>
										<div class="min-w-0">
											<div class="flex flex-wrap items-center gap-2">
												<span class="text-xs font-semibold text-neutral-950">
													{session.device_label || 'Active Device'}
												</span>
												{#if session.is_current}
													<span
														class="inline-flex items-center rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[10px] font-medium text-emerald-800"
													>
														This Device
													</span>
												{:else}
													<span
														class="inline-flex items-center rounded-full border border-neutral-200 bg-neutral-100 px-2 py-0.5 text-[10px] font-medium text-neutral-600"
													>
														Other Active Device
													</span>
												{/if}
											</div>
											<div class="mt-0.5 flex items-center gap-2 text-[11px] text-neutral-500">
												<span
													>{formatRelativeActive(session.last_active_at, session.is_current)}</span
												>
												{#if session.os && session.os !== 'Unknown'}
													<span class="text-neutral-300">·</span>
													<span>{session.os}</span>
												{/if}
											</div>
										</div>
									</div>

									{#if session.is_current}
										<Button
											variant="outline"
											size="sm"
											onclick={() => (isSignOutOpen = true)}
											class="self-start border-neutral-300 bg-white text-xs text-neutral-800 hover:bg-neutral-100 sm:self-auto"
										>
											<LogOut class="mr-1.5 h-3.5 w-3.5 text-neutral-600" />
											<span>Sign Out</span>
										</Button>
									{:else}
										<Button
											variant="outline"
											size="sm"
											disabled={revokingSessionId === session.id}
											onclick={() => handleRevokeSession(session)}
											class="self-start border-rose-200 bg-white text-xs text-rose-700 hover:bg-rose-50 hover:text-rose-900 disabled:opacity-50 sm:self-auto"
										>
											{#if revokingSessionId === session.id}
												<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
												<span>Revoking...</span>
											{:else}
												<LogOut class="mr-1.5 h-3.5 w-3.5 text-rose-600" />
												<span>Revoke Access</span>
											{/if}
										</Button>
									{/if}
								</div>
							{/each}
						{/if}
					</div>

					<!-- Global Session Termination -->
					<div
						class="mt-5 flex flex-col gap-2 border-t border-neutral-200/60 pt-4 sm:flex-row sm:items-center sm:justify-between"
					>
						<div>
							<p class="text-xs font-medium text-neutral-900">Security Emergency?</p>
							<p class="text-[11px] text-neutral-500">
								Revoke all active sessions across all devices immediately.
							</p>
						</div>
						<Button
							variant="outline"
							size="sm"
							onclick={() => (isLogoutAllOpen = true)}
							class="self-start border-rose-300 bg-white text-xs text-rose-700 hover:bg-rose-50 hover:text-rose-900 sm:self-auto"
						>
							<Smartphone class="mr-1.5 h-3.5 w-3.5 text-rose-600" />
							<span>Log Out All Devices</span>
						</Button>
					</div>
				</div>

				<!-- Progressive Web Application (PWA) Shell Card -->
				<div class="rounded-2xl border border-neutral-200/80 bg-white p-6 shadow-xs">
					<div class="flex items-center justify-between">
						<div>
							<h3 class="text-sm font-semibold tracking-tight text-neutral-950">
								Progressive Web App (PWA)
							</h3>
							<p class="text-xs text-neutral-500">
								Offline pass resilience, standalone display, and home-screen access.
							</p>
						</div>
						{#if pwa.isInstalled}
							<span
								class="inline-flex items-center gap-1.5 rounded-full bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-800"
							>
								<CheckCircle2 class="h-3.5 w-3.5 text-emerald-600" />
								<span>Installed</span>
							</span>
						{:else}
							<span
								class="inline-flex items-center gap-1.5 rounded-full bg-neutral-100 px-2.5 py-0.5 text-xs font-medium text-neutral-700"
							>
								<Globe class="h-3.5 w-3.5 text-neutral-500" />
								<span>Web Browser</span>
							</span>
						{/if}
					</div>

					<div class="mt-4 flex flex-col gap-3 rounded-xl border border-neutral-100 bg-[#fbfbfa] p-4 text-xs text-neutral-600 sm:flex-row sm:items-center sm:justify-between">
						<div class="space-y-1">
							<div class="font-medium text-neutral-900">
								{#if pwa.isInstalled}
									Standalone Application Active
								{:else if pwa.canInstall}
									Ready for One-Tap Installation
								{:else}
									Offline Shell & Cache Active
								{/if}
							</div>
							<p class="text-[11px] text-neutral-500">
								{#if pwa.isInstalled}
									Banesons Society is running with standalone windowing and offline gate pass resilience.
								{:else if pwa.canInstall}
									Install Banesons Society to your dock or home screen for faster entry and instant gate passes.
								{:else}
									Static assets and offline gate fallback are cached on this device via service worker.
								{/if}
							</p>
						</div>

						{#if pwa.canInstall}
							<Button
								variant="primary"
								size="sm"
								onclick={async () => {
									const accepted = await pwa.promptInstall();
									if (accepted) {
										toast.success('Society App installed successfully');
									}
								}}
								class="self-start sm:self-auto"
							>
								<Smartphone class="mr-1.5 h-3.5 w-3.5" />
								<span>Install App</span>
							</Button>
						{/if}
					</div>
				</div>
			</div>
		</div>

		<!-- Authenticated Visitor Passes & Visits -->
		<div class="mt-12 border-t border-[#e2e2de] pt-8">
			<MyVisitorHistory />
		</div>

		<!-- Activity Timeline Section -->
		<div class="mt-12 border-t border-[#e2e2de] pt-8">
			<ActivityTimeline />
		</div>
	{/if}
</div>

<SignOutDialog isOpen={isSignOutOpen} onClose={() => (isSignOutOpen = false)} />
<LogoutAllDialog isOpen={isLogoutAllOpen} onClose={() => (isLogoutAllOpen = false)} />
