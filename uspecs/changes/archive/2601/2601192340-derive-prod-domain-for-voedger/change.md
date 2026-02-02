---
uspecs.registered_at: 2026-01-19T17:43:07Z
uspecs.change_id: 260119-derive-prod-domain-for-voedger
uspecs.baseline: 981c4924796798fcb9ce828b9892e2f8c1dd965f
uspecs.archived_at: 2026-01-19T23:40:11Z
---

# Change request: Derive prod domain for Voedger

## Problem

Voedger currently lacks a formal domain specification that describes its production domain structure, contexts, participants, and relationships. Without this specification, it is difficult to maintain a clear understanding of the system's functional boundaries, context interactions, and external actor relationships. This makes it harder to plan features, communicate architecture, and ensure consistency across the codebase.

## Solution overview

Create a comprehensive production domain specification for Voedger following the uspecs approach. This will include:

- Formal domain definition in Markdown format (uspecs/specs/prod--domain.md) following the structure from uspecs/u/ex-domain-prod.md
- System description: what Voedger is (not capabilities or features)
- Identification of all contexts within the production domain (apps, storage, routing, auth, extensions, monitoring)
- Definition of external actors:
  - Roles: Application Developer, System Administrator
  - Systems: Client, ScyllaDB, BBolt, Amazon DynamoDB, ACME
- Documentation of relationships between contexts and external actors using |service| notation
- Alignment with existing codebase structure and capabilities
