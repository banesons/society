# ADR 037: Canonical User Profiles and Global Identity Resolution

## Status
Accepted

## Context
In previous iterations of the personal profile feature ([[035-personal-profile-experience-and-identity|ADR 035]]), personal profile fields (`first_name`, `last_name`, `email`) were queried and updated against the `society_members` table under the assumption that users would have active society assignments.

This produced a critical defect:
1. When a platform `SUPER_ADMIN` or any user without an active society membership updated their profile at `/app/profile`, the update matched zero rows in `society_members`.
2. The endpoint falsely reported `success: true` while persisting nothing to the database.
3. Upon subsequent retrieval or browser refresh, `first_name`, `last_name`, and `email` returned empty, and the display name fell back to a hardcoded placeholder `"Verified Resident"`, misidentifying platform administrators and unassigned accounts.
4. If a user had multiple society memberships, updating the profile globally mutated tenant-scoped member records, violating tenant boundaries.

A proper canonical global profile was needed that decouples user personal identity from society tenancy and platform role assignments.

---

## Decision

### 1. Canonical Storage Model (`user_profiles` Table)
A dedicated `user_profiles` table was introduced (`supabase/migrations/20261029000000_create_user_profiles_schema.sql`):
- **Foreign Key**: `user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE`.
- **Columns**: `first_name`, `last_name`, `email`, `profile_photo_url`, `created_at`, `updated_at`.
- **Decoupling**: `users` remains the pure authentication identity (`phone`, `status`, `sessions`), while `user_profiles` holds personal profile attributes.
- **Migration & Backfill**: The migration backfills `user_profiles` for all existing users in `users`, populating existing names from active `society_members` where present.

### 2. Tenant & Society Isolation
- Editing `/api/v1/auth/profile` updates **only** the authenticated user's canonical `user_profiles` record.
- Tenant-scoped `society_members` records are **not** overwritten. Society-specific member data (e.g. designation, unit relationship records) maintains strict tenant isolation.
- No society membership records are created as a side effect for platform administrators.

### 3. Server-Side Trusted Context & Input Validation
- The user identifier is extracted strictly from `middleware.GetUserID(r.Context())`. Client-supplied user IDs in request bodies are ignored.
- **Validation**:
  - `first_name`: Required, non-empty after trimming, max 100 characters.
  - `last_name`: Optional, max 100 characters.
  - `email`: Optional, normalized to lowercase, validated email format, max 255 characters.
- **Phone Immutability**: Verified phone number is strictly read-only and cannot be changed via profile endpoints.
- Transactional updates guarantee that only verified, persisted values are returned in responses.

### 4. Deterministic Display-Name Hierarchy
Replaced hardcoded `"Verified Resident"` with a deterministic resolution policy:
1. `first_name + " " + last_name` (if both non-empty).
2. `first_name` (if `last_name` is empty).
3. Fallback derived from verified account identity: `FormatDisplayPhone(phone)` (e.g. `+91 99670 89101`).

Generic role titles (such as "Verified Resident") are never used as personal names for users.

---

## Consequences

### Positive
- Platform administrators, society administrators, gatekeepers, and residents can maintain their personal profiles uniformly.
- Profile changes persist across browser refreshes and subsequent sessions.
- Tenant isolation is strictly preserved: global profiles do not mutate unrelated society records.
- Verified phone numbers remain immutable authentication anchors.
