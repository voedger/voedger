# Implementation plan: Make q.cluster.VSqlUpdate2 cover all c.cluster.VSqlUpdate features

## Technical design

- [x] update: [apps/logging--td.md](../../specs/prod/apps/logging--td.md) - the **VSqlUpdate shim** block is adjusted to reflect the actual shim behavior after the refactor (see [decs.md](decs.md) Shim log levels and fall-through)
  - change: reroute announcement level `Info` → `Verbose`
  - add: `Verbose` offset reporting line (`LogWLogOffset=<log> (to be sent to the client as CurrentWLogOffset), CUDWLogOffset=<cud>`)
  - remove: APIv2 body parse / args marshal error entries - these failures now fall through to the standard command processor which owns the error reporting

## Construction

### Schema and constants

- [x] update: [pkg/cluster/appws.vsql](../../../pkg/cluster/appws.vsql)
  - add: `NewID ref` field (nullable) to `TYPE VSqlUpdate2Result` so `q.cluster.VSqlUpdate2` can surface the record id allocated by `c.sys.CUD` on `insert table`; non-insert rows emit `NullRecordID`
  - change: drop `NOT NULL` from `CUDWLogOffset int64` - the field becomes nullable so non-CUD DML kinds can emit it as zero without colliding with the legitimate `NullOffset` value per [decs.md](decs.md) (CUDWLogOffset result for non-CUD DML)
  - note: `COMMAND VSqlUpdate(VSqlUpdateParams) RETURNS VSqlUpdateResult` stays registered per [decs.md](decs.md) (Retention of c.cluster.VSqlUpdate)
- no change: [pkg/cluster/consts.go](../../../pkg/cluster/consts.go) - `field_NewID` and `qNameVSqlUpdate2Result` are already defined from prior changes and are reused as-is

### Handlers and DML helpers

- [x] update: [pkg/cluster/impl_table.go](../../../pkg/cluster/impl_table.go)
  - change: `insertTable` signature to `(cudWLogOffset istructs.Offset, newID istructs.RecordID, err error)` - no more `istructs.IState` / `istructs.IIntents` params; the caller is responsible for propagating `NewID` to the caller-appropriate result sink
- [x] update: [pkg/cluster/impl_vsqlupdate.go](../../../pkg/cluster/impl_vsqlupdate.go)
  - change: `dispatchDML` signature to `(cudWLogOffset istructs.Offset, newID istructs.RecordID, err error)` with no `istructs.IState` / `istructs.IIntents` params; the helper is now callable from both command and query processor contexts
  - change: `provideExecCmdVSqlUpdate` writes `NewID` onto the command's `VSqlUpdateResult` via `args.State` / `args.Intents` when `dispatchDML` returns a non-null id, preserving the legacy command response shape
- [x] update: [pkg/cluster/impl_vsqlupdate2.go](../../../pkg/cluster/impl_vsqlupdate2.go)
  - remove: the `InsertTable` rejection branch in `provideExecQryVSqlUpdate2`
  - add: call `logVSqlUpdate` then `dispatchDML` and surface `LogWLogOffset` / `CUDWLogOffset` / `NewID` on the `vSqlUpdate2Result` row
  - add: `AsRecordID(field_NewID)` accessor on `vSqlUpdate2Result`
  - add: `timeu.ITime` parameter on `provideExecQryVSqlUpdate2` so `dispatchDML` can stamp the current time for `update corrupted`
- [x] update: [pkg/cluster/provide.go](../../../pkg/cluster/provide.go)
  - add: pass `time` into `provideExecQryVSqlUpdate2`
- no change: `pkg/cluster/impl_updatecorrupted.go`, `pkg/cluster/impl_updateunlogged.go` - the kind-specific helpers are reused as-is by `dispatchDML`

### Router

- no change: [pkg/router/impl_http.go](../../../pkg/router/impl_http.go) - no VSqlUpdate-related changes in this file on this branch
- [x] update: [pkg/router/impl_vsqlupdate_shim.go](../../../pkg/router/impl_vsqlupdate_shim.go) - the shim stays as the rerouting / reshaping mechanism per [decs.md](decs.md) (Router shim retention), but its internals are refactored:
  - change: `dispatchVSqlUpdateShim_V2` returns `bool`; body / args / `SendRequest` preflight failures return `false` so the caller falls through to the standard command processor instead of writing the shim's own error envelope (see [decs.md](decs.md) Shim log levels and fall-through)
  - change: the reroute announcement and the new offset reporting line are emitted at `Verbose` level through `logger.VerboseCtx`; `dispatchVSqlUpdateShim_V2` moves the announcement after the preflight so it only fires when the shim actually commits to the reroute
  - add: `finalizeShimResponse` helper extracted from both dispatchers - captures the extract / log / reshape / flush sequence; it also emits the new `LogWLogOffset=<log> (to be sent to the client as CurrentWLogOffset), CUDWLogOffset=<cud>` line on success
  - add: `buildCmdResponse(logOffset, newID, offsetKey, resultKey)` shared JSON builder for the V1 (`CurrentWLogOffset` / `Result`) and V2 (`currentWLogOffset` / `result`) command envelope shapes; the builder surfaces `NewID` under the result key when the underlying `insert table` returned a non-null id
  - rename: extractor helpers to `extractFromQryVSqlUpdate2ResponseV1` / `_V2`; both now return `(logOffset, cudOffset, newID)`
  - change: `rewriteVSqlUpdateBody` widens the V1 `fields` selector to pull `LogWLogOffset`, `CUDWLogOffset`, `NewID`
  - change: `dispatchVSqlUpdateShim_V2` rewrites `args` / `keys` / method / api-path / qname on `busRequest` to call the query instead of the command
- [x] update: [pkg/router/impl_apiv2.go](../../../pkg/router/impl_apiv2.go) - merges the shim invocation into the branch condition (`if isShim && dispatchVSqlUpdateShim_V2(...) { return }`) so a `false` return from the shim lets the request fall through to the standard command processor path

### Lint configuration

- no change: [.golangci.yml](../../../.golangci.yml) - the `"to"` entry in the `revive` `add-constant` allowlist is still needed for the shim's `rerouting X to Y` message

### Tests

- [x] update: [pkg/sys/it/impl_vsqlupdate_test.go](../../../pkg/sys/it/impl_vsqlupdate_test.go)
  - unskip: `TestVSqlUpdate_BasicUsage_InsertTable` - the legacy command path now supports insert table end-to-end through the refactored `dispatchDML` / `insertTable`; asserts the shim log via `checkVSqlUpdateShimLog`
  - replace: `TestVSqlUpdate2_RejectsNonUpdate` with `TestVSqlUpdate2_DirectQuery_AllKinds` - direct-query success cases against `q.cluster.VSqlUpdate2` for `insert table` (asserts `NewID > 0` and `CUDWLogOffset > 0`), `update table`, `unlogged update`, and `update corrupted`; `unlogged insert` direct-query coverage is transitive through the shim path exercised by `TestVSqlUpdate_BasicUsage_DirectInsert`
  - add: `checkVSqlUpdateShimLog` helper that asserts both shim log lines (`rerouting ...` and `LogWLogOffset=... sent to the client as CurrentWLogOffset ... CUDWLogOffset=...`) at `LogLevelVerbose` under `stage=routing.vsqlupdate`
  - update: `TestVSqlUpdate_BasicUsage_UpdateTable` apiv1 / apiv2 subtests to capture logs at `LogLevelVerbose` and call `checkVSqlUpdateShimLog`
  - update: `TestVSqlUpdate_BasicUsage_Corrupted`, `TestVSqlUpdate_BasicUsage_DirectUpdate_View` (`basic`), `TestVSqlUpdate_BasicUsage_DirectUpdate_Record` (`basic`) and `TestVSqlUpdate_BasicUsage_DirectInsert` (`basic`) to capture logs at `LogLevelVerbose` and call `checkVSqlUpdateShimLog`
  - keep: `TestVSqlUpdate_NoDeadlockOnSharedCommandProcessor` unchanged - the regression is guarded by the shim path which remains in effect
