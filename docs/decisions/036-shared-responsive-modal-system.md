# ADR 036: Shared Responsive Modal System and Adaptive Presentation

## Status
Accepted

## Context
Across CHS Neu, various components previously created custom or ad-hoc modal dialogs, leading to inconsistencies in positioning, mobile responsiveness, focus management, and accessibility. Furthermore, desktop dialogs in the application follow a specific editorial positioning aesthetic: they are not vertically centered, but rather top-aligned with an exact offset from the top of the viewport.

To provide a consistent, production-grade dialog experience across the entire application, CHS Neu required a shared, reusable responsive modal component (`ResponsiveModal`) that satisfies the following core architectural requirements:

1. **Desktop and Tablet Top-Alignment Contract**:
   - The modal dialog must be horizontally centered (`left: 50%; transform: translateX(-50%);`).
   - Its top edge must be positioned **strictly 180px from the top of the viewport** (`top: 180px;`). It must never be vertically centered.
   - Max width is bounded by `w-[min(520px, calc(100vw - 48px))]` with configurable presets (`sm`, `md`, `lg`, `xl`).
   - Max height is bounded by `calc(100vh - 220px)` with internal scrolling to ensure content remains accessible across diverse viewport heights.

2. **Mobile Bottom Sheet Presentation**:
   - On mobile viewports, the modal automatically transitions to an anchored bottom sheet (`position: fixed; inset-x: 0; bottom: 0;`).
   - Spans full width with rounded top corners (`rounded-t-[2rem]`).
   - Respects safe-area insets via `pb-[max(1.25rem, env(safe-area-inset-bottom))]`.
   - Includes subtle top drag handle indicator and vertical overflow scrolling bounded to `max-h-[85vh]`.

3. **Unified Responsive Abstraction (`screen.isMobile`)**:
   - Leverages the existing shared reactive store `screen` (`src/lib/screen.svelte.ts`) and its `isMobile` property (`width < 768px`).
   - Forbids introducing parallel width listeners, ad-hoc media query listeners, or competing breakpoint enums.

4. **Production-Grade Accessibility & Focus Management**:
   - Accessible dialog semantics: `role="dialog"`, `aria-modal="true"`, `aria-labelledby`, `aria-describedby`.
   - Automatic focus entry into the first focusable element (or dialog panel) on mount.
   - Full keyboard focus trapping within the dialog on `Tab` and `Shift+Tab`.
   - Focus restoration to the previously active element upon dialog dismissal.
   - Global body scroll locking (`overflow: hidden`) during presentation, with automatic restoration upon unmount/closure.
   - Escape key dismissal and backdrop click dismissal, both safely disabled when non-dismissible operations are running.

5. **Visual Identity & Design Tokens**:
   - Adheres strictly to the established CHS typography standard: **`Hedvig Letters Sans`** and **`Hedvig Letters Serif`**.
   - Near-white surface (`bg-white`), hairline borders (`border-neutral-200/90`), and dimmed backdrop (`bg-neutral-950/45` with subtle `backdrop-blur-2xs`).
   - Destructive mode support providing tailored borders (`border-rose-200/90`) and subtle shadows.

---

## Decision

### 1. Created Shared `ResponsiveModal` Component
Created `frontend/src/lib/components/ui/responsive-modal.svelte`:
- **Props**:
  - `isOpen: boolean`
  - `onClose: () => void`
  - `title?: string`
  - `description?: string`
  - `icon?: Snippet`
  - `headerActions?: Snippet`
  - `footer?: Snippet`
  - `children?: Snippet`
  - `maxWidth?: 'sm' | 'md' | 'lg' | 'xl'`
  - `isDismissible?: boolean` (defaults to `true`)
  - `closeOnBackdrop?: boolean` (defaults to `true`)
  - `closeOnEscape?: boolean` (defaults to `true`)
  - `showCloseButton?: boolean` (defaults to `true`)
  - `destructive?: boolean` (defaults to `false`)
  - `ariaLabel?: string`
- **Transitions**:
  - Desktop: Subtle vertical fly (`y: -16, duration: 180`) preserving the exact `top: 180px` resting position.
  - Mobile: Bottom-up slide (`y: 250, duration: 220`).
  - Backdrop: Soft fade (`duration: 150`).

### 2. Migrated Global Session Revocation (`LogoutAllDialog`)
Refactored `frontend/src/lib/components/auth/logout-all-dialog.svelte`:
- Wrapped with `ResponsiveModal`.
- Configured with `destructive={true}`, `isDismissible={!isLoading}`, `maxWidth="md"`.
- Prevents concurrent submissions via `isLoading` guard.
- Handles API errors via structured alert banner.
- Preserves backend session revocation guarantees and redirects to `/login` upon success.

### 3. Migrated Administrative Confirmation Dialogs
Standardized administrative status dialogs to use `ResponsiveModal`:
- `frontend/src/lib/components/admin/status-dialog.svelte` (Platform admin status enable/disable).
- `frontend/src/lib/components/admin/society-status-dialog.svelte` (Society suspension and reactivation review).

---

## Consequences

### Positive
- Universal visual and behavioural consistency for all confirmation and operational dialogs across CHS Neu.
- Desktop viewports reliably adhere to the strict 180px top offset specification.
- Mobile viewports receive an ergonomic, safe-area-aware bottom sheet without duplicate markup.
- Accessible focus trapping, keyboard navigation, and scroll locking are handled centrally in a single primitive.
- Zero competing mobile-detection systems or additional external npm dependencies.
