# 21 — Production Deployment and Domain Topology (`society.banesons.com`)

## Purpose
Specifies the production topology, domain structure, hosting targets, and environmental configuration for launching Cooperative Housing Society (CHS) Neu on its official production domain:

> **Production Application Endpoint**: `https://society.banesons.com`

---

## 1. System Topology Overview

```mermaid
graph TD
    User["End User / Browser / PWA\n(Resident / Admin / Gatekeeper)"]
    
    subgraph Edge["Global & Regional Edge (banesons.com DNS)"]
        CF["Cloudflare DNS / Edge Network\n(SSL Termination, DDoS, Edge Caching)"]
    end

    subgraph FrontendHosting["Frontend Tier (society.banesons.com)"]
        Pages["Cloudflare Pages / Vercel\n(SvelteKit 2 SPA / SSR)\nGlobal Edge (Mumbai, Delhi, BLR PoPs)"]
    end

    subgraph BackendCompute["Backend Compute Tier (api.society.banesons.com)"]
        GoAPI["Go 1.24 Chi REST API Container\n(Docker Scratch / Distroless)\nRegion: AWS Mumbai (ap-south-1)"]
    end

    subgraph DataServices["Data & External Services Tier"]
        DB[("Supabase Cloud PostgreSQL\nRegion: AWS Mumbai (ap-south-1)\naws-0-ap-south-1.pooler.supabase.com")]
        Storage["Supabase Storage / Cloudflare R2\n(Encrypted Society Documents & Invoices)"]
        FCM["Firebase Cloud Messaging (FCM)\n(Web Push Notifications)"]
        SMS["MSG91 SMS Gateway\n(Transactional OTP via Indian DLT Routes)"]
    end

    User -->|"HTTPS (society.banesons.com)"| CF
    CF -->|"Route /*"| Pages
    Pages -->|"REST API Requests (/api/v1/*)"| GoAPI
    CF -.->|"Optional Cloudflare Route /api/*"| GoAPI
    GoAPI -->|"Internal pgxpool (<2ms latency)"| DB
    GoAPI -->|"Document Binary Uploads"| Storage
    GoAPI -->|"Push Dispatch"| FCM
    GoAPI -->|"OTP Dispatch"| SMS
```

---

## 2. Where to Deploy What: Recommendation Matrix

| Tier | Component | Recommended Deployment Platform | Region / Location | Rationale & Tradeoffs |
| :--- | :--- | :--- | :--- | :--- |
| **Frontend** | SvelteKit 2 Web App | **Cloudflare Pages** (Alternative: **Vercel**) | Global Edge (Indian PoPs) | SvelteKit's `@sveltejs/adapter-auto` builds seamlessly for Cloudflare/Vercel. Provides instant SSL for `society.banesons.com`, zero-egress asset caching, and lightning-fast TTFB across Indian 4G/5G networks. |
| **Backend API** | Go Chi REST Service | **AWS ECS Fargate** or **DigitalOcean App Platform** or **Fly.io** | **Mumbai (`ap-south-1` / `bom`)** | **Crucial co-location**: Supabase PostgreSQL is in AWS Mumbai (`ap-south-1`). Hosting the Go container in Mumbai provides <2ms database latency. Go binary uses <30MB RAM and starts in milliseconds. |
| **Database** | PostgreSQL 15 | **Supabase Cloud** | **AWS Mumbai (`ap-south-1`)** | Pre-configured connection pooler (PgBouncer port 6543), built-in RLS, automated backups, and 21 applied database migrations. |
| **Object Storage** | Document Storage | **Supabase Storage** (or **Cloudflare R2**) | **AWS Mumbai / APAC** | Houses PDF invoices, visitor badge photos, society registration certificates, and AGM minutes with strict access control. |
| **Push Gateway** | Web Push / PWA | **Firebase Cloud Messaging (FCM)** | Google Global Infrastructure | Standardized browser push delivery for Chrome, Edge, Safari, and Android PWA. |
| **SMS Gateway** | Auth OTPs | **MSG91** | India Telecom Routes | TRAI/DLT-compliant transactional SMS delivery for Indian mobile numbers (`+91`). |

---

## 3. Co-location & Latency Principle

For gate security and resident interactions, latency predictability is non-negotiable:
- Security guards processing visitor arrivals cannot tolerate 200ms transatlantic database query round-trips.
- **The Golden Rule**: **Always deploy the Go API backend in the same cloud region as the database (`ap-south-1` Mumbai)**.
  - Go Backend (`ap-south-1`) ↔ Supabase Cloud (`ap-south-1`): **~1–3 ms** round-trip.
  - If backend were deployed in US-East (`us-east-1`), each API request making 3 SQL queries would suffer **~600ms** latency penalty.

---

## 4. Ingress Routing & Domain Layout

Two production routing models are supported:

### Option A: Subdomain Ingress (Default)
- **Frontend App**: `https://society.banesons.com`
- **Backend API**: `https://api.society.banesons.com/api/v1`
- **CORS Handling**: Backend `ALLOWED_ORIGINS` explicitly includes `https://society.banesons.com` with `AllowCredentials: true`.

### Option B: Unified Same-Origin Ingress (Cloudflare Reverse Proxy)
- **Frontend App**: `https://society.banesons.com/*`
- **Backend API**: `https://society.banesons.com/api/*`
- **Cloudflare Origin Rule / Worker**: Transparently routes `/api/*` to the Go backend container.
- **Advantage**: Zero CORS preflight (`OPTIONS`) round-trips, eliminating network overhead on mobile devices.

---

## 5. Production Environment Variables

### Frontend Configuration (`frontend/.env` / Cloudflare Pages Environment)

| Variable | Production Value | Sensitivity | Description |
| :--- | :--- | :--- | :--- |
| `PUBLIC_API_URL` | `https://api.society.banesons.com/api/v1` | Public | Base URL for REST API calls |
| `PUBLIC_SUPABASE_URL` | `https://glpjpqpbhdcmgmwlnkay.supabase.co` | Public | Supabase project endpoint |
| `PUBLIC_SUPABASE_ANON_KEY` | `eyJhbGciOi...` | Public | Public anonymous key |
| `PUBLIC_FIREBASE_API_KEY` | `AIzaSy...` | Public | Firebase Web API key |
| `PUBLIC_FIREBASE_AUTH_DOMAIN` | `chsneu-2021.firebaseapp.com` | Public | Firebase Auth domain |
| `PUBLIC_FIREBASE_PROJECT_ID` | `chsneu-2021` | Public | Firebase Project identifier |
| `PUBLIC_FIREBASE_STORAGE_BUCKET`| `chsneu-2021.firebasestorage.app` | Public | Firebase storage bucket |
| `PUBLIC_FIREBASE_MESSAGING_SENDER_ID` | `799965614182` | Public | Firebase sender ID |
| `PUBLIC_FIREBASE_APP_ID` | `1:799965614182:web:...` | Public | Firebase Web App ID |
| `PUBLIC_FIREBASE_MEASUREMENT_ID` | `G-0QKMNF14J1` | Public | Google Analytics / RUM ID |
| `PUBLIC_FIREBASE_WEB_PUSH_VAPID_PUBLIC_KEY` | `<VAPID_KEY>` | Public | Web Push VAPID public key |

### Backend Configuration (`backend/.env` / Production Secrets Manager)

| Variable | Production Value | Sensitivity | Description |
| :--- | :--- | :--- | :--- |
| `PORT` | `8080` | Non-sensitive | Container listen port |
| `ENV` | `production` | Non-sensitive | Enables production validation & disables mock bypasses |
| `ALLOWED_ORIGINS` | `https://society.banesons.com` | Non-sensitive | Authorized frontend CORS origins |
| `DATABASE_URL` | `postgresql://postgres.[ref]:[pass]@aws-0-ap-south-1.pooler.supabase.com:6543/postgres` | **Secret** | Supabase connection string (pooler) |
| `JWT_PRIVATE_KEY` | `<Base64_Ed25519_Key>` | **Critical Secret**| 64-byte Ed25519 seed for persistent token signing |
| `SUPER_ADMIN_PHONE` | `919967089101` | Non-sensitive | Root platform administrator phone |
| `RESEND_API_KEY` | `re_...` | **Secret** | Resend email API token |
| `RESEND_FROM_EMAIL` | `notifications@society.banesons.com` | Non-sensitive | Verified sender address |
| `FIREBASE_PROJECT_ID`| `chsneu-2021` | Non-sensitive | Firebase project ID |
| `FIREBASE_CREDENTIALS_FILE` | `/etc/secrets/firebase-sa.json` | **Secret** | Path to decrypted service account JSON |
| `SMS_PROVIDER` | `msg91` | Non-sensitive | Production SMS provider |
| `MSG91_AUTH_KEY` | `<MSG91_KEY>` | **Secret** | MSG91 API authentication key |
| `MSG91_TEMPLATE_ID` | `<TEMPLATE_ID>` | Non-sensitive | Approved DLT OTP template ID |
| `MSG91_SENDER_ID` | `CHSNEU` | Non-sensitive | Registered 6-character sender ID |

---

## 6. Production Step-by-Step Deployment Runbook

### Step 1: DNS Configuration (`banesons.com`)
In your DNS provider (Cloudflare recommended):
1. **Frontend**: Add `CNAME` record:
   - Name: `society`
   - Target: `<project-name>.pages.dev` (or `cname.vercel-dns.com`)
   - Proxy status: `Proxied` (Orange cloud)
2. **Backend API**: Add `CNAME` or `A` record:
   - Name: `api.society`
   - Target: Backend load balancer / container domain (e.g. `chs-backend.fly.dev` or AWS ALB)
   - Proxy status: `Proxied` or `DNS only` depending on setup.

### Step 2: Deploy Frontend to Cloudflare Pages
1. Connect Git repository (`chs_neu`) in Cloudflare Pages dashboard.
2. Build settings:
   - Framework preset: `SvelteKit`
   - Build command: `npm run build`
   - Build output directory: `.svelte-kit/cloudflare` (or standard automatic output)
   - Root directory: `/frontend`
3. Add environment variables listed in Section 5.
4. Assign Custom Domain: `society.banesons.com`. Cloudflare automatically provisions an SSL/TLS edge certificate.

### Step 3: Deploy Backend Container to AWS / Fly.io / DO (Mumbai)
1. Build production container using standard multi-stage Dockerfile:
   ```dockerfile
   FROM golang:1.24-alpine AS builder
   WORKDIR /app
   COPY go.mod go.sum ./
   RUN go mod download
   COPY . .
   RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /chs-api ./cmd/api

   FROM gcr.io/distroless/static:nonroot
   WORKDIR /
   COPY --from=builder /chs-api /chs-api
   USER nonroot:nonroot
   EXPOSE 8080
   ENTRYPOINT ["/chs-api"]
   ```
2. Deploy to AWS ECS (region `ap-south-1`) or Fly.io (`fly deploy --region bom`) or DigitalOcean App Platform (`blr1`).
3. Inject production environment variables and secrets via the hosting platform's secrets manager.
4. Mount health check probe: `GET /health` (expects HTTP 200).

### Step 4: Verification & Smoke Testing
1. Verify API health probe:
   ```bash
   curl -i https://api.society.banesons.com/health
   ```
   Must return: `{"success":true,"data":{"status":"healthy","database":{"connected":true}}}`
2. Verify CORS headers:
   ```bash
   curl -i -X OPTIONS https://api.society.banesons.com/api/v1/platform/societies \
     -H "Origin: https://society.banesons.com" \
     -H "Access-Control-Request-Method: GET"
   ```
   Must return: `Access-Control-Allow-Origin: https://society.banesons.com`
3. Verify End-to-End Browser Login:
   - Navigate to `https://society.banesons.com/login` in browser.
   - Enter mobile number, receive OTP via MSG91, submit, and confirm authenticated landing on dashboard.

---

## Related Documentation
- [[01-system-overview|System Overview]]
- [[02-backend|Backend Architecture]]
- [[03-frontend|Frontend Architecture]]
- [[../decisions/031-production-domain-and-deployment-topology|ADR 031: Production Domain Configuration]]
