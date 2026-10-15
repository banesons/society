<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import Card from '$lib/components/ui/card.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { normalizePhoneInput, isValidIndianPhone, formatFullPhone } from '$lib/phone';
	import { KeyRound, ArrowRight, RotateCw, X, ShieldAlert, Sparkles } from '@lucide/svelte';

	let {
		isOpen = false,
		onClose
	}: {
		isOpen: boolean;
		onClose: () => void;
	} = $props();

	let step = $state<'phone' | 'otp'>('phone');
	let phoneInput = $state('');
	let otpInput = $state('');
	let devOtp = $state<string | null>(null);
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);
	let cooldownSeconds = $state(0);
	let cooldownTimer: ReturnType<typeof setInterval> | null = null;

	function startCooldown(seconds: number) {
		cooldownSeconds = seconds;
		if (cooldownTimer) clearInterval(cooldownTimer);
		cooldownTimer = setInterval(() => {
			if (cooldownSeconds > 0) {
				cooldownSeconds -= 1;
			} else {
				if (cooldownTimer) clearInterval(cooldownTimer);
			}
		}, 1000);
	}

	async function handleSendOTP() {
		if (isSubmitting) return;
		errorMessage = null;
		const cleanPhone = normalizePhoneInput(phoneInput);
		if (!cleanPhone) {
			errorMessage = 'Please enter your mobile number';
			return;
		}

		if (!isValidIndianPhone(cleanPhone)) {
			errorMessage = 'Please enter a valid Indian mobile number (e.g. 919820123456 or 9820123456)';
			return;
		}

		isSubmitting = true;
		try {
			const res = await auth.sendOTP(cleanPhone);
			step = 'otp';
			startCooldown(res.resend_after_seconds);
			if (res.otp) {
				devOtp = res.otp;
				otpInput = res.otp; // Auto-fill in development!
			}
		} catch (err: unknown) {
			errorMessage = err instanceof Error ? err.message : 'Failed to dispatch verification code';
		} finally {
			isSubmitting = false;
		}
	}

	async function handleVerifyOTP() {
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
			step = 'phone';
			phoneInput = '';
			otpInput = '';
			devOtp = null;
			onClose();
		} catch (err: unknown) {
			errorMessage = err instanceof Error ? err.message : 'Verification failed';
		} finally {
			isSubmitting = false;
		}
	}

	function handleResend() {
		if (cooldownSeconds > 0) return;
		handleSendOTP();
	}

	function fillTestNumber(num: string) {
		phoneInput = num;
		if (step === 'otp') {
			otpInput = '123456';
		}
	}
</script>

{#if isOpen}
	<div
		class="animate-in fade-in fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs duration-200"
	>
		<Card
			class="relative w-full max-w-md rounded-3xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-2xl"
		>
			<!-- Close button -->
			<button
				onclick={onClose}
				class="absolute top-4 right-4 rounded-full p-1 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
				aria-label="Close dialog"
			>
				<X class="h-5 w-5" />
			</button>

			<div class="mb-6">
				<Badge variant="default" class="mb-2">S01 · Identity</Badge>
				<h2 class="text-xl font-bold tracking-tight text-neutral-950">
					{step === 'phone' ? 'Sign In / Register' : 'Enter Verification Code'}
				</h2>
				<p class="mt-1 text-xs text-neutral-500">
					{step === 'phone'
						? 'Enter your mobile number to receive a one-time passcode.'
						: `Sent to ${formatFullPhone(phoneInput)}. Enter the 6-digit code.`}
				</p>
			</div>

			{#if errorMessage}
				<div
					class="mb-4 flex items-start gap-2.5 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
				>
					<ShieldAlert class="h-4 w-4 shrink-0 text-rose-600" />
					<p>{errorMessage}</p>
				</div>
			{/if}

			{#if step === 'phone'}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						handleSendOTP();
					}}
					class="space-y-4"
				>
					<div>
						<label for="phone" class="mb-1.5 block text-xs font-medium text-neutral-700">
							Mobile Number
						</label>
						<div
							class="relative flex items-center overflow-hidden rounded-xl border border-neutral-300 bg-white transition-colors focus-within:border-neutral-950"
						>
							<span
								class="inline-flex items-center border-r border-neutral-200 bg-neutral-100 px-3 text-sm font-semibold text-neutral-600 select-none"
							>
								+
							</span>
							<input
								id="phone"
								type="tel"
								bind:value={phoneInput}
								placeholder="919820123456"
								class="w-full bg-transparent px-3 py-2.5 font-mono text-sm text-neutral-900 placeholder-neutral-400 focus:outline-hidden"
								disabled={isSubmitting}
								autocomplete="tel"
							/>
						</div>
					</div>

					<div class="pt-2">
						<Button type="submit" variant="primary" class="w-full" disabled={isSubmitting}>
							{#if isSubmitting}
								<RotateCw class="mr-1 h-4 w-4 animate-spin" />
								<span>Sending OTP...</span>
							{:else}
								<span>Get Verification Code</span>
								<ArrowRight class="ml-1 h-4 w-4" />
							{/if}
						</Button>
					</div>

					<!-- Dev hints -->
					<div class="mt-4 border-t border-neutral-100 pt-4 text-[11px] text-neutral-500">
						<span class="font-medium text-neutral-700">Development Test Numbers:</span>
						<div class="mt-1.5 flex gap-2">
							<button
								type="button"
								onclick={() => fillTestNumber('919967089101')}
								class="rounded-full bg-neutral-100 px-2.5 py-1 font-mono text-xs text-neutral-800 transition-colors hover:bg-neutral-200"
							>
								919967089101
							</button>
							<button
								type="button"
								onclick={() => fillTestNumber('918422862001')}
								class="rounded-full bg-neutral-100 px-2.5 py-1 font-mono text-xs text-neutral-800 transition-colors hover:bg-neutral-200"
							>
								918422862001
							</button>
						</div>
					</div>
				</form>
			{:else}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						handleVerifyOTP();
					}}
					class="space-y-4"
				>
					<div>
						<label for="otp" class="mb-1.5 block text-xs font-medium text-neutral-700">
							6-Digit Passcode
						</label>
						<div class="relative flex items-center">
							<KeyRound class="absolute left-3 h-4 w-4 text-neutral-400" />
							<input
								id="otp"
								type="text"
								bind:value={otpInput}
								maxlength={6}
								placeholder="123456"
								class="w-full rounded-xl border border-neutral-300 bg-white py-2.5 pr-4 pl-10 text-center font-mono text-base tracking-widest text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
								disabled={isSubmitting}
								autocomplete="one-time-code"
							/>
						</div>

						{#if devOtp}
							<div
								class="mt-3 flex items-center justify-between rounded-2xl border border-amber-200 bg-amber-50 p-3 text-xs text-amber-900"
							>
								<div class="flex items-center gap-1.5">
									<Sparkles class="h-3.5 w-3.5 text-amber-600" />
									<span>Dev OTP:</span>
									<span class="font-mono text-sm font-bold tracking-widest text-neutral-950"
										>{devOtp}</span
									>
								</div>
								<span class="text-[10px] text-amber-700">(Auto-filled)</span>
							</div>
						{/if}
					</div>

					<div class="pt-2">
						<Button type="submit" variant="primary" class="w-full" disabled={isSubmitting}>
							{#if isSubmitting}
								<RotateCw class="mr-1 h-4 w-4 animate-spin" />
								<span>Verifying...</span>
							{:else}
								<span>Verify & Continue</span>
							{/if}
						</Button>
					</div>

					<div class="flex items-center justify-between pt-2 text-xs">
						<button
							type="button"
							onclick={() => {
								step = 'phone';
								errorMessage = null;
								devOtp = null;
							}}
							class="text-neutral-500 transition-colors hover:text-neutral-900"
						>
							Change Number
						</button>

						{#if cooldownSeconds > 0}
							<span class="font-mono text-neutral-400">
								Resend in {cooldownSeconds}s
							</span>
						{:else}
							<button
								type="button"
								onclick={handleResend}
								class="font-medium text-neutral-900 underline underline-offset-2 hover:text-black"
							>
								Resend Code
							</button>
						{/if}
					</div>

					<!-- Dev hints -->
					<div
						class="mt-4 flex items-center justify-between border-t border-neutral-100 pt-3 text-[11px] text-neutral-500"
					>
						<span>Development Test Code:</span>
						<button
							type="button"
							onclick={() => {
								otpInput = '123456';
							}}
							class="rounded-full bg-neutral-100 px-2.5 py-0.5 font-mono text-neutral-800 transition-colors hover:bg-neutral-200"
						>
							Use 123456
						</button>
					</div>
				</form>
			{/if}
		</Card>
	</div>
{/if}
