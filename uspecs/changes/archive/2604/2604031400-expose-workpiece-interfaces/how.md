# How: Replace anonymous workpiece casts with named processor interfaces

## Approach

Define two named interfaces in `pkg/processors/types.go` that replace all anonymous workpiece casts:

```go
type IProcessorWorkpiece interface {
    pipeline.IWorkpiece
    AppPartitions() appparts.IAppPartitions
    AppPartition() appparts.IAppPartition
    GetPrincipals() []iauthnz.Principal
    Roles() []appdef.QName
    ResetRateLimit(appdef.QName, appdef.OperationKind)
    LogCtx() context.Context
}

type IProjectorWorkpiece interface {
    pipeline.IWorkpiece
    AppPartition() appparts.IAppPartition
    Event() istructs.IPLogEvent
    LogCtx() context.Context
    PLogOffset() istructs.Offset
}
```

No import cycles: `pkg/processors` already imports `appparts`, `iauthnz`, `istructs`, and `pipeline`.

At each call site, replace `args.Workpiece.(interface{ Method() Type })` with `args.Workpiece.(processors.IProcessorWorkpiece)`. For the sync actualizer, replace the local `syncActualizerWorkpiece` interface in `pkg/processors/actualizers/impl.go` with `processors.IProjectorWorkpiece`.

Special case: `pkg/sys/workspace/impl.go` casts workpiece for `Context()` which is already available on `istructs.IState` — replace with `args.State.Context()`, no workpiece cast needed.

No changes to `istructs.IState`, `PrepareArgs`, `ExecCommandArgs`, `ExecQueryArgs`, or any factory functions.

References:

- [pkg/processors/types.go](../../../../pkg/processors/types.go)
- [pkg/processors/command/impl.go](../../../../pkg/processors/command/impl.go)
- [pkg/processors/query/impl.go](../../../../pkg/processors/query/impl.go)
- [pkg/processors/query2/util.go](../../../../pkg/processors/query2/util.go)
- [pkg/processors/actualizers/impl.go](../../../../pkg/processors/actualizers/impl.go)
- [pkg/cluster/impl_vsqlupdate.go](../../../../pkg/cluster/impl_vsqlupdate.go)
- [pkg/registry/impl_createlogin.go](../../../../pkg/registry/impl_createlogin.go)
- [pkg/sys/sqlquery/impl.go](../../../../pkg/sys/sqlquery/impl.go)
- [pkg/sys/verifier/impl.go](../../../../pkg/sys/verifier/impl.go)
- [pkg/sys/authnz/impl_enrichprincipaltoken.go](../../../../pkg/sys/authnz/impl_enrichprincipaltoken.go)
- [pkg/sys/workspace/impl.go](../../../../pkg/sys/workspace/impl.go)
