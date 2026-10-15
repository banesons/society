<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Logo from '$lib/components/ui/logo.svelte';
	import AuthVisualCard from '$lib/components/auth/auth-visual-card.svelte';
	import { normalizePhoneInput, isValidIndianPhone, formatFullPhone } from '$lib/phone';
	import {
		KeyRound,
		ArrowRight,
		RotateCw,
		AlertCircle,
		CheckCircle2,
		Loader2,
		ArrowLeft,
		Sparkles
	} from '@lucide/svelte';

	import { resolvePostLoginRoute } from '$lib/routing';
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
		const resolution = resolvePostLoginRoute(auth.user);
		if (resolution.type === 'navigate') {
			goto(resolution.path);
		} else if (resolution.type === 'select_society') {
			multipleSocieties = resolution.assignments;
			isSelectorOpen = true;
		} else {
			goto('/unauthorized');
		}
	}

	onMount(() => {
		if (auth.isAuthenticated && auth.user) {
			handlePostLoginRouting();
		}
	});

	function handlePhoneChange(e: Event) {
		const target = e.target as HTMLInputElement;
		// Strictly allow only integer digits and clamp to 10 digits
		const digits = target.value.replace(/\D/g, '').slice(0, 10);
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
</svelte:head>

<!-- Outer Page: Fullscreen on mobile, centered editorial card on desktop -->
<div
	class="mx-auto flex min-h-screen w-full items-center justify-center p-0 sm:max-w-7xl sm:px-6 sm:py-8"
>
	<!-- Split Container Card: full bleed on mobile, rounded card on tablet/desktop -->
	<div
		class="grid min-h-screen w-full grid-cols-1 overflow-hidden border-0 border-neutral-200/90 bg-white p-4 shadow-none sm:min-h-0 sm:max-w-6xl sm:rounded-[2.5rem] sm:border sm:p-5 sm:shadow-2xl lg:grid-cols-12"
	>
		<!-- Left Column: Primary Interaction (Constant layout structure across Phone & OTP) -->
		<div class="flex flex-col justify-between p-4 sm:p-8 md:p-10 lg:col-span-6 lg:p-12 xl:p-14">
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
							Resident Access
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
							class="mt-3 font-sans text-5xl leading-[0.9] font-extrabold tracking-tight text-neutral-950 sm:text-6xl lg:text-7xl"
						>
							Verify your<br />passcode.
						</h1>
						<p class="mt-3 font-sans text-xs leading-relaxed text-neutral-500 sm:text-sm">
							Enter the 6-digit verification code sent to
							<span class="font-mono font-semibold text-neutral-900"
								>{formatFullPhone(phoneInput)}</span
							>
						</p>
					{/if}
				</div>

				<!-- Error Alert -->
				{#if errorMessage}
					<div
						role="alert"
						class="mt-6 flex items-start gap-2.5 rounded-2xl border border-rose-200 bg-rose-50/90 p-3.5 font-sans text-xs text-rose-800"
					>
						<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
						<span>{errorMessage}</span>
					</div>
				{/if}

				<!-- Interactive Form Section -->
				{#if step === 'phone'}
					<form onsubmit={handleSendOTP} class="mt-8 space-y-5">
						<div>
							<label for="login-phone" class="block font-sans text-xs font-medium text-neutral-700">
								Mobile Phone Number <span class="text-rose-500">*</span>
							</label>
							<div
								class="mt-2 flex rounded-2xl border border-neutral-300 bg-white shadow-xs transition-all focus-within:border-neutral-950 focus-within:ring-2 focus-within:ring-neutral-950/10"
							>
								<span
									class="inline-flex items-center rounded-l-2xl border-r border-neutral-200 bg-neutral-50 px-4 font-mono text-sm font-semibold text-neutral-700 select-none"
								>
									+91
								</span>
								<input
									id="login-phone"
									name="phone"
									type="tel"
									inputmode="numeric"
									pattern="[0-9]*"
									maxlength="10"
									value={phoneInput}
									oninput={handlePhoneChange}
									placeholder="9820123456"
									disabled={isSubmitting}
									autocomplete="tel"
									required
									class="block w-full bg-transparent px-4 py-3.5 font-mono text-base text-neutral-950 placeholder-neutral-400 focus:outline-hidden disabled:opacity-50"
								/>
							</div>
							<p class="mt-2 font-sans text-[11px] text-neutral-400">
								10-digit Indian mobile number (e.g. <span class="font-mono text-neutral-600"
									>9967089101</span
								>,
								<span class="font-mono text-neutral-600">9184228620</span>).
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
								<span>Continue</span>
								<ArrowRight
									class="ml-2 h-4 w-4 transition-transform duration-300 group-hover:translate-x-1.5"
								/>
							{/if}
						</Button>
					</form>
				{:else}
					<form onsubmit={handleVerifyOTP} class="mt-8 space-y-5">
						<div>
							<div class="flex items-center justify-between">
								<label for="login-otp" class="block font-sans text-xs font-medium text-neutral-700">
									6-Digit Verification Code <span class="text-rose-500">*</span>
								</label>
								<button
									type="button"
									onclick={handleEditPhone}
									class="cursor-pointer font-sans text-xs font-medium text-neutral-950 underline underline-offset-4 transition-colors hover:opacity-75"
								>
									Change number
								</button>
							</div>

							<div class="relative mt-2">
								<div
									class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4 text-neutral-400"
								>
									<KeyRound class="h-4 w-4" />
								</div>
								<input
									id="login-otp"
									name="otp"
									type="text"
									inputmode="numeric"
									pattern="[0-9]*"
									maxlength="6"
									value={otpInput}
									oninput={handleOtpChange}
									placeholder="123456"
									disabled={isSubmitting}
									autocomplete="one-time-code"
									required
									class="block w-full rounded-2xl border border-neutral-300 bg-white py-3.5 pr-4 pl-12 text-center font-mono text-xl tracking-[0.4em] text-neutral-950 placeholder-neutral-300 shadow-xs transition-all focus:border-neutral-950 focus:ring-2 focus:ring-neutral-950/10 focus:outline-hidden disabled:opacity-50"
								/>
							</div>

							{#if devOtp}
								<div
									class="mt-3 flex items-center justify-between rounded-2xl border border-amber-200 bg-amber-50/80 p-3.5 font-sans text-xs text-amber-900"
								>
									<div class="flex items-center gap-2">
										<Sparkles class="h-4 w-4 text-amber-600" />
										<span>Dev code:</span>
										<span class="font-mono text-sm font-bold tracking-widest text-neutral-950">
											{devOtp}
										</span>
									</div>
									<span class="font-sans text-[10px] text-amber-700">(Auto-filled)</span>
								</div>
							{/if}
						</div>

						<div class="flex items-center justify-between font-sans text-xs">
							<span class="text-neutral-500">Didn't receive code?</span>
							{#if cooldown > 0}
								<!-- Elegantly animated countdown indicator -->
								<span
									class="inline-flex items-center gap-2 rounded-full border border-neutral-200 bg-neutral-100/90 px-3 py-1 font-sans text-xs text-neutral-700 transition-all"
								>
									<span class="relative flex h-2 w-2">
										<span
											class="absolute inline-flex h-full w-full animate-ping rounded-full bg-amber-400 opacity-75"
										></span>
										<span class="relative inline-flex h-2 w-2 rounded-full bg-amber-500"></span>
									</span>
									<span
										>Resend in <span class="font-mono font-semibold text-neutral-900"
											>{cooldown}s</span
										></span
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

			<!-- Bottom Sign Up text -->
			<div class="mt-10 pt-6 text-center font-sans text-xs text-neutral-500">
				<span>Don't have an account? </span>
				<span
					role="button"
					tabindex="0"
					class="cursor-pointer font-semibold text-neutral-950 underline underline-offset-4 transition-colors hover:text-black"
				>
					Sign up
				</span>
				<div class="mt-2 text-[11px] text-neutral-400">
					Safe and secure sign-in protected by community encryption.
				</div>
			</div>
		</div>

		<!-- Right Column: Visual Composition (Reused via AuthVisualCard component) -->
		<AuthVisualCard />
	</div>

	<!-- Multi-Society Mandatory Post-Login Selector -->
	<SocietySelectorDialog
		isOpen={isSelectorOpen}
		societies={multipleSocieties}
		isDismissible={false}
		onClose={() => (isSelectorOpen = false)}
		onSelect={(soc) => goto(`/society/${soc.society_id}`)}
	/>
</div>
