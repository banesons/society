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

## API Documentation

- [[api/01-api-overview|01 — API Overview]]: API versioning (`/api/v1`), response envelopes, request ID, and health endpoint.
- [[api/02-authentication-api|02 — Authentication API]]: OTP send/verify, token refresh, logout, logout-all, and `/me`.
- [[api/03-platform-administration-api|03 — Platform Administration API]]: Admin roster listing, creation, status updates, and audit trail inspection.
- [[api/04-society-api|04 — Society Onboarding API]]: Society registration, details, profile updates, activation, suspension, and reason catalogue.
- [[api/05-society-administration-api|05 — Society Administration API]]: Society administrator assignment, listing, revocation, and society profile access.
- [[api/06-society-structure-api|06 — Society Structure API]]: Buildings, wings, deterministic unit preview/generation, gates, and staffing policy.
- [[api/07-members-and-relationships-api|07 — Members & Relationships API]]: Member registration, directory search, profile view, unit relationship lifecycle, and occupancy.

## Database Schema Documentation

- [[database/01-authentication-schema|01 — Authentication Database Schema]]: `users`, `auth_otps`, and `user_sessions` tables.
- [[database/02-platform-administration-schema|02 — Platform Administration Database Schema]]: `platform_roles` and `platform_audit_logs` tables.
- [[database/03-society-schema|03 — Society Database Schema]]: `societies` and `society_suspension_reasons` tables, lifecycle constraints, and indexes.
- [[database/04-society-administration-schema|04 — Society Administration Database Schema]]: `society_user_roles` mapping table, active assignment uniqueness, and constraints.
- [[database/05-society-structure-schema|05 — Society Structure Database Schema]]: `society_buildings`, `society_wings`, `society_units`, `society_gates`, and `society_staffing_requirements`.
- [[database/06-members-and-relationships-schema|06 — Members & Relationships Database Schema]]: `society_members` person directory and `unit_relationships` temporal associations.

## Security Documentation

- [[security/01-authentication-security|01 — Authentication Security]]: Rate limiting, brute-force mitigation, token rotation, and PII masking.
- [[security/02-platform-administration-security|02 — Platform Administration Security]]: Privilege escalation prevention, immediate session revocation, and audit log immutability.
- [[security/03-society-security|03 — Society Security]]: Tenant boundary protection, application-level isolation, and fail-closed audit integrity.
- [[security/04-society-administration-security|04 — Society Administration Security]]: Society context validation, immediate assignment revocation, and horizontal isolation.
- [[security/05-society-structure-security|05 — Society Structure Security]]: Physical hierarchy boundary checks, bulk generation limits, and cross-society isolation.
- [[security/06-members-and-relationships-security|06 — Members & Relationships Security]]: Resident PII protection, zero privilege escalation, cross-society relationship forgery prevention.

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
