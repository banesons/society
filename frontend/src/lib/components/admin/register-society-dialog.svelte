<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { societyApi, type SocietyItem } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import {
		Building2,
		X,
		AlertCircle,
		CheckCircle2,
		Loader2,
		Copy,
		Check,
		ArrowRight
	} from '@lucide/svelte';

	let {
		isOpen = false,
		onClose,
		onSuccess
	}: {
		isOpen: boolean;
		onClose: () => void;
		onSuccess?: () => void;
	} = $props();

	let name = $state('');
	let registrationNumber = $state('');
	let addressLine1 = $state('');
	let addressLine2 = $state('');
	let locality = $state('');
	let city = $state('Mumbai');
	let stateName = $state('Maharashtra');
	let pincode = $state('');
	let contactEmail = $state('');
	let contactPhone = $state('');

	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let createdSociety = $state<SocietyItem | null>(null);
	let copied = $state(false);

	function resetForm() {
		name = '';
		registrationNumber = '';
		addressLine1 = '';
		addressLine2 = '';
		locality = '';
		city = 'Mumbai';
		stateName = 'Maharashtra';
		pincode = '';
		contactEmail = '';
		contactPhone = '';
		error = null;
		createdSociety = null;
		copied = false;
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		error = null;

		const trimmedName = name.trim();
		if (trimmedName.length < 3) {
			error = 'Society name must be at least 3 characters.';
			return;
		}

		const trimmedPin = pincode.trim();
		if (!/^[1-9][0-9]{5}$/.test(trimmedPin)) {
			error = 'Please enter a valid 6-digit Indian PIN code.';
			return;
		}

		isLoading = true;
		try {
			const soc = await societyApi.createSociety({
				name: trimmedName,
				address_line1: addressLine1.trim(),
				address_line2: addressLine2.trim() || undefined,
				locality: locality.trim(),
				city: city.trim(),
				state: stateName.trim(),
				pincode: trimmedPin,
				registration_number: registrationNumber.trim() || undefined,
				contact_email: contactEmail.trim() || undefined,
				contact_phone: contactPhone.trim() || undefined
			});

			createdSociety = soc;
			if (onSuccess) {
				onSuccess();
			}
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message || 'Failed to register society.';
			} else {
				error = 'An unexpected error occurred. Please try again.';
			}
		} finally {
			isLoading = false;
		}
	}

	async function copyCode() {
		if (!createdSociety) return;
		try {
			await navigator.clipboard.writeText(createdSociety.society_code);
			copied = true;
			setTimeout(() => {
				copied = false;
			}, 2000);
		} catch {
			// fallback
		}
	}

	function handleClose() {
		resetForm();
		onClose();
	}
</script>

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6"
		role="dialog"
		aria-labelledby="register-society-title"
		aria-modal="true"
	>
		<!-- Backdrop -->
		<button
			type="button"
			tabindex="-1"
			class="fixed inset-0 bg-slate-950/80 backdrop-blur-sm transition-opacity"
			onclick={() => !isLoading && handleClose()}
			aria-label="Close dialog overlay"
		></button>

		<!-- Modal Container -->
		<div
			class="relative max-h-[90vh] w-full max-w-xl overflow-y-auto rounded-2xl border border-slate-800 bg-slate-900 p-6 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-slate-800/80 pb-4">
				<div class="flex items-center gap-2.5">
					<div
						class="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-500/10 text-indigo-400"
					>
						<Building2 class="h-5 w-5" />
					</div>
					<div>
						<h2 id="register-society-title" class="text-base font-semibold text-slate-100">
							Register Housing Society
						</h2>
						<p class="text-xs text-slate-400">
							Provision a new tenant on CHS platform with status PENDING
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={() => !isLoading && handleClose()}
					disabled={isLoading}
					class="rounded-lg p-1.5 text-slate-400 hover:bg-slate-800 hover:text-slate-200 disabled:opacity-50"
					aria-label="Close dialog"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if createdSociety}
				<!-- Success State Displaying Generated Society Code -->
				<div class="mt-6 space-y-5 text-center">
					<div
						class="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-emerald-500/10 text-emerald-400"
					>
						<CheckCircle2 class="h-8 w-8" />
					</div>

					<div>
						<h3 class="text-base font-bold text-slate-100">Society Registered Successfully!</h3>
						<p class="mt-1 text-xs text-slate-400">
							The society has been created in <span class="font-semibold text-amber-400"
								>PENDING</span
							> status.
						</p>
					</div>

					<div class="rounded-xl border border-indigo-500/30 bg-slate-950/80 p-5 text-left">
						<div class="text-[11px] font-medium tracking-wider text-slate-400 uppercase">
							Generated Society Code
						</div>
						<div class="mt-1 flex items-center justify-between">
							<span class="font-mono text-xl font-bold tracking-wide text-indigo-300">
								{createdSociety.society_code}
							</span>
							<Button
								variant="outline"
								size="sm"
								onclick={copyCode}
								class="border-indigo-500/40 text-xs"
							>
								{#if copied}
									<Check class="mr-1 h-3.5 w-3.5 text-emerald-400" />
									<span class="text-emerald-400">Copied</span>
								{:else}
									<Copy class="mr-1 h-3.5 w-3.5 text-indigo-400" />
									<span>Copy Code</span>
								{/if}
							</Button>
						</div>
						<p class="mt-2 text-[11px] text-slate-400">
							This permanent immutable identifier identifies the society across support, audits, and
							administration.
						</p>
					</div>

					<div class="rounded-xl border border-slate-800 bg-slate-950/50 p-4 text-left text-xs">
						<div class="grid grid-cols-2 gap-2 text-slate-300">
							<div>
								<span class="text-slate-400">Name:</span>
								<strong>{createdSociety.name}</strong>
							</div>
							<div>
								<span class="text-slate-400">Location:</span>
								{createdSociety.city}, {createdSociety.state}
							</div>
							<div>
								<span class="text-slate-400">PIN Code:</span>
								{createdSociety.pincode}
							</div>
							<div>
								<span class="text-slate-400">Status:</span>
								<Badge variant="warning">{createdSociety.status}</Badge>
							</div>
						</div>
					</div>

					<div class="flex justify-end pt-2">
						<Button variant="primary" size="md" onclick={handleClose}>
							<span>Done & Return to Roster</span>
							<ArrowRight class="ml-1.5 h-4 w-4" />
						</Button>
					</div>
				</div>
			{:else}
				<!-- Registration Form -->
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

					<div>
						<label for="society-name" class="block text-xs font-medium text-slate-300">
							Society Name *
						</label>
						<input
							id="society-name"
							type="text"
							bind:value={name}
							placeholder="e.g. Gokuldham Co-operative Housing Society"
							disabled={isLoading}
							required
							class="mt-1 block w-full rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none disabled:opacity-50"
						/>
					</div>

					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<div>
							<label for="society-reg-no" class="block text-xs font-medium text-slate-300">
								Registration Number (Optional)
							</label>
							<input
								id="society-reg-no"
								type="text"
								bind:value={registrationNumber}
								placeholder="e.g. MUM/CHS/12345/2020"
								disabled={isLoading}
								class="mt-1 block w-full rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none disabled:opacity-50"
							/>
						</div>

						<div>
							<label for="society-locality" class="block text-xs font-medium text-slate-300">
								Locality / Neighborhood *
							</label>
							<input
								id="society-locality"
								type="text"
								bind:value={locality}
								placeholder="e.g. Film City Road, Goregaon East"
								disabled={isLoading}
								required
								class="mt-1 block w-full rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none disabled:opacity-50"
							/>
						</div>
					</div>

					<div>
						<label for="society-addr-1" class="block text-xs font-medium text-slate-300">
							Address Line 1 *
						</label>
						<input
							id="society-addr-1"
							type="text"
							bind:value={addressLine1}
							placeholder="e.g. Plot No. 10, Sector 14"
							disabled={isLoading}
							required
							class="mt-1 block w-full rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none disabled:opacity-50"
						/>
					</div>

					<div>
						<label for="society-addr-2" class="block text-xs font-medium text-slate-300">
							Address Line 2 (Optional)
						</label>
						<input
							id="society-addr-2"
							type="text"
							bind:value={addressLine2}
							placeholder="e.g. Near Film City Gate 1"
							disabled={isLoading}
							class="mt-1 block w-full rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none disabled:opacity-50"
						/>
					</div>

					<div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
						<div>
							<label for="society-city" class="block text-xs font-medium text-slate-300">
								City *
							</label>
							<input
								id="society-city"
								type="text"
								bind:value={city}
								placeholder="e.g. Mumbai"
								disabled={isLoading}
								required
								class="mt-1 block w-full rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none disabled:opacity-50"
							/>
						</div>

						<div>
							<label for="society-state" class="block text-xs font-medium text-slate-300">
								State *
							</label>
							<input
								id="society-state"
								type="text"
								bind:value={stateName}
								placeholder="e.g. Maharashtra"
								disabled={isLoading}
								required
								class="mt-1 block w-full rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none disabled:opacity-50"
							/>
						</div>

						<div>
							<label for="society-pincode" class="block text-xs font-medium text-slate-300">
								PIN Code *
							</label>
							<input
								id="society-pincode"
								type="text"
								inputmode="numeric"
								maxlength="6"
								bind:value={pincode}
								placeholder="400063"
								disabled={isLoading}
								required
								class="mt-1 block w-full rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none disabled:opacity-50"
							/>
						</div>
					</div>

					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<div>
							<label for="society-email" class="block text-xs font-medium text-slate-300">
								Contact Email (Optional)
							</label>
							<input
								id="society-email"
								type="email"
								bind:value={contactEmail}
								placeholder="admin@gokuldham.org"
								disabled={isLoading}
								class="mt-1 block w-full rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none disabled:opacity-50"
							/>
						</div>

						<div>
							<label for="society-phone" class="block text-xs font-medium text-slate-300">
								Contact Phone (Optional)
							</label>
							<input
								id="society-phone"
								type="tel"
								bind:value={contactPhone}
								placeholder="9820123456"
								disabled={isLoading}
								class="mt-1 block w-full rounded-lg border border-slate-700 bg-slate-950/80 px-3 py-2 text-xs text-slate-100 placeholder-slate-500 focus:border-indigo-500 focus:outline-none disabled:opacity-50"
							/>
						</div>
					</div>

					<div class="flex items-center justify-end gap-3 border-t border-slate-800/80 pt-3">
						<Button
							type="button"
							variant="ghost"
							size="sm"
							onclick={handleClose}
							disabled={isLoading}
						>
							Cancel
						</Button>
						<Button type="submit" variant="primary" size="sm" disabled={isLoading}>
							{#if isLoading}
								<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
								<span>Registering...</span>
							{:else}
								<span>Register Society</span>
							{/if}
						</Button>
					</div>
				</form>
			{/if}
		</div>
	</div>
{/if}
