<script lang="ts">
	import type { Snippet } from 'svelte';
	import { screen } from '$lib/screen.svelte';
	import { X } from '@lucide/svelte';
	import { fly, fade } from 'svelte/transition';

	interface Props {
		isOpen: boolean;
		onClose: () => void;
		title?: string;
		description?: string;
		icon?: Snippet;
		headerActions?: Snippet;
		footer?: Snippet;
		children?: Snippet;
		maxWidth?: 'sm' | 'md' | 'lg' | 'xl';
		isDismissible?: boolean;
		closeOnBackdrop?: boolean;
		closeOnEscape?: boolean;
		showCloseButton?: boolean;
		destructive?: boolean;
		ariaLabel?: string;
	}

	let {
		isOpen = false,
		onClose,
		title,
		description,
		icon,
		headerActions,
		footer,
		children,
		maxWidth = 'md',
		isDismissible = true,
		closeOnBackdrop = true,
		closeOnEscape = true,
		showCloseButton = true,
		destructive = false,
		ariaLabel
	}: Props = $props();

	let previouslyFocused = $state<HTMLElement | null>(null);
	let panelRef = $state<HTMLDivElement | null>(null);

	// Width mappings for desktop / tablet dialog panels
	const maxWidthClasses = {
		sm: 'sm:max-w-sm',
		md: 'sm:max-w-md',
		lg: 'sm:max-w-lg',
		xl: 'sm:max-w-xl'
	};

	// Focus management and body scroll locking
	$effect(() => {
		if (isOpen) {
			if (typeof document !== 'undefined') {
				previouslyFocused = document.activeElement as HTMLElement | null;
				// Prevent underlying page scrolling
				const originalOverflow = document.body.style.overflow;
				document.body.style.overflow = 'hidden';

				// Auto-focus the dialog panel without unintended page scrolling
				setTimeout(() => {
					if (panelRef) {
						const focusable = panelRef.querySelector<HTMLElement>(
							'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
						);
						if (focusable) {
							focusable.focus({ preventScroll: true });
						} else {
							panelRef.focus({ preventScroll: true });
						}
					}
				}, 50);

				return () => {
					document.body.style.overflow = originalOverflow;
					if (previouslyFocused && typeof previouslyFocused.focus === 'function') {
						previouslyFocused.focus({ preventScroll: true });
					}
				};
			}
		}
	});

	function handleKeyDown(e: KeyboardEvent) {
		if (!isOpen) return;

		if (e.key === 'Escape' && isDismissible && closeOnEscape) {
			e.stopPropagation();
			onClose();
			return;
		}

		// Focus trapping inside active modal
		if (e.key === 'Tab' && panelRef) {
			const focusables = Array.from(
				panelRef.querySelectorAll<HTMLElement>(
					'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
				)
			).filter((el) => el.offsetParent !== null);

			if (focusables.length === 0) {
				e.preventDefault();
				return;
			}

			const first = focusables[0];
			const last = focusables[focusables.length - 1];

			if (e.shiftKey && document.activeElement === first) {
				e.preventDefault();
				last.focus({ preventScroll: true });
			} else if (!e.shiftKey && document.activeElement === last) {
				e.preventDefault();
				first.focus({ preventScroll: true });
			}
		}
	}

	function handleBackdropClick() {
		if (isDismissible && closeOnBackdrop) {
			onClose();
		}
	}
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if isOpen}
	<div
		class="fixed inset-0 z-50 overflow-x-hidden overflow-y-auto"
		role="dialog"
		aria-modal="true"
		aria-label={ariaLabel || title || 'Dialog'}
		aria-labelledby={title ? 'responsive-modal-title' : undefined}
		aria-describedby={description ? 'responsive-modal-desc' : undefined}
	>
		<!-- Dimmed Backdrop -->
		<button
			type="button"
			tabindex="-1"
			transition:fade={{ duration: 150 }}
			onclick={handleBackdropClick}
			class="backdrop-blur-2xs fixed inset-0 h-full w-full cursor-default border-0 bg-neutral-950/45"
			aria-label="Close dialog overlay"
		></button>

		<!-- Responsive Presentation:
		     - Mobile (< 768px via screen.isMobile): Bottom Sheet anchored to bottom (bottom: 0, left: 0, right: 0)
		     - Desktop/Tablet (>= 768px): Fixed 180px top offset, horizontally centered (top: 180px, left: 50%, -translate-x-1/2)
		-->
		{#if screen.isMobile}
			<!-- Mobile Bottom Sheet -->
			<div
				bind:this={panelRef}
				tabindex="-1"
				transition:fly={{ y: 250, duration: 220 }}
				class="fixed inset-x-0 bottom-0 z-10 flex max-h-[85vh] flex-col rounded-t-[2rem] border-t bg-white p-5 pb-[max(1.25rem,env(safe-area-inset-bottom))] text-neutral-900 shadow-2xl outline-hidden {destructive
					? 'border-rose-200/90'
					: 'border-neutral-200/90'}"
			>
				<!-- Subtle drag handle indicator -->
				<div class="mx-auto mb-3 h-1 w-10 shrink-0 rounded-full bg-neutral-300/80"></div>

				<!-- Header -->
				{#if title || icon || showCloseButton}
					<div class="flex items-center justify-between border-b border-neutral-100 pb-3">
						<div class="flex items-center gap-2.5">
							{#if icon}
								{@render icon()}
							{/if}
							<div>
								{#if title}
									<h2
										id="responsive-modal-title"
										class="font-serif text-base font-semibold tracking-tight text-neutral-950"
									>
										{title}
									</h2>
								{/if}
								{#if description}
									<p id="responsive-modal-desc" class="text-xs text-neutral-500">
										{description}
									</p>
								{/if}
							</div>
						</div>

						<div class="flex items-center gap-1.5">
							{#if headerActions}
								{@render headerActions()}
							{/if}
							{#if showCloseButton && isDismissible}
								<button
									type="button"
									onclick={onClose}
									class="rounded-full p-2 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-900 focus:ring-2 focus:ring-neutral-950 focus:outline-hidden"
									aria-label="Close dialog"
								>
									<X class="h-4 w-4" />
								</button>
							{/if}
						</div>
					</div>
				{/if}

				<!-- Content Body -->
				<div class="mt-3 flex-1 overflow-y-auto pr-1">
					{#if children}
						{@render children()}
					{/if}
				</div>

				<!-- Footer Actions -->
				{#if footer}
					<div class="mt-4 border-t border-neutral-100 pt-3">
						{@render footer()}
					</div>
				{/if}
			</div>
		{:else}
			<!-- Desktop / Tablet Top-Aligned Dialog: Exactly 180px from viewport top, horizontally centered -->
			<div
				bind:this={panelRef}
				tabindex="-1"
				transition:fly={{ y: -16, duration: 180 }}
				style="top: 180px; left: 50%; transform: translateX(-50%);"
				class="fixed z-10 flex max-h-[calc(100vh-220px)] w-[min(520px,calc(100vw-48px))] {maxWidthClasses[
					maxWidth
				]} flex-col rounded-3xl border bg-white p-6 text-neutral-900 shadow-2xl outline-hidden {destructive
					? 'border-rose-200/90 shadow-rose-950/5'
					: 'border-neutral-200/90'}"
			>
				<!-- Header -->
				{#if title || icon || showCloseButton}
					<div class="flex items-center justify-between border-b border-neutral-100 pb-4">
						<div class="flex items-center gap-2.5">
							{#if icon}
								{@render icon()}
							{/if}
							<div>
								{#if title}
									<h2
										id="responsive-modal-title"
										class="font-serif text-base font-semibold tracking-tight text-neutral-950"
									>
										{title}
									</h2>
								{/if}
								{#if description}
									<p id="responsive-modal-desc" class="text-xs text-neutral-500">
										{description}
									</p>
								{/if}
							</div>
						</div>

						<div class="flex items-center gap-1.5">
							{#if headerActions}
								{@render headerActions()}
							{/if}
							{#if showCloseButton && isDismissible}
								<button
									type="button"
									onclick={onClose}
									class="rounded-full p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-900 focus:ring-2 focus:ring-neutral-950 focus:outline-hidden"
									aria-label="Close dialog"
								>
									<X class="h-4 w-4" />
								</button>
							{/if}
						</div>
					</div>
				{/if}

				<!-- Content Body -->
				<div class="mt-4 flex-1 overflow-y-auto pr-1">
					{#if children}
						{@render children()}
					{/if}
				</div>

				<!-- Footer Actions -->
				{#if footer}
					<div class="mt-4 border-t border-neutral-100 pt-3">
						{@render footer()}
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/if}
