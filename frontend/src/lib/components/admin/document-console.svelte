<script lang="ts">
	import {
		societyApi,
		type SocietyDocument,
		type DocumentCategory,
		type DocumentVersion,
		type DocumentClassification
	} from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import {
		FileText,
		UploadCloud,
		Download,
		History,
		Archive,
		RotateCcw,
		Search,
		Loader2,
		AlertCircle,
		CheckCircle2,
		X,
		Plus,
		Lock
	} from '@lucide/svelte';

	interface Props {
		societyId: string;
	}

	let { societyId }: Props = $props();

	// State variables
	let documents = $state<SocietyDocument[]>([]);
	let categories = $state<DocumentCategory[]>([]);
	let totalCount = $state(0);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let successMsg = $state<string | null>(null);

	// Filters
	let selectedCategory = $state<string>('');
	let selectedClassification = $state<string>('');
	let selectedStatus = $state<string>('ACTIVE');
	let searchQuery = $state<string>('');

	// Modals state
	let isUploadModalOpen = $state(false);
	let isVersionModalOpen = $state(false);
	let isDetailsModalOpen = $state(false);
	let isArchiveModalOpen = $state(false);

	let selectedDoc = $state<SocietyDocument | null>(null);
	let docVersions = $state<DocumentVersion[]>([]);
	let isLoadingVersions = $state(false);

	// Upload Form State
	let uploadTitle = $state('');
	let uploadDesc = $state('');
	let uploadCategory = $state('');
	let uploadClassification = $state<DocumentClassification>('SOCIETY_RESIDENTS');
	let uploadChangeSummary = $state('');
	let selectedFile = $state<File | null>(null);
	let isSubmitting = $state(false);
	let uploadError = $state<string | null>(null);

	// Archive Form State
	let archiveReason = $state('');
	let isArchiving = $state(false);
	let archiveError = $state<string | null>(null);

	// Version Upload State
	let newVersionFile = $state<File | null>(null);
	let newVersionSummary = $state('');
	let isUploadingVersion = $state(false);
	let versionError = $state<string | null>(null);

	// Load categories & documents
	async function loadCategories() {
		try {
			const res = await societyApi.listDocumentCategories(societyId);
			categories = res.categories;
		} catch (err: unknown) {
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
				classification: selectedClassification || undefined,
				status: selectedStatus || undefined,
				search: searchQuery || undefined,
				limit: 50,
				offset: 0
			});
			documents = res.documents;
			totalCount = res.total;
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
		void selectedClassification;
		void selectedStatus;
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

	const MAX_ORIGINAL_FILE_SIZE = 10 * 1024 * 1024; // 10 MB

	function handleFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files && input.files[0]) {
			const file = input.files[0];
			if (file.size > MAX_ORIGINAL_FILE_SIZE) {
				uploadError = `Selected file (${formatBytes(file.size)}) exceeds maximum allowed selection limit of 10 MB.`;
				selectedFile = null;
				input.value = '';
				return;
			}
			selectedFile = file;
			uploadError = null;
			// Pre-fill title if empty
			if (!uploadTitle) {
				const baseName = selectedFile.name.replace(/\.[^/.]+$/, '');
				uploadTitle = baseName.replace(/[-_]/g, ' ');
			}
		}
	}

	function handleVersionFileSelect(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files && input.files[0]) {
			const file = input.files[0];
			if (file.size > MAX_ORIGINAL_FILE_SIZE) {
				versionError = `Selected file (${formatBytes(file.size)}) exceeds maximum allowed selection limit of 10 MB.`;
				newVersionFile = null;
				input.value = '';
				return;
			}
			newVersionFile = file;
			versionError = null;
		}
	}

	async function handleUploadSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!selectedFile) {
			uploadError = 'Please select a file to upload';
			return;
		}
		if (!uploadTitle.trim()) {
			uploadError = 'Document title is required';
			return;
		}
		if (!uploadCategory) {
			uploadError = 'Please select a category';
			return;
		}

		isSubmitting = true;
		uploadError = null;

		const formData = new FormData();
		formData.append('file', selectedFile);
		formData.append('title', uploadTitle.trim());
		if (uploadDesc.trim()) formData.append('description', uploadDesc.trim());
		formData.append('category_id', uploadCategory);
		formData.append('classification', uploadClassification);
		if (uploadChangeSummary.trim()) formData.append('change_summary', uploadChangeSummary.trim());

		try {
			await societyApi.uploadDocument(societyId, formData);
			isUploadModalOpen = false;
			selectedFile = null;
			uploadTitle = '';
			uploadDesc = '';
			uploadChangeSummary = '';
			successMsg = 'Document uploaded successfully';
			setTimeout(() => (successMsg = null), 4000);
			loadDocuments();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				uploadError = err.message;
			} else {
				uploadError = 'Upload failed';
			}
		} finally {
			isSubmitting = false;
		}
	}

	async function openDetailsModal(doc: SocietyDocument) {
		selectedDoc = doc;
		isDetailsModalOpen = true;
		isLoadingVersions = true;
		try {
			const res = await societyApi.listDocumentVersions(societyId, doc.id);
			docVersions = res.versions;
		} catch (err) {
			console.error('Failed to load version history', err);
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

	function openVersionModal(doc: SocietyDocument) {
		selectedDoc = doc;
		newVersionFile = null;
		newVersionSummary = '';
		versionError = null;
		isVersionModalOpen = true;
	}

	async function handleVersionSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!selectedDoc || !newVersionFile) {
			versionError = 'Please choose a replacement file';
			return;
		}
		if ((selectedDoc.version_count || 1) >= 20) {
			versionError = 'Maximum version limit of 20 reached for this document.';
			return;
		}

		isUploadingVersion = true;
		versionError = null;

		const formData = new FormData();
		formData.append('file', newVersionFile);
		if (newVersionSummary.trim()) {
			formData.append('change_summary', newVersionSummary.trim());
		}

		try {
			await societyApi.uploadDocumentVersion(societyId, selectedDoc.id, formData);
			isVersionModalOpen = false;
			newVersionFile = null;
			newVersionSummary = '';
			successMsg = `Version updated for ${selectedDoc.title}`;
			setTimeout(() => (successMsg = null), 4000);
			loadDocuments();
			if (isDetailsModalOpen && selectedDoc) {
				openDetailsModal(selectedDoc);
			}
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				versionError = err.message;
			} else {
				versionError = 'Version upload failed';
			}
		} finally {
			isUploadingVersion = false;
		}
	}

	function openArchiveModal(doc: SocietyDocument) {
		selectedDoc = doc;
		archiveReason = '';
		archiveError = null;
		isArchiveModalOpen = true;
	}

	async function handleArchiveSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!selectedDoc) return;
		if (!archiveReason.trim()) {
			archiveError = 'Reason for archiving is required';
			return;
		}

		isArchiving = true;
		archiveError = null;

		try {
			await societyApi.archiveDocument(societyId, selectedDoc.id, archiveReason.trim());
			isArchiveModalOpen = false;
			successMsg = `Archived document: ${selectedDoc.title}`;
			setTimeout(() => (successMsg = null), 4000);
			loadDocuments();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				archiveError = err.message;
			} else {
				archiveError = 'Archiving failed';
			}
		} finally {
			isArchiving = false;
		}
	}

	async function handleRestore(doc: SocietyDocument) {
		if (!confirm(`Restore document "${doc.title}" to active status?`)) return;
		try {
			await societyApi.restoreDocument(societyId, doc.id);
			successMsg = `Restored document: ${doc.title}`;
			setTimeout(() => (successMsg = null), 4000);
			loadDocuments();
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				alert(`Restore failed: ${err.message}`);
			} else {
				alert('Failed to restore document');
			}
		}
	}
</script>

<div class="rounded-3xl border border-neutral-200 bg-white/95 p-6 shadow-xs sm:p-8" id="documents">
	<!-- Header -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2.5">
				<div
					class="flex h-10 w-10 items-center justify-center rounded-xl bg-neutral-100 text-neutral-900"
				>
					<FileText class="h-5 w-5" />
				</div>
				<div>
					<h2 class="text-lg font-bold tracking-tight text-neutral-950 sm:text-xl">
						Documents & Official Records
					</h2>
					<p class="text-xs text-neutral-500">
						Centralized repository for statutory certificates, meeting minutes, and society bylaws ({totalCount}
						records).
					</p>
				</div>
			</div>
		</div>

		<div class="flex items-center gap-2">
			<Button variant="primary" size="sm" onclick={() => (isUploadModalOpen = true)}>
				<UploadCloud class="mr-1.5 h-4 w-4" />
				<span>Upload Document</span>
			</Button>
		</div>
	</div>

	<!-- Alert Messages -->
	{#if successMsg}
		<div
			class="mt-4 flex items-center gap-2 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-xs text-emerald-800"
		>
			<CheckCircle2 class="h-4 w-4 shrink-0 text-emerald-600" />
			<span>{successMsg}</span>
		</div>
	{/if}

	{#if error}
		<div
			class="mt-4 flex items-center gap-2 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-xs text-rose-800"
		>
			<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
			<span>{error}</span>
		</div>
	{/if}

	<!-- Filter Controls -->
	<div
		class="mt-6 flex flex-col gap-3 rounded-2xl border border-neutral-200 bg-neutral-50/70 p-4 sm:flex-row sm:items-center sm:justify-between"
	>
		<!-- Search -->
		<div class="relative flex-1">
			<Search class="absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-neutral-400" />
			<input
				id="document-search-input"
				type="text"
				placeholder="Search by title, notes, or doc reference..."
				bind:value={searchQuery}
				class="w-full rounded-xl border border-neutral-200 bg-white py-2 pr-3 pl-9 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-900 focus:outline-none"
			/>
		</div>

		<!-- Category Filter -->
		<div class="flex flex-wrap items-center gap-2">
			<select
				id="document-category-filter"
				bind:value={selectedCategory}
				class="rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-none"
			>
				<option value="">All Categories</option>
				{#each categories as cat (cat.id)}
					<option value={cat.id}>{cat.name}</option>
				{/each}
			</select>

			<!-- Classification Filter -->
			<select
				id="document-classification-filter"
				bind:value={selectedClassification}
				class="rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-none"
			>
				<option value="">All Classifications</option>
				<option value="SOCIETY_RESIDENTS">Residents Visible</option>
				<option value="COMMITTEE">Committee Only</option>
				<option value="SOC_ADMIN_ONLY">Admin Confidential</option>
			</select>

			<!-- Status Filter -->
			<select
				id="document-status-filter"
				bind:value={selectedStatus}
				class="rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-none"
			>
				<option value="ACTIVE">Active Records</option>
				<option value="ARCHIVED">Archived Records</option>
			</select>
		</div>
	</div>

	<!-- Documents Table / Grid -->
	<div class="mt-6 overflow-hidden rounded-2xl border border-neutral-200 bg-white">
		{#if isLoading}
			<div class="flex min-h-[220px] flex-col items-center justify-center text-neutral-400">
				<Loader2 class="h-6 w-6 animate-spin text-neutral-900" />
				<p class="mt-2 text-xs">Loading records catalogue...</p>
			</div>
		{:else if documents.length === 0}
			<div
				class="flex min-h-[220px] flex-col items-center justify-center px-4 py-8 text-center text-neutral-400"
			>
				<FileText class="h-10 w-10 stroke-1 text-neutral-300" />
				<h3 class="mt-3 text-sm font-semibold text-neutral-800">No documents found</h3>
				<p class="mt-1 max-w-sm text-xs text-neutral-500">
					No records match the current filter criteria. Upload the society's first official record.
				</p>
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-left text-xs">
					<thead class="border-b border-neutral-200 bg-neutral-50/80 font-medium text-neutral-500">
						<tr>
							<th class="px-4 py-3">Document Title & Number</th>
							<th class="px-4 py-3">Category</th>
							<th class="px-4 py-3">Classification</th>
							<th class="px-4 py-3">Version & Size</th>
							<th class="px-4 py-3">Uploaded</th>
							<th class="px-4 py-3 text-right">Actions</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-100">
						{#each documents as doc (doc.id)}
							<tr class="transition-colors hover:bg-neutral-50/60">
								<!-- Title & Ref -->
								<td class="px-4 py-3.5">
									<div class="font-semibold text-neutral-950">{doc.title}</div>
									<div
										class="mt-0.5 flex items-center gap-1.5 font-mono text-[10px] text-neutral-400"
									>
										<span>{doc.document_number}</span>
										{#if doc.status === 'ARCHIVED'}
											<span
												class="py-0.2 rounded bg-rose-100 px-1 font-sans font-medium text-rose-700"
												>Archived</span
											>
										{/if}
									</div>
								</td>

								<!-- Category -->
								<td class="px-4 py-3.5">
									<span
										class="rounded-lg bg-neutral-100 px-2 py-1 text-[11px] font-medium text-neutral-700"
									>
										{doc.category_name || 'General'}
									</span>
								</td>

								<!-- Classification -->
								<td class="px-4 py-3.5">
									{#if doc.classification === 'SOC_ADMIN_ONLY'}
										<Badge variant="destructive" class="text-[10px]">
											<Lock class="mr-1 h-3 w-3" />
											Admin Only
										</Badge>
									{:else if doc.classification === 'COMMITTEE'}
										<Badge variant="warning" class="text-[10px]">Committee</Badge>
									{:else}
										<Badge
											variant="outline"
											class="border-emerald-200 bg-emerald-50 text-[10px] text-emerald-800"
										>
											Residents
										</Badge>
									{/if}
								</td>

								<!-- Version & File -->
								<td class="px-4 py-3.5">
									<div class="flex items-center gap-1.5">
										<span
											class="rounded bg-neutral-100 px-1.5 py-0.5 font-mono text-[10px] font-bold text-neutral-900"
										>
											v{doc.current_version?.version_number ?? 1}
										</span>
										<span class="text-neutral-500">
											({formatBytes(doc.current_version?.file_size ?? 0)})
										</span>
									</div>
									<div
										class="mt-0.5 max-w-[140px] truncate text-[10px] text-neutral-400"
										title={doc.current_version?.file_name}
									>
										{doc.current_version?.file_name || '—'}
									</div>
								</td>

								<!-- Uploaded -->
								<td class="px-4 py-3.5">
									<div class="text-neutral-800">{formatDate(doc.created_at)}</div>
									<div class="text-[10px] text-neutral-400">{doc.created_by_name || 'Staff'}</div>
								</td>

								<!-- Actions -->
								<td class="px-4 py-3.5 text-right">
									<div class="flex items-center justify-end gap-1.5">
										<!-- Download / Preview -->
										<button
											id="btn-download-{doc.id}"
											type="button"
											title="Download"
											class="rounded-lg border border-neutral-200 p-1.5 text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900"
											onclick={() => triggerDownload(doc, undefined, false)}
										>
											<Download class="h-3.5 w-3.5" />
										</button>

										<!-- Version History / Details -->
										<button
											id="btn-history-{doc.id}"
											type="button"
											title="Version History & Details"
											class="rounded-lg border border-neutral-200 p-1.5 text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900"
											onclick={() => openDetailsModal(doc)}
										>
											<History class="h-3.5 w-3.5" />
										</button>

										<!-- Upload New Version (Active only) -->
										{#if doc.status === 'ACTIVE'}
											<button
												id="btn-new-version-{doc.id}"
												type="button"
												title="Upload New Version"
												class="rounded-lg border border-neutral-200 p-1.5 text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900"
												onclick={() => openVersionModal(doc)}
											>
												<Plus class="h-3.5 w-3.5" />
											</button>
										{/if}

										<!-- Archive or Restore -->
										{#if doc.status === 'ACTIVE'}
											<button
												id="btn-archive-{doc.id}"
												type="button"
												title="Archive Record"
												class="rounded-lg border border-neutral-200 p-1.5 text-neutral-400 hover:border-rose-200 hover:bg-rose-50 hover:text-rose-600"
												onclick={() => openArchiveModal(doc)}
											>
												<Archive class="h-3.5 w-3.5" />
											</button>
										{:else}
											<button
												id="btn-restore-{doc.id}"
												type="button"
												title="Restore Document"
												class="rounded-lg border border-neutral-200 p-1.5 text-emerald-600 hover:bg-emerald-50"
												onclick={() => handleRestore(doc)}
											>
												<RotateCcw class="h-3.5 w-3.5" />
											</button>
										{/if}
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</div>

<!-- Upload Document Modal -->
{#if isUploadModalOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
		role="dialog"
	>
		<div
			class="w-full max-w-lg rounded-3xl border border-neutral-200 bg-white p-6 shadow-xl sm:p-8"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div class="flex items-center gap-2">
					<UploadCloud class="h-5 w-5 text-neutral-900" />
					<h3 class="text-base font-bold text-neutral-950">Upload Society Record</h3>
				</div>
				<button
					type="button"
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isUploadModalOpen = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if uploadError}
				<div class="mt-4 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800">
					{uploadError}
				</div>
			{/if}

			<form onsubmit={handleUploadSubmit} class="mt-4 space-y-4 text-xs">
				<!-- File Picker -->
				<div>
					<label class="block font-medium text-neutral-700"
						>Document File (Max 10MB original, accepted artifact up to 5MB) *</label
					>
					<div
						class="mt-1 flex items-center justify-center rounded-2xl border-2 border-dashed border-neutral-200 bg-neutral-50/60 p-4 transition-colors hover:bg-neutral-50"
					>
						<input
							type="file"
							accept=".pdf,.png,.jpg,.jpeg,.webp,.docx,.xlsx"
							onchange={handleFileSelect}
							class="cursor-pointer text-xs text-neutral-500 file:mr-3 file:rounded-xl file:border-0 file:bg-neutral-900 file:px-3 file:py-1.5 file:text-xs file:font-semibold file:text-white hover:file:bg-neutral-800"
						/>
					</div>
					<div class="mt-1.5 flex flex-wrap gap-1 text-[10px] text-neutral-400">
						<span
							>Allowed formats: PDF, PNG, JPEG, WebP, Word (.docx), Excel (.xlsx). Lossless PNG
							optimization applied when beneficial.</span
						>
					</div>
				</div>

				<!-- Title -->
				<div>
					<label class="block font-medium text-neutral-700">Document Title *</label>
					<input
						type="text"
						bind:value={uploadTitle}
						placeholder="e.g. Society Registered Bylaws 2026"
						class="mt-1 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-none"
						required
					/>
				</div>

				<!-- Category -->
				<div>
					<label class="block font-medium text-neutral-700">Category *</label>
					<select
						bind:value={uploadCategory}
						class="mt-1 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-none"
						required
					>
						<option value="">Select Category...</option>
						{#each categories as cat (cat.id)}
							<option value={cat.id}>{cat.name}</option>
						{/each}
					</select>
				</div>

				<!-- Classification -->
				<div>
					<label class="block font-medium text-neutral-700">Access Classification *</label>
					<select
						bind:value={uploadClassification}
						class="mt-1 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-none"
					>
						<option value="SOCIETY_RESIDENTS">Residents Visible (All Society Members)</option>
						<option value="COMMITTEE">Managing Committee Only</option>
						<option value="SOC_ADMIN_ONLY">Admin Only (Confidential Records)</option>
					</select>
				</div>

				<!-- Description -->
				<div>
					<label class="block font-medium text-neutral-700">Description / Notes</label>
					<textarea
						bind:value={uploadDesc}
						rows="2"
						placeholder="Optional summary, agenda notes, or statutory references..."
						class="mt-1 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-none"
					></textarea>
				</div>

				<!-- Actions -->
				<div class="mt-6 flex justify-end gap-2 border-t border-neutral-100 pt-4">
					<Button
						variant="outline"
						size="sm"
						type="button"
						onclick={() => (isUploadModalOpen = false)}
					>
						Cancel
					</Button>
					<Button variant="primary" size="sm" type="submit" disabled={isSubmitting}>
						{#if isSubmitting}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Uploading...</span>
						{:else}
							<span>Confirm & Upload</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- Document Details & Version History Modal -->
{#if isDetailsModalOpen && selectedDoc}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
		role="dialog"
	>
		<div
			class="w-full max-w-2xl rounded-3xl border border-neutral-200 bg-white p-6 shadow-xl sm:p-8"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div>
					<div class="flex items-center gap-2">
						<FileText class="h-5 w-5 text-neutral-900" />
						<h3 class="text-base font-bold text-neutral-950">{selectedDoc.title}</h3>
					</div>
					<p class="mt-0.5 font-mono text-xs text-neutral-400">{selectedDoc.document_number}</p>
				</div>
				<button
					type="button"
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isDetailsModalOpen = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<!-- Meta summary -->
			<div class="mt-4 grid grid-cols-2 gap-3 text-xs sm:grid-cols-4">
				<div class="rounded-xl border border-neutral-100 bg-neutral-50 p-2.5">
					<span class="text-[10px] text-neutral-400">Category</span>
					<p class="font-medium text-neutral-900">{selectedDoc.category_name}</p>
				</div>
				<div class="rounded-xl border border-neutral-100 bg-neutral-50 p-2.5">
					<span class="text-[10px] text-neutral-400">Classification</span>
					<p class="font-medium text-neutral-900">{selectedDoc.classification}</p>
				</div>
				<div class="rounded-xl border border-neutral-100 bg-neutral-50 p-2.5">
					<span class="text-[10px] text-neutral-400">Created Date</span>
					<p class="font-medium text-neutral-900">{formatDate(selectedDoc.created_at)}</p>
				</div>
				<div class="rounded-xl border border-neutral-100 bg-neutral-50 p-2.5">
					<span class="text-[10px] text-neutral-400">Status</span>
					<p class="font-medium text-neutral-900">{selectedDoc.status}</p>
				</div>
			</div>

			{#if selectedDoc.description}
				<div
					class="mt-3 rounded-xl border border-neutral-100 bg-neutral-50/60 p-3 text-xs text-neutral-600"
				>
					{selectedDoc.description}
				</div>
			{/if}

			<!-- Version History Header -->
			<div class="mt-6 flex items-center justify-between">
				<h4 class="text-xs font-bold tracking-wider text-neutral-500 uppercase">
					Version History ({docVersions.length} revisions)
				</h4>
				{#if selectedDoc.status === 'ACTIVE'}
					<Button
						variant="outline"
						size="sm"
						onclick={() => {
							if (selectedDoc) openVersionModal(selectedDoc);
						}}
					>
						<Plus class="mr-1 h-3.5 w-3.5" />
						<span>New Version</span>
					</Button>
				{/if}
			</div>

			<!-- Versions Table -->
			<div class="mt-3 overflow-hidden rounded-xl border border-neutral-200">
				{#if isLoadingVersions}
					<div class="flex h-32 items-center justify-center text-neutral-400">
						<Loader2 class="h-5 w-5 animate-spin" />
					</div>
				{:else}
					<table class="w-full text-left text-xs">
						<thead
							class="border-b border-neutral-200 bg-neutral-50 text-[11px] font-medium text-neutral-500"
						>
							<tr>
								<th class="px-3 py-2">Ver</th>
								<th class="px-3 py-2">File & Notes</th>
								<th class="px-3 py-2">Size</th>
								<th class="px-3 py-2">Uploaded</th>
								<th class="px-3 py-2 text-right">Action</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-100">
							{#each docVersions as v (v.id)}
								<tr class="hover:bg-neutral-50/50">
									<td class="px-3 py-2.5 font-mono font-bold text-neutral-900"
										>v{v.version_number}</td
									>
									<td class="px-3 py-2.5">
										<div class="font-medium text-neutral-900">{v.file_name}</div>
										{#if v.change_summary}
											<div class="text-[10px] text-neutral-500">{v.change_summary}</div>
										{/if}
										<div
											class="max-w-xs truncate font-mono text-[9px] text-neutral-400"
											title={v.file_hash}
										>
											SHA: {v.file_hash.substring(0, 16)}...
										</div>
									</td>
									<td class="px-3 py-2.5 text-neutral-600">{formatBytes(v.file_size)}</td>
									<td class="px-3 py-2.5">
										<div>{formatDate(v.created_at)}</div>
										<div class="text-[10px] text-neutral-400">{v.uploaded_by_name || 'Staff'}</div>
									</td>
									<td class="px-3 py-2.5 text-right">
										<button
											type="button"
											class="rounded-lg border border-neutral-200 p-1 text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900"
											onclick={() => {
												if (selectedDoc) triggerDownload(selectedDoc, v.id, false);
											}}
											title="Download revision"
										>
											<Download class="h-3.5 w-3.5" />
										</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
			</div>

			<div class="mt-6 flex justify-end">
				<Button variant="outline" size="sm" onclick={() => (isDetailsModalOpen = false)}>
					Close
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Upload New Version Modal -->
{#if isVersionModalOpen && selectedDoc}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
		role="dialog"
	>
		<div
			class="w-full max-w-md rounded-3xl border border-neutral-200 bg-white p-6 shadow-xl sm:p-8"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div class="flex items-center gap-2">
					<UploadCloud class="h-5 w-5 text-neutral-900" />
					<h3 class="text-base font-bold text-neutral-950">Upload New Revision</h3>
				</div>
				<button
					type="button"
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isVersionModalOpen = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<p class="mt-2 text-xs text-neutral-500">
				Uploading a new version for <strong class="text-neutral-900">{selectedDoc.title}</strong>
				({selectedDoc.document_number}).
			</p>
			<p class="mt-1 text-[11px] text-neutral-500">
				Current version count: <span class="font-semibold text-neutral-700"
					>{selectedDoc.version_count || 1} of 20 max</span
				>
			</p>

			{#if (selectedDoc.version_count || 1) >= 20}
				<div
					class="mt-2 rounded-xl border border-amber-200 bg-amber-50 p-2.5 text-[11px] text-amber-800"
				>
					This document has reached the maximum limit of 20 versions. No further revisions can be
					added.
				</div>
			{/if}

			{#if versionError}
				<div class="mt-3 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800">
					{versionError}
				</div>
			{/if}

			<form onsubmit={handleVersionSubmit} class="mt-4 space-y-4 text-xs">
				<div>
					<label class="block font-medium text-neutral-700"
						>Select Replacement File (Max 10MB original, accepted artifact up to 5MB) *</label
					>
					<div
						class="mt-1 flex items-center justify-center rounded-2xl border-2 border-dashed border-neutral-200 bg-neutral-50/60 p-4"
					>
						<input
							id="version-file-input"
							type="file"
							accept=".pdf,.png,.jpg,.jpeg,.webp,.docx,.xlsx"
							onchange={handleVersionFileSelect}
							class="cursor-pointer text-xs text-neutral-500 file:mr-3 file:rounded-xl file:border-0 file:bg-neutral-900 file:px-3 file:py-1.5 file:text-xs file:font-semibold file:text-white hover:file:bg-neutral-800"
							disabled={(selectedDoc.version_count || 1) >= 20}
							required
						/>
					</div>
				</div>

				<div>
					<label class="block font-medium text-neutral-700">Change Summary / Revision Notes</label>
					<textarea
						id="version-summary-input"
						bind:value={newVersionSummary}
						rows="2"
						placeholder="Explain what changed in this version..."
						class="mt-1 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-none"
						disabled={(selectedDoc.version_count || 1) >= 20}></textarea>
				</div>

				<div class="mt-6 flex justify-end gap-2 border-t border-neutral-100 pt-4">
					<Button
						variant="outline"
						size="sm"
						type="button"
						onclick={() => (isVersionModalOpen = false)}
					>
						Cancel
					</Button>
					<Button
						id="btn-upload-version-submit"
						variant="primary"
						size="sm"
						type="submit"
						disabled={isUploadingVersion || (selectedDoc.version_count || 1) >= 20}
					>
						{#if isUploadingVersion}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Uploading...</span>
						{:else}
							<span>Commit Revision</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- Archive Document Modal -->
{#if isArchiveModalOpen && selectedDoc}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
		role="dialog"
	>
		<div
			class="w-full max-w-md rounded-3xl border border-neutral-200 bg-white p-6 shadow-xl sm:p-8"
		>
			<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
				<div class="flex items-center gap-2 text-rose-600">
					<Archive class="h-5 w-5" />
					<h3 class="text-base font-bold text-neutral-950">Archive Document</h3>
				</div>
				<button
					type="button"
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isArchiveModalOpen = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<p class="mt-3 text-xs leading-relaxed text-neutral-600">
				Archiving <strong class="text-neutral-900">{selectedDoc.title}</strong> will hide it from normal
				resident browsing. Historical versions remain preserved for audit compliance.
			</p>

			{#if archiveError}
				<div class="mt-3 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800">
					{archiveError}
				</div>
			{/if}

			<form onsubmit={handleArchiveSubmit} class="mt-4 space-y-4 text-xs">
				<div>
					<label class="block font-medium text-neutral-700">Reason for Archiving *</label>
					<textarea
						id="archive-reason-input"
						bind:value={archiveReason}
						rows="3"
						placeholder="Mandatory reason (e.g. Superseded by 2027 registered deed)..."
						class="mt-1 w-full rounded-xl border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-900 focus:border-neutral-900 focus:outline-none"
						required></textarea>
				</div>

				<div class="mt-6 flex justify-end gap-2 border-t border-neutral-100 pt-4">
					<Button
						variant="outline"
						size="sm"
						type="button"
						onclick={() => (isArchiveModalOpen = false)}
					>
						Cancel
					</Button>
					<Button
						id="btn-confirm-archive"
						variant="destructive"
						size="sm"
						type="submit"
						disabled={isArchiving}
					>
						{#if isArchiving}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
							<span>Archiving...</span>
						{:else}
							<span>Confirm Archival</span>
						{/if}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
