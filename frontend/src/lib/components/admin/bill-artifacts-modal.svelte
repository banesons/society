<script lang="ts">
	import {
		societyApi,
		type MaintenanceBillPDFArtifact
	} from '$lib/api/society';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import PdfPreviewModal from '$lib/components/ui/pdf-preview-modal.svelte';
	import { toast } from '$lib/toast.svelte';
	import { triggerFileDownload, formatBytes } from '$lib/utils';
	import {
		X,
		FileText,
		Download,
		Loader2,
		RefreshCw,
		CheckCircle2,
		ShieldCheck,
		AlertCircle,
		Eye,
		Sparkles
	} from '@lucide/svelte';

	interface Props {
		isOpen: boolean;
		societyId: string;
		billId: string;
		billNumber: string;
		onClose: () => void;
		onArtifactGenerated?: () => void;
	}

	let { isOpen, societyId, billId, billNumber, onClose, onArtifactGenerated }: Props = $props();

	let artifacts = $state<MaintenanceBillPDFArtifact[]>([]);
	let isLoading = $state<boolean>(false);
	let error = $state<string | null>(null);
	let isGenerating = $state<boolean>(false);
	let downloadingArtifactId = $state<string | null>(null);

	// PDF Preview state
	let isPreviewOpen = $state<boolean>(false);
	let previewArtifact = $state<MaintenanceBillPDFArtifact | null>(null);

	function openArtifactPreview(artifact: MaintenanceBillPDFArtifact) {
		previewArtifact = artifact;
		isPreviewOpen = true;
	}

	$effect(() => {
		if (isOpen && societyId && billId) {
			loadArtifacts();
		}
	});

	async function loadArtifacts() {
		isLoading = true;
		error = null;
		try {
			const res = await societyApi.listBillArtifacts(societyId, billId);
			artifacts = res.artifacts || [];
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load PDF artifacts';
		} finally {
			isLoading = false;
		}
	}

	async function handleGeneratePdf() {
		if (isGenerating) return;
		isGenerating = true;
		try {
			const artifact = await societyApi.generateBillPdf(societyId, billId);
			toast.success(`PDF generated for bill ${billNumber}`);
			await loadArtifacts();
			onArtifactGenerated?.();
		} catch (err: unknown) {
			const msg = err instanceof Error ? err.message : 'Failed to generate PDF';
			toast.error(msg);
		} finally {
			isGenerating = false;
		}
	}

	async function handleDownloadArtifact(artifact: MaintenanceBillPDFArtifact) {
		if (downloadingArtifactId === artifact.id) return;
		downloadingArtifactId = artifact.id;
		try {
			const blob = await societyApi.downloadArtifact(societyId, artifact.id);
			triggerFileDownload(blob, artifact.file_name);
			toast.success(`Downloaded ${artifact.file_name}`);
		} catch (err: unknown) {
			const msg = err instanceof Error ? err.message : 'Failed to download artifact';
			toast.error(msg);
		} finally {
			downloadingArtifactId = null;
		}
	}
</script>

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div
			class="relative w-full max-w-xl rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl"
		>
			<div class="flex items-center justify-between border-b border-neutral-200 pb-3">
				<div>
					<h3 class="text-sm font-semibold text-neutral-950">
						PDF Artifacts — <span class="font-mono text-neutral-600">{billNumber}</span>
					</h3>
					<p class="text-[11px] text-neutral-500">
						Immutable generated document snapshots and SHA-256 verified records.
					</p>
				</div>
				<button
					class="text-neutral-400 transition-colors hover:text-neutral-700"
					onclick={onClose}
					aria-label="Close"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			<div class="mt-4">
				{#if isLoading}
					<div class="flex items-center justify-center py-10 text-neutral-400">
						<Loader2 class="mr-2 h-5 w-5 animate-spin" />
						<span class="text-xs">Loading artifacts...</span>
					</div>
				{:else if error}
					<div class="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800">
						<div class="flex items-center gap-2">
							<AlertCircle class="h-4 w-4 text-rose-600" />
							<span class="font-medium">Failed to retrieve artifacts</span>
						</div>
						<p class="mt-1 text-rose-700">{error}</p>
					</div>
				{:else if artifacts.length === 0}
					<div class="rounded-xl border border-dashed border-neutral-300 p-8 text-center">
						<FileText class="mx-auto h-8 w-8 text-neutral-400" />
						<p class="mt-2 text-xs font-semibold text-neutral-900">No PDF Artifacts Stored Yet</p>
						<p class="mt-1 text-[11px] text-neutral-500">
							This bill has not had a server-side PDF generated. Click below to generate.
						</p>
						<div class="mt-4">
							<Button
								variant="primary"
								size="sm"
								onclick={handleGeneratePdf}
								disabled={isGenerating}
							>
								{#if isGenerating}
									<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
									<span>Generating PDF...</span>
								{:else}
									<RefreshCw class="mr-1.5 h-3.5 w-3.5" />
									<span>Generate PDF Now</span>
								{/if}
							</Button>
						</div>
					</div>
				{:else}
					<div class="space-y-3">
						<div class="flex items-center justify-between">
							<span class="text-xs text-neutral-600"
								>{artifacts.length} {artifacts.length === 1 ? 'artifact' : 'artifacts'} recorded</span
							>
							<Button
								variant="outline"
								size="sm"
								onclick={handleGeneratePdf}
								disabled={isGenerating}
							>
								{#if isGenerating}
									<Loader2 class="mr-1.5 h-3 w-3 animate-spin" />
									<span>Generating...</span>
								{:else}
									<RefreshCw class="mr-1.5 h-3 w-3" />
									<span>Regenerate PDF</span>
								{/if}
							</Button>
						</div>

						<div class="max-h-72 overflow-y-auto divide-y divide-neutral-100 rounded-xl border border-neutral-200 bg-neutral-50/50">
							{#each artifacts as art, i (art.id)}
								<div class="p-3.5 {i === 0 ? 'bg-white' : ''}">
									<div class="flex flex-col sm:flex-row sm:items-start justify-between gap-3">
										<div class="space-y-1">
											<div class="flex flex-wrap items-center gap-2">
												<FileText class="h-4 w-4 text-neutral-700" />
												<span class="font-mono text-xs font-semibold text-neutral-950">
													{art.file_name}
												</span>
												{#if i === 0}
													<Badge variant="success" class="text-[9px]">
														Current Version
													</Badge>
												{/if}
												<Badge variant="outline" class="text-[9px]">
													{art.artifact_type}
												</Badge>
												{#if art.has_watermark}
													<span class="flex items-center text-[10px] text-emerald-700 font-medium">
														<ShieldCheck class="mr-1 h-3 w-3" />
														Watermarked
													</span>
												{/if}
											</div>

											<div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-neutral-500">
												<span>Size: <strong class="font-mono text-neutral-700">{formatBytes(art.file_size_bytes)}</strong></span>
												<span>·</span>
												<span>Generated: {new Date(art.created_at).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' })}</span>
											</div>

											<div class="flex items-center gap-1.5 pt-0.5 text-[10px] text-neutral-400">
												<span class="font-mono text-[9px] uppercase text-neutral-500">SHA-256:</span>
												<code class="rounded bg-neutral-100 px-1 py-0.5 font-mono text-[9px] text-neutral-600">
													{art.sha256_checksum.slice(0, 16)}...{art.sha256_checksum.slice(-8)}
												</code>
											</div>
										</div>

										<div class="flex items-center gap-1.5 shrink-0">
											<Button
												variant="outline"
												size="sm"
												onclick={() => openArtifactPreview(art)}
												title="Preview PDF Document"
											>
												<Eye class="mr-1 h-3 w-3" />
												<span>Preview</span>
											</Button>
											<Button
												variant="outline"
												size="sm"
												onclick={() => handleDownloadArtifact(art)}
												disabled={downloadingArtifactId === art.id}
											>
												{#if downloadingArtifactId === art.id}
													<Loader2 class="mr-1 h-3 w-3 animate-spin" />
													<span>Downloading...</span>
												{:else}
													<Download class="mr-1 h-3 w-3" />
													<span>Download</span>
												{/if}
											</Button>
										</div>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}
			</div>

			<div class="mt-6 flex justify-end">
				<Button variant="outline" size="sm" onclick={onClose}>
					<span>Close</span>
				</Button>
			</div>
		</div>
	</div>
{/if}

{#if isPreviewOpen && previewArtifact}
	<PdfPreviewModal
		isOpen={isPreviewOpen}
		title={previewArtifact.file_name}
		subtitle={`Bill ${billNumber} · Generated ${new Date(previewArtifact.created_at).toLocaleDateString('en-IN')}`}
		filename={previewArtifact.file_name}
		fetchPdf={() => societyApi.downloadArtifact(societyId, previewArtifact!.id)}
		onClose={() => {
			isPreviewOpen = false;
			previewArtifact = null;
		}}
	/>
{/if}
