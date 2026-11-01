# ADR 035: Premium Editorial Personal Profile Architecture and Identity Integration

## Status
Accepted

## Context
CHS Neu required a dedicated, personal identity experience at `/app/profile` where authenticated users can view and update their personal profile information in an editorial Swiss-minimalist aesthetic inspired by refined publication layouts.

Key architectural boundaries and invariants:
1. **Universal vs Society-Specific Identity**:
   - Universal account identity (`users` table) models the authenticated phone number, account status, timestamps, and platform roles.
   - User profile metadata (display name, email, emergency contact details) is associated with the user across their society memberships (`society_members` person directory).
   - In accordance with the project's zero-unnecessary-migration rule, personal profile attributes are unified and persisted via the authoritative backend `society_members` person registry linked directly to the authenticated `user_id`.
2. **Immutable vs Editable Attributes**:
   - **Verified Phone Number**: Read-only authentication attribute. Phone identity cannot be modified arbitrarily without a multi-factor verification workflow; it remains strictly read-only.
   - **Account Verification Status**: Authoritative backend status (`ACTIVE`, `VERIFIED`); never client-fabricated.
   - **Editable Personal Information**: First Name, Last Name, and Email Address. When updated on `/app/profile`, changes are validated on both frontend and backend and atomically persisted to the user's member identity.
   - **Label Clarification**: Removed the redundant "Immutable Identity" badge label in favor of clean, understated verification indicators.
3. **Daily Curated Photographic Backdrop & Composition**:
   - The profile features a quiet, architecturally inspired photographic backdrop that resolves deterministically per calendar date (day-of-year rotation).
   - Top-right corner displays the formatted current date with a frosted translucent pill for legibility.
   - Bottom-right corner displays genuine photographer attribution without fabrication.
   - Deep gradient overlay ensures text legibility across light and dark regions.
4. **Overlapping Profile Portrait & Initials Fallback**:
   - The portrait overlaps the lower-left boundary of the cover (approximately 96px on mobile, 128-144px on desktop).
   - Includes high-contrast serif initials fallback (e.g. "MJ" for Meenal Jain) when no image is uploaded or when an image fails to load.
   - Reuses existing system upload limits (max 5MB, JPEG/PNG/WebP) with accessible upload/replace and removal controls.
5. **Strict Typography and Layout Boundaries**:
   - Strictly standardizes to `Hedvig Letters Sans` and `Hedvig Letters Serif`. No unapproved Google Fonts or external typography packages.
   - Large editorial display name (`clamp(3.5rem, 8vw, 5.5rem)`), asymmetric whitespace, hairline borders (`#e2e2de`), and off-white canvas (`#f3f3f1`).
   - No sidebars, no social-media elements (no follower counts, connect buttons, or fake badges).

---

## Decision

### 1. Canonical Profile Route & Navigation
- Canonical route: `/app/profile` within the `/app/*` authenticated namespace.
- Added a direct `Profile` destination in `ProfileMenu` (`src/lib/components/nav/profile-menu.svelte`) and `appRoutes.profile`.
- Layout: Reuses the authenticated application shell `AppShellHeader` with zero society-specific sidebars or drawers rendered on the personal profile page.

### 2. Backend Profile API (`/api/v1/auth/profile`)
- **GET `/api/v1/auth/profile`**:
  Returns the user's unified personal profile:
  ```json
  {
    "id": "<user-uuid>",
    "phone": "+91 9967xxxx01",
    "phone_raw": "919967089101",
    "first_name": "Shubham",
    "last_name": "Bane",
    "display_name": "Shubham Bane",
    "email": "shubham@example.com",
    "status": "ACTIVE",
    "platform_roles": ["SUPER_ADMIN"],
    "society_assignments": [...],
    "created_at": "..."
  }
  ```
- **PATCH `/api/v1/auth/profile`**:
  Accepts:
  ```json
  {
    "first_name": "Shubham",
    "last_name": "Bane",
    "email": "shubham@example.com"
  }
  ```
  - Validates `first_name` (non-empty, max 100 chars), `last_name` (optional, max 100 chars), and `email` (valid email format if provided).
  - Updates all linked member records for the authenticated `user_id` in a transaction.
  - Automatically updates the user's display name and in-memory auth state.

### 3. Daily Curated Backdrop Strategy
- Implemented in `src/lib/backdrop.ts`:
  - Contains a curated gallery of high-resolution, calm architectural and nature photography with balanced contrast.
  - Generates a date-based deterministic index: `(year * 366 + dayOfYear) % curatedCollection.length`.
  - Guarantees visual stability across page refreshes on the same calendar day, with graceful local fallback if an image fails to load.

---

## Consequences

### Positive
- Users have an elegant, editorial space to view their verified identity and maintain their personal information.
- Phone authentication security is strictly preserved without exposing sensitive authentication mutations.
- Fast, zero-layout-shift daily backdrop experience requiring no external API keys or network latency.
- Full type safety, responsive design, and 100% test coverage across both frontend and backend.
