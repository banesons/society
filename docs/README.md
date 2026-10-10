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

## API Documentation

- [[api/01-api-overview|01 — API Overview]]: API versioning (`/api/v1`), response envelopes, request ID, and health endpoint.
- [[api/02-authentication-api|02 — Authentication API]]: OTP send/verify, token refresh, logout, logout-all, and `/me`.

## Database Schema Documentation

- [[database/01-authentication-schema|01 — Authentication Database Schema]]: `users`, `auth_otps`, and `user_sessions` tables.

## Security Documentation

- [[security/01-authentication-security|01 — Authentication Security]]: Rate limiting, brute-force mitigation, token rotation, and PII masking.

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
