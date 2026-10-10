<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth.svelte';
	import Card from '$lib/components/ui/card.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import {
		Shield,
		Phone,
		KeyRound,
		ArrowRight,
		RotateCw,
		AlertCircle,
		CheckCircle2,
		Loader2,
		ArrowLeft
	} from '@lucide/svelte';

	let step = $state<'phone' | 'otp'>('phone');
	let phoneInput = $state('');
	let otpInput = $state('');
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);
	let cooldown = $state(0);
	let cooldownTimer: ReturnType<typeof setInterval> | null = null;

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

	function validatePhone(input: string): boolean {
		const digits = input.replace(/\D/g, '');
		if (digits.length === 10 && /^[6-9]/.test(digits)) return true;
		if (digits.length === 12 && digits.startsWith('91') && /^[6-9]/.test(digits.slice(2)))
			return true;
		if (digits.length === 11 && digits.startsWith('0') && /^[6-9]/.test(digits.slice(1)))
			return true;
		return false;
	}

	function redirectBasedOnRole() {
		if (auth.isSuperAdmin) {
			goto('/admin');
		} else {
			goto('/unauthorized');
		}
	}

	onMount(() => {
		// If already authenticated on load, route accordingly
		if (auth.isAuthenticated && auth.user) {
			redirectBasedOnRole();
		}
	});

	async function handleSendOTP(e: SubmitEvent) {
		e.preventDefault();
		if (isSubmitting) return;
		errorMessage = null;

		const trimmed = phoneInput.trim();
		if (!trimmed) {
			errorMessage = 'Please enter your mobile phone number';
			return;
		}

		if (!validatePhone(trimmed)) {
			errorMessage = 'Please enter a valid 10-digit Indian mobile number';
			return;
		}

		isSubmitting = true;
		try {
			const resendSeconds = await auth.sendOTP(trimmed);
			step = 'otp';
			startCooldown(resendSeconds);
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
			await auth.verifyOTP(phoneInput.trim(), otpInput.trim());
			if (cooldownTimer) clearInterval(cooldownTimer);
			redirectBasedOnRole();
		} catch (err: unknown) {
			errorMessage = err instanceof Error ? err.message : 'Verification failed';
		} finally {
			isSubmitting = false;
		}
	}

	function handleEditPhone() {
		step = 'phone';
		otpInput = '';
		errorMessage = null;
	}
</script>

<svelte:head>
	<title>Sign In — CHS Platform</title>
	<meta name="description" content="Sign in to CHS Platform using mobile verification" />
</svelte:head>

<div class="flex min-h-[75vh] items-center justify-center px-4 py-10">
	<div class="w-full max-w-md space-y-6">
		<!-- Back to Home link -->
		<div class="flex items-center justify-between text-xs">
			<a
				href="/"
				class="inline-flex items-center gap-1.5 text-slate-400 transition-colors hover:text-slate-200"
			>
				<ArrowLeft class="h-3.5 w-3.5" />
				<span>Back to Home</span>
			</a>
			<Badge variant="default" class="border-indigo-500/30 bg-indigo-500/10 text-indigo-400">
				Ed25519 JWT Auth
			</Badge>
		</div>

		<!-- Main Card -->
		<Card class="border-slate-800 bg-slate-900/90 p-8 shadow-2xl">
			<!-- Header -->
			<div class="text-center">
				<div
					class="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-tr from-indigo-600 to-violet-500 text-white shadow-lg shadow-indigo-600/30"
				>
					<Shield class="h-6 w-6" />
				</div>
				<h1 class="mt-4 text-xl font-bold tracking-tight text-white">Sign In to CHS Platform</h1>
				<p class="mt-1 text-xs text-slate-400">
					{#if step === 'phone'}
						Enter your mobile phone number to receive a verification code
					{:else}
						Enter the 6-digit verification code sent to <span class="font-mono text-slate-200"
							>{phoneInput}</span
						>
					{/if}
				</p>
			</div>

			<!-- Error Alert -->
			{#if errorMessage}
				<div
					role="alert"
					class="mt-6 flex items-start gap-2.5 rounded-xl border border-rose-500/30 bg-rose-500/10 p-3 text-xs text-rose-300"
				>
					<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-400" />
					<span>{errorMessage}</span>
				</div>
			{/if}

			<!-- Step 1: Phone Form -->
			{#if step === 'phone'}
				<form onsubmit={handleSendOTP} class="mt-6 space-y-4">
					<div>
						<label for="login-phone" class="block text-xs font-medium text-slate-300">
							Mobile Phone Number
						</label>
						<div class="relative mt-1.5">
							<div
								class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5 text-slate-400"
							>
								<Phone class="h-4 w-4" />
							</div>
							<input
								id="login-phone"
								name="phone"
								type="tel"
								bind:value={phoneInput}
								placeholder="9967089101"
								disabled={isSubmitting}
								autocomplete="tel"
								required
								class="block w-full rounded-xl border border-slate-700 bg-slate-950/80 py-2.5 pr-3 pl-10 text-sm text-slate-100 placeholder-slate-500 transition-colors focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 focus:outline-none disabled:opacity-50"
							/>
						</div>
						<p class="mt-1.5 text-[11px] text-slate-400">
							Indian mobile number (+91). Test numbers: <span class="font-mono text-slate-300"
								>9967089101</span
							>, <span class="font-mono text-slate-300">8422862001</span>.
						</p>
					</div>

					<Button
						type="submit"
						variant="primary"
						size="md"
						disabled={isSubmitting}
						class="mt-2 w-full"
					>
						{#if isSubmitting}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
							<span>Sending Code...</span>
						{:else}
							<span>Send Verification Code</span>
							<ArrowRight class="ml-2 h-4 w-4" />
						{/if}
					</Button>
				</form>

				<!-- Step 2: OTP Form -->
			{:else}
				<form onsubmit={handleVerifyOTP} class="mt-6 space-y-4">
					<div>
						<div class="flex items-center justify-between">
							<label for="login-otp" class="block text-xs font-medium text-slate-300">
								6-Digit Verification Code
							</label>
							<button
								type="button"
								onclick={handleEditPhone}
								class="text-[11px] text-indigo-400 transition-colors hover:text-indigo-300"
							>
								Change Number
							</button>
						</div>
						<div class="relative mt-1.5">
							<div
								class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5 text-slate-400"
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
								bind:value={otpInput}
								placeholder="123456"
								disabled={isSubmitting}
								autocomplete="one-time-code"
								required
								class="block w-full rounded-xl border border-slate-700 bg-slate-950/80 py-2.5 pr-3 pl-10 text-center font-mono text-base tracking-widest text-slate-100 placeholder-slate-500 transition-colors focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 focus:outline-none disabled:opacity-50"
							/>
						</div>
					</div>

					<div class="flex items-center justify-between pt-1 text-xs">
						<span class="text-slate-400">Didn't receive code?</span>
						{#if cooldown > 0}
							<span class="font-mono text-slate-400">
								Resend in {cooldown}s
							</span>
						{:else}
							<button
								type="button"
								disabled={isSubmitting}
								onclick={() => auth.sendOTP(phoneInput.trim()).then((s) => startCooldown(s))}
								class="inline-flex items-center gap-1 text-indigo-400 hover:text-indigo-300 disabled:opacity-50"
							>
								<RotateCw class="h-3 w-3" />
								<span>Resend Code</span>
							</button>
						{/if}
					</div>

					<Button
						type="submit"
						variant="primary"
						size="md"
						disabled={isSubmitting}
						class="mt-2 w-full"
					>
						{#if isSubmitting}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
							<span>Verifying...</span>
						{:else}
							<span>Verify & Sign In</span>
							<CheckCircle2 class="ml-2 h-4 w-4" />
						{/if}
					</Button>
				</form>
			{/if}
		</Card>

		<!-- Footer note -->
		<div class="text-center text-xs text-slate-500">
			Protected by server-side rate limiting and automatic 2-device FIFO session policy.
		</div>
	</div>
</div>
