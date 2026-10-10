# 02 — Platform Administration Security Controls

## Threat Analysis & Countermeasures

| Threat | Impact | Countermeasure |
| :--- | :--- | :--- |
| **Unauthorized Privilege Escalation** | Attacker elevates account to Admin/Super Admin | - No elevation endpoints in API.<br>- Strict `RequireSuperAdmin` Go middleware.<br>- Initial `SUPER_ADMIN` assigned exclusively via server-side config/bootstrap.<br>- Existing roles are never silently overwritten.<br>- Creation of `SUPER_ADMIN` via UI/API is blocked. |
| **Disabled Admin Session Retention** | Suspended admin continues accessing admin functions | - Disabling an admin updates `platform_roles.status = 'DISABLED'` and revokes all active sessions in `user_sessions`.<br>- Base identity `users.status` remains `'ACTIVE'`.<br>- Go authorization checks verify database role status on administrative operations. |
| **Audit Log Tampering / Repudiation** | Attacker deletes or modifies audit logs | - `platform_audit_logs` is append-only with database trigger forbidding UPDATE and DELETE.<br>- Protects against application-layer tampering and standard DB roles; note that privileged database owners can alter triggers, so DB credentials must be secured.<br>- Audit records are written in the same transaction as administrative mutations; failure to record audit rolls back the action.<br>- No authentication secrets are ever logged. |
| **Super Admin Self-Lockout** | Super admin accidentally disables own account | - Handler checks `actor_id != target_user_id` when updating admin status.<br>- Bootstrap Super Admin cannot be disabled through the API (`CANNOT_DISABLE_SELF`). |
| **PII Exposure in Admin Consoles** | Administrator phone numbers scraped | - Masked phone formatting (`+91 9820xxxx56`) in all admin list and detail responses by default. |

---

## Authorization Boundaries

1. **Server-Side Enforcement**:
   - The Go backend is the sole authority for administrative permissions.
   - Frontend route visibility, navigation hiding, or role flags are UX conveniences and NEVER treated as security barriers.
2. **Context Integrity**:
   - `RequireSuperAdmin` middleware extracts the caller's identity strictly from the verified Ed25519 Bearer token in the request context, never from client-supplied body parameters or query strings.
3. **Audit Trail Accountability**:
   - Every status modification, creation, or reactivation must log the authenticated caller ID, target user ID, IP address, and timestamp.

---

## Related Documentation
- [[../architecture/06-platform-administration|Platform Administration Architecture]]
- [[../decisions/012-platform-role-separation|ADR 012: Platform Role Separation]]
- [[../decisions/013-super-admin-bootstrap|ADR 013: Super Admin Bootstrap]]
- [[../decisions/014-platform-audit-trail|ADR 014: Platform Audit Trail]]
