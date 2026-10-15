# ADR 031: Production Domain Configuration and Deployment Topology (`society.banesons.com`)

## Status
Accepted

## Context
Cooperative Housing Society (CHS) Neu is graduating from development environments (`localhost:5173`, `localhost:8080`) to its designated production domain:

> **Production Application Host**: `https://society.banesons.com`

End users (Platform Administrators, Society Administrators, Committee Members, Residents, and Gatekeepers) will navigate directly to `society.banesons.com` for authentication and daily society operations.

The production infrastructure must satisfy:
1. **Low-Latency Co-location**: CHS database is hosted in Supabase Cloud AWS Mumbai (`ap-south-1`). To ensure sub-second response times for gate check-ins and billing queries, backend compute must reside within the same geographic region.
2. **Edge Performance & Caching**: SvelteKit static assets, fonts, icons, and bundles must be cached across Indian edge points-of-presence (Mumbai, Delhi, Chennai, Bengaluru) for rapid loading on mobile networks.
3. **CORS & Origin Security**: Production API requests originating from `https://society.banesons.com` must be authorized with credentials, while rejecting unauthorized third-party origins.
4. **SSL & Automated Certificate Lifecycle**: Production HTTPS must be strictly enforced with HTTP Strict Transport Security (HSTS).

---

## Decision

### 1. Domain Architecture & Ingress Routing
The platform adopts `banesons.com` as the root organization domain and `society.banesons.com` as the primary application endpoint:

| Role | Hostname / Path | Service | Ingress Provider |
| :--- | :--- | :--- | :--- |
| **Frontend Web App** | `https://society.banesons.com` | SvelteKit 2 SPA / SSR | Cloudflare Pages / Vercel |
| **Backend REST API (Subdomain)** | `https://api.society.banesons.com/api/v1` | Go 1.24 Chi API Service | Container Runtime (AWS / Fly.io / DO) |
| **Backend REST API (Unified Path)** | `https://society.banesons.com/api/v1` | Reverse Proxy Rule to API Origin | Cloudflare Origin Rule (Optional) |

### 2. Deployment Recommendations by Tier

#### Tier 1: Frontend (`society.banesons.com`)
- **Recommended Platform**: **Cloudflare Pages** (or **Vercel**).
- **Rationale**:
  - The SvelteKit repository uses `@sveltejs/adapter-auto`, which natively detects Cloudflare Pages and Vercel without requiring custom adapter refactoring.
  - Cloudflare provides zero-egress edge caching, built-in DDoS mitigation, automated TLS certificates for `society.banesons.com`, and Indian edge locations.
- **Environment Variables**:
  - `PUBLIC_API_URL`: `https://api.society.banesons.com/api/v1` (or `/api/v1` if unified via Cloudflare reverse proxy).
  - `PUBLIC_SUPABASE_URL`: `https://glpjpqpbhdcmgmwlnkay.supabase.co`
  - `PUBLIC_SUPABASE_ANON_KEY`: Client-safe anonymous public key.
  - `PUBLIC_FIREBASE_*`: Web push and performance monitoring client keys.

#### Tier 2: Backend REST API (`api.society.banesons.com`)
- **Recommended Platform**: **AWS ECS Fargate** or **DigitalOcean App Platform / Droplet** or **Fly.io** deployed in **Mumbai (`ap-south-1` / `bom`)**.
- **Rationale**:
  - The Go backend compiles into a single, highly efficient, standalone static binary requiring only ~20–30 MB RAM per instance.
  - Deploying compute in AWS Mumbai (`ap-south-1`) places the Go binary adjacent to the Supabase PostgreSQL cluster (`aws-0-ap-south-1.pooler.supabase.com`), achieving sub-2ms database connection latency.
- **Environment Variables**:
  - `PORT`: `8080` (or injected by runtime).
  - `ENV`: `production`.
  - `ALLOWED_ORIGINS`: `https://society.banesons.com` (plus local dev ports if running staging).
  - `DATABASE_URL`: `postgresql://postgres.[ref]:[pass]@aws-0-ap-south-1.pooler.supabase.com:6543/postgres`
  - `JWT_PRIVATE_KEY`: Base64 Ed25519 64-byte private key.
  - `FIREBASE_CREDENTIALS_FILE`: Absolute path to decrypted service-account JSON in production secrets store.
  - `MSG91_AUTH_KEY`, `MSG91_TEMPLATE_ID`, `MSG91_SENDER_ID`: Production SMS gateway credentials.

#### Tier 3: Database & Object Storage
- **Database**: Supabase PostgreSQL (`glpjpqpbhdcmgmwlnkay`) in `ap-south-1` (Mumbai).
- **Document Storage**: Supabase Storage bucket (`society-documents`) or Cloudflare R2 / AWS S3 (`ap-south-1`) for society documents and invoices.

---

### 3. Cross-Origin Resource Sharing (CORS) Configuration
The Go backend defaults to and enforces:
```go
ALLOWED_ORIGINS="http://localhost:5173,http://localhost:4173,https://society.banesons.com"
```
Headers permitted:
- `Accept`, `Authorization`, `Content-Type`, `X-Request-ID`, `X-Society-ID`.
Exposed headers:
- `Link`, `X-Request-ID`, `X-Society-ID`.
`AllowCredentials: true` is configured to support authenticated sessions.

---

## Consequences

### Positive
- Unified branding and single memorable URL (`society.banesons.com`) for all resident and administrative interactions.
- Geographic co-location of Go compute with database cluster in Mumbai guarantees lowest possible p95 latency across India.
- Zero-cost, high-speed frontend delivery via Cloudflare/Vercel edge network with automatic SSL.
- Clear separation of concerns between static frontend hosting and containerized backend business logic.

### Negative
- Requires configuring DNS records (CNAME for `society.banesons.com` and A/CNAME for `api.society.banesons.com`) under the `banesons.com` registrar.
- Backend secrets (`JWT_PRIVATE_KEY`, Firebase service account, MSG91 auth key) must be securely provisioned into the container runtime.

---

## Related Documentation
- [[../architecture/01-system-overview|System Overview]]
- [[../architecture/21-production-deployment-and-domain-topology|Production Deployment & Domain Topology]]
- [[001-go-sveltekit-supabase|ADR 001: Go, SvelteKit, and Supabase]]
- [[006-development-environment|ADR 006: Development Topology]]
