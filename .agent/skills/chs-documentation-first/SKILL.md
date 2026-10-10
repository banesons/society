---
name: chs-documentation-first
description: >-
  Enforces documentation-first development and synchronization between /docs
  and implementation for the CHS project. Use whenever planning, implementing,
  refactoring, or reviewing domain models, APIs, roles, permissions, security,
  or architectural workflows in CHS.
---

# CHS Documentation-First Development

## Purpose

This skill governs development work for the CHS project.

The `/docs` directory at the repository root is the source of truth for product requirements, domain behavior, architecture, API contracts, security rules, workflows, decisions, and development conventions.

Documentation is a first-class project artifact.

The agent MUST keep documentation and implementation synchronized.

---

## Core Principle

Always follow:

**Read → Understand → Plan → Implement → Validate → Document → Verify**

Never:

**Implement → Document later**

---

## 1. Before Any Implementation

Before modifying code, the agent MUST:

1. Inspect the repository structure.
2. Inspect `/docs`.
3. Identify documentation relevant to the requested task.
4. Read all relevant documents.
5. Follow internal documentation links where relevant.
6. Search for related concepts, entities, APIs, roles, workflows, and decisions.
7. Determine whether the requested behavior already exists.
8. Determine whether documentation needs to change.

No production implementation should begin before this review.

---

## 2. Documentation Is Authoritative

Treat `/docs` as the project’s intended behavior.

Do not assume existing code is correct merely because it exists.

If documentation and implementation disagree:

1. Identify the discrepancy.
2. Determine whether the requested task changes the documented behavior.
3. Update the documentation when the intended behavior changes.
4. Then update the implementation.

Never silently introduce behavior that contradicts documented decisions.

---

## 3. Use Repository Exploration

When locating existing implementation, use available repository exploration capabilities efficiently.

If repository exploration tools are available, use them to locate:

- Files
- Symbols
- API handlers
- Services
- Database models
- Existing workflows
- Authorization checks
- Related tests

Do not read the entire repository unnecessarily. Use targeted exploration first, then open the relevant files.

Documentation review is still mandatory even when repository exploration is used.

---

## 4. Change Impact Analysis

Before implementation, determine which areas may be affected:

- Product requirements
- Domain model
- User roles
- Permissions
- Authentication
- Sessions
- API contracts
- Database schema
- Business workflows
- Security
- PII handling
- Encryption
- Tests
- Architecture

If a change affects any of these, update the corresponding documentation.

---

## 5. Requirements That Are Not Documented

If the requested behavior is not sufficiently defined:

Do not invent important product behavior. Instead:

1. Identify the missing decision.
2. Document the proposed behavior if it is clear from the task.
3. If the decision materially affects product behavior, permissions, security, or data modeling, ask for clarification before implementation.

Minor implementation details may be decided by the agent when they do not alter documented product behavior.

---

## 6. Documentation Before New Concepts

When introducing a meaningful new concept, document it before or alongside implementation.

Examples:
- New role
- New permission
- New entity
- New society state
- New API behavior
- New authentication behavior
- New security mechanism
- New workflow
- New architectural pattern

The documentation should describe the intended behavior, not merely describe the code.

---

## 7. Documentation After Implementation

After implementation, the agent MUST check whether documentation needs to be updated.

Update documentation when implementation changes:

- API methods or paths
- Request/response contracts
- Domain entities
- Relationships
- Database behavior
- Roles
- Permissions
- Authentication
- Sessions
- Security
- PII handling
- Encryption
- Business rules
- Workflow / state transitions
- User-visible behavior
- Architecture

Documentation must reflect the final implemented behavior.

---

## 8. API Documentation

Document every meaningful public API. Include, where applicable:

- Method
- Path
- Authentication
- Required headers
- Authorization
- Request payload
- Response payload
- Validation rules
- Errors
- Pagination
- PII behavior
- Security considerations

CHS API authentication currently follows:
```http
Authorization: Bearer <JWT>
X-Session-ID: <session-id>
```

Server-derived request information includes:
- IP address
- User-Agent
- Request ID

Do not require clients to provide server-derived information such as their own IP address.

---

## 9. API Data Minimization

Never send the complete Society dataset to the browser.

APIs should return only the data required for the current operation or screen.

Prefer modular, scoped APIs:
```text
Society
 ├── Overview API
 ├── Members API
 ├── Units API
 ├── Maintenance API
 ├── Charges API
 └── Gate API
```
rather than:
```text
Society API
 └── entire society dataset
```

Use pagination, filtering, scoped queries, and minimal response models.

---

## 10. Encrypted API Envelope

CHS may use an encrypted API envelope for API payloads:
```json
{
  "payload": "<encrypted-data>"
}
```

The encrypted envelope is an additional confidentiality layer. It does NOT replace:
- HTTPS
- Authentication
- Authorization
- Input validation
- Session validation
- Data minimization

Never invent custom cryptography. Any cryptographic implementation must use established primitives and document:
- Algorithm
- Key handling
- Nonce/IV handling
- Authentication/integrity
- Key rotation
- Replay considerations

---

## 11. PII Handling

CHS follows a minimum-disclosure principle.

The database may contain the complete legitimate value. API responses should expose only the information required for the operation.

Default display masking examples:
- Mobile: `9967089101` → `9967xxxx1`
- PAN: `EHBH123456` → `EHBHxxxx6`
- Email: `shubham@gmail.com` → `shub****m@gmail.com`
- Name: `Shubham` → `Shub****m`

Do not expose full sensitive identifiers merely because the backend has access to them. Full PII may only be returned when explicitly required and authorized by the workflow.

---

## 12. Society Data

The Society is a core CHS domain. Current high-level relationships include:

```text
Society
 ├── Units
 ├── Members
 ├── Owners
 ├── Tenants
 ├── Family members
 ├── Society administration
 ├── Society designations
 └── Gatekeepers
```

Do not introduce additional roles or relationships without documenting them.

---

## 13. Role Model

Do not treat every Society designation as a global platform role.

CHS separates:

**Platform access**:
- `SUPER_ADMIN`
- `ADMIN`

**Society-level access**:
- `SOC_ADMIN`
- `GATEKEEPER`

**Society designations**:
- Secretary, Treasurer, Chairman, etc.

**Unit relationships**:
- `PRIMARY_OWNER`
- `SECONDARY_OWNER`
- `OWNER_FAMILY`
- `PRIMARY_TENANT`
- `SECONDARY_TENANT`
- `TENANT_FAMILY`

A user may have multiple relationships across societies and units. Do not assume one user = one role.

---

## 14. Society Onboarding

Current Society onboarding workflow:

```text
End User
   ↓
Create Society
   ↓
PENDING
   ↓
Platform Review
   ├── APPROVED → ACTIVE
   └── REJECTED
```

Platform administration:
- `SUPER_ADMIN` → creates `ADMIN`
- `ADMIN` can be mapped to one or more societies

Society-level administration is represented by:
- `SOC_ADMIN`

Do not expand this workflow without updating the relevant documentation.

---

## 15. Obsidian Documentation

All CHS documentation must remain usable as an Obsidian knowledge base:

- Use Markdown.
- Prefer Obsidian wikilinks:
  - `[[../identity/02-roles-and-permissions]]`
  - `[[../domain/02-society]]`
  - `[[../api/03-society]]`
- Meaningful documents must link to related documents.
- Do not create isolated documentation files.

---

## 16. Documentation Index

The root documentation entry point is `/docs/README.md`.

New major documentation must be linked from the appropriate index. Maintain a navigable hierarchy:

```text
docs/
├── README.md
├── product/
├── identity/
├── domain/
├── api/
├── architecture/
└── decisions/
```

Create documents as the project becomes defined; do not create empty placeholder files.

---

## 17. Architecture Decisions

Record meaningful architectural decisions (ADRs). Examples include:
- Go + SvelteKit + Supabase
- JWT authentication
- Session model
- Encrypted API envelope
- PII masking
- Society lifecycle
- Role model
- Data minimization

Use decision records for significant decisions. Do not create decision records for trivial implementation details.

---

## 18. Documentation Links

When creating or moving a document:
1. Link it to its parent/index.
2. Link it to relevant related documents.
3. Add backlinks where useful.
4. Check existing links pointing to the document.
5. Ensure the document remains discoverable.
6. Avoid orphaned documents.

---

## 19. Validation

Before considering a task complete:
1. Run relevant tests.
2. Validate the changed API/domain behavior.
3. Compare implementation against documentation.
4. Check documentation links affected by the change.
5. Search for stale terminology or old behavior if the change replaced an existing concept.

---

## 20. Definition of Done

A task is not complete until:
- Relevant `/docs` files were reviewed.
- Related documentation was followed.
- Existing behavior was understood.
- Documentation conflicts were identified.
- Required decisions were documented.
- Implementation matches documented behavior.
- Tests/validation pass.
- API documentation is updated when APIs change.
- Domain documentation is updated when domain behavior changes.
- Security documentation is updated when security behavior changes.
- Obsidian links are updated for new/changed documents.
- No obvious broken links remain.
- No intentional documentation drift remains.

---

## 21. Final Enforcement Workflow

For every task, strictly execute the following sequence:

```text
Understand
    ↓
Inspect /docs
    ↓
Read related documentation
    ↓
Explore implementation
    ↓
Impact analysis
    ↓
Document decisions
    ↓
Implement
    ↓
Test
    ↓
Synchronize documentation
    ↓
Verify links
    ↓
Complete
```

The agent must never knowingly leave the project in a state where implementation and `/docs` describe different behavior.

Documentation and implementation are maintained as one unified system.
