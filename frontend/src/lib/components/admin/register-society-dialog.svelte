<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { societyApi, type SocietyItem } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { normalizePhoneInput } from '$lib/phone';
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
				contact_phone: contactPhone.trim() ? normalizePhoneInput(contactPhone) : undefined
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
			class="fixed inset-0 bg-neutral-950/40 backdrop-blur-xs transition-opacity"
			onclick={() => !isLoading && handleClose()}
			aria-label="Close dialog overlay"
		></button>

		<!-- Modal Container -->
		<div
			class="relative max-h-[90vh] w-full max-w-xl overflow-y-auto rounded-3xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div class="flex items-center gap-2.5">
					<div
						class="flex h-9 w-9 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
					>
						<Building2 class="h-4 w-4" />
					</div>
					<div>
						<h2
							id="register-society-title"
							class="text-base font-semibold tracking-tight text-neutral-900"
						>
							Register Housing Society
						</h2>
						<p class="text-xs text-neutral-500">
							Provision a new tenant on CHS platform with status PENDING
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={() => !isLoading && handleClose()}
					disabled={isLoading}
					class="rounded-full p-1.5 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900 disabled:opacity-50"
					aria-label="Close dialog"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if createdSociety}
				<!-- Success State Displaying Generated Society Code -->
				<div class="mt-6 space-y-5 text-center">
					<div
						class="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-emerald-50 text-emerald-700"
					>
						<CheckCircle2 class="h-7 w-7" />
					</div>

					<div>
						<h3 class="text-base font-bold text-neutral-900">Society Registered Successfully!</h3>
						<p class="mt-1 text-xs text-neutral-500">
							The society has been created in <span class="font-semibold text-amber-700"
								>PENDING</span
							> status.
						</p>
					</div>

					<div class="rounded-2xl border border-neutral-200 bg-neutral-50 p-5 text-left">
						<div class="text-[11px] font-semibold tracking-wider text-neutral-500 uppercase">
							Generated Society Code
						</div>
						<div class="mt-1 flex items-center justify-between">
							<span class="font-mono text-xl font-bold tracking-wide text-neutral-950">
								{createdSociety.society_code}
							</span>
							<Button
								variant="outline"
								size="sm"
								onclick={copyCode}
								class="border-neutral-300 text-xs hover:bg-neutral-100"
							>
								{#if copied}
									<Check class="mr-1 h-3.5 w-3.5 text-emerald-600" />
									<span class="text-emerald-700">Copied</span>
								{:else}
									<Copy class="mr-1 h-3.5 w-3.5 text-neutral-700" />
									<span>Copy Code</span>
								{/if}
							</Button>
						</div>
						<p class="mt-2 text-[11px] text-neutral-500">
							This permanent immutable identifier identifies the society across support, audits, and
							administration.
						</p>
					</div>

					<div class="rounded-2xl border border-neutral-200 bg-neutral-50/60 p-4 text-left text-xs">
						<div class="grid grid-cols-2 gap-2 text-neutral-700">
							<div>
								<span class="text-neutral-500">Name:</span>
								<strong class="ml-1 text-neutral-900">{createdSociety.name}</strong>
							</div>
							<div>
								<span class="text-neutral-500">Location:</span>
								<span class="ml-1 text-neutral-800"
									>{createdSociety.city}, {createdSociety.state}</span
								>
							</div>
							<div>
								<span class="text-neutral-500">PIN Code:</span>
								<span class="ml-1 font-mono text-neutral-800">{createdSociety.pincode}</span>
							</div>
							<div>
								<span class="text-neutral-500">Status:</span>
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
							class="flex items-start gap-2.5 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
						>
							<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
							<span>{error}</span>
						</div>
					{/if}

					<div>
						<label for="society-name" class="block text-xs font-medium text-neutral-700">
							Society Name *
						</label>
						<input
							id="society-name"
							type="text"
							bind:value={name}
							placeholder="e.g. Gokuldham Co-operative Housing Society"
							disabled={isLoading}
							required
							class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
						/>
					</div>

					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<div>
							<label for="society-reg-no" class="block text-xs font-medium text-neutral-700">
								Registration Number (Optional)
							</label>
							<input
								id="society-reg-no"
								type="text"
								bind:value={registrationNumber}
								placeholder="e.g. MUM/CHS/12345/2020"
								disabled={isLoading}
								class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
							/>
						</div>

						<div>
							<label for="society-locality" class="block text-xs font-medium text-neutral-700">
								Locality / Neighborhood *
							</label>
							<input
								id="society-locality"
								type="text"
								bind:value={locality}
								placeholder="e.g. Film City Road, Goregaon East"
								disabled={isLoading}
								required
								class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
							/>
						</div>
					</div>

					<div>
						<label for="society-addr-1" class="block text-xs font-medium text-neutral-700">
							Address Line 1 *
						</label>
						<input
							id="society-addr-1"
							type="text"
							bind:value={addressLine1}
							placeholder="e.g. Plot No. 10, Sector 14"
							disabled={isLoading}
							required
							class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
						/>
					</div>

					<div>
						<label for="society-addr-2" class="block text-xs font-medium text-neutral-700">
							Address Line 2 (Optional)
						</label>
						<input
							id="society-addr-2"
							type="text"
							bind:value={addressLine2}
							placeholder="e.g. Near Film City Gate 1"
							disabled={isLoading}
							class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
						/>
					</div>

					<div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
						<div>
							<label for="society-city" class="block text-xs font-medium text-neutral-700">
								City *
							</label>
							<input
								id="society-city"
								type="text"
								bind:value={city}
								placeholder="e.g. Mumbai"
								disabled={isLoading}
								required
								class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
							/>
						</div>

						<div>
							<label for="society-state" class="block text-xs font-medium text-neutral-700">
								State *
							</label>
							<input
								id="society-state"
								type="text"
								bind:value={stateName}
								placeholder="e.g. Maharashtra"
								disabled={isLoading}
								required
								class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
							/>
						</div>

						<div>
							<label for="society-pincode" class="block text-xs font-medium text-neutral-700">
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
								class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 font-mono text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
							/>
						</div>
					</div>

					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<div>
							<label for="society-email" class="block text-xs font-medium text-neutral-700">
								Contact Email (Optional)
							</label>
							<input
								id="society-email"
								type="email"
								bind:value={contactEmail}
								placeholder="admin@gokuldham.org"
								disabled={isLoading}
								class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
							/>
						</div>

						<div>
							<label for="society-phone" class="block text-xs font-medium text-neutral-700">
								Contact Phone (Optional)
							</label>
							<div
								class="mt-1 flex overflow-hidden rounded-xl border border-neutral-300 bg-white transition-colors focus-within:border-neutral-950"
							>
								<span
									class="inline-flex items-center border-r border-neutral-200 bg-neutral-100 px-2.5 text-xs font-semibold text-neutral-600 select-none"
								>
									+
								</span>
								<input
									id="society-phone"
									type="tel"
									bind:value={contactPhone}
									placeholder="919820123456"
									disabled={isLoading}
									class="w-full bg-transparent px-3 py-2 font-mono text-xs text-neutral-900 placeholder-neutral-400 focus:outline-hidden disabled:opacity-50"
								/>
							</div>
						</div>
					</div>

					<div class="flex items-center justify-end gap-3 border-t border-neutral-100 pt-3">
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
