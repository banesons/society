<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { societyApi, type SocietyAdminAssignment } from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import {
		UserPlus,
		ShieldAlert,
		Trash2,
		Loader2,
		AlertCircle,
		CheckCircle2,
		Phone,
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

	function validatePhone(input: string): boolean {
		const digits = input.replace(/\D/g, '');
		if (digits.length === 10 && /^[6-9]/.test(digits)) return true;
		if (digits.length === 12 && digits.startsWith('91') && /^[6-9]/.test(digits.slice(2)))
			return true;
		if (digits.length === 11 && digits.startsWith('0') && /^[6-9]/.test(digits.slice(1)))
			return true;
		return false;
	}

	async function handleAssignSubmit(e: SubmitEvent) {
		e.preventDefault();
		assignError = null;
		assignSuccess = null;

		const trimmed = assignPhone.trim();
		if (!trimmed) {
			assignError = 'Please enter a valid mobile number.';
			return;
		}

		if (!validatePhone(trimmed)) {
			assignError = 'Please enter a valid 10-digit Indian mobile number.';
			return;
		}

		isAssigning = true;
		try {
			await societyApi.assignSocietyAdmin(societyId, {
				phone: trimmed,
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

<div class="space-y-4">
	<!-- Actions & Filter Bar -->
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div class="flex items-center gap-2">
			<label class="flex cursor-pointer items-center gap-2 text-xs text-slate-300">
				<input
					type="checkbox"
					bind:checked={showRevoked}
					class="rounded border-slate-700 bg-slate-800 text-indigo-600 focus:ring-indigo-500"
				/>
				<span>Include Revoked History</span>
			</label>
			<button
				type="button"
				onclick={loadAdmins}
				class="rounded p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
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
			class="flex items-start gap-2.5 rounded-xl border border-rose-500/30 bg-rose-500/10 p-3 text-xs text-rose-300"
		>
			<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-400" />
			<span>{error}</span>
		</div>
	{/if}

	<!-- Roster List -->
	{#if isLoading}
		<div class="flex flex-col items-center justify-center py-10 text-slate-400">
			<Loader2 class="h-6 w-6 animate-spin text-indigo-500" />
			<span class="mt-2 text-xs">Loading society administrators...</span>
		</div>
	{:else if admins.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-xl border border-slate-800/80 bg-slate-950/40 py-10 text-center"
		>
			<ShieldAlert class="h-8 w-8 text-slate-600" />
			<p class="mt-2 text-xs font-medium text-slate-300">No Administrators Assigned</p>
			<p class="mt-1 text-[11px] text-slate-500">
				Assign an existing user by mobile phone to administer this society.
			</p>
		</div>
	{:else}
		<div
			class="divide-y divide-slate-800/80 overflow-hidden rounded-xl border border-slate-800 bg-slate-950/50"
		>
			{#each admins as admin (admin.id)}
				<div class="flex items-center justify-between p-3 transition-colors hover:bg-slate-900/60">
					<div class="flex items-center gap-3">
						<div
							class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-500/10 font-mono text-xs font-semibold text-indigo-400"
						>
							SA
						</div>
						<div>
							<div class="flex items-center gap-2">
								<span class="font-mono text-xs font-medium text-slate-200">
									{admin.user_phone_masked}
								</span>
								<Badge variant={admin.status === 'ACTIVE' ? 'success' : 'destructive'}>
									{admin.status}
								</Badge>
								<Badge variant="outline" class="text-[10px] text-slate-400">
									{admin.role}
								</Badge>
							</div>
							<div class="mt-0.5 flex items-center gap-2 text-[11px] text-slate-400">
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
								class="text-rose-400 hover:bg-rose-500/10 hover:text-rose-300"
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
			class="fixed inset-0 bg-slate-950/80 backdrop-blur-sm transition-opacity"
			onclick={() => {
				if (!isAssigning) isAssignOpen = false;
			}}
			aria-label="Close dialog overlay"
		></button>

		<div
			class="relative w-full max-w-md rounded-2xl border border-slate-800 bg-slate-900 p-6 shadow-2xl"
		>
			<div class="flex items-center gap-2.5 border-b border-slate-800/80 pb-4">
				<div
					class="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-500/10 text-indigo-400"
				>
					<UserPlus class="h-5 w-5" />
				</div>
				<div>
					<h3 id="assign-admin-title" class="text-base font-semibold text-slate-100">
						Assign Society Administrator
					</h3>
					<p class="text-xs text-slate-400">Target user must already be registered via OTP.</p>
				</div>
			</div>

			<form onsubmit={handleAssignSubmit} class="mt-4 space-y-4">
				{#if assignError}
					<div
						class="flex items-start gap-2 rounded-xl border border-rose-500/30 bg-rose-500/10 p-3 text-xs text-rose-300"
					>
						<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-400" />
						<span>{assignError}</span>
					</div>
				{/if}

				{#if assignSuccess}
					<div
						class="flex items-start gap-2 rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-3 text-xs text-emerald-300"
					>
						<CheckCircle2 class="mt-0.5 h-4 w-4 shrink-0 text-emerald-400" />
						<span>{assignSuccess}</span>
					</div>
				{/if}

				<div>
					<label for="assign-phone-input" class="block text-xs font-medium text-slate-300">
						Mobile Phone Number
					</label>
					<div class="relative mt-1">
						<div
							class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-slate-500"
						>
							<Phone class="h-4 w-4" />
						</div>
						<input
							id="assign-phone-input"
							type="tel"
							bind:value={assignPhone}
							placeholder="+91 98201 23456"
							disabled={isAssigning}
							required
							class="w-full rounded-xl border border-slate-800 bg-slate-950/60 py-2.5 pr-4 pl-9 font-mono text-xs text-slate-100 placeholder-slate-500 transition-colors focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 focus:outline-none disabled:opacity-50"
						/>
					</div>
					<p class="mt-1 text-[11px] text-slate-500">
						Accepts standard 10-digit mobile or +91 format.
					</p>
				</div>

				<div class="rounded-xl border border-slate-800 bg-slate-950/40 p-3">
					<div class="flex items-center justify-between text-xs">
						<span class="text-slate-400">Assigned Role:</span>
						<Badge variant="outline" class="font-mono text-indigo-300">SOC_ADMIN</Badge>
					</div>
					<p class="mt-1.5 text-[11px] leading-relaxed text-slate-500">
						Society Administrator holds permissions strictly inside this society tenant. Platform
						administration roles are not affected.
					</p>
				</div>

				<div class="flex items-center justify-end gap-2 border-t border-slate-800/80 pt-2">
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
			class="fixed inset-0 bg-slate-950/80 backdrop-blur-sm transition-opacity"
			onclick={() => {
				if (!isRevoking) adminToRevoke = null;
			}}
			aria-label="Close dialog overlay"
		></button>

		<div
			class="relative w-full max-w-md rounded-2xl border border-slate-800 bg-slate-900 p-6 shadow-2xl"
		>
			<div class="flex items-center gap-2.5 border-b border-slate-800/80 pb-4">
				<div
					class="flex h-9 w-9 items-center justify-center rounded-lg bg-rose-500/10 text-rose-400"
				>
					<ShieldAlert class="h-5 w-5" />
				</div>
				<div>
					<h3 id="revoke-modal-title" class="text-base font-semibold text-slate-100">
						Revoke Society Administrator
					</h3>
					<p class="text-xs text-slate-400">Immediate access revocation.</p>
				</div>
			</div>

			<div class="mt-4 space-y-3">
				{#if revokeError}
					<div
						class="flex items-start gap-2 rounded-xl border border-rose-500/30 bg-rose-500/10 p-3 text-xs text-rose-300"
					>
						<AlertCircle class="mt-0.5 h-4 w-4 shrink-0 text-rose-400" />
						<span>{revokeError}</span>
					</div>
				{/if}

				<p class="text-xs leading-relaxed text-slate-300">
					Are you sure you want to revoke <strong class="font-mono text-slate-100"
						>{adminToRevoke.user_phone_masked}</strong
					> as administrator for this society?
				</p>

				<div
					class="rounded-xl border border-rose-500/20 bg-rose-500/5 p-3 text-xs leading-relaxed text-rose-300/90"
				>
					Revocation takes effect immediately without waiting for token expiry. Historical
					assignment records will be preserved.
				</div>
			</div>

			<div class="mt-6 flex items-center justify-end gap-2 border-t border-slate-800/80 pt-3">
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
					variant="primary"
					class="bg-rose-600 text-white hover:bg-rose-700"
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
