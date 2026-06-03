# Context subsystem architecture: prod/apps/deployment

Apps deployment subsystem architecture for registering built-in and sidecar applications, deploying their partitions, and rejecting incompatible redeployments. Context-level overview: [arch.md](./arch.md).

## External actors

Roles:

- `@VADeveloper`
  - Supplies built-in app packages and sidecar app modules consumed at deployment time.

- `@Admin`
  - Operates the VVM process and triggers redeployment by restarting the VVM with updated app definitions.

## Scenarios overview

- **`Bootstrap VVM`**
  - On VVM startup initialize the cluster app workspace, deploy the cluster app partition, register every built-in and sidecar app through `c.cluster.DeployApp`, and deploy their partitions.

- **`Register app`**
  - The cluster app validates compatibility against the previously stored `cluster.App` descriptor and writes the descriptor on first deployment.

- **`Deploy app partitions`**
  - The app partitions engine builds extension engine pools, deploys partition-scoped projector actualizers, and deploys schedulers per partition.

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
    |
    v
Cluster app
    |
    +-- [c.cluster.DeployApp]
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
  - Runs `btstrp.Bootstrap` once on VVM startup: initializes the cluster app workspace via `cluster.InitAppWS`, deploys the cluster app partition, then calls `c.cluster.DeployApp` for each built-in and sidecar app and deploys their partitions through the app partitions engine.
  - Path to file: [pkg/btstrp/impl.go](../../../../pkg/btstrp/impl.go)

### Cluster app

- `[c.cluster.DeployApp]`
  - Command on the cluster app workspace that takes `AppDeploymentDescriptor` (`AppQName`, `NumPartitions`, `NumAppWorkspaces`), rejects redeployment of `sys.cluster`, and either creates the `cluster.App` record or rejects mismatched `NumPartitions` / `NumAppWorkspaces` with `409 Conflict`.
  - Path to file: [pkg/cluster/impl_deployapp.go](../../../../pkg/cluster/impl_deployapp.go)

- `[cluster.App uniqueness check]`
  - Looks up the `cluster.App` WDoc by unique `AppQName` to decide whether to insert a new descriptor or run the compatibility check.
  - Path to file: [pkg/cluster/appws.vsql](../../../../pkg/cluster/appws.vsql)

### App partitions engine

- `[DeployApp]`
  - Registers an application with the engine, builds `IAppStructs` (built-in path via `IAppStructsProvider.BuiltIn`, sidecar path via `IAppStructsProvider.New` keyed by `extModuleURLs`), and constructs per-`ProcessorKind` extension engine pools sized by `EnginePoolSize`. Redeploying the same app panics.
  - Path to file: [pkg/appparts/impl.go](../../../../pkg/appparts/impl.go)

- `[DeployAppPartitions]`
  - For each partition ID creates an `appPartitionRT`, then concurrently deploys per-partition actualizers and schedulers using `IActualizerRunner` / `ISchedulerRunner`.
  - Path to file: [pkg/appparts/impl.go](../../../../pkg/appparts/impl.go)

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
  -> [DeployApp](sys.cluster) -> [DeployAppPartitions](sys.cluster, [0])
  -> for each built-in/sidecar app:
       [c.cluster.DeployApp] -> [cluster.App uniqueness check] -> [(cluster.App)]
  -> for each built-in/sidecar app:
       [DeployApp](app) -> [DeployAppPartitions](app, [0..NumParts-1])
```

### Register app

```text
[c.cluster.DeployApp]
  -> [cluster.App uniqueness check]
       not found -> insert [(cluster.App)] {AppQName, NumPartitions, NumAppWorkspaces}
       found     -> compare NumPartitions, NumAppWorkspaces
                    mismatch -> 409 Conflict
                    match    -> ok
```

### Deploy app partitions

```text
[DeployApp]
  -> [VSQL parser] + [AppDef builder] -> appdef.IAppDef
  -> [Extension presence validator]
  -> build [extension engine pools] per ProcessorKind
[DeployAppPartitions]
  -> per partition:
       deploy projector actualizers (async)
       deploy schedulers (async)
```

### Reject incompatible redeployment

```text
[Bootstrap operator]
  -> [c.cluster.DeployApp] -> [cluster.App uniqueness check]
       descriptor mismatch (NumPartitions / NumAppWorkspaces) -> 409 Conflict, bootstrap fails
  -> [DeployApp]
       -> [Extension presence validator]
            VSQL-declared extension missing in code      -> panic, bootstrap fails
            built-in extension missing in VSQL           -> panic, bootstrap fails
```
