<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Logo from '$lib/components/ui/logo.svelte';
	import AuthVisualCard from '$lib/components/auth/auth-visual-card.svelte';
	import { normalizePhoneInput, isValidIndianPhone, formatFullPhone } from '$lib/phone';
	import {
		ArrowRight,
		RotateCw,
		AlertCircle,
		CheckCircle2,
		Loader2,
		ArrowLeft,
		Sparkles
	} from '@lucide/svelte';

	import { resolvePostLoginRoute, appRoutes } from '$lib/routing';
	import type { UserSocietyAssignment } from '$lib/api/society';
	import SocietySelectorDialog from '$lib/components/nav/society-selector-dialog.svelte';

	let step = $state<'phone' | 'otp'>('phone');
	let phoneInput = $state('');
	let otpInput = $state('');
	let devOtp = $state<string | null>(null);
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);
	let cooldown = $state(0);
	let cooldownTimer: ReturnType<typeof setInterval> | null = null;
	let isSelectorOpen = $state(false);
	let multipleSocieties = $state<UserSocietyAssignment[]>([]);

	let returnToParam = $derived(page.url.searchParams.get('returnTo'));

	function startCooldown(seconds: number) {
		cooldown = seconds;
		if (cooldownTimer) clearInterval(cooldownTimer);
		cooldownTimer = setInterval(() => {
			if (cooldown > 0) {
				cooldown -= 1;
			} else if (cooldownTimer) {
				clearInterval(cooldownTimer);
			}
		}, 1000);
	}

	function handlePostLoginRouting() {
		const resolution = resolvePostLoginRoute(auth.user, returnToParam);
		if (resolution.type === 'navigate') {
			goto(resolution.path, { replaceState: true });
		} else if (resolution.type === 'select_society') {
			multipleSocieties = resolution.assignments;
			isSelectorOpen = true;
		} else {
			goto('/unauthorized', { replaceState: true });
		}
	}

	onMount(() => {
		if (auth.isAuthenticated && auth.user) {
			handlePostLoginRouting();
		}
	});

	function handlePhoneChange(e: Event) {
		const target = e.target as HTMLInputElement;
		let digits = target.value.replace(/\D/g, '');
		if (digits.length >= 12 && digits.startsWith('91')) {
			digits = digits.slice(2);
		} else if (digits.length === 11 && digits.startsWith('0')) {
			digits = digits.slice(1);
		}
		digits = digits.slice(0, 10);
		phoneInput = digits;
		target.value = digits;
	}

	function handleOtpChange(e: Event) {
		const target = e.target as HTMLInputElement;
		const digits = target.value.replace(/\D/g, '').slice(0, 6);
		otpInput = digits;
		target.value = digits;
	}

	async function handleSendOTP(e: SubmitEvent) {
		e.preventDefault();
		if (isSubmitting) return;
		errorMessage = null;

		const cleanPhone = normalizePhoneInput(phoneInput);
		if (!cleanPhone) {
			errorMessage = 'Please enter your mobile phone number';
			return;
		}

		if (!isValidIndianPhone(cleanPhone)) {
			errorMessage = 'Please enter a valid 10-digit Indian mobile number';
			return;
		}

		isSubmitting = true;
		try {
			const res = await auth.sendOTP(cleanPhone);
			step = 'otp';
			startCooldown(res.resend_after_seconds);
			if (res.otp) {
				devOtp = res.otp;
				otpInput = res.otp; // Auto-fill in development
			}
		} catch (err: unknown) {
			errorMessage = err instanceof Error ? err.message : 'Failed to dispatch verification code';
		} finally {
			isSubmitting = false;
		}
	}

	async function handleVerifyOTP(e: SubmitEvent) {
		e.preventDefault();
		if (isSubmitting) return;
		errorMessage = null;

		if (!otpInput.trim() || otpInput.length < 6) {
			errorMessage = 'Please enter the 6-digit verification code';
			return;
		}

		isSubmitting = true;
		try {
			const cleanPhone = normalizePhoneInput(phoneInput);
			await auth.verifyOTP(cleanPhone, otpInput.trim());
			if (cooldownTimer) clearInterval(cooldownTimer);
			handlePostLoginRouting();
		} catch (err: unknown) {
			errorMessage = err instanceof Error ? err.message : 'Verification failed';
		} finally {
			isSubmitting = false;
		}
	}

	function handleEditPhone() {
		step = 'phone';
		otpInput = '';
		devOtp = null;
		errorMessage = null;
	}
</script>

<svelte:head>
	<title>Sign in — chs.</title>
	<meta name="description" content="Sign in to CHS Platform using mobile verification" />
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<!-- Fullscreen Edge-to-Edge Split Layout -->
<div class="relative min-h-screen w-full bg-white">
	<div class="grid min-h-screen w-full grid-cols-1 lg:grid-cols-12">
		<!-- Left Column: Primary Interaction -->
		<div
			class="flex min-h-screen flex-col justify-between p-6 sm:p-10 md:p-12 lg:col-span-6 lg:p-14 xl:p-20"
		>
			<!-- Top Navigation Bar -->
			<div>
				<div class="flex items-center justify-between">
					<a
						href="/"
						class="group inline-flex items-center gap-2 rounded-full border border-neutral-300 bg-white px-3.5 py-1.5 font-sans text-xs font-semibold text-neutral-800 shadow-2xs transition-all hover:border-neutral-950 hover:bg-neutral-950 hover:text-white hover:shadow-xs"
					>
						<ArrowLeft
							class="h-3.5 w-3.5 transition-transform duration-200 group-hover:-translate-x-1"
						/>
						<span>back to home</span>
					</a>
					<Logo size="sm" />
				</div>

				<!-- Section Kicker & Massive Grotesque Headline -->
				<div class="mt-8">
					<span class="font-sans text-xs font-medium tracking-wide text-neutral-500">
						{#if step === 'phone'}
							Resident & Administrator Access
						{:else}
							Identity Verification
						{/if}
					</span>

					{#if step === 'phone'}
						<h1
							class="mt-3 font-sans text-5xl leading-[0.9] font-extrabold tracking-tight text-neutral-950 sm:text-6xl lg:text-7xl"
						>
							Welcome to<br />
							<span class="group/title relative inline-block cursor-default pb-2">
								chs<span class="text-neutral-900">.</span>
								<span
									class="absolute bottom-0 left-0 h-[3.5px] w-12 bg-neutral-950 transition-all duration-300 group-hover/title:w-full sm:w-16"
								></span>
							</span>
						</h1>
						<p class="mt-3 font-sans text-xs leading-relaxed text-neutral-500 sm:text-sm">
							Enter your registered mobile phone number to receive a secure one-time passcode.
						</p>
					{:else}
						<h1
							class="mt-3 font-sans text-4xl leading-tight font-extrabold tracking-tight text-neutral-950 sm:text-5xl"
						>
							Check your phone.
						</h1>
						<div class="mt-3 flex items-center gap-2 text-sm text-neutral-600">
							<span>Code sent to</span>
							<span class="font-mono font-semibold text-neutral-950">
								{formatFullPhone(normalizePhoneInput(phoneInput))}
							</span>
							<button
								type="button"
								onclick={handleEditPhone}
								class="ml-1 cursor-pointer text-xs font-medium text-neutral-950 underline underline-offset-4 hover:opacity-75"
							>
								Change
							</button>
						</div>
					{/if}
				</div>

				<!-- Error Banner -->
				{#if errorMessage}
					<div
						class="mt-6 flex items-start gap-3 rounded-2xl border border-rose-200 bg-rose-50/80 p-4 text-xs text-rose-800"
					>
						<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
						<p class="leading-relaxed">{errorMessage}</p>
					</div>
				{/if}

				<!-- Dev OTP Notice Banner -->
				{#if devOtp && step === 'otp'}
					<div
						class="mt-6 flex items-center justify-between rounded-2xl border border-emerald-200 bg-emerald-50/80 p-4 text-xs text-emerald-900"
					>
						<div class="flex items-center gap-2">
							<Sparkles class="h-4 w-4 text-emerald-600" />
							<span>Development Passcode:</span>
							<span class="font-mono text-sm font-bold tracking-widest text-emerald-950">
								{devOtp}
							</span>
						</div>
						<span class="text-[10px] text-emerald-700">Auto-filled</span>
					</div>
				{/if}

				<!-- Step 1: Phone Form -->
				{#if step === 'phone'}
					<form onsubmit={handleSendOTP} class="mt-8 space-y-6">
						<div>
							<label
								for="phone-input"
								class="block font-sans text-xs font-semibold tracking-wider text-neutral-900 uppercase"
							>
								Mobile Phone Number
							</label>
							<div class="relative mt-2">
								<div
									class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4 font-mono text-sm font-medium text-neutral-500"
								>
									+91
								</div>
								<input
									id="phone-input"
									type="tel"
									inputmode="numeric"
									autocomplete="tel"
									placeholder="98200 12345"
									value={phoneInput}
									oninput={handlePhoneChange}
									disabled={isSubmitting}
									class="w-full rounded-2xl border border-neutral-300 bg-white py-4 pr-4 pl-14 font-mono text-base font-medium tracking-wider text-neutral-950 placeholder:font-sans placeholder:text-sm placeholder:tracking-normal placeholder:text-neutral-400 focus:border-neutral-950 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden disabled:opacity-50"
								/>
							</div>
							<p class="mt-2 font-sans text-[11px] text-neutral-500">
								We'll send an SMS with a 6-digit verification code. Standard carrier rates may
								apply.
							</p>
						</div>

						<Button
							type="submit"
							variant="primary"
							size="lg"
							disabled={isSubmitting || phoneInput.length < 10}
							class="group w-full rounded-2xl py-4 font-sans text-sm font-semibold shadow-md active:scale-[0.99]"
						>
							{#if isSubmitting}
								<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								<span>Sending code...</span>
							{:else}
								<span>Continue with Phone</span>
								<ArrowRight
									class="ml-2 h-4 w-4 transition-transform duration-300 group-hover:translate-x-1"
								/>
							{/if}
						</Button>
					</form>

					<!-- Step 2: OTP Form -->
				{:else}
					<form onsubmit={handleVerifyOTP} class="mt-8 space-y-6">
						<div>
							<label
								for="otp-input"
								class="block font-sans text-xs font-semibold tracking-wider text-neutral-900 uppercase"
							>
								Enter 6-Digit Passcode
							</label>
							<div class="relative mt-2">
								<input
									id="otp-input"
									type="text"
									inputmode="numeric"
									autocomplete="one-time-code"
									maxlength="6"
									placeholder="&bull;&bull;&bull;&bull;&bull;&bull;"
									value={otpInput}
									oninput={handleOtpChange}
									disabled={isSubmitting}
									class="w-full rounded-2xl border border-neutral-300 bg-white py-4 text-center font-mono text-2xl font-bold tracking-[0.5em] text-neutral-950 placeholder:tracking-[0.5em] placeholder:text-neutral-300 focus:border-neutral-950 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden disabled:opacity-50"
								/>
							</div>
						</div>

						<div class="flex items-center justify-between font-sans text-xs">
							{#if cooldown > 0}
								<span class="text-neutral-500">
									Resend available in <strong class="font-mono text-neutral-950">{cooldown}s</strong
									>
								</span>
							{:else}
								<button
									type="button"
									disabled={isSubmitting}
									onclick={() =>
										auth.sendOTP(normalizePhoneInput(phoneInput)).then((r) => {
											startCooldown(r.resend_after_seconds);
											if (r.otp) {
												devOtp = r.otp;
												otpInput = r.otp;
											}
										})}
									class="inline-flex cursor-pointer items-center gap-1.5 font-sans font-medium text-neutral-950 underline underline-offset-4 hover:opacity-75 disabled:opacity-50"
								>
									<RotateCw class="h-3 w-3" />
									<span>Resend code</span>
								</button>
							{/if}
						</div>

						<Button
							type="submit"
							variant="primary"
							size="lg"
							disabled={isSubmitting || otpInput.length < 6}
							class="group w-full rounded-2xl py-4 font-sans text-sm font-semibold shadow-md active:scale-[0.99]"
						>
							{#if isSubmitting}
								<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								<span>Verifying...</span>
							{:else}
								<span>Verify & Sign In</span>
								<CheckCircle2
									class="ml-2 h-4 w-4 transition-transform duration-300 group-hover:scale-110"
								/>
							{/if}
						</Button>
					</form>
				{/if}
			</div>

			<!-- Bottom Footer info -->
			<div class="mt-10 pt-6 text-center font-sans text-xs text-neutral-500">
				<span>Protected by CHS session governance with two-device concurrent session control.</span>
			</div>
		</div>

		<!-- Right Column: Visual Composition -->
		<AuthVisualCard />
	</div>

	<!-- Multi-Society Mandatory Post-Login Selector -->
	<SocietySelectorDialog
		isOpen={isSelectorOpen}
		societies={multipleSocieties}
		isDismissible={false}
		onClose={() => (isSelectorOpen = false)}
		onSelect={(soc) => goto(appRoutes.society(soc.society_id), { replaceState: true })}
	/>
</div>
