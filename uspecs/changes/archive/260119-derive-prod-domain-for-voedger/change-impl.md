# Implementation plan: Derive prod domain for Voedger

## Functional design

- [x] create: `[prod/prod--domain.md](../../specs/prod/prod--domain.md)`
  - System section: All-in-one server platform for development and operation of specialized applications distributed worldwide
  - External actors section with roles: VADeveloper, Admin
  - External actors section with systems: Client, DBMS (ScyllaDB, BBolt, DynamoDB), ACME
  - Context map with mermaid diagram showing relationships between contexts
  - Contexts: apps, storage, routing, auth, extensions, monitoring
  - For each context: description and relationships with external actors using |supplier-customer| notation
