# How: Make q.cluster.VSqlUpdate2 cover all c.cluster.VSqlUpdate features

## Approach

- Leave `c.cluster.VSqlUpdate` alone - its extension, handler (`provideExecCmdVSqlUpdate`), and `provide.go` registration stay untouched so unmigrated callers keep working against the same resource and response shape
- Refactor the DML helpers in `pkg/cluster/impl_table.go` and `pkg/cluster/impl_vsqlupdate.go` so the kind dispatch is expressible without command-only plumbing: `insertTable` returns `(cudWLogOffset, newID, err)` instead of writing to `istructs.IIntents`, and `dispatchDML` returns `(cudWLogOffset, newID, err)` with no `IState` / `IIntents` parameters; the legacy command handler writes `NewID` to the command result itself through intents
- In `pkg/cluster/impl_vsqlupdate2.go`, drop the `InsertTable` rejection and route every DML kind (`update table`, `insert table`, `unlogged update`, `update corrupted`, `unlogged insert`) through the query handler; the query calls `c.cluster.LogVSqlUpdate` first and then `dispatchDML` - the same helper the legacy command uses
- Add a `NewID ref` field to `TYPE VSqlUpdate2Result` in `pkg/cluster/appws.vsql`; `insert table` emits the id allocated by `c.sys.CUD`, every other kind emits `NullRecordID`
- Make `CUDWLogOffset` nullable (drop `NOT NULL` on the schema field); only `update table` and `insert table` populate it because they are the only kinds that invoke `c.sys.CUD`, and non-CUD kinds emit `NullOffset` - callers that do not need the value omit it from the `fields` / `keys` selector per [decs.md](decs.md) (CUDWLogOffset result for non-CUD DML)
- Keep the AIR-3656 / AIR-3661 router shim (`dispatchVSqlUpdateShim_V1` / `_V2`) in place: it continues to intercept `c.cluster.VSqlUpdate` requests, reroute them to `q.cluster.VSqlUpdate2`, and reshape the query response back into the legacy command envelope; the shim is the mechanism that both preserves backward compatibility for unmigrated callers and prevents the command-processor self-deadlock on `c.sys.CUD`
- Refactor the shim to share the extract / log / reshape / flush sequence between V1 and V2 via `finalizeShimResponse` + `buildCmdResponse`; make `dispatchVSqlUpdateShim_V2` return a boolean fall-through signal so APIv2 preflight failures delegate error reporting to the standard command processor (see [decs.md](decs.md) Shim log levels and fall-through)
- Inject `timeu.ITime` into `provideExecQryVSqlUpdate2` via `provide.go` so `update corrupted` can stamp the current time through the shared `dispatchDML` helper

## Logging

- Reduce the shim operational logs (`rerouting ...` and the new offset reporting line `LogWLogOffset=<log> (to be sent to the client as CurrentWLogOffset), CUDWLogOffset=<cud>`) to `Verbose` level per [decs.md](decs.md) (Shim log levels and fall-through); integration tests capture at `LogLevelVerbose` via `checkVSqlUpdateShimLog`
- Drop the APIv2 shim-specific `routing.vsqlupdate.error` entries for body parse / args marshal / initial `SendRequest` failures - `dispatchVSqlUpdateShim_V2` returns `false` on any of these and the router falls through to the standard command processor path, which owns the error reporting
- Keep the shim-reply failure entry (`routing.vsqlupdate.error`, status / respErr / body) - it guards the flush-time error path where the shim has already committed to the response
- The new query path gets standard processor logging via `qp.*` for `q.cluster.VSqlUpdate2` and `cp.*` for the downstream `c.cluster.LogVSqlUpdate` and `c.sys.CUD` invocations

## Tests

- Unskip `TestVSqlUpdate_BasicUsage_InsertTable` in `pkg/sys/it/impl_vsqlupdate_test.go` - the legacy command path now supports insert table end-to-end through the refactored `dispatchDML` / `insertTable`; assert the shim log via `checkVSqlUpdateShimLog`
- Replace `TestVSqlUpdate2_RejectsNonUpdate` with `TestVSqlUpdate2_DirectQuery_AllKinds` - direct-query success cases against `q.cluster.VSqlUpdate2` for `insert table` (`NewID > 0`, `CUDWLogOffset > 0`), `update table`, `unlogged update`, and `update corrupted`
- Add `checkVSqlUpdateShimLog` helper asserting both shim log lines (`rerouting ...` and `LogWLogOffset=... sent to the client as CurrentWLogOffset ... CUDWLogOffset=...`) at `LogLevelVerbose` under `stage=routing.vsqlupdate`; adopt it in the shim-touching test families (`UpdateTable`, `InsertTable`, `Corrupted`, `DirectUpdate_View` basic, `DirectUpdate_Record` basic, `DirectInsert` basic)
- Keep `TestVSqlUpdate_NoDeadlockOnSharedCommandProcessor` unchanged - it guards `c.cluster.VSqlUpdate` behavior through the shim path which remains in effect

References:

- [pkg/cluster/appws.vsql](../../../../../pkg/cluster/appws.vsql)
- [pkg/cluster/impl_vsqlupdate.go](../../../../../pkg/cluster/impl_vsqlupdate.go)
- [pkg/cluster/impl_vsqlupdate2.go](../../../../../pkg/cluster/impl_vsqlupdate2.go)
- [pkg/cluster/provide.go](../../../../../pkg/cluster/provide.go)
- [pkg/router/impl_http.go](../../../../../pkg/router/impl_http.go)
- [pkg/router/impl_apiv2.go](../../../../../pkg/router/impl_apiv2.go)
- [pkg/router/impl_vsqlupdate_shim.go](../../../../../pkg/router/impl_vsqlupdate_shim.go)
- [pkg/sys/it/impl_vsqlupdate_test.go](../../../../../pkg/sys/it/impl_vsqlupdate_test.go)
- [.golangci.yml](../../../../../.golangci.yml)
- [decs.md](decs.md)
- [uspecs/specs/prod/apps/logging--td.md](../../../../specs/prod/apps/logging--td.md)
