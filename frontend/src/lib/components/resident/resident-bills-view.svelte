<script lang="ts">
	import {
		societyApi,
		type MaintenanceBill,
		type MaintenanceBillDetail,
		type ResidentPaymentItem,
		type MaintenanceReceipt
	} from '$lib/api/society';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import {
		Receipt,
		AlertCircle,
		Loader2,
		RefreshCw,
		Eye,
		X,
		FileText,
		Printer
	} from '@lucide/svelte';

	interface Props {
		societyId: string;
	}

	let { societyId }: Props = $props();

	let activeTab = $state<'bills' | 'payments'>('bills');

	let bills = $state<MaintenanceBill[]>([]);
	let totalBills = $state<number>(0);
	let payments = $state<ResidentPaymentItem[]>([]);
	let totalPayments = $state<number>(0);
	let isLoading = $state<boolean>(true);
	let error = $state<string | null>(null);

	// Details Modal
	let isDetailModalOpen = $state<boolean>(false);
	let selectedBill = $state<MaintenanceBillDetail | null>(null);
	let isLoadingDetail = $state<boolean>(false);
	let detailError = $state<string | null>(null);

	// Receipt Modal
	let isReceiptModalOpen = $state<boolean>(false);
	let selectedReceipt = $state<MaintenanceReceipt | null>(null);
	let isLoadingReceipt = $state<boolean>(false);
	let receiptError = $state<string | null>(null);

	async function loadAllResidentData() {
		if (!societyId) return;
		isLoading = true;
		error = null;
		try {
			await Promise.all([loadResidentBills(), loadResidentPayments()]);
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load maintenance records';
		} finally {
			isLoading = false;
		}
	}

	async function loadResidentBills() {
		try {
			const res = await societyApi.listResidentBills(societyId);
			bills = res.bills || [];
			totalBills = res.total || 0;
		} catch (err: unknown) {
			console.error('Failed to load maintenance bills', err);
		}
	}

	async function loadResidentPayments() {
		try {
			const res = await societyApi.listResidentPayments(societyId);
			payments = res.payments || [];
			totalPayments = res.total || 0;
		} catch (err: unknown) {
			console.error('Failed to load resident payments', err);
		}
	}

	async function openBillDetail(billId: string) {
		isDetailModalOpen = true;
		isLoadingDetail = true;
		selectedBill = null;
		detailError = null;
		try {
			const res = await societyApi.getResidentBillDetails(societyId, billId);
			selectedBill = res.bill;
		} catch (err: unknown) {
			detailError = err instanceof Error ? err.message : 'Failed to load bill details';
		} finally {
			isLoadingDetail = false;
		}
	}

	async function openResidentReceipt(receiptId: string) {
		isReceiptModalOpen = true;
		isLoadingReceipt = true;
		selectedReceipt = null;
		receiptError = null;
		try {
			const res = await societyApi.getResidentReceipt(societyId, receiptId);
			selectedReceipt = res.receipt;
		} catch (err: unknown) {
			receiptError = err instanceof Error ? err.message : 'Failed to load receipt details';
		} finally {
			isLoadingReceipt = false;
		}
	}

	function formatCurrency(amount: number) {
		return new Intl.NumberFormat('en-IN', {
			style: 'currency',
			currency: 'INR',
			maximumFractionDigits: 2
		}).format(amount);
	}

	function getStatusBadge(status: string) {
		switch (status) {
			case 'ISSUED':
				return { variant: 'default' as const, label: 'Payment Due' };
			case 'OVERDUE':
				return { variant: 'destructive' as const, label: 'Overdue' };
			case 'VOIDED':
				return { variant: 'outline' as const, label: 'Voided' };
			default:
				return { variant: 'outline' as const, label: status };
		}
	}

	$effect(() => {
		if (societyId) {
			loadAllResidentData();
		}
	});
</script>

<div class="rounded-2xl border border-neutral-200 bg-white/90 p-6 shadow-xs">
	<!-- Header -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<div class="flex items-center gap-2">
				<div
					class="flex h-8 w-8 items-center justify-center rounded-full bg-neutral-100 text-neutral-900"
				>
					<Receipt class="h-4 w-4" />
				</div>
				<h2 class="text-base font-semibold tracking-tight text-neutral-950">
					Unit Maintenance Dues & Receipts
				</h2>
				<Badge variant="outline" class="font-mono text-[10px] text-neutral-700">
					{totalBills}
					{totalBills === 1 ? 'Bill' : 'Bills'} • {totalPayments}
					{totalPayments === 1 ? 'Payment' : 'Payments'}
				</Badge>
			</div>
			<p class="mt-1 text-xs text-neutral-500">
				Official maintenance assessments, itemized breakdowns, and payment receipts issued for your
				authorized unit.
			</p>
		</div>

		<div class="flex items-center gap-2">
			<Button variant="outline" size="sm" onclick={loadAllResidentData} disabled={isLoading}>
				<RefreshCw class="mr-1.5 h-3.5 w-3.5 {isLoading ? 'animate-spin' : ''}" />
				<span>Refresh</span>
			</Button>
		</div>
	</div>

	<!-- Navigation Tabs -->
	<div class="mt-6 flex border-b border-neutral-200 text-xs font-medium">
		<button
			class="flex items-center gap-2 border-b-2 px-4 py-2.5 transition-colors {activeTab === 'bills'
				? 'border-neutral-950 text-neutral-950'
				: 'border-transparent text-neutral-500 hover:text-neutral-900'}"
			onclick={() => (activeTab = 'bills')}
		>
			<FileText class="h-3.5 w-3.5" />
			<span>Assessments & Bills ({totalBills})</span>
		</button>
		<button
			class="flex items-center gap-2 border-b-2 px-4 py-2.5 transition-colors {activeTab ===
			'payments'
				? 'border-neutral-950 text-neutral-950'
				: 'border-transparent text-neutral-500 hover:text-neutral-900'}"
			onclick={() => (activeTab = 'payments')}
		>
			<Receipt class="h-3.5 w-3.5" />
			<span>Payment Receipts ({totalPayments})</span>
		</button>
	</div>

	<!-- Content Body -->
	{#if isLoading}
		<div class="flex items-center justify-center py-12 text-neutral-400">
			<Loader2 class="mr-2 h-5 w-5 animate-spin" />
			<span class="text-xs">Loading maintenance records...</span>
		</div>
	{:else if error}
		<div class="mt-4 rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800">
			<div class="flex items-center gap-2">
				<AlertCircle class="h-4 w-4 text-rose-600" />
				<span class="font-medium">Failed to load maintenance records</span>
			</div>
			<p class="mt-1 text-rose-700">{error}</p>
		</div>
	{:else if activeTab === 'bills'}
		{#if bills.length === 0}
			<div class="mt-4 rounded-xl border border-dashed border-neutral-300 p-8 text-center">
				<Receipt class="mx-auto h-8 w-8 text-neutral-400" />
				<p class="mt-2 text-sm font-medium text-neutral-900">No maintenance bills issued</p>
				<p class="mt-1 text-xs text-neutral-500">
					There are no outstanding or past maintenance bills recorded for your unit.
				</p>
			</div>
		{:else}
			<div class="mt-4 overflow-x-auto rounded-xl border border-neutral-200">
				<table class="w-full text-left text-xs">
					<thead
						class="border-b border-neutral-200 bg-neutral-50 text-[11px] font-medium text-neutral-600"
					>
						<tr>
							<th class="px-4 py-3">Bill Number</th>
							<th class="px-4 py-3">Unit / Flat</th>
							<th class="px-4 py-3">Billing Period</th>
							<th class="px-4 py-3">Issue Date</th>
							<th class="px-4 py-3">Due Date</th>
							<th class="px-4 py-3 text-right">Assessment Total</th>
							<th class="px-4 py-3 text-center">Status</th>
							<th class="px-4 py-3 text-right">Actions</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-200 bg-white">
						{#each bills as b (b.id)}
							{@const badge = getStatusBadge(b.status)}
							<tr class="hover:bg-neutral-50/60">
								<td class="px-4 py-3 font-mono font-medium text-neutral-950">{b.bill_number}</td>
								<td class="px-4 py-3 font-medium text-neutral-900">
									{b.unit_identifier || 'Unit'}
									{#if b.building_name}
										<span class="text-[10px] text-neutral-500">({b.building_name})</span>
									{/if}
								</td>
								<td class="px-4 py-3 text-neutral-600">{b.period_name || '—'}</td>
								<td class="px-4 py-3 text-neutral-600"
									>{b.bill_date ? b.bill_date.slice(0, 10) : '—'}</td
								>
								<td
									class="px-4 py-3 font-medium {b.status === 'OVERDUE'
										? 'font-bold text-rose-600'
										: 'text-neutral-700'}"
								>
									{b.due_date ? b.due_date.slice(0, 10) : '—'}
								</td>
								<td class="px-4 py-3 text-right font-semibold text-neutral-950">
									{formatCurrency(b.total_amount)}
								</td>
								<td class="px-4 py-3 text-center">
									<Badge variant={badge.variant} class="font-mono text-[10px]">
										{badge.label}
									</Badge>
								</td>
								<td class="px-4 py-3 text-right">
									<Button variant="outline" size="sm" onclick={() => openBillDetail(b.id)}>
										<Eye class="mr-1 h-3 w-3" />
										<span>Breakdown</span>
									</Button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{:else if activeTab === 'payments'}
		{#if payments.length === 0}
			<div class="mt-4 rounded-xl border border-dashed border-neutral-300 p-8 text-center">
				<Receipt class="mx-auto h-8 w-8 text-neutral-400" />
				<p class="mt-2 text-sm font-medium text-neutral-900">No payment receipts recorded</p>
				<p class="mt-1 text-xs text-neutral-500">
					Payments recorded by society administration for your unit will appear here with official
					receipts.
				</p>
			</div>
		{:else}
			<div class="mt-4 overflow-x-auto rounded-xl border border-neutral-200">
				<table class="w-full text-left text-xs">
					<thead
						class="border-b border-neutral-200 bg-neutral-50 text-[11px] font-medium text-neutral-600"
					>
						<tr>
							<th class="px-4 py-3">Receipt Number</th>
							<th class="px-4 py-3">Payment Ref</th>
							<th class="px-4 py-3">Bill Number</th>
							<th class="px-4 py-3">Unit / Flat</th>
							<th class="px-4 py-3">Payment Date</th>
							<th class="px-4 py-3">Method</th>
							<th class="px-4 py-3 text-right">Amount Paid</th>
							<th class="px-4 py-3 text-center">Status</th>
							<th class="px-4 py-3 text-right">Receipt</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-neutral-200 bg-white">
						{#each payments as p (p.payment_id)}
							<tr class="hover:bg-neutral-50/60">
								<td class="px-4 py-3 font-mono font-semibold text-neutral-950">
									{p.receipt_number}
								</td>
								<td class="px-4 py-3 font-mono text-neutral-600">{p.payment_reference}</td>
								<td class="px-4 py-3 font-mono text-neutral-700">{p.bill_number}</td>
								<td class="px-4 py-3 font-medium text-neutral-900">{p.unit_identifier}</td>
								<td class="px-4 py-3 text-neutral-600">
									{p.payment_date ? p.payment_date.slice(0, 10) : '—'}
								</td>
								<td class="px-4 py-3 font-medium text-neutral-700">{p.payment_method}</td>
								<td class="px-4 py-3 text-right font-bold text-emerald-700">
									{formatCurrency(p.amount)}
								</td>
								<td class="px-4 py-3 text-center">
									<Badge
										variant={p.status === 'REVERSED' ? 'destructive' : 'default'}
										class="font-mono text-[10px] {p.status === 'REVERSED'
											? 'border-rose-300 bg-rose-50 text-rose-700'
											: 'border-emerald-300 bg-emerald-50 text-emerald-800'}"
									>
										{p.status}
									</Badge>
								</td>
								<td class="px-4 py-3 text-right">
									<Button
										variant="outline"
										size="sm"
										onclick={() => openResidentReceipt(p.receipt_id)}
									>
										<Printer class="mr-1 h-3 w-3" />
										<span>Receipt</span>
									</Button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}
</div>

<!-- Resident Bill Details Breakdown Dialog -->
{#if isDetailModalOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div
			class="relative w-full max-w-2xl rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl"
		>
			<div class="flex items-center justify-between border-b border-neutral-200 pb-4">
				<div>
					<h3 class="text-base font-semibold text-neutral-950">Maintenance Statement</h3>
					<p class="text-xs text-neutral-500">Preserved itemized breakdown of charges.</p>
				</div>
				<button
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isDetailModalOpen = false)}
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			{#if isLoadingDetail}
				<div class="flex items-center justify-center py-12 text-neutral-400">
					<Loader2 class="mr-2 h-6 w-6 animate-spin" />
					<span class="text-xs">Loading breakdown...</span>
				</div>
			{:else if detailError}
				<div class="mt-4 rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800">
					{detailError}
				</div>
			{:else if selectedBill}
				<div class="mt-4 space-y-4">
					<!-- Meta Overview -->
					<div
						class="grid grid-cols-2 gap-3 rounded-xl border border-neutral-200 bg-neutral-50 p-3 text-xs sm:grid-cols-4"
					>
						<div>
							<span class="text-[11px] text-neutral-500">Invoice #</span>
							<p class="font-mono font-medium text-neutral-900">{selectedBill.bill_number}</p>
						</div>
						<div>
							<span class="text-[11px] text-neutral-500">Unit</span>
							<p class="font-medium text-neutral-900">{selectedBill.unit_identifier || 'Unit'}</p>
						</div>
						<div>
							<span class="text-[11px] text-neutral-500">Billing Cycle</span>
							<p class="font-medium text-neutral-900">{selectedBill.period_name || '—'}</p>
						</div>
						<div>
							<span class="text-[11px] text-neutral-500">Due Date</span>
							<p
								class="font-mono font-medium {selectedBill.status === 'OVERDUE'
									? 'font-bold text-rose-600'
									: 'text-neutral-900'}"
							>
								{selectedBill.due_date ? selectedBill.due_date.slice(0, 10) : '—'}
							</p>
						</div>
					</div>

					<!-- Itemized snapshot -->
					<div>
						<h4 class="mb-2 text-xs font-semibold text-neutral-800">Itemized Assessment Details</h4>
						<div class="overflow-x-auto rounded-xl border border-neutral-200">
							<table class="w-full text-left text-xs">
								<thead
									class="border-b border-neutral-200 bg-neutral-50 text-[11px] font-medium text-neutral-600"
								>
									<tr>
										<th class="px-4 py-2.5">Category</th>
										<th class="px-4 py-2.5">Description</th>
										<th class="px-4 py-2.5 text-right">Rate</th>
										<th class="px-4 py-2.5 text-right">Qty</th>
										<th class="px-4 py-2.5 text-right">Line Total</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-neutral-200 bg-white">
									{#each selectedBill.items as item (item.id)}
										<tr>
											<td class="px-4 py-2.5 font-medium text-neutral-900">{item.category_name}</td>
											<td class="px-4 py-2.5 text-neutral-600">{item.description}</td>
											<td class="px-4 py-2.5 text-right font-mono text-neutral-700"
												>{formatCurrency(item.rate)}</td
											>
											<td class="px-4 py-2.5 text-right font-mono text-neutral-700"
												>{item.quantity}</td
											>
											<td class="px-4 py-2.5 text-right font-mono font-semibold text-neutral-950">
												{formatCurrency(item.amount)}
											</td>
										</tr>
									{/each}
								</tbody>
								<tfoot class="border-t-2 border-neutral-200 bg-neutral-50/70 text-xs font-semibold">
									<tr>
										<td colspan="4" class="px-4 py-3 text-right text-neutral-700"
											>Total Obligation ({selectedBill.currency}):</td
										>
										<td class="px-4 py-3 text-right font-mono text-sm text-neutral-950">
											{formatCurrency(selectedBill.total_amount)}
										</td>
									</tr>
								</tfoot>
							</table>
						</div>
					</div>

					{#if selectedBill.status === 'OVERDUE'}
						<div
							class="flex items-center gap-2 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800"
						>
							<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
							<span
								>This maintenance bill is past its due date ({selectedBill.due_date
									? selectedBill.due_date.slice(0, 10)
									: ''}).</span
							>
						</div>
					{:else if selectedBill.status === 'VOIDED'}
						<div
							class="rounded-xl border border-neutral-200 bg-neutral-50 p-3 text-xs text-neutral-600"
						>
							<strong>Bill Voided:</strong> This statement has been formally voided by administration.
						</div>
					{/if}
				</div>
			{/if}

			<div class="mt-6 flex justify-end">
				<Button variant="outline" size="sm" onclick={() => (isDetailModalOpen = false)}>
					<span>Close</span>
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- Resident Printable Receipt Modal (S12) -->
{#if isReceiptModalOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div class="w-full max-w-md rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl">
			<div class="flex items-center justify-between border-b border-neutral-200 pb-3">
				<div class="flex items-center gap-2">
					<Receipt class="h-4 w-4 text-emerald-700" />
					<h3 class="text-sm font-semibold text-neutral-950">Official Payment Receipt</h3>
				</div>
				<button
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isReceiptModalOpen = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if isLoadingReceipt}
				<div class="flex items-center justify-center py-12 text-neutral-400">
					<Loader2 class="mr-2 h-5 w-5 animate-spin" />
					<span class="text-xs">Loading receipt details...</span>
				</div>
			{:else if receiptError}
				<div class="mt-3 rounded-lg border border-rose-200 bg-rose-50 p-2.5 text-xs text-rose-700">
					{receiptError}
				</div>
			{:else if selectedReceipt}
				<!-- Clean printable receipt voucher -->
				<div
					class="mt-4 rounded-xl border border-neutral-200 bg-neutral-50/50 p-4 font-sans text-xs print:border-none print:p-0"
				>
					<!-- Society Header -->
					<div class="border-b border-neutral-200 pb-3 text-center">
						<h4 class="text-sm font-bold tracking-wide text-neutral-950 uppercase">
							{selectedReceipt.society_name || 'Co-operative Housing Society'}
						</h4>
						<p class="font-mono text-[10px] text-neutral-500">
							Code: {selectedReceipt.society_code || 'SOC'} • Official Maintenance Receipt
						</p>
					</div>

					<!-- Receipt Numbers & Status -->
					<div class="mt-3 flex items-center justify-between">
						<div>
							<span class="text-[10px] tracking-wider text-neutral-400 uppercase">Receipt No</span>
							<p class="font-mono text-xs font-bold text-neutral-900">
								{selectedReceipt.receipt_number}
							</p>
						</div>
						<div class="text-right">
							<Badge
								variant={selectedReceipt.status === 'REVERSED' ? 'destructive' : 'default'}
								class="font-mono text-[10px] {selectedReceipt.status === 'REVERSED'
									? 'border-rose-300 bg-rose-50 text-rose-700'
									: 'border-emerald-300 bg-emerald-50 text-emerald-800'}"
							>
								{selectedReceipt.status}
							</Badge>
						</div>
					</div>

					<!-- Details Grid -->
					<div
						class="mt-3 divide-y divide-neutral-200/60 rounded-lg border border-neutral-200/80 bg-white p-3 text-xs"
					>
						<div class="flex justify-between py-1.5">
							<span class="text-neutral-500">Date:</span>
							<span class="font-medium text-neutral-900"
								>{selectedReceipt.receipt_date
									? selectedReceipt.receipt_date.slice(0, 10)
									: '—'}</span
							>
						</div>
						<div class="flex justify-between py-1.5">
							<span class="text-neutral-500">Payment Reference:</span>
							<span class="font-mono font-medium text-neutral-900"
								>{selectedReceipt.payment_reference || '—'}</span
							>
						</div>
						<div class="flex justify-between py-1.5">
							<span class="text-neutral-500">Bill Number:</span>
							<span class="font-mono font-medium text-neutral-900"
								>{selectedReceipt.bill_number || '—'}</span
							>
						</div>
						<div class="flex justify-between py-1.5">
							<span class="text-neutral-500">Unit / Flat:</span>
							<span class="font-medium text-neutral-900"
								>{selectedReceipt.unit_identifier || '—'}</span
							>
						</div>
						<div class="flex justify-between py-1.5">
							<span class="text-neutral-500">Received From:</span>
							<span class="font-medium text-neutral-900"
								>{selectedReceipt.payer_name || 'Resident'}</span
							>
						</div>
						<div class="flex justify-between py-1.5">
							<span class="text-neutral-500">Method:</span>
							<span class="font-medium text-neutral-900"
								>{selectedReceipt.payment_method || '—'}</span
							>
						</div>
						{#if selectedReceipt.external_reference}
							<div class="flex justify-between py-1.5">
								<span class="text-neutral-500">Instrument / Ref:</span>
								<span class="font-mono font-medium text-neutral-900"
									>{selectedReceipt.external_reference}</span
								>
							</div>
						{/if}
						<div
							class="flex justify-between border-t border-neutral-300 py-2 pt-2 text-sm font-bold"
						>
							<span class="text-neutral-900">Total Received:</span>
							<span class="text-emerald-700">{formatCurrency(selectedReceipt.total_amount)}</span>
						</div>
					</div>

					{#if selectedReceipt.status === 'REVERSED'}
						<div
							class="mt-3 rounded-lg border border-rose-200 bg-rose-50 p-2 text-[11px] text-rose-800"
						>
							<strong>Reversal Notice:</strong> This collection record has been reversed. The original
							receipt is preserved for audit trail.
						</div>
					{/if}
				</div>

				<div class="mt-5 flex justify-end gap-2">
					<Button variant="outline" size="sm" onclick={() => (isReceiptModalOpen = false)}>
						<span>Close</span>
					</Button>
					<Button variant="primary" size="sm" onclick={() => window.print()}>
						<Printer class="mr-1.5 h-3.5 w-3.5" />
						<span>Print Receipt</span>
					</Button>
				</div>
			{/if}
		</div>
	</div>
{/if}
