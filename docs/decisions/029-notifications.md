# ADR 029: Notification System Architecture (Firebase Cloud Messaging + Resend Email)

## Status
Accepted

## Context
CHS requires an omnichannel notification system to inform residents, unit owners, committee members, and platform administrators about critical real-time lifecycle events:
- **S09 Visitor Operations**: Immediate visitor approval requests, visitor arrivals, gate passes, pass check-ins.
- **S10 Society Communication**: Urgent broadcast announcements published to eligible audience segments.
- **S11 Maintenance Foundation**: Maintenance bills issued to unit owners with payment deadlines.
- **S12 Maintenance Collections**: Payment receipts and payment reversal notices.

Prior to S15, CHS lacked an asynchronous notification pipeline and device token registry. The solution must integrate seamlessly into the existing Go/Chi backend and SvelteKit frontend without introducing heavy distributed broker infrastructure (e.g., Kafka, RabbitMQ) or competing databases (e.g., Firestore).

## Decision

1. **Dual Provider Architecture**:
   - **Push Notifications**: Firebase Cloud Messaging (FCM) using the modern HTTP v1 protocol via the official `firebase.google.com/go/v4` SDK. FCM handles web push notifications to modern desktop and mobile browsers, and provides a foundation for native mobile push.
   - **Transactional Email**: Resend using the official `github.com/resend/resend-go/v4` SDK. Resend delivers transactional emails for bills, receipts, payment reversals, and formal notices.

2. **Durable Transactional Outbox Pattern**:
   - Business mutations in S09–S12 write durable event records into PostgreSQL table `notification_outbox` *within the same transaction* as the primary business record.
   - External network calls to FCM or Resend are **never** executed inside database transactions and **never** hold database row locks.
   - A PostgreSQL-backed delivery worker in Go polls `notification_outbox` using `FOR UPDATE SKIP LOCKED` for concurrent multi-instance safety.

3. **Persistent In-App Notification Ledger**:
   - Every user-facing notification creates a row in `notifications` (in-app inbox) tied to `recipient_user_id` and optional `society_id`.
   - Delivery attempts per channel are tracked in `notification_deliveries` with explicit status transitions (`PENDING` -> `PROCESSING` -> `DELIVERED` / `FAILED` / `PERMANENTLY_FAILED`).
   - Read state is tracked in `notifications.read_at` with fast unread badge query indexes.

4. **Device Registry & Sensitive Token Handling**:
   - Browsers and client devices register their FCM registration tokens via `POST /api/v1/notifications/devices`.
   - Tokens are tied to authenticated user sessions server-side.
   - Multiple devices per user are supported.
   - Invalid/unregistered token responses from FCM (`UNREGISTERED`, `INVALID_ARGUMENT`) automatically deactivate the specific device record without affecting other user devices.

5. **Security & Data Minimization**:
   - Push notification lock-screen payloads contain minimal non-sensitive summaries and routing metadata (`action_url`, `reference_id`). Sensitive personal details, financial account numbers, OTPs, and private visitor details are omitted from lock-screen text.
   - Resend API keys and Firebase service account credentials remain strictly backend-only.
   - Service account keys must never be committed to source control or exposed in chat.

## Consequences

### Positive
- Reliable asynchronous delivery: temporary provider outages do not block business operations or user requests.
- Multi-replica safety without external message brokers using PostgreSQL `SKIP LOCKED`.
- Resilient retry policy with bounded exponential backoff and jitter.
- In-app notification center provides continuous auditability and user visibility even if push or email delivery fails or is blocked.
- Clean separation of provider interfaces allowing easy unit testing with deterministic mocks.

### Negative
- Asynchronous processing introduces slight delivery latency (typically 1–3 seconds).
- Potential duplicate delivery if provider response is lost over the network during an in-flight send (documented at-least-once provider delivery semantics).
