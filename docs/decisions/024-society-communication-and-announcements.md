# ADR 024: Society Communication, Announcements, and Audience Targeting

## Context
Cooperative Housing Societies require an official, authoritative communication channel through which elected administrators and managing committees can broadcast announcements, notices, and critical operational alerts to residents and occupants.

Prior to S10, the platform supported physical structure formation (S05), member-to-unit directory associations (S06), designation management (S07), and gate operations (S08/S09), but had no official channel for society-wide or audience-scoped communications.

Requirements for S10:
1. Society administrators (`SOC_ADMIN`) need to create, draft, edit, publish, and archive announcements.
2. Announcements must support audience targeting: society-wide (`SOCIETY`), specific building (`BUILDING`), specific wing (`WING`), or specific unit (`UNIT`).
3. Cross-society targeting must be strictly prohibited and architecturally impossible.
4. Residents (`RESIDENT`) must have a unified chronological announcement feed displaying only published announcements matching their authorized resident occupancy.
5. Residents must be able to track read/unread state idempotently with accurate unread badge counters.
6. Archival of announcements removes them from the resident feed while preserving full administrative history.
7. General notification delivery (SMS, WhatsApp, Push, Email) belongs to S15 and must not be implemented here; however, a clean domain event boundary must be maintained.
8. Platform administrators (`ADMIN`) must not receive implicit society access.
9. All privileged operations must emit immutable, fail-closed transactional audit records to `platform_audit_logs`.

## Decisions

### 1. Announcement Lifecycle State Machine
Announcements follow a strict three-state lifecycle:
```
DRAFT ──(Publish)──> PUBLISHED ──(Archive)──> ARCHIVED
  │                     │
  └───(Update Draft)────┘──(Update Body/Category - Preserves Original published_at)
```
- **DRAFT**: Created by authorized `SOC_ADMIN`. Editable. Invisible to residents.
- **PUBLISHED**: Accessible by residents belonging to the targeted audience. `published_at` and `published_by` recorded immutably. Edits to published announcements update `updated_at` without resetting `published_at`, preserving publication chronological sequencing.
- **ARCHIVED**: Transitioned by `SOC_ADMIN`. Excluded from standard resident feed and unread calculations. Retained for administrative historical compliance.

### 2. Normalized Audience Targeting & Composite Foreign Key Isolation
Audience types are strictly restricted to:
- `SOCIETY`: Broadcast to all members and residents of the society.
- `BUILDING`: Scoped to occupants of `target_building_id`.
- `WING`: Scoped to occupants of `target_wing_id`.
- `UNIT`: Scoped to occupants of `target_unit_id`.

To prevent cross-society target tampering, the database adds composite foreign keys linking `(target_building_id, society_id)`, `(target_wing_id, society_id)`, and `(target_unit_id, society_id)` directly to the physical structure tables. Target references crossing society boundaries are rejected at the database engine constraint level.

### 3. Resident Audience Visibility Resolution
Resident visibility is evaluated dynamically server-side using active `unit_relationships` and `society_members`:
- A resident matches a `SOCIETY` announcement if they have an active `society_members` or `unit_relationships` record in the society.
- A resident matches a `BUILDING`, `WING`, or `UNIT` announcement if they hold an active `unit_relationships` (`status = 'ACTIVE'`) record to a unit mapped to that target building, wing, or unit ID.
- Gatekeepers are limited to viewing society-wide announcements relevant to perimeter operations and cannot manage communications.

### 4. Per-User Read Tracking & Idempotency
Read state is isolated into `society_announcement_reads`:
- Composite unique constraint `(announcement_id, user_id)` guarantees idempotency (`ON CONFLICT DO NOTHING`).
- Marking an announcement as read requires verifying that the user is authorized to view that specific announcement.
- Unread counts query only accessible, `PUBLISHED` announcements where no read record exists for `user_id`.

### 5. Boundary with S15 Notification Engine
S10 focuses exclusively on the canonical data storage, lifecycle, API, and resident feed. Outbound SMS dispatches, push notifications, and email delivery belong to S15. S10 triggers internal audit logging but does not invoke external dispatchers.

## Consequences

### Positive
- Strict tenant and audience isolation prevents data leaks between societies or unrelated buildings.
- Clean separation between draft preparation, active publication, and administrative archival.
- Idempotent read tracking scales efficiently using covering indexes.
- Audit logging provides full provenance of who created, published, or archived communications.

### Trade-offs & Mitigations
- Occupancy changes: If a resident moves out, their historical read records are retained, but inactive relationships prevent seeing newly published building-specific announcements.
- Single target per announcement: Announcements in S10 target either the entire society or one specific building/wing/unit. Multi-building targeting is achieved via multiple announcements or society-wide broadcast.

## Status
Accepted

## Related Documentation
- [[../architecture/15-society-communication-announcements|Architecture 15: Society Communication]]
- [[../database/10-society-communication-schema|Database Schema 10: Society Communication Schema]]
- [[../api/11-society-communication-api|API 11: Society Communication API]]
- [[../security/10-society-communication-security|Security 10: Society Communication Security]]
- [[../sprints/S10-society-communication|S10 Sprint Specification]]
