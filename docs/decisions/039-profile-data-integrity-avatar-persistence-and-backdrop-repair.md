# ADR 039: Profile Data Integrity, Canonical Identity Propagation, and Backdrop Repair

## Status
Accepted

## Context
Following the implementation of canonical user profiles ([[037-canonical-user-profiles-and-identity-resolution|ADR 037]]) and navigation rails ([[038-navigation-rail-and-modular-society-routes|ADR 038]]), several critical defects and identity inconsistencies were observed on `/app/profile`:
1. **Contaminated Profile Heading and Initials ("Hacker" / "HA")**:
   - The profile heading displayed `"Hacker"` and avatar initials displayed `"HA"` instead of the user's persisted name (`Shuubham Bane`).
   - Root cause investigation revealed that the backend test suite (`profile_persistence_test.go` Test F) executed against the live/shared Supabase database using the primary seed user (`919967089101`) and injected `"first_name": "Hacker"` to test cross-user isolation. Because the test mutated the authenticated seed user without restoring it in cleanup, the production database record was left polluted with `first_name = "Hacker"`.
2. **Missing Cover Photograph & Misleading Fallback**:
   - The daily backdrop photo returned HTTP 404 from the external image provider for the current calendar date index.
   - When the image failed to load, the UI rendered a camera icon inside a generic gray box, giving the misleading impression of an interactive cover photo upload button.
   - Furthermore, the photographer attribution (`"Photo by Mads Eneqvist"`) and location (`"Oslo, Norway"`) remained rendered on top of the gray fallback despite the photograph having failed to load.
3. **Disjoint Header & Sidebar Identity**:
   - `GET /api/v1/auth/me` and `POST /api/v1/auth/otp/verify` returned only account authentication fields (`id`, `phone`, `status`, `platform_roles`, `society_assignments`), omitting canonical profile attributes (`first_name`, `last_name`, `display_name`, `email`, `profile_photo_url`).
   - Consequently, `auth.user` in the frontend store lacked profile identity, forcing `NavigationRail` and `ProfileMenu` to fall back to the phone number (`+91 99670 89101`) rather than displaying the user's canonical display name.
4. **Avatar Persistence Gap**:
   - While `user_profiles` schema defined `profile_photo_url TEXT`, `UpdateProfileParams` and `UpdateProfileHandler` did not accept or persist `profile_photo_url`.
   - The frontend stored avatar previews strictly in `localStorage`, failing to persist uploaded portraits canonically across devices and sessions.

---

## Decision

### 1. Unified Canonical Identity Propagation
- Enriched `GET /api/v1/auth/me` and `POST /api/v1/auth/otp/verify` to resolve and return canonical profile fields alongside account authentication:
  - `first_name`
  - `last_name`
  - `display_name`
  - `email`
  - `profile_photo_url`
- Updated the reactive `auth` store (`src/lib/auth.svelte.ts`) and `UserProfile` type to maintain canonical identity.
- Added an in-place `auth.updateProfile(...)` synchronization method so that saving profile edits on `/app/profile` instantly updates the sidebar navigation rail, top header, and profile menu without requiring a page reload.

### 2. Avatar Photo Canonical Persistence
- Added `profile_photo_url` to `UpdateProfileParams` in the Go backend (`internal/auth/profile.go`) and `UpdateProfileHandler` (`internal/server/auth_handlers.go`).
- Supports storing canonical avatar image references (HTTPS URLs or validated Data URLs within system limits).
- When a user uploads a portrait or removes it on `/app/profile`, the change is transactionally persisted to `user_profiles.profile_photo_url` via `PATCH /api/v1/auth/profile`.
- When no avatar is uploaded or an image fails to load, the avatar falls back cleanly to deterministic initials generated from the canonical display name.

### 3. Backdrop Repair and Defensive Fallback
- Updated the curated gallery in `src/lib/backdrop.ts` to replace the defunct 404 image reference with a verified, active architectural photograph of Oslo, Norway (by Mikita Yo, HTTP 200 confirmed).
- In `src/routes/app/profile/+page.svelte`:
  - Conditioned location, title, and photographer attribution display on `!backdropFailed`, ensuring no orphaned credits are shown when an image is unavailable.
  - Replaced the misleading `<Camera />` icon in the fallback state with a clean, subtle geometric backdrop pattern that clearly communicates a decorative backdrop rather than an upload control.

### 4. Test Isolation and Seed Protection
- Hardened `backend/internal/server/profile_persistence_test.go` with automated `t.Cleanup` hooks that unconditionally restore `superAdminID`'s canonical profile (`Shuubham`, `Bane`, `shubham@banesons.com`) after test execution.
- Isolated spoofing tests so that test payloads never leave dirty data on primary seed accounts.

---

## Consequences

### Positive
- `/app/profile` displays consistent, canonical identity across the heading, avatar, form, top header, and navigation rail.
- Profile changes persist across browser refreshes, sessions, and devices.
- Cover backdrop loads reliably, with clean, honest fallback states when network issues occur.
- Test suites run cleanly without contaminating the development or demonstration environment.
