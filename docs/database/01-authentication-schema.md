# 01 — Authentication Database Schema

## Overview

The S01 authentication schema defines three tables in Supabase PostgreSQL:
1. `users`: Universal identity master record.
2. `auth_otps`: Transient verification tokens.
3. `user_sessions`: Active device sessions and refresh token hashes.

```text
       ┌───────────────────────┐
       │         users         │
       ├───────────────────────┤
       │ id (PK, UUID)         │
       │ phone (UNIQUE, TEXT)  │
       │ status (TEXT)         │
       │ created_at            │
       │ updated_at            │
       └───────────┬───────────┘
                   │ 1
                   │
                   │ N
       ┌───────────▼───────────┐         ┌───────────────────────┐
       │     user_sessions     │         │       auth_otps       │
       ├───────────────────────┤         ├───────────────────────┤
       │ id (PK, UUID)         │         │ id (PK, UUID)         │
       │ user_id (FK, UUID)    │         │ phone (TEXT)          │
       │ refresh_token_hash    │         │ otp_hash (TEXT)       │
       │ user_agent            │         │ attempts (INT)        │
       │ client_ip             │         │ expires_at            │
       │ expires_at            │         │ consumed_at           │
       │ revoked_at            │         │ created_at            │
       │ created_at            │         │ updated_at            │
       │ last_active_at        │         └───────────────────────┘
       └───────────────────────┘
```

---

## 1. `users` Table

Stores universal user identity.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Immutable internal identifier |
| `phone` | `TEXT` | `NOT NULL UNIQUE` | Canonical E.164 phone (`+91XXXXXXXXXX`) |
| `status` | `TEXT` | `NOT NULL DEFAULT 'ACTIVE'` | Account status (`ACTIVE`, `SUSPENDED`) |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | Record creation timestamp |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | Record update timestamp |

**Indexes**:
- `users_phone_idx` ON `users(phone)`

---

## 2. `auth_otps` Table

Stores transient hashed verification codes.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | OTP transaction ID |
| `phone` | `TEXT` | `NOT NULL` | Canonical E.164 target phone |
| `otp_hash` | `TEXT` | `NOT NULL` | Salted SHA-256 hash of the 6-digit OTP |
| `attempts` | `INT` | `NOT NULL DEFAULT 0` | Failed verification attempts (max 3) |
| `expires_at` | `TIMESTAMPTZ` | `NOT NULL` | Valid for 5 minutes from creation |
| `consumed_at`| `TIMESTAMPTZ` | `NULL` | Timestamp when successfully verified |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | Issuance timestamp |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | Last attempt timestamp |

**Indexes**:
- `auth_otps_phone_idx` ON `auth_otps(phone)`
- `auth_otps_expires_at_idx` ON `auth_otps(expires_at)`

---

## 3. `user_sessions` Table

Tracks active device sessions, refresh hashes, and device limits.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Session ID embedded in JWT |
| `user_id` | `UUID` | `NOT NULL REFERENCES users(id) ON DELETE CASCADE` | Associated user identity |
| `refresh_token_hash` | `TEXT` | `NOT NULL` | SHA-256 hash of active refresh token |
| `user_agent` | `TEXT` | `NULL` | Client browser / device string |
| `client_ip` | `TEXT` | `NULL` | Client network address |
| `expires_at` | `TIMESTAMPTZ` | `NOT NULL` | 30 days from creation |
| `revoked_at` | `TIMESTAMPTZ` | `NULL` | Set on logout or token reuse |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | Session establishment timestamp |
| `last_active_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | Last activity timestamp |

**Indexes**:
- `user_sessions_user_id_idx` ON `user_sessions(user_id)`
- `user_sessions_active_idx` ON `user_sessions(user_id, expires_at, revoked_at)`
- `user_sessions_refresh_hash_idx` ON `user_sessions(refresh_token_hash)`

---

## Maintenance & Cleanup

Expired/consumed OTPs are pruned periodically by a background vacuum query or cleanup hook:
```sql
DELETE FROM auth_otps WHERE expires_at < NOW() - INTERVAL '1 day';
```

## Related Documentation
- [[../architecture/05-authentication|Authentication Architecture]]
- [[../security/01-authentication-security|Authentication Security]]
