# AIR-3446: processors: Expose workpiece interfaces and LogContext via IState/PrepareArgs

- **Type:** Sub-task
- **Status:** In Progress
- **Assignee:** d.gribanov@dev.untill.com
- **URL:** https://untill.atlassian.net/browse/AIR-3446

## Why

Workpieces that go through command and query processors have interfaces that are missing in `IState` but required for certain functions. In these cases the workpiece is cast to an anonymous interface with the required method, like here:

https://github.com/voedger/voedger/blob/5860bbfcd545246903d6a74ffcd40a2162abc828/pkg/sys/verifier/impl.go#L163

```go
limitsResetter := args.Workpiece.(interface {
    ResetRateLimit(appdef.QName, appdef.OperationKind)
})
```

That looks like a hack. `LogContext` should also be exposed somewhere in `IState` or in `PrepareArgs` to make it possible to implement [AIR-3469](https://untill.atlassian.net/browse/AIR-3469).

## What

- Determine all places where the workpiece is cast to anonymous interfaces (in commands, queries, and projectors) to identify all interfaces required for functions and projectors
- Include all determined interfaces in `IState` or `PrepareArgs`; investigate where the better placement is
- Update technical design for logging if necessary

