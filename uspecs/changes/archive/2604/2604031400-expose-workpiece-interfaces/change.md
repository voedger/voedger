---
registered_at: 2026-04-03T07:32:17Z
change_id: 2604030732-expose-workpiece-interfaces
baseline: 5860bbfcd545246903d6a74ffcd40a2162abc828
issue_url: https://untill.atlassian.net/browse/AIR-3446
archived_at: 2026-04-03T14:00:42Z
---

# Change request: Replace anonymous workpiece casts with named processor interfaces

## Why

Command handlers, query handlers, and projectors cast `args.Workpiece` (typed `interface{}`) to anonymous interfaces to access processor-level methods like `AppPartitions()`, `ResetRateLimit()`, and `GetPrincipals()`. These anonymous casts are not discoverable, not checked against the workpiece implementation at a single definition point, and duplicated across multiple files. `LogCtx` also needs to be exposed for structured logging in extensions (AIR-3469).

See [issue.md](issue.md) for details.

## What

Define named interfaces in `pkg/processors`:

- `IProcessorWorkpiece` for command and query handlers — provides `AppPartitions()`, `AppPartition()`, `GetPrincipals()`, `Roles()`, `ResetRateLimit()`, `LogCtx()`
- `IProjectorWorkpiece` for sync actualizer projectors — provides `AppPartition()`, `Event()`, `LogCtx()`, `PLogOffset()`

Replace anonymous casts with named interface casts at all affected call sites:

- `pkg/cluster/impl_vsqlupdate.go` — `AppPartitions()`
- `pkg/registry/impl_createlogin.go` — `AppPartitions()`
- `pkg/sys/sqlquery/impl.go` — `AppPartitions()`, `AppPartition()`, `Roles()`
- `pkg/sys/verifier/impl.go` — `ResetRateLimit()`
- `pkg/sys/authnz/impl_enrichprincipaltoken.go` — `GetPrincipals()`
- `pkg/sys/workspace/impl.go` — replace `Workpiece` cast with `args.State.Context()` (already available on `IState`)
- `pkg/processors/actualizers/impl.go` — replace `syncActualizerWorkpiece` with `IProjectorWorkpiece`

Add compile-time assertions and missing interface methods:

- `pkg/processors/command/types.go` — compile-time assertion `var _ processors.IProcessorWorkpiece = (*cmdWorkpiece)(nil)`
- `pkg/processors/command/impl.go` — rename `logCtxForSyncProjectors`/`LogCtxForSyncProjector()` to `logCtx`/`LogCtx()`, add `GetPrincipals()`, `ResetRateLimit()`, `Roles()` methods
- `pkg/processors/query/impl.go` — compile-time assertion, add `LogCtx()` method
- `pkg/processors/query2/util.go` — compile-time assertion upgraded from `pipeline.IWorkpiece` to `processors.IProcessorWorkpiece`, add `AppPartitions()`, `AppPartition()`, `GetPrincipals()`, `Roles()`, `LogCtx()` methods

Move `qNameCDocWorkspaceDescriptor` to break import cycle:

- `pkg/processors/consts.go` — add `qNameCDocWorkspaceDescriptor`
- `pkg/processors/utils.go` — replace `appdef.QNameCDocWorkspaceDescriptor` with local `qNameCDocWorkspaceDescriptor`
