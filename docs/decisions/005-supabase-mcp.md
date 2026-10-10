# ADR 005: Supabase MCP Integration and Migration Authority

## Status
Accepted

## Context
Antigravity and human developers have access to a live Supabase MCP connection capable of executing SQL, querying schema, listing tables, and reading logs directly from Supabase Cloud. Clear boundaries must be established so that ad-hoc MCP operations do not bypass version-controlled database migrations.

## Decision
1. **Migration Authority**: The version-controlled files under `supabase/migrations/` in this repository remain the sole authoritative source of truth for the database schema.
2. **Role of Supabase MCP**:
   - Supabase MCP is approved for:
     - Real-time schema inspection and diagnostic validation.
     - Live database status checks and query log inspection.
     - Execution of migrations against Supabase Cloud during automated or controlled deployment steps.
   - Supabase MCP must **never** be used to create, modify, or drop tables or columns ad-hoc without a corresponding tracked migration file in `supabase/migrations/`.
3. **No Business Tables in S00**: Sprint S00 establishes migration infrastructure only; no business or domain tables are to be created in public schema until their respective sprints begin.

## Consequences
- Schema evolution remains reproducible, auditable in git history, and reversible.
- MCP serves as an observability and execution tool rather than an untracked backdoor.

## Related Documentation
- [[../architecture/04-supabase|Supabase Architecture]]
- [[006-development-environment|Development Environment]]
