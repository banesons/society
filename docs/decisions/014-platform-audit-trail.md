# ADR 014: Append-Only Platform Audit Trail Architecture

## Status
Proposed (Sprint S02)

## Context
Administrative actions—such as creating an administrator, disabling accounts, or altering security settings—carry significant operational and security implications. For accountability, forensic auditing, and compliance, all administrative events must be captured immutably without risk of tampering.

## Decision
1. **Dedicated Audit Table**:
   - Create `platform_audit_logs` storing administrative actions.
   - Schema captures:
     - `id`: UUID primary key
     - `actor_id`: UUID of the authenticated administrator performing the action
     - `action`: Canonical action string (e.g., `ADMIN_CREATED`, `ADMIN_DISABLED`, `ADMIN_REACTIVATED`, `SUPER_ADMIN_BOOTSTRAPPED`)
     - `target_user_id`: UUID of the affected user (nullable for platform-wide actions)
     - `details`: `JSONB` document holding structured context (e.g., previous status, new status, reasons)
     - `client_ip`: Source IP derived from trusted server context
     - `user_agent`: Client User-Agent string
     - `created_at`: Immutable timestamp (`DEFAULT NOW()`)
2. **Transactional Integrity**:
   - Every administrative change (creation, status update, reactivation) must write its corresponding audit record **in the same database transaction (`pgx.Tx`)**.
   - If writing the audit record fails for any reason, the **entire administrative transaction is rolled back**.
   - Administrative mutations cannot succeed without a persistent audit record.
3. **Database-Level Immutability Enforcement**:
   - `platform_audit_logs` is strictly append-only.
   - A PostgreSQL trigger (`platform_audit_logs_immutability_trg`) aborts any `UPDATE` or `DELETE` statement with an exception.
   - Ordinary application queries and standard service roles cannot alter or purge audit records. Note: a sufficiently privileged database owner can alter or disable triggers, so raw database credentials must be strictly guarded. The trigger is designed to defend against application-layer compromise and normal database queries.
4. **Zero Authentication Secrets in Logs**:
   - Audit logs must NEVER capture OTP values, plaintext passwords, JWTs, private keys, or session tokens.
   - Target phone numbers in audit metadata are stored masked or in canonical format.

## Consequences
- Full accountability for administrative actions across the platform.
- Forensically auditable history of which Super Admin enabled or disabled which Admin.
- High query performance with indexing on `(actor_id, created_at)` and `(target_user_id, created_at)`.
- Immutability enforced against application-level operations and standard application connection roles.

## Related Documentation
- [[012-platform-role-separation|ADR 012: Platform Role Separation]]
- [[../architecture/06-platform-administration|Platform Administration Architecture]]
- [[../database/02-platform-administration-schema|Platform Administration Schema]]
- [[../security/02-platform-administration-security|Platform Administration Security]]
