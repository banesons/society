# 02 — Backend Architecture

## Design Overview

The CHS backend is a modular monolith written in Go. It prioritizes simplicity, mechanical sympathy, fast execution, and practical layering without gratuitous abstraction.

## Layering Model

The backend is structured into clear layers within `internal/`:

```text
HTTP Request
     │
     ▼
[Middleware Layer]
├── RequestID Middleware (extracts or generates X-Request-ID)
├── Structured Logging Middleware (records duration, method, path, status)
├── Recovery Middleware (gracefully handles panics)
└── CORS Middleware (controls cross-origin access)
     │
     ▼
[Handler Layer]
├── Parses and validates incoming HTTP requests
├── Binds payloads into typed Go structs
└── Formats standard JSON success/error responses
     │
     ▼
[Service Layer]
├── Orchestrates business use cases
├── Enforces domain invariants and business rules
└── Coordinates between repositories and external services
     │
     ▼
[Repository Layer]
├── Manages SQL queries against PostgreSQL
└── Maps database rows to domain models
     │
     ▼
Supabase PostgreSQL
```

## Layering Rules

1. **Handlers** deal only with HTTP concerns (status codes, headers, query parameters, serialization).
2. **Services** execute domain logic and are agnostic to HTTP transport.
3. **Repositories** execute SQL queries and interact with PostgreSQL.
4. **No Premature Abstraction**: We do not introduce interfaces where only a single concrete implementation exists, unless required for unit testing.
5. **No Speculative Domains**: During S00, domain packages such as `internal/society` or `internal/gate` are NOT created. They are added as their respective sprints begin.

## Core Middleware

- **Request ID**: Ensures every request context has a unique identifier, returned in the `X-Request-ID` response header.
- **Structured Logger**: Employs Go's standard `log/slog` library to emit structured JSON logs tagged with `request_id`, `method`, `path`, `status`, and `duration_ms`.
- **CORS**: Configured with explicit allowed origins, headers, and methods.

## Configuration Management

Configuration is loaded once during server bootstrap from environment variables or a local `.env` file via a centralized `internal/config` package:
- Environment variables are never queried ad-hoc inside business handlers.
- Secrets are kept out of source code.
- `.env` is ignored by Git; `.env.example` documents all required keys.

## Related Documentation
- [[01-system-overview|System Overview]]
- [[../api/01-api-overview|API Overview]]
- [[../decisions/002-api-contract|ADR 002: API Contract]]
