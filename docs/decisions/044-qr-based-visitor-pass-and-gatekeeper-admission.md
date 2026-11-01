# 044 — QR-Based Visitor Pass, Self-Registration, OTP Verification, and Gatekeeper Admission

## Context and Problem Statement

CHS Neu previously supported resident-expected visitors and walk-in visitor registration recorded manually by gatekeepers. However, high-density residential societies require self-registration at physical entrance gates to reduce gatekeeper workload and prevent manual data entry bottlenecks.

A production-ready visitor self-registration system must satisfy several non-negotiable operational, security, and privacy constraints:
1. Physical entrance QR codes must be opaque public registration tokens that do not leak internal society identifiers or personal data.
2. Visitor registration must establish mobile number possession via OTP verification without creating permanent accounts or granting resident/society membership privileges.
3. Visitor photograph capture must be supported on mobile devices for visual human verification at the gate, stored securely with tenant isolation.
4. Registration must NOT automatically admit the visitor; physical entry requires explicit, atomic gatekeeper decision (`ALLOW` or `REJECT`).
5. Gatekeepers must NEVER receive unmasked visitor phone numbers or extraneous user profile information.
6. Returning authenticated users must be able to view their cross-society visitor passes and check-in timeline matching their verified phone number.

---

## Decisions

### 1. Unified Canonical Domain Model (Zero Architecture Duplication)
- **Entities**:
  - `society_gate_qrs`: Represents administrator-configured public entrance registration points. Opaque, cryptographically random 256-bit hex tokens (`vqr_<hex>`). Associated with a society and optionally an entry gate. Can be rotated or revoked by `SOC_ADMIN` without invalidating existing visits.
  - `society_visits`: Reused as the single canonical visit record. Added nullable `qr_id`, `visitor_photo_url`, `coming_from`, and `pass_reference`. Nullable `created_by` accommodates anonymous public registrations.
  - Added new canonical lifecycle state: `REGISTERED_PENDING_GATE_VERIFICATION` and visit type `QR_SELF_REGISTER`.
  - Reused `society_visitor_identities` for tenant-scoped person tracking.
- **Safety**: Public registration never creates resident accounts, user records, or society memberships.

### 2. Opaque Token Resolution & Stateless OTP Verification
- **Public QR Resolution**: Anonymous `GET /public/visit-qr/{token}` resolves society name, code, and entry gate name. Inactive, revoked, or non-existent tokens return `404 Not Found`.
- **OTP Verification**: Reuses existing OTP generation and rate limiting. Upon successful OTP verification, the backend issues a signed, stateless HMAC-SHA256 verification token (`GeneratePhoneVerificationToken`) with a 15-minute expiry.
- **Payload Binding**: The self-registration handler verifies that the submitted mobile number matches the HMAC signature. If the user changes their phone number, verification is invalidated and must be repeated.

### 3. Idempotent Registration & Visitor Pass Issuance
- **Atomic Registration**: `RegisterVisitorByQR` checks for duplicate active registrations for the same visitor at the society within the last 15 minutes, preventing double-tap duplicate visits.
- **Pass Reference**: Assigns short, legible, uppercase pass references (`VP-XXXXXX`), displayed on the public live pass tracking page (`/visit/pass/{passReference}`).
- **Live Status Polling**: The visitor pass UI polls every 4-5 seconds; upon gatekeeper admission, the UI immediately flips to "Checked In" without page reload.

### 4. Gatekeeper Queue & Concurrency-Safe Admission
- **Queue Integration**: Added dedicated "QR Registrations" tab in Gate Console (`gate-console.svelte`) with live pending count badges.
- **Strict Data Minimization**: Backend DTO `PendingQRVisitDTO` serializes ONLY `visitor_phone_masked`. Raw phone numbers, user profile fields, and IP addresses are excluded at the Go struct layer.
- **Concurrency Safety**: Admission is executed via `AdmitQRVisitor` utilizing row-locking (`SELECT ... FOR UPDATE`):
  - On `APPROVE`: Transitions to `CHECKED_IN`, sets entry gate and actual check-in timestamp.
  - On `REJECT`: Transitions to `REJECTED`, records rejection reason.
  - Concurrent race conditions between two guards result in `HTTP 409 Conflict`.
- **Cross-Gate Checkout Preserved**: Visits admitted via QR can exit through any authorized gate in the society (`check_out_at` and `exit_gate_id` recorded independently).

### 5. Authenticated Visitor History & Prefill
- **User History**: Exposed `GET /visitors/my-history` authenticated endpoint. Resolves the caller's verified mobile number and lists all historical visits across all societies where `phone = user_phone`.
- **Profile Integration**: Created `MyVisitorHistory` component in `frontend/src/lib/components/profile/my-visitor-history.svelte`, embedded in `/app/profile`.
- **Registration Prefill**: When an authenticated user scans an entrance QR, the form automatically prefills their verified name and phone number.

---

## Consequences & Safety Guarantees

- **Production Migration Safety**: Migration `20261031000000_create_visitor_qr_and_self_registration.sql` applied cleanly without resetting, dropping, or mutating existing records.
- **Cross-Tenant Isolation**: Composite foreign keys and tenant-scoped queries prevent cross-society data leakage.
- **Auditing**: All QR generations, rotations, revocations, and gatekeeper admissions are recorded in `platform_audit_logs`.
- **Visual Design**: Follows Swiss editorial minimalism (#f3f3f1, white cards, #e2e2de borders, Hedvig Letters fonts).

### Security Audit Hardenings (Remediation & Staging Gate Verification)
Following a focused security audit and post-audit remediation pass, the following invariants were implemented, verified, and locked with regression tests:
1. **Replay & Cross-Context Prevention**: Phone verification HMAC tokens cryptographically bind the QR context and a single-use random nonce (`phone|qrHash|nonce|expiresAt|sig`). Tokens cannot be replayed for multiple registrations or reused across different QR entrance points.
2. **Cryptographic Key Derivation & Safe Failure**: Verification HMAC keys are derived using domain separation (`chs_neu:visitor_verification_hmac_v1`) from the Ed25519 signing seed when `VerificationSecret` is unset, with runtime ephemeral fallback in test environments. No static, hardcoded secrets exist in the codebase; uninitialized configs fail safely.
3. **Bounded In-Memory Replay Guard & Distributed Limitation**: The `sync.Map` replay guard opportunistically purges expired entries (`pruneExpiredNonces`) during token validation, ensuring bounded memory consumption. **Deployment Limitation**: Because replay tracking is currently in-memory, distributed deployments with multiple stateless replicas require sticky IP/session routing per gate or a future Redis/Postgres-backed durable nonce store to enforce distributed single-use guarantees across instances.
4. **128-Bit Pass Reference Entropy**: References function as bearer credentials for pass tracking and are generated with 128-bit cryptographic entropy (`VP-` followed by 32 hex chars, ~3.4 × 10³⁸ combinations) with collision retry loops, rendering enumeration attacks computationally impossible.
5. **Rate Limiting & Data Minimization**: Public pass lookups are throttled to 60 requests/minute per client IP (returning 429 Too Many Requests). Responses strictly mask phone numbers (`+91 98200 *****`) and exclude internal resident emails, flat IDs, or raw identity records.
6. **Visitor-History Identity Disambiguation**: `GET /api/v1/visitors/my-history` never relies on phone equality alone. Visits are only returned if ownership is verified (`v.created_by = user_id`) or if the caller has a matching canonical profile name with zero conflicting visitor names on that phone number. Ambiguous records (shared family phones, missing profile names, duplicate names) are excluded to prevent cross-person data exposure.
7. **Photo Privacy & SSRF Elimination**: Accepted visitor photos are stored directly as base64 Data URLs (or CDN URLs) without backend HTTP fetching, eliminating SSRF attack vectors. Remote private storage bucket policies remain classified as NOT VERIFIED since bucket definitions reside outside the repository.
8. **Duty Session Tracking**: Admissions record active gatekeeper duty sessions when present in `society_gate_events`.

## Related Documents
- [[../api/10-gate-visitor-operations-api]]
- [[../database/09-gate-visitor-operations-schema]]
- [[023-visitor-lifecycle-and-gate-operations]]
- [[041-gatekeeper-experience-visitor-identity-and-activity-timeline]]
- [[043-sidebar-collapsibility-and-gatekeeper-workspace-routing]]
