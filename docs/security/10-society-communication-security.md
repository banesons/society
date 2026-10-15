# 10 — Society Communication Security

## Overview
This document specifies the security requirements, threat mitigations, and authorization boundaries for Sprint S10 (Society Communication & Announcements).

---

## 1. Threat Model & Mitigations

| Threat | Risk Level | Mitigation |
| :--- | :--- | :--- |
| **Cross-Society Target Forgery** | High | Database composite foreign keys `(target_building_id, society_id)`, `(target_wing_id, society_id)`, and `(target_unit_id, society_id)` reject mismatched society references. |
| **Unauthorized Notice Publication** | High | Publication endpoints (`POST .../publish`, `POST .../archive`) strictly require verified `SOC_ADMIN` or `SUPER_ADMIN` authorization via middleware. |
| **Draft Exposure to Residents** | Medium | Feed and detail queries enforce `status = 'PUBLISHED'` for non-administrator callers. |
| **Cross-Building Audience Leakage** | Medium | Dynamic audience matching joins active `unit_relationships` and `society_units` server-side. Caller cannot manipulate SQL filters. |
| **Read Receipt Forgery / Denial of Service** | Low | Read requests extract authenticated `user_id` strictly from trusted JWT context; unique composite index prevents row inflation. |
| **Platform ADMIN Privilege Spill** | High | Platform `ADMIN` has no automatic society announcement permissions. Requires explicit `society_user_roles` assignment. |

---

## 2. Role Permissions Matrix

| Operation | `SUPER_ADMIN` | Platform `ADMIN` | `SOC_ADMIN` | `RESIDENT` | `GATEKEEPER` |
| :--- | :---: | :---: | :---: | :---: | :---: |
| Create Draft Announcement | ✅ | ❌ | ✅ | ❌ | ❌ |
| Edit Draft Announcement | ✅ | ❌ | ✅ | ❌ | ❌ |
| Publish Announcement | ✅ | ❌ | ✅ | ❌ | ❌ |
| Archive Announcement | ✅ | ❌ | ✅ | ❌ | ❌ |
| List Admin Announcements | ✅ | ❌ | ✅ | ❌ | ❌ |
| View Published Feed | ✅ | ❌ | ✅ | ✅ (Audience-scoped) | ✅ (Society-wide) |
| Mark Announcement Read | ✅ | ❌ | ✅ | ✅ (Audience-scoped) | ✅ (Society-wide) |
| Get Unread Count | ✅ | ❌ | ✅ | ✅ (Audience-scoped) | ✅ (Society-wide) |

---

## 3. Audit Immutability & Fail-Closed Behavior
All privileged mutations (`ANNOUNCEMENT_CREATED`, `ANNOUNCEMENT_UPDATED`, `ANNOUNCEMENT_PUBLISHED`, `ANNOUNCEMENT_ARCHIVED`) invoke `LogPlatformActionTx` within the same database transaction.
If audit row insertion fails or table is locked, the database transaction rolls back immediately with HTTP 500 `INTERNAL_ERROR`.

---

## 4. Related Documentation
- [[../decisions/024-society-communication-and-announcements|ADR 024: Society Communication]]
- [[../architecture/15-society-communication-announcements|Architecture 15: Society Communication]]
- [[../database/10-society-communication-schema|Database Schema 10: Society Communication Schema]]
- [[../api/11-society-communication-api|API 11: Society Communication API]]
- [[../sprints/S10-society-communication|S10 Sprint Specification]]
