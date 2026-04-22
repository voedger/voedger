# How: Replace c.cluster.VSqlUpdate with q.cluster.VSqlUpdate2

## Approach

- Start implementation from a deterministic failing integration test in `pkg/sys/it/impl_vsqlupdate_test.go` that reproduces the original hang: use `it.NewOwnVITConfig` with `NumCommandProcessors = 1` so `c.cluster.VSqlUpdate` and its downstream `c.sys.CUD` are guaranteed to land on the same command processor, issue the call with a short HTTP deadline, and expect a successful response within the deadline - this test must fail before any fix (deadline exceeded due to self-deadlock) and pass after `q.cluster.VSqlUpdate2` plus the router shim are in place
- Declare new extensions in the cluster app schema `pkg/cluster/appws.vsql` alongside existing `VSqlUpdate`:
  - `COMMAND LogVSqlUpdate(VSqlUpdateParams)` - no-op, exists only to get the params logged into WLog
  - `QUERY VSqlUpdate2(VSqlUpdateParams) RETURNS VSqlUpdate2Result` where the result carries WLog offsets of the `LogVSqlUpdate` event and of the downstream CUD event (no `NewID` is tracked - `VSqlUpdate2` is for updates, not inserts)
- Implement in package `pkg/cluster`:
  - `c.cluster.LogVSqlUpdate` is wired with `istructsmem.NullCommandExec` directly in `provide.go` - no new handler file needed (logging is done by the command processor itself via WLog)
  - `impl_vsqlupdate2.go`: query closure that
    - parses/validates the SQL using existing `parseAndValidateQuery` from `pkg/cluster/impl_vsqlupdate.go` - the helper already relies on `args.Workpiece.(processors.IProcessorWorkpiece)`, implemented by both command and query workpieces, so it works unchanged for queries
    - invokes `c.cluster.LogVSqlUpdate` via `federation.IFederation` against `clusterapp.ClusterAppWSID` and captures its `CurrentWLogOffset`
    - performs the actual DML by reusing the existing `updateTable`/`insertTable`/`updateCorrupted`/`updateUnlogged` dispatch (extracted into an unexported helper shared with `provideExecCmdVSqlUpdate`), capturing the target CUD's WLog offset when applicable
    - emits a single result row with `LogWLogOffset`, `CUDWLogOffset` via the query callback
  - Register both extensions in `pkg/cluster/provide.go` using `istructsmem.NewCommandFunction` / `istructsmem.NewQueryFunction`
- Router-side compatibility, per API version (response shape must match the entry point):
  - API v1 (`pkg/router/impl_http.go`, `RequestHandler_V1`): detect `busRequest.Resource == "c.cluster.VSqlUpdate"`, rewrite to `q.cluster.VSqlUpdate2`, execute, and emit the v1 command response shape (`CurrentWLogOffset`, `NewIDs`)
  - API v2 (`pkg/router/impl_apiv2.go`, `requestHandlerV2_extension` commands branch): detect `busRequest.QName == cluster.VSqlUpdate`, rewrite to `q.cluster.VSqlUpdate2` (switch `APIPath` to `APIPath_Queries`), execute, and emit the v2 command response shape
  - The two shims are separate on purpose: each must produce a response in the format of its own API version
  - Keep the old `c.cluster.VSqlUpdate` extension registered until deprecation lands to avoid breaking direct handler paths used in integration tests
- Authorization: grant `EXECUTE` on `q.cluster.VSqlUpdate2` and `c.cluster.LogVSqlUpdate` to the same roles that currently execute `c.cluster.VSqlUpdate` (system principals), in `pkg/cluster/appws.vsql`
- Tests:
  - Un-skip the existing tests in `pkg/sys/it/impl_vsqlupdate_test.go` (currently `t.Skip("https://github.com/voedger/voedger/issues/3845")`) - they assert end-to-end behavior of `c.cluster.VSqlUpdate` and must pass after the fix through the router shim
  - Add subtests hitting `q.cluster.VSqlUpdate2` directly for every existing scenario (update/insert table, update corrupted, unlogged update/insert), asserting returned `LogWLogOffset` and `CUDWLogOffset`
  - Add one new deterministic regression test using `it.NewOwnVITConfig` with `NumCommandProcessors = 1` and a short HTTP deadline
  - Add router-level subtests covering both v1 and v2 rewrites of `c.cluster.VSqlUpdate` -> `q.cluster.VSqlUpdate2`
- Deprecation path (tracked in `change.md`, not executed in this change unless requested):
  - After frontend/Live migrate, remove `c.cluster.VSqlUpdate` from `appws.vsql`, delete `provideExecCmdVSqlUpdate`, drop the router shim, and update the comments in `pkg/processors/command/impl.go` that reference `c.cluster.VSqlUpdate`

References:

- [pkg/cluster/appws.vsql](../../../pkg/cluster/appws.vsql)
- [pkg/cluster/provide.go](../../../pkg/cluster/provide.go)
- [pkg/cluster/impl_vsqlupdate.go](../../../pkg/cluster/impl_vsqlupdate.go)
- [pkg/cluster/consts.go](../../../pkg/cluster/consts.go)
- [pkg/processors/command/impl.go](../../../pkg/processors/command/impl.go)
- [pkg/router/impl_apiv2.go](../../../pkg/router/impl_apiv2.go)
- [pkg/sys/it/impl_vsqlupdate_test.go](../../../pkg/sys/it/impl_vsqlupdate_test.go)
