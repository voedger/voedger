# Domain: prod

## System

All-in-one server platform for development and operation of specialized applications distributed worldwide.

## External actors

Roles:

- 👤VADeveloper
  - Develops Voedger applications using VSQL and WASM extensions
- 👤Admin
  - Deploys and manages Voedger clusters and infrastructure

Systems:

- ⚙️Client
  - External application that interacts with Voedger platform via HTTP/HTTPS APIs
- ⚙️DBMS
  - Database management system for data persistence
  - Implementations: ScyllaDB, BBolt, Amazon DynamoDB
- ⚙️ACME
  - Automated certificate management and provisioning service

## Context map

```mermaid
flowchart TB
    storage[("🎯 storage<br/>Data persistence & retrieval")]:::S
    apps["🎯 apps<br/>Application lifecycle"]:::S
    extensions["🎯 extensions<br/>WASM runtime"]:::S
    auth["🎯 auth<br/>Authentication & authorization"]:::S
    routing["🎯 routing<br/>Request routing"]:::S
    monitoring["🎯 monitoring<br/>Metrics & observability"]:::S

    storage -->|"Persist events & state<br/>Retrieve data (CQRS)"| apps
    extensions -->|"Execute WASM extensions<br/>Manage extension state"| apps
    auth -->|"Validate permissions<br/>Enforce access policies"| apps
    apps -->|"Resolve requests<br/>Discover endpoints"| routing
    auth -->|"Authenticate requests<br/>Validate JWT tokens"| routing
    storage -->|"Performance metrics<br/>Capacity tracking"| monitoring
    apps -->|"Performance metrics<br/>Workspace statistics"| monitoring
    auth -->|"Security context<br/>Access policies"| extensions

    classDef S fill:#B5FFFF,color:#333
```

Detailed relationships between contexts:

- 🎯storage -> |supplier-customer| 🎯apps
  - Persist application events and state
  - Retrieve application data with CQRS patterns
- 🎯extensions -> |supplier-customer| 🎯apps
  - Execute WASM extensions for commands and queries
  - Manage extension state and resources
- 🎯auth -> |supplier-customer| 🎯apps
  - Validate user permissions for application operations
  - Enforce workspace-level access policies
- 🎯apps -> |supplier-customer| 🎯routing
  - Resolve incoming requests to target applications
  - Discover application endpoints and workspaces
- 🎯auth -> |supplier-customer| 🎯routing
  - Authenticate incoming requests
  - Validate JWT tokens
- 🎯storage -> |supplier-customer| 🎯monitoring
  - Provide storage performance and health metrics
  - Track storage capacity and usage
- 🎯apps -> |supplier-customer| 🎯monitoring
  - Collect application performance metrics
  - Track workspace and partition statistics
- 🎯auth -> |supplier-customer| 🎯extensions
  - Provide security context for extension execution
  - Enforce access policies within extensions

## Contexts

### apps

Application lifecycle management including deployment, versioning, and workspace management.

Relationships with external actors:

- 🎯apps -> |supplier-customer| 👤VADeveloper
  - Define data schemas using VSQL
  - Develop WASM extensions
  - Deploy applications to Voedger platform
- 🎯apps -> |supplier-customer| ⚙️Client
  - Execute commands and queries
  - Access workspace data
  - Interact with application features via HTTP/HTTPS APIs

### storage

Data persistence and retrieval with event sourcing, CQRS, and multi-backend support.

Relationships with external actors:

- ⚙️DBMS -> |supplier-customer| 🎯storage
  - Store and retrieve data with consistency guarantees
  - Support multiple backend implementations (ScyllaDB, BBolt, DynamoDB)

### routing

Request routing, domain management, and HTTPS certificate provisioning.

Relationships with external actors:

- 🎯routing -> |supplier-customer| 👤Admin
  - Configure domain routing and certificates
  - Deploy Voedger clusters
  - Manage infrastructure settings
- ⚙️ACME -> |supplier-customer| 🎯routing
  - Obtain SSL/TLS certificates for configured domains
  - Handle ACME HTTP-01 challenges
  - Renew certificates automatically

### auth

Authentication, authorization, and token management.

### extensions

WASM extension runtime and lifecycle management.

### monitoring

System metrics, application monitoring, and observability.

Relationships with external actors:

- 🎯monitoring -> |supplier-customer| 👤Admin
  - Access monitoring dashboards
  - View system metrics and logs
  - Configure alerts and thresholds
