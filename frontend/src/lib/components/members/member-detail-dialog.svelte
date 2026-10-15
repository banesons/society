<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import {
		societyApi,
		type SocietyMember,
		type UnitRelationship,
		type UpdateMemberPayload
	} from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { formatFullPhone, normalizePhoneInput } from '$lib/phone';
	import {
		User,
		X,
		AlertCircle,
		Loader2,
		Phone,
		Mail,
		Home,
		KeyRound,
		ShieldAlert,
		Edit2,
		Check,
		UserMinus
	} from '@lucide/svelte';

	let {
		isOpen = false,
		societyId,
		memberId,
		onClose,
		onDepartClick,
		onRefresh
	}: {
		isOpen: boolean;
		societyId: string;
		memberId: string | null;
		onClose: () => void;
		onDepartClick?: (member: SocietyMember) => void;
		onRefresh?: () => void;
	} = $props();

	let member = $state<SocietyMember | null>(null);
	let relationships = $state<UnitRelationship[]>([]);
	let isLoading = $state(false);
	let error = $state<string | null>(null);

	// Edit Mode
	let isEditing = $state(false);
	let editFirstName = $state('');
	let editLastName = $state('');
	let editEmail = $state('');
	let editEmergencyName = $state('');
	let editEmergencyPhone = $state('');
	let editStatus = $state<'ACTIVE' | 'INACTIVE'>('ACTIVE');
	let isSaving = $state(false);

	async function loadDetails() {
		if (!memberId) return;
		isLoading = true;
		error = null;

		try {
			const res = await societyApi.getMember(societyId, memberId);
			member = res.member;
			relationships = res.relationships;

			editFirstName = res.member.first_name;
			editLastName = res.member.last_name || '';
			editEmail = res.member.email || '';
			editEmergencyName = res.member.emergency_contact_name || '';
			editEmergencyPhone = res.member.emergency_contact_phone || '';
			editStatus = res.member.status === 'INACTIVE' ? 'INACTIVE' : 'ACTIVE';
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message || 'Failed to load member profile.';
			} else {
				error = 'Failed to load member.';
			}
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (isOpen && memberId) {
			isEditing = false;
			loadDetails();
		}
	});

	async function handleSaveEdit(e: SubmitEvent) {
		e.preventDefault();
		if (!memberId || !member) return;

		isSaving = true;
		error = null;

		try {
			let cleanEmergencyPhone: string | undefined = undefined;
			if (editEmergencyPhone.trim()) {
				cleanEmergencyPhone = normalizePhoneInput(editEmergencyPhone);
			}

			const payload: UpdateMemberPayload = {
				first_name: editFirstName.trim(),
				last_name: editLastName.trim() || undefined,
				email: editEmail.trim() || undefined,
				emergency_contact_name: editEmergencyName.trim() || undefined,
				emergency_contact_phone: cleanEmergencyPhone,
				status: editStatus
			};

			const res = await societyApi.updateMember(societyId, memberId, payload);
			member = res.member;
			isEditing = false;
			onRefresh?.();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message || 'Failed to update member.';
			} else {
				error = 'Failed to update member profile.';
			}
		} finally {
			isSaving = false;
		}
	}

	function getStatusVariant(status: string): 'default' | 'success' | 'warning' | 'destructive' {
		switch (status) {
			case 'ACTIVE':
				return 'success';
			case 'INACTIVE':
				return 'warning';
			case 'LEFT':
				return 'destructive';
			default:
				return 'default';
		}
	}
</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape' && isOpen) onClose();
	}}
/>

{#if isOpen && memberId}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
		role="dialog"
		aria-modal="true"
		onclick={(e) => {
			if (e.target === e.currentTarget) onClose();
		}}
	>
		<div
			class="relative max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-3xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-2xl transition-all"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div class="flex items-center gap-3">
					<div
						class="flex h-10 w-10 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
					>
						<User class="h-5 w-5" />
					</div>
					<div>
						<h3 class="text-base font-semibold tracking-tight text-neutral-950">
							{#if member}
								{member.first_name} {member.last_name || ''}
							{:else}
								Member Profile
							{/if}
						</h3>
						<p class="text-xs text-neutral-500">Society Member & Tenancy Details</p>
					</div>
				</div>
				<button
					onclick={onClose}
					class="rounded-full p-1.5 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			{#if isLoading}
				<div class="flex min-h-[250px] flex-col items-center justify-center text-neutral-500">
					<Loader2 class="h-8 w-8 animate-spin text-neutral-900" />
					<p class="mt-3 text-xs">Loading member profile & units...</p>
				</div>
			{:else if error}
				<div
					class="mt-4 flex items-start gap-2.5 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
				>
					<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
					<span>{error}</span>
				</div>
			{:else if member}
				<!-- Profile Card -->
				<div class="mt-4 rounded-2xl border border-neutral-200 bg-neutral-50/70 p-4">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2">
							<Badge variant={getStatusVariant(member.status)}>
								{member.status}
							</Badge>
							{#if member.user_id}
								<Badge
									variant="outline"
									class="border-emerald-200 bg-emerald-50 text-[10px] text-emerald-800"
								>
									Verified Account Linked
								</Badge>
							{:else}
								<Badge
									variant="outline"
									class="border-amber-200 bg-amber-50 text-[10px] text-amber-800"
								>
									Pre-registered (Unverified)
								</Badge>
							{/if}
						</div>

						<div class="flex items-center gap-2">
							{#if !isEditing && member.status !== 'LEFT'}
								<Button variant="ghost" size="sm" onclick={() => (isEditing = true)}>
									<Edit2 class="mr-1.5 h-3.5 w-3.5" />
									<span>Edit</span>
								</Button>
								<Button
									variant="outline"
									size="sm"
									class="border-amber-300 text-amber-800 hover:bg-amber-100"
									onclick={() => member && onDepartClick?.(member)}
								>
									<UserMinus class="mr-1.5 h-3.5 w-3.5" />
									<span>Record Departure</span>
								</Button>
							{/if}
						</div>
					</div>

					{#if isEditing}
						<form onsubmit={handleSaveEdit} class="mt-4 space-y-3">
							<div class="grid grid-cols-2 gap-3">
								<div>
									<label for="edit-fn" class="block text-[11px] font-medium text-neutral-700"
										>First Name *</label
									>
									<input
										id="edit-fn"
										type="text"
										required
										bind:value={editFirstName}
										class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-hidden"
									/>
								</div>
								<div>
									<label for="edit-ln" class="block text-[11px] font-medium text-neutral-700"
										>Last Name</label
									>
									<input
										id="edit-ln"
										type="text"
										bind:value={editLastName}
										class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-hidden"
									/>
								</div>
							</div>

							<div class="grid grid-cols-2 gap-3">
								<div>
									<label for="edit-em" class="block text-[11px] font-medium text-neutral-700"
										>Email</label
									>
									<input
										id="edit-em"
										type="email"
										bind:value={editEmail}
										class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-hidden"
									/>
								</div>
								<div>
									<label for="edit-st" class="block text-[11px] font-medium text-neutral-700"
										>Status</label
									>
									<select
										id="edit-st"
										bind:value={editStatus}
										class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-hidden"
									>
										<option value="ACTIVE">ACTIVE</option>
										<option value="INACTIVE">INACTIVE</option>
									</select>
								</div>
							</div>

							<div class="grid grid-cols-2 gap-3">
								<div>
									<label for="edit-ecn" class="block text-[11px] font-medium text-neutral-700"
										>Emergency Contact Name</label
									>
									<input
										id="edit-ecn"
										type="text"
										bind:value={editEmergencyName}
										class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-hidden"
									/>
								</div>
								<div>
									<label for="edit-ecp" class="block text-[11px] font-medium text-neutral-700"
										>Emergency Contact Phone</label
									>
									<div
										class="mt-1 flex overflow-hidden rounded-xl border border-neutral-300 bg-white focus-within:border-neutral-950"
									>
										<span
											class="inline-flex items-center border-r border-neutral-200 bg-neutral-100 px-2 text-xs font-semibold text-neutral-600 select-none"
										>
											+
										</span>
										<input
											id="edit-ecp"
											type="tel"
											bind:value={editEmergencyPhone}
											placeholder="919820123457"
											class="w-full bg-transparent px-2.5 py-1.5 font-mono text-xs text-neutral-900 placeholder-neutral-400 focus:outline-hidden"
										/>
									</div>
								</div>
							</div>

							<div class="flex items-center justify-end gap-2 border-t border-neutral-200 pt-2">
								<Button
									variant="ghost"
									size="sm"
									onclick={() => (isEditing = false)}
									disabled={isSaving}
								>
									Cancel
								</Button>
								<Button variant="primary" size="sm" type="submit" disabled={isSaving}>
									{#if isSaving}
										<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
										<span>Saving...</span>
									{:else}
										<Check class="mr-1.5 h-3.5 w-3.5" />
										<span>Save Changes</span>
									{/if}
								</Button>
							</div>
						</form>
					{:else}
						<div class="mt-4 grid grid-cols-2 gap-4 text-xs text-neutral-700">
							<div class="flex items-center gap-2">
								<Phone class="h-3.5 w-3.5 text-neutral-500" />
								<span class="text-neutral-500">Phone:</span>
								<span class="font-mono font-medium text-neutral-900"
									>{formatFullPhone(member.phone || member.phone_masked)}</span
								>
							</div>
							<div class="flex items-center gap-2">
								<Mail class="h-3.5 w-3.5 text-neutral-500" />
								<span class="text-neutral-500">Email:</span>
								<span class="font-medium text-neutral-900">{member.email || '—'}</span>
							</div>
							<div class="flex items-center gap-2">
								<ShieldAlert class="h-3.5 w-3.5 text-amber-600" />
								<span class="text-neutral-500">Emergency:</span>
								<span class="font-medium text-neutral-900">
									{member.emergency_contact_name
										? `${member.emergency_contact_name} (${formatFullPhone(member.emergency_contact_phone)})`
										: '—'}
								</span>
							</div>
							<div class="flex items-center gap-2">
								<span class="text-neutral-500">Member Since:</span>
								<span class="font-mono font-medium text-neutral-900"
									>{new Date(member.created_at).toLocaleDateString()}</span
								>
							</div>
						</div>
					{/if}
				</div>

				<!-- Unit Relationships Section -->
				<div class="mt-6">
					<div class="flex items-center justify-between">
						<h4 class="text-xs font-semibold tracking-wider text-neutral-500 uppercase">
							Unit Relationships & Tenancies ({relationships.length})
						</h4>
					</div>

					{#if relationships.length === 0}
						<div
							class="mt-3 rounded-2xl border border-dashed border-neutral-200 bg-neutral-50/50 p-6 text-center text-xs text-neutral-500"
						>
							No unit relationships linked to this member yet.
						</div>
					{:else}
						<div class="mt-3 space-y-2">
							{#each relationships as rel (rel.id)}
								<div
									class="flex flex-col gap-2 rounded-2xl border border-neutral-200 bg-neutral-50/60 p-3 sm:flex-row sm:items-center sm:justify-between"
								>
									<div class="flex items-center gap-3">
										<div
											class="flex h-9 w-9 items-center justify-center rounded-full {rel.relationship_type.includes(
												'OWNER'
											)
												? 'bg-emerald-100 text-emerald-800'
												: 'bg-neutral-100 text-neutral-800'}"
										>
											{#if rel.relationship_type.includes('OWNER')}
												<Home class="h-4 w-4" />
											{:else}
												<KeyRound class="h-4 w-4" />
											{/if}
										</div>
										<div>
											<div class="flex items-center gap-2">
												<span class="text-xs font-semibold text-neutral-950"
													>{rel.unit_identifier || 'Unit'}</span
												>
												<Badge variant="outline" class="bg-white text-[10px]">
													{rel.relationship_type}
												</Badge>
												{#if rel.ownership_share != null}
													<span class="font-mono text-[10px] font-semibold text-emerald-700">
														{rel.ownership_share}% Share
													</span>
												{/if}
											</div>
											<div class="mt-0.5 flex items-center gap-2 text-[11px] text-neutral-500">
												<span
													>{rel.is_occupant ? 'Resident Occupant' : 'Absentee / Non-occupant'}</span
												>
												<span class="text-neutral-300">·</span>
												<span>Since {rel.start_date || '—'}</span>
												{#if rel.end_date}
													<span class="text-neutral-300">·</span>
													<span class="text-rose-600">Ended {rel.end_date}</span>
												{/if}
											</div>
										</div>
									</div>

									<div class="flex items-center gap-2">
										<Badge
											variant={rel.status === 'ACTIVE' ? 'success' : 'destructive'}
											class="text-[10px]"
										>
											{rel.status}
										</Badge>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/if}

			<div class="mt-6 flex items-center justify-end border-t border-neutral-100 pt-4">
				<Button variant="ghost" size="sm" onclick={onClose}>Close</Button>
			</div>
		</div>
	</div>
{/if}
