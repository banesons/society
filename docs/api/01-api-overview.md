# 01 — API Overview

## Base URL & Versioning

All CHS backend APIs are versioned under the `/api/v1` prefix.
- Local Base URL: `http://localhost:8080/api/v1`

## Standard Request Headers

| Header | Required | Description |
| :--- | :--- | :--- |
| `Content-Type` | Optional (default: `application/json`) | Payload format |
| `X-Request-ID` | Optional | Client correlation ID. If omitted, the server generates one. |

## Standard Response Envelopes

### Success Envelope (`2xx`)

```json
{
  "success": true,
  "data": {},
  "meta": null
}
```

- `data`: Contains the response payload.
- `meta`: Optional pagination or operation metadata.

### Error Envelope (`4xx`, `5xx`)

```json
{
  "success": false,
  "error": {
    "code": "BAD_REQUEST",
    "message": "Human-readable explanation of error"
  },
  "request_id": "01HX6M2..."
}
```

## Standard Status & Error Codes

| HTTP Status | Error Code | Description |
| :--- | :--- | :--- |
| `400` | `BAD_REQUEST` | Malformed body, invalid query parameter, or validation error |
| `404` | `NOT_FOUND` | Resource does not exist |
| `405` | `METHOD_NOT_ALLOWED` | HTTP method not permitted for this endpoint |
| `500` | `INTERNAL_SERVER_ERROR` | Unhandled server error (generic message sent to client) |

## Baseline Endpoints (S00)

### 1. Health Check

Verifies backend liveness and database connectivity status without exposing sensitive internal secrets.

- **Method**: `GET`
- **Path**: `/api/v1/health`
- **Authentication**: None (Public)
- **Response Format**:
  ```json
  {
    "success": true,
    "data": {
      "status": "healthy",
      "version": "1.0.0",
      "database": {
        "connected": true,
        "latency_ms": 12
      }
    },
    "meta": null
  }
  ```

## Related Documentation
- [[../architecture/02-backend|Backend Architecture]]
- [[../decisions/002-api-contract|ADR 002: Unified API Contract]]
- [[../decisions/003-http-methods|ADR 003: Semantic HTTP Methods]]
