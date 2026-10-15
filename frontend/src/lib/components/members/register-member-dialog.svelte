<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import { societyApi, type SocietyMember } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { normalizePhoneInput, isValidIndianPhone } from '$lib/phone';
	import { UserPlus, X, AlertCircle, Loader2 } from '@lucide/svelte';

	let {
		isOpen = false,
		societyId,
		onClose,
		onSuccess
	}: {
		isOpen: boolean;
		societyId: string;
		onClose: () => void;
		onSuccess?: (member: SocietyMember) => void;
	} = $props();

	let firstName = $state('');
	let lastName = $state('');
	let phone = $state('');
	let email = $state('');
	let emergencyName = $state('');
	let emergencyPhone = $state('');

	let isLoading = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		if (isOpen) {
			firstName = '';
			lastName = '';
			phone = '';
			email = '';
			emergencyName = '';
			emergencyPhone = '';
			error = null;
		}
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		error = null;

		const trimmedFirst = firstName.trim();
		const cleanPhone = normalizePhoneInput(phone);

		if (!trimmedFirst) {
			error = 'First name is required.';
			return;
		}

		if (!isValidIndianPhone(cleanPhone)) {
			error = 'Valid Indian mobile phone number is required (e.g. 919876543210).';
			return;
		}

		let cleanEmergencyPhone: string | undefined = undefined;
		if (emergencyPhone.trim()) {
			cleanEmergencyPhone = normalizePhoneInput(emergencyPhone);
			if (!isValidIndianPhone(cleanEmergencyPhone)) {
				error = 'Emergency contact phone must be a valid Indian mobile number (e.g. 919876543211).';
				return;
			}
		}

		isLoading = true;

		try {
			const res = await societyApi.registerMember(societyId, {
				first_name: trimmedFirst,
				last_name: lastName.trim() || undefined,
				phone: cleanPhone,
				email: email.trim() || undefined,
				emergency_contact_name: emergencyName.trim() || undefined,
				emergency_contact_phone: cleanEmergencyPhone
			});

			onSuccess?.(res.member);
			onClose();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.code === 'MEMBER_ALREADY_EXISTS' || err.statusCode === 409) {
					error = 'An active member with this phone number is already registered in this society.';
				} else {
					error = err.message || 'Failed to register member.';
				}
			} else {
				error = 'Failed to register member.';
			}
		} finally {
			isLoading = false;
		}
	}
</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape' && isOpen) onClose();
	}}
/>

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
		role="dialog"
		aria-modal="true"
		onclick={(e) => {
			if (e.target === e.currentTarget) onClose();
		}}
	>
		<div
			class="relative w-full max-w-lg rounded-3xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div class="flex items-center gap-3">
					<div
						class="flex h-10 w-10 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
					>
						<UserPlus class="h-5 w-5" />
					</div>
					<div>
						<h3 class="text-base font-semibold tracking-tight text-neutral-950">
							Register Society Member
						</h3>
						<p class="text-xs text-neutral-500">
							Add an owner, tenant, or resident to this society
						</p>
					</div>
				</div>
				<button
					onclick={onClose}
					class="rounded-full p-1.5 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if error}
				<div
					class="mt-4 flex items-start gap-2.5 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
				>
					<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
					<span>{error}</span>
				</div>
			{/if}

			<form onsubmit={handleSubmit} class="mt-4 space-y-4">
				<div class="grid grid-cols-2 gap-3">
					<div>
						<label for="first-name" class="block text-xs font-medium text-neutral-700"
							>First Name *</label
						>
						<input
							id="first-name"
							type="text"
							required
							bind:value={firstName}
							placeholder="e.g. Vikram"
							class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
						/>
					</div>
					<div>
						<label for="last-name" class="block text-xs font-medium text-neutral-700"
							>Last Name</label
						>
						<input
							id="last-name"
							type="text"
							bind:value={lastName}
							placeholder="e.g. Malhotra"
							class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
						/>
					</div>
				</div>

				<div class="grid grid-cols-2 gap-3">
					<div>
						<label for="phone" class="block text-xs font-medium text-neutral-700"> Phone * </label>
						<div
							class="mt-1 flex overflow-hidden rounded-xl border border-neutral-300 bg-white transition-colors focus-within:border-neutral-950"
						>
							<span
								class="inline-flex items-center border-r border-neutral-200 bg-neutral-100 px-3 text-sm font-semibold text-neutral-600 select-none"
							>
								+
							</span>
							<input
								id="phone"
								type="tel"
								required
								bind:value={phone}
								placeholder="919820123456"
								class="w-full bg-transparent px-3 py-2 font-mono text-sm text-neutral-900 placeholder-neutral-400 focus:outline-hidden"
							/>
						</div>
						<p class="mt-1 text-[10px] text-neutral-500">
							Links automatically when user verifies via OTP
						</p>
					</div>
					<div>
						<label for="email" class="block text-xs font-medium text-neutral-700">Email</label>
						<input
							id="email"
							type="email"
							bind:value={email}
							placeholder="vikram@example.com"
							class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
						/>
					</div>
				</div>

				<div class="border-t border-neutral-100 pt-3">
					<p class="text-xs font-medium text-neutral-700">Emergency Contact (Optional)</p>
					<div class="mt-2 grid grid-cols-2 gap-3">
						<div>
							<label for="emergency-name" class="block text-[11px] text-neutral-500"
								>Contact Name</label
							>
							<input
								id="emergency-name"
								type="text"
								bind:value={emergencyName}
								placeholder="e.g. Priya Malhotra"
								class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
							/>
						</div>
						<div>
							<label for="emergency-phone" class="block text-[11px] text-neutral-500"
								>Contact Phone</label
							>
							<div
								class="mt-1 flex overflow-hidden rounded-xl border border-neutral-300 bg-white transition-colors focus-within:border-neutral-950"
							>
								<span
									class="inline-flex items-center border-r border-neutral-200 bg-neutral-100 px-2.5 text-xs font-semibold text-neutral-600 select-none"
								>
									+
								</span>
								<input
									id="emergency-phone"
									type="tel"
									bind:value={emergencyPhone}
									placeholder="919820123457"
									class="w-full bg-transparent px-2.5 py-1.5 font-mono text-xs text-neutral-900 placeholder-neutral-400 focus:outline-hidden"
								/>
							</div>
						</div>
					</div>
				</div>

				<div class="mt-6 flex items-center justify-end gap-3 border-t border-neutral-100 pt-4">
					<Button variant="ghost" size="sm" onclick={onClose} disabled={isLoading}>Cancel</Button>
					<Button variant="primary" size="sm" type="submit" disabled={isLoading}>
						{#if isLoading}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Registering...</span>
						{:else}
							<span>Register Member</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
