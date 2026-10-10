# ADR 001: Go, SvelteKit, and Supabase Core Stack

## Status
Accepted

## Context
Cooperative Housing Society (CHS) management and Gate operations demand high reliability, strict data isolation, responsive real-time capabilities, and maintainable software architecture. We need a backend capable of robust business logic and high-throughput concurrency, a reactive and performant frontend, and a managed relational database platform.

## Decision
We select the following core technology stack for CHS:
1. **Backend**: **Go** using the **Chi** HTTP router. Go provides static typing, high concurrency performance, clean compilation to single binaries, and minimal memory overhead.
2. **Frontend**: **SvelteKit** with **Tailwind CSS**, styled via a lightweight Rhea / shadcn-svelte component approach, and **TanStack Query** for server-state synchronization. SvelteKit provides fast load times, concise reactive syntax, and an ergonomic developer experience.
3. **Database & Platform**: **Supabase Cloud (PostgreSQL)**. Supabase offers managed PostgreSQL, built-in connection pooling, database migrations, and future authentication/storage primitives without the burden of self-hosting database infrastructure in early sprints.

## Consequences
- The architecture is established as a modular monolith in Go with clean layering.
- The frontend and backend run as distinct local processes in development, both connecting to a centralized Supabase Cloud instance.
- Avoids premature microservices, Kubernetes, or complex event brokers while laying a solid foundation for growth.

## Related Documentation
- [[../architecture/01-system-overview|System Overview]]
- [[../architecture/02-backend|Backend Architecture]]
- [[../architecture/03-frontend|Frontend Architecture]]
- [[../architecture/04-supabase|Supabase Architecture]]
