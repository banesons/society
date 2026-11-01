<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import * as pdfjsLib from 'pdfjs-dist';
	import pdfWorkerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url';
	import Button from '$lib/components/ui/button.svelte';
	import {
		X,
		Download,
		ChevronLeft,
		ChevronRight,
		ZoomIn,
		ZoomOut,
		Maximize2,
		Minimize2,
		Loader2,
		AlertCircle,
		RefreshCw,
		FileText
	} from '@lucide/svelte';
	import { triggerFileDownload } from '$lib/utils';

	// Configure PDF.js worker in browser
	if (typeof window !== 'undefined' && !pdfjsLib.GlobalWorkerOptions.workerSrc) {
		try {
			pdfjsLib.GlobalWorkerOptions.workerSrc = pdfWorkerUrl;
		} catch (err) {
			console.warn('Failed to assign PDF.js worker, running in fallback mode:', err);
		}
	}

	interface Props {
		isOpen: boolean;
		title: string;
		subtitle?: string;
		pdfBlob?: Blob | null;
		fetchPdf?: () => Promise<Blob>;
		filename?: string;
		onClose: () => void;
	}

	let {
		isOpen,
		title,
		subtitle,
		pdfBlob = null,
		fetchPdf,
		filename = 'document.pdf',
		onClose
	}: Props = $props();

	// Component state
	let loadedBlob = $state<Blob | null>(null);
	let isLoading = $state<boolean>(true);
	let errorMessage = $state<string | null>(null);

	let pdfDoc = $state<any>(null);
	let totalPages = $state<number>(0);
	let currentPage = $state<number>(1);
	let scale = $state<number>(1.0);
	let isRendering = $state<boolean>(false);

	let canvasEl: HTMLCanvasElement | null = $state(null);
	let containerEl: HTMLDivElement | null = $state(null);
	let activeRenderTask: any = null;

	// Initial load and document parsing
	$effect(() => {
		if (isOpen) {
			initDocument();
		} else {
			cleanupDocument();
		}
	});

	async function initDocument() {
		isLoading = true;
		errorMessage = null;
		currentPage = 1;

		try {
			let blob = pdfBlob;
			if (!blob && fetchPdf) {
				blob = await fetchPdf();
			}

			if (!blob || blob.size === 0) {
				throw new Error('Received an empty or invalid PDF document from server');
			}

			loadedBlob = blob;
			const arrayBuffer = await blob.arrayBuffer();

			// Load with PDF.js
			const loadingTask = pdfjsLib.getDocument({
				data: new Uint8Array(arrayBuffer),
				cMapUrl: 'https://cdn.jsdelivr.net/npm/pdfjs-dist@4.10.38/cmaps/',
				cMapPacked: true
			});

			const doc = await loadingTask.promise;
			pdfDoc = doc;
			totalPages = doc.numPages;

			// Auto calculate initial fit-to-width
			await calculateFitToWidth(doc);
			await renderCurrentPage();
		} catch (err: unknown) {
			console.error('Failed to load PDF document:', err);
			if (err instanceof Error) {
				errorMessage = err.message;
			} else {
				errorMessage = 'Failed to parse and render PDF document';
			}
		} finally {
			isLoading = false;
		}
	}

	async function calculateFitToWidth(docToUse?: any) {
		const doc = docToUse || pdfDoc;
		if (!doc || !containerEl) {
			scale = 1.0;
			return;
		}

		try {
			const page = await doc.getPage(currentPage);
			const unscaledViewport = page.getViewport({ scale: 1.0 });
			const containerWidth = containerEl.clientWidth - 48; // padding margin
			if (containerWidth > 0 && unscaledViewport.width > 0) {
				const autoScale = containerWidth / unscaledViewport.width;
				// Clamp reasonable bounds for desktop / mobile
				scale = Math.min(Math.max(autoScale, 0.5), 2.0);
			} else {
				scale = 1.0;
			}
		} catch {
			scale = 1.0;
		}
	}

	async function fitToWidth() {
		await calculateFitToWidth();
		await renderCurrentPage();
	}

	async function fitToPage() {
		if (!pdfDoc || !containerEl) return;
		try {
			const page = await pdfDoc.getPage(currentPage);
			const unscaledViewport = page.getViewport({ scale: 1.0 });
			const containerHeight = containerEl.clientHeight - 64;
			if (containerHeight > 0 && unscaledViewport.height > 0) {
				const autoScale = containerHeight / unscaledViewport.height;
				scale = Math.min(Math.max(autoScale, 0.4), 2.0);
				await renderCurrentPage();
			}
		} catch (err) {
			console.warn('Fit to page calculation failed:', err);
		}
	}

	async function renderCurrentPage() {
		if (!pdfDoc || !canvasEl) return;

		// Cancel any pending render task to avoid collision
		if (activeRenderTask) {
			activeRenderTask.cancel();
			activeRenderTask = null;
		}

		isRendering = true;
		try {
			const page = await pdfDoc.getPage(currentPage);
			const viewport = page.getViewport({ scale });

			// Use devicePixelRatio for crisp rendering on HiDPI displays
			const dpr = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1;
			canvasEl.width = Math.floor(viewport.width * dpr);
			canvasEl.height = Math.floor(viewport.height * dpr);
			canvasEl.style.width = `${Math.floor(viewport.width)}px`;
			canvasEl.style.height = `${Math.floor(viewport.height)}px`;

			const ctx = canvasEl.getContext('2d');
			if (!ctx) return;

			ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

			const renderContext = {
				canvasContext: ctx,
				viewport: viewport
			};

			activeRenderTask = page.render(renderContext);
			await activeRenderTask.promise;
		} catch (err: any) {
			// Ignore cancelled renders caused by fast page-turning or zooming
			if (err?.name !== 'RenderingCancelledException') {
				console.error('Error rendering PDF page:', err);
			}
		} finally {
			isRendering = false;
			activeRenderTask = null;
		}
	}

	function changePage(delta: number) {
		const newPage = currentPage + delta;
		if (newPage >= 1 && newPage <= totalPages) {
			currentPage = newPage;
			renderCurrentPage();
		}
	}

	function changeZoom(factor: number) {
		const newScale = Math.round((scale * factor) * 100) / 100;
		if (newScale >= 0.4 && newScale <= 3.0) {
			scale = newScale;
			renderCurrentPage();
		}
	}

	function handleDownload() {
		if (loadedBlob) {
			triggerFileDownload(loadedBlob, filename);
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (!isOpen) return;

		if (event.key === 'Escape') {
			event.preventDefault();
			onClose();
		} else if (event.key === 'ArrowLeft' || event.key === 'PageUp') {
			event.preventDefault();
			changePage(-1);
		} else if (event.key === 'ArrowRight' || event.key === 'PageDown') {
			event.preventDefault();
			changePage(1);
		} else if (event.key === '+' || (event.key === '=' && event.metaKey)) {
			event.preventDefault();
			changeZoom(1.2);
		} else if (event.key === '-' || (event.key === '-' && event.metaKey)) {
			event.preventDefault();
			changeZoom(0.8);
		}
	}

	function cleanupDocument() {
		if (activeRenderTask) {
			activeRenderTask.cancel();
			activeRenderTask = null;
		}
		if (pdfDoc) {
			pdfDoc.destroy();
			pdfDoc = null;
		}
		loadedBlob = null;
		totalPages = 0;
		currentPage = 1;
		scale = 1.0;
		errorMessage = null;
	}

	onDestroy(() => {
		cleanupDocument();
	});
</script>

<svelte:window onkeydown={handleKeydown} />

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex flex-col bg-neutral-950/80 backdrop-blur-xs transition-opacity"
		role="dialog"
		aria-modal="true"
		aria-label={`PDF Preview: ${title}`}
	>
		<!-- Editorial Toolbar -->
		<header
			class="flex h-14 shrink-0 items-center justify-between border-b border-[#e2e2de] bg-[#f3f3f1] px-4 text-[#121212] select-none"
		>
			<!-- Document Title -->
			<div class="flex items-center gap-3 overflow-hidden pr-2">
				<div
					class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-[#e2e2de] bg-white text-neutral-800"
				>
					<FileText class="h-4 w-4 text-emerald-800" />
				</div>
				<div class="min-w-0">
					<h2 class="truncate text-xs font-semibold tracking-tight text-[#121212]">
						{title}
					</h2>
					{#if subtitle}
						<p class="truncate text-[10px] text-neutral-500">{subtitle}</p>
					{/if}
				</div>
			</div>

			<!-- Center Controls: Multi-page & Zoom -->
			{#if !isLoading && !errorMessage && totalPages > 0}
				<div class="hidden items-center gap-1 sm:flex">
					<!-- Page Navigation -->
					<div
						class="flex items-center rounded-lg border border-[#e2e2de] bg-white p-0.5 shadow-2xs"
					>
						<button
							class="rounded p-1 text-neutral-600 transition-colors hover:bg-neutral-100 disabled:opacity-30 disabled:hover:bg-transparent"
							onclick={() => changePage(-1)}
							disabled={currentPage <= 1 || isRendering}
							title="Previous page (Left arrow)"
							aria-label="Previous page"
						>
							<ChevronLeft class="h-4 w-4" />
						</button>
						<span class="px-2 font-mono text-[11px] font-medium text-neutral-700">
							{currentPage} / {totalPages}
						</span>
						<button
							class="rounded p-1 text-neutral-600 transition-colors hover:bg-neutral-100 disabled:opacity-30 disabled:hover:bg-transparent"
							onclick={() => changePage(1)}
							disabled={currentPage >= totalPages || isRendering}
							title="Next page (Right arrow)"
							aria-label="Next page"
						>
							<ChevronRight class="h-4 w-4" />
						</button>
					</div>

					<div class="mx-1 h-4 w-px bg-[#e2e2de]"></div>

					<!-- Zoom Controls -->
					<div
						class="flex items-center rounded-lg border border-[#e2e2de] bg-white p-0.5 shadow-2xs"
					>
						<button
							class="rounded p-1 text-neutral-600 transition-colors hover:bg-neutral-100 disabled:opacity-30"
							onclick={() => changeZoom(0.8)}
							disabled={scale <= 0.4 || isRendering}
							title="Zoom out (-)"
							aria-label="Zoom out"
						>
							<ZoomOut class="h-4 w-4" />
						</button>
						<span class="w-12 text-center font-mono text-[11px] font-medium text-neutral-700">
							{Math.round(scale * 100)}%
						</span>
						<button
							class="rounded p-1 text-neutral-600 transition-colors hover:bg-neutral-100 disabled:opacity-30"
							onclick={() => changeZoom(1.2)}
							disabled={scale >= 3.0 || isRendering}
							title="Zoom in (+)"
							aria-label="Zoom in"
						>
							<ZoomIn class="h-4 w-4" />
						</button>
					</div>

					<div class="mx-1 h-4 w-px bg-[#e2e2de]"></div>

					<!-- Fit Modes -->
					<div
						class="flex items-center rounded-lg border border-[#e2e2de] bg-white p-0.5 shadow-2xs"
					>
						<button
							class="flex items-center gap-1 rounded px-2 py-1 text-[11px] font-medium text-neutral-700 transition-colors hover:bg-neutral-100"
							onclick={fitToWidth}
							title="Fit to width"
							aria-label="Fit to width"
						>
							<Maximize2 class="h-3 w-3" />
							<span>Fit Width</span>
						</button>
						<button
							class="flex items-center gap-1 rounded px-2 py-1 text-[11px] font-medium text-neutral-700 transition-colors hover:bg-neutral-100"
							onclick={fitToPage}
							title="Fit to page"
							aria-label="Fit to page"
						>
							<Minimize2 class="h-3 w-3" />
							<span>Fit Page</span>
						</button>
					</div>
				</div>
			{/if}

			<!-- Actions: Download & Close -->
			<div class="flex items-center gap-2">
				{#if loadedBlob}
					<Button
						variant="outline"
						size="sm"
						onclick={handleDownload}
						title="Download PDF document"
						aria-label="Download PDF document"
						class="border-[#e2e2de] bg-white text-xs font-medium text-[#121212] shadow-2xs hover:bg-neutral-50"
					>
						<Download class="mr-1.5 h-3.5 w-3.5" />
						<span class="hidden sm:inline">Download</span>
					</Button>
				{/if}

				<button
					class="flex h-8 w-8 items-center justify-center rounded-lg border border-[#e2e2de] bg-white text-neutral-600 shadow-2xs transition-colors hover:bg-neutral-100 hover:text-neutral-900"
					onclick={onClose}
					title="Close viewer (Escape)"
					aria-label="Close preview"
				>
					<X class="h-4 w-4" />
				</button>
			</div>
		</header>

		<!-- Mobile Compact Controls Sub-bar -->
		{#if !isLoading && !errorMessage && totalPages > 0}
			<div
				class="flex h-10 shrink-0 items-center justify-between border-b border-[#e2e2de] bg-[#f9f9f8] px-4 text-xs sm:hidden"
			>
				<div class="flex items-center gap-1">
					<button
						class="rounded p-1 text-neutral-700 disabled:opacity-30"
						onclick={() => changePage(-1)}
						disabled={currentPage <= 1 || isRendering}
					>
						<ChevronLeft class="h-4 w-4" />
					</button>
					<span class="font-mono text-[11px] font-semibold text-neutral-800">
						{currentPage} / {totalPages}
					</span>
					<button
						class="rounded p-1 text-neutral-700 disabled:opacity-30"
						onclick={() => changePage(1)}
						disabled={currentPage >= totalPages || isRendering}
					>
						<ChevronRight class="h-4 w-4" />
					</button>
				</div>

				<div class="flex items-center gap-1">
					<button
						class="rounded p-1 text-neutral-700 disabled:opacity-30"
						onclick={() => changeZoom(0.8)}
						disabled={scale <= 0.4 || isRendering}
					>
						<ZoomOut class="h-4 w-4" />
					</button>
					<span class="font-mono text-[11px] text-neutral-700">{Math.round(scale * 100)}%</span>
					<button
						class="rounded p-1 text-neutral-700 disabled:opacity-30"
						onclick={() => changeZoom(1.2)}
						disabled={scale >= 3.0 || isRendering}
					>
						<ZoomIn class="h-4 w-4" />
					</button>
					<button
						class="ml-1 rounded border border-[#e2e2de] bg-white px-2 py-0.5 text-[10px] font-medium"
						onclick={fitToWidth}
					>
						Fit
					</button>
				</div>
			</div>
		{/if}

		<!-- Document View Canvas Area -->
		<div
			bind:this={containerEl}
			class="relative flex flex-1 items-start justify-center overflow-auto bg-[#e5e5e2] p-4 sm:p-8"
		>
			{#if isLoading}
				<div class="flex flex-col items-center justify-center gap-3 py-24 text-neutral-600">
					<Loader2 class="h-8 w-8 animate-spin text-neutral-700" />
					<p class="text-xs font-medium">Rendering PDF invoice...</p>
					<p class="text-[11px] text-neutral-500">Preparing vector typography and watermarks</p>
				</div>
			{:else if errorMessage}
				<div
					class="max-w-md rounded-2xl border border-rose-200 bg-white p-6 text-center shadow-lg"
				>
					<div
						class="mx-auto flex h-10 w-10 items-center justify-center rounded-full bg-rose-50 text-rose-600"
					>
						<AlertCircle class="h-5 w-5" />
					</div>
					<h3 class="mt-3 text-sm font-semibold text-neutral-900">Unable to Render PDF Preview</h3>
					<p class="mt-1 text-xs text-neutral-600">{errorMessage}</p>

					<div class="mt-5 flex justify-center gap-2">
						<Button variant="outline" size="sm" onclick={initDocument}>
							<RefreshCw class="mr-1.5 h-3.5 w-3.5" />
							<span>Try Again</span>
						</Button>
						{#if loadedBlob}
							<Button variant="primary" size="sm" onclick={handleDownload}>
								<Download class="mr-1.5 h-3.5 w-3.5" />
								<span>Download File</span>
							</Button>
						{/if}
					</div>
				</div>
			{:else}
				<div class="relative flex flex-col items-center">
					{#if isRendering}
						<div
							class="absolute inset-0 z-10 flex items-center justify-center bg-white/50 backdrop-blur-2xs"
						>
							<Loader2 class="h-6 w-6 animate-spin text-neutral-800" />
						</div>
					{/if}
					<canvas
						bind:this={canvasEl}
						class="rounded-xs border border-[#e2e2de] bg-white shadow-xl transition-all duration-150"
					></canvas>
				</div>
			{/if}
		</div>
	</div>
{/if}
