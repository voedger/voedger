# Domain: prod

## System

All-in-one server platform for development and operation of specialized applications distributed worldwide.

## External actors

Roles:

- 👤VADeveloper
  - Develops Voedger applications using VSQL and WASM extensions
- 👤Admin
  - Deploys and manages Voedger clusters and infrastructure
- 👤WorkspaceOwner
  - Manages workspace content, invitations, and member roles
- 👤AuthenticatedUser
  - Any user with a valid auth token, can join/leave workspaces

Systems:

- ⚙️Client
  - External application that interacts with Voedger platform via HTTP/HTTPS APIs
- ⚙️DBMS
  - Database management system for data persistence
  - Implementations: ScyllaDB, BBolt, Amazon DynamoDB
- ⚙️ACME
  - Automated certificate management and provisioning service

## Contexts

### apps

Application lifecycle management including deployment, versioning, and workspace management.

Relationships with external actors:

- 📦apps -> |supplier-customer| 👤VADeveloper
  - Define data schemas using VSQL
  - Develop WASM extensions
  - Deploy applications to Voedger platform
- 📦apps -> |supplier-customer| ⚙️Client
  - Execute commands and queries
  - Access workspace data
  - Interact with application features via HTTP/HTTPS APIs

### storage

Data persistence and retrieval with event sourcing, CQRS, and multi-backend support.

Relationships with external actors:

- ⚙️DBMS -> |supplier-customer| 📦storage
  - Store and retrieve data with consistency guarantees
  - Support multiple backend implementations (ScyllaDB, BBolt, DynamoDB)

### routing

Request routing, domain management, and HTTPS certificate provisioning.

Relationships with external actors:

- 📦routing -> |supplier-customer| 👤Admin
  - Configure domain routing and certificates
  - Deploy Voedger clusters
  - Manage infrastructure settings
- ⚙️ACME -> |supplier-customer| 📦routing
  - Obtain SSL/TLS certificates for configured domains
  - Handle ACME HTTP-01 challenges
  - Renew certificates automatically

### auth

Authentication, authorization, token management, and workspace membership.

Relationships with external actors:

- 📦auth -> |supplier-customer| ⚙️Client
  - Authenticate requests with JWT tokens
- 📦auth -> |supplier-customer| 👤WorkspaceOwner
  - Manage workspace membership (invite, cancel, update roles)
- 📦auth -> |supplier-customer| 👤AuthenticatedUser
  - Join workspace, leave workspace

### extensions

WASM extension runtime and lifecycle management.

### observability

System metrics, logs, traces, and insights for understanding system behavior and performance.

Relationships with external actors:

- 📦observability -> |supplier-customer| 👤Admin
  - Access observability dashboards
  - View system metrics, logs, and traces
  - Configure alerts and thresholds
