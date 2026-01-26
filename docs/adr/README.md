# Architecture Decision Records (ADR)

This directory contains Architecture Decision Records for SYNCTACLES.

## What is an ADR?

An Architecture Decision Record (ADR) captures an important architectural decision made along with its context and consequences.

## Format

Each ADR follows this structure:
- **Title:** Short descriptive name
- **Status:** Accepted | Rejected | Deprecated | Superseded
- **Date:** When the decision was made
- **Context:** What is the issue we're facing?
- **Decision:** What decision did we make?
- **Consequences:** What are the positive and negative consequences?

## Index

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [0001](0001-defer-github-branch-protection.md) | Defer GitHub Branch Protection Until Traction | Accepted | 2026-01-26 |

---

## Creating a New ADR

1. Copy template from existing ADR
2. Increment number (next is 0002)
3. Fill in all sections
4. Update this README index
5. Commit with message: `docs: add ADR-XXXX - [title]`

---

*ADRs help us remember why we made certain decisions, especially when revisiting them months or years later.*
