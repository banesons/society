# CHS — Master Sprint Index

This document defines the master sprint roadmap for the initial Cooperative Housing Society (CHS) platform build, focusing on **Society Management** and **Gate Management**.

Restaurant/POS (**Restro**) is explicitly out of scope for sprints S00–S18 and will be planned as a separate sprint roadmap (S19+) once the society and gate foundations are stable.

---

## Sprint Philosophy

- **Unified Full-Stack Increments**: Every sprint contains both **Backend (BE)** and **Frontend (FE)** scope. No separate BE or FE sprints.
- **Usable Increments**: Every sprint delivers an end-to-end usable outcome wherever practical.
- **Concise Definitions**: Sprint definitions are kept high-level at this planning stage. Detailed implementation plans are created when each sprint begins.
- **Documentation-First**: All technical, domain, and API decisions must be synchronized in `/docs` per [[../../.agent/skills/chs-documentation-first/SKILL|chs-documentation-first]].

---

## Master Sprint Sequence

| Sprint | Area | Scope Summary | Link |
| :--- | :--- | :--- | :--- |
| **S00** | Foundation | Project setup, docs, environments, Supabase, Go, SvelteKit | [[S00-foundation]] |
| **S01** | Identity & Authentication | User identity, OTP/login, JWT, sessions | [[S01-identity-authentication]] |
| **S02** | Platform Administration | Super Admin, Admin creation, admin management | [[S02-platform-administration]] |
| **S03** | Society Onboarding | Society creation, pending/approval/activation lifecycle | [[S03-society-onboarding]] |
| **S04** | Society Administration | Society Admin, society mapping, administration access | [[S04-society-administration]] |
| **S05** | Society Structure | Buildings/wings, units/flats | [[S05-society-structure]] |
| **S06** | Members & Relationships | Owners, tenants, family members | [[S06-members-relationships]] |
| **S07** | Society Roles & Designations | Secretary, Treasurer, Chairman, predefined designations | [[S07-society-roles-designations]] |
| **S08** | Gate Foundation | Gate configuration, gatekeepers, access foundation | [[S08-gate-foundation]] |
| **S09** | Gate Operations | Visitors, entries/exits, approvals, gate operations | [[S09-gate-operations]] |
| **S10** | Society Communication | Notices, announcements, basic notifications | [[S10-society-communication]] |
| **S11** | Maintenance Foundation | Maintenance configuration, charges, billing periods | [[S11-maintenance-foundation]] |
| **S12** | Maintenance Operations | Bills, payments, dues, statements | [[S12-maintenance-operations]] |
| **S13** | Documents & Records | Society/member documents and document management | [[S13-documents-records]] |
| **S14** | Audit & Administration | Activity logs, audit trails, administrative history | [[S14-audit-administration]] |
| **S15** | Notifications | Notification infrastructure and preferences | [[S15-notifications]] |
| **S16** | Dashboard & Reporting | Society/admin dashboards and operational reports | [[S16-dashboard-reporting]] |
| **S17** | Security & Hardening | Authorization review, PII, encryption, performance | [[S17-security-hardening]] |
| **S18** | Production Readiness | Deployment, monitoring, backups, CI/CD, production | [[S18-production-readiness]] |

---

## Sprint Obsidian Index

- [[S00-foundation]]
- [[S01-identity-authentication]]
- [[S02-platform-administration]]
- [[S03-society-onboarding]]
- [[S04-society-administration]]
- [[S05-society-structure]]
- [[S06-members-relationships]]
- [[S07-society-roles-designations]]
- [[S08-gate-foundation]]
- [[S09-gate-operations]]
- [[S10-society-communication]]
- [[S11-maintenance-foundation]]
- [[S12-maintenance-operations]]
- [[S13-documents-records]]
- [[S14-audit-administration]]
- [[S15-notifications]]
- [[S16-dashboard-reporting]]
- [[S17-security-hardening]]
- [[S18-production-readiness]]

---

## Sprint Dependency Chain

```text
S00 Foundation
      ↓
S01 Identity & Authentication
      ↓
S02 Platform Administration
      ↓
S03 Society Onboarding
      ↓
S04 Society Administration
      ↓
S05 Society Structure
      ↓
S06 Members & Relationships
      ↓
S07 Roles & Designations
      ↓
S08 Gate Foundation
      ↓
S09 Gate Operations
      ↓
S10 Communication
      ↓
S11 Maintenance Foundation
      ↓
S12 Maintenance Operations
      ↓
S13 Documents
      ↓
S14 Audit
      ↓
S15 Notifications
      ↓
S16 Dashboard & Reporting
      ↓
S17 Security & Hardening
      ↓
S18 Production Readiness
```

> Some later sprints may execute partially in parallel once foundations are in place, but the chain above defines the initial dependency sequence.

---

## Out of Scope: Restro (POS)

Restaurant/POS (**Restro**) is explicitly **NOT** part of sprints S00–S18.
After the core Society Management + Gate foundation is stable:
- **S19+** → Restro Product
Restro will receive its own standalone sprint roadmap and requirements specification.

---

## Planning Constraints & Change Protocol

The sprint roadmap is a living planning artifact, not permanently immutable:
- If requirements reveal that a sprint should be split, merged, reordered, added, or removed, update this master sprint index and document the decision.
- Never silently modify the sprint sequence.

---

## Related Documentation

- [[../README|CHS Documentation Index]]
- [[../eagle_view|Eagle View Architecture Specification]]
