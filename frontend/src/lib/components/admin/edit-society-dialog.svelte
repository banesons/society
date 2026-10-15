<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import { societyApi, type SocietyItem } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { normalizePhoneInput } from '$lib/phone';
	import { Edit3, X, AlertCircle, Loader2, Layers } from '@lucide/svelte';

	let {
		isOpen = false,
		society = null,
		onClose,
		onSuccess,
		onOpenFormation
	}: {
		isOpen: boolean;
		society: SocietyItem | null;
		onClose: () => void;
		onSuccess?: () => void;
		onOpenFormation?: (society: SocietyItem) => void;
	} = $props();

	let name = $state('');
	let registrationNumber = $state('');
	let addressLine1 = $state('');
	let addressLine2 = $state('');
	let locality = $state('');
	let city = $state('');
	let stateName = $state('');
	let pincode = $state('');
	let contactEmail = $state('');
	let contactPhone = $state('');

	let isLoading = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		if (society) {
			name = society.name;
			registrationNumber = society.registration_number || '';
			addressLine1 = society.address_line1;
			addressLine2 = society.address_line2 || '';
			locality = society.locality;
			city = society.city;
			stateName = society.state;
			pincode = society.pincode;
			contactEmail = society.contact_email || '';
			contactPhone = society.contact_phone || '';
			error = null;
		}
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!society) return;
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
			await societyApi.updateSociety(society.id, {
				name: trimmedName,
				registration_number: registrationNumber.trim() || undefined,
				address_line1: addressLine1.trim(),
				address_line2: addressLine2.trim() || undefined,
				locality: locality.trim(),
				city: city.trim(),
				state: stateName.trim(),
				pincode: trimmedPin,
				contact_email: contactEmail.trim() || undefined,
				contact_phone: contactPhone.trim() ? normalizePhoneInput(contactPhone) : undefined
			});

			if (onSuccess) {
				onSuccess();
			}
			onClose();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message || 'Failed to update society profile.';
			} else {
				error = 'An unexpected error occurred. Please try again.';
			}
		} finally {
			isLoading = false;
		}
	}
</script>

{#if isOpen && society}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6"
		role="dialog"
		aria-labelledby="edit-society-title"
		aria-modal="true"
	>
		<!-- Backdrop -->
		<button
			type="button"
			tabindex="-1"
			class="fixed inset-0 bg-neutral-950/40 backdrop-blur-xs transition-opacity"
			onclick={() => !isLoading && onClose()}
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
						<Edit3 class="h-4 w-4" />
					</div>
					<div>
						<h2
							id="edit-society-title"
							class="text-base font-semibold tracking-tight text-neutral-900"
						>
							Edit Society Profile
						</h2>
						<p class="text-xs text-neutral-500">
							Update operational profile for <span class="font-mono font-bold text-neutral-900"
								>{society.society_code}</span
							>
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={() => !isLoading && onClose()}
					disabled={isLoading}
					class="rounded-full p-1.5 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900 disabled:opacity-50"
					aria-label="Close dialog"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

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

				<!-- Read-only Society Code Note -->
				<div
					class="flex items-center justify-between rounded-xl border border-neutral-200 bg-neutral-50 px-3.5 py-2.5 text-xs text-neutral-600"
				>
					<span>Society Code (Immutable):</span>
					<span class="rounded bg-neutral-100 px-2 py-0.5 font-mono font-bold text-neutral-900"
						>{society.society_code}</span
					>
				</div>

				{#if onOpenFormation && society.status === 'ACTIVE'}
					<div
						class="flex items-center justify-between gap-3 rounded-2xl border border-neutral-200 bg-neutral-50/80 p-3.5 text-xs"
					>
						<div class="flex items-center gap-2 text-neutral-800">
							<Layers class="h-4 w-4 shrink-0 text-neutral-900" />
							<span>Configure physical structure (buildings, wings, units & gates)?</span>
						</div>
						<Button
							type="button"
							variant="outline"
							size="sm"
							class="shrink-0 border-neutral-300 text-xs hover:bg-neutral-100"
							onclick={() => society && onOpenFormation(society)}
						>
							Open Formation Wizard
						</Button>
					</div>
				{/if}

				<div>
					<label for="edit-society-name" class="block text-xs font-medium text-neutral-700">
						Society Name *
					</label>
					<input
						id="edit-society-name"
						type="text"
						bind:value={name}
						disabled={isLoading}
						required
						class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
					/>
				</div>

				<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
					<div>
						<label for="edit-society-reg-no" class="block text-xs font-medium text-neutral-700">
							Registration Number
						</label>
						<input
							id="edit-society-reg-no"
							type="text"
							bind:value={registrationNumber}
							disabled={isLoading}
							class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
						/>
					</div>

					<div>
						<label for="edit-society-locality" class="block text-xs font-medium text-neutral-700">
							Locality / Area *
						</label>
						<input
							id="edit-society-locality"
							type="text"
							bind:value={locality}
							disabled={isLoading}
							required
							class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
						/>
					</div>
				</div>

				<div>
					<label for="edit-society-addr-1" class="block text-xs font-medium text-neutral-700">
						Address Line 1 *
					</label>
					<input
						id="edit-society-addr-1"
						type="text"
						bind:value={addressLine1}
						disabled={isLoading}
						required
						class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
					/>
				</div>

				<div>
					<label for="edit-society-addr-2" class="block text-xs font-medium text-neutral-700">
						Address Line 2 (Optional)
					</label>
					<input
						id="edit-society-addr-2"
						type="text"
						bind:value={addressLine2}
						disabled={isLoading}
						class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
					/>
				</div>

				<div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
					<div>
						<label for="edit-society-city" class="block text-xs font-medium text-neutral-700">
							City *
						</label>
						<input
							id="edit-society-city"
							type="text"
							bind:value={city}
							disabled={isLoading}
							required
							class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
						/>
					</div>

					<div>
						<label for="edit-society-state" class="block text-xs font-medium text-neutral-700">
							State *
						</label>
						<input
							id="edit-society-state"
							type="text"
							bind:value={stateName}
							disabled={isLoading}
							required
							class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
						/>
					</div>

					<div>
						<label for="edit-society-pincode" class="block text-xs font-medium text-neutral-700">
							PIN Code *
						</label>
						<input
							id="edit-society-pincode"
							type="text"
							maxlength="6"
							bind:value={pincode}
							disabled={isLoading}
							required
							class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 font-mono text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
						/>
					</div>
				</div>

				<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
					<div>
						<label for="edit-society-email" class="block text-xs font-medium text-neutral-700">
							Contact Email
						</label>
						<input
							id="edit-society-email"
							type="email"
							bind:value={contactEmail}
							disabled={isLoading}
							class="mt-1 block w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden disabled:opacity-50"
						/>
					</div>

					<div>
						<label for="edit-society-phone" class="block text-xs font-medium text-neutral-700">
							Contact Phone
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
								id="edit-society-phone"
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
						onclick={() => onClose()}
						disabled={isLoading}
					>
						Cancel
					</Button>
					<Button type="submit" variant="primary" size="sm" disabled={isLoading}>
						{#if isLoading}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Saving...</span>
						{:else}
							<span>Save Profile</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
