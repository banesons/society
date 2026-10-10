# CHS — Eagle View

```text
                              CHS PLATFORM
                                   │
                 ┌─────────────────┴─────────────────┐
                 │                                   │
          PLATFORM MANAGEMENT                  SOCIETY ECOSYSTEM
                 │                                   │
          ┌──────┴──────┐              ┌────────────┴────────────┐
          │             │              │                         │
      SUPER_ADMIN     ADMIN         SOCIETY                    GATE
          │             │              │                         │
          │       Map to 1..N          │                    Gate Operations
          │         Societies          │
          │                            │
          └──────────────┐             │
                         │             │
                    SOC_ADMIN ─────────┘
                         │
              ┌──────────┼───────────┐
              │          │           │
           Members     Units      Society Roles
              │                      │
       ┌──────┴──────┐        ┌──────┴────────┐
       │             │        │               │
     Owners        Tenants  Secretary      Treasurer
       │             │        │               │
   ┌───┼───┐     ┌───┼───┐    │               │
   │   │   │     │   │   │    └──── etc. ─────┘
   │   │   │     │   │   │
Primary Secondary Family  Primary Secondary Family
 Owner   Owner            Tenant  Tenant
```

---

## 1. The Fundamental Hierarchy

The system starts with **People**, **Organizations/Societies**, and **Units**.

### Person Relationships

```text
Person
  │
  ├── Platform relationship
  │      ├── Super Admin
  │      └── Admin
  │
  └── Society relationship
         ├── Soc Admin
         ├── Society Designation
         ├── Owner
         ├── Tenant
         └── Gatekeeper
```

### Society Structure

```text
Society
   │
   └── Units / Flats
          │
          ├── Owners
          │     ├── Primary
          │     ├── Secondary
          │     └── Family
          │
          └── Tenants
                ├── Primary
                ├── Secondary
                └── Family
```

---

## 2. Society Onboarding

This is the first feature we are building. There are two entry paths:

### Path A: Platform-Created

```text
SUPER_ADMIN
     │
     ▼
Create Admin
     │
     ▼
Assign Admin → Society
```

> **Note:** An `ADMIN` can manage multiple societies.

### Path B: User-Created

```text
END USER
   │
   ▼
Create Society
   │
   ▼
PENDING
   │
   ▼
Platform Review
   │
   ├── APPROVED ──► ACTIVE
   │
   └── REJECTED
```

Once approved, the appropriate society administration can be established.

---

## 3. Identity & Authentication

Every person has **one** platform identity.

### Authentication Flow

```text
                    USER
                     │
              Mobile / Identity
                     │
                     ▼
                   Login
                     │
                  OTP / Auth
                     │
                     ▼
                    JWT
                     │
                 Session ID
                     │
                     ▼
              Authenticated API
```

### API Baseline Headers

Client sends:
```http
Authorization: Bearer <JWT>
X-Session-ID: <session-id>
```

Server-side context captured on every request:
- **IP**
- **User-Agent**
- **Request ID**

---

## 4. Authorization Model

We are deliberately **not** treating every designation as a global RBAC role.

### Relationship Model

```text
                    USER
                      │
        ┌─────────────┴─────────────┐
        │                           │
 Platform Access              Society Relationship
        │                           │
 ┌──────┴──────┐          ┌─────────┼─────────┐
 │             │          │         │         │
SUPER_ADMIN   ADMIN    SOC_ADMIN  MEMBER   GATEKEEPER
                                      │
                             ┌────────┴────────┐
                             │                 │
                           OWNER             TENANT
                             │                 │
                         Unit-based        Unit-based
```

### Society Designations

Society designations sit alongside the relationship rather than replacing it:

```text
Society Member
      │
      └── Designation
           ├── Secretary
           ├── Treasurer
           ├── Chairman
           └── ...
```

### Multi-Society Example

A single person can hold different roles across different societies simultaneously:

```text
Person A
 ├── Society A
 │     ├── SOC_ADMIN
 │     └── Secretary
 │
 └── Society B
       └── PRIMARY_OWNER
             └── Unit B-402
```

This flexibility is intentional.

---

## 5. Society Itself

At the highest level, a society encompasses:

```text
SOCIETY
│
├── Basic Information
├── Buildings / Wings
├── Units / Flats
├── Members
├── Owners
├── Tenants
├── Family Members
├── Society Administration
├── Committee / Designations
├── Gatekeepers
├── Maintenance
├── Charges
├── Notices
└── Gate
```

> [!NOTE]
> We are **not** building all of these now. Our initial scope is the identity and onboarding foundation that allows these modules to exist later.

---

## 6. API Architecture

The API is organized strictly around **domains**, not random endpoints.

```text
                    Go API
                      │
       ┌──────────────┼──────────────┐
       │              │              │
     Auth          Platform        Society
       │              │              │
       │          Super Admin        │
       │          Admin              │
       │                             │
       └──────────────┬──────────────┘
                      │
                 Future Modules
                      │
              ┌───────┴────────┐
              │                │
             Gate            Restro
```

> **Scope Note:** `Restro` is explicitly out of scope for now.

---

## 7. API Security & Data Philosophy

### Request Pipeline

```text
Request ──► HTTPS ──► JWT ──► Session ──► Authorization ──► Validation ──► Business Logic ──► Database
```

### Response Pipeline

```text
Database ──► Only Required Data ──► PII Masking ──► Pagination ──► Encrypted API Envelope ──► SvelteKit
```

### Hard Rules

> [!IMPORTANT]
> **Never send the entire society dataset to the browser.**
> - A **members** screen gets members data only.
> - A **maintenance** screen gets maintenance data only.
> - A **unit** screen gets unit data only.
> - All large datasets **must** be paginated.

---

## 8. PII Policy

The database retains the actual values. The API exposes only what the particular authorized operation requires, masked by default.

### Default API Presentation

| Field | Raw Value | Masked Output |
| :--- | :--- | :--- |
| **Mobile** | `9967089101` | `9967xxxx1` |
| **PAN** | `EHBH123456` | `EHBHxxxx6` |
| **Email** | `shubham@gmail.com` | `shub****m@gmail.com` |
| **Name** | `Shubham` | `Shub****m` |

---

## 9. Technology Eagle View

Keeping the architecture intentionally simple and boring:

```text
                       CHS
                        │
             ┌──────────┴──────────┐
             │                     │
         SvelteKit                Go
          Frontend              Backend
             │                     │
             └──────────┬──────────┘
                        │
                    PostgreSQL
                        │
                    Supabase
                        │
          ┌─────────────┼─────────────┐
          │             │             │
         Auth         Storage       Realtime
```

### What We Are NOT Doing For Now

- ❌ No `bane_core`
- ❌ No microservices
- ❌ No Kubernetes
- ❌ No Restro
- ❌ No elaborate event architecture
- ❌ No premature abstractions

---

## 10. What We’re Actually Building First

### Phase 1: Identity + Society Onboarding

```text
1. Super Admin
      ↓
2. Authentication
      ↓
3. Admin creation
      ↓
4. Society creation
      ↓
5. Society PENDING state
      ↓
6. Society approval
      ↓
7. Society activation
      ↓
8. Admin ↔ Society mapping
      ↓
9. SOC_ADMIN establishment
      ↓
10. Basic role/designation foundation
```

### Future Phases

| Phase | Module | Scope |
| :--- | :--- | :--- |
| **Phase 2** | **Society Structure** | Buildings / Wings, Units, Members |
| **Phase 3** | **Ownership & Tenancy** | Primary, Secondary, Family |
| **Phase 4** | **Society Administration** | Secretary, Treasurer, and other designations |
| **Phase 5** | **Gate** | Gatekeeper + gate operations |
| **Phase 6** | **Maintenance** | Charges, Collections, Statements |

> `Restro` stays parked indefinitely.

---

## The Single Most Important Principle

> ### **One identity → many relationships → scoped access.**
>
> *(Not: One user = one role)*

This principle gives us the flexibility needed for a real society platform without turning the initial system into an unmanageable RBAC monster.
