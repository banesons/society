# Banesons Society (`society.banesons.com`)

[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go)](https://golang.org)
[![SvelteKit](https://img.shields.io/badge/SvelteKit-2.0-FF3E00?style=flat&logo=svelte)](https://svelte.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?style=flat&logo=postgresql)](https://www.postgresql.org)
[![Supabase](https://img.shields.io/badge/Supabase-ap--south--1-3ECF8E?style=flat&logo=supabase)](https://supabase.com)
[![License](https://img.shields.io/badge/License-Proprietary-red.svg)]()

**Banesons Society** is a high-performance, documentation-first cloud operating system engineered for residential housing societies, gated communities, and multi-tower apartment complexes. 

Production URL: [https://society.banesons.com](https://society.banesons.com)

---

## Architecture Overview

```text
┌────────────────────────────────────────────────────────────────────────┐
│                          End-User Browser / PWA                        │
│                 https://society.banesons.com (Global Edge)             │
│                (SvelteKit 2, TailwindCSS v4, Svelte 5 Runes)           │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ HTTPS / REST API (/api/v1/*)
                                    │ Ed25519 Bearer Token + X-Session-ID
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        Go 1.24 Chi REST API Engine                     │
│                  https://api.society.banesons.com                      │
│                  Region: AWS Mumbai (ap-south-1)                       │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Security Middleware (Strict CORS, Headers, Rate Limiting, Size)  │  │
│  ├──────────────────────────────────────────────────────────────────┤  │
│  │ 17 Scoped Domain Handlers (Auth, Members, Gate, Finance, Docs)   │  │
│  ├──────────────────────────────────────────────────────────────────┤  │
│  │ Pure pgx/v5 Connection Pooler (<2ms Query Execution)            │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ Encrypted TLS Connection (Port 6543)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                   Supabase Cloud PostgreSQL Database                   │
│                     Region: AWS Mumbai (ap-south-1)                    │
│   • 21 Normalized Schemas     • Immutability Triggers (Audit/Bills)   │
│   • Multi-Tenant Isolation    • Row-Level Locking on Allocations       │
└────────────────────────────────────────────────────────────────────────┘
```

---

## Core Capabilities

1. **Identity & Stateful Authentication**:
   - Phone-number primary identity with canonical E.164 normalization.
   - Dual-factor OTP verification with pluggable SMS routing (MSG91).
   - Ed25519 asymmetric cryptographic JWTs with server-tracked sessions.
   - Strict 2-device concurrent login limit with immediate revocation cascades.
2. **Platform & Society Governance**:
   - Segregated Platform Administration (`SUPER_ADMIN`, `ADMIN`) and Society-level roles (`SOC_ADMIN`, `GATEKEEPER`).
   - Society onboarding workflow with administrative review, approval, and audit trails.
3. **Complex Physical Topology & Occupancy**:
   - Multi-building and multi-wing layout engine with deterministic batch unit generation.
   - Temporal resident directory: Primary/Secondary Owners, Tenants, and Family occupants.
4. **Gate Perimeter Security & Visitor Lifecycle**:
   - Digital guard console with shift management and attendance checkpoints.
   - Real-time visitor arrival processing: Digital passes, vehicle registration, and resident approval modals.
5. **Financial Operations & Maintenance Billing**:
   - Recurring and one-time charge templates (flat, per-sqft, or consumption-based).
   - Idempotent monthly billing runs with immutable line items.
   - Payment allocation engine with exact NUMERIC arithmetic, receipt generation, and reversal tracking.
6. **Encrypted Records Vault**:
   - Private document storage with magic-byte signature validation and SHA-256 integrity checks.
   - Multi-tenant classification (AGM Minutes, Bylaws, Audits) with immutable versioning.
7. **Tamper-Proof Audit Trail**:
   - System-wide append-only audit ledger protected by PostgreSQL database immutability triggers.
   - Server-side metadata sanitization engine redacting sensitive financial and personal details.
8. **Omnichannel Real-Time Notifications**:
   - Transactional outbox pattern with background worker polling.
   - Web Push (FCM HTTP v1) for desktop/mobile browsers, transactional email (Resend), and SMS (MSG91).

---

## Repository Monorepo Structure

```text
society/
├── .agent/skills/         # Antigravity agent skills & documentation-first workflows
├── docs/                  # Authoritative single source of truth (Obsidian-compatible)
│   ├── README.md          # Master documentation registry
│   ├── eagle_view.md      # Platform domain model & architecture bible
│   ├── sprints/           # S00–S18 Sprint audit specifications
│   ├── architecture/      # 21 Architectural blueprints (Systems, Gate, Billing, etc.)
│   ├── api/               # 17 Versioned REST API specifications (/api/v1/*)
│   ├── database/          # 15 Database schema definitions & constraints
│   ├── security/          # 15 Security threat models & isolation policies
│   └── decisions/         # 31 Architecture Decision Records (ADRs)
├── backend/               # Go 1.24 REST API service
│   ├── cmd/api/           # Production API entry point
│   ├── cmd/seed/          # Canonical demo seed & reset runner
│   ├── internal/          # Domain services, repositories, and Chi HTTP handlers
│   └── .env.example       # Template backend environment variables
├── frontend/              # SvelteKit 2 client web application
│   ├── src/               # UI components, stores, and API clients
│   └── .env.example       # Template frontend environment variables
├── supabase/              # Database migrations & pooler config
│   └── migrations/        # 21 Applied SQL migration files
├── .gitignore             # Comprehensive root gitignore
└── README.md              # Master project documentation (this file)
```

---

## Local Development Setup

### Prerequisites
- **Go**: 1.24+
- **Node.js**: 22+ (npm 10+)
- **PostgreSQL**: Accessible Supabase PostgreSQL instance (or local PostgreSQL)

### 1. Backend Configuration
```bash
cd backend
cp .env.example .env
# Configure DATABASE_URL, SUPABASE_URL, and SUPABASE_ANON_KEY in backend/.env

# Run API server
go run ./cmd/api
```
The backend initializes on `http://localhost:8080`. Verify health:
```bash
curl http://localhost:8080/health
```

### 2. Frontend Configuration
```bash
cd frontend
cp .env.example .env
# Default PUBLIC_API_URL points to http://localhost:8080/api/v1

# Install dependencies and launch dev server
npm install
npm run dev
```
Open `http://localhost:5173` in your browser.

---

## Verification & Quality Gates

Every change must satisfy all 5 automated quality gates:

```bash
# 1. Backend test suite (100% passing across all packages)
cd backend && go test -count=1 ./...

# 2. Frontend unit tests (160 tests passing)
cd frontend && npm test

# 3. Frontend type checking
cd frontend && npm run check

# 4. Frontend linting & code formatting
cd frontend && npm run lint

# 5. Production frontend bundle build
cd frontend && npm run build
```

---

## Production Deployment Topology

See [docs/architecture/21-production-deployment-and-domain-topology.md](docs/architecture/21-production-deployment-and-domain-topology.md) and [ADR 031](docs/decisions/031-production-domain-and-deployment-topology.md) for full deployment runbooks.

- **Frontend (`https://society.banesons.com`)**: Cloudflare Pages or Vercel.
- **Backend API (`https://api.society.banesons.com`)**: Docker container on AWS ECS Fargate or Fly.io co-located in **Mumbai (`ap-south-1`)**.
- **Database**: Supabase Cloud PostgreSQL in **AWS Mumbai (`ap-south-1`)**.

---

## Documentation

The `docs/` directory is the **authoritative single source of truth** for this project.
Refer to [docs/README.md](docs/README.md) for complete index of APIs, schemas, security threat models, and architectural decisions.
