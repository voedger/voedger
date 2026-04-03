# Implementation plan: Replace anonymous workpiece casts with named processor interfaces

## Construction

### Interface definitions

- [x] update: [pkg/processors/types.go](../../../../pkg/processors/types.go)
  - add: `IProcessorWorkpiece` interface (embeds `pipeline.IWorkpiece`, exposes `AppPartitions()`, `AppPartition()`, `GetPrincipals()`, `Roles()`, `ResetRateLimit()`, `LogCtx()`)
  - add: `IProjectorWorkpiece` interface (embeds `pipeline.IWorkpiece`, exposes `AppPartition()`, `Event()`, `LogCtx()`, `PLogOffset()`)

### Call site updates

- [x] update: [pkg/cluster/impl_vsqlupdate.go](../../../../pkg/cluster/impl_vsqlupdate.go)
  - replace: anonymous cast to `interface{ AppPartitions() }` with `processors.IProcessorWorkpiece`

- [x] update: [pkg/registry/impl_createlogin.go](../../../../pkg/registry/impl_createlogin.go)
  - replace: anonymous cast to `interface{ AppPartitions() }` with `processors.IProcessorWorkpiece`

- [x] update: [pkg/sys/sqlquery/impl.go](../../../../pkg/sys/sqlquery/impl.go)
  - replace: anonymous casts to `interface{ AppPartitions() }`, `interface{ AppPartition() }`, `interface{ Roles() }` with `processors.IProcessorWorkpiece`

- [x] update: [pkg/sys/verifier/impl.go](../../../../pkg/sys/verifier/impl.go)
  - replace: anonymous cast to `interface{ ResetRateLimit() }` with `processors.IProcessorWorkpiece`

- [-] update: [pkg/sys/authnz/impl_enrichprincipaltoken.go](../../../../pkg/sys/authnz/impl_enrichprincipaltoken.go) — skipped: import cycle (`processors` → `sys/authnz` → `processors`)
  - replace: anonymous cast to `interface{ GetPrincipals() }` with `processors.IProcessorWorkpiece`

- [x] update: [pkg/sys/workspace/impl.go](../../../../pkg/sys/workspace/impl.go)
  - replace: `args.Workpiece.(interface{ Context() context.Context }).Context()` with `args.State.Context()`

### Sync actualizer

- [x] update: [pkg/processors/actualizers/impl.go](../../../../pkg/processors/actualizers/impl.go)
  - remove: local `syncActualizerWorkpiece` interface
  - replace: all `syncActualizerWorkpiece` references with `processors.IProjectorWorkpiece`
  - rename: `LogCtxForSyncProjector()` usage to `LogCtx()` (requires updating the workpiece implementation)

### Query processor `LogCtx()` implementations

- [x] update: [pkg/processors/query/impl.go](../../../../pkg/processors/query/impl.go)
  - add: `func (qw *queryWork) LogCtx() context.Context` returning `qw.msg.RequestCtx()`

- [x] update: [pkg/processors/query2/util.go](../../../../pkg/processors/query2/util.go)
  - add: `func (qw *queryWork) LogCtx() context.Context` returning `qw.msg.RequestCtx()`
