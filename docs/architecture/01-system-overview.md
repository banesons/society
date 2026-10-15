# 01 — System Overview

## System Architecture

The Cooperative Housing Society (CHS) platform is architected as a lightweight, high-performance web platform focused initially on Society Management and Gate Security.

```text
┌────────────────────────────────────────────────────────┐
│                   Browser Client                       │
│              (SvelteKit Frontend :5173)                │
└──────────────────────────┬─────────────────────────────┘
                           │ HTTP / JSON (/api/v1/*)
                           │ X-Request-ID
                           ▼
┌────────────────────────────────────────────────────────┐
│                   Go Backend (:8080)                   │
│   ┌────────────────────────────────────────────────┐   │
│   │ Middleware (RequestID, Logger, Recover, CORS)   │   │
│   └──────────────────────┬─────────────────────────┘   │
│                          ▼                             │
│   ┌────────────────────────────────────────────────┐   │
│   │ Handlers (Health, /api/v1/*)                   │   │
│   └──────────────────────┬─────────────────────────┘   │
│                          ▼                             │
│   ┌────────────────────────────────────────────────┐   │
│   │ Services (Business Logic)                      │   │
│   └──────────────────────┬─────────────────────────┘   │
│                          ▼                             │
│   ┌────────────────────────────────────────────────┐   │
│   │ Repositories (Data Access)                     │   │
│   └──────────────────────┬─────────────────────────┘   │
└──────────────────────────┼─────────────────────────────┘
                           │ TLS / TCP (Database Connection)
                           ▼
┌────────────────────────────────────────────────────────┐
│               Supabase Cloud (PostgreSQL)              │
│                  Region: ap-south-1                    │
└────────────────────────────────────────────────────────┘
```

## Monorepo Layout

```text
chs_neu/
├── .agent/skills/         # Agent skills & development workflows
├── docs/                  # Authoritative documentation source of truth
├── backend/               # Go backend service (Chi router)
│   ├── cmd/api/           # Application entry point
│   ├── internal/          # Private application code (config, server, middleware, etc.)
│   ├── .env.example       # Template backend environment config
│   └── go.mod             # Go module definition
├── frontend/              # SvelteKit client application
│   ├── src/               # SvelteKit source code
│   │   ├── lib/           # Shared modules (API client, UI components)
│   │   └── routes/        # Application routes & layouts
│   ├── .env.example       # Template frontend environment config
│   └── package.json       # Node package manifest
├── supabase/              # Supabase configuration & migrations
│   ├── config.toml        # Supabase CLI project config
│   └── migrations/        # SQL migration files (schema source of truth)
├── .github/workflows/     # CI verification workflows
├── .gitignore             # Root git ignore specification
└── README.md              # Project overview & quickstart
```

## Communication Principles

1. **Versioned REST APIs**: All endpoints are mounted under `/api/v1`.
2. **Semantic HTTP**: Methods match action semantics (`GET`, `POST`, `PUT`/`PATCH`, `DELETE`).
3. **Structured Tracing**: Every request is tagged with an `X-Request-ID` header.
4. **Standard Envelope**: Predictable success and error JSON envelopes across all responses.

## Related Documentation
- [[02-backend|Backend Architecture]]
- [[03-frontend|Frontend Architecture]]
- [[04-supabase|Supabase Architecture]]
- [[21-production-deployment-and-domain-topology|Production Deployment & Domain Topology]]
- [[../decisions/001-go-sveltekit-supabase|ADR 001: Core Stack]]
- [[../decisions/006-development-environment|ADR 006: Development Topology]]
- [[../decisions/031-production-domain-and-deployment-topology|ADR 031: Production Domain Configuration]]
