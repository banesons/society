# CHS — Cooperative Housing Society Platform

Cooperative Housing Society (CHS) is a modern web platform engineered for housing society governance, member administration, gate operations, and financial accounting.

## Architecture

- **Backend**: Go with [Chi](https://github.com/go-chi/chi) HTTP router (Modular Monolith)
- **Frontend**: SvelteKit with Tailwind CSS, Rhea/shadcn-svelte component patterns, and TanStack Query
- **Database & Platform**: Supabase Cloud (PostgreSQL 17+ in `ap-south-1`)

## Repository Structure

```text
.
├── .agent/skills/       # Antigravity agent skills & workflows
├── docs/                # Project single source of truth (Obsidian-compatible)
│   ├── README.md        # Central documentation index
│   ├── eagle_view.md    # Master architecture & domain design
│   ├── sprints/         # S00–S18 sprint specifications
│   ├── architecture/    # Detailed system, backend, frontend, & Supabase architecture
│   ├── api/             # API conventions & specifications
│   └── decisions/       # Architecture Decision Records (ADRs)
├── backend/             # Go HTTP API service (/api/v1)
├── frontend/            # SvelteKit client application
├── supabase/            # Supabase config and SQL migrations
└── .github/workflows/   # CI verification workflows
```

## Quickstart

### Prerequisites
- Go 1.22+
- Node.js 20+ (Node 22 recommended)
- Supabase account with linked project

### Backend Setup
```bash
cd backend
cp .env.example .env
# Edit .env with your Supabase credentials
go run ./cmd/api
```
Backend will start on `http://localhost:8080`.

Verify health endpoint:
```bash
curl http://localhost:8080/api/v1/health
```

### Frontend Setup
```bash
cd frontend
cp .env.example .env
# Edit .env with PUBLIC_API_URL and PUBLIC_SUPABASE credentials
npm install
npm run dev
```
Frontend will start on `http://localhost:5173`.

## Documentation
The `docs/` folder is the authoritative source of truth. See [docs/README.md](file:///Users/shubham/Developer/chs_neu/docs/README.md) for details.
