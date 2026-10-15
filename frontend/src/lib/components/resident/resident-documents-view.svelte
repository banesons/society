<script lang="ts">
	import {
		societyApi,
		type SocietyDocument,
		type DocumentCategory,
		type DocumentVersion
	} from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import Button from '$lib/components/ui/button.svelte';
	import {
		FileText,
		Download,
		History,
		Search,
		Loader2,
		AlertCircle,
		X,
		BookOpen
	} from '@lucide/svelte';

	interface Props {
		societyId: string;
	}

	let { societyId }: Props = $props();

	let documents = $state<SocietyDocument[]>([]);
	let categories = $state<DocumentCategory[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	let selectedCategory = $state<string>('');
	let searchQuery = $state<string>('');

	// Details & History modal
	let isDetailsOpen = $state(false);
	let selectedDoc = $state<SocietyDocument | null>(null);
	let docVersions = $state<DocumentVersion[]>([]);
	let isLoadingVersions = $state(false);

	async function loadCategories() {
		try {
			const res = await societyApi.listDocumentCategories(societyId);
			categories = res.categories;
		} catch (err) {
			console.error('Failed to load categories', err);
		}
	}

	async function loadDocuments() {
		if (!societyId) return;
		isLoading = true;
		error = null;
		try {
			const res = await societyApi.listDocuments(societyId, {
				category_id: selectedCategory || undefined,
				classification: 'SOCIETY_RESIDENTS',
				status: 'ACTIVE',
				search: searchQuery || undefined,
				limit: 50,
				offset: 0
			});
			documents = res.documents;
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				error = err.message;
			} else {
				error = 'Failed to load society documents';
			}
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (societyId) {
			loadCategories();
		}
	});

	$effect(() => {
		void selectedCategory;
		void searchQuery;
		if (societyId) {
			loadDocuments();
		}
	});

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
	}

	function formatDate(iso: string): string {
		if (!iso) return '—';
		return new Date(iso).toLocaleDateString('en-IN', {
			day: '2-digit',
			month: 'short',
			year: 'numeric'
		});
	}

	async function openDetails(doc: SocietyDocument) {
		selectedDoc = doc;
		isDetailsOpen = true;
		isLoadingVersions = true;
		try {
			const res = await societyApi.listDocumentVersions(societyId, doc.id);
			docVersions = res.versions;
		} catch (err) {
			console.error('Failed to load versions', err);
		} finally {
			isLoadingVersions = false;
		}
	}

	async function triggerDownload(
		doc: SocietyDocument,
		versionId?: string,
		inline: boolean = false
	) {
		const verId = versionId || doc.current_version_id;
		if (!verId) return;

		try {
			const blob = await societyApi.downloadDocumentVersion(societyId, doc.id, verId, inline);
			const url = window.URL.createObjectURL(blob);
			if (inline) {
				window.open(url, '_blank');
			} else {
				const a = document.createElement('a');
				a.href = url;
				a.download = doc.current_version?.file_name || `${doc.title}.pdf`;
				document.body.appendChild(a);
				a.click();
				window.URL.revokeObjectURL(url);
				document.body.removeChild(a);
			}
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				alert(`Download failed: ${err.message}`);
			} else {
				alert('Failed to download document');
			}
		}
	}
</script>

<div
	class="rounded-3xl border border-neutral-200 bg-white/95 p-6 shadow-xs sm:p-8"
	id="resident-documents"
>
	<!-- Header -->
	<div class="flex items-center gap-2.5">
		<div
			class="flex h-10 w-10 items-center justify-center rounded-xl bg-neutral-100 text-neutral-900"
		>
			<BookOpen class="h-5 w-5" />
		</div>
		<div>
			<h2 class="text-lg font-bold tracking-tight text-neutral-950 sm:text-xl">
				Society Records & Documentation
			</h2>
			<p class="text-xs text-neutral-500">
				Access official registered society bylaws, circulars, meeting minutes, and compliance
				certificates.
			</p>
		</div>
	</div>

	{#if error}
		<div
			class="mt-4 flex items-center gap-2 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-xs text-rose-800"
		>
			<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
			<span>{error}</span>
		</div>
	{/if}

	<!-- Search & Filter -->
	<div
		class="mt-6 flex flex-col gap-3 rounded-2xl border border-neutral-200 bg-neutral-50/70 p-4 sm:flex-row sm:items-center sm:justify-between"
	>
		<div class="relative flex-1">
			<Search class="absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-neutral-400" />
			<input
				type="text"
				placeholder="Search society bylaws, notices, or minutes..."
				bind:value={searchQuery}
				class="w-full rounded-xl border border-neutral-200 bg-white py-2 pr-3 pl-9 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-900 focus:outline-none"
			/>
		</div>

		<select
			bind:value={selectedCategory}
			class="rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-none"
		>
			<option value="">All Categories</option>
			{#each categories as cat (cat.id)}
				<option value={cat.id}>{cat.name}</option>
			{/each}
		</select>
	</div>

	<!-- Documents List -->
	<div class="mt-6">
		{#if isLoading}
			<div class="flex min-h-[180px] flex-col items-center justify-center text-neutral-400">
				<Loader2 class="h-6 w-6 animate-spin text-neutral-900" />
				<p class="mt-2 text-xs">Loading society documents...</p>
			</div>
		{:else if documents.length === 0}
			<div
				class="flex min-h-[180px] flex-col items-center justify-center px-4 py-8 text-center text-neutral-400"
			>
				<FileText class="h-10 w-10 stroke-1 text-neutral-300" />
				<h3 class="mt-3 text-sm font-semibold text-neutral-800">No documents published</h3>
				<p class="mt-1 text-xs text-neutral-500">
					No official society documents match your current selection.
				</p>
			</div>
		{:else}
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each documents as doc (doc.id)}
					<div
						id="resident-doc-{doc.id}"
						class="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white p-5 shadow-2xs transition-colors hover:border-neutral-300"
					>
						<div>
							<div class="flex items-start justify-between gap-2">
								<span
									class="rounded-lg bg-neutral-100 px-2 py-0.5 text-[10px] font-semibold text-neutral-700"
								>
									{doc.category_name || 'General'}
								</span>
								<span class="font-mono text-[10px] text-neutral-400">
									v{doc.current_version?.version_number ?? 1}
								</span>
							</div>

							<h3 class="mt-2 line-clamp-1 text-sm font-bold text-neutral-950" title={doc.title}>
								{doc.title}
							</h3>

							{#if doc.description}
								<p class="mt-1 line-clamp-2 text-xs text-neutral-500" title={doc.description}>
									{doc.description}
								</p>
							{/if}

							<div class="mt-3 flex items-center gap-2 text-[11px] text-neutral-400">
								<span>{formatBytes(doc.current_version?.file_size ?? 0)}</span>
								<span>·</span>
								<span>{formatDate(doc.created_at)}</span>
							</div>
						</div>

						<div class="mt-4 flex items-center gap-2 border-t border-neutral-100 pt-3">
							<Button
								id="btn-resident-download-{doc.id}"
								variant="primary"
								size="sm"
								class="flex-1"
								onclick={() => triggerDownload(doc, undefined, false)}
							>
								<Download class="mr-1.5 h-3.5 w-3.5" />
								<span>Download</span>
							</Button>
							<Button
								id="btn-resident-history-{doc.id}"
								variant="outline"
								size="sm"
								onclick={() => openDetails(doc)}
							>
								<History class="h-3.5 w-3.5" />
							</Button>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

<!-- Resident Details Modal -->
{#if isDetailsOpen && selectedDoc}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
		role="dialog"
	>
		<div
			class="w-full max-w-lg rounded-3xl border border-neutral-200 bg-white p-6 shadow-xl sm:p-8"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div>
					<h3 class="text-base font-bold text-neutral-950">{selectedDoc.title}</h3>
					<p class="mt-0.5 font-mono text-xs text-neutral-400">{selectedDoc.document_number}</p>
				</div>
				<button
					type="button"
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isDetailsOpen = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<div class="mt-4 space-y-3 text-xs">
				<div class="flex justify-between border-b border-neutral-100 pb-2">
					<span class="text-neutral-500">Category:</span>
					<span class="font-medium text-neutral-900">{selectedDoc.category_name}</span>
				</div>
				<div class="flex justify-between border-b border-neutral-100 pb-2">
					<span class="text-neutral-500">Publication Date:</span>
					<span class="font-medium text-neutral-900">{formatDate(selectedDoc.created_at)}</span>
				</div>
				{#if selectedDoc.description}
					<div class="border-b border-neutral-100 pb-2">
						<span class="mb-1 block text-neutral-500">Description:</span>
						<p class="leading-relaxed text-neutral-700">{selectedDoc.description}</p>
					</div>
				{/if}
			</div>

			<!-- Available Revisions -->
			<div class="mt-4">
				<h4 class="mb-2 text-xs font-bold tracking-wider text-neutral-500 uppercase">
					Available Revisions
				</h4>
				{#if isLoadingVersions}
					<div class="flex h-20 items-center justify-center text-neutral-400">
						<Loader2 class="h-4 w-4 animate-spin" />
					</div>
				{:else}
					<div
						class="divide-y divide-neutral-100 overflow-hidden rounded-xl border border-neutral-200"
					>
						{#each docVersions as v (v.id)}
							<div class="flex items-center justify-between p-3 text-xs hover:bg-neutral-50">
								<div>
									<div class="flex items-center gap-2">
										<span class="font-mono font-bold text-neutral-900">v{v.version_number}</span>
										<span class="font-medium text-neutral-700">{v.file_name}</span>
									</div>
									<div class="mt-0.5 text-[10px] text-neutral-400">
										{formatBytes(v.file_size)} · {formatDate(v.created_at)}
									</div>
								</div>
								<button
									type="button"
									class="rounded-lg border border-neutral-200 p-1.5 text-neutral-600 hover:bg-neutral-100"
									onclick={() => {
										if (selectedDoc) triggerDownload(selectedDoc, v.id, false);
									}}
									title="Download"
								>
									<Download class="h-3.5 w-3.5" />
								</button>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<div class="mt-6 flex justify-end">
				<Button variant="outline" size="sm" onclick={() => (isDetailsOpen = false)}>Close</Button>
			</div>
		</div>
	</div>
{/if}
