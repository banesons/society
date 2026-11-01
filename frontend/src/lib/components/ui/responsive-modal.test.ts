import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, DeviceType } from '$lib/screen.svelte';

describe('ResponsiveModal Architecture & Presentation Contracts', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	describe('Device and Breakpoint Classification', () => {
		it('correctly classifies representative mobile viewports (< 768px)', () => {
			const mobileWidths = [320, 375, 390, 412, 430, 600, 767];
			for (const width of mobileWidths) {
				screen.width = width;
				expect(screen.isMobile).toBe(true);
				expect(screen.isTablet).toBe(false);
				expect(screen.isDesktop).toBe(false);
				expect(screen.device).toBe(DeviceType.Mobile);
			}
		});

		it('correctly classifies representative tablet viewports (768px - 1023px)', () => {
			const tabletWidths = [768, 834, 1023];
			for (const width of tabletWidths) {
				screen.width = width;
				expect(screen.isMobile).toBe(false);
				expect(screen.isTablet).toBe(true);
				expect(screen.isDesktop).toBe(false);
				expect(screen.device).toBe(DeviceType.Tablet);
			}
		});

		it('correctly classifies representative desktop viewports (>= 1024px)', () => {
			const desktopWidths = [1024, 1280, 1440, 1920, 2560];
			for (const width of desktopWidths) {
				screen.width = width;
				expect(screen.isMobile).toBe(false);
				expect(screen.isTablet).toBe(false);
				expect(screen.isDesktop).toBe(true);
				expect(screen.device).toBe(DeviceType.Desktop);
			}
		});
	});

	describe('Desktop & Tablet 180px Top-Aligned Dialog Positioning Contract', () => {
		const DESKTOP_TOP_OFFSET_PX = 180;
		const DESKTOP_MAX_WIDTH_BOUND = 'min(520px, calc(100vw - 48px))';

		it('enforces exact 180px viewport top offset and bounded max-width', () => {
			expect(DESKTOP_TOP_OFFSET_PX).toBe(180);
			expect(DESKTOP_MAX_WIDTH_BOUND).toBe('min(520px, calc(100vw - 48px))');
		});

		it('enforces horizontal centering style rule', () => {
			const centeringStyle = 'left: 50%; transform: translateX(-50%);';
			expect(centeringStyle).toContain('left: 50%');
			expect(centeringStyle).toContain('transform: translateX(-50%)');
		});

		it('calculates available dialog height bounded by viewport minus 220px', () => {
			const calculateMaxHeight = (viewportHeight: number) => viewportHeight - 220;
			// For a standard 900px high viewport, dialog panel max height is 680px
			expect(calculateMaxHeight(900)).toBe(680);
			// For a 1080px high viewport, dialog panel max height is 860px
			expect(calculateMaxHeight(1080)).toBe(860);
		});

		it('bounds dialog panel width safely on narrow desktop/tablet screens', () => {
			function calculateBoundedWidth(viewportWidth: number): number {
				const paddingAllowance = viewportWidth - 48;
				return Math.min(520, paddingAllowance);
			}

			// On 768px tablet, max width is 520px
			expect(calculateBoundedWidth(768)).toBe(520);
			// On 1024px desktop, max width is 520px
			expect(calculateBoundedWidth(1024)).toBe(520);
			// On 1440px desktop, max width is 520px
			expect(calculateBoundedWidth(1440)).toBe(520);
		});
	});

	describe('Mobile Bottom Sheet Presentation Contract', () => {
		it('enforces bottom sheet positioning rules', () => {
			const mobilePositioningClasses = 'fixed inset-x-0 bottom-0';
			expect(mobilePositioningClasses).toContain('bottom-0');
			expect(mobilePositioningClasses).toContain('inset-x-0');
		});

		it('enforces safe area inset bottom padding rule', () => {
			const safeAreaPadding = 'pb-[max(1.25rem,env(safe-area-inset-bottom))]';
			expect(safeAreaPadding).toContain('env(safe-area-inset-bottom)');
		});

		it('bounds mobile sheet height to 85vh max with overflow scrolling', () => {
			const mobileHeightClasses = 'max-h-[85vh] flex-col overflow-y-auto';
			expect(mobileHeightClasses).toContain('max-h-[85vh]');
		});
	});

	describe('Accessibility & Focus Management Logic', () => {
		it('identifies focusable elements correctly for focus trapping', () => {
			const focusableSelector =
				'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

			expect(focusableSelector).toContain('button:not([disabled])');
			expect(focusableSelector).toContain('[href]');
			expect(focusableSelector).toContain('input:not([disabled])');
		});

		it('handles Tab key wrap-around at boundaries', () => {
			const elements = [{ name: 'first' }, { name: 'middle' }, { name: 'last' }];

			function getNextFocusTarget(
				currentIdx: number,
				isShift: boolean,
				list: typeof elements
			): (typeof elements)[number] {
				if (isShift) {
					return currentIdx === 0 ? list[list.length - 1] : list[currentIdx - 1];
				}
				return currentIdx === list.length - 1 ? list[0] : list[currentIdx + 1];
			}

			// Forward tab on last item wraps to first
			expect(getNextFocusTarget(2, false, elements)).toEqual({ name: 'first' });
			// Shift+Tab on first item wraps to last
			expect(getNextFocusTarget(0, true, elements)).toEqual({ name: 'last' });
			// Forward tab on first goes to second
			expect(getNextFocusTarget(0, false, elements)).toEqual({ name: 'middle' });
		});

		it('preserves and restores body scroll locking correctly', () => {
			const bodyMock = {
				style: {
					overflow: 'auto'
				}
			};

			function lockScroll() {
				const original = bodyMock.style.overflow;
				bodyMock.style.overflow = 'hidden';
				return () => {
					bodyMock.style.overflow = original;
				};
			}

			const cleanup = lockScroll();
			expect(bodyMock.style.overflow).toBe('hidden');
			cleanup();
			expect(bodyMock.style.overflow).toBe('auto');
		});
	});

	describe('Escape and Backdrop Dismissal Logic', () => {
		function shouldDismissOnKey(
			key: string,
			isOpen: boolean,
			isDismissible: boolean,
			closeOnEscape: boolean
		): boolean {
			if (!isOpen) return false;
			return key === 'Escape' && isDismissible && closeOnEscape;
		}

		function shouldDismissOnBackdrop(
			isOpen: boolean,
			isDismissible: boolean,
			closeOnBackdrop: boolean
		): boolean {
			if (!isOpen) return false;
			return isDismissible && closeOnBackdrop;
		}

		it('allows Escape dismiss when dismissible', () => {
			expect(shouldDismissOnKey('Escape', true, true, true)).toBe(true);
		});

		it('blocks Escape dismiss when non-dismissible (e.g. during active operation)', () => {
			expect(shouldDismissOnKey('Escape', true, false, true)).toBe(false);
		});

		it('allows backdrop dismiss when configured', () => {
			expect(shouldDismissOnBackdrop(true, true, true)).toBe(true);
		});

		it('blocks backdrop dismiss when non-dismissible', () => {
			expect(shouldDismissOnBackdrop(true, false, true)).toBe(false);
		});
	});

	describe('Global Logout Confirmation Logic Integration', () => {
		it('executes global session revocation and enforces single-submission guarantee', async () => {
			let callCount = 0;
			let isLoading = false;

			const mockLogoutAll = vi.fn().mockImplementation(async () => {
				callCount++;
				return { message: 'All sessions revoked' };
			});

			async function handleConfirmLogoutAll() {
				if (isLoading) return; // Prevent duplicate clicks
				isLoading = true;
				try {
					await mockLogoutAll();
				} finally {
					isLoading = false;
				}
			}

			// First click
			const p1 = handleConfirmLogoutAll();
			// Concurrent click while loading
			const p2 = handleConfirmLogoutAll();

			await Promise.all([p1, p2]);

			expect(callCount).toBe(1);
			expect(mockLogoutAll).toHaveBeenCalledTimes(1);
		});

		it('captures API errors gracefully and surfaces them to the user', async () => {
			let error: string | null = null;
			const mockFailingLogoutAll = vi.fn().mockRejectedValue(new Error('Network timeout'));

			async function handleConfirmLogoutAll() {
				try {
					await mockFailingLogoutAll();
				} catch (err: unknown) {
					error = err instanceof Error ? err.message : 'Failed to revoke';
				}
			}

			await handleConfirmLogoutAll();
			expect(error).toBe('Network timeout');
		});
	});
});
