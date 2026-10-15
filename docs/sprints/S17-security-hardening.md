# S17 — Security & Hardening Specification

## 1. Executive Summary & Objective
S17 conducts a thorough, evidence-based security audit and hardening sprint across the CHS platform. The goal is to verify existing defenses, identify security regressions or hardening gaps across all 9 architectural surfaces, implement safe scoped remediations, and prove security compliance through automated regression tests and real browser evidence.

S16 Dashboard & Reporting is closed. S15 remains conditionally complete: actual FCM physical device delivery and Firebase service-account key rotation are tracked as separate operational prerequisites.

---

## 2. Security Audit Scope & Findings

### 2.1 Audit Surface Analysis (9 Categories)

| Category | Evaluated Controls | Status |
| :--- | :--- | :--- |
| **1. Authentication** | OTP generation, 5m expiry, 3-attempt lock, 60s cooldown, phone/IP rate limits, constant-time compare, Ed25519 JWT validation, refresh token rotation, 2-device FIFO session replacement. | **HARDENED** (Added IP rate limiting on verify). |
| **2. Authorization & Tenancy**| Platform role separation (`SUPER_ADMIN`, `ADMIN`), society admin / gatekeeper / resident scoping, object-level checks, cross-society rejection (`403 FORBIDDEN`), client header validation. | **VERIFIED SECURE** (Header declared in CORS). |
| **3. Secrets & Configuration**| Zero secrets in git, `.env` gitignored, Firebase credentials stored outside repository with `0600` permissions, production configuration validation on startup. | **HARDENED** (Added strict production validation). |
| **4. Documents & Storage**| Classification scoping (`SOC_ADMIN_ONLY`, `COMMITTEE`, `SOCIETY_RESIDENTS`), magic-byte inspection, executable/HTML blocking, 10MB limits, proxy streaming with `nosniff`. | **VERIFIED SECURE**. |
| **5. API Security**| Global request size capping, HTTP security headers (`nosniff`, `DENY`, `strict-origin`), CORS allowed headers, rate limiting. | **HARDENED** (Added security headers & body cap). |
| **6. Audit Integrity**| Database trigger preventing `UPDATE` and `DELETE` on `platform_audit_logs`, recursive sensitive field redaction (passwords, tokens, OTPs, keys), phone masking. | **VERIFIED SECURE**. |
| **7. Notifications**| Authenticated device registration/deactivation, outbox concurrency safety (`SKIP LOCKED`), data minimization in push lock-screen payloads. | **VERIFIED SECURE**. |
| **8. Database & Dependencies**| 100% SQL query parameterization, transaction isolation, dependency audit (`npm audit fix`). | **HARDENED** (Fixed `source-map-js`). |
| **9. Production Readiness**| Health endpoint data minimization, `/developer` route restricted in production, secure startup defaults, fail-fast on database failure. | **HARDENED** (Protected `/developer` route). |

---

### 2.2 Security Findings Matrix

| Finding ID | Severity | Category | Component | Description & Evidence | Remediation | Migration? |
| :--- | :---: | :---: | :--- | :--- | :--- | :---: |
| **SEC-S17-01** | **MEDIUM** | Auth | `internal/auth/limiter.go`, `server/auth_handlers.go` | IP-level rate limiting was enforced on `SendOTP` but absent on `VerifyOTP`. An attacker could attempt brute-force guesses across multiple phone numbers from a single IP. | Add `CheckAndRecordVerifyFailureIP` (max 20 failed verification attempts per IP per 10m). | None |
| **SEC-S17-02** | **INFO** | Auth | `internal/auth/service.go` | Account enumeration is prevented by design via Just-In-Time (JIT) provisioning. `SendOTP` does not query users table; `VerifyOTP` auto-provisions non-existent users. | Confirmed secure; no change required. | None |
| **SEC-S17-03** | **INFO** | Auth | `internal/auth/otp.go` | Constant-time comparison (`subtle.ConstantTimeCompare`) prevents timing side-channel attacks on OTP hash checks. | Confirmed secure; no change required. | None |
| **SEC-S17-04** | **INFO** | Auth | `internal/auth/service.go` | Cryptographic refresh token rotation is strictly enforced. Prior tokens are invalidated immediately; revoked sessions reject refresh. | Confirmed secure; no change required. | None |
| **SEC-S17-05** | **INFO** | Auth | `internal/auth/service.go` | Automatic FIFO session eviction enforces the 2-device concurrency limit inside serializing PostgreSQL transactions (`FOR UPDATE`). | Confirmed secure; no change required. | None |
| **SEC-S17-06** | **INFO** | Authz | `internal/middleware/society.go` | Comprehensive multi-tenant boundary checks verify active society status, active user role assignment, and header consistency across all society endpoints. | Confirmed secure via `TestCrossSocietyAuthorization`. | None |
| **SEC-S17-07** | **LOW** | Authz | `internal/middleware/cors.go` | Header `X-Society-ID` was inspected by society middleware but omitted from CORS `AllowedHeaders`, causing browser preflight rejections. | Add `"X-Society-ID"` to CORS `AllowedHeaders`. | None |
| **SEC-S17-08** | **INFO** | Secrets | `backend/.env` | Firebase service account credentials file is confirmed outside repository with `0600` permissions. No secrets tracked in git. | Confirmed secure. Key rotation remains operator task. | None |
| **SEC-S17-09** | **HIGH** | Config | `internal/config/config.go`, `cmd/api/main.go` | Server silently fell back to ephemeral Ed25519 keys and continued without a database in production mode if misconfigured, risking session invalidation across restarts. | Implement `config.Validate()` enforcing `DatabaseURL`, valid `JWTPrivateKey`, and empty `DevTestOTP` in production. Fail fast on startup. | None |
| **SEC-S17-10** | **INFO** | Storage | `internal/society/documents.go` | Document access strictly enforces classification scopes (`SOC_ADMIN_ONLY`, `COMMITTEE`, `SOCIETY_RESIDENTS`). Files are validated by magic bytes and SHA-256 hashed. | Confirmed secure; no change required. | None |
| **SEC-S17-11** | **MEDIUM** | API | `internal/server/routes.go` | HTTP security response headers (`X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy`, `Permissions-Policy`) were not globally set. | Add `middleware.SecurityHeaders` to base Chi router stack. | None |
| **SEC-S17-12** | **MEDIUM** | API | `internal/server/routes.go` | General JSON endpoints lacked a global request payload size limit, risking memory exhaustion from oversized POST payloads. | Add `chimiddleware.RequestSize(10 << 20)` (10MB) to base router stack. | None |
| **SEC-S17-13** | **INFO** | Audit | `internal/platform/audit.go`, migration `20261010000000` | Database trigger forbids `UPDATE`/`DELETE` on `platform_audit_logs`. `SanitizeAuditDetails` redacts sensitive fields and masks phone numbers. | Confirmed secure; no change required. | None |
| **SEC-S17-14** | **INFO** | Notif | `internal/server/notification_handlers.go` | Push lock-screen payloads omit financial and sensitive data. Device registration and deletion are bound strictly to caller's authenticated user ID. | Confirmed secure; no change required. | None |
| **SEC-S17-15** | **LOW** | Deps | `frontend/package.json` | `source-map-js <= 1.2.1` vulnerable to ReDoS (GHSA-68fv-2mgg-jv7q). | Resolved: upgraded to `source-map-js 1.2.2` via `npm audit fix`. | None |
| **SEC-S17-16** | **INFO** | Deps | `frontend/node_modules/@grpc/grpc-js` | Transitive `@grpc/grpc-js` alert inside `@firebase/firestore`. CHS does not use Firestore or gRPC. Zero client exposure. | Accepted informational risk; no action needed. | None |
| **SEC-S17-17** | **INFO** | Database | `internal/society/*`, `internal/platform/*` | 100% of SQL queries across all 17 repositories use parameterized placeholders (`$1, $2, ...`). Zero string concatenation. | Confirmed secure; no change required. | None |
| **SEC-S17-18** | **MEDIUM** | Readiness | `routes/developer/+page.svelte` | The `/developer` diagnostics page was accessible in production builds without an environment guard or administrative check. | Guard `/developer` route: render access denied unless `dev` mode or user is `SUPER_ADMIN`. | None |
| **SEC-S17-19** | **INFO** | Readiness | `server/health.go` | Health check returns minimal status (`healthy`, `operational`, `latency_ms`). It exposes no database topology, credentials, or internal IPs. | Confirmed secure; no change required. | None |

---

## 3. Acceptance Criteria Matrix

| ID | Criterion | Verification Target | Status |
| :--- | :--- | :--- | :---: |
| **AC-S17-01** | Production Config Validation | `config.Validate()` fails fast if `DatabaseURL` or `JWTPrivateKey` is missing or `DevTestOTP` is set in production. | **PASS** |
| **AC-S17-02** | Security Response Headers | Every API response includes `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, and `Referrer-Policy`. | **PASS** |
| **AC-S17-03** | Global Request Size Limit | POST requests exceeding 10MB on JSON endpoints are rejected with `413 Request Entity Too Large` / `400 Bad Request`. | **PASS** |
| **AC-S17-04** | CORS Allowed Headers | `X-Society-ID` is explicitly permitted in CORS configuration for browser clients. | **PASS** |
| **AC-S17-05** | Verify OTP IP Throttling | Repeated failed OTP verification attempts from the same IP are rate-limited (`429 Too Many Requests`). | **PASS** |
| **AC-S17-06** | Protected Developer Route | Navigating to `/developer` in non-development mode blocks access for unauthorized visitors. | **PASS** |
| **AC-S17-07** | Dependency Vulnerabilities | Resolved fixable npm vulnerabilities (`source-map-js`). | **PASS** |
| **AC-S17-08** | Database Query Security | Verified 100% parameterized SQL queries and immutable audit log trigger. | **PASS** |
| **AC-S17-09** | Quality Gates Passing | `go test -count=1 ./...`, `npm test`, `npm run check`, `npm run lint`, `npm run build` all pass clean. | **PASS** |
| **AC-S17-10** | Database Immutability | Zero migrations executed; zero records truncated or deleted. | **PASS** |

---

## 4. Rollback & Operational Considerations
- **No Database Migrations**: Zero schema changes; rollback is a clean git revert if needed.
- **Session Preservation**: All existing user sessions and JWT signing keys remain intact.
- **Outstanding Operator Actions**:
  1. FCM real device delivery testing requires a genuine device registration token (S15 prerequisite).
  2. Firebase service-account private key rotation in Firebase Console (remediation of previously exposed key).

---

## 5. Related Documentation
- [[../decisions/011-two-device-limit|ADR 011: Two-Device Limit with Automatic FIFO Replacement]]
- [[../decisions/028-audit-and-administration|ADR 028: Unified Audit and Administration]]
- [[../decisions/029-notifications|ADR 029: Notification System Architecture]]
- [[../decisions/030-client-performance-monitoring|ADR 030: Client-Side Performance Monitoring]]
- [[../security/01-authentication-security|Security 01: Authentication Security]]
- [[../security/14-audit-and-administration-security|Security 14: Audit Security]]
- [[../security/15-notifications-security|Security 15: Notifications Security]]

---

## 6. Status
COMPLETE (Audit completed, remediations implemented, verified with automated test suite and browser evidence).
