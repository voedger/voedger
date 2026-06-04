---
change_id: 2606030916-derive-app-context-architecture
type: docs
issue_url: https://untill.atlassian.net/browse/AIR-4154
---

# Change request: Derive `apps` context architecture

Refs:

- [AIR-4154: voedger: derive architecture from app context](./issue-AIR-4154.md)

## Why

The `apps` context listed in `uspecs/specs/prod/domain.md` has no Context Architecture (`apps/arch.md`); existing files under `uspecs/specs/prod/apps/` cover narrower concerns (sequences, VVM orchestration, logging) but none provides a top-level architecture reference for deployment and processing.

## What

Add architecture specifications for the `apps` context, derived from existing documentation and the codebase:

- Add a Context Architecture at `uspecs/specs/prod/apps/arch.md` that overviews the `apps` context and links to its subsystem architectures
- Add a Context Subsystem Architecture `uspecs/specs/prod/apps/arch-deployment.md` covering vsql DDL, deployment descriptors, the app partitions engine (deploy and undeploy of app partitions), VVM startup bootstrap (`pkg/btstrp`: initializing the cluster-app workspace, registering built-in and sidecar apps via `c.cluster.DeployApp`, and deploying their app partitions), and protection against incompatible schema and deployment descriptor changes
- Add a Context Subsystem Architecture `uspecs/specs/prod/apps/arch-processing.md` covering the processor pipelines for Command, Query v1, Query v2, Sync Actualizer, Async Actualizer, Scheduler, BLOB, and N10N; authnz/ACL enforcement points within those pipelines (principal token validation and ACL checks delegated to the `auth` context); response sending; and the app partitions engine (invoke extensions on app partitions during request processing)
- Introduce the app partitions engine once in `arch.md` as a shared component and link to both subsystem chapters for its deployment-time and processing-time roles
- Leave the existing `arch-sequences.md`, `arch-vvm-orch.md`, `arch2-sequences.md`, and `logging--td.md` untouched and reference them from `arch.md` where relevant
- Give reviewers and contributors a single architecture reference for the `apps` context that complements `uspecs/specs/prod/domain.md`

## How

Decisions:

- Adopt uspecs-td conventions: `arch.md` for the Context Architecture, `arch-{subsystem}.md` for Context Subsystem Architectures
- Introduce the app partitions engine once in `apps/arch.md` and cross-link from both subsystem chapters for its deploy/undeploy and invoke-extensions roles
- Derive `apps/arch-deployment.md` from `pkg/btstrp`, `pkg/cluster` (`c.cluster.DeployApp`), `pkg/appparts` (`AppDeploymentDescriptor`, `DeployApp`, `DeployAppPartitions`), and VSQL DDL handling in `pkg/parser`
- Derive `apps/arch-processing.md` from `pkg/processors/{command,query,query2,actualizers,schedulers,blobber,n10n}`, with authnz/ACL enforcement points described as calls into `pkg/iauthnz` and response sending via `pkg/bus`
- Link existing `apps/arch-sequences.md`, `apps/arch-vvm-orch.md`, `apps/arch2-sequences.md`, and `apps/logging--td.md` from `apps/arch.md` without modifying them

Out of scope:

- Modifying or consolidating existing `apps/arch-sequences.md`, `apps/arch-vvm-orch.md`, `apps/arch2-sequences.md`, `apps/logging--td.md`
- Cluster-provisioning bootstrap performed by `cmd/ctool`
- Authnz/ACL policy and identity model owned by the `auth` context
- Any behavior change to deployment or processing code paths

References:

- [apps context in the domain specification](../../../../../uspecs/specs/prod/domain.md)
- [existing apps arch-sequences subsystem](../../../../../uspecs/specs/prod/apps/arch-sequences.md)
- [existing apps arch-vvm-orch subsystem](../../../../../uspecs/specs/prod/apps/arch-vvm-orch.md)
- [auth context architecture cross-referenced for authnz/ACL](../../../../../uspecs/specs/prod/auth/arch.md)
- [VVM startup bootstrap package](../../../../../pkg/btstrp)
- [cluster app DeployApp command](../../../../../pkg/cluster)
- [app partitions engine package](../../../../../pkg/appparts)
- [processors packages (command, query, query2, actualizers, schedulers, blobber, n10n)](../../../../../pkg/processors)
- [VSQL parser package](../../../../../pkg/parser)
- [authnz interface used at enforcement points](../../../../../pkg/iauthnz)
- [response sender bus package](../../../../../pkg/bus)

## Technical design

- [x] create: [apps/arch.md](../../../../specs/prod/apps/arch.md)
  - Context Architecture for the `apps` context: overview, shared app partitions engine, links to subsystem chapters and existing `arch-*.md`
- [x] create: [apps/arch-deployment.md](../../../../specs/prod/apps/arch-deployment.md)
  - Context Subsystem Architecture for deployment: vsql DDL, deployment descriptors, app partitions engine (deploy/undeploy), VVM startup bootstrap, schema/descriptor change protection
- [x] create: [apps/arch-processing.md](../../../../specs/prod/apps/arch-processing.md)
  - Context Subsystem Architecture for processing: processor pipelines, authnz/ACL enforcement, response sending, app partitions engine (invoke extensions)
