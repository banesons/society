# 04 — Supabase & Database Architecture

## Overview

CHS relies on **Supabase Cloud** as its PostgreSQL database provider and operational platform.

## Configuration & Environments

- **Project Ref**: `glpjpqpbhdcmgmwlnkay`
- **Database Engine**: PostgreSQL 17.6+
- **Region**: `ap-south-1` (Mumbai)
- **Local Application Services**: Connect to Supabase Cloud via connection string or API keys specified in `backend/.env` and `frontend/.env`.

## Migration Strategy

Database schema is strictly version-controlled using standard migration files located in `supabase/migrations/`:
```text
supabase/
├── config.toml
└── migrations/
    └── 20260928000000_initial_schema.sql (when created)
```

### Rules for Schema Changes
1. **Migrations Are Authoritative**: The SQL files in `supabase/migrations/` represent the sole source of truth for the database schema.
2. **No Ad-Hoc Dashboard Edits**: Tables and constraints must not be created or altered directly in the Supabase Dashboard without a tracked migration file.
3. **No Business Tables in S00**: During S00, zero business tables (users, societies, units, etc.) are created.
4. **Supabase MCP Usage**: The Supabase MCP server is used by agents and developers to inspect live database status, query schemas, and verify migrations. It does not replace the migration pipeline.

## Connection Boundaries

- **Backend (Go)**: Connects to PostgreSQL directly using connection pooling (e.g. pgxpool) with standard credentials.
- **Frontend (SvelteKit)**: Uses the public Supabase client initialized only with the public `anon` key (`PUBLIC_SUPABASE_URL` and `PUBLIC_SUPABASE_ANON_KEY`). Frontend NEVER receives service-role keys or direct database passwords.

## Related Documentation
- [[01-system-overview|System Overview]]
- [[../decisions/005-supabase-mcp|ADR 005: Supabase MCP]]
- [[../decisions/006-development-environment|ADR 006: Development Topology]]
