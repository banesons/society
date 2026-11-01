<script lang="ts">
	import ResponsiveModal from '$lib/components/ui/responsive-modal.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import { societyApi, type Gate, type GateQR } from '$lib/api/society';
	import {
		QrCode,
		RefreshCw,
		Printer,
		Trash2,
		Check,
		Copy,
		ExternalLink,
		Loader2,
		ShieldCheck
	} from '@lucide/svelte';

	let {
		societyId,
		gate,
		open = $bindable(false),
		onClose
	}: {
		societyId: string;
		gate?: Gate | null;
		open: boolean;
		onClose?: () => void;
	} = $props();

	let qrs = $state<GateQR[]>([]);
	let activeQR = $derived(qrs.find((q) => q.status === 'ACTIVE') || null);
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	let isGenerating = $state(false);
	let isRotating = $state(false);
	let isRevoking = $state(false);
	let copied = $state(false);

	let visitUrl = $derived.by(() => {
		if (!activeQR) return '';
		if (typeof window !== 'undefined') {
			return `${window.location.origin}/visit/${activeQR.token}`;
		}
		return `https://society.banesons.com/visit/${activeQR.token}`;
	});

	async function loadQRs() {
		if (!societyId) return;
		isLoading = true;
		error = null;
		try {
			const res = await societyApi.listGateQRs(societyId, gate?.id);
			qrs = res.gate_qrs;
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load QR configurations';
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (open && societyId) {
			loadQRs();
		}
	});

	async function handleGenerateQR() {
		isGenerating = true;
		try {
			const label = gate ? `${gate.name} Visitor QR` : 'Society Entrance QR';
			await societyApi.createGateQR(societyId, {
				gate_id: gate?.id,
				label
			});
			await loadQRs();
		} catch (err: unknown) {
			alert(err instanceof Error ? err.message : 'Failed to generate QR code');
		} finally {
			isGenerating = false;
		}
	}

	async function handleRotateQR() {
		if (!activeQR) return;
		if (
			!confirm(
				'Rotating this QR code will immediately invalidate the previous printed QR. Continue?'
			)
		) {
			return;
		}
		isRotating = true;
		try {
			await societyApi.rotateGateQR(societyId, activeQR.id);
			await loadQRs();
		} catch (err: unknown) {
			alert(err instanceof Error ? err.message : 'Failed to rotate QR code');
		} finally {
			isRotating = false;
		}
	}

	async function handleRevokeQR() {
		if (!activeQR) return;
		if (
			!confirm(
				'Are you sure you want to revoke this QR code? Visitors will no longer be able to register through it.'
			)
		) {
			return;
		}
		isRevoking = true;
		try {
			await societyApi.revokeGateQR(societyId, activeQR.id);
			await loadQRs();
		} catch (err: unknown) {
			alert(err instanceof Error ? err.message : 'Failed to revoke QR code');
		} finally {
			isRevoking = false;
		}
	}

	function handleCopyLink() {
		if (!visitUrl) return;
		navigator.clipboard.writeText(visitUrl);
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}

	function handlePrint() {
		window.print();
	}
</script>

<ResponsiveModal
	isOpen={open}
	title={gate ? `Visitor QR — ${gate.name}` : 'Society Visitor Registration QR'}
	description="Physical entrance QR configuration for visitor self-registration"
	onClose={() => {
		open = false;
		if (onClose) onClose();
	}}
>
	<div class="space-y-6">
		{#if error}
			<div class="rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800">
				{error}
			</div>
		{/if}

		{#if isLoading}
			<div class="p-8 text-center text-xs text-neutral-500">
				<Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin text-neutral-400" />
				Loading QR configuration...
			</div>
		{:else if !activeQR}
			<!-- No active QR -> generate button -->
			<div
				class="space-y-3 rounded-2xl border border-dashed border-[#e2e2de] bg-neutral-50 p-8 text-center"
			>
				<QrCode class="mx-auto h-10 w-10 text-neutral-400" />
				<div>
					<h4 class="text-xs font-semibold text-neutral-800">No Active QR Code</h4>
					<p class="mt-0.5 text-[11px] text-neutral-500">
						Generate a secure, opaque QR code for visitors to scan at {gate?.name ||
							'this entrance'}.
					</p>
				</div>
				<Button
					size="sm"
					class="bg-[#121212] text-xs text-white"
					onclick={handleGenerateQR}
					disabled={isGenerating}
				>
					{#if isGenerating}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					{/if}
					Generate Entrance QR
				</Button>
			</div>
		{:else}
			<!-- Active QR Display & Actions -->
			<div class="space-y-5">
				<!-- High Contrast Printable Card -->
				<div
					class="space-y-4 rounded-2xl border border-[#e2e2de] bg-white p-6 text-center shadow-sm"
					id="printable-qr"
				>
					<div class="flex items-center justify-between border-b border-[#e2e2de] pb-3">
						<span class="flex items-center gap-1.5 text-xs font-semibold text-neutral-800">
							<ShieldCheck class="h-4 w-4 text-emerald-600" />
							Verified Entrance Pass
						</span>
						<Badge
							variant="outline"
							class="border-emerald-200 bg-emerald-50 text-[10px] text-emerald-800"
						>
							Active QR
						</Badge>
					</div>

					<!-- QR Visual / Placeholder using SVG -->
					<div class="mx-auto inline-block rounded-2xl border border-[#e2e2de] bg-neutral-50 p-4">
						<svg
							class="mx-auto h-44 w-44 text-neutral-900"
							viewBox="0 0 100 100"
							fill="currentColor"
						>
							<!-- Stylized High-Density QR code pattern -->
							<rect x="0" y="0" width="30" height="30" rx="3" fill="#121212" />
							<rect x="5" y="5" width="20" height="20" rx="2" fill="#ffffff" />
							<rect x="10" y="10" width="10" height="10" rx="1" fill="#121212" />

							<rect x="70" y="0" width="30" height="30" rx="3" fill="#121212" />
							<rect x="75" y="5" width="20" height="20" rx="2" fill="#ffffff" />
							<rect x="80" y="10" width="10" height="10" rx="1" fill="#121212" />

							<rect x="0" y="70" width="30" height="30" rx="3" fill="#121212" />
							<rect x="5" y="75" width="20" height="20" rx="2" fill="#ffffff" />
							<rect x="10" y="80" width="10" height="10" rx="1" fill="#121212" />

							<!-- Matrix dots -->
							<rect x="36" y="8" width="6" height="6" fill="#121212" />
							<rect x="48" y="12" width="6" height="6" fill="#121212" />
							<rect x="56" y="24" width="6" height="6" fill="#121212" />
							<rect x="36" y="36" width="8" height="8" rx="1" fill="#121212" />
							<rect x="48" y="48" width="8" height="8" rx="1" fill="#121212" />
							<rect x="64" y="40" width="6" height="6" fill="#121212" />
							<rect x="12" y="44" width="6" height="6" fill="#121212" />
							<rect x="24" y="52" width="6" height="6" fill="#121212" />
							<rect x="40" y="68" width="6" height="6" fill="#121212" />
							<rect x="52" y="76" width="6" height="6" fill="#121212" />
							<rect x="68" y="68" width="8" height="8" rx="1" fill="#121212" />
							<rect x="84" y="76" width="6" height="6" fill="#121212" />
							<rect x="76" y="88" width="6" height="6" fill="#121212" />
						</svg>
					</div>

					<div>
						<h3 class="text-sm font-bold tracking-tight text-neutral-900">
							{gate ? gate.name : 'Society Entrance'}
						</h3>
						<p class="mt-0.5 text-[11px] text-neutral-500">
							Scan with mobile camera to register for visitor clearance
						</p>
					</div>

					<!-- Public URL Link -->
					<div
						class="flex items-center gap-2 rounded-xl border border-[#e2e2de] bg-neutral-50 p-2 font-mono text-[11px] text-neutral-600"
					>
						<span class="flex-1 truncate text-left">{visitUrl}</span>
						<Button variant="ghost" size="sm" class="h-7 px-2 text-xs" onclick={handleCopyLink}>
							{#if copied}
								<Check class="mr-1 h-3.5 w-3.5 text-emerald-600" />
								<span>Copied</span>
							{:else}
								<Copy class="mr-1 h-3.5 w-3.5" />
								<span>Copy</span>
							{/if}
						</Button>
					</div>
				</div>

				<!-- Actions Bar -->
				<div
					class="flex flex-wrap items-center justify-between gap-2 border-t border-[#e2e2de] pt-2"
				>
					<div class="flex items-center gap-2">
						<Button variant="outline" size="sm" class="text-xs" onclick={handlePrint}>
							<Printer class="mr-1.5 h-3.5 w-3.5" />
							Print QR
						</Button>

						<Button
							variant="outline"
							size="sm"
							class="text-xs"
							onclick={() => window.open(visitUrl, '_blank')}
						>
							<ExternalLink class="mr-1.5 h-3.5 w-3.5" />
							Preview Form
						</Button>
					</div>

					<div class="flex items-center gap-2">
						<Button
							variant="outline"
							size="sm"
							class="border-amber-200 text-xs text-amber-700 hover:bg-amber-50"
							onclick={handleRotateQR}
							disabled={isRotating}
						>
							{#if isRotating}
								<Loader2 class="mr-1 h-3.5 w-3.5 animate-spin" />
							{:else}
								<RefreshCw class="mr-1 h-3.5 w-3.5" />
							{/if}
							Rotate Token
						</Button>

						<Button
							variant="outline"
							size="sm"
							class="border-rose-200 text-xs text-rose-700 hover:bg-rose-50"
							onclick={handleRevokeQR}
							disabled={isRevoking}
						>
							{#if isRevoking}
								<Loader2 class="mr-1 h-3.5 w-3.5 animate-spin" />
							{:else}
								<Trash2 class="mr-1 h-3.5 w-3.5" />
							{/if}
							Revoke
						</Button>
					</div>
				</div>
			</div>
		{/if}
	</div>
</ResponsiveModal>
