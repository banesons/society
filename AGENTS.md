# CHS Agent Rules

## Core Guidelines

1. **Documentation-First Development**:
   - The `/docs` directory is the single source of truth for product requirements, architecture, API contracts, security rules, and workflows.
   - Always activate and follow the `chs-documentation-first` skill (`.agent/skills/chs-documentation-first/SKILL.md`) before planning or writing any code.
   - Never implement features or modify code without first reviewing relevant `/docs` files.

2. **Synchronization & Definition of Done**:
   - Keep `/docs` and code synchronized in the same task.
   - If implementation changes an API, domain model, role, workflow, or security rule, update `/docs` immediately.
   - Maintain Obsidian wikilink integrity across all documentation files.
