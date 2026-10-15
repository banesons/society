<script lang="ts">
	import Badge from '$lib/components/ui/badge.svelte';
	import { Building2, Users, ShieldCheck, Check } from '@lucide/svelte';
</script>

<div
	class="relative hidden flex-col justify-between overflow-hidden rounded-[2rem] bg-neutral-950 p-8 text-white sm:p-10 md:flex lg:col-span-6 lg:p-12"
>
	<!-- Ambient lighting gradients -->
	<div
		class="pointer-events-none absolute -top-24 -left-24 h-96 w-96 rounded-full bg-neutral-800/40 blur-3xl"
	></div>
	<div
		class="pointer-events-none absolute -right-24 -bottom-24 h-96 w-96 rounded-full bg-neutral-700/20 blur-3xl"
	></div>

	<!-- Top Branding -->
	<div class="relative z-10 flex items-center justify-between">
		<div class="flex items-center gap-2">
			<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-white text-neutral-950">
				<Building2 class="h-4 w-4" />
			</div>
			<span class="font-serif text-sm font-semibold tracking-wide text-neutral-200">
				chs. Platform
			</span>
		</div>
		<Badge
			variant="outline"
			class="border-neutral-700 bg-neutral-900/80 font-sans text-[11px] text-neutral-300"
		>
			Verified & secure
		</Badge>
	</div>

	<!-- Center: Layered 3D-styled UI Cards & Abstract Composition -->
	<div class="relative z-10 my-10 flex flex-col items-center justify-center">
		<!-- Decorative wireframe / subtle background shapes -->
		<div class="pointer-events-none absolute inset-0 flex items-center justify-center opacity-10">
			<div class="h-72 w-72 rounded-full border border-dashed border-white"></div>
			<div class="absolute h-96 w-96 rounded-full border border-white"></div>
		</div>

		<div class="relative w-full max-w-md space-y-4">
			<!-- Floating Card 1: Top stats badge -->
			<div
				class="ml-auto w-11/12 rounded-2xl border border-neutral-700/80 bg-neutral-900/90 p-4 shadow-xl backdrop-blur-md transition-transform hover:-translate-y-1 sm:w-4/5"
			>
				<div class="flex items-center justify-between">
					<span class="font-sans text-[11px] font-medium text-neutral-400">Managed Societies</span>
					<span
						class="rounded-full bg-emerald-500/20 px-2 py-0.5 font-sans text-[10px] font-semibold text-emerald-400"
					>
						Active
					</span>
				</div>
				<div class="mt-2 flex items-baseline justify-between">
					<p class="font-serif text-2xl font-bold tracking-tight text-white">2,480 Units</p>
					<span class="font-sans text-xs text-neutral-400">99.9% Uptime</span>
				</div>
				<div class="mt-2.5 h-1.5 w-full overflow-hidden rounded-full bg-neutral-800">
					<div class="h-full w-4/5 rounded-full bg-white"></div>
				</div>
			</div>

			<!-- Floating Card 2: Main feature card -->
			<div
				class="relative z-20 rounded-3xl border border-neutral-200 bg-white p-5 text-neutral-900 shadow-2xl transition-transform hover:-translate-y-1"
			>
				<div class="flex items-center justify-between">
					<Badge
						variant="outline"
						class="border-neutral-300 bg-neutral-100 font-sans text-[10px] text-neutral-700"
					>
						Resident Community
					</Badge>
					<div class="flex items-center gap-1 font-sans text-[11px] font-medium text-neutral-500">
						<ShieldCheck class="h-3.5 w-3.5 text-neutral-900" />
						<span>Safe & verified</span>
					</div>
				</div>

				<div class="mt-4">
					<h3 class="font-serif text-lg font-bold tracking-tight text-neutral-950">
						Residential Governance Made Simple
					</h3>
					<p class="mt-1 font-sans text-xs leading-relaxed text-neutral-600">
						Private, transparent communication and verified access for community members and
						families.
					</p>
				</div>

				<div
					class="mt-4 flex items-center justify-between border-t border-neutral-100 pt-3 font-sans text-[11px]"
				>
					<span class="text-neutral-500">Protected account</span>
					<span class="font-semibold text-neutral-950">Verified residents only</span>
				</div>
			</div>

			<!-- Floating Card 3: Gate & Resident Badge -->
			<div
				class="mr-auto flex w-11/12 items-center justify-between rounded-2xl border border-neutral-800 bg-neutral-900/95 p-3.5 shadow-xl sm:w-3/4"
			>
				<div class="flex items-center gap-3">
					<div
						class="flex h-8 w-8 items-center justify-center rounded-xl bg-neutral-800 text-white"
					>
						<Users class="h-4 w-4" />
					</div>
					<div>
						<p class="font-sans text-xs font-semibold text-white">Verified Residents</p>
						<p class="font-sans text-[10px] text-neutral-400">Instant, secure sign-in</p>
					</div>
				</div>
				<div
					class="flex h-6 w-6 items-center justify-center rounded-full bg-emerald-500/20 text-emerald-400"
				>
					<Check class="h-3.5 w-3.5" />
				</div>
			</div>
		</div>
	</div>

	<!-- Bottom Note -->
	<div
		class="relative z-10 flex items-center justify-between border-t border-neutral-800/80 pt-4 font-sans text-xs text-neutral-400"
	>
		<span>Designed for modern residential communities</span>
		<span class="font-sans text-[11px]">v1.0.0</span>
	</div>
</div>
