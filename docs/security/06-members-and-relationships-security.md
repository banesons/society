# 06 — Members & Relationships Security Specification

## 1. Threat Model & Principles

Sprint S06 manages resident identities, ownership records, tenancy agreements, and contact details.

Core Security Principles:
1. **Zero Privilege Escalation**: Member status, ownership, and tenancy convey **zero** platform or society administration permissions.
2. **Multi-Tenant Boundary Hardening**: Barred by database composite foreign keys `(unit_id, society_id)` and `(member_id, society_id)`.
3. **Data Minimization & PII Masking**: In roster views and public directory listings, phone numbers (`+91 9967xxxx01`) and emails (`ra****@example.com`) are masked. Full details returned only on explicit single-entity lookup.
4. **Authentication Integrity**: Society administrators cannot create synthetic platform authentication credentials. Unverified contact records in `society_members` remain separate from verified `users` until the person authenticates via OTP.
5. **Non-Destructive Deactivation & Title Safety**: Deactivating a member cannot silently extinguish ownership claims. Ownership transfers and occupancy terminations require explicit confirmation and audit logging.
6. **Concurrency Race Condition Mitigation**:
   - Aggregate ownership shares protected via pessimistic unit row locking (`SELECT ... FOR UPDATE`).
   - Departure race conditions mitigated via cryptographic version tokens (`409 PREVIEW_STALE`).
7. **Transactional Audit Logging**: All changes to members and occupancy records are permanently audited in `platform_audit_logs`.

---

## 2. Authorization Matrix

| Endpoint | Anonymous | Platform ADMIN | SOC_ADMIN (Foreign) | SOC_ADMIN (Assigned) | SUPER_ADMIN |
| :--- | :---: | :---: | :---: | :---: | :---: |
| `GET /api/v1/societies/:id/members` | 401 | 403 | 403 | **200 (Masked)** | **200 (Masked)** |
| `POST /api/v1/societies/:id/members` | 401 | 403 | 403 | **201** | **201** |
| `GET /api/v1/societies/:id/members/:mId` | 401 | 403 | 403 | **200 (Full)** | **200 (Full)** |
| `PATCH /api/v1/societies/:id/members/:mId` | 401 | 403 | 403 | **200** | **200** |
| `POST /api/v1/societies/:id/members/:mId/departure/preview` | 401 | 403 | 403 | **200** | **200** |
| `POST /api/v1/societies/:id/members/:mId/departure/confirm` | 401 | 403 | 403 | **200** | **200** |
| `GET /api/v1/societies/:id/units/:uId/relationships` | 401 | 403 | 403 | **200** | **200** |
| `POST /api/v1/societies/:id/units/:uId/relationships` | 401 | 403 | 403 | **201** | **201** |
| `POST /api/v1/societies/:id/units/:uId/relationships/:rId/end` | 401 | 403 | 403 | **200** | **200** |

---

## 3. Data Minimization & PII Policy

In accordance with [[../decisions/002-api-contract|ADR 002]] and `chs-documentation-first` Section 11:
- Roster responses mask phone numbers: `+91 9967xxxx01`.
- Roster responses mask emails: `ra****@example.com`.
- Full contact details are only returned on explicit single-entity lookup by an authorized administrator.
- Members cannot access other members' private contact details.

---

## 4. Cross-Society Attack Mitigation

- The middleware verifies that `:id` matches the caller's authorized society.
- Database engine composite foreign keys physically reject any `unit_relationships` insert where `unit_id` or `member_id` does not match `society_id`.
- Any mismatch is rejected with `400 Bad Request` or `404 Not Found`.

---

## 5. Audit Logging Specifications

Mutations are audited transactionally in `platform_audit_logs`:
- Actions: `MEMBER_REGISTERED`, `MEMBER_UPDATED`, `MEMBER_STATUS_CHANGED`, `MEMBER_DEPARTED`, `UNIT_RELATIONSHIP_CREATED`, `UNIT_RELATIONSHIP_ENDED`.
- Actor: Authenticated `user_id`.
- Tenant: `society_id`.
- Metadata: Entity identifiers and masked phone numbers only; zero unmasked PII or credentials recorded.

---

## 6. References
- [[../decisions/012-platform-role-separation|ADR 012: Platform Role Separation]]
- [[../decisions/016-society-role-and-assignment-model|ADR 016: Society Role and Administrator Assignment Model]]
- [[../decisions/018-member-and-relationship-model|ADR 018: Member and Relationship Model]]
- [[../api/07-members-and-relationships-api|07 — Members & Relationships API Specification]]
