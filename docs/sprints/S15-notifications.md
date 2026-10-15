# S15 — Notifications (Firebase Cloud Messaging + Resend Email)

## Objective
Implement an omnichannel notification system across CHS using Firebase Cloud Messaging (FCM HTTP v1) for web/device push notifications, Resend for transactional email, and PostgreSQL for durable outbox processing and in-app notification inbox. Connect real-time domain events from S09 (Visitors), S10 (Announcements), S11 (Maintenance Bills), and S12 (Payments) with strict recipient resolution, tenant isolation, and sensitive data minimization.

---

## Acceptance Criteria Matrix

| ID | Criterion | Status | Evidence |
| :--- | :--- | :--- | :--- |
| **AC-S15-01** | Notification architecture and ADR documented | PASS | ADR 029, Architecture 20, Schema 15, API 16, Security 15 created, synchronized, and verified. |
| **AC-S15-02** | Forward-only database migration for notifications schema | PASS | `20261028000000_create_notifications_schema.sql` applied forward-only to Supabase Cloud; verified by `cmd/verify`. |
| **AC-S15-03** | Resend email provider implemented with official SDK | PASS | Integrated `github.com/resend/resend-go/v4` in `internal/notification/resend.go`. Sanitized errors, no exposed credentials. |
| **AC-S15-04** | Resend live transactional email provider acceptance | PASS | Live Resend API test executed to `delivered@resend.dev`. Provider accepted with message ID `01a13a2e-440d-7bed-83f5-7f32f3b05197`. |
| **AC-S15-05** | FCM push provider implemented with official Admin SDK | PASS | Integrated `firebase.google.com/go/v4/messaging` in `internal/notification/fcm.go` using HTTP v1 protocol. |
| **AC-S15-06** | Real FCM device push delivery acceptance | BLOCKED | Service-account credentials loaded and verified via `TestLiveFCM_CredentialsInitialization` (Google FCM API responded in 0.43s). However, no physical test device or real FCM browser token is currently registered in `user_devices` (only mock fallback token exists due to unset browser VAPID key). Real push receipt on an end-user device remains blocked pending real device registration. |
| **AC-S15-07** | Device registration API (`POST /api/v1/notifications/devices`) | PASS | Implemented in `server/notification_handlers.go`, tested in `server/notification_test.go`. Strict user ownership derivation. |
| **AC-S15-08** | Device deactivation API (`DELETE /api/v1/notifications/devices/{id}`) | PASS | Implemented in `server/notification_handlers.go`, tested in `server/notification_test.go`. Cross-user access prohibited. |
| **AC-S15-09** | List notifications API (`GET /api/v1/notifications`) | PASS | Deterministic pagination, user-scoping, unread filtering tested in `notification_test.go` and `server/notification_test.go`. |
| **AC-S15-10** | Unread notification count API (`GET /api/v1/notifications/unread-count`) | PASS | Tested in `server/notification_test.go` with strict tenant and recipient user isolation. |
| **AC-S15-11** | Mark notification read API (`PATCH /api/v1/notifications/{id}/read`) | PASS | Tested in `server/notification_test.go` with IDOR rejection assertions. |
| **AC-S15-12** | Mark all notifications read API (`POST /api/v1/notifications/mark-all-read`) | PASS | Tested in `server/notification_test.go` with bulk recipient user update. |
| **AC-S15-13** | Transactional outbox event creation on domain mutations | PASS | Enqueues wired in S09 visitors (`visitors.go`), S10 announcements (`announcements.go`), S11 bills and S12 payments (`maintenance.go`). |
| **AC-S15-14** | Durable delivery worker with exponential backoff and jitter | PASS | `internal/notification/worker.go` tested with concurrent claims (`FOR UPDATE SKIP LOCKED`) and backoff in `notification_test.go`. |
| **AC-S15-15** | Worker crash-recovery and inbox creation idempotency | PASS | Recovered abandoned leases (`locked_until`), checked existing notifications by `outbox_event_id` and reference; duplicate sends skipped. |
| **AC-S15-16** | FCM invalid-token feedback and auto-deactivation | PASS | Verified in `worker.go` and `notification_test.go` (`messaging.IsUnregistered`, `messaging.IsInvalidArgument` -> `DeactivateToken`). |
| **AC-S15-17** | Email template rendering and HTML escaping | PASS | Tested via `html.EscapeString` in `worker.go` and mock provider test. |
| **AC-S15-18** | Push payload data minimization (no sensitive PII on lock screen) | PASS | Verified minimal title/body with safe routing metadata in `worker.go`. |
| **AC-S15-19** | Frontend notification inbox and unread count badge in shell | PASS | `notification-center.svelte` mounted in `society-shell-header.svelte`. Tested in `frontend/src/lib/api/notifications.test.ts`. |
| **AC-S15-20** | Frontend web push registration with service worker | PASS | `firebase-messaging-sw.js` and `src/lib/push.ts` implemented and tested. |
| **AC-S15-21** | Live real browser acceptance (Chrome CDP) | PASS | Real browser session executed on `http://localhost:5173`. Authenticated with OTP, navigated to society console, toggled notification bell, verified Notification Center panel, Enable Desktop Push banner, and inbox state. Recorded in `s15_browser_acceptance_1791979528263.webp` with screenshot `notification_center_open_1791979778001.png`. |
| **AC-S15-22** | Backend quality gates pass | PASS | `gofmt`, `go vet ./...`, `go build ./...`, and `go test -count=1 ./...` all passing. |
| **AC-S15-23** | Frontend quality gates pass | PASS | `npm run check` (0 errors), `npm run lint` (0 errors), `npm test` (150 tests pass), `npm run build` (passing). |
| **AC-S15-24** | Database invariant verification passes with data preserved | PASS | `go run cmd/verify/main.go` verified all 14 schema sections live with 0 rows deleted (5,212 audit records preserved). |

---

## Related Documentation
- [[../decisions/029-notifications|ADR 029: Notification System Architecture]]
- [[../architecture/20-notifications|Architecture 20: Notification System]]
- [[../database/15-notifications-schema|Database Schema 15: Notifications Schema]]
- [[../api/16-notifications-api|API 16: Notifications API]]
- [[../security/15-notifications-security|Security 15: Notifications Security]]

---

## Status
CONDITIONALLY COMPLETE (All core implementation, PostgreSQL schema, outbox worker, Resend live email provider acceptance, frontend notification UI, automated quality gates, and live real-browser acceptance PASS. Real FCM physical/browser device delivery remains BLOCKED pending registration of a genuine FCM token. Resend provider acceptance is explicitly recorded as provider-level queue acceptance, distinct from confirmed mailbox receipt. Firebase credential storage remediation and physical device delivery are tracked as operational prerequisites before full closure).
