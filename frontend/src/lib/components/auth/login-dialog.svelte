<script lang="ts">
	import { auth } from '$lib/auth.svelte';
	import Card from '$lib/components/ui/card.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { KeyRound, ArrowRight, RotateCw, X, ShieldAlert } from '@lucide/svelte';

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
		if (!phoneInput.trim()) {
			errorMessage = 'Please enter your mobile number';
			return;
		}

		isSubmitting = true;
		try {
			const resendSeconds = await auth.sendOTP(phoneInput);
			step = 'otp';
			startCooldown(resendSeconds);
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
			await auth.verifyOTP(phoneInput, otpInput);
			if (cooldownTimer) clearInterval(cooldownTimer);
			step = 'phone';
			phoneInput = '';
			otpInput = '';
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
		class="animate-in fade-in fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 p-4 backdrop-blur-sm duration-200"
	>
		<Card class="relative w-full max-w-md border-slate-800 bg-slate-900 p-6 shadow-2xl">
			<!-- Close button -->
			<button
				onclick={onClose}
				class="absolute top-4 right-4 text-slate-400 transition-colors hover:text-slate-200"
				aria-label="Close dialog"
			>
				<X class="h-5 w-5" />
			</button>

			<div class="mb-6">
				<Badge variant="default" class="mb-2">S01 · Identity</Badge>
				<h2 class="text-xl font-bold tracking-tight text-white">
					{step === 'phone' ? 'Sign In / Register' : 'Enter Verification Code'}
				</h2>
				<p class="mt-1 text-xs text-slate-400">
					{step === 'phone'
						? 'Enter your mobile number to receive a one-time passcode.'
						: `Sent to ${phoneInput}. Enter the 6-digit code.`}
				</p>
			</div>

			{#if errorMessage}
				<div
					class="mb-4 flex items-start gap-2.5 rounded-lg border border-rose-500/30 bg-rose-500/10 p-3 text-xs text-rose-300"
				>
					<ShieldAlert class="h-4 w-4 shrink-0 text-rose-400" />
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
						<label for="phone" class="mb-1.5 block text-xs font-medium text-slate-300">
							Mobile Number
						</label>
						<div class="relative flex items-center">
							<span class="absolute left-3 font-mono text-xs text-slate-500 select-none">+91</span>
							<input
								id="phone"
								type="tel"
								bind:value={phoneInput}
								placeholder="9967089101"
								class="w-full rounded-lg border border-slate-700 bg-slate-950 py-2.5 pr-4 pl-11 text-sm text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 focus:outline-none"
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
					<div class="mt-4 border-t border-slate-800 pt-4 text-[11px] text-slate-400">
						<span class="font-medium text-slate-300">Development Test Numbers:</span>
						<div class="mt-1.5 flex gap-2">
							<button
								type="button"
								onclick={() => fillTestNumber('9967089101')}
								class="rounded bg-slate-800 px-2 py-1 font-mono text-slate-300 hover:bg-slate-700"
							>
								9967089101
							</button>
							<button
								type="button"
								onclick={() => fillTestNumber('8422862001')}
								class="rounded bg-slate-800 px-2 py-1 font-mono text-slate-300 hover:bg-slate-700"
							>
								8422862001
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
						<label for="otp" class="mb-1.5 block text-xs font-medium text-slate-300">
							6-Digit Passcode
						</label>
						<div class="relative flex items-center">
							<KeyRound class="absolute left-3 h-4 w-4 text-slate-500" />
							<input
								id="otp"
								type="text"
								bind:value={otpInput}
								maxlength={6}
								placeholder="123456"
								class="w-full rounded-lg border border-slate-700 bg-slate-950 py-2.5 pr-4 pl-10 text-center font-mono text-base tracking-widest text-slate-100 placeholder-slate-600 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 focus:outline-none"
								disabled={isSubmitting}
								autocomplete="one-time-code"
							/>
						</div>
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
							}}
							class="text-slate-400 hover:text-slate-200"
						>
							Change Number
						</button>

						{#if cooldownSeconds > 0}
							<span class="font-mono text-slate-500">
								Resend in {cooldownSeconds}s
							</span>
						{:else}
							<button
								type="button"
								onclick={handleResend}
								class="font-medium text-indigo-400 hover:text-indigo-300"
							>
								Resend Code
							</button>
						{/if}
					</div>

					<!-- Dev hints -->
					<div
						class="mt-4 flex items-center justify-between border-t border-slate-800 pt-3 text-[11px] text-slate-400"
					>
						<span>Development Test Code:</span>
						<button
							type="button"
							onclick={() => {
								otpInput = '123456';
							}}
							class="rounded bg-slate-800 px-2 py-0.5 font-mono text-indigo-400 hover:bg-slate-700"
						>
							Use 123456
						</button>
					</div>
				</form>
			{/if}
		</Card>
	</div>
{/if}
