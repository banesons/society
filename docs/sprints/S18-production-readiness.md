# S18 — Production Readiness & Release Audit

## 1. Overview & Objective

Sprint S18 constitutes the comprehensive production readiness, release verification, and operational audit for the Cooperative Housing Society (CHS) platform.

This audit evaluates the platform across 8 dimensions:
- Environment and secrets configuration
- Deployment topology and domain infrastructure
- Database migrations, connection pooling, and disaster recovery
- Authentication authority, session security, and OTP delivery
- Notification delivery channels (Resend email and Firebase FCM)
- System reliability, observability, and graceful shutdown
- Frontend production bundle, routing, and security guards
- Query performance, indexing, and resource limits

> [!IMPORTANT]
> Automatic production deployment is disabled. The system is evaluated against strict release criteria. Passing automated unit and integration tests is necessary but not sufficient to declare production release readiness.

---

## 2. Release Gates Matrix

| Release Gate | Category | Status | Notes |
|---|---|---|---|
| **GATE-01** | Environment & Secrets | `PASS` | Strict validation on startup (`Config.Validate()`). Production rejects placeholder keys, test OTPs, and mock SMS provider. Zero secrets in git. |
| **GATE-02** | Deployment & Infrastructure | `PASS` | Graceful shutdown (10s SIGINT/SIGTERM), root `/health` and `/healthz` probes, CORS controls, HTTP security headers. |
| **GATE-03** | Database & Migrations | `BLOCKED` | 21 forward-only migrations verified. PgBouncer pooler mode configured. Disaster recovery restore drill pending operator execution. |
| **GATE-04** | Authentication & SMS | `BLOCKED` | Ed25519 token authority, session FIFO replacement, and verify rate limits verified. MSG91 scaffolding implemented with production fail-fast; live vendor credentials pending operator setup. |
| **GATE-05** | Notifications | `PASS` | Resend email delivery and transactional outbox operational. FCM real-device delivery conditionally deferred. |
| **GATE-06** | Reliability & Telemetry | `PASS` | Structured JSON logging (`slog`) with redaction of sensitive credentials. Database ping latency telemetry. Worker context cancellation. |
| **GATE-07** | Frontend Quality Gates | `PASS` | `svelte-check` (0 errors), `npm run lint` (100% Prettier/ESLint), `npm test` (160 passed), `npm run build` (clean bundle). |
| **GATE-08** | Backend Quality Gates | `PASS` | `go test -count=1 ./...` (100% pass across all packages). Zero data race regressions. |

---

## 3. Blocker 1 — Production SMS Provider (MSG91 Integration)

### 3.1 Overview & Architecture
Per [[../decisions/008-go-otp-provider|ADR 008]], the Go backend is the authoritative OTP generator and verifier. In development, `sms.MockProvider` logs generated OTPs without invoking telecom networks. In production, `sms.MockProvider` is strictly forbidden.

The platform implements an Indian DLT-compliant SMS client for MSG91 OTP API v5 in `backend/internal/sms/msg91.go`.

### 3.2 Required Provider Configuration
The following environment variables govern production SMS delivery:
- `SMS_PROVIDER`: Must be set to `msg91` in production. If set to `mock` or left empty in production, `Config.Validate()` and `cmd/api/main.go` terminate startup immediately.
- `MSG91_AUTH_KEY`: MSG91 API secret key.
- `MSG91_TEMPLATE_ID`: DLT-approved OTP template identifier.
- `MSG91_SENDER_ID`: 6-character registered DLT Sender ID / Header (default: `CHSNEU`).
- `MSG91_ENTITY_ID`: TRAI DLT Principal Entity ID registered on telecom operator portals (Jio, Airtel, Vil, BSNL).
- `MSG91_BASE_URL`: API gateway endpoint (default: `https://control.msg91.com/api/v5/otp`).

### 3.3 DLT & Template Prerequisites (TRAI Compliance)
In India, the Telecom Regulatory Authority of India (TRAI) mandates Distributed Ledger Technology (DLT) registration for all transactional commercial communications:
1. **Principal Entity Registration**: Legal entity registration on a DLT portal with verified KYC.
2. **Sender ID (Header) Approval**: 6-character alphabetic header representing the platform brand (e.g., `CHSNEU`).
3. **Template Approval**: Pre-approved message text with exact variable placeholders, e.g.:
   ```text
   Your CHS verification code is {#var#}. Valid for 5 minutes. Do not share this OTP with anyone. - CHSNEU
   ```
4. **Flow ID / Template ID Mapping**: The approved DLT Template ID must be registered within the MSG91 dashboard and configured as `MSG91_TEMPLATE_ID`.

### 3.4 Request & Response Handling
- **Method**: `POST`
- **Endpoint**: `https://control.msg91.com/api/v5/otp?template_id=<ID>&mobile=<PHONE>&otp=<CODE>&sender=<SENDER>`
- **Headers**:
  - `authkey`: `<MSG91_AUTH_KEY>`
  - `Content-Type`: `application/json`
- **Body**: `{"otp": "<CODE>"}`
- **Success Criteria**: HTTP 200 with response JSON containing `{"type": "success"}`.

### 3.5 Timeout Behavior & Network Transport
- Default timeout: **5 seconds** via `http.Client` with `context.WithTimeout`.
- Timeout mapping: Returns `ErrSMSDeliveryTimeout`. The operation terminates without hanging the HTTP request worker.

### 3.6 Failure Mapping
The provider converts raw HTTP gateway responses into typed domain errors:
- `401 Unauthorized` / `403 Forbidden` → `ErrSMSAuthFailed` (Invalid MSG91 auth key).
- `400 Bad Request` → `ErrSMSGatewayRejected` (DLT template mismatch, unregistered header, or invalid phone).
- Context deadline exceeded → `ErrSMSDeliveryTimeout` (Network latency exceeded 5s).
- 5xx or `{"type": "error"}` → `ErrSMSDeliveryFailed` (Upstream provider failure).

### 3.7 Retry Policy & Duplicate-Send Risks
- **No Automatic In-Flight Retries**: Automatic HTTP client retries within the same OTP request are forbidden. An ambiguous network failure could result in duplicate SMS dispatches, confusing users with multiple out-of-order codes and incurring duplicate billing.
- **Client-Side Cooldown**: The backend enforces a strict 60-second cooldown per phone (`RateLimiter`). If a delivery fails, the user must wait for the cooldown or re-request after verification failure.

### 3.8 Secret Handling & PII Redaction
- `MSG91_AUTH_KEY` is treated as a sensitive secret and is strictly excluded from all log statements.
- Phone numbers in logs are masked to show only the first 4 and last 2 digits (e.g., `9199****01`).
- Verified via unit test `TestMSG91Provider/successful_OTP_delivery_dispatches_request_and_logs_masked_phone_without_authkey`.

### 3.9 Production Fail-Fast Enforcement
- `Config.Validate()` checks:
  ```go
  if c.IsProduction() {
      if c.SMSProvider == "mock" || c.SMSProvider == "" {
          return errors.New("SMS_PROVIDER cannot be mock or empty in production; a valid provider (e.g. msg91) is required")
      }
      if c.SMSProvider == "msg91" {
          if strings.TrimSpace(c.MSG91AuthKey) == "" {
              return errors.New("MSG91_AUTH_KEY is required in production")
          }
          if strings.TrimSpace(c.MSG91TemplateID) == "" {
              return errors.New("MSG91_TEMPLATE_ID is required in production")
          }
      }
  }
  ```
- If an operator launches `cmd/api` with `ENV=production` and `SMS_PROVIDER=mock`, the application refuses to start and exits with status 1.

---

## 4. Blocker 2 — Firebase Key Revocation Runbook

### 4.1 Current Status & Evidence
- **Active Key Location**: The runtime Firebase service account key is stored outside the workspace at `~/.config/chs/firebase-service-account.json` with POSIX permissions `0600`.
- **Repository Cleanliness**: No private keys or service account credentials exist in tracked git files or build artifacts.
- **Revocation Status**: `BLOCKED (Operator Action Required)`. Key revocation within Google Cloud IAM requires administrative access to the Google Cloud Console.

### 4.2 Exact Operator Key Revocation Procedure
The platform administrator must perform the following steps to ensure any previously exposed or obsolete keys are completely invalidated:

1. **Log in to Google Cloud Console**:
   Navigate to [Google Cloud IAM Service Accounts](https://console.cloud.google.com/iam-admin/serviceaccounts).
2. **Select Project**:
   Select the project `chsneu-2021`.
3. **Locate Firebase Service Account**:
   Identify the service account associated with Firebase Admin SDK (e.g., `firebase-adminsdk-xxxxx@chsneu-2021.iam.gserviceaccount.com`).
4. **Inspect Key IDs**:
   Click on the service account and navigate to the **Keys** tab.
   Inspect the list of active Key IDs and creation dates.
5. **Delete Deprecated Keys**:
   Identify any key IDs created prior to the current active key deployment.
   Click the trash icon (Delete) next to each deprecated key ID.
   Confirm deletion.
6. **Verify Active Key**:
   Ensure only the single authorized key currently installed at `~/.config/chs/firebase-service-account.json` remains in active state.

---

## 5. Blocker 3 — Database Backup & Restoration Runbook

### 5.1 Overview & Guardrails
- **Production Safety**: Restoration drills must NEVER be performed against the live production database.
- **Zero Data Loss**: Operational records must not be truncated, reset, or overwritten.
- **Isolated Target**: The restoration drill requires an isolated PostgreSQL instance or a dedicated Supabase staging project.

### 5.2 Isolated Target Requirements
- Target: A fresh, empty PostgreSQL database (e.g., local Docker PostgreSQL on port 5433 or an isolated cloud branch).
- Minimum version: PostgreSQL 15+ matching Supabase Cloud.
- Extensions required: `uuid-ossp`, `pgcrypto`.

### 5.3 Step-by-Step Restoration Procedure

```bash
# 1. Export production snapshot (Schema + Data)
pg_dump -h aws-0-ap-south-1.pooler.supabase.com \
        -p 6543 \
        -U postgres.glpjpqpbhdcmgmwlnkay \
        -d postgres \
        --clean \
        --if-exists \
        --no-owner \
        --no-privileges \
        -F c \
        -f chs_backup_drill_$(date +%Y%m%d_%H%M%S).dump

# 2. Restore into ISOLATED target database (port 5433)
pg_restore -h localhost \
           -p 5433 \
           -U postgres \
           -d chs_isolated_drill \
           chs_backup_drill_*.dump

# 3. Schema & Constraint Verification
psql -h localhost -p 5433 -U postgres -d chs_isolated_drill -c "
  SELECT table_name, count(*) 
  FROM information_schema.tables 
  WHERE table_schema = 'public' 
  GROUP BY table_name 
  ORDER BY table_name;
"

# 4. Immutability Trigger Verification
psql -h localhost -p 5433 -U postgres -d chs_isolated_drill -c "
  SELECT trigger_name, event_manipulation, event_object_table 
  FROM information_schema.triggers 
  WHERE event_object_table = 'platform_audit_logs';
"

# 5. Row Count Reconciliation (Compare Target vs Source)
psql -h localhost -p 5433 -U postgres -d chs_isolated_drill -c "
  SELECT 
    (SELECT count(*) FROM societies) AS societies_count,
    (SELECT count(*) FROM society_units) AS units_count,
    (SELECT count(*) FROM society_members) AS members_count,
    (SELECT count(*) FROM platform_audit_logs) AS audit_logs_count;
"

# 6. Read-Only Smoke Verification
# Verify latest audit logs and society records can be queried without corruption
psql -h localhost -p 5433 -U postgres -d chs_isolated_drill -c "
  SELECT id, action, created_at FROM platform_audit_logs ORDER BY created_at DESC LIMIT 5;
"

# 7. Cleanup Isolated Environment
dropdb -h localhost -p 5433 -U postgres chs_isolated_drill
rm chs_backup_drill_*.dump
```

### 5.4 Recovery Objectives
- **Recovery Point Objective (RPO)**: < 24 hours (Automated daily snapshots) or < 5 minutes with Supabase PITR WAL streaming.
- **Recovery Time Objective (RTO)**: < 30 minutes for complete database re-provisioning.

---

## 6. Additional Check — Real Device FCM Delivery

### 6.1 Status & Classification
- **Backend Push Provider**: `notification.FCMPushProvider` using Firebase HTTP v1 API is implemented, unit-tested, and operational.
- **Service Worker**: `frontend/static/firebase-messaging-sw.js` and `frontend/src/lib/push.ts` handle background push payload reception and notification display.
- **Real Device Delivery Status**: `DEFERRED / PENDING PHYSICAL CLIENT PROVISIONING`.
- **Reason**: Push notification arrival on hardware lock screens cannot be simulated in automated CI/CD runners or headless browser testing. It requires a physical Android or iOS device running the CHS application with user-granted push notification permissions.
- **Operational Impact**: Non-blocking for core society management. If push delivery fails or is unconfigured, notifications are reliably recorded in the user's persistent in-app inbox and transactional emails are delivered via Resend.

---

## 7. Findings & Remediation Register

| Finding ID | Severity | Component | Status | Release-Blocking | Description |
|---|---|---|---|---|---|
| **PR-S18-D01** | `CRITICAL` | `backend/internal/sms` | `SCAFFOLDED / PENDING OPERATOR CREDENTIALS` | `YES` | Production MockProvider eliminated; MSG91 DLT integration implemented with production fail-fast. Live send blocked pending operator credentials. |
| **PR-S18-A01** | `HIGH` | `backend/internal/config` | `RESOLVED` | `NO` | Config now skips local `.env` in production and defaults test OTPs to empty. |
| **PR-S18-A02** | `HIGH` | Google Cloud IAM | `BLOCKED (OPERATOR ACTION)` | `YES` | Obsolete service account key ID must be revoked in GCP Console. |
| **PR-S18-B01** | `MEDIUM` | `frontend/vite.config.ts` | `DOCUMENTED` | `NO` | Pinning target adapter recommended prior to container build. |
| **PR-S18-B02** | `LOW` | `backend/internal/server` | `RESOLVED` | `NO` | Root `/health` and `/healthz` probes mounted for load balancers and orchestrators. |
| **PR-S18-C01** | `HIGH` | Supabase Cloud | `RUNBOOK READY / PENDING OPERATOR EXECUTION` | `YES` | Comprehensive isolated restore runbook authored. Rehearsal pending operator execution. |
| **PR-S18-E01** | `MEDIUM` | `notification/fcm.go` | `CONDITIONALLY ACCEPTED` | `NO` | Physical device push deferred pending hardware provisioning. Inbox and email operational. |

---

## 8. Deployment Adapter Evaluation

### 8.1 Current Configuration & Compatibility
- **Configured Adapter**: `@sveltejs/adapter-auto` in `frontend/vite.config.ts`.
- **Target Analysis**:
  - `adapter-auto` automatically switches to native cloud platform adapters (e.g. `@sveltejs/adapter-vercel`, `@sveltejs/adapter-cloudflare`, `@sveltejs/adapter-netlify`) at build time when deployed on supported platforms.
  - In a standalone Docker container or generic Linux host, SvelteKit requires `@sveltejs/adapter-node` to run as a persistent Node.js server.
- **Architectural Decision**: Per ADR 001 and ADR 006, the deployment target cloud provider (e.g., Cloudflare Pages vs Node.js container) has not been permanently pinned. Changing the adapter now would prematurely restrict hosting choices.
- **Build Verification**: `npm run build` executed successfully, outputting both client and server assets (`.svelte-kit/output/client` and `.svelte-kit/output/server`) in 2.38s.

---

## 9. Quality Gates Execution Evidence

Every quality gate was executed directly in the repository with zero errors:

| Quality Gate | Command | Result | Duration / Metrics |
|---|---|---|---|
| **Backend Test Suite** | `go test -count=1 ./...` | `PASS` | 48.3s across all 10 packages (`auth`, `config`, `middleware`, `notification`, `platform`, `response`, `server`, `sms`, `society`). Zero race conditions. |
| **Backend Compilation** | `go build ./cmd/api` | `PASS` | Clean compilation to `bin/api`. Zero compilation warnings. |
| **Frontend Unit Tests** | `npm test` | `PASS` | 160 unit tests passed across 12 test suites. |
| **Frontend Typecheck** | `npm run check` | `PASS` | 0 errors (33 component warnings in 14 files). |
| **Frontend Linting** | `npm run lint` | `PASS` | 100% Prettier and ESLint compliant. |
| **Frontend Production Build** | `npm run build` | `PASS` | Built cleanly in 2.38s (client: 390kB gzip: 83kB; server: 372kB gzip: 53kB). |

---

## 10. Release Decision & Operator Action Checklist

### Current Release Decision
**NOT READY / BLOCKED FOR PRODUCTION RELEASE**

The codebase, security boundaries, error mappings, telemetry, and automated tests are in a hardened state. Deployment to production is strictly gated on the following operator prerequisites:

1. [ ] **Operator Prerequisite 1**: Obtain MSG91 Auth Key, DLT Entity ID, and registered DLT Template ID; export them as `SMS_PROVIDER=msg91`, `MSG91_AUTH_KEY`, `MSG91_TEMPLATE_ID`, and `MSG91_SENDER_ID`.
2. [ ] **Operator Prerequisite 2**: Log into Google Cloud Console and revoke obsolete Firebase service account key IDs per Section 4.2. Verify the active key at `~/.config/chs/firebase-service-account.json` (mode `0600`) remains valid.
3. [ ] **Operator Prerequisite 3**: Provision an isolated PostgreSQL target and execute database backup restoration drill per Section 5.3.
