# Context architecture: prod/apps

Apps context architecture for deploying Voedger applications and processing their commands, queries, projectors, jobs, BLOBs, and notifications on application partitions.

## External actors

Roles:

- `@VADeveloper`
  - Builds application schemas and extensions delivered as built-in or sidecar apps.

- `@Admin`
  - Operates the cluster and triggers cluster-level DML through the cluster app.

Systems:

- `*Client`
  - External application invoking commands, queries, BLOB transfers, and N10n subscriptions over Voedger HTTP APIs.

## Scenarios overview

- **`Deploy application`**
  - Bootstrap the cluster app, register a built-in or sidecar app via `c.cluster.DeployApp`, and deploy its partitions, projectors, and schedulers.

- **`Process client request`**
  - Authenticate the caller, authorize against the workspace ACL, invoke the matching extension on an application partition, and return a response.

## Components

### Layers

```text
External actors
    |
    +-- @VADeveloper
    +-- @Admin
    +-- *Client
    |
    v
Apps subsystems
    |
    +-- [[Deployment]]
    +-- [[Processing]]
    +-- [[VVM orchestration]]
    +-- [[Sequences]]
    |
    v
Shared engine
    |
    +-- [App partitions engine]
```

### Apps subsystems

- `[[Deployment]]`
  - Bootstraps the VVM, registers built-in and sidecar apps in the cluster app, and deploys their partitions; checks app compatibility on every redeploy.
  - Path to file: [arch-deployment.md](./arch-deployment.md)

- `[[Processing]]`
  - Runs the command, query (v1 and v2), sync and async actualizer, scheduler, BLOB, and N10n pipelines that invoke extensions on application partitions.
  - Path to file: [arch-processing.md](./arch-processing.md)

- `[[VVM orchestration]]`
  - Acquires and renews VVM leadership and sequences goroutine startup and shutdown so apps subsystems run only while leadership is held.
  - Path to file: [arch-vvm-orch.md](./arch-vvm-orch.md)

- `[[Sequences]]`
  - Generates per-partition PLog offsets, per-workspace WLog offsets, and record IDs consumed by the command pipeline.
  - Path to file: [arch-sequences.md](./arch-sequences.md)
  - Proposed (not implemented) redesign for scalable sequences: [arch2-sequences.md](./arch2-sequences.md)

### Shared engine

- `[App partitions engine]`
  - Process-wide manager of deployed applications and partitions; `[[Deployment]]` calls `DeployApp` / `DeployAppPartitions`, `[[Processing]]` borrows partitions and invokes extensions through `IAppPartition.Invoke` and `IAppPartition.DoSyncActualizer`.
  - Path to package: [pkg/appparts](../../../../pkg/appparts)

## Scenarios

### Deploy application

```text
[[Deployment]]
  -> [App partitions engine]: DeployApp(sys.cluster), DeployAppPartitions(sys.cluster)
  -> [App partitions engine]: DeployApp(builtin/sidecar app), DeployAppPartitions(...)
```

Details: [arch-deployment.md](./arch-deployment.md).

### Process client request

```text
*Client
  -> [[Processing]]: authenticate, authorize, dispatch by pipeline kind
  -> [App partitions engine]: Borrow(app, partitionID, processorKind)
  -> [App partitions engine]: Invoke(extension) or DoSyncActualizer(...)
  -> *Client: response
```

Details: [arch-processing.md](./arch-processing.md).

## Cross-cutting concerns

### Context dependencies

- The `auth` context owns principal token validation and ACL policy consumed by `[[Processing]]` at every pipeline's authnz enforcement points; see [../auth/arch.md](../auth/arch.md).
- The `storage` context persists app definitions, events, records, views, and BLOBs accessed by `[[Deployment]]` and `[[Processing]]` through `istructs.IAppStructs`.
- The `extensions` context provides the WASM and built-in extension engines invoked by `[App partitions engine]`.
- The `observability` context receives metrics, structured logs, and traces produced by the processing pipelines; logging conventions are described in [logging--td.md](./logging--td.md).
