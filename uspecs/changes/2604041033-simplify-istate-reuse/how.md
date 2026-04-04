# How: Simplify IState reuse in processors

## Approach

There are 4 places where IState is created and reused across processors. Each uses closure-based indirection to feed per-request data into the state. The command processor case is the most overcomplicated.

### Command processor (main target)

Currently `pkg/processors/command/types.go` has a `hostStateProvider` struct with 12 fields mirroring data already available on `cmdWorkpiece`, plus 12 trivial getter methods (e.g. `getAppStructs`, `getWSID`, `getCUD`). These getters are passed as closures to `ProvideCommandProcessorStateFactory` at init time. On each request, `hostStateProvider.get()` copies 12 values from the workpiece into the provider, and the closures read them back.

Simplification: pass closures that read directly from a `*cmdWorkpiece` pointer instead of maintaining a separate mirror struct. Replace `hostStateProvider` with `reusableHostState` — a struct with a `*cmdWorkpiece` field and an `IHostState`. Pass `func() { return wp.appStructs }` style closures at construction. On each request, just set the workpiece pointer using `bind()` func — no field-by-field copy needed. This eliminates `hostStateProvider`, its 12 getter methods, and the `get()` method entirely.

The state is created once per command processor goroutine in `pkg/processors/command/provide.go` (`newReusableHostState`) and reused across requests via `getHostState` in `pkg/processors/command/impl.go`. The `cmdWorkpiece` field referencing it is named `hostState`.

### Query processors (v1 and v2)

Both `pkg/processors/query/impl.go` and `pkg/processors/query2/impl.go` create a new state per request via `ProvideQueryProcessorStateFactory()` with inline closures reading from `queryWork`. These closures already read from the workpiece directly (e.g. `func() istructs.IAppStructs { return qw.appStructs }`). This is already essentially the target pattern — no intermediate mirror struct. The overhead here is that a full new state with all storages is allocated per request rather than reused. This could be improved by adopting the same reuse pattern as the command processor (create once, update workpiece pointer), but the query processor state has additional complexity (`sendPrevQueryObject`, `resultValueBuilder`) that makes reuse less straightforward.

### Sync actualizer

`pkg/processors/actualizers/impl.go` creates one state per projector in `newSyncBranch` using closures that read from an `eventService` struct. The `eventService` is updated per event. This is already a clean pattern with minimal indirection — `eventService` has only 2 fields (`event`, `appStructs`) and serves a legitimate purpose since multiple projector branches share it.

### Async actualizer

`pkg/processors/actualizers/async.go` creates one state per projector in `asyncActualizer.init()` using closures bound to the `asyncProjector` instance (e.g. `p.borrowedAppStructs`, `p.WSIDProvider`, `p.EventProvider`). These closures read from `asyncProjector` fields that are updated per event in `DoAsync`. This is already clean — the closures reference the projector directly without an intermediate mirror struct.

### Summary

The only place that needs simplification is the command processor's `hostStateProvider`. The query processors, sync actualizer, and async actualizer all already use direct closure patterns without unnecessary intermediate state mirroring.

References:

- [pkg/processors/command/types.go](../../pkg/processors/command/types.go)
- [pkg/processors/command/impl.go](../../pkg/processors/command/impl.go)
- [pkg/processors/command/provide.go](../../pkg/processors/command/provide.go)
- [pkg/state/stateprovide/impl_command_processor_state.go](../../pkg/state/stateprovide/impl_command_processor_state.go)
- [pkg/processors/query/impl.go](../../pkg/processors/query/impl.go)
- [pkg/processors/query2/impl.go](../../pkg/processors/query2/impl.go)
- [pkg/state/stateprovide/impl_query_processor_state.go](../../pkg/state/stateprovide/impl_query_processor_state.go)
- [pkg/processors/actualizers/impl.go](../../pkg/processors/actualizers/impl.go)
- [pkg/processors/actualizers/async.go](../../pkg/processors/actualizers/async.go)
- [pkg/state/stateprovide/impl_host_state.go](../../pkg/state/stateprovide/impl_host_state.go)
