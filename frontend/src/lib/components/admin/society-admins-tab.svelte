<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { societyApi, type SocietyAdminAssignment } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import { formatFullPhone, normalizePhoneInput, isValidIndianPhone } from '$lib/phone';
	import {
		UserPlus,
		ShieldAlert,
		Trash2,
		Loader2,
		AlertCircle,
		CheckCircle2,
		RefreshCw
	} from '@lucide/svelte';

	let {
		societyId
	}: {
		societyId: string;
	} = $props();

	let admins = $state<SocietyAdminAssignment[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let showRevoked = $state(false);

	// Assign modal state
	let isAssignOpen = $state(false);
	let assignPhone = $state('');
	let isAssigning = $state(false);
	let assignError = $state<string | null>(null);
	let assignSuccess = $state<string | null>(null);

	// Revoke confirmation modal state
	let adminToRevoke = $state<SocietyAdminAssignment | null>(null);
	let isRevoking = $state(false);
	let revokeError = $state<string | null>(null);

	async function loadAdmins() {
		isLoading = true;
		error = null;
		try {
			const res = await societyApi.listSocietyAdmins(societyId, {
				status: showRevoked ? '' : 'ACTIVE'
			});
			admins = res.items;
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message;
			} else {
				error = 'Failed to load society administrators.';
			}
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		// Re-fetch when showRevoked changes
		if (showRevoked !== undefined && societyId) {
			loadAdmins();
		}
	});

	async function handleAssignSubmit(e: SubmitEvent) {
		e.preventDefault();
		assignError = null;
		assignSuccess = null;

		const cleanPhone = normalizePhoneInput(assignPhone);
		if (!cleanPhone) {
			assignError = 'Please enter a valid mobile number.';
			return;
		}

		if (!isValidIndianPhone(cleanPhone)) {
			assignError = 'Please enter a valid Indian mobile number (e.g. 919820123456).';
			return;
		}

		isAssigning = true;
		try {
			await societyApi.assignSocietyAdmin(societyId, {
				phone: cleanPhone,
				role: 'SOC_ADMIN'
			});
			assignSuccess = 'Society administrator successfully assigned.';
			assignPhone = '';
			await loadAdmins();
			setTimeout(() => {
				assignSuccess = null;
				isAssignOpen = false;
			}, 1200);
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.code === 'USER_NOT_FOUND') {
					assignError =
						'No registered user found with this phone number. The user must register via OTP first.';
				} else if (
					err.code === 'SOCIETY_ADMIN_ALREADY_ASSIGNED' ||
					err.code === 'ALREADY_ASSIGNED'
				) {
					assignError = 'This user is already an active administrator for this society.';
				} else {
					assignError = err.message;
				}
			} else {
				assignError = 'Failed to assign society administrator.';
			}
		} finally {
			isAssigning = false;
		}
	}

	async function confirmRevoke() {
		if (!adminToRevoke) return;
		isRevoking = true;
		revokeError = null;

		try {
			await societyApi.revokeSocietyAdmin(societyId, adminToRevoke.id);
			adminToRevoke = null;
			await loadAdmins();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				revokeError = err.message;
			} else {
				revokeError = 'Failed to revoke administrator assignment.';
			}
		} finally {
			isRevoking = false;
		}
	}

	function formatDate(iso: string) {
		try {
			return new Date(iso).toLocaleDateString('en-IN', {
				year: 'numeric',
				month: 'short',
				day: 'numeric'
			});
		} catch {
			return iso;
		}
	}
</script>

<div class="space-y-4 text-neutral-900">
	<!-- Actions & Filter Bar -->
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div class="flex items-center gap-2">
			<label class="flex cursor-pointer items-center gap-2 text-xs text-neutral-600">
				<input
					type="checkbox"
					bind:checked={showRevoked}
					class="rounded border-neutral-300 text-neutral-900 focus:ring-neutral-950"
				/>
				<span>Include Revoked History</span>
			</label>
			<button
				type="button"
				onclick={loadAdmins}
				class="rounded-full p-1.5 text-neutral-500 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
				title="Refresh roster"
				aria-label="Refresh administrators roster"
			>
				<RefreshCw class="h-3.5 w-3.5" />
			</button>
		</div>

		<Button
			variant="primary"
			size="sm"
			onclick={() => {
				assignError = null;
				assignSuccess = null;
				assignPhone = '';
				isAssignOpen = true;
			}}
		>
			<UserPlus class="mr-1.5 h-3.5 w-3.5" />
			<span>Assign Admin</span>
		</Button>
	</div>

	<!-- Error Alert -->
	{#if error}
		<div
			class="flex items-start gap-2.5 rounded-2xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
		>
			<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
			<span>{error}</span>
		</div>
	{/if}

	<!-- Roster List -->
	{#if isLoading}
		<div class="flex flex-col items-center justify-center py-10 text-neutral-500">
			<Loader2 class="h-6 w-6 animate-spin text-neutral-900" />
			<span class="mt-2 text-xs">Loading society administrators...</span>
		</div>
	{:else if admins.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-2xl border border-neutral-200 bg-neutral-50/60 py-10 text-center"
		>
			<ShieldAlert class="h-8 w-8 text-neutral-400" />
			<p class="mt-2 text-xs font-semibold text-neutral-800">No Administrators Assigned</p>
			<p class="mt-1 text-[11px] text-neutral-500">
				Assign an existing user by mobile phone to administer this society.
			</p>
		</div>
	{:else}
		<div
			class="divide-y divide-neutral-100 overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-xs"
		>
			{#each admins as admin (admin.id)}
				<div
					class="flex items-center justify-between p-3.5 transition-colors hover:bg-neutral-50/60"
				>
					<div class="flex items-center gap-3">
						<div
							class="flex h-8 w-8 items-center justify-center rounded-full bg-neutral-100 font-mono text-xs font-bold text-neutral-800"
						>
							SA
						</div>
						<div>
							<div class="flex items-center gap-2">
								<span class="font-mono text-xs font-semibold text-neutral-900">
									{formatFullPhone(admin.user_phone_masked)}
								</span>
								<Badge variant={admin.status === 'ACTIVE' ? 'success' : 'destructive'}>
									{admin.status}
								</Badge>
								<Badge variant="outline" class="text-[10px] text-neutral-600">
									{admin.role}
								</Badge>
							</div>
							<div class="mt-0.5 flex items-center gap-2 text-[11px] text-neutral-500">
								<span>Assigned: {formatDate(admin.created_at)}</span>
								{#if admin.status === 'REVOKED' && admin.revoked_at}
									<span>· Revoked: {formatDate(admin.revoked_at)}</span>
								{/if}
							</div>
						</div>
					</div>

					<div>
						{#if admin.status === 'ACTIVE'}
							<Button
								variant="ghost"
								size="sm"
								class="text-rose-600 hover:bg-rose-50 hover:text-rose-700"
								onclick={() => {
									revokeError = null;
									adminToRevoke = admin;
								}}
							>
								<Trash2 class="mr-1 h-3.5 w-3.5" />
								<span>Revoke</span>
							</Button>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Assign Administrator Modal -->
{#if isAssignOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6"
		role="dialog"
		aria-labelledby="assign-admin-title"
		aria-modal="true"
	>
		<button
			type="button"
			tabindex="-1"
			class="fixed inset-0 bg-neutral-950/40 backdrop-blur-xs transition-opacity"
			onclick={() => {
				if (!isAssigning) isAssignOpen = false;
			}}
			aria-label="Close dialog overlay"
		></button>

		<div
			class="relative w-full max-w-md rounded-3xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-2xl"
		>
			<div class="flex items-center gap-2.5 border-b border-neutral-100 pb-4">
				<div
					class="flex h-9 w-9 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
				>
					<UserPlus class="h-4 w-4" />
				</div>
				<div>
					<h3
						id="assign-admin-title"
						class="text-base font-semibold tracking-tight text-neutral-900"
					>
						Assign Society Administrator
					</h3>
					<p class="text-xs text-neutral-500">Target user must already be registered via OTP.</p>
				</div>
			</div>

			<form onsubmit={handleAssignSubmit} class="mt-4 space-y-4">
				{#if assignError}
					<div
						class="flex items-start gap-2 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
					>
						<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
						<span>{assignError}</span>
					</div>
				{/if}

				{#if assignSuccess}
					<div
						class="flex items-start gap-2 rounded-xl border border-emerald-200 bg-emerald-50 p-3 text-xs text-emerald-800"
					>
						<CheckCircle2 class="mt-0.5 h-4 w-4 shrink-0 text-emerald-600" />
						<span>{assignSuccess}</span>
					</div>
				{/if}

				<div>
					<label for="assign-phone-input" class="block text-xs font-medium text-neutral-700">
						Mobile Phone Number
					</label>
					<div
						class="mt-1 flex overflow-hidden rounded-xl border border-neutral-300 bg-white transition-colors focus-within:border-neutral-950"
					>
						<span
							class="inline-flex items-center border-r border-neutral-200 bg-neutral-100 px-3 text-xs font-semibold text-neutral-600 select-none"
						>
							+
						</span>
						<input
							id="assign-phone-input"
							type="tel"
							bind:value={assignPhone}
							placeholder="919820123456"
							disabled={isAssigning}
							required
							class="w-full bg-transparent px-3 py-2.5 font-mono text-xs text-neutral-900 placeholder-neutral-400 focus:outline-hidden disabled:opacity-50"
						/>
					</div>
					<p class="mt-1 text-[11px] text-neutral-500">Indian mobile number (e.g. 919820123456).</p>
				</div>

				<div class="rounded-2xl border border-neutral-200 bg-neutral-50/60 p-3.5">
					<div class="flex items-center justify-between text-xs">
						<span class="text-neutral-600">Assigned Role:</span>
						<Badge variant="outline" class="bg-neutral-100 font-mono text-neutral-900"
							>SOC_ADMIN</Badge
						>
					</div>
					<p class="mt-1.5 text-[11px] leading-relaxed text-neutral-500">
						Society Administrator holds permissions strictly inside this society tenant. Platform
						administration roles are not affected.
					</p>
				</div>

				<div class="flex items-center justify-end gap-2 border-t border-neutral-100 pt-3">
					<Button
						type="button"
						variant="ghost"
						size="sm"
						disabled={isAssigning}
						onclick={() => (isAssignOpen = false)}
					>
						Cancel
					</Button>
					<Button type="submit" variant="primary" size="sm" disabled={isAssigning}>
						{#if isAssigning}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Assigning...</span>
						{:else}
							<span>Assign SOC_ADMIN</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- Revoke Confirmation Modal -->
{#if adminToRevoke}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6"
		role="dialog"
		aria-labelledby="revoke-modal-title"
		aria-modal="true"
	>
		<button
			type="button"
			tabindex="-1"
			class="fixed inset-0 bg-neutral-950/40 backdrop-blur-xs transition-opacity"
			onclick={() => {
				if (!isRevoking) adminToRevoke = null;
			}}
			aria-label="Close dialog overlay"
		></button>

		<div
			class="relative w-full max-w-md rounded-3xl border border-neutral-200 bg-white p-6 text-neutral-900 shadow-2xl"
		>
			<div class="flex items-center gap-2.5 border-b border-neutral-100 pb-4">
				<div class="flex h-9 w-9 items-center justify-center rounded-full bg-rose-50 text-rose-600">
					<ShieldAlert class="h-4 w-4" />
				</div>
				<div>
					<h3
						id="revoke-modal-title"
						class="text-base font-semibold tracking-tight text-neutral-900"
					>
						Revoke Society Administrator
					</h3>
					<p class="text-xs text-neutral-500">Immediate access revocation.</p>
				</div>
			</div>

			<div class="mt-4 space-y-3">
				{#if revokeError}
					<div
						class="flex items-start gap-2 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
					>
						<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
						<span>{revokeError}</span>
					</div>
				{/if}

				<p class="text-xs leading-relaxed text-neutral-700">
					Are you sure you want to revoke <strong class="font-mono text-neutral-900"
						>{formatFullPhone(adminToRevoke.user_phone_masked)}</strong
					> as administrator for this society?
				</p>

				<div
					class="rounded-2xl border border-rose-200 bg-rose-50/60 p-3.5 text-xs leading-relaxed text-rose-900"
				>
					Revocation takes effect immediately without waiting for token expiry. Historical
					assignment records will be preserved.
				</div>
			</div>

			<div class="mt-6 flex items-center justify-end gap-2 border-t border-neutral-100 pt-3">
				<Button
					type="button"
					variant="ghost"
					size="sm"
					disabled={isRevoking}
					onclick={() => (adminToRevoke = null)}
				>
					Cancel
				</Button>
				<Button
					type="button"
					variant="destructive"
					size="sm"
					disabled={isRevoking}
					onclick={confirmRevoke}
				>
					{#if isRevoking}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
						<span>Revoking...</span>
					{:else}
						<span>Confirm Revocation</span>
					{/if}
				</Button>
			</div>
		</div>
	</div>
{/if}
