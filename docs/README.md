# CHS Documentation Index

Welcome to the central documentation index for the Cooperative Housing Society (CHS) project. This documentation serves as the authoritative source of truth for architecture, domain models, APIs, and workflows.

## Overview & Architecture

- [[eagle_view|Eagle View Overview]]: High-level architecture, tech stack (Go, SvelteKit, Supabase), core domain models, authentication, security, and roadmap.
- [[sprints/README|Master Sprint Index]]: Roadmap, dependencies, and definitions for sprints S00–S18.

## Technical Architecture

- [[architecture/01-system-overview|01 — System Overview]]: Monorepo layout, system topology, and request flows.
- [[architecture/02-backend|02 — Backend Architecture]]: Modular monolith in Go, Chi router, layering, middleware.
- [[architecture/03-frontend|03 — Frontend Architecture]]: SvelteKit, Tailwind CSS, TanStack Query, central API client.
- [[architecture/04-supabase|04 — Supabase Architecture]]: Supabase Cloud integration, PostgreSQL, migrations, MCP usage.
- [[architecture/05-authentication|05 — Authentication Architecture]]: Phone identity, Go OTP, Ed25519 JWT, sessions, and device limits.
- [[architecture/07-society-onboarding|07 — Society Onboarding Architecture]]: Multi-tenant foundation, society registration, lifecycle state machine, and Super Admin console integration.
- [[architecture/08-society-administration|08 — Society Administration Architecture]]: Society Admin assignment, many-to-many tenancy, active society context switching, and immediate revocation.
- [[architecture/09-society-structure|09 — Society Structure Architecture]]: Buildings, wings, floors, unit auto-generation engine, gates, and gatekeeper staffing.
- [[architecture/10-members-and-relationships|10 — Members & Relationships Architecture]]: People directory, unit relationship modeling, ownership vs occupancy separation, and history preservation.
- [[architecture/11-society-designations-and-appointments|11 — Society Designations and Appointments Architecture]]: Society designation catalogue, committee appointments, term preservation, and role segregation.
- [[architecture/12-gate-foundation-and-staffing|12 — Gate Foundation and Staffing Architecture]]: Operational gate configuration, gatekeeper accounts, shift schedules, and duty sessions.
- [[architecture/13-development-seed-data|13 — Development Seed Data and Reset Architecture]]: Controlled development dataset, dependency-ordered reset, and demo baseline.
- [[architecture/14-gate-visitor-operations|14 — Gate Visitor Operations Architecture]]: Expected visits, walk-ins, resident approvals, hashed passes, and atomic ingress/egress.
- [[architecture/15-society-communication-announcements|15 — Society Communication Architecture]]: Society communication channel, draft/published/archived lifecycle, audience resolution engine, and read tracking.
- [[architecture/16-maintenance-foundation|16 — Maintenance Foundation Architecture]]: Society maintenance structure, charge categories, recurring/one-off fee configs, idempotent billing runs, and immutable line items.
- [[architecture/17-maintenance-operations-and-collections|17 — Maintenance Operations and Collections Architecture]]: Offline payment recording, bill allocation, receipt generation, outstanding balances, and reversal lifecycles.
- [[architecture/18-documents-and-records|18 — Documents and Records Architecture]]: Private object storage via Supabase Storage, multi-tenant RBAC classifications, append-only versioning, streaming downloads, and soft-archival lifecycle.
- [[architecture/19-audit-and-administration|19 — Audit and Administration Architecture]]: Unified audit trail, server-side pagination, search/filtering, multi-tenant isolation, and metadata redaction.
- [[architecture/20-notifications|20 — Notifications Architecture]]: Omnichannel delivery (FCM HTTP v1 + Resend), transactional outbox pattern, device registry, and worker lifecycle.
- [[architecture/21-production-deployment-and-domain-topology|21 — Production Deployment and Domain Topology]]: Production domain society.banesons.com, hosting recommendations, ingress routing, co-location latency principles, and environment specifications.

## API Documentation

- [[api/01-api-overview|01 — API Overview]]: API versioning (`/api/v1`), response envelopes, request ID, and health endpoint.
- [[api/02-authentication-api|02 — Authentication API]]: OTP send/verify, token refresh, logout, logout-all, and `/me`.
- [[api/03-platform-administration-api|03 — Platform Administration API]]: Admin roster listing, creation, status updates, and audit trail inspection.
- [[api/04-society-api|04 — Society Onboarding API]]: Society registration, details, profile updates, activation, suspension, and reason catalogue.
- [[api/05-society-administration-api|05 — Society Administration API]]: Society administrator assignment, listing, revocation, and society profile access.
- [[api/06-society-structure-api|06 — Society Structure API]]: Buildings, wings, deterministic unit preview/generation, gates, and staffing policy.
- [[api/07-members-and-relationships-api|07 — Members & Relationships API]]: Member registration, directory search, profile view, unit relationship lifecycle, and occupancy.
- [[api/08-society-designations-and-appointments-api|08 — Society Designations and Appointments API]]: Designation catalogue CRUD, committee appointment lifecycle, term end/revocation, and roster views.
- [[api/09-gate-foundation-and-staffing-api|09 — Gate Foundation and Staffing API]]: Gate operational parameters, gatekeeper staff roster, shift templates, and duty check-in/out.
- [[api/10-gate-visitor-operations-api|10 — Gate Visitor Operations API]]: Visitor registration, approvals, pass verification, check-in, check-out, and metrics.
- [[api/11-society-communication-api|11 — Society Communication API]]: Announcement CRUD, publication lifecycle, resident feed, read receipts, and unread counts.
- [[api/12-maintenance-foundation-api|12 — Maintenance Foundation API]]: Charge categories, fee configs, billing periods, idempotent bill generation, register, and resident bills.
- [[api/13-maintenance-operations-and-collections-api|13 — Maintenance Operations and Collections API]]: Administrative payment recording, bill allocations, receipts, reversals, collections register, and resident payment history.
- [[api/14-documents-and-records-api|14 — Documents and Records API]]: Document category catalogue, document upload, version revisions, streaming downloads, and archival lifecycles.
- [[api/15-audit-and-administration-api|15 — Audit and Administration API]]: Global platform audit logs, society-scoped audit history, filter parameters, and event detail views.
- [[api/16-notifications-api|16 — Notifications API]]: In-app inbox listing, unread count badge, mark read, bulk mark read, device registration, and deactivation.
- [[api/17-dashboard-reporting-api|17 — Dashboard & Reporting API]]: Platform health summary, society operations and collections scorecards, and gate shift metrics.

## Database Schema Documentation

- [[database/01-authentication-schema|01 — Authentication Database Schema]]: `users`, `auth_otps`, and `user_sessions` tables.
- [[database/02-platform-administration-schema|02 — Platform Administration Database Schema]]: `platform_roles` and `platform_audit_logs` tables.
- [[database/03-society-schema|03 — Society Database Schema]]: `societies` and `society_suspension_reasons` tables, lifecycle constraints, and indexes.
- [[database/04-society-administration-schema|04 — Society Administration Database Schema]]: `society_user_roles` mapping table, active assignment uniqueness, and constraints.
- [[database/05-society-structure-schema|05 — Society Structure Database Schema]]: `society_buildings`, `society_wings`, `society_units`, `society_gates`, and `society_staffing_requirements`.
- [[database/06-members-and-relationships-schema|06 — Members & Relationships Database Schema]]: `society_members` person directory and `unit_relationships` temporal associations.
- [[database/07-society-designations-and-appointments-schema|07 — Society Designations and Appointments Database Schema]]: `society_designations` catalogue and `society_committee_appointments` tables.
- [[database/08-gate-foundation-and-staffing-schema|08 — Gate Foundation and Staffing Database Schema]]: `society_gates` operational extensions, `society_gate_shifts`, and `society_gate_sessions`.
- [[database/09-gate-visitor-operations-schema|09 — Gate Visitor Operations Database Schema]]: `society_visits`, `society_visitor_passes`, `society_visit_approvals`, and `society_gate_events`.
- [[database/10-society-communication-schema|10 — Society Communication Database Schema]]: `society_announcements` and `society_announcement_reads` tables, audience FK constraints, and indexes.
- [[database/11-maintenance-foundation-schema|11 — Maintenance Foundation Database Schema]]: `maintenance_charge_categories`, `maintenance_charge_configs`, `maintenance_billing_periods`, `maintenance_bills`, and `maintenance_bill_items`.
- [[database/12-maintenance-operations-and-collections-schema|12 — Maintenance Operations and Collections Database Schema]]: `maintenance_payments`, `maintenance_payment_allocations`, `maintenance_receipts`, sequences, and immutability triggers.
- [[database/13-documents-and-records-schema|13 — Documents and Records Database Schema]]: `document_categories`, `society_documents`, `document_versions`, sequential number generator, and version immutability triggers.
- [[database/14-audit-and-administration-schema|14 — Audit and Administration Database Schema]]: Append-only audit table extension, society reference, immutability trigger, and dual indexes.
- [[database/15-notifications-schema|15 — Notifications Database Schema]]: `user_devices`, `notifications`, `notification_deliveries`, and `notification_outbox`.

## Security Documentation

- [[security/01-authentication-security|01 — Authentication Security]]: Rate limiting, brute-force mitigation, token rotation, and PII masking.
- [[security/02-platform-administration-security|02 — Platform Administration Security]]: Privilege escalation prevention, immediate session revocation, and audit log immutability.
- [[security/03-society-security|03 — Society Security]]: Tenant boundary protection, application-level isolation, and fail-closed audit integrity.
- [[security/04-society-administration-security|04 — Society Administration Security]]: Society context validation, immediate assignment revocation, and horizontal isolation.
- [[security/05-society-structure-security|05 — Society Structure Security]]: Physical hierarchy boundary checks, bulk generation limits, and cross-society isolation.
- [[security/06-members-and-relationships-security|06 — Members & Relationships Security]]: Resident PII protection, zero privilege escalation, cross-society relationship forgery prevention.
- [[security/07-society-designations-and-appointments-security|07 — Society Designations and Appointments Security]]: Zero implicit privilege escalation, designation vs authorization segregation, and audit integrity.
- [[security/08-gate-foundation-and-staffing-security|08 — Gate Foundation and Staffing Security]]: Gatekeeper role boundaries, duty session concurrency, and perimeter audit integrity.
- [[security/09-gate-visitor-operations-security|09 — Gate Visitor Operations Security]]: Visitor perimeter access, pass entropy, replay prevention, and concurrency row locks.
- [[security/10-society-communication-security|10 — Society Communication Security]]: Tenant boundary protection, audience isolation, fail-closed audit logging, and role privilege boundaries.
- [[security/11-maintenance-foundation-security|11 — Maintenance Foundation Security]]: Multi-tenant financial boundary isolation, IDOR prevention, immutable snapshots, and transactional audit logging.
- [[security/12-maintenance-operations-and-collections-security|12 — Maintenance Operations and Collections Security]]: Payment allocation row locks, exact NUMERIC arithmetic, idempotent retries, immutable receipts, and reversal audit integrity.
- [[security/13-documents-and-records-security|13 — Documents and Records Security]]: Multi-tenant storage isolation, magic-byte signature validation, SHA-256 integrity, safe header injection, and horizontal IDOR defense.
- [[security/14-audit-and-administration-security|14 — Audit and Administration Security]]: Threat model, multi-tenant isolation, role segregation, recursive metadata redaction engine, and trigger protections.
- [[security/15-notifications-security|15 — Notifications Security]]: Backend credential confinement, server-derived identity, lock-screen payload minimization, HTML email sanitization, and token lifecycle.

## Architecture Decision Records (ADRs)

- [[decisions/001-go-sveltekit-supabase|ADR 001]]: Go, SvelteKit, and Supabase Core Stack.
- [[decisions/002-api-contract|ADR 002]]: Unified API Response Contract and Request Tracing.
- [[decisions/003-http-methods|ADR 003]]: Semantic HTTP Methods and Security Model.
- [[decisions/004-payload-encryption|ADR 004]]: Exclusion of Application-Level Payload Encryption.
- [[decisions/005-supabase-mcp|ADR 005]]: Supabase MCP Integration and Migration Authority.
- [[decisions/006-development-environment|ADR 006]]: Local Development Topology and Cloud Supabase.
- [[decisions/007-phone-identity|ADR 007]]: Phone Number as Primary Identity and Canonical Normalization.
- [[decisions/008-go-otp-provider|ADR 008]]: Go Backend as OTP Authority and Pluggable SMS Provider.
- [[decisions/009-go-jwt-authority|ADR 009]]: Go Backend as JWT Authority with Ed25519 Signing.
- [[decisions/010-session-model|ADR 010]]: Stateful Session Model with Refresh Token Rotation.
- [[decisions/011-two-device-limit|ADR 011]]: Strict Two-Device Concurrent Session Limit.
- [[decisions/012-platform-role-separation|ADR 012]]: Separation of Platform Administration from Society Domain Roles.
- [[decisions/013-super-admin-bootstrap|ADR 013]]: Idempotent Super Admin Bootstrapping Mechanism.
- [[decisions/014-platform-audit-trail|ADR 014]]: Append-Only Platform Audit Trail Architecture.
- [[decisions/015-society-lifecycle-and-tenancy|ADR 015]]: Society Lifecycle, Tenancy Boundary, and Multi-Tenant Isolation Strategy.
- [[decisions/016-society-role-and-assignment-model|ADR 016]]: Society Role and Administrator Assignment Model.
- [[decisions/017-society-structure-and-formation|ADR 017]]: Society Structure, Formation Workflow, and Unit Generation Model.
- [[decisions/018-member-and-relationship-model|ADR 018]]: Society Member and Unit Relationship Model.
- [[decisions/019-society-designations-and-appointments|ADR 019]]: Society Designations, Committee Appointments, and Role Segregation.
- [[decisions/020-gate-foundation-and-staffing-operations|ADR 020]]: Gate Foundation, Gatekeeper Roles, and Staffing Operations.
- [[decisions/021-development-database-reset-and-seed|ADR 021]]: Development Database Reset and Controlled Demo Data Seeding.
- [[decisions/022-society-application-shell-and-navigation|ADR 022]]: Society Application Shell and Navigation Architecture.
- [[decisions/023-visitor-lifecycle-and-gate-operations|ADR 023]]: Visitor Lifecycle and Gate Operations.
- [[decisions/024-society-communication-and-announcements|ADR 024]]: Society Communication, Announcement Lifecycle, Audience Targeting, and Read Tracking.
- [[decisions/025-maintenance-foundation-and-billing|ADR 025]]: Maintenance Foundation, Charge Configuration, Billing Cycles, and Immutable Line Items.
- [[decisions/026-maintenance-operations-and-collections|ADR 026]]: Maintenance Operations, Payment Recording, Allocations, Receipts, and Reversals.
- [[decisions/027-documents-and-records|ADR 027]]: Documents and Records Lifecycle, Private Object Storage, and Version Immutability.
- [[decisions/028-audit-and-administration|ADR 028]]: Unified Audit and Administration Architecture, Single Ledger Extension, and Tenant Isolation.
- [[decisions/029-notifications|ADR 029]]: Notification System Architecture (Firebase Cloud Messaging + Resend Email).
- [[decisions/030-client-performance-monitoring|ADR 030]]: Client-Side Performance Monitoring (Firebase Performance).
- [[decisions/031-production-domain-and-deployment-topology|ADR 031]]: Production Domain Configuration and Deployment Topology (society.banesons.com).

## Sprint Trackers

- [[sprints/S06-members-relationships|S06 — Members & Relationships]]: Status: `COMPLETED`.
- [[sprints/S07-society-roles-designations|S07 — Society Roles & Designations]]: Status: `COMPLETED`.
- [[sprints/S08-gate-foundation|S08 — Gate Foundation & Staffing]]: Status: `COMPLETED`.
- [[sprints/S09-gate-operations|S09 — Gate Operations & Visitor Management]]: Status: `COMPLETED`.
- [[sprints/S10-society-communication|S10 — Society Communication & Announcements]]: Status: `COMPLETED`.
- [[sprints/S11-maintenance-foundation|S11 — Maintenance Foundation]]: Status: `COMPLETED`.
- [[sprints/S12-maintenance-operations-and-collections|S12 — Maintenance Operations & Collections]]: Status: `COMPLETED`.
- [[sprints/S13-documents-and-records|S13 — Documents & Records]]: Status: `COMPLETED`.
- [[sprints/S14-audit-and-administration|S14 — Audit & Administration]]: Status: `COMPLETED`.
- [[sprints/S15-notifications|S15 — Notifications]]: Status: `COMPLETED` (FCM device delivery deferred).
- [[sprints/S16-dashboard-and-reporting|S16 — Dashboard & Reporting]]: Status: `COMPLETED`.
- [[sprints/S17-security-hardening|S17 — Security & Hardening]]: Status: `COMPLETED`.
- [[sprints/S18-production-readiness|S18 — Production Readiness & Release]]: Status: `AUDITED / BLOCKED ON OPERATOR PREREQUISITES`.


## Structure Roadmap

```text
docs/
├── README.md               # Central entry point (this file)
├── eagle_view.md           # Master architectural & domain specification
├── sprints/                # Sprint roadmap & definitions (S00–S18)
├── architecture/           # Technical designs & diagrams
├── api/                    # API specifications & contracts
├── database/               # Database schema specifications
├── security/               # Security specifications & threat models
├── decisions/              # Architecture Decision Records (ADRs)
├── product/                # Platform requirements & workflows (Future)
└── domain/                 # User, society, unit domain entities (S03+)
```
