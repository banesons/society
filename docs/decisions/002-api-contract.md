# ADR 002: Unified API Response Contract and Request Tracing

## Status
Accepted

## Context
A fragmented API response format across endpoints leads to fragile client-side handling, defensive boilerplate, and poor error visibility. Furthermore, diagnosing issues across distributed client-server interactions requires reliable end-to-end request tracing.

## Decision
1. **API Versioning**: All public and operational endpoints must be prefixed with `/api/v1`.
2. **Unified Response Envelope**:
   - **Success Response (`2xx`)**:
     ```json
     {
       "success": true,
       "data": { ... },
       "meta": null
     }
     ```
     `data` contains the primary payload (object or array). `meta` provides optional metadata such as pagination `{ "page": 1, "per_page": 20, "total": 150 }`.
   - **Error Response (`4xx`, `5xx`)**:
     ```json
     {
       "success": false,
       "error": {
         "code": "VALIDATION_ERROR",
         "message": "Human-readable description of error"
       },
       "request_id": "01HX6M2... or UUID"
     }
     ```
3. **Request Tracing via `X-Request-ID`**:
   - Inbound requests with a valid `X-Request-ID` header retain that ID in request context.
   - If `X-Request-ID` is missing or malformed, the backend generates a cryptographically random or ULID/UUID request identifier.
   - The final `X-Request-ID` is written to all HTTP response headers and included in error envelopes and structured server logs.

## Consequences
- The frontend central API client can reliably inspect `response.data.success` and provide uniform error feedback.
- Developers can correlate client-side errors directly to backend logs using the request ID.

## Related Documentation
- [[../api/01-api-overview|API Overview]]
- [[../architecture/02-backend|Backend Architecture]]
