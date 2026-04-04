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

- [x] update: [pkg/sys/authnz/impl_enrichprincipaltoken.go](../../../../pkg/sys/authnz/impl_enrichprincipaltoken.go)
  - replace: anonymous cast to `interface{ GetPrincipals() }` with `processors.IProcessorWorkpiece`

- [x] update: [pkg/sys/workspace/impl.go](../../../../pkg/sys/workspace/impl.go)
  - replace: `args.Workpiece.(interface{ Context() context.Context }).Context()` with `args.State.Context()`

### Sync actualizer

- [x] update: [pkg/processors/actualizers/impl.go](../../../../pkg/processors/actualizers/impl.go)
  - remove: local `syncActualizerWorkpiece` interface
  - replace: all `syncActualizerWorkpiece` references with `processors.IProjectorWorkpiece`
  - rename: `LogCtxForSyncProjector()` usage to `LogCtx()` (requires updating the workpiece implementation)

### Command processor workpiece

- [x] update: [pkg/processors/command/types.go](../../../../pkg/processors/command/types.go)
  - add: compile-time assertion `var _ processors.IProcessorWorkpiece = (*cmdWorkpiece)(nil)`
  - rename: field `logCtxForSyncProjectors` to `logCtx`

- [x] update: [pkg/processors/command/impl.go](../../../../pkg/processors/command/impl.go)
  - rename: `LogCtxForSyncProjector()` to `LogCtx()`, returning `c.logCtx`
  - add: `GetPrincipals()`, `ResetRateLimit()`, `Roles()` methods

- [x] update: [pkg/processors/command/provide.go](../../../../pkg/processors/command/provide.go)
  - rename: `logCtxForSyncProjectors` references to `logCtx`

### Query processor workpiece updates

- [x] update: [pkg/processors/query/impl.go](../../../../pkg/processors/query/impl.go)
  - add: compile-time assertion `var _ processors.IProcessorWorkpiece = (*queryWork)(nil)`
  - add: `func (qw *queryWork) LogCtx() context.Context` returning `qw.msg.RequestCtx()`

- [x] update: [pkg/processors/query2/util.go](../../../../pkg/processors/query2/util.go)
  - upgrade: compile-time assertion from `pipeline.IWorkpiece` to `processors.IProcessorWorkpiece`
  - add: `AppPartitions()`, `AppPartition()`, `GetPrincipals()`, `Roles()` methods
  - add: `func (qw *queryWork) LogCtx() context.Context` returning `qw.msg.RequestCtx()`

### Import cycle fix

- [x] update: [pkg/processors/consts.go](../../../../pkg/processors/consts.go)
  - add: `qNameCDocWorkspaceDescriptor` variable

- [x] update: [pkg/processors/utils.go](../../../../pkg/processors/utils.go)
  - replace: `appdef.QNameCDocWorkspaceDescriptor` with local `qNameCDocWorkspaceDescriptor`
  - remove: `authnz` import

### Test updates

- [x] update: [pkg/processors/actualizers/impl_helpers_test.go](../../../../pkg/processors/actualizers/impl_helpers_test.go)
  - rename: `LogCtxForSyncProjector()` to `LogCtx()`

- [x] update: [pkg/sys/collection/collection_utils_test.go](../../../../pkg/sys/collection/collection_utils_test.go)
  - rename: `LogCtxForSyncProjector()` to `LogCtx()`
