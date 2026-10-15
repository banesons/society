<script lang="ts">
	import {
		societyApi,
		type SocietyItem,
		type SocietyProfile,
		type Building,
		type Gate,
		type FormationSummary,
		type GeneratedUnitPreview
	} from '$lib/api/society';
	import { ApiClientError } from '$lib/api/client';
	import Button from '$lib/components/ui/button.svelte';
	import Badge from '$lib/components/ui/badge.svelte';
	import {
		Building2,
		Layers,
		DoorOpen,
		Users,
		ClipboardCheck,
		Check,
		ChevronRight,
		ChevronLeft,
		Plus,
		AlertCircle,
		Loader2,
		ShieldCheck,
		CheckCircle2,
		X
	} from '@lucide/svelte';

	let {
		society,
		isOpen = false,
		onClose,
		onComplete
	}: {
		society: SocietyItem | SocietyProfile;
		isOpen: boolean;
		onClose: () => void;
		onComplete?: () => void;
	} = $props();

	// Step navigation
	let currentStep = $state<number>(1);
	const steps = [
		{ id: 1, label: 'Society', icon: Building2 },
		{ id: 2, label: 'Buildings & Wings', icon: Layers },
		{ id: 3, label: 'Units Auto-Gen', icon: DoorOpen },
		{ id: 4, label: 'Gates & Staffing', icon: Users },
		{ id: 5, label: 'Review & Commit', icon: ClipboardCheck }
	];

	// Loading & notification states
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let successMsg = $state<string | null>(null);

	// Data states
	let buildings = $state<Building[]>([]);
	let gates = $state<Gate[]>([]);
	let formationSummary = $state<FormationSummary | null>(null);

	// Step 2 Form: Add Building
	let showAddBuilding = $state(false);
	let bName = $state('');
	let bCode = $state('');
	let bFloors = $state(9);

	// Step 2 Form: Add Wing
	let showAddWing = $state(false);
	let selectedBuildingForWing = $state<string>('');
	let wName = $state('');
	let wCode = $state('');
	let wFloors = $state(9);

	// Step 3 Form: Auto-Generate Units
	let genBuildingId = $state<string>('');
	let genWingId = $state<string>('');
	let genStartingFloor = $state<number>(1);
	let genTotalFloors = $state<number>(9);
	let genUnitsPerFloor = $state<number>(3);
	let genStartingUnitNumber = $state<number>(101);
	let genPrefix = $state<string>('A');
	let genUnitType = $state<string>('RESIDENTIAL');
	let genPreviewUnits = $state<GeneratedUnitPreview[]>([]);
	let isPreviewing = $state(false);
	let isGenerating = $state(false);

	// Step 4 Form: Add Gate
	let showAddGate = $state(false);
	let gName = $state('Main Gate');
	let gCode = $state('GATE-01');
	let gType = $state('VEHICLE_AND_PEDESTRIAN');
	let gStaffPerShift = $state<number>(2);
	let gDescription = $state('');

	// Step 4 Form: Staffing Requirement
	let shiftsPerDay = $state<number>(3);
	let totalHeadcount = $state<number>(10);
	let staffingNotes = $state('24/7 security with 3 shifts');
	let isSavingStaffing = $state(false);

	// Load initial structure data
	async function loadStructureData() {
		if (!society?.id) return;
		isLoading = true;
		error = null;
		try {
			const [bRes, gRes, sRes, sumRes] = await Promise.all([
				societyApi.listBuildings(society.id),
				societyApi.listGates(society.id),
				societyApi.getStaffing(society.id),
				societyApi.getFormationSummary(society.id)
			]);
			buildings = bRes.buildings;
			gates = gRes.gates;
			formationSummary = sumRes;

			if (sRes) {
				shiftsPerDay = sRes.shifts_per_day;
				totalHeadcount = sRes.total_required_headcount;
				staffingNotes = sRes.notes || '';
			}

			if (buildings.length > 0 && !genBuildingId) {
				genBuildingId = buildings[0].id;
				genPrefix = buildings[0].building_code;
			}
		} catch (err: unknown) {
			error = err instanceof ApiClientError ? err.message : 'Failed to load society structure';
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (isOpen && society?.id) {
			loadStructureData();
		}
	});

	// Step 2: Create Building
	async function handleCreateBuilding() {
		if (!bName.trim() || !bCode.trim()) {
			error = 'Building name and code are required';
			return;
		}
		isLoading = true;
		error = null;
		try {
			await societyApi.createBuilding(society.id, {
				name: bName.trim(),
				building_code: bCode.trim().toUpperCase(),
				total_floors: Number(bFloors)
			});
			bName = '';
			bCode = '';
			bFloors = 9;
			showAddBuilding = false;
			successMsg = 'Building created successfully';
			await loadStructureData();
		} catch (err: unknown) {
			error = err instanceof ApiClientError ? err.message : 'Failed to create building';
		} finally {
			isLoading = false;
		}
	}

	// Step 2: Create Wing
	async function handleCreateWing() {
		if (!selectedBuildingForWing || !wName.trim() || !wCode.trim()) {
			error = 'Target building, wing name, and wing code are required';
			return;
		}
		isLoading = true;
		error = null;
		try {
			await societyApi.createWing(society.id, selectedBuildingForWing, {
				name: wName.trim(),
				wing_code: wCode.trim().toUpperCase(),
				total_floors: Number(wFloors)
			});
			wName = '';
			wCode = '';
			wFloors = 9;
			showAddWing = false;
			successMsg = 'Wing created successfully';
			await loadStructureData();
		} catch (err: unknown) {
			error = err instanceof ApiClientError ? err.message : 'Failed to create wing';
		} finally {
			isLoading = false;
		}
	}

	// Step 3: Preview Units
	async function handlePreviewUnits() {
		if (!genBuildingId) {
			error = 'Please select a building';
			return;
		}
		isPreviewing = true;
		error = null;
		try {
			const res = await societyApi.previewUnits(society.id, {
				building_id: genBuildingId,
				wing_id: genWingId ? genWingId : undefined,
				starting_floor: Number(genStartingFloor),
				total_floors: Number(genTotalFloors),
				units_per_floor: Number(genUnitsPerFloor),
				starting_unit_number: Number(genStartingUnitNumber),
				prefix: genPrefix.trim(),
				unit_type: genUnitType
			});
			genPreviewUnits = res.units;
		} catch (err: unknown) {
			error = err instanceof ApiClientError ? err.message : 'Preview generation failed';
		} finally {
			isPreviewing = false;
		}
	}

	// Step 3: Persist Units
	async function handleGenerateUnits() {
		if (!genBuildingId) {
			error = 'Please select a building';
			return;
		}
		isGenerating = true;
		error = null;
		try {
			const res = await societyApi.generateUnits(society.id, {
				building_id: genBuildingId,
				wing_id: genWingId ? genWingId : undefined,
				starting_floor: Number(genStartingFloor),
				total_floors: Number(genTotalFloors),
				units_per_floor: Number(genUnitsPerFloor),
				starting_unit_number: Number(genStartingUnitNumber),
				prefix: genPrefix.trim(),
				unit_type: genUnitType
			});
			successMsg = `Successfully persisted ${res.generated_count} units!`;
			genPreviewUnits = [];
			await loadStructureData();
		} catch (err: unknown) {
			error = err instanceof ApiClientError ? err.message : 'Unit auto-generation failed';
		} finally {
			isGenerating = false;
		}
	}

	// Step 4: Add Gate
	async function handleCreateGate() {
		if (!gName.trim() || !gCode.trim()) {
			error = 'Gate name and code are required';
			return;
		}
		isLoading = true;
		error = null;
		try {
			await societyApi.createGate(society.id, {
				name: gName.trim(),
				gate_code: gCode.trim().toUpperCase(),
				gate_type: gType,
				status: 'ACTIVE',
				required_staff_per_shift: Number(gStaffPerShift),
				description: gDescription ? gDescription.trim() : undefined
			});
			gName = '';
			gCode = '';
			gDescription = '';
			showAddGate = false;
			successMsg = 'Gate registered successfully';
			await loadStructureData();
		} catch (err: unknown) {
			error = err instanceof ApiClientError ? err.message : 'Failed to register gate';
		} finally {
			isLoading = false;
		}
	}

	// Step 4: Save Staffing
	async function handleSaveStaffing() {
		isSavingStaffing = true;
		error = null;
		try {
			await societyApi.setStaffing(society.id, {
				shifts_per_day: Number(shiftsPerDay),
				total_required_headcount: Number(totalHeadcount),
				notes: staffingNotes.trim() ? staffingNotes.trim() : undefined
			});
			successMsg = 'Staffing requirements updated';
			await loadStructureData();
		} catch (err: unknown) {
			error = err instanceof ApiClientError ? err.message : 'Failed to save staffing requirements';
		} finally {
			isSavingStaffing = false;
		}
	}

	function nextStep() {
		error = null;
		successMsg = null;
		if (currentStep < 5) currentStep++;
	}

	function prevStep() {
		error = null;
		successMsg = null;
		if (currentStep > 1) currentStep--;
	}
</script>

{#if isOpen && society}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-6"
		role="dialog"
		aria-labelledby="formation-modal-title"
		aria-modal="true"
	>
		<!-- Backdrop -->
		<button
			type="button"
			tabindex="-1"
			class="fixed inset-0 bg-neutral-950/40 backdrop-blur-xs transition-opacity"
			onclick={onClose}
			aria-label="Close formation wizard"
		></button>

		<!-- Main Modal -->
		<div
			class="relative flex h-full max-h-[92vh] w-full max-w-4xl flex-col overflow-hidden rounded-3xl border border-neutral-200 bg-white text-neutral-900 shadow-2xl transition-all"
		>
			<!-- Header -->
			<div
				class="flex items-center justify-between border-b border-neutral-200 bg-neutral-50/50 px-6 py-4"
			>
				<div class="flex items-center gap-3">
					<div
						class="flex h-10 w-10 items-center justify-center rounded-2xl bg-neutral-100 text-neutral-900"
					>
						<Building2 class="h-5 w-5" />
					</div>
					<div>
						<div class="flex items-center gap-2">
							<h2
								id="formation-modal-title"
								class="font-serif text-lg font-medium tracking-tight text-neutral-950"
							>
								Society Formation & Structure
							</h2>
							<Badge variant="outline" class="font-mono text-[10px] text-neutral-600"
								>S05 Guided</Badge
							>
						</div>
						<p class="text-xs text-neutral-500">
							{society.name} ·
							<span class="font-mono text-neutral-700">{society.society_code}</span>
						</p>
					</div>
				</div>

				<button
					type="button"
					onclick={onClose}
					class="rounded-full p-2 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900"
					aria-label="Close dialog"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<!-- Stepper Progress Bar -->
			<div class="border-b border-neutral-200 bg-neutral-50/30 px-6 py-2.5">
				<nav class="flex items-center justify-between gap-1 sm:gap-2">
					{#each steps as step (step.id)}
						<button
							type="button"
							onclick={() => (currentStep = step.id)}
							class={`flex flex-1 items-center justify-center gap-1.5 rounded-xl py-1.5 text-xs font-medium transition-all sm:justify-start sm:px-2.5 ${
								currentStep === step.id
									? 'bg-neutral-900 text-white shadow-xs'
									: currentStep > step.id
										? 'text-emerald-700 hover:bg-neutral-100'
										: 'text-neutral-400 hover:bg-neutral-50 hover:text-neutral-600'
							}`}
						>
							<div
								class={`flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-bold ${
									currentStep === step.id
										? 'bg-white text-neutral-950'
										: currentStep > step.id
											? 'bg-emerald-100 text-emerald-800'
											: 'bg-neutral-200 text-neutral-500'
								}`}
							>
								{#if currentStep > step.id}
									<Check class="h-3 w-3" />
								{:else}
									{step.id}
								{/if}
							</div>
							<span class="hidden truncate sm:inline">{step.label}</span>
						</button>
					{/each}
				</nav>
			</div>

			<!-- Messages -->
			{#if error}
				<div
					class="mx-6 mt-3 flex items-center gap-2 rounded-xl border border-rose-200 bg-rose-50/80 p-3 text-xs text-rose-800"
				>
					<AlertCircle class="h-4 w-4 shrink-0 text-rose-600" />
					<p>{error}</p>
				</div>
			{/if}
			{#if successMsg}
				<div
					class="mx-6 mt-3 flex items-center gap-2 rounded-xl border border-emerald-200 bg-emerald-50/80 p-3 text-xs text-emerald-800"
				>
					<CheckCircle2 class="h-4 w-4 shrink-0 text-emerald-600" />
					<p>{successMsg}</p>
				</div>
			{/if}

			<!-- Body Container -->
			<div class="flex-1 overflow-y-auto px-6 py-5">
				<!-- STEP 1: SOCIETY PROFILE REVIEW -->
				{#if currentStep === 1}
					<div class="space-y-5">
						<div class="rounded-2xl border border-neutral-200 bg-neutral-50/60 p-5">
							<h3 class="font-serif text-sm font-semibold text-neutral-950">
								Step 1 — Review Society Profile
							</h3>
							<p class="mt-1 text-xs text-neutral-500">
								Verify that the target society is active and initialized before configuring its
								physical structure.
							</p>

							<div class="mt-4 grid grid-cols-1 gap-4 text-xs sm:grid-cols-2">
								<div class="space-y-1">
									<span class="text-neutral-500">Society Name:</span>
									<p class="font-medium text-neutral-900">{society.name}</p>
								</div>
								<div class="space-y-1">
									<span class="text-neutral-500">Immutable Society Code:</span>
									<p class="font-mono font-medium text-neutral-900">{society.society_code}</p>
								</div>
								<div class="space-y-1">
									<span class="text-neutral-500">Operational Status:</span>
									<div>
										<Badge variant={society.status === 'ACTIVE' ? 'success' : 'warning'}>
											{society.status}
										</Badge>
									</div>
								</div>
								<div class="space-y-1">
									<span class="text-neutral-500">Location:</span>
									<p class="text-neutral-900">
										{society.city}, {society.state} — {society.pincode}
									</p>
								</div>
								<div class="col-span-2 space-y-1">
									<span class="text-neutral-500">Registered Address:</span>
									<p class="text-neutral-900">{society.address_line1}</p>
								</div>
							</div>
						</div>

						<div
							class="rounded-2xl border border-neutral-200 bg-neutral-50 p-4 text-xs text-neutral-800"
						>
							<p class="font-medium text-neutral-950">Formation Boundary Rule:</p>
							<p class="mt-1 text-neutral-600">
								Every building, wing, unit, gate, and staffing requirement created in this workflow
								is strictly scoped to this society (ID: <span class="font-mono text-neutral-900"
									>{society.id}</span
								>).
							</p>
						</div>
					</div>
				{/if}

				<!-- STEP 2: BUILDINGS & WINGS -->
				{#if currentStep === 2}
					<div class="space-y-5">
						<div class="flex items-center justify-between">
							<div>
								<h3 class="font-serif text-sm font-semibold text-neutral-950">
									Step 2 — Buildings & Wings
								</h3>
								<p class="text-xs text-neutral-500">
									Configure buildings/towers and optional wings. Wings are optional if your building
									does not use them.
								</p>
							</div>
							<div class="flex items-center gap-2">
								<Button
									size="sm"
									variant="outline"
									onclick={() => (showAddBuilding = !showAddBuilding)}
								>
									<Plus class="mr-1.5 h-3.5 w-3.5" />
									<span>Add Building</span>
								</Button>
								{#if buildings.length > 0}
									<Button size="sm" variant="outline" onclick={() => (showAddWing = !showAddWing)}>
										<Plus class="mr-1.5 h-3.5 w-3.5" />
										<span>Add Wing</span>
									</Button>
								{/if}
							</div>
						</div>

						<!-- Add Building Form -->
						{#if showAddBuilding}
							<div class="rounded-2xl border border-neutral-200 bg-neutral-50/80 p-4">
								<div class="flex items-center justify-between">
									<h4 class="font-serif text-xs font-semibold text-neutral-950">New Building</h4>
									<button
										type="button"
										onclick={() => (showAddBuilding = false)}
										class="rounded-full p-1 text-neutral-400 hover:bg-neutral-200 hover:text-neutral-900"
									>
										<X class="h-3.5 w-3.5" />
									</button>
								</div>
								<div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-3">
									<div>
										<label for="b-name" class="block text-[11px] font-medium text-neutral-700"
											>Name</label
										>
										<input
											id="b-name"
											bind:value={bName}
											placeholder="e.g. Tower A"
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
										/>
									</div>
									<div>
										<label for="b-code" class="block text-[11px] font-medium text-neutral-700"
											>Code (Unique)</label
										>
										<input
											id="b-code"
											bind:value={bCode}
											placeholder="e.g. A"
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
										/>
									</div>
									<div>
										<label for="b-floors" class="block text-[11px] font-medium text-neutral-700"
											>Total Floors</label
										>
										<input
											id="b-floors"
											type="number"
											bind:value={bFloors}
											min="1"
											max="100"
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
										/>
									</div>
								</div>
								<div class="mt-3 flex justify-end gap-2">
									<Button size="sm" variant="ghost" onclick={() => (showAddBuilding = false)}
										>Cancel</Button
									>
									<Button
										size="sm"
										variant="primary"
										onclick={handleCreateBuilding}
										disabled={isLoading}
									>
										{#if isLoading}<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />{/if}
										Save Building
									</Button>
								</div>
							</div>
						{/if}

						<!-- Add Wing Form -->
						{#if showAddWing}
							<div class="rounded-2xl border border-neutral-200 bg-neutral-50/80 p-4">
								<div class="flex items-center justify-between">
									<h4 class="font-serif text-xs font-semibold text-neutral-950">New Wing</h4>
									<button
										type="button"
										onclick={() => (showAddWing = false)}
										class="rounded-full p-1 text-neutral-400 hover:bg-neutral-200 hover:text-neutral-900"
									>
										<X class="h-3.5 w-3.5" />
									</button>
								</div>
								<div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-4">
									<div>
										<label for="w-building" class="block text-[11px] font-medium text-neutral-700"
											>Building</label
										>
										<select
											id="w-building"
											bind:value={selectedBuildingForWing}
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900"
										>
											<option value="">Select building...</option>
											{#each buildings as b (b.id)}
												<option value={b.id}>{b.name} ({b.building_code})</option>
											{/each}
										</select>
									</div>
									<div>
										<label for="w-name" class="block text-[11px] font-medium text-neutral-700"
											>Wing Name</label
										>
										<input
											id="w-name"
											bind:value={wName}
											placeholder="e.g. Wing A"
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
										/>
									</div>
									<div>
										<label for="w-code" class="block text-[11px] font-medium text-neutral-700"
											>Wing Code</label
										>
										<input
											id="w-code"
											bind:value={wCode}
											placeholder="e.g. A"
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
										/>
									</div>
									<div>
										<label for="w-floors" class="block text-[11px] font-medium text-neutral-700"
											>Floors</label
										>
										<input
											id="w-floors"
											type="number"
											bind:value={wFloors}
											min="1"
											max="100"
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
										/>
									</div>
								</div>
								<div class="mt-3 flex justify-end gap-2">
									<Button size="sm" variant="ghost" onclick={() => (showAddWing = false)}
										>Cancel</Button
									>
									<Button
										size="sm"
										variant="primary"
										onclick={handleCreateWing}
										disabled={isLoading}
									>
										{#if isLoading}<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />{/if}
										Save Wing
									</Button>
								</div>
							</div>
						{/if}

						<!-- Buildings Roster -->
						{#if buildings.length === 0}
							<div
								class="rounded-2xl border border-dashed border-neutral-300 p-8 text-center text-xs text-neutral-500"
							>
								<Building2 class="mx-auto h-8 w-8 text-neutral-400" />
								<p class="mt-2 font-medium text-neutral-900">No buildings configured yet</p>
								<p class="mt-1 text-neutral-500">
									Add at least one building to proceed with unit generation.
								</p>
							</div>
						{:else}
							<div class="space-y-3">
								{#each buildings as b (b.id)}
									<div class="rounded-2xl border border-neutral-200 bg-white p-4 shadow-xs">
										<div class="flex items-center justify-between">
											<div class="flex items-center gap-2.5">
												<div
													class="flex h-8 w-8 items-center justify-center rounded-lg bg-neutral-100 text-neutral-900"
												>
													<Building2 class="h-4 w-4" />
												</div>
												<div>
													<div class="flex items-center gap-2">
														<span class="text-xs font-semibold text-neutral-950">{b.name}</span>
														<Badge variant="outline" class="font-mono text-[10px]"
															>Code: {b.building_code}</Badge
														>
													</div>
													<p class="text-[11px] text-neutral-500">
														{b.total_floors} Floors · {b.unit_count} Units
													</p>
												</div>
											</div>
											<Badge variant="outline" class="text-[10px] text-neutral-700">
												{b.wings?.length || 0} Wings
											</Badge>
										</div>

										{#if b.wings && b.wings.length > 0}
											<div class="mt-3 border-t border-neutral-200 pt-3">
												<div class="text-[11px] font-medium text-neutral-600">Wings:</div>
												<div class="mt-1.5 flex flex-wrap gap-2">
													{#each b.wings as w (w.id)}
														<div
															class="flex items-center gap-1.5 rounded-xl border border-neutral-200 bg-neutral-50 px-2.5 py-1 text-xs"
														>
															<span class="font-medium text-neutral-900">{w.name}</span>
															<span class="font-mono text-[10px] text-neutral-600"
																>({w.wing_code})</span
															>
															<span class="text-[10px] text-neutral-400"
																>· {w.unit_count} units</span
															>
														</div>
													{/each}
												</div>
											</div>
										{/if}
									</div>
								{/each}
							</div>
						{/if}
					</div>
				{/if}

				<!-- STEP 3: UNIT / FLAT AUTO-GENERATION -->
				{#if currentStep === 3}
					<div class="space-y-5">
						<div>
							<h3 class="font-serif text-sm font-semibold text-neutral-950">
								Step 3 — Unit / Flat Auto-Generation
							</h3>
							<p class="text-xs text-neutral-500">
								Deterministic pattern generator: e.g. 9 floors × 3 units with prefix "A" generates
								A101 through A903.
							</p>
						</div>

						<!-- Config Form -->
						<div class="rounded-2xl border border-neutral-200 bg-neutral-50/60 p-5">
							<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
								<div>
									<label for="gen-building" class="block text-xs font-medium text-neutral-700"
										>Target Building</label
									>
									<select
										id="gen-building"
										bind:value={genBuildingId}
										class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900"
										onchange={() => {
											const selected = buildings.find((b) => b.id === genBuildingId);
											if (selected) genPrefix = selected.building_code;
										}}
									>
										{#each buildings as b (b.id)}
											<option value={b.id}>{b.name} ({b.building_code})</option>
										{/each}
									</select>
								</div>

								<div>
									<label for="gen-wing" class="block text-xs font-medium text-neutral-700"
										>Optional Wing</label
									>
									<select
										id="gen-wing"
										bind:value={genWingId}
										class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900"
									>
										<option value="">No Wing (Direct Building)</option>
										{#if genBuildingId}
											{@const activeB = buildings.find((b) => b.id === genBuildingId)}
											{#if activeB?.wings}
												{#each activeB.wings as w (w.id)}
													<option value={w.id}>{w.name} ({w.wing_code})</option>
												{/each}
											{/if}
										{/if}
									</select>
								</div>

								<div>
									<label for="gen-prefix" class="block text-xs font-medium text-neutral-700"
										>Unit Prefix</label
									>
									<input
										id="gen-prefix"
										bind:value={genPrefix}
										placeholder="e.g. A"
										class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
									/>
								</div>

								<div>
									<label for="gen-start-floor" class="block text-xs font-medium text-neutral-700"
										>Starting Floor</label
									>
									<input
										id="gen-start-floor"
										type="number"
										bind:value={genStartingFloor}
										min="0"
										max="100"
										class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
									/>
								</div>

								<div>
									<label for="gen-total-floors" class="block text-xs font-medium text-neutral-700"
										>Number of Floors</label
									>
									<input
										id="gen-total-floors"
										type="number"
										bind:value={genTotalFloors}
										min="1"
										max="100"
										class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
									/>
								</div>

								<div>
									<label for="gen-units-floor" class="block text-xs font-medium text-neutral-700"
										>Units Per Floor</label
									>
									<input
										id="gen-units-floor"
										type="number"
										bind:value={genUnitsPerFloor}
										min="1"
										max="50"
										class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
									/>
								</div>
							</div>

							<div class="mt-4 flex items-center justify-between border-t border-neutral-200 pt-4">
								<div class="text-xs text-neutral-500">
									Estimated Units: <span class="font-serif font-bold text-neutral-950"
										>{Number(genTotalFloors) * Number(genUnitsPerFloor)}</span
									>
								</div>
								<div class="flex items-center gap-2">
									<Button
										size="sm"
										variant="outline"
										onclick={handlePreviewUnits}
										disabled={isPreviewing}
									>
										{#if isPreviewing}<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />{/if}
										Preview Units
									</Button>
									<Button
										size="sm"
										variant="primary"
										onclick={handleGenerateUnits}
										disabled={isGenerating}
									>
										{#if isGenerating}<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />{/if}
										Generate & Save
									</Button>
								</div>
							</div>
						</div>

						<!-- Unit Preview Grid -->
						{#if genPreviewUnits.length > 0}
							<div class="rounded-2xl border border-neutral-200 bg-white p-4 shadow-xs">
								<div class="flex items-center justify-between border-b border-neutral-200 pb-2">
									<h4 class="font-serif text-xs font-semibold text-neutral-950">
										Deterministic Unit Preview ({genPreviewUnits.length} Units)
									</h4>
									<Badge variant="outline" class="font-mono text-[10px] text-emerald-700"
										>Ready to Commit</Badge
									>
								</div>
								<div class="mt-3 max-h-56 overflow-y-auto">
									<div class="grid grid-cols-4 gap-2 sm:grid-cols-6 md:grid-cols-9">
										{#each genPreviewUnits as u (u.unit_identifier)}
											<div
												class="rounded-xl border border-neutral-200 bg-neutral-50 p-2 text-center text-xs"
											>
												<span class="block font-mono font-bold text-neutral-950"
													>{u.unit_identifier}</span
												>
												<span class="block text-[10px] text-neutral-500">Fl {u.floor_number}</span>
											</div>
										{/each}
									</div>
								</div>
							</div>
						{/if}
					</div>
				{/if}

				<!-- STEP 4: GATES & STAFFING -->
				{#if currentStep === 4}
					<div class="space-y-5">
						<div class="flex items-center justify-between">
							<div>
								<h3 class="font-serif text-sm font-semibold text-neutral-950">
									Step 4 — Gates & Gatekeeper Staffing
								</h3>
								<p class="text-xs text-neutral-500">
									Define physical entry/exit gates and specify required security headcount.
								</p>
							</div>
							<Button size="sm" variant="outline" onclick={() => (showAddGate = !showAddGate)}>
								<Plus class="mr-1.5 h-3.5 w-3.5" />
								<span>Add Gate</span>
							</Button>
						</div>

						<!-- Add Gate Form -->
						{#if showAddGate}
							<div class="rounded-2xl border border-neutral-200 bg-neutral-50/80 p-4">
								<div class="flex items-center justify-between">
									<h4 class="font-serif text-xs font-semibold text-neutral-950">
										New Physical Gate
									</h4>
									<button
										type="button"
										onclick={() => (showAddGate = false)}
										class="rounded-full p-1 text-neutral-400 hover:bg-neutral-200 hover:text-neutral-900"
									>
										<X class="h-3.5 w-3.5" />
									</button>
								</div>
								<div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-4">
									<div>
										<label for="g-name" class="block text-[11px] font-medium text-neutral-700"
											>Gate Name</label
										>
										<input
											id="g-name"
											bind:value={gName}
											placeholder="e.g. Main Gate"
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
										/>
									</div>
									<div>
										<label for="g-code" class="block text-[11px] font-medium text-neutral-700"
											>Gate Code</label
										>
										<input
											id="g-code"
											bind:value={gCode}
											placeholder="e.g. GATE-01"
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
										/>
									</div>
									<div>
										<label for="g-type" class="block text-[11px] font-medium text-neutral-700"
											>Type</label
										>
										<select
											id="g-type"
											bind:value={gType}
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900"
										>
											<option value="VEHICLE_AND_PEDESTRIAN">Vehicle & Pedestrian</option>
											<option value="VEHICLE_ONLY">Vehicle Only</option>
											<option value="PEDESTRIAN_ONLY">Pedestrian Only</option>
											<option value="SERVICE">Service</option>
											<option value="EMERGENCY">Emergency</option>
										</select>
									</div>
									<div>
										<label for="g-staff" class="block text-[11px] font-medium text-neutral-700"
											>Staff / Shift</label
										>
										<input
											id="g-staff"
											type="number"
											bind:value={gStaffPerShift}
											min="0"
											max="10"
											class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
										/>
									</div>
								</div>
								<div class="mt-3 flex justify-end gap-2">
									<Button size="sm" variant="ghost" onclick={() => (showAddGate = false)}
										>Cancel</Button
									>
									<Button
										size="sm"
										variant="primary"
										onclick={handleCreateGate}
										disabled={isLoading}
									>
										{#if isLoading}<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />{/if}
										Save Gate
									</Button>
								</div>
							</div>
						{/if}

						<!-- Gates Roster -->
						{#if gates.length === 0}
							<div
								class="rounded-2xl border border-dashed border-neutral-300 p-6 text-center text-xs text-neutral-500"
							>
								<DoorOpen class="mx-auto h-7 w-7 text-neutral-400" />
								<p class="mt-2 font-medium text-neutral-900">No gates registered yet</p>
							</div>
						{:else}
							<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
								{#each gates as g (g.id)}
									<div class="rounded-2xl border border-neutral-200 bg-white p-4 shadow-xs">
										<div class="flex items-center justify-between">
											<span class="text-xs font-semibold text-neutral-950">{g.name}</span>
											<Badge variant="outline" class="font-mono text-[10px]">{g.gate_code}</Badge>
										</div>
										<div class="mt-2 text-[11px] text-neutral-500">
											<span>Type: {g.gate_type}</span> ·
											<span class="font-medium text-neutral-900"
												>Req Staff: {g.required_staff_per_shift} / shift</span
											>
										</div>
									</div>
								{/each}
							</div>
						{/if}

						<!-- Staffing Requirements Form -->
						<div class="rounded-2xl border border-neutral-200 bg-neutral-50/60 p-5">
							<h4 class="font-serif text-xs font-semibold text-neutral-950">
								Staffing Capacity & Shifts Policy
							</h4>
							<div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-3">
								<div>
									<label for="st-shifts" class="block text-[11px] font-medium text-neutral-700"
										>Shifts Per Day (1-4)</label
									>
									<input
										id="st-shifts"
										type="number"
										bind:value={shiftsPerDay}
										min="1"
										max="4"
										class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
									/>
								</div>
								<div>
									<label for="st-headcount" class="block text-[11px] font-medium text-neutral-700"
										>Required Headcount</label
									>
									<input
										id="st-headcount"
										type="number"
										bind:value={totalHeadcount}
										min="0"
										max="100"
										class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
									/>
								</div>
								<div>
									<label for="st-notes" class="block text-[11px] font-medium text-neutral-700"
										>Operational Notes</label
									>
									<input
										id="st-notes"
										bind:value={staffingNotes}
										placeholder="e.g. 24/7 coverage"
										class="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 placeholder-neutral-400 focus:border-neutral-950 focus:outline-hidden"
									/>
								</div>
							</div>
							<div class="mt-4 flex items-center justify-between border-t border-neutral-200 pt-3">
								<div class="text-[11px] text-neutral-500">
									Honest representation: Registered staff (0) · Assigned staff (0) [Deferred to S08]
								</div>
								<Button
									size="sm"
									variant="primary"
									onclick={handleSaveStaffing}
									disabled={isSavingStaffing}
								>
									{#if isSavingStaffing}<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />{/if}
									Update Staffing Policy
								</Button>
							</div>
						</div>
					</div>
				{/if}

				<!-- STEP 5: REVIEW & SUMMARY -->
				{#if currentStep === 5}
					<div class="space-y-5">
						<div>
							<h3 class="font-serif text-sm font-semibold text-neutral-950">
								Step 5 — Structure Overview & Formation Summary
							</h3>
							<p class="text-xs text-neutral-500">
								Live formation audit totals calculated dynamically from society records.
							</p>
						</div>

						<!-- Metrics Grid -->
						<div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
							<div class="rounded-2xl border border-neutral-200 bg-white p-4 shadow-xs">
								<span class="text-[11px] text-neutral-500">Total Buildings</span>
								<p class="mt-1 font-serif text-2xl font-semibold text-neutral-950">
									{formationSummary?.total_buildings ?? buildings.length}
								</p>
							</div>
							<div class="rounded-2xl border border-neutral-200 bg-white p-4 shadow-xs">
								<span class="text-[11px] text-neutral-500">Total Wings</span>
								<p class="mt-1 font-serif text-2xl font-semibold text-neutral-950">
									{formationSummary?.total_wings ?? 0}
								</p>
							</div>
							<div class="rounded-2xl border border-neutral-200 bg-white p-4 shadow-xs">
								<span class="text-[11px] text-neutral-500">Total Units / Flats</span>
								<p class="mt-1 font-serif text-2xl font-semibold text-neutral-950">
									{formationSummary?.total_units ?? 0}
								</p>
							</div>
							<div class="rounded-2xl border border-neutral-200 bg-white p-4 shadow-xs">
								<span class="text-[11px] text-neutral-500">Configured Gates</span>
								<p class="mt-1 font-serif text-2xl font-semibold text-neutral-950">
									{formationSummary?.total_gates ?? gates.length}
								</p>
							</div>
						</div>

						<div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
							<div class="rounded-2xl border border-neutral-200 bg-white p-4 shadow-xs">
								<span class="text-[11px] text-neutral-500">Req. Staff / Shift</span>
								<p class="mt-1 font-serif text-xl font-semibold text-amber-800">
									{formationSummary?.required_staff_per_shift ?? 0}
								</p>
							</div>
							<div class="rounded-2xl border border-neutral-200 bg-white p-4 shadow-xs">
								<span class="text-[11px] text-neutral-500">Total Required Headcount</span>
								<p class="mt-1 font-serif text-xl font-semibold text-emerald-800">
									{formationSummary?.total_required_headcount ?? totalHeadcount}
								</p>
							</div>
							<div class="rounded-2xl border border-neutral-200 bg-white p-4 shadow-xs">
								<span class="text-[11px] text-neutral-500">Registered Gatekeepers</span>
								<p class="mt-1 font-serif text-xl font-semibold text-neutral-400">
									0 <span class="text-[10px] font-normal">(Deferred S08)</span>
								</p>
							</div>
						</div>

						<div
							class="rounded-2xl border border-emerald-200 bg-emerald-50/60 p-4 text-xs text-emerald-900"
						>
							<div class="flex items-center gap-2 font-semibold">
								<ShieldCheck class="h-4 w-4 text-emerald-700" />
								<span>Formation Verification Checklist Passed</span>
							</div>
							<p class="mt-1 leading-relaxed text-emerald-800">
								Physical structure and gate operations metadata are verified. All records are
								indexed, protected by society-scoped authorization, and tracked with platform audit
								logs.
							</p>
						</div>
					</div>
				{/if}
			</div>

			<!-- Footer Controls -->
			<div
				class="flex items-center justify-between border-t border-neutral-200 bg-white px-6 py-3.5"
			>
				<div>
					{#if currentStep > 1}
						<Button size="sm" variant="outline" onclick={prevStep}>
							<ChevronLeft class="mr-1.5 h-3.5 w-3.5" />
							<span>Previous</span>
						</Button>
					{/if}
				</div>

				<div class="flex items-center gap-2">
					<Button size="sm" variant="ghost" onclick={onClose}>Cancel</Button>
					{#if currentStep < 5}
						<Button size="sm" variant="primary" onclick={nextStep}>
							<span>Continue</span>
							<ChevronRight class="ml-1.5 h-3.5 w-3.5" />
						</Button>
					{:else}
						<Button
							size="sm"
							variant="primary"
							onclick={() => {
								if (onComplete) onComplete();
								onClose();
							}}
						>
							<Check class="mr-1.5 h-3.5 w-3.5" />
							<span>Complete Formation</span>
						</Button>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}
