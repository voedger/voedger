# Context subsystem architecture: prod/apps/deployment

Apps deployment subsystem architecture for registering built-in and sidecar applications, initializing missing recovery checkpoints, deploying partitions, and rejecting incompatible redeployments. Context-level overview: [arch.md](./arch.md).

## External actors

Roles:

- `@VADeveloper`
  - Supplies built-in app packages and sidecar app modules consumed at deployment time.

- `@Admin`
  - Operates the VVM process and triggers redeployment by restarting the VVM with updated app definitions.

## Scenarios overview

- **`Bootstrap VVM`**
  - On VVM startup initialize the cluster app workspace, register each application's structures, initialize its missing recovery checkpoints, and deploy only the returned eligible partition IDs.

- **`Bootstrap with an unavailable cluster partition`**
  - Return an error and stop before processing other applications; the cluster application's sole partition is required for bootstrap.

- **`Register app`**
  - The cluster app validates compatibility against the previously stored `cluster.App` descriptor and writes the descriptor on first deployment.

- **`Preserve declared partition count after partial initialization`**
  - Keep the descriptor's full partition count for compatibility and routing; use successful checkpoint IDs only to select runtime partitions.

- **`Initialize recovery checkpoints`**
  - Before partition workers start, process every declared partition under a per-application CPU bound and wait for all scheduled attempts. Return deployable IDs and partition errors for bootstrap to log; non-cluster application partition failures do not block readiness.

- **`Deploy app partitions`**
  - App registration builds extension engine pools without partition runtimes. Partition deployment creates runtimes, asynchronous projector actualizers, and schedulers only for the supplied eligible IDs.

- **`Reject incompatible redeployment`**
  - Schema declarations and the deployment descriptor are cross-checked on every redeployment so a partially upgraded process cannot start with mismatched extensions or partition layout.

Undeploy is not implemented: applications and partitions live for the lifetime of the VVM process; redeployment is performed by restarting the VVM with updated app definitions.

## Components

### Layers

```text
External actors
    |
    +-- @VADeveloper
    +-- @Admin
    |
    v
Bootstrap
    |
    +-- [Bootstrap operator]
    +-- [Checkpoint initial recovery]
    |
    v
Cluster app
    |
    +-- [c.cluster.DeployApp]
    +-- [Deployment descriptor validator]
    +-- [cluster.App uniqueness check]
    |
    v
App partitions engine
    |
    +-- [DeployApp]
    +-- [DeployAppPartitions]
    +-- [Extension presence validator]
    |
    v
Schema build
    |
    +-- [VSQL parser]
    +-- [AppDef builder]
    |
    v
State
    |
    +-- [(cluster.App)]
    +-- [(AppStructs)]
```

### Bootstrap

- `[Bootstrap operator]`
  - Runs `btstrp.Bootstrap` with the VVM service context. App registration precedes checkpoint initialization; only returned deployable IDs are passed directly to `[DeployAppPartitions]`, after all scheduled attempts finish. An empty slice skips deployment. The gate applies to cluster, built-in, and sidecar apps, initialized sequentially in bootstrap order.
  - Logs each returned partition-error map entry once through `bootstrap.initial_recovery.error`, with app QName, partition ID, failed operation, and underlying cause. The map is used only for logging, not retained in app partitions or propagated to requests. A fatal setup/context error prevents deployment from that invocation, even if some returned IDs are ready; collected partition errors can still be logged.
  - Requires successful initialization and deployment of the cluster application's sole partition. Returns an error when it fails or has no eligible ID, before processing any other application.
  - Keeps `initializeAppCheckpoints` and `deployAppPartitions` separate. The former logs failures and returns eligible IDs; the latter deploys those IDs. Neither changes the application's descriptor count.
  - Supplies `AppDeploymentDescriptor.NumParts` to `[c.cluster.DeployApp]` and `[DeployApp]`, independently of the number of IDs returned by checkpoint initialization.
  - Uses `btstrp.SplitBuiltInApps` to identify the required cluster app, preserve the remaining built-in app order, and exclude apps deployed through the sidecar path.
  - impl: [pkg/btstrp/impl.go](../../../../pkg/btstrp/impl.go)
  - impl: [pkg/btstrp/utils.go](../../../../pkg/btstrp/utils.go)

- `[Checkpoint initial recovery]`
  - `checkpoints.IInitialRecovery.InitializeCheckpoints(ctx, appQName, numPartitions)` returns `([]istructs.PartitionID, map[istructs.PartitionID]error, error)`: deployable IDs, operation-specific errors keyed by failed partition ID, and an optional fatal app-structures setup or context error. With a nil fatal error, the slice and map are disjoint and cover `0..numPartitions-1` exactly once. An empty slice and complete failure map with a nil fatal error are valid.
  - Initializes persistent checkpoints only, not runtime sequence state. Checkpoint calculation, completion-marker persistence, wrapped error details, and cancellation behavior are specified in [Initial recovery](./recovery--td.md#initial-recovery). Partition-failure logging belongs to `[Bootstrap operator]`.
  - decl: [pkg/checkpoints/types.go](../../../../pkg/checkpoints/types.go)

### Cluster app

- `[c.cluster.DeployApp]`
  - Command on the cluster app workspace that takes `AppDeploymentDescriptor` (`AppQName`, `NumPartitions`, `NumAppWorkspaces`), rejects redeployment of `sys.cluster`, and uses `[Deployment descriptor validator]` for compatibility. Only this healthy command path creates a missing `cluster.App` record and provisions app workspaces; an existing compatible descriptor does not repeat provisioning.
  - `NumPartitions` comes from the configured descriptor's total `NumParts`, including undeployed partitions. The command neither inspects checkpoint failures nor derives this count from runtime partitions. On first deployment it persists the full count; on redeployment it rejects a genuine descriptor change with the existing `409 Conflict`.
  - impl: [pkg/cluster/impl_deployapp.go](../../../../pkg/cluster/impl_deployapp.go)

- `[Deployment descriptor validator]`
  - Read-only lookup and compatibility check used by `[c.cluster.DeployApp]`. Uses `[cluster.App uniqueness check]` and reads the stored descriptor; distinguishes absent, compatible, incompatible, and lookup-error outcomes without provisioning or application event writes.
  - Mismatched `NumPartitions` or `NumAppWorkspaces` preserves the existing `409 Conflict` error and message, regardless of how many partitions can initialize or deploy.
  - impl: [pkg/cluster/impl_deployapp.go](../../../../pkg/cluster/impl_deployapp.go)

- `[cluster.App uniqueness check]`
  - Looks up the `cluster.App` WDoc by unique `AppQName` for `[Deployment descriptor validator]`.
  - impl: [pkg/cluster/appws.vsql](../../../../pkg/cluster/appws.vsql)

### App partitions engine

- `[DeployApp]`
  - Registers an application with the engine, builds `IAppStructs` (built-in path via `IAppStructsProvider.BuiltIn`, sidecar path via `IAppStructsProvider.New` keyed by `extModuleURLs`), and constructs per-`ProcessorKind` extension engine pools sized by `EnginePoolSize`. Redeploying the same app panics.
  - Retains the full declared partition count and routing metadata even when no partitions deploy. Registration creates no `appPartitionRT`, asynchronous actualizer, or scheduler; building app structures and engine pools is not partition deployment.
  - impl: [pkg/appparts/impl.go](../../../../pkg/appparts/impl.go)

- `[DeployAppPartitions]`
  - For each supplied deployable ID creates an `appPartitionRT`, then concurrently deploys per-partition actualizers and schedulers using `IActualizerRunner` / `ISchedulerRunner`. Excluded IDs have no runtime partition, asynchronous projector (including the checkpoint projector), or scheduled job.
  - Uses the existing `IAppPartitions` contract and errors without a deployment-error registry. `Borrow` for an excluded ID returns the existing `appparts.ErrNotFound`; the command processor returns `500 Internal Server Error` with `partition not found`. A missing application instead returns `400 Bad Request` with `application not found`. Identity is carried by structured log attributes, not repeated in these request errors. The complete classification policy is specified in [recovery request handling](./recovery--td.md#recovery-on-command-request).
  - decl: [pkg/appparts/interface.go](../../../../pkg/appparts/interface.go)
  - impl: [pkg/appparts/impl.go](../../../../pkg/appparts/impl.go)

- `[Extension presence validator]`
  - Cross-checks the parsed `IAppDef` against the built-in extension registry so VSQL-declared extensions exist in code and stateless code-only extensions are tied to imported packages.
  - Path to file: [pkg/appparts/impl_app.go](../../../../pkg/appparts/impl_app.go)

### Schema build

- `[VSQL parser]`
  - Parses each `parser.PackageFS` into a `PackageSchemaAST` and merges them into an `AppSchemaAST` per app.
  - Path to file: [pkg/parser/provide.go](../../../../pkg/parser/provide.go)

- `[AppDef builder]`
  - Materializes the merged schema into `appdef.IAppDef` via `parser.BuildAppDefs` and `builder.New`; the result is passed to `[DeployApp]`.
  - Path to file: [pkg/vvm/impl_builder.go](../../../../pkg/vvm/impl_builder.go)

### State

- `[(cluster.App)]`
  - WDoc in the cluster app workspace holding `AppQName`, `NumPartitions`, `NumAppWorkspaces` with a unique constraint on `AppQName`; written on first deployment and read on every redeployment.
  - Path to file: [pkg/cluster/appws.vsql](../../../../pkg/cluster/appws.vsql)

- `[(AppStructs)]`
  - In-memory per-app structures (events, records, view records, resources, validators) used by the processing subsystem; created during `[DeployApp]`.
  - Path to file: [pkg/istructsmem/appstruct-types.go](../../../../pkg/istructsmem/appstruct-types.go)

## Scenarios

### Bootstrap VVM

```text
[Bootstrap operator]
  -> initClusterAppWS -> [(AppStructs)] for sys.cluster
  -> [DeployApp](sys.cluster)
  -> [Checkpoint initial recovery](sys.cluster, declared partition count)
       - log each returned partition error once
       - fatal error -> stop bootstrap; do not deploy returned IDs
       - eligible cluster ID -> [DeployAppPartitions](sys.cluster, deployable IDs)
       - no eligible cluster ID -> return error; stop bootstrap
  -> if cluster partition deployed:
       for each built-in/sidecar app, sequentially:
         [c.cluster.DeployApp]: descriptor's full NumParts; validate; provision only if absent
         [DeployApp](app, full declared partition count)
         -> [Checkpoint initial recovery](app, NumParts)
              - log each returned partition error once
              - fatal error -> stop bootstrap; do not deploy returned IDs
         -> [DeployAppPartitions](app, deployable IDs), only when nonempty
  - ready after bootstrap completes, even if any or all non-cluster application partitions failed
```

### Bootstrap with an unavailable cluster partition

```text
[Bootstrap operator]
  - sys.cluster registered; its partition excluded by initial recovery
  - log the partition's initialization error
  - do not call or retry c.cluster.DeployApp against the excluded partition
  - return an error; no other application registration or partition deployment
  - server does not become ready
```

Cluster workspace-initialization, fatal checkpoint-initialization, and partition-
deployment errors also stop bootstrap. Retrying startup after resolving the cause
uses the normal cluster command path.

### Register app

```text
[c.cluster.DeployApp]
  -> [Deployment descriptor validator]
       -> [cluster.App uniqueness check] -> [(cluster.App)]
       - found, incompatible -> existing 409 Conflict
       - lookup error -> fail
       - found, compatible -> ok; do not provision again
       - absent -> return to command:
           insert [(cluster.App)] {AppQName, NumPartitions, NumAppWorkspaces}
           provision app workspaces
```

### Preserve declared partition count after partial initialization

```text
[Bootstrap operator]
  - configured application descriptor: NumParts=3
  -> [c.cluster.DeployApp]: NumPartitions=3, independently of checkpoint outcomes
       -> [Deployment descriptor validator] -> [(cluster.App)]
            - absent: persist NumPartitions=3 and provision workspaces
            - present, count 3: compatible; retain descriptor
            - present, different count: 409 Conflict; stop bootstrap
  -> [DeployApp]: register full count 3
  -> [Checkpoint initial recovery]: InitializeCheckpoints(app, 3)
       - partition 1 fails; return eligible IDs [0, 2]
  -> [DeployAppPartitions]: start IDs [0, 2]; preserve their identities
  - AppPartsCount remains 3; WSID 10 remains assigned to partition 1
  - command for WSID 10 returns 500: partition not found
  - no reduction to count 2, rerouting to partition 0, or writes for that command
```

The same rule applies with no eligible IDs: a non-cluster application retains
its descriptor count and routing metadata while all its runtime partitions are
undeployed. Availability must never change `WSID % NumPartitions` or rewrite the
persisted descriptor.

### Initialize recovery checkpoints

```text
[Bootstrap operator]
  -> [Checkpoint initial recovery]: InitializeCheckpoints(ctx, appQName, NumPartitions)
      - sample worker limit = runtime.GOMAXPROCS(0) once for this application's call
      - enumerate complete declared range 0..NumPartitions-1
      - present completion marker, including zero -> add ID to deployable IDs
      - process missing partitions with at most worker-limit attempts in flight:
          scan each PLog sequentially from FirstOffset through actual log end
          persist workspace checkpoints before partition completion marker
          successful reconstruction -> add ID to deployable IDs
      - checkpoint-read, scan, or write failure -> add operation-wrapped error to map;
          omit failed ID; do not cancel siblings
      - wait for every scheduled attempt
      - setup failure before scheduling or context cancellation -> fatal error
      - return deployable IDs, partition-error map, fatal setup/context error or nil
  - log each returned map entry once with app QName and partition ID
  - fatal error -> stop bootstrap; do not deploy even partial ready results
  -> [DeployAppPartitions](app, deployable IDs), only if fatal is nil and IDs nonempty
  - for non-cluster apps, continue bootstrap even when every partition is in the error map
  - for the cluster app, no eligible partition -> return an error and stop bootstrap
```

### Deploy app partitions

```text
[VSQL parser]
  -> [AppDef builder] -> appdef.IAppDef
[Bootstrap operator]
  -> [DeployApp]: appdef.IAppDef, full declared partition count
       -> [Extension presence validator]
       - build extension engine pools per ProcessorKind; no partition runtimes
  -> [Checkpoint initial recovery]: wait for all scheduled partition attempts
       - partition errors -> log once; fatal error -> stop bootstrap
  -> [DeployAppPartitions]: only returned deployable IDs, if any
       - per supplied ID:
           create appPartitionRT
           deploy projector actualizers (async)
           deploy schedulers (async)
       - excluded IDs: no runtime partition, asynchronous projectors, or jobs
```

### Reject incompatible redeployment

```text
[Bootstrap operator]
  - cluster partition must already be deployed
  -> [c.cluster.DeployApp]: descriptor's full NumPartitions and NumAppWorkspaces
       -> [Deployment descriptor validator]
       -> [cluster.App uniqueness check] -> [(cluster.App)]
       - descriptor mismatch -> existing 409 Conflict, bootstrap fails
  -> [DeployApp]
       -> [Extension presence validator]
            VSQL-declared extension missing in code      -> panic, bootstrap fails
            built-in extension missing in VSQL           -> panic, bootstrap fails
```

## Cross-cutting concerns

### Bootstrap failure and readiness

- Every `[Checkpoint initial recovery]` invocation returns only after joining its scheduled attempts; a partition-scoped failure does not cancel sibling attempts. Every `[Bootstrap operator]` deployment call uses only eligible IDs from an invocation with a nil fatal error.
- Every non-cluster application's partition-scoped initial-recovery error, including the all-partitions-failed case, is nonfatal to bootstrap and readiness. The cluster application's sole partition must initialize and deploy successfully; otherwise bootstrap returns an error. Every setup error before scheduling and every bootstrap-context cancellation remains fatal.
- Every app registered by `[DeployApp]` retains its declared routing metadata independently of runtime partition deployment. Every excluded partition stays undeployed for the current run, even if storage becomes healthy; requests cannot retry initial recovery or deploy it. A later bootstrap can retry missing completion markers; in-process redeployment is outside this change.
- Every configured non-cluster app processed by bootstrap follows descriptor compatibility validation through `[c.cluster.DeployApp]` before its partitions start. Cluster-partition failure stops bootstrap before this path; descriptor incompatibility and lookup errors also stop bootstrap.
- Every declared partition count supplied to `[c.cluster.DeployApp]`, `[DeployApp]`, `AppPartsCount`, or routing comes from the application's descriptor. Failed initialization may change only the IDs passed to `[DeployAppPartitions]`; it must not reduce the count, renumber IDs, rewrite the descriptor, or remap workspaces to available partitions.
