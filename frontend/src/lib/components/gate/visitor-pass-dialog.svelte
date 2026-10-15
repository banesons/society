<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import type { Visit, VisitorPass } from '$lib/api/society';
	import { formatPhoneDisplay } from '$lib/phone';
	import { X, QrCode, Copy, Check, Clock, User, Home } from '@lucide/svelte';

	let {
		isOpen,
		onClose,
		visit,
		pass
	}: {
		isOpen: boolean;
		onClose: () => void;
		visit: Visit | null;
		pass: VisitorPass | null;
	} = $props();

	let copied = $state(false);

	function copyPassCode() {
		if (!pass?.pass_code) return;
		navigator.clipboard.writeText(pass.pass_code);
		copied = true;
		setTimeout(() => {
			copied = false;
		}, 2000);
	}
</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape' && isOpen) onClose();
	}}
/>

{#if isOpen && visit}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-xs"
		role="dialog"
		aria-modal="true"
		onclick={(e) => {
			if (e.target === e.currentTarget) onClose();
		}}
	>
		<div class="w-full max-w-md rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl">
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div class="flex items-center gap-2">
					<div
						class="flex h-8 w-8 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
					>
						<QrCode class="h-4 w-4" />
					</div>
					<div>
						<h3 class="text-sm font-semibold tracking-tight text-neutral-900">
							Visitor Entry Pass
						</h3>
						<p class="text-[11px] text-neutral-500">Authorized society access credentials</p>
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

			<div class="mt-5 space-y-4 text-xs">
				<!-- Pass Code Highlight Card -->
				{#if pass}
					<div
						class="flex flex-col items-center justify-center rounded-xl border border-neutral-200 bg-neutral-50/80 p-5 text-center"
					>
						<span class="text-[11px] font-medium tracking-wider text-neutral-500 uppercase"
							>Pass Code</span
						>
						<div
							class="mt-1 flex items-center gap-2 font-mono text-2xl font-bold tracking-widest text-neutral-950"
						>
							<span>{pass.pass_code}</span>
							<button
								type="button"
								onclick={copyPassCode}
								class="rounded-md p-1 text-neutral-400 hover:bg-neutral-200 hover:text-neutral-700"
								title="Copy Pass Code"
							>
								{#if copied}
									<Check class="h-4 w-4 text-emerald-600" />
								{:else}
									<Copy class="h-4 w-4" />
								{/if}
							</button>
						</div>
						<div class="mt-2 flex items-center gap-1.5 text-[11px] text-neutral-500">
							<Clock class="h-3.5 w-3.5" />
							<span
								>Valid until: {new Date(pass.valid_until).toLocaleTimeString([], {
									hour: '2-digit',
									minute: '2-digit'
								})} ({new Date(pass.valid_until).toLocaleDateString()})</span
							>
						</div>
					</div>
				{/if}

				<!-- Visit Details -->
				<div class="space-y-2.5 rounded-xl border border-neutral-100 bg-white p-3.5">
					<div class="flex items-center justify-between text-neutral-700">
						<span class="flex items-center gap-1.5 text-neutral-500">
							<User class="h-3.5 w-3.5" /> Visitor
						</span>
						<span class="font-medium text-neutral-900">{visit.visitor_name}</span>
					</div>

					<div class="flex items-center justify-between text-neutral-700">
						<span class="text-neutral-500">Phone</span>
						<span class="font-mono text-neutral-900"
							>{formatPhoneDisplay(visit.visitor_phone_masked)}</span
						>
					</div>

					<div class="flex items-center justify-between text-neutral-700">
						<span class="flex items-center gap-1.5 text-neutral-500">
							<Home class="h-3.5 w-3.5" /> Destination Unit
						</span>
						<span class="font-mono font-medium text-neutral-900"
							>{visit.building_name} — {visit.unit_identifier}</span
						>
					</div>

					<div class="flex items-center justify-between text-neutral-700">
						<span class="text-neutral-500">Purpose</span>
						<Badge variant="outline" class="font-mono text-[10px]">{visit.purpose}</Badge>
					</div>

					{#if visit.vehicle_number}
						<div class="flex items-center justify-between text-neutral-700">
							<span class="text-neutral-500">Vehicle</span>
							<span class="font-mono text-neutral-900">{visit.vehicle_number}</span>
						</div>
					{/if}

					<div
						class="flex items-center justify-between border-t border-neutral-100 pt-2 text-neutral-700"
					>
						<span class="text-neutral-500">Status</span>
						<Badge
							variant={visit.status === 'APPROVED' || visit.status === 'CHECKED_IN'
								? 'success'
								: 'default'}
							class="text-[10px]"
						>
							{visit.status}
						</Badge>
					</div>
				</div>
			</div>

			<div class="mt-6 flex justify-end">
				<Button variant="primary" size="sm" onclick={onClose}>Done</Button>
			</div>
		</div>
	</div>
{/if}
