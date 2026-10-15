# Architecture 20: Notification System (Firebase Cloud Messaging + Resend Email)

## 1. Overview
The S15 Notification System provides a high-reliability, asynchronous, multi-channel notification engine for CHS. It connects critical domain events across Visitor Operations (S09), Announcements (S10), Maintenance Billing (S11), and Payments (S12) to end users via:
1. **In-App Notifications**: Stored in PostgreSQL, exposed via authenticated REST endpoints, and displayed with unread count badges in the SvelteKit application shell.
2. **Web Push Notifications**: Delivered via Firebase Cloud Messaging (FCM HTTP v1) to subscribed browsers and devices.
3. **Transactional Emails**: Delivered via Resend to user email addresses for formal records (bills, receipts, payment reversals).

---

## 2. Component Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Domain Mutations                         │
│  (S09 Visitors, S10 Announcements, S11 Bills, S12 Payments)     │
└────────────────────────────────┬────────────────────────────────┘
                                 │ Writes event within DB Tx
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│             PostgreSQL Table: notification_outbox              │
│       (id, event_type, idempotency_key, payload, status)        │
└────────────────────────────────┬────────────────────────────────┘
                                 │ Polled via FOR UPDATE SKIP LOCKED
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Go Delivery Worker Daemon                    │
│                                                                 │
│ 1. Claims pending outbox batch                                  │
│ 2. Resolves recipients & audience permissions                   │
│ 3. Creates `notifications` (inbox) & `notification_deliveries`  │
│ 4. Invokes external providers (outside DB transactions)         │
│ 5. Updates delivery status & handles backoff/jitter             │
└───────────────┬───────────────────────────────┬─────────────────┘
                │                               │
                ▼                               ▼
┌──────────────────────────────┐ ┌────────────────────────────────┐
│   Firebase Cloud Messaging   │ │         Resend Email           │
│     (FCM HTTP v1 Push)       │ │     (Transactional Email)      │
│  - Browser Web Push          │ │  - Maintenance Bills           │
│  - Device Token Feedback     │ │  - Payment Receipts            │
│  - Token Invalidation        │ │  - Payment Reversals           │
└──────────────────────────────┘ └────────────────────────────────┘
```

---

## 3. Provider Integrations

### 3.1 Push: Firebase Cloud Messaging (FCM)
- Protocol: Modern FCM HTTP v1 API.
- SDK: Official `firebase.google.com/go/v4/messaging`.
- Credential Strategy: Loaded via `FIREBASE_CREDENTIALS_FILE` or Application Default Credentials.
- Project ID: `chsneu-2021`.
- Token Feedback: When FCM responds with code `UNREGISTERED` or `INVALID_ARGUMENT`, the specific device token is marked `INACTIVE` in `user_devices` to prevent wasting resources on dead tokens.

### 3.2 Transactional Email: Resend
- SDK: Official `github.com/resend/resend-go/v4`.
- Authentication: `RESEND_API_KEY`.
- Sender: `RESEND_FROM_EMAIL` (default: `onboarding@resend.dev` in development).
- Recipient Resolution: Server-side only from authenticated user account data (`users.email` / `members.email`). Never accepts unverified client-supplied email addresses.
- HTML Escaping: All user-generated text (names, titles, amounts) is HTML-escaped to prevent email client HTML injection.

---

## 4. Recipient & Event Policy Matrix

| Domain Event | Idempotency Key Pattern | Channels | Recipient Resolution | Lock-Screen Push Summary |
| :--- | :--- | :--- | :--- | :--- |
| **VISITOR_APPROVAL_REQUESTED** | `VISIT_REQ:{visitID}` | In-App, FCM | Resident occupant(s) of destination unit | "Visitor Waiting: Approval requested at {gate_name}" |
| **VISIT_APPROVED** | `VISIT_APP:{visitID}` | In-App, FCM | Visitor user or creating resident | "Visit Approved: Pass valid for {unit_number}" |
| **VISIT_REJECTED** | `VISIT_REJ:{visitID}` | In-App, FCM | Visitor user or creating resident | "Visit Request Declined by Resident" |
| **ANNOUNCEMENT_PUBLISHED** | `ANN_PUB:{announcementID}:{userID}` | In-App, FCM | Active members matching audience scope | "New Announcement: {title}" |
| **BILL_ISSUED** | `BILL_ISS:{billID}:{userID}` | In-App, FCM, Email | Unit primary and secondary owners | "New Maintenance Bill: Due {due_date}" |
| **PAYMENT_RECORDED** | `PAY_REC:{paymentID}:{userID}` | In-App, FCM, Email | Paying resident / unit owner | "Payment Received: Receipt #{receipt_no}" |
| **PAYMENT_REVERSED** | `PAY_REV:{paymentID}:{userID}` | In-App, FCM, Email | Paying resident / unit owner | "Payment Reversal: Receipt #{receipt_no} reversed" |

---

## 5. Worker Lifecycle & Fault Tolerance
1. **Concurrency Control**: Multiple worker instances can run safely across replicas using `FOR UPDATE SKIP LOCKED` on `notification_outbox`.
2. **Lease & Recovery**: In-flight jobs are stamped with `locked_by` and `locked_until = NOW() + INTERVAL '2 minutes'`. If a worker crashes, orphaned jobs become claimable upon lock expiration.
3. **Exponential Backoff**: Transient provider errors retry with exponential backoff: `base * 2^attempt + jitter` up to 3 attempts.
4. **Permanent Errors**: Invalid recipient email or unregistered device tokens are classified as permanent failures and marked `PERMANENTLY_FAILED` without unnecessary retries.

---

## 6. Related Documentation
- [[../decisions/029-notifications|ADR 029: Notification System Architecture]]
- [[../database/15-notifications-schema|Database Schema 15: Notifications Schema]]
- [[../api/16-notifications-api|API 16: Notifications API]]
- [[../security/15-notifications-security|Security 15: Notifications Security]]
- [[../sprints/S15-notifications|Sprint S15: Notifications]]
