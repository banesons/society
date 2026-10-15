<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import { societyApi, type Visit, type VisitorPass, type Unit } from '$lib/api/society';
	import { X, UserPlus, Calendar, Loader2, AlertCircle } from '@lucide/svelte';

	let {
		societyId,
		isOpen,
		mode = 'walk_in', // 'walk_in' | 'expected'
		onClose,
		onSuccess
	}: {
		societyId: string;
		isOpen: boolean;
		mode?: 'walk_in' | 'expected';
		onClose: () => void;
		onSuccess: (visit: Visit, pass?: VisitorPass) => void;
	} = $props();

	let units = $state<Unit[]>([]);
	let isLoadingUnits = $state(false);

	let unitId = $state('');
	let visitorName = $state('');
	let visitorPhone = $state('');
	let purpose = $state('GUEST');
	let vehicleNumber = $state('');
	let expectedArrival = $state('');

	let isSubmitting = $state(false);
	let error = $state<string | null>(null);

	const purposes = [
		{ value: 'GUEST', label: 'Guest / Visitor' },
		{ value: 'DELIVERY', label: 'Delivery / Courier' },
		{ value: 'CAB', label: 'Cab / Taxi' },
		{ value: 'SERVICE', label: 'Home Service' },
		{ value: 'MAINTENANCE', label: 'Maintenance / Repair' },
		{ value: 'OTHER', label: 'Other' }
	];

	async function loadUnits() {
		if (!societyId) return;
		isLoadingUnits = true;
		try {
			const res = await societyApi.listUnits(societyId, { limit: 200 });
			units = res.units;
			if (units.length > 0 && !unitId) {
				unitId = units[0].id;
			}
		} catch (err: unknown) {
			console.error('Failed to load units', err);
		} finally {
			isLoadingUnits = false;
		}
	}

	$effect(() => {
		if (isOpen) {
			error = null;
			loadUnits();
		}
	});

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!unitId || !visitorName.trim() || !visitorPhone.trim()) {
			error = 'Please fill all required fields.';
			return;
		}

		if (visitorName.trim().length > 60) {
			error = 'Visitor name cannot exceed 60 characters.';
			return;
		}
		const cleanPhone = visitorPhone.trim().replace(/[\s-]/g, '');
		if (cleanPhone.length < 10 || cleanPhone.length > 15) {
			error = 'Mobile phone must be between 10 and 15 digits.';
			return;
		}
		if (vehicleNumber.trim().length > 16) {
			error = 'Vehicle plate cannot exceed 16 characters.';
			return;
		}

		isSubmitting = true;
		error = null;

		try {
			if (mode === 'expected') {
				const res = await societyApi.registerExpectedVisit(societyId, {
					unit_id: unitId,
					visitor_name: visitorName.trim(),
					visitor_phone: visitorPhone.trim(),
					purpose,
					vehicle_number: vehicleNumber.trim() ? vehicleNumber.trim().toUpperCase() : undefined,
					expected_arrival: expectedArrival ? new Date(expectedArrival).toISOString() : undefined
				});
				onSuccess(res.visit, res.pass);
			} else {
				const res = await societyApi.registerWalkInVisit(societyId, {
					unit_id: unitId,
					visitor_name: visitorName.trim(),
					visitor_phone: visitorPhone.trim(),
					purpose,
					vehicle_number: vehicleNumber.trim() ? vehicleNumber.trim().toUpperCase() : undefined
				});
				onSuccess(res.visit);
			}
			onClose();
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to register visitor';
		} finally {
			isSubmitting = false;
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
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-xs"
		role="dialog"
		aria-modal="true"
		onclick={(e) => {
			if (e.target === e.currentTarget) onClose();
		}}
	>
		<div class="w-full max-w-lg rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl">
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div class="flex items-center gap-2">
					<div
						class="flex h-8 w-8 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
					>
						{#if mode === 'expected'}
							<Calendar class="h-4 w-4" />
						{:else}
							<UserPlus class="h-4 w-4" />
						{/if}
					</div>
					<div>
						<h3 class="text-sm font-semibold tracking-tight text-neutral-900">
							{mode === 'expected' ? 'Pre-Register Expected Visitor' : 'Register Walk-In Visitor'}
						</h3>
						<p class="text-[11px] text-neutral-500">
							{mode === 'expected'
								? 'Create scheduled access with verified pass'
								: 'Gate arrival intake and resident notification'}
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={onClose}
					class="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if error}
				<div
					class="mt-4 flex items-center gap-2 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
				>
					<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
					<span>{error}</span>
				</div>
			{/if}

			<form onsubmit={handleSubmit} class="mt-4 space-y-4 text-xs">
				<!-- Destination Unit -->
				<div>
					<label for="reg-unit" class="mb-1 block font-medium text-neutral-700"
						>Destination Flat / Unit *</label
					>
					{#if isLoadingUnits}
						<div
							class="flex h-9 items-center gap-2 rounded-xl border border-neutral-200 bg-neutral-50 px-3 text-neutral-500"
						>
							<Loader2 class="h-3.5 w-3.5 animate-spin" />
							<span>Loading units...</span>
						</div>
					{:else}
						<select
							id="reg-unit"
							bind:value={unitId}
							required
							class="w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-neutral-900 outline-none focus:border-neutral-900"
						>
							{#each units as u (u.id)}
								<option value={u.id}>{u.unit_identifier} (Floor {u.floor_number})</option>
							{/each}
						</select>
					{/if}
				</div>

				<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
					<!-- Visitor Name -->
					<div>
						<label for="reg-name" class="mb-1 block font-medium text-neutral-700"
							>Visitor Full Name *</label
						>
						<input
							id="reg-name"
							type="text"
							bind:value={visitorName}
							placeholder="e.g. Rahul Sharma"
							maxlength="60"
							required
							class="w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-neutral-900 outline-none focus:border-neutral-900"
						/>
					</div>

					<!-- Visitor Phone -->
					<div>
						<label for="reg-phone" class="mb-1 block font-medium text-neutral-700"
							>Mobile Phone *</label
						>
						<input
							id="reg-phone"
							type="tel"
							bind:value={visitorPhone}
							placeholder="e.g. +919820011223"
							maxlength="15"
							required
							class="w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 font-mono text-neutral-900 outline-none focus:border-neutral-900"
						/>
					</div>
				</div>

				<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
					<!-- Purpose -->
					<div>
						<label for="reg-purpose" class="mb-1 block font-medium text-neutral-700"
							>Visit Purpose</label
						>
						<select
							id="reg-purpose"
							bind:value={purpose}
							class="w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-neutral-900 outline-none focus:border-neutral-900"
						>
							{#each purposes as p (p.value)}
								<option value={p.value}>{p.label}</option>
							{/each}
						</select>
					</div>

					<!-- Vehicle Number (Optional) -->
					<div>
						<label for="reg-vehicle" class="mb-1 block font-medium text-neutral-700"
							>Vehicle Plate (Optional)</label
						>
						<input
							id="reg-vehicle"
							type="text"
							bind:value={vehicleNumber}
							placeholder="e.g. MH02AB1234"
							maxlength="16"
							class="w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 font-mono text-neutral-900 uppercase outline-none focus:border-neutral-900"
						/>
					</div>
				</div>

				{#if mode === 'expected'}
					<div>
						<label for="reg-arrival" class="mb-1 block font-medium text-neutral-700"
							>Expected Arrival Time</label
						>
						<input
							id="reg-arrival"
							type="datetime-local"
							bind:value={expectedArrival}
							class="w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 font-mono text-neutral-900 outline-none focus:border-neutral-900"
						/>
					</div>
				{/if}
				<div class="mt-6 flex justify-end gap-2 border-t border-neutral-100 pt-4">
					<Button variant="outline" size="sm" onclick={onClose} disabled={isSubmitting}>
						Cancel
					</Button>
					<Button variant="primary" size="sm" type="submit" disabled={isSubmitting}>
						{#if isSubmitting}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Registering...</span>
						{:else}
							<span>{mode === 'expected' ? 'Create Expected Visit' : 'Register Walk-In'}</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
