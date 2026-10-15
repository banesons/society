<script lang="ts">
	import {
		societyApi,
		type MaintenanceChargeCategory,
		type MaintenanceChargeConfig,
		type MaintenanceBillingPeriod,
		type MaintenanceBill,
		type MaintenanceBillDetail,
		type BillGenerationPreview,
		type BillGenerationResult,
		type CollectionRegisterItem,
		type CollectionSummary,
		type MaintenanceReceipt,
		type PaymentMethod
	} from '$lib/api/society';
	import Badge from '$lib/components/ui/badge.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import {
		Wrench,
		Plus,
		FileText,
		Calendar,
		Coins,
		Tag,
		CheckCircle2,
		Loader2,
		RefreshCw,
		X,
		Eye,
		Ban,
		CreditCard,
		Receipt,
		RotateCcw,
		Printer,
		Search
	} from '@lucide/svelte';

	interface Props {
		societyId: string;
	}

	let { societyId }: Props = $props();

	// Active tab: 'collections' | 'bills' | 'periods' | 'configs' | 'categories'
	let activeTab = $state<'collections' | 'bills' | 'periods' | 'configs' | 'categories'>(
		'collections'
	);

	// Global / Tab loading & error
	let isLoading = $state<boolean>(true);
	let error = $state<string | null>(null);

	// Data states
	let bills = $state<MaintenanceBill[]>([]);
	let totalBills = $state<number>(0);
	let periods = $state<MaintenanceBillingPeriod[]>([]);
	let configs = $state<MaintenanceChargeConfig[]>([]);
	let categories = $state<MaintenanceChargeCategory[]>([]);

	// Collections Register states
	let collections = $state<CollectionRegisterItem[]>([]);
	let totalCollections = $state<number>(0);
	let collectionSummary = $state<CollectionSummary | null>(null);
	let collPeriodFilter = $state<string>('');
	let collStatusFilter = $state<string>('');
	let collSearch = $state<string>('');
	let isLoadingCollections = $state<boolean>(false);

	// Record Payment modal
	let isRecordPaymentModalOpen = $state<boolean>(false);
	let recordingBill = $state<CollectionRegisterItem | null>(null);
	let payAmount = $state<number>(0);
	let payMethod = $state<PaymentMethod>('BANK_TRANSFER');
	let payDate = $state<string>(new Date().toISOString().slice(0, 10));
	let payExternalRef = $state<string>('');
	let payPayerName = $state<string>('');
	let payNotes = $state<string>('');
	let isRecordingPayment = $state<boolean>(false);
	let recordPaymentError = $state<string | null>(null);

	// Printable Receipt modal
	let isReceiptModalOpen = $state<boolean>(false);
	let selectedReceipt = $state<MaintenanceReceipt | null>(null);
	let isLoadingReceipt = $state<boolean>(false);
	let receiptError = $state<string | null>(null);

	// Reverse Payment modal
	let isReversalModalOpen = $state<boolean>(false);
	let reversingPaymentId = $state<string | null>(null);
	let reversingPaymentRef = $state<string>('');
	let reversingBillNumber = $state<string>('');
	let reversalReason = $state<string>('');
	let isReversing = $state<boolean>(false);
	let reversalError = $state<string | null>(null);

	// Bills filter
	let billPeriodFilter = $state<string>('');
	let billStatusFilter = $state<string>('');

	// Bill Details modal
	let isDetailModalOpen = $state<boolean>(false);
	let selectedBill = $state<MaintenanceBillDetail | null>(null);
	let isLoadingDetail = $state<boolean>(false);

	// Void Bill modal
	let isVoidModalOpen = $state<boolean>(false);
	let voidingBillId = $state<string | null>(null);
	let voidReason = $state<string>('');
	let isVoiding = $state<boolean>(false);
	let voidError = $state<string | null>(null);

	// New Category modal
	let isCategoryModalOpen = $state<boolean>(false);
	let categoryName = $state<string>('');
	let categoryDescription = $state<string>('');
	let isSavingCategory = $state<boolean>(false);
	let categoryError = $state<string | null>(null);

	// New Config modal
	let isConfigModalOpen = $state<boolean>(false);
	let configCategoryId = $state<string>('');
	let configName = $state<string>('');
	let configChargeType = $state<'FLAT_RECURRING' | 'PER_UNIT_RECURRING' | 'ONE_TIME'>(
		'FLAT_RECURRING'
	);
	let configAmount = $state<number>(0);
	let configFrequency = $state<'MONTHLY' | 'QUARTERLY' | 'ANNUAL' | 'ONE_TIME'>('MONTHLY');
	let configEffectiveFrom = $state<string>(new Date().toISOString().slice(0, 10));
	let isSavingConfig = $state<boolean>(false);
	let configError = $state<string | null>(null);

	// New Period modal
	let isPeriodModalOpen = $state<boolean>(false);
	let periodName = $state<string>('');
	let periodStart = $state<string>(new Date().toISOString().slice(0, 10));
	let periodEnd = $state<string>(new Date(Date.now() + 30 * 86400000).toISOString().slice(0, 10));
	let periodDueDate = $state<string>(
		new Date(Date.now() + 40 * 86400000).toISOString().slice(0, 10)
	);
	let isSavingPeriod = $state<boolean>(false);
	let periodError = $state<string | null>(null);

	// Bill Generation modal
	let isGenerateModalOpen = $state<boolean>(false);
	let generatingPeriodId = $state<string>('');
	let generationPreview = $state<BillGenerationPreview | null>(null);
	let generationResult = $state<BillGenerationResult | null>(null);
	let isLoadingPreview = $state<boolean>(false);
	let isGenerating = $state<boolean>(false);
	let generationError = $state<string | null>(null);

	// Fetch handlers
	async function loadAllData() {
		if (!societyId) return;
		isLoading = true;
		error = null;
		try {
			await Promise.all([
				loadCollections(),
				loadBills(),
				loadPeriods(),
				loadConfigs(),
				loadCategories()
			]);
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to load maintenance data';
		} finally {
			isLoading = false;
		}
	}

	async function loadCollections() {
		isLoadingCollections = true;
		try {
			const [regRes, sumRes] = await Promise.all([
				societyApi.listCollectionRegister(societyId, {
					period_id: collPeriodFilter || undefined,
					status: collStatusFilter || undefined,
					search: collSearch || undefined,
					limit: 50
				}),
				societyApi.getCollectionSummary(societyId, collPeriodFilter || undefined)
			]);
			collections = regRes.items || [];
			totalCollections = regRes.total || 0;
			collectionSummary = sumRes.summary || null;
		} catch (err: unknown) {
			console.error('Failed to load collections', err);
		} finally {
			isLoadingCollections = false;
		}
	}

	async function loadBills() {
		try {
			const res = await societyApi.listBills(societyId, {
				period_id: billPeriodFilter || undefined,
				status: billStatusFilter || undefined,
				limit: 50
			});
			bills = res.items || [];
			totalBills = res.total || 0;
		} catch (err: unknown) {
			console.error('Failed to load bills', err);
		}
	}

	async function loadPeriods() {
		try {
			const res = await societyApi.listBillingPeriods(societyId);
			periods = res.periods || [];
		} catch (err: unknown) {
			console.error('Failed to load periods', err);
		}
	}

	async function loadConfigs() {
		try {
			const res = await societyApi.listChargeConfigs(societyId);
			configs = res.configs || [];
		} catch (err: unknown) {
			console.error('Failed to load configs', err);
		}
	}

	async function loadCategories() {
		try {
			const res = await societyApi.listChargeCategories(societyId);
			categories = res.categories || [];
		} catch (err: unknown) {
			console.error('Failed to load categories', err);
		}
	}

	// Action: Open Bill Details
	async function openBillDetail(billId: string) {
		isDetailModalOpen = true;
		isLoadingDetail = true;
		selectedBill = null;
		try {
			const res = await societyApi.getBillDetails(societyId, billId);
			selectedBill = res.bill;
		} catch (err: unknown) {
			console.error('Failed to load bill detail', err);
		} finally {
			isLoadingDetail = false;
		}
	}

	// Action: Open Void Modal
	function openVoidModal(billId: string) {
		voidingBillId = billId;
		voidReason = '';
		voidError = null;
		isVoidModalOpen = true;
	}

	async function handleVoidBill() {
		if (!voidingBillId || !voidReason.trim()) {
			voidError = 'Please provide a valid void reason';
			return;
		}
		isVoiding = true;
		voidError = null;
		try {
			await societyApi.voidBill(societyId, voidingBillId, voidReason.trim());
			isVoidModalOpen = false;
			await loadBills();
			if (selectedBill && selectedBill.id === voidingBillId) {
				await openBillDetail(voidingBillId);
			}
		} catch (err: unknown) {
			voidError = err instanceof Error ? err.message : 'Failed to void bill';
		} finally {
			isVoiding = false;
		}
	}

	// Action: Create Category
	async function handleCreateCategory() {
		if (!categoryName.trim()) {
			categoryError = 'Category name is required';
			return;
		}
		isSavingCategory = true;
		categoryError = null;
		try {
			await societyApi.createChargeCategory(societyId, {
				name: categoryName.trim(),
				description: categoryDescription.trim() || undefined
			});
			isCategoryModalOpen = false;
			categoryName = '';
			categoryDescription = '';
			await loadCategories();
		} catch (err: unknown) {
			categoryError = err instanceof Error ? err.message : 'Failed to create category';
		} finally {
			isSavingCategory = false;
		}
	}

	// Action: Create Config
	async function handleCreateConfig() {
		if (!configCategoryId) {
			configError = 'Please select a charge category';
			return;
		}
		if (!configName.trim()) {
			configError = 'Configuration name is required';
			return;
		}
		if (configAmount <= 0) {
			configError = 'Amount must be strictly greater than zero';
			return;
		}
		isSavingConfig = true;
		configError = null;
		try {
			await societyApi.createChargeConfig(societyId, {
				category_id: configCategoryId,
				name: configName.trim(),
				charge_type: configChargeType,
				amount: Number(configAmount),
				billing_frequency: configFrequency,
				effective_from: configEffectiveFrom
			});
			isConfigModalOpen = false;
			configName = '';
			configAmount = 0;
			await loadConfigs();
		} catch (err: unknown) {
			configError = err instanceof Error ? err.message : 'Failed to create configuration';
		} finally {
			isSavingConfig = false;
		}
	}

	// Action: Deactivate Config
	async function handleDeactivateConfig(configId: string) {
		if (
			!confirm(
				'Are you sure you want to deactivate this charge configuration? Future bills will not include it.'
			)
		) {
			return;
		}
		try {
			await societyApi.deactivateChargeConfig(societyId, configId);
			await loadConfigs();
		} catch (err: unknown) {
			alert(err instanceof Error ? err.message : 'Failed to deactivate configuration');
		}
	}

	// Action: Create Period
	async function handleCreatePeriod() {
		if (!periodName.trim()) {
			periodError = 'Period name is required';
			return;
		}
		if (periodEnd < periodStart) {
			periodError = 'End date must be on or after start date';
			return;
		}
		if (periodDueDate < periodEnd) {
			periodError = 'Due date must be on or after period end date';
			return;
		}
		isSavingPeriod = true;
		periodError = null;
		try {
			await societyApi.createBillingPeriod(societyId, {
				period_name: periodName.trim(),
				period_start: periodStart,
				period_end: periodEnd,
				due_date: periodDueDate
			});
			isPeriodModalOpen = false;
			periodName = '';
			await loadPeriods();
		} catch (err: unknown) {
			periodError = err instanceof Error ? err.message : 'Failed to create billing period';
		} finally {
			isSavingPeriod = false;
		}
	}

	// Action: Open Generation Modal
	async function openGenerateModal(periodId: string) {
		generatingPeriodId = periodId;
		generationPreview = null;
		generationResult = null;
		generationError = null;
		isGenerateModalOpen = true;
		isLoadingPreview = true;
		try {
			const res = await societyApi.previewBillGeneration(societyId, periodId);
			generationPreview = res.preview;
		} catch (err: unknown) {
			generationError = err instanceof Error ? err.message : 'Failed to preview bill generation';
		} finally {
			isLoadingPreview = false;
		}
	}

	// Action: Execute Bill Generation
	async function handleExecuteGeneration() {
		if (!generatingPeriodId) return;
		isGenerating = true;
		generationError = null;
		try {
			const res = await societyApi.generateBills(societyId, generatingPeriodId);
			generationResult = res.result;
			await loadBills();
			await loadPeriods();
		} catch (err: unknown) {
			generationError = err instanceof Error ? err.message : 'Failed to generate bills';
		} finally {
			isGenerating = false;
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
				return { variant: 'default' as const, label: 'Issued' };
			case 'OVERDUE':
				return { variant: 'destructive' as const, label: 'Overdue' };
			case 'VOIDED':
				return { variant: 'outline' as const, label: 'Voided' };
			default:
				return { variant: 'outline' as const, label: status };
		}
	}

	function getCollectionStatusBadge(status: string) {
		switch (status) {
			case 'PAID':
				return {
					variant: 'default' as const,
					label: 'Paid',
					badgeClass: 'border-emerald-300 bg-emerald-50 text-emerald-800'
				};
			case 'PARTIALLY_PAID':
				return {
					variant: 'outline' as const,
					label: 'Partially Paid',
					badgeClass: 'border-amber-300 bg-amber-50 text-amber-800'
				};
			case 'UNPAID':
				return {
					variant: 'destructive' as const,
					label: 'Unpaid',
					badgeClass: 'border-rose-300 bg-rose-50 text-rose-800'
				};
			case 'VOIDED':
				return {
					variant: 'outline' as const,
					label: 'Voided',
					badgeClass: 'border-neutral-200 bg-neutral-100 text-neutral-500'
				};
			default:
				return { variant: 'outline' as const, label: status, badgeClass: '' };
		}
	}

	function openRecordPaymentModal(item: CollectionRegisterItem) {
		recordingBill = item;
		payAmount = item.outstanding_balance;
		payMethod = 'BANK_TRANSFER';
		payDate = new Date().toISOString().slice(0, 10);
		payExternalRef = '';
		payPayerName = '';
		payNotes = '';
		recordPaymentError = null;
		isRecordPaymentModalOpen = true;
	}

	async function handleRecordPayment() {
		if (!recordingBill) return;
		if (payAmount <= 0) {
			recordPaymentError = 'Payment amount must be greater than zero';
			return;
		}
		if (payAmount > recordingBill.outstanding_balance) {
			recordPaymentError = `Payment amount cannot exceed outstanding balance (${formatCurrency(recordingBill.outstanding_balance)})`;
			return;
		}
		isRecordingPayment = true;
		recordPaymentError = null;
		try {
			const idempotencyKey = crypto.randomUUID();
			const result = await societyApi.recordPayment(
				societyId,
				{
					bill_id: recordingBill.bill_id,
					amount: Number(payAmount),
					payment_method: payMethod,
					payment_date: payDate,
					external_reference: payExternalRef.trim() || undefined,
					payer_name: payPayerName.trim() || undefined,
					notes: payNotes.trim() || undefined
				},
				idempotencyKey
			);

			isRecordPaymentModalOpen = false;
			selectedReceipt = result.receipt;
			isReceiptModalOpen = true;

			await Promise.all([loadCollections(), loadBills()]);
		} catch (err: unknown) {
			recordPaymentError = err instanceof Error ? err.message : 'Failed to record payment';
		} finally {
			isRecordingPayment = false;
		}
	}

	async function openReceiptModal(receiptId: string) {
		isReceiptModalOpen = true;
		isLoadingReceipt = true;
		selectedReceipt = null;
		receiptError = null;
		try {
			const res = await societyApi.getReceiptDetails(societyId, receiptId);
			selectedReceipt = res.receipt;
		} catch (err: unknown) {
			receiptError = err instanceof Error ? err.message : 'Failed to load receipt';
		} finally {
			isLoadingReceipt = false;
		}
	}

	function openReversalModal(paymentId: string, paymentRef: string, billNumber: string) {
		reversingPaymentId = paymentId;
		reversingPaymentRef = paymentRef;
		reversingBillNumber = billNumber;
		reversalReason = '';
		reversalError = null;
		isReversalModalOpen = true;
	}

	async function handleReversePayment() {
		if (!reversingPaymentId || !reversalReason.trim()) {
			reversalError = 'A non-empty reversal reason is mandatory';
			return;
		}
		isReversing = true;
		reversalError = null;
		try {
			await societyApi.reversePayment(societyId, reversingPaymentId, reversalReason.trim());
			isReversalModalOpen = false;
			await Promise.all([loadCollections(), loadBills()]);
			if (selectedBill) {
				await openBillDetail(selectedBill.id);
			}
		} catch (err: unknown) {
			reversalError = err instanceof Error ? err.message : 'Failed to reverse payment';
		} finally {
			isReversing = false;
		}
	}

	$effect(() => {
		if (societyId) {
			loadAllData();
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
					<Wrench class="h-4 w-4" />
				</div>
				<h2 class="text-base font-semibold tracking-tight text-neutral-950">
					Maintenance & Billing Console
				</h2>
				<Badge
					variant="outline"
					class="border-emerald-200 bg-emerald-50 font-mono text-[10px] text-emerald-800"
				>
					S11 & S12 Collections
				</Badge>
			</div>
			<p class="mt-1 text-xs text-neutral-500">
				Manage society charge categories, recurring fee structures, billing periods, idempotent unit
				bill generation, and collection recording.
			</p>
		</div>

		<div class="flex items-center gap-2">
			<Button variant="outline" size="sm" onclick={loadAllData} disabled={isLoading}>
				<RefreshCw class="mr-1.5 h-3.5 w-3.5 {isLoading ? 'animate-spin' : ''}" />
				<span>Refresh</span>
			</Button>

			{#if activeTab === 'categories'}
				<Button variant="primary" size="sm" onclick={() => (isCategoryModalOpen = true)}>
					<Plus class="mr-1.5 h-3.5 w-3.5" />
					<span>New Category</span>
				</Button>
			{:else if activeTab === 'configs'}
				<Button variant="primary" size="sm" onclick={() => (isConfigModalOpen = true)}>
					<Plus class="mr-1.5 h-3.5 w-3.5" />
					<span>Add Charge Config</span>
				</Button>
			{:else if activeTab === 'periods'}
				<Button variant="primary" size="sm" onclick={() => (isPeriodModalOpen = true)}>
					<Plus class="mr-1.5 h-3.5 w-3.5" />
					<span>New Billing Period</span>
				</Button>
			{/if}
		</div>
	</div>

	{#if error}
		<div class="mt-4 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800">
			{error}
		</div>
	{/if}

	<!-- Navigation Tabs -->
	<div class="mt-6 flex border-b border-neutral-200 text-xs font-medium">
		<button
			class="flex items-center gap-2 border-b-2 px-4 py-2.5 transition-colors {activeTab ===
			'collections'
				? 'border-neutral-950 text-neutral-950'
				: 'border-transparent text-neutral-500 hover:text-neutral-900'}"
			onclick={() => (activeTab = 'collections')}
		>
			<Receipt class="h-3.5 w-3.5" />
			<span>Collections Register ({totalCollections})</span>
		</button>
		<button
			class="flex items-center gap-2 border-b-2 px-4 py-2.5 transition-colors {activeTab === 'bills'
				? 'border-neutral-950 text-neutral-950'
				: 'border-transparent text-neutral-500 hover:text-neutral-900'}"
			onclick={() => (activeTab = 'bills')}
		>
			<FileText class="h-3.5 w-3.5" />
			<span>Bills Register ({totalBills})</span>
		</button>
		<button
			class="flex items-center gap-2 border-b-2 px-4 py-2.5 transition-colors {activeTab ===
			'periods'
				? 'border-neutral-950 text-neutral-950'
				: 'border-transparent text-neutral-500 hover:text-neutral-900'}"
			onclick={() => (activeTab = 'periods')}
		>
			<Calendar class="h-3.5 w-3.5" />
			<span>Billing Periods ({periods.length})</span>
		</button>
		<button
			class="flex items-center gap-2 border-b-2 px-4 py-2.5 transition-colors {activeTab ===
			'configs'
				? 'border-neutral-950 text-neutral-950'
				: 'border-transparent text-neutral-500 hover:text-neutral-900'}"
			onclick={() => (activeTab = 'configs')}
		>
			<Coins class="h-3.5 w-3.5" />
			<span>Charge Configs ({configs.length})</span>
		</button>
		<button
			class="flex items-center gap-2 border-b-2 px-4 py-2.5 transition-colors {activeTab ===
			'categories'
				? 'border-neutral-950 text-neutral-950'
				: 'border-transparent text-neutral-500 hover:text-neutral-900'}"
			onclick={() => (activeTab = 'categories')}
		>
			<Tag class="h-3.5 w-3.5" />
			<span>Categories ({categories.length})</span>
		</button>
	</div>

	<!-- Tab 0: Collections Register (S12) -->
	{#if activeTab === 'collections'}
		<div class="mt-4 space-y-4">
			<!-- Summary KPI Cards -->
			{#if collectionSummary}
				<div class="grid grid-cols-2 gap-3 text-xs sm:grid-cols-4">
					<div class="rounded-xl border border-neutral-200 bg-white p-3 shadow-2xs">
						<span class="text-[11px] font-medium text-neutral-500">Total Billed</span>
						<p class="mt-1 text-base font-bold text-neutral-950">
							{formatCurrency(collectionSummary.total_billed)}
						</p>
						<p class="mt-0.5 text-[10px] text-neutral-400">Total assessments in period</p>
					</div>
					<div class="rounded-xl border border-emerald-200 bg-emerald-50/50 p-3 shadow-2xs">
						<span class="text-[11px] font-medium text-emerald-800">Recorded Collections</span>
						<p class="mt-1 text-base font-bold text-emerald-700">
							{formatCurrency(collectionSummary.total_collected)}
						</p>
						<p class="mt-0.5 text-[10px] text-emerald-600">Active allocated payments</p>
					</div>
					<div class="rounded-xl border border-rose-200 bg-rose-50/50 p-3 shadow-2xs">
						<span class="text-[11px] font-medium text-rose-800">Outstanding Balance</span>
						<p class="mt-1 text-base font-bold text-rose-700">
							{formatCurrency(collectionSummary.total_outstanding)}
						</p>
						<p class="mt-0.5 text-[10px] text-rose-600">Pending on non-voided bills</p>
					</div>
					<div class="rounded-xl border border-neutral-200 bg-neutral-50 p-3 shadow-2xs">
						<span class="text-[11px] font-medium text-neutral-500">Collection Status Counts</span>
						<div class="mt-1 flex items-center gap-2 text-xs font-semibold">
							<span class="text-rose-600" title="Unpaid"
								>{collectionSummary.unpaid_bills_count} Unpaid</span
							>
							<span class="text-neutral-300">•</span>
							<span class="text-amber-600" title="Partially Paid"
								>{collectionSummary.partially_paid_bills_count} Partial</span
							>
							<span class="text-neutral-300">•</span>
							<span class="text-emerald-600" title="Paid"
								>{collectionSummary.paid_bills_count} Paid</span
							>
						</div>
						{#if collectionSummary.reversed_payments_count > 0}
							<p class="mt-0.5 text-[10px] text-neutral-400">
								{collectionSummary.reversed_payments_count} reversed ({formatCurrency(
									collectionSummary.reversed_amount
								)})
							</p>
						{:else}
							<p class="mt-0.5 text-[10px] text-neutral-400">0 reversals</p>
						{/if}
					</div>
				</div>
			{/if}

			<!-- Filters -->
			<div
				class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-200 bg-neutral-50/70 p-3"
			>
				<div class="flex flex-wrap items-center gap-3">
					<div class="flex items-center gap-2">
						<label for="filter-coll-period" class="text-xs text-neutral-500">Period:</label>
						<select
							id="filter-coll-period"
							class="rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
							bind:value={collPeriodFilter}
							onchange={loadCollections}
						>
							<option value="">All Periods</option>
							{#each periods as p (p.id)}
								<option value={p.id}>{p.period_name}</option>
							{/each}
						</select>
					</div>

					<div class="flex items-center gap-2">
						<label for="filter-coll-status" class="text-xs text-neutral-500"
							>Collection Status:</label
						>
						<select
							id="filter-coll-status"
							class="rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
							bind:value={collStatusFilter}
							onchange={loadCollections}
						>
							<option value="">All Statuses</option>
							<option value="UNPAID">Unpaid</option>
							<option value="PARTIALLY_PAID">Partially Paid</option>
							<option value="PAID">Paid</option>
							<option value="VOIDED">Voided</option>
						</select>
					</div>

					<div class="flex items-center gap-1.5">
						<div class="relative">
							<Search
								class="pointer-events-none absolute top-2 left-2.5 h-3.5 w-3.5 text-neutral-400"
							/>
							<input
								type="text"
								placeholder="Search bill # or unit..."
								class="rounded-lg border border-neutral-300 bg-white py-1.5 pr-3 pl-8 text-xs text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-950 focus:outline-none"
								bind:value={collSearch}
								onkeydown={(e) => {
									if (e.key === 'Enter') loadCollections();
								}}
							/>
						</div>
						<Button variant="outline" size="sm" onclick={loadCollections}>
							<span>Search</span>
						</Button>
					</div>
				</div>

				<div class="font-mono text-[11px] text-neutral-500">
					{totalCollections}
					{totalCollections === 1 ? 'record' : 'records'}
				</div>
			</div>

			<!-- Register Table -->
			{#if isLoadingCollections}
				<div class="flex items-center justify-center py-12 text-neutral-400">
					<Loader2 class="mr-2 h-5 w-5 animate-spin" />
					<span class="text-xs">Loading collection register...</span>
				</div>
			{:else if collections.length === 0}
				<div class="rounded-xl border border-dashed border-neutral-300 p-8 text-center">
					<Receipt class="mx-auto h-8 w-8 text-neutral-400" />
					<p class="mt-2 text-sm font-medium text-neutral-900">No collection records found</p>
					<p class="mt-1 text-xs text-neutral-500">
						Generate bills for an active period to start tracking collections and payments.
					</p>
				</div>
			{:else}
				<div class="overflow-x-auto rounded-xl border border-neutral-200">
					<table class="w-full text-left text-xs">
						<thead
							class="border-b border-neutral-200 bg-neutral-50 text-[11px] font-medium text-neutral-600"
						>
							<tr>
								<th class="px-4 py-3">Bill Number</th>
								<th class="px-4 py-3">Unit / Flat</th>
								<th class="px-4 py-3">Period</th>
								<th class="px-4 py-3">Due Date</th>
								<th class="px-4 py-3 text-right">Billed Total</th>
								<th class="px-4 py-3 text-right">Paid Amount</th>
								<th class="px-4 py-3 text-right">Outstanding</th>
								<th class="px-4 py-3 text-center">Collection Status</th>
								<th class="px-4 py-3 text-right">Actions</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-200 bg-white">
							{#each collections as item (item.bill_id)}
								{@const collBadge = getCollectionStatusBadge(item.collection_status)}
								<tr class="hover:bg-neutral-50/60">
									<td class="px-4 py-3 font-mono font-medium text-neutral-950">
										<button
											class="font-semibold text-neutral-900 underline-offset-2 hover:text-neutral-950 hover:underline"
											onclick={() => openBillDetail(item.bill_id)}
										>
											{item.bill_number}
										</button>
										{#if item.bill_status === 'VOIDED'}
											<span class="ml-1 text-[10px] text-neutral-400">(Voided)</span>
										{/if}
									</td>
									<td class="px-4 py-3 font-medium text-neutral-900">
										{item.unit_identifier || 'Unit'}
									</td>
									<td class="px-4 py-3 text-neutral-600">{item.period_name}</td>
									<td class="px-4 py-3 font-mono text-neutral-600">
										{item.due_date ? item.due_date.slice(0, 10) : '—'}
									</td>
									<td class="px-4 py-3 text-right font-medium text-neutral-900">
										{formatCurrency(item.total_amount)}
									</td>
									<td class="px-4 py-3 text-right font-semibold text-emerald-700">
										{formatCurrency(item.total_paid)}
									</td>
									<td
										class="px-4 py-3 text-right font-bold {item.outstanding_balance > 0
											? 'text-rose-600'
											: 'text-neutral-500'}"
									>
										{formatCurrency(item.outstanding_balance)}
									</td>
									<td class="px-4 py-3 text-center">
										<Badge
											variant={collBadge.variant}
											class="font-mono text-[10px] {collBadge.badgeClass}"
										>
											{collBadge.label}
										</Badge>
									</td>
									<td class="px-4 py-3 text-right">
										<div class="flex items-center justify-end gap-1.5">
											{#if item.outstanding_balance > 0 && item.bill_status !== 'VOIDED'}
												<Button
													variant="primary"
													size="sm"
													onclick={() => openRecordPaymentModal(item)}
												>
													<CreditCard class="mr-1 h-3 w-3" />
													<span>Record Payment</span>
												</Button>
											{/if}
											{#if item.total_paid > 0}
												<Button
													variant="outline"
													size="sm"
													onclick={() => openReceiptModal(item.bill_id)}
													title="View Receipt"
												>
													<Receipt class="mr-1 h-3 w-3" />
													<span>Receipt</span>
												</Button>
												<Button
													variant="outline"
													size="sm"
													class="text-rose-600 hover:text-rose-700"
													onclick={() =>
														openReversalModal(item.bill_id, item.bill_number, item.bill_number)}
													title="Reverse payment record"
												>
													<RotateCcw class="mr-1 h-3 w-3" />
													<span>Reverse</span>
												</Button>
											{/if}
											<Button
												variant="outline"
												size="sm"
												onclick={() => openBillDetail(item.bill_id)}
											>
												<Eye class="mr-1 h-3 w-3" />
												<span>Details</span>
											</Button>
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Tab 1: Bills Register -->
	{#if activeTab === 'bills'}
		<div class="mt-4">
			<!-- Filters -->
			<div class="mb-4 flex flex-wrap items-center gap-3">
				<div class="flex items-center gap-2">
					<label for="filter-period" class="text-xs text-neutral-500">Period:</label>
					<select
						id="filter-period"
						class="rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
						bind:value={billPeriodFilter}
						onchange={loadBills}
					>
						<option value="">All Periods</option>
						{#each periods as p (p.id)}
							<option value={p.id}>{p.period_name}</option>
						{/each}
					</select>
				</div>

				<div class="flex items-center gap-2">
					<label for="filter-status" class="text-xs text-neutral-500">Status:</label>
					<select
						id="filter-status"
						class="rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
						bind:value={billStatusFilter}
						onchange={loadBills}
					>
						<option value="">All Statuses</option>
						<option value="ISSUED">Issued</option>
						<option value="OVERDUE">Overdue</option>
						<option value="VOIDED">Voided</option>
					</select>
				</div>
			</div>

			<!-- Bills Table -->
			{#if isLoading}
				<div class="flex items-center justify-center py-12 text-neutral-400">
					<Loader2 class="mr-2 h-5 w-5 animate-spin" />
					<span class="text-xs">Loading bills register...</span>
				</div>
			{:else if bills.length === 0}
				<div class="rounded-xl border border-dashed border-neutral-300 p-8 text-center">
					<FileText class="mx-auto h-8 w-8 text-neutral-400" />
					<p class="mt-2 text-sm font-medium text-neutral-900">No maintenance bills found</p>
					<p class="mt-1 text-xs text-neutral-500">
						Generate bills from the "Billing Periods" tab to create unit invoices.
					</p>
				</div>
			{:else}
				<div class="overflow-x-auto rounded-xl border border-neutral-200">
					<table class="w-full text-left text-xs">
						<thead
							class="border-b border-neutral-200 bg-neutral-50 text-[11px] font-medium text-neutral-600"
						>
							<tr>
								<th class="px-4 py-3">Bill Number</th>
								<th class="px-4 py-3">Unit / Flat</th>
								<th class="px-4 py-3">Period</th>
								<th class="px-4 py-3">Bill Date</th>
								<th class="px-4 py-3">Due Date</th>
								<th class="px-4 py-3 text-right">Total Amount</th>
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
											? 'text-rose-600'
											: 'text-neutral-600'}"
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
										<div class="flex items-center justify-end gap-1.5">
											<Button variant="outline" size="sm" onclick={() => openBillDetail(b.id)}>
												<Eye class="mr-1 h-3 w-3" />
												<span>Details</span>
											</Button>
											{#if b.status !== 'VOIDED'}
												<Button
													variant="outline"
													size="sm"
													class="text-rose-600 hover:text-rose-700"
													onclick={() => openVoidModal(b.id)}
												>
													<Ban class="mr-1 h-3 w-3" />
													<span>Void</span>
												</Button>
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
	{/if}

	<!-- Tab 2: Billing Periods -->
	{#if activeTab === 'periods'}
		<div class="mt-4">
			{#if periods.length === 0}
				<div class="rounded-xl border border-dashed border-neutral-300 p-8 text-center">
					<Calendar class="mx-auto h-8 w-8 text-neutral-400" />
					<p class="mt-2 text-sm font-medium text-neutral-900">No billing periods configured</p>
					<p class="mt-1 text-xs text-neutral-500">
						Create a billing period to start generating maintenance bills.
					</p>
					<div class="mt-4">
						<Button variant="primary" size="sm" onclick={() => (isPeriodModalOpen = true)}>
							<Plus class="mr-1.5 h-3.5 w-3.5" />
							<span>Create First Period</span>
						</Button>
					</div>
				</div>
			{:else}
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
					{#each periods as p (p.id)}
						<div
							class="rounded-xl border border-neutral-200 bg-white p-4 shadow-2xs hover:border-neutral-300"
						>
							<div class="flex items-center justify-between">
								<h3 class="text-sm font-semibold text-neutral-950">{p.period_name}</h3>
								<Badge variant={p.status === 'ACTIVE' ? 'success' : 'outline'} class="text-[10px]">
									{p.status}
								</Badge>
							</div>

							<div class="mt-3 space-y-1.5 text-xs text-neutral-600">
								<div class="flex justify-between">
									<span>Start Date:</span>
									<span class="font-mono text-neutral-900">{p.period_start.slice(0, 10)}</span>
								</div>
								<div class="flex justify-between">
									<span>End Date:</span>
									<span class="font-mono text-neutral-900">{p.period_end.slice(0, 10)}</span>
								</div>
								<div class="flex justify-between">
									<span>Payment Due:</span>
									<span class="font-mono font-medium text-neutral-900"
										>{p.due_date.slice(0, 10)}</span
									>
								</div>
							</div>

							<div class="mt-4 flex items-center justify-between border-t border-neutral-100 pt-3">
								<Button variant="primary" size="sm" onclick={() => openGenerateModal(p.id)}>
									<Wrench class="mr-1.5 h-3 w-3" />
									<span>Generate Bills</span>
								</Button>
								<Button
									variant="outline"
									size="sm"
									onclick={() => {
										billPeriodFilter = p.id;
										activeTab = 'bills';
										loadBills();
									}}
								>
									<span>View Bills</span>
								</Button>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<!-- Tab 3: Charge Configurations -->
	{#if activeTab === 'configs'}
		<div class="mt-4">
			{#if configs.length === 0}
				<div class="rounded-xl border border-dashed border-neutral-300 p-8 text-center">
					<Coins class="mx-auto h-8 w-8 text-neutral-400" />
					<p class="mt-2 text-sm font-medium text-neutral-900">No charge configurations defined</p>
					<p class="mt-1 text-xs text-neutral-500">
						Configure recurring or per-unit fees that will be applied to generated bills.
					</p>
					<div class="mt-4">
						<Button variant="primary" size="sm" onclick={() => (isConfigModalOpen = true)}>
							<Plus class="mr-1.5 h-3.5 w-3.5" />
							<span>Add Charge Config</span>
						</Button>
					</div>
				</div>
			{:else}
				<div class="overflow-x-auto rounded-xl border border-neutral-200">
					<table class="w-full text-left text-xs">
						<thead
							class="border-b border-neutral-200 bg-neutral-50 text-[11px] font-medium text-neutral-600"
						>
							<tr>
								<th class="px-4 py-3">Charge Name</th>
								<th class="px-4 py-3">Category</th>
								<th class="px-4 py-3">Charge Model</th>
								<th class="px-4 py-3">Frequency</th>
								<th class="px-4 py-3 text-right">Amount (₹)</th>
								<th class="px-4 py-3">Effective Range</th>
								<th class="px-4 py-3 text-center">Status</th>
								<th class="px-4 py-3 text-right">Actions</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-neutral-200 bg-white">
							{#each configs as c (c.id)}
								<tr class="hover:bg-neutral-50/60 {c.status === 'INACTIVE' ? 'opacity-60' : ''}">
									<td class="px-4 py-3 font-medium text-neutral-950">{c.name}</td>
									<td class="px-4 py-3 text-neutral-600">{c.category_name || 'General'}</td>
									<td class="px-4 py-3 font-mono text-[11px] text-neutral-700">{c.charge_type}</td>
									<td class="px-4 py-3 text-neutral-600">{c.billing_frequency}</td>
									<td class="px-4 py-3 text-right font-semibold text-neutral-950">
										{formatCurrency(c.amount)}
									</td>
									<td class="px-4 py-3 text-[11px] text-neutral-600">
										{c.effective_from ? c.effective_from.slice(0, 10) : '—'}
										{#if c.effective_until}
											to {c.effective_until.slice(0, 10)}
										{/if}
									</td>
									<td class="px-4 py-3 text-center">
										<Badge
											variant={c.status === 'ACTIVE' ? 'success' : 'outline'}
											class="text-[10px]"
										>
											{c.status}
										</Badge>
									</td>
									<td class="px-4 py-3 text-right">
										{#if c.status === 'ACTIVE'}
											<Button
												variant="outline"
												size="sm"
												class="text-rose-600 hover:text-rose-700"
												onclick={() => handleDeactivateConfig(c.id)}
											>
												<span>Deactivate</span>
											</Button>
										{:else}
											<span class="text-[11px] text-neutral-400 italic">Inactive</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Tab 4: Charge Categories -->
	{#if activeTab === 'categories'}
		<div class="mt-4">
			{#if categories.length === 0}
				<div class="rounded-xl border border-dashed border-neutral-300 p-8 text-center">
					<Tag class="mx-auto h-8 w-8 text-neutral-400" />
					<p class="mt-2 text-sm font-medium text-neutral-900">No charge categories configured</p>
					<p class="mt-1 text-xs text-neutral-500">
						Add categories like Monthly Maintenance, Sinking Fund, or Repair Fund.
					</p>
					<div class="mt-4">
						<Button variant="primary" size="sm" onclick={() => (isCategoryModalOpen = true)}>
							<Plus class="mr-1.5 h-3.5 w-3.5" />
							<span>Add Category</span>
						</Button>
					</div>
				</div>
			{:else}
				<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
					{#each categories as cat (cat.id)}
						<div class="rounded-xl border border-neutral-200 bg-white p-4 shadow-2xs">
							<div class="flex items-center justify-between">
								<h3 class="text-sm font-semibold text-neutral-950">{cat.name}</h3>
								<Badge
									variant={cat.status === 'ACTIVE' ? 'success' : 'outline'}
									class="text-[10px]"
								>
									{cat.status}
								</Badge>
							</div>
							<p class="mt-2 text-xs text-neutral-500">
								{cat.description || 'No description provided.'}
							</p>
							<div class="mt-3 text-[10px] text-neutral-400">
								Created: {cat.created_at ? cat.created_at.slice(0, 10) : '—'}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>

<!-- ======================================================== -->
<!-- MODALS -->
<!-- ======================================================== -->

<!-- 1. Bill Details Dialog -->
{#if isDetailModalOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div
			class="relative w-full max-w-2xl rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl"
		>
			<div class="flex items-center justify-between border-b border-neutral-200 pb-4">
				<div>
					<h3 class="text-base font-semibold text-neutral-950">Maintenance Bill Details</h3>
					<p class="text-xs text-neutral-500">Immutable financial snapshot and itemized charges.</p>
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
					<span class="text-xs">Loading bill breakdown...</span>
				</div>
			{:else if selectedBill}
				<div class="mt-4 space-y-4">
					<!-- Meta banner -->
					<div class="grid grid-cols-2 gap-3 rounded-xl bg-neutral-50 p-3 text-xs sm:grid-cols-4">
						<div>
							<span class="text-[11px] text-neutral-500">Bill Number</span>
							<p class="font-mono font-medium text-neutral-900">{selectedBill.bill_number}</p>
						</div>
						<div>
							<span class="text-[11px] text-neutral-500">Unit / Flat</span>
							<p class="font-medium text-neutral-900">{selectedBill.unit_identifier || 'Unit'}</p>
						</div>
						<div>
							<span class="text-[11px] text-neutral-500">Billing Period</span>
							<p class="font-medium text-neutral-900">{selectedBill.period_name || '—'}</p>
						</div>
						<div>
							<span class="text-[11px] text-neutral-500">Due Date</span>
							<p
								class="font-mono font-medium {selectedBill.status === 'OVERDUE'
									? 'text-rose-600'
									: 'text-neutral-900'}"
							>
								{selectedBill.due_date ? selectedBill.due_date.slice(0, 10) : '—'}
							</p>
						</div>
					</div>

					<!-- Line Items Table -->
					<div>
						<h4 class="mb-2 text-xs font-semibold text-neutral-800">Itemized Financial Snapshot</h4>
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
											>Bill Total ({selectedBill.currency}):</td
										>
										<td class="px-4 py-3 text-right font-mono text-sm text-neutral-950">
											{formatCurrency(selectedBill.total_amount)}
										</td>
									</tr>
								</tfoot>
							</table>
						</div>
					</div>

					{#if selectedBill.status === 'VOIDED'}
						<div class="rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800">
							<strong>Bill Voided:</strong>
							{selectedBill.void_reason || 'No reason specified'}
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

<!-- 2. Void Bill Modal -->
{#if isVoidModalOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div class="w-full max-w-md rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl">
			<div class="flex items-center justify-between border-b border-neutral-200 pb-3">
				<h3 class="text-sm font-semibold text-neutral-950">Void Maintenance Bill</h3>
				<button
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isVoidModalOpen = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<p class="mt-3 text-xs text-neutral-500">
				Voiding a bill preserves its audit history and line items while permanently removing it from
				active receivable obligations.
			</p>

			{#if voidError}
				<div class="mt-3 rounded-lg border border-rose-200 bg-rose-50 p-2.5 text-xs text-rose-700">
					{voidError}
				</div>
			{/if}

			<div class="mt-4">
				<label for="void-reason" class="mb-1 block text-xs font-medium text-neutral-700">
					Reason for Voiding <span class="text-rose-500">*</span>
				</label>
				<textarea
					id="void-reason"
					rows="3"
					class="w-full rounded-lg border border-neutral-300 p-2 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
					placeholder="e.g. Assessment adjustment or duplicate period assignment error"
					bind:value={voidReason}></textarea>
			</div>

			<div class="mt-5 flex justify-end gap-2">
				<Button
					variant="outline"
					size="sm"
					onclick={() => (isVoidModalOpen = false)}
					disabled={isVoiding}
				>
					<span>Cancel</span>
				</Button>
				<Button
					variant="primary"
					size="sm"
					class="bg-rose-600 text-white hover:bg-rose-700"
					onclick={handleVoidBill}
					disabled={isVoiding}
				>
					{#if isVoiding}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					{/if}
					<span>Confirm Void</span>
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- 3. New Category Modal -->
{#if isCategoryModalOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div class="w-full max-w-md rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl">
			<div class="flex items-center justify-between border-b border-neutral-200 pb-3">
				<h3 class="text-sm font-semibold text-neutral-950">Create Charge Category</h3>
				<button
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isCategoryModalOpen = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if categoryError}
				<div class="mt-3 rounded-lg border border-rose-200 bg-rose-50 p-2.5 text-xs text-rose-700">
					{categoryError}
				</div>
			{/if}

			<div class="mt-4 space-y-3">
				<div>
					<label for="cat-name" class="mb-1 block text-xs font-medium text-neutral-700"
						>Category Name <span class="text-rose-500">*</span></label
					>
					<input
						id="cat-name"
						type="text"
						class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
						placeholder="e.g. Monthly Maintenance, Sinking Fund, Parking"
						bind:value={categoryName}
					/>
				</div>
				<div>
					<label for="cat-desc" class="mb-1 block text-xs font-medium text-neutral-700"
						>Description</label
					>
					<textarea
						id="cat-desc"
						rows="2"
						class="w-full rounded-lg border border-neutral-300 p-2 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
						placeholder="Optional description of the charge scope"
						bind:value={categoryDescription}></textarea>
				</div>
			</div>

			<div class="mt-5 flex justify-end gap-2">
				<Button
					variant="outline"
					size="sm"
					onclick={() => (isCategoryModalOpen = false)}
					disabled={isSavingCategory}
				>
					<span>Cancel</span>
				</Button>
				<Button
					variant="primary"
					size="sm"
					onclick={handleCreateCategory}
					disabled={isSavingCategory}
				>
					{#if isSavingCategory}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					{/if}
					<span>Create Category</span>
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- 4. New Charge Config Modal -->
{#if isConfigModalOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div class="w-full max-w-lg rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl">
			<div class="flex items-center justify-between border-b border-neutral-200 pb-3">
				<h3 class="text-sm font-semibold text-neutral-950">Add Charge Configuration</h3>
				<button
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isConfigModalOpen = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if configError}
				<div class="mt-3 rounded-lg border border-rose-200 bg-rose-50 p-2.5 text-xs text-rose-700">
					{configError}
				</div>
			{/if}

			<div class="mt-4 space-y-3">
				<div>
					<label for="cfg-cat" class="mb-1 block text-xs font-medium text-neutral-700"
						>Charge Category <span class="text-rose-500">*</span></label
					>
					<select
						id="cfg-cat"
						class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
						bind:value={configCategoryId}
					>
						<option value="">Select a category...</option>
						{#each categories as cat (cat.id)}
							<option value={cat.id}>{cat.name}</option>
						{/each}
					</select>
				</div>

				<div>
					<label for="cfg-name" class="mb-1 block text-xs font-medium text-neutral-700"
						>Config Name <span class="text-rose-500">*</span></label
					>
					<input
						id="cfg-name"
						type="text"
						class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
						placeholder="e.g. Standard Flat Assessment Rate"
						bind:value={configName}
					/>
				</div>

				<div class="grid grid-cols-2 gap-3">
					<div>
						<label for="cfg-model" class="mb-1 block text-xs font-medium text-neutral-700"
							>Charge Model</label
						>
						<select
							id="cfg-model"
							class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
							bind:value={configChargeType}
						>
							<option value="FLAT_RECURRING">Flat Recurring</option>
							<option value="PER_UNIT_RECURRING">Per Unit Recurring</option>
							<option value="ONE_TIME">One Time</option>
						</select>
					</div>
					<div>
						<label for="cfg-freq" class="mb-1 block text-xs font-medium text-neutral-700"
							>Billing Frequency</label
						>
						<select
							id="cfg-freq"
							class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
							bind:value={configFrequency}
						>
							<option value="MONTHLY">Monthly</option>
							<option value="QUARTERLY">Quarterly</option>
							<option value="ANNUAL">Annual</option>
							<option value="ONE_TIME">One Time</option>
						</select>
					</div>
				</div>

				<div class="grid grid-cols-2 gap-3">
					<div>
						<label for="cfg-amount" class="mb-1 block text-xs font-medium text-neutral-700"
							>Amount (INR ₹) <span class="text-rose-500">*</span></label
						>
						<input
							id="cfg-amount"
							type="number"
							step="0.01"
							class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
							placeholder="2500.00"
							bind:value={configAmount}
						/>
					</div>
					<div>
						<label for="cfg-eff" class="mb-1 block text-xs font-medium text-neutral-700"
							>Effective From</label
						>
						<input
							id="cfg-eff"
							type="date"
							class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
							bind:value={configEffectiveFrom}
						/>
					</div>
				</div>
			</div>

			<div class="mt-5 flex justify-end gap-2">
				<Button
					variant="outline"
					size="sm"
					onclick={() => (isConfigModalOpen = false)}
					disabled={isSavingConfig}
				>
					<span>Cancel</span>
				</Button>
				<Button variant="primary" size="sm" onclick={handleCreateConfig} disabled={isSavingConfig}>
					{#if isSavingConfig}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					{/if}
					<span>Save Configuration</span>
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- 5. New Period Modal -->
{#if isPeriodModalOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div class="w-full max-w-md rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl">
			<div class="flex items-center justify-between border-b border-neutral-200 pb-3">
				<h3 class="text-sm font-semibold text-neutral-950">Create Billing Period</h3>
				<button
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isPeriodModalOpen = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if periodError}
				<div class="mt-3 rounded-lg border border-rose-200 bg-rose-50 p-2.5 text-xs text-rose-700">
					{periodError}
				</div>
			{/if}

			<div class="mt-4 space-y-3">
				<div>
					<label for="p-name" class="mb-1 block text-xs font-medium text-neutral-700"
						>Period Label <span class="text-rose-500">*</span></label
					>
					<input
						id="p-name"
						type="text"
						class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
						placeholder="e.g. October 2026 Maintenance"
						bind:value={periodName}
					/>
				</div>

				<div class="grid grid-cols-2 gap-3">
					<div>
						<label for="p-start" class="mb-1 block text-xs font-medium text-neutral-700"
							>Period Start</label
						>
						<input
							id="p-start"
							type="date"
							class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
							bind:value={periodStart}
						/>
					</div>
					<div>
						<label for="p-end" class="mb-1 block text-xs font-medium text-neutral-700"
							>Period End</label
						>
						<input
							id="p-end"
							type="date"
							class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
							bind:value={periodEnd}
						/>
					</div>
				</div>

				<div>
					<label for="p-due" class="mb-1 block text-xs font-medium text-neutral-700"
						>Payment Due Date <span class="text-rose-500">*</span></label
					>
					<input
						id="p-due"
						type="date"
						class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
						bind:value={periodDueDate}
					/>
					<p class="mt-1 text-[11px] text-neutral-500">
						Bills not settled by this date will be displayed as Overdue.
					</p>
				</div>
			</div>

			<div class="mt-5 flex justify-end gap-2">
				<Button
					variant="outline"
					size="sm"
					onclick={() => (isPeriodModalOpen = false)}
					disabled={isSavingPeriod}
				>
					<span>Cancel</span>
				</Button>
				<Button variant="primary" size="sm" onclick={handleCreatePeriod} disabled={isSavingPeriod}>
					{#if isSavingPeriod}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					{/if}
					<span>Create Period</span>
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- 6. Generate Bills Modal (Preview & Confirm Workflow) -->
{#if isGenerateModalOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div class="w-full max-w-lg rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl">
			<div class="flex items-center justify-between border-b border-neutral-200 pb-3">
				<div>
					<h3 class="text-sm font-semibold text-neutral-950">Generate Maintenance Bills</h3>
					<p class="text-xs text-neutral-500">
						Pre-flight calculation and idempotent invoice generation.
					</p>
				</div>
				<button
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isGenerateModalOpen = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if generationError}
				<div class="mt-3 rounded-lg border border-rose-200 bg-rose-50 p-2.5 text-xs text-rose-700">
					{generationError}
				</div>
			{/if}

			{#if isLoadingPreview}
				<div class="flex items-center justify-center py-12 text-neutral-400">
					<Loader2 class="mr-2 h-6 w-6 animate-spin" />
					<span class="text-xs">Calculating eligible units & charge rules...</span>
				</div>
			{:else if generationResult}
				<!-- Result state -->
				<div class="mt-4 rounded-xl border border-emerald-200 bg-emerald-50/70 p-4">
					<div class="flex items-center gap-2 text-sm font-semibold text-emerald-900">
						<CheckCircle2 class="h-4 w-4 text-emerald-600" />
						<span>Bill Generation Complete</span>
					</div>
					<div class="mt-3 grid grid-cols-3 gap-2 text-center text-xs">
						<div class="rounded-lg border border-emerald-200 bg-white p-2.5">
							<span class="text-[10px] text-neutral-500">Bills Created</span>
							<p class="text-base font-bold text-neutral-950">{generationResult.bills_created}</p>
						</div>
						<div class="rounded-lg border border-emerald-200 bg-white p-2.5">
							<span class="text-[10px] text-neutral-500">Units Skipped</span>
							<p class="text-base font-bold text-neutral-600">{generationResult.units_skipped}</p>
						</div>
						<div class="rounded-lg border border-emerald-200 bg-white p-2.5">
							<span class="text-[10px] text-neutral-500">Total Invoiced</span>
							<p class="text-base font-bold text-emerald-800">
								{formatCurrency(generationResult.total_billed_amount)}
							</p>
						</div>
					</div>
				</div>

				<div class="mt-5 flex justify-end gap-2">
					<Button
						variant="primary"
						size="sm"
						onclick={() => {
							isGenerateModalOpen = false;
							activeTab = 'bills';
						}}
					>
						<span>View Generated Bills</span>
					</Button>
				</div>
			{:else if generationPreview}
				<!-- Preview state -->
				<div class="mt-4 space-y-3">
					<div class="space-y-1.5 rounded-xl border border-neutral-200 bg-neutral-50 p-3 text-xs">
						<div class="flex justify-between">
							<span class="text-neutral-500">Period:</span>
							<span class="font-semibold text-neutral-900">{generationPreview.period_name}</span>
						</div>
						<div class="flex justify-between">
							<span class="text-neutral-500">Coverage:</span>
							<span class="font-mono text-neutral-700">
								{generationPreview.period_start.slice(0, 10)} to {generationPreview.period_end.slice(
									0,
									10
								)}
							</span>
						</div>
						<div class="flex justify-between">
							<span class="text-neutral-500">Due Date:</span>
							<span class="font-mono font-medium text-neutral-900"
								>{generationPreview.due_date.slice(0, 10)}</span
							>
						</div>
					</div>

					<div class="grid grid-cols-2 gap-3 text-xs sm:grid-cols-4">
						<div class="rounded-xl border border-neutral-200 p-2.5 text-center">
							<span class="text-[10px] text-neutral-500">Eligible Units</span>
							<p class="text-sm font-bold text-neutral-950">
								{generationPreview.eligible_unit_count}
							</p>
						</div>
						<div class="rounded-xl border border-neutral-200 p-2.5 text-center">
							<span class="text-[10px] text-neutral-500">Already Billed</span>
							<p class="text-sm font-bold text-neutral-600">
								{generationPreview.already_billed_count}
							</p>
						</div>
						<div class="rounded-xl border border-neutral-200 bg-neutral-50 p-2.5 text-center">
							<span class="text-[10px] text-neutral-500">To Generate</span>
							<p class="text-sm font-bold text-neutral-950">
								{generationPreview.units_to_bill_count}
							</p>
						</div>
						<div class="rounded-xl border border-neutral-200 p-2.5 text-center">
							<span class="text-[10px] text-neutral-500">Rate / Unit</span>
							<p class="text-sm font-bold text-neutral-900">
								{formatCurrency(generationPreview.estimated_total_per_unit)}
							</p>
						</div>
					</div>

					<div class="rounded-xl border border-amber-200 bg-amber-50/70 p-3 text-xs text-amber-900">
						<strong>Financial Invariant Notice:</strong>
						Generation creates legal financial records with fixed snapshot amounts. Idempotency is enforced
						by DB constraints — repeated clicks will not create duplicate bills.
					</div>
				</div>

				<div class="mt-5 flex justify-end gap-2">
					<Button
						variant="outline"
						size="sm"
						onclick={() => (isGenerateModalOpen = false)}
						disabled={isGenerating}
					>
						<span>Cancel</span>
					</Button>
					<Button
						variant="primary"
						size="sm"
						onclick={handleExecuteGeneration}
						disabled={isGenerating || generationPreview.units_to_bill_count === 0}
					>
						{#if isGenerating}
							<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
						{/if}
						<span>Generate {generationPreview.units_to_bill_count} Bills</span>
					</Button>
				</div>
			{/if}
		</div>
	</div>
{/if}

<!-- 7. Record Offline Payment Modal (S12) -->
{#if isRecordPaymentModalOpen && recordingBill}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div class="w-full max-w-lg rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl">
			<div class="flex items-center justify-between border-b border-neutral-200 pb-3">
				<div>
					<h3 class="text-sm font-semibold text-neutral-950">Record Offline Collection</h3>
					<p class="text-xs text-neutral-500">Record money received and allocate to bill.</p>
				</div>
				<button
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isRecordPaymentModalOpen = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if recordPaymentError}
				<div class="mt-3 rounded-lg border border-rose-200 bg-rose-50 p-2.5 text-xs text-rose-700">
					{recordPaymentError}
				</div>
			{/if}

			<!-- Target bill banner -->
			<div class="mt-4 rounded-xl border border-neutral-200 bg-neutral-50 p-3 text-xs">
				<div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
					<div>
						<span class="text-[10px] text-neutral-500">Bill Number</span>
						<p class="font-mono font-bold text-neutral-900">{recordingBill.bill_number}</p>
					</div>
					<div>
						<span class="text-[10px] text-neutral-500">Unit / Flat</span>
						<p class="font-medium text-neutral-900">{recordingBill.unit_identifier}</p>
					</div>
					<div>
						<span class="text-[10px] text-neutral-500">Total Billed</span>
						<p class="font-medium text-neutral-700">{formatCurrency(recordingBill.total_amount)}</p>
					</div>
					<div>
						<span class="text-[10px] text-neutral-500">Outstanding</span>
						<p class="font-bold text-rose-600">
							{formatCurrency(recordingBill.outstanding_balance)}
						</p>
					</div>
				</div>
			</div>

			<div class="mt-4 space-y-3 text-xs">
				<!-- Amount input -->
				<div>
					<div class="mb-1 flex items-center justify-between">
						<label for="pay-amount" class="font-medium text-neutral-700">
							Amount Received (₹) <span class="text-rose-500">*</span>
						</label>
						<button
							type="button"
							class="text-[11px] font-semibold text-emerald-700 hover:underline"
							onclick={() => {
								if (recordingBill) payAmount = recordingBill.outstanding_balance;
							}}
						>
							Pay Full ({formatCurrency(recordingBill.outstanding_balance)})
						</button>
					</div>
					<input
						id="pay-amount"
						type="number"
						step="0.01"
						min="0.01"
						max={recordingBill.outstanding_balance}
						class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 font-mono text-sm font-semibold text-neutral-900 focus:border-neutral-950 focus:outline-none"
						bind:value={payAmount}
					/>
					{#if payAmount > 0 && payAmount < recordingBill.outstanding_balance}
						<p class="mt-1 text-[11px] text-amber-700">
							Partial payment: {formatCurrency(recordingBill.outstanding_balance - payAmount)} will remain
							outstanding.
						</p>
					{/if}
				</div>

				<div class="grid grid-cols-2 gap-3">
					<div>
						<label for="pay-method" class="mb-1 block font-medium text-neutral-700">
							Payment Method <span class="text-rose-500">*</span>
						</label>
						<select
							id="pay-method"
							class="w-full rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
							bind:value={payMethod}
						>
							<option value="BANK_TRANSFER">Bank Transfer (NEFT/RTGS/IMPS)</option>
							<option value="UPI_RECORDED_OFFLINE">UPI (Recorded Offline)</option>
							<option value="CHEQUE">Cheque</option>
							<option value="CASH">Cash</option>
							<option value="OTHER">Other</option>
						</select>
					</div>
					<div>
						<label for="pay-date" class="mb-1 block font-medium text-neutral-700">
							Payment Date <span class="text-rose-500">*</span>
						</label>
						<input
							id="pay-date"
							type="date"
							class="w-full rounded-lg border border-neutral-300 px-2.5 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
							bind:value={payDate}
						/>
					</div>
				</div>

				<div>
					<label for="pay-ref" class="mb-1 block font-medium text-neutral-700">
						External Reference / Instrument #
					</label>
					<input
						id="pay-ref"
						type="text"
						placeholder="e.g. UTR12345678, Cheque #452109"
						class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
						bind:value={payExternalRef}
					/>
				</div>

				<div class="grid grid-cols-2 gap-3">
					<div>
						<label for="pay-payer" class="mb-1 block font-medium text-neutral-700">
							Payer Name
						</label>
						<input
							id="pay-payer"
							type="text"
							placeholder="e.g. Vijay Godbole"
							class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
							bind:value={payPayerName}
						/>
					</div>
					<div>
						<label for="pay-notes" class="mb-1 block font-medium text-neutral-700">
							Administrative Notes
						</label>
						<input
							id="pay-notes"
							type="text"
							placeholder="Bank statement verified"
							class="w-full rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
							bind:value={payNotes}
						/>
					</div>
				</div>
			</div>

			<div class="mt-5 flex justify-end gap-2">
				<Button
					variant="outline"
					size="sm"
					onclick={() => (isRecordPaymentModalOpen = false)}
					disabled={isRecordingPayment}
				>
					<span>Cancel</span>
				</Button>
				<Button
					variant="primary"
					size="sm"
					onclick={handleRecordPayment}
					disabled={isRecordingPayment || payAmount <= 0}
				>
					{#if isRecordingPayment}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					{/if}
					<span>Confirm & Issue Receipt</span>
				</Button>
			</div>
		</div>
	</div>
{/if}

<!-- 8. Printable Receipt Modal (S12) -->
{#if isReceiptModalOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div class="w-full max-w-md rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl">
			<div class="flex items-center justify-between border-b border-neutral-200 pb-3">
				<div class="flex items-center gap-2">
					<Receipt class="h-4 w-4 text-emerald-700" />
					<h3 class="text-sm font-semibold text-neutral-950">Payment Receipt</h3>
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

<!-- 9. Payment Reversal Modal (S12) -->
{#if isReversalModalOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs"
	>
		<div class="w-full max-w-md rounded-2xl border border-neutral-200 bg-white p-6 shadow-xl">
			<div class="flex items-center justify-between border-b border-neutral-200 pb-3">
				<div class="flex items-center gap-2">
					<RotateCcw class="h-4 w-4 text-rose-600" />
					<h3 class="text-sm font-semibold text-neutral-950">Reverse Payment</h3>
				</div>
				<button
					class="text-neutral-400 hover:text-neutral-700"
					onclick={() => (isReversalModalOpen = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if reversalError}
				<div class="mt-3 rounded-lg border border-rose-200 bg-rose-50 p-2.5 text-xs text-rose-700">
					{reversalError}
				</div>
			{/if}

			<div class="mt-3 space-y-3 text-xs">
				<div class="rounded-xl border border-rose-200 bg-rose-50/70 p-3 text-rose-900">
					<strong>Audit Notice:</strong> Reversing payment
					<span class="font-mono font-semibold">{reversingPaymentRef}</span> will restore the
					outstanding balance on bill
					<span class="font-mono font-semibold">{reversingBillNumber}</span>. The original receipt
					and allocation history are preserved with REVERSED status. This action corrects records
					only and does not initiate an automated bank refund.
				</div>

				<div>
					<label for="rev-reason" class="mb-1 block font-medium text-neutral-700">
						Mandatory Reversal Reason <span class="text-rose-500">*</span>
					</label>
					<textarea
						id="rev-reason"
						rows="3"
						placeholder="e.g. Cheque bounced, entered incorrect amount, duplicate entry"
						class="w-full rounded-lg border border-neutral-300 p-2.5 text-xs text-neutral-900 focus:border-neutral-950 focus:outline-none"
						bind:value={reversalReason}></textarea>
				</div>
			</div>

			<div class="mt-5 flex justify-end gap-2">
				<Button
					variant="outline"
					size="sm"
					onclick={() => (isReversalModalOpen = false)}
					disabled={isReversing}
				>
					<span>Cancel</span>
				</Button>
				<Button
					variant="destructive"
					size="sm"
					onclick={handleReversePayment}
					disabled={isReversing || !reversalReason.trim()}
				>
					{#if isReversing}
						<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
					{/if}
					<span>Confirm Reversal</span>
				</Button>
			</div>
		</div>
	</div>
{/if}
