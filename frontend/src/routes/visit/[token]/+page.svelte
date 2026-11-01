<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { societyApi, type PublicQRContext, type PublicVisitorPass } from '$lib/api/society';
	import { auth } from '$lib/auth.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import {
		ShieldCheck,
		Camera,
		CheckCircle2,
		AlertCircle,
		Loader2,
		RefreshCw,
		User,
		ArrowRight,
		Clock,
		XCircle,
		MapPin
	} from '@lucide/svelte';

	let token = $derived(page.params.token ?? '');

	// Registration context
	let qrContext = $state<PublicQRContext | null>(null);
	let isLoadingContext = $state(true);
	let contextError = $state<string | null>(null);

	// Form fields
	let fullName = $state('');
	let phone = $state('');
	let purpose = $state('GUEST');
	let purposeOther = $state('');
	let comingFrom = $state('');
	let photoDataUrl = $state<string | null>(null);

	// OTP state
	let otpCode = $state('');
	let isOtpSent = $state(false);
	let isOtpSending = $state(false);
	let isOtpVerifying = $state(false);
	let verificationToken = $state<string | null>(null);
	let isPhoneVerified = $state(false);
	let otpError = $state<string | null>(null);
	let cooldownSeconds = $state(0);
	let cooldownTimer: ReturnType<typeof setInterval> | null = null;
	let devTestOtp = $state<string | null>(null);

	// Submission state
	let isSubmitting = $state(false);
	let submitError = $state<string | null>(null);

	// Issued digital pass
	let pass = $state<PublicVisitorPass | null>(null);
	let pollInterval: ReturnType<typeof setInterval> | null = null;

	// Purpose options
	const purposeOptions = [
		{ value: 'GUEST', label: 'Meeting Resident / Personal Guest' },
		{ value: 'DELIVERY', label: 'Delivery / Courier / Parcel' },
		{ value: 'CAB', label: 'Cab / Ride Pickup / Drop' },
		{ value: 'SERVICE', label: 'Maintenance / Service / Repair' },
		{ value: 'DOMESTIC_HELP', label: 'Domestic Staff / Help' },
		{ value: 'OTHER', label: 'Other (Specify Below)' }
	];

	async function loadContext() {
		if (!token) return;
		isLoadingContext = true;
		contextError = null;

		try {
			const res = await societyApi.resolvePublicVisitQR(token);
			qrContext = res;

			// Prefill for returning authenticated users
			if (auth.user) {
				if (auth.user.display_name && !fullName) {
					fullName = auth.user.display_name;
				}
				if (auth.user.phone && !phone) {
					phone = auth.user.phone.replace(/^91/, '');
				}
			}
		} catch (err: unknown) {
			contextError =
				err instanceof Error
					? err.message
					: 'This visitor registration point is invalid, inactive, or has been revoked.';
		} finally {
			isLoadingContext = false;
		}
	}

	onMount(() => {
		loadContext();
	});

	onDestroy(() => {
		if (cooldownTimer) clearInterval(cooldownTimer);
		if (pollInterval) clearInterval(pollInterval);
	});

	function startCooldown(sec: number) {
		cooldownSeconds = sec;
		if (cooldownTimer) clearInterval(cooldownTimer);
		cooldownTimer = setInterval(() => {
			if (cooldownSeconds > 0) {
				cooldownSeconds--;
			} else if (cooldownTimer) {
				clearInterval(cooldownTimer);
				cooldownTimer = null;
			}
		}, 1000);
	}

	async function handleSendOTP() {
		if (!phone.trim()) {
			otpError = 'Please enter a 10-digit mobile number';
			return;
		}
		otpError = null;
		isOtpSending = true;

		try {
			const res = await societyApi.sendVisitorOTP(phone, token);
			isOtpSent = true;
			startCooldown(res.resend_after_seconds || 30);
			if (res.otp) {
				devTestOtp = res.otp;
				otpCode = res.otp; // convenient for dev testing
			}
		} catch (err: unknown) {
			otpError = err instanceof Error ? err.message : 'Failed to dispatch verification code';
		} finally {
			isOtpSending = false;
		}
	}

	async function handleVerifyOTP() {
		if (!otpCode.trim() || otpCode.length < 4) {
			otpError = 'Please enter the verification code sent to your phone';
			return;
		}
		otpError = null;
		isOtpVerifying = true;

		try {
			const res = await societyApi.verifyVisitorOTP(phone, otpCode, token);
			verificationToken = res.verification_token;
			isPhoneVerified = true;
		} catch (err: unknown) {
			otpError = err instanceof Error ? err.message : 'Invalid verification code';
		} finally {
			isOtpVerifying = false;
		}
	}

	function handlePhotoSelect(event: Event) {
		const target = event.target as HTMLInputElement;
		if (target.files && target.files[0]) {
			const file = target.files[0];
			if (file.size > 5 * 1024 * 1024) {
				alert('Image size exceeds 5MB limit. Please choose a smaller photo.');
				return;
			}
			const reader = new FileReader();
			reader.onload = (e) => {
				photoDataUrl = e.target?.result as string;
			};
			reader.readAsDataURL(file);
		}
	}

	async function handleSubmitRegistration(e: Event) {
		e.preventDefault();
		if (!isPhoneVerified || !verificationToken) {
			submitError = 'Please verify your mobile number with OTP first.';
			return;
		}
		if (!fullName.trim()) {
			submitError = 'Full name is required.';
			return;
		}

		isSubmitting = true;
		submitError = null;

		try {
			const res = await societyApi.registerVisitorByQR({
				qr_token: token,
				full_name: fullName.trim(),
				phone: phone.trim(),
				verification_token: verificationToken,
				purpose,
				purpose_other: purpose === 'OTHER' ? purposeOther.trim() : undefined,
				coming_from: comingFrom.trim() || undefined,
				visitor_photo_url: photoDataUrl || undefined
			});

			pass = res;
			startPassStatusPolling(res.pass_reference);
		} catch (err: unknown) {
			submitError = err instanceof Error ? err.message : 'Registration failed. Please try again.';
		} finally {
			isSubmitting = false;
		}
	}

	function startPassStatusPolling(passRef: string) {
		if (pollInterval) clearInterval(pollInterval);
		pollInterval = setInterval(async () => {
			try {
				const latest = await societyApi.getPublicVisitorPass(passRef);
				pass = latest;
				if ((latest.status === 'CHECKED_IN' || latest.status === 'REJECTED') && pollInterval) {
					clearInterval(pollInterval);
					pollInterval = null;
				}
			} catch {
				// Ignore polling errors
			}
		}, 4000);
	}
</script>

<svelte:head>
	<title
		>{qrContext
			? `${qrContext.society_name} — Visitor Pass`
			: 'Visitor Self-Registration — CHS Neu'}</title
	>
</svelte:head>

<div
	class="min-h-screen bg-[#f3f3f1] pb-16 font-sans text-[#121212] selection:bg-neutral-900 selection:text-white"
>
	<!-- Top Bar -->
	<header class="sticky top-0 z-30 border-b border-[#e2e2de] bg-white/80 backdrop-blur-md">
		<div class="mx-auto flex h-14 max-w-xl items-center justify-between px-4">
			<div class="flex items-center gap-2">
				<div
					class="flex h-6 w-6 items-center justify-center rounded bg-[#121212] text-xs font-bold tracking-wider text-white"
				>
					CHS
				</div>
				<span class="text-xs font-semibold tracking-wide text-neutral-800 uppercase">
					Gate Clearance System
				</span>
			</div>

			<Badge
				variant="outline"
				class="border-neutral-300 font-mono text-[10px] tracking-wider uppercase"
			>
				Public Pass
			</Badge>
		</div>
	</header>

	<main class="mx-auto max-w-xl px-4 pt-6">
		{#if isLoadingContext}
			<div
				class="animate-pulse space-y-4 rounded-2xl border border-[#e2e2de] bg-white p-8 text-center shadow-sm"
			>
				<Loader2 class="mx-auto h-8 w-8 animate-spin text-neutral-400" />
				<p class="text-sm text-neutral-600">Verifying society entrance registration point...</p>
			</div>
		{:else if contextError}
			<div
				class="space-y-4 rounded-2xl border border-rose-200 bg-rose-50/60 p-6 text-center shadow-sm"
			>
				<div
					class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-rose-100 text-rose-600"
				>
					<XCircle class="h-6 w-6" />
				</div>
				<div class="space-y-1">
					<h2 class="text-base font-semibold text-rose-900">Registration Unavailable</h2>
					<p class="mx-auto max-w-sm text-xs text-rose-700">{contextError}</p>
				</div>
				<Button variant="outline" size="sm" onclick={loadContext} class="border-rose-200 text-xs">
					<RefreshCw class="mr-1.5 h-3.5 w-3.5" />
					Retry Scan
				</Button>
			</div>
		{:else if pass}
			<!-- DIGITAL VISITOR PASS CONFIRMATION -->
			<div class="space-y-6">
				<!-- Status Banner -->
				{#if pass.status === 'CHECKED_IN'}
					<div
						class="flex items-start gap-3.5 rounded-2xl border border-emerald-200 bg-emerald-50 p-5 text-emerald-950 shadow-sm"
					>
						<CheckCircle2 class="mt-0.5 h-6 w-6 shrink-0 text-emerald-600" />
						<div>
							<h3 class="text-sm font-semibold">Entry Approved & Checked In</h3>
							<p class="mt-0.5 text-xs text-emerald-800">
								Security clearance granted at {pass.gate_name || 'main gate'}. You may proceed into
								the society.
							</p>
						</div>
					</div>
				{:else if pass.status === 'REJECTED'}
					<div
						class="flex items-start gap-3.5 rounded-2xl border border-rose-200 bg-rose-50 p-5 text-rose-950 shadow-sm"
					>
						<XCircle class="mt-0.5 h-6 w-6 shrink-0 text-rose-600" />
						<div>
							<h3 class="text-sm font-semibold">Entry Denied by Security</h3>
							<p class="mt-0.5 text-xs text-rose-800">
								Your visit request was declined at the gate. Please check with resident or society
								reception.
							</p>
						</div>
					</div>
				{:else}
					<div
						class="flex animate-pulse items-start gap-3.5 rounded-2xl border border-amber-200 bg-amber-50/80 p-5 text-amber-950 shadow-sm"
					>
						<Clock class="mt-0.5 h-6 w-6 shrink-0 text-amber-600" />
						<div>
							<h3 class="text-sm font-semibold">Awaiting Security Verification</h3>
							<p class="mt-0.5 text-xs text-amber-800">
								Registration submitted! Please show this pass to security for gatekeeper check-in.
							</p>
						</div>
					</div>
				{/if}

				<!-- Digital Pass Card -->
				<div class="overflow-hidden rounded-2xl border border-[#e2e2de] bg-white shadow-sm">
					<!-- Pass Header -->
					<div class="flex items-center justify-between bg-neutral-900 p-5 text-white">
						<div>
							<p class="font-mono text-[10px] tracking-widest text-neutral-400 uppercase">
								Visitor Pass
							</p>
							<h2 class="mt-0.5 text-xl font-bold tracking-tight text-white">
								{pass.society_name}
							</h2>
							{#if pass.gate_name}
								<p class="mt-0.5 flex items-center gap-1 text-xs text-neutral-300">
									<MapPin class="h-3 w-3 text-neutral-400" />
									{pass.gate_name}
								</p>
							{/if}
						</div>

						<div class="text-right">
							<p class="font-mono text-[10px] tracking-widest text-neutral-400 uppercase">
								Pass Code
							</p>
							<div class="font-mono text-lg font-bold tracking-wider text-emerald-400">
								{pass.pass_reference}
							</div>
						</div>
					</div>

					<!-- Pass Body -->
					<div class="space-y-6 p-6">
						<!-- Photo & Primary Details -->
						<div class="flex items-center gap-4 border-b border-[#e2e2de] pb-6">
							{#if pass.visitor_photo_url}
								<img
									src={pass.visitor_photo_url}
									alt={pass.visitor_name}
									class="h-16 w-16 shrink-0 rounded-xl border border-[#e2e2de] bg-neutral-100 object-cover shadow-sm"
								/>
							{:else}
								<div
									class="flex h-16 w-16 shrink-0 items-center justify-center rounded-xl border border-[#e2e2de] bg-neutral-100 text-neutral-400"
								>
									<User class="h-8 w-8" />
								</div>
							{/if}

							<div class="space-y-1">
								<h3 class="text-base font-semibold text-neutral-900">{pass.visitor_name}</h3>
								<p class="font-mono text-xs text-neutral-500">{pass.visitor_phone_masked}</p>
								<Badge variant="outline" class="border-neutral-300 font-mono text-[10px]">
									{pass.purpose.replace(/_/g, ' ')}
								</Badge>
							</div>
						</div>

						<!-- Grid Info -->
						<div class="grid grid-cols-2 gap-4 text-xs">
							<div>
								<span class="block text-[10px] tracking-wider text-neutral-400 uppercase"
									>Status</span
								>
								<span class="font-medium text-neutral-800">
									{pass.status.replace(/_/g, ' ')}
								</span>
							</div>

							<div>
								<span class="block text-[10px] tracking-wider text-neutral-400 uppercase"
									>Registered At</span
								>
								<span class="font-medium text-neutral-800">
									{new Date(pass.registered_at).toLocaleTimeString([], {
										hour: '2-digit',
										minute: '2-digit'
									})}
								</span>
							</div>

							{#if pass.coming_from}
								<div class="col-span-2">
									<span class="block text-[10px] tracking-wider text-neutral-400 uppercase"
										>Origin</span
									>
									<span class="font-medium text-neutral-800">{pass.coming_from}</span>
								</div>
							{/if}

							{#if pass.check_in_at}
								<div>
									<span class="block text-[10px] tracking-wider text-neutral-400 uppercase"
										>Check In</span
									>
									<span class="font-medium text-emerald-700">
										{new Date(pass.check_in_at).toLocaleTimeString([], {
											hour: '2-digit',
											minute: '2-digit'
										})}
									</span>
								</div>
							{/if}
						</div>
					</div>

					<!-- Pass Footer -->
					<div class="border-t border-[#e2e2de] bg-neutral-50 p-4 text-center">
						<p class="text-[11px] text-neutral-500">
							Keep this page open until verified at the gate. Status updates automatically.
						</p>
					</div>
				</div>
			</div>
		{:else if qrContext}
			<!-- SELF-REGISTRATION FORM -->
			<div class="space-y-6">
				<!-- Verified Entrance Header Banner -->
				<div class="space-y-2 rounded-2xl border border-[#e2e2de] bg-white p-5 shadow-sm">
					<div class="flex items-center justify-between">
						<span class="inline-flex items-center gap-1.5 text-xs font-semibold text-emerald-700">
							<ShieldCheck class="h-4 w-4 text-emerald-600" />
							Verified Entrance QR
						</span>
						<span class="font-mono text-[10px] tracking-wider text-neutral-400 uppercase">
							{qrContext.society_code}
						</span>
					</div>

					<h1 class="text-lg font-bold tracking-tight text-neutral-900">
						{qrContext.society_name}
					</h1>

					{#if qrContext.gate_name}
						<p class="flex items-center gap-1 text-xs text-neutral-600">
							<MapPin class="h-3.5 w-3.5 text-neutral-400" />
							Entry Gate: <span class="font-medium text-neutral-800">{qrContext.gate_name}</span>
						</p>
					{/if}
				</div>

				<!-- Form Card -->
				<form
					onsubmit={handleSubmitRegistration}
					class="space-y-6 rounded-2xl border border-[#e2e2de] bg-white p-6 shadow-sm"
				>
					<div>
						<h2 class="text-sm font-semibold tracking-tight text-neutral-900">
							Visitor Self-Registration
						</h2>
						<p class="mt-0.5 text-xs text-neutral-500">
							Please provide your details for security admission.
						</p>
					</div>

					{#if submitError}
						<div
							class="flex items-center gap-2 rounded-xl border border-rose-200 bg-rose-50 p-3.5 text-xs text-rose-800"
						>
							<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
							<span>{submitError}</span>
						</div>
					{/if}

					<!-- Section 1: Mobile & OTP Verification -->
					<div class="space-y-3 border-b border-[#e2e2de] pb-6">
						<label class="block text-xs font-semibold text-neutral-700">
							Mobile Number <span class="text-rose-500">*</span>
						</label>

						<div class="flex gap-2">
							<div class="relative flex-1">
								<span class="absolute top-2.5 left-3 font-mono text-xs text-neutral-400">+91</span>
								<input
									type="tel"
									maxlength="10"
									bind:value={phone}
									disabled={isPhoneVerified}
									placeholder="9820012345"
									class="w-full rounded-xl border border-[#e2e2de] py-2 pr-3 pl-11 font-mono text-xs focus:border-neutral-900 focus:outline-none disabled:bg-neutral-100 disabled:text-neutral-500"
								/>
							</div>

							{#if !isPhoneVerified}
								<Button
									type="button"
									variant="outline"
									size="sm"
									onclick={handleSendOTP}
									disabled={isOtpSending || cooldownSeconds > 0 || !phone.trim()}
									class="text-xs font-medium whitespace-nowrap"
								>
									{#if isOtpSending}
										<Loader2 class="mr-1 h-3 w-3 animate-spin" />
									{/if}
									{cooldownSeconds > 0 ? `${cooldownSeconds}s` : isOtpSent ? 'Resend' : 'Send OTP'}
								</Button>
							{:else}
								<Badge
									variant="outline"
									class="border-emerald-200 bg-emerald-50 px-2.5 py-1 text-xs text-emerald-700"
								>
									<CheckCircle2 class="mr-1 h-3.5 w-3.5" />
									Verified
								</Badge>
							{/if}
						</div>

						<!-- OTP Input Form -->
						{#if isOtpSent && !isPhoneVerified}
							<div class="mt-3 space-y-3 rounded-xl border border-[#e2e2de] bg-neutral-50 p-3.5">
								<div class="flex items-center justify-between">
									<span class="text-[11px] font-medium text-neutral-700"
										>Enter 6-Digit Verification Code</span
									>
									{#if devTestOtp}
										<span class="font-mono text-[10px] text-neutral-500">Dev OTP: {devTestOtp}</span
										>
									{/if}
								</div>

								<div class="flex gap-2">
									<input
										type="text"
										maxlength="6"
										bind:value={otpCode}
										placeholder="123456"
										class="flex-1 rounded-xl border border-[#e2e2de] bg-white px-3 py-2 text-center font-mono text-sm tracking-widest focus:border-neutral-900 focus:outline-none"
									/>
									<Button
										type="button"
										size="sm"
										onclick={handleVerifyOTP}
										disabled={isOtpVerifying || !otpCode.trim()}
										class="bg-[#121212] text-xs text-white"
									>
										{#if isOtpVerifying}
											<Loader2 class="mr-1 h-3 w-3 animate-spin" />
										{/if}
										Verify
									</Button>
								</div>

								{#if otpError}
									<p class="text-[11px] text-rose-600">{otpError}</p>
								{/if}
							</div>
						{/if}
					</div>

					<!-- Section 2: Visitor Details -->
					<div class="space-y-4 border-b border-[#e2e2de] pb-6">
						<div>
							<label class="mb-1 block text-xs font-semibold text-neutral-700">
								Full Name <span class="text-rose-500">*</span>
							</label>
							<input
								type="text"
								bind:value={fullName}
								placeholder="e.g. Rajesh Sharma"
								class="w-full rounded-xl border border-[#e2e2de] px-3 py-2 text-xs focus:border-neutral-900 focus:outline-none"
								required
							/>
						</div>

						<div>
							<label class="mb-1 block text-xs font-semibold text-neutral-700">
								Purpose of Visit <span class="text-rose-500">*</span>
							</label>
							<select
								bind:value={purpose}
								class="w-full rounded-xl border border-[#e2e2de] bg-white px-3 py-2 text-xs focus:border-neutral-900 focus:outline-none"
							>
								{#each purposeOptions as opt (opt.value)}
									<option value={opt.value}>{opt.label}</option>
								{/each}
							</select>
						</div>

						{#if purpose === 'OTHER'}
							<div>
								<label class="mb-1 block text-xs font-semibold text-neutral-700">
									Specify Purpose
								</label>
								<input
									type="text"
									bind:value={purposeOther}
									placeholder="Please describe reason for visit"
									class="w-full rounded-xl border border-[#e2e2de] px-3 py-2 text-xs focus:border-neutral-900 focus:outline-none"
								/>
							</div>
						{/if}

						<div>
							<label class="mb-1 block text-xs font-semibold text-neutral-700">
								Coming From (Origin)
							</label>
							<input
								type="text"
								bind:value={comingFrom}
								placeholder="e.g. Andheri, Delivery Company"
								class="w-full rounded-xl border border-[#e2e2de] px-3 py-2 text-xs focus:border-neutral-900 focus:outline-none"
							/>
						</div>
					</div>

					<!-- Section 3: Photograph -->
					<div class="space-y-3 border-b border-[#e2e2de] pb-6">
						<div class="flex items-center justify-between">
							<label class="block text-xs font-semibold text-neutral-700">
								Visitor Photograph
							</label>
							<span class="text-[10px] text-neutral-500">Human verification at gate</span>
						</div>

						{#if photoDataUrl}
							<div class="flex items-center gap-4">
								<img
									src={photoDataUrl}
									alt="Visitor Preview"
									class="h-20 w-20 rounded-xl border border-[#e2e2de] bg-neutral-100 object-cover shadow-sm"
								/>
								<Button
									type="button"
									variant="outline"
									size="sm"
									onclick={() => (photoDataUrl = null)}
									class="text-xs"
								>
									Retake Photograph
								</Button>
							</div>
						{:else}
							<label
								class="flex cursor-pointer flex-col items-center justify-center rounded-2xl border-2 border-dashed border-[#e2e2de] bg-neutral-50 p-6 transition hover:border-neutral-400"
							>
								<Camera class="mb-2 h-6 w-6 text-neutral-400" />
								<span class="text-xs font-semibold text-neutral-800">Tap to Capture Photograph</span
								>
								<span class="mt-0.5 text-[10px] text-neutral-500"
									>Use camera or upload clear face portrait</span
								>
								<input
									type="file"
									accept="image/*"
									capture="user"
									onchange={handlePhotoSelect}
									class="hidden"
								/>
							</label>
						{/if}
					</div>

					<!-- Submit Action -->
					<Button
						type="submit"
						disabled={!isPhoneVerified || !fullName.trim() || isSubmitting}
						class="w-full rounded-xl bg-[#121212] py-2.5 text-xs font-semibold text-white shadow-sm hover:bg-neutral-800"
					>
						{#if isSubmitting}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
							Submitting Registration...
						{:else}
							<span>Generate Visitor Pass</span>
							<ArrowRight class="ml-2 h-4 w-4" />
						{/if}
					</Button>
				</form>
			</div>
		{/if}
	</main>
</div>
