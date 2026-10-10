<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import { platformApi } from '$lib/api/platform';
	import { ApiClientError } from '$lib/api/client';
	import { UserPlus, X, AlertCircle, CheckCircle2, Loader2, Phone } from '@lucide/svelte';

	let {
		isOpen = false,
		onClose,
		onSuccess
	}: {
		isOpen: boolean;
		onClose: () => void;
		onSuccess?: () => void;
	} = $props();

	let phone = $state('');
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let successMsg = $state<string | null>(null);

	// Client-side validation: must normalize to 10 digits starting with 6, 7, 8, or 9
	function validatePhone(input: string): boolean {
		const digits = input.replace(/\D/g, '');
		if (digits.length === 10 && /^[6-9]/.test(digits)) return true;
		if (digits.length === 12 && digits.startsWith('91') && /^[6-9]/.test(digits.slice(2)))
			return true;
		if (digits.length === 11 && digits.startsWith('0') && /^[6-9]/.test(digits.slice(1)))
			return true;
		return false;
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		successMsg = null;

		const trimmed = phone.trim();
		if (!trimmed) {
			error = 'Please enter a valid mobile number.';
			return;
		}

		if (!validatePhone(trimmed)) {
			error = 'Please enter a valid 10-digit Indian mobile number starting with 6, 7, 8, or 9.';
			return;
		}

		isLoading = true;
		try {
			await platformApi.createAdmin(trimmed);
			successMsg = 'Platform Administrator successfully created.';
			phone = '';
			if (onSuccess) {
				onSuccess();
			}
			setTimeout(() => {
				successMsg = null;
				onClose();
			}, 1200);
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.code === 'ADMIN_ALREADY_EXISTS') {
					error = 'This mobile number is already assigned a platform administrative role.';
				} else if (err.code === 'INVALID_PHONE') {
					error = 'Invalid phone number format. Please provide a valid Indian mobile number.';
				} else if (err.code === 'FORBIDDEN') {
					error = 'Super Admin privileges required to provision administrators.';
				} else {
					error = err.message || 'Failed to create administrator.';
				}
			} else {
				error = 'An unexpected error occurred. Please try again.';
			}
		} finally {
			isLoading = false;
		}
	}

	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Escape' && isOpen && !isLoading) {
			onClose();
		}
	}
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6"
		role="dialog"
		aria-labelledby="create-admin-title"
		aria-modal="true"
	>
		<!-- Backdrop -->
		<button
			type="button"
			tabindex="-1"
			class="fixed inset-0 bg-slate-950/80 backdrop-blur-sm transition-opacity"
			onclick={() => !isLoading && onClose()}
			aria-label="Close dialog overlay"
		></button>

		<!-- Modal Container -->
		<div
			class="relative w-full max-w-md rounded-2xl border border-slate-800 bg-slate-900 p-6 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-slate-800/80 pb-4">
				<div class="flex items-center gap-2.5">
					<div
						class="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-500/10 text-indigo-400"
					>
						<UserPlus class="h-5 w-5" />
					</div>
					<div>
						<h2 id="create-admin-title" class="text-base font-semibold text-slate-100">
							Create Administrator
						</h2>
						<p class="text-xs text-slate-400">Assign platform ADMIN role to a mobile identity</p>
					</div>
				</div>
				<button
					type="button"
					onclick={() => !isLoading && onClose()}
					disabled={isLoading}
					class="rounded-lg p-1.5 text-slate-400 hover:bg-slate-800 hover:text-slate-200 disabled:opacity-50"
					aria-label="Close dialog"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<form onsubmit={handleSubmit} class="mt-5 space-y-4">
				{#if error}
					<div
						role="alert"
						class="flex items-start gap-2.5 rounded-lg border border-rose-500/30 bg-rose-500/10 p-3 text-xs text-rose-300"
					>
						<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-400" />
						<span>{error}</span>
					</div>
				{/if}

				{#if successMsg}
					<div
						role="status"
						class="flex items-center gap-2.5 rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-3 text-xs text-emerald-300"
					>
						<CheckCircle2 class="h-4 w-4 shrink-0 text-emerald-400" />
						<span>{successMsg}</span>
					</div>
				{/if}

				<div>
					<label for="admin-phone" class="block text-xs font-medium text-slate-300">
						Mobile Phone Number
					</label>
					<div class="relative mt-1.5">
						<div
							class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-slate-400"
						>
							<Phone class="h-4 w-4" />
						</div>
						<input
							id="admin-phone"
							name="phone"
							type="tel"
							bind:value={phone}
							placeholder="9820123456"
							disabled={isLoading}
							autocomplete="tel"
							required
							class="block w-full rounded-lg border border-slate-700 bg-slate-950/80 py-2.5 pr-3 pl-9 text-sm text-slate-100 placeholder-slate-500 transition-colors focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 focus:outline-none disabled:opacity-50"
						/>
					</div>
					<p class="mt-1.5 text-[11px] text-slate-400">
						Enter 10-digit Indian mobile number. The administrator will sign in using standard OTP
						verification.
					</p>
				</div>

				<div class="flex items-center justify-end gap-3 pt-3">
					<Button
						type="button"
						variant="ghost"
						size="sm"
						onclick={() => onClose()}
						disabled={isLoading}
					>
						Cancel
					</Button>
					<Button type="submit" variant="primary" size="sm" disabled={isLoading}>
						{#if isLoading}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Creating...</span>
						{:else}
							<span>Create Admin</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
