# Database Schema 15: Notifications Schema

## 1. Overview
The S15 schema introduces four forward-only tables to manage user devices, in-app notifications, delivery outcomes, and transactional outbox events.

Migration: `20261028000000_create_notifications_schema.sql`

---

## 2. Table Definitions

### 2.1 `user_devices`
Stores registered client push tokens for authenticated users across Web, Android, and iOS clients.

```sql
CREATE TABLE user_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id TEXT NOT NULL,
    fcm_token TEXT NOT NULL,
    platform TEXT NOT NULL CHECK (platform IN ('WEB', 'ANDROID', 'IOS')),
    user_agent TEXT,
    client_ip TEXT,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'REVOKED')),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_devices_user_device UNIQUE (user_id, device_id)
);

CREATE INDEX idx_user_devices_token ON user_devices (fcm_token);
CREATE INDEX idx_user_devices_user_active ON user_devices (user_id, status) WHERE status = 'ACTIVE';
```

### 2.2 `notifications`
Represents the user's canonical in-app notification ledger (inbox).

```sql
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    society_id UUID REFERENCES societies(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    action_url TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    reference_type TEXT,
    reference_id UUID,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_recipient_created ON notifications (recipient_user_id, created_at DESC);
CREATE INDEX idx_notifications_recipient_unread ON notifications (recipient_user_id, created_at DESC) WHERE read_at IS NULL;
CREATE INDEX idx_notifications_society ON notifications (society_id, created_at DESC) WHERE society_id IS NOT NULL;
```

### 2.3 `notification_deliveries`
Tracks individual outbound delivery attempts per channel (FCM or EMAIL).

```sql
CREATE TABLE notification_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    channel TEXT NOT NULL CHECK (channel IN ('FCM', 'EMAIL')),
    target TEXT NOT NULL,
    device_id UUID REFERENCES user_devices(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PROCESSING', 'DELIVERED', 'FAILED', 'PERMANENTLY_FAILED', 'SKIPPED')),
    attempt_count INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    provider_message_id TEXT,
    error_code TEXT,
    error_detail TEXT,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deliveries_pending ON notification_deliveries (status, next_retry_at) 
WHERE status IN ('PENDING', 'PROCESSING');
CREATE INDEX idx_deliveries_notification ON notification_deliveries (notification_id);
```

### 2.4 `notification_outbox`
Guarantees at-least-once durable event processing from domain mutations.

```sql
CREATE TABLE notification_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED')),
    attempt_count INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_until TIMESTAMPTZ,
    locked_by TEXT,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_pending ON notification_outbox (status, next_attempt_at)
WHERE status IN ('PENDING', 'PROCESSING');
```

---

## 3. Related Documentation
- [[../decisions/029-notifications|ADR 029: Notification System Architecture]]
- [[../architecture/20-notifications|Architecture 20: Notification System]]
- [[../api/16-notifications-api|API 16: Notifications API]]
- [[../security/15-notifications-security|Security 15: Notifications Security]]
