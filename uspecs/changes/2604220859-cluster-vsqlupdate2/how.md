# How: Replace c.cluster.VSqlUpdate with q.cluster.VSqlUpdate2

## Approach

- Start implementation from a deterministic failing integration test in `pkg/sys/it/impl_vsqlupdate_test.go` that reproduces the original hang: use `it.NewOwnVITConfig` with `NumCommandProcessors = 1` so `c.cluster.VSqlUpdate` and its downstream `c.sys.CUD` are guaranteed to land on the same command processor, issue the call with a short HTTP deadline, and expect a successful response within the deadline - this test must fail before any fix (deadline exceeded due to self-deadlock) and pass after `q.cluster.VSqlUpdate2` plus the router shim are in place
- Declare new extensions in the cluster app schema `pkg/cluster/appws.vsql` alongside existing `VSqlUpdate`:
  - `COMMAND LogVSqlUpdate(VSqlUpdateParams)` - no-op, exists only to get the params logged into WLog
  - `QUERY VSqlUpdate2(VSqlUpdateParams) RETURNS VSqlUpdate2Result` where the result carries WLog offsets of the `LogVSqlUpdate` event and of the downstream CUD event (no `NewID` is tracked - `VSqlUpdate2` is for updates, not inserts)
- Implement in package `pkg/cluster`:
  - `c.cluster.LogVSqlUpdate` is wired with `istructsmem.NullCommandExec` directly in `provide.go` - no new handler file needed (logging is done by the command processor itself via WLog)
  - `impl_vsqlupdate2.go`: query closure that
    - parses/validates the SQL using existing `parseAndValidateQuery` from `pkg/cluster/impl_vsqlupdate.go` (its first parameter is relaxed to `workpiece interface{}` so it accepts both command and query workpieces)
    - rejects `dml.OpKind_InsertTable` with `http.StatusBadRequest` because the query path has no `istructs.IIntents` to allocate `NewID`; `insert table` stays on the legacy command path
    - invokes `c.cluster.LogVSqlUpdate` via `federation.IFederation` against `args.WSID` and captures its `CurrentWLogOffset`
    - performs the actual DML by reusing the existing `updateTable`/`updateCorrupted`/`updateUnlogged` dispatch (extracted into an unexported `dispatchDML` helper shared with `provideExecCmdVSqlUpdate`); `updateTable` now returns the target CUD's WLog offset (it reads `CurrentWLogOffset` from the `c.sys.CUD` response and no longer uses `WithDiscardResponse`)
    - emits a single result row with `LogWLogOffset`, `CUDWLogOffset` via the query callback
  - Register both extensions in `pkg/cluster/provide.go` using `istructsmem.NewCommandFunction` / `istructsmem.NewQueryFunction`
- Router-side compatibility, per API version (response shape must match the entry point):
  - API v1 (`pkg/router/impl_http.go`, `RequestHandler_V1`): detect `busRequest.Resource == "c.cluster.VSqlUpdate"` and `dml.OpKind_UpdateTable` in the body, rewrite to `q.cluster.VSqlUpdate2`, execute, and emit the v1 command response shape (`{"CurrentWLogOffset":<LogWLogOffset>}`)
  - API v2 (`pkg/router/impl_apiv2.go`, shared `sendRequestAndReadResponse`): detect `busRequest.QName == cluster.VSqlUpdate` and `dml.OpKind_UpdateTable` in the body, rewrite to `q.cluster.VSqlUpdate2` (switch `APIPath` to `APIPath_Queries`), execute, and emit the v2 command response shape (`{"currentWLogOffset":<LogWLogOffset>}`); placing the shim in the shared helper (instead of in `requestHandlerV2_extension`) avoids duplicating the `wsQueryLimiter` gating logic and covers every v2 entry point that calls `sendRequestAndReadResponse`
  - In both versions the shim dispatch is placed inside the same `wsQueryLimiter`-protected block as native queries: the shim runs on the query processor, so it must be throttled together with native queries
  - The two shims are separate on purpose: each must produce a response in the format of its own API version
  - Both shims use a shared `capturingResponseWriter` to reuse `reply_v1` / `reply_v2` for the underlying streaming; the captured body is parsed with `federation.QueryResponse` / `federation.FuncResponse` to read `LogWLogOffset`
  - Observability: the reroute is announced once per request at level `Info` with stage `routing.vsqlupdate`; shim-specific failures (body parse, args marshal, downstream reply failure) are logged at `Error` with stage `routing.vsqlupdate.error` right before the reply is flushed; transport errors on `SendRequest` reuse the existing `routing.send2vvm.error` stage
  - Keep the old `c.cluster.VSqlUpdate` extension registered until deprecation lands to avoid breaking direct handler paths used in integration tests
- Authorization: no GRANT statement is added for the new extensions; the existing `c.cluster.VSqlUpdate` is not granted either - access to cluster-app extensions is gated by the system principal check
- Tests:
  - Un-skip the existing tests in `pkg/sys/it/impl_vsqlupdate_test.go` (`TestVSqlUpdate_BasicUsage_UpdateTable`, `TestVSqlUpdate_BasicUsage_InsertTable`, `TestDirectUpdateManyTypes`) - they assert end-to-end behavior of `c.cluster.VSqlUpdate` and must pass after the fix through the router shim
  - Add `TestVSqlUpdate2_DirectQuery` hitting `q.cluster.VSqlUpdate2` directly and asserting returned `LogWLogOffset` and `CUDWLogOffset`
  - Add one new deterministic regression test using `it.NewOwnVITConfig` with `NumCommandProcessors = 1` and a short HTTP deadline
  - Cover the v2 shim with an `apiv2` subtest inside `TestVSqlUpdate_BasicUsage_UpdateTable` (the original v1 path is the default for every other test that posts via `vit.PostApp`)
  - Assert the reroute log in both `apiv1` and `apiv2` subtests of `TestVSqlUpdate_BasicUsage_UpdateTable` using `logger.StartCapture` + `EventuallyHasLine` on the `routing.vsqlupdate` stage to pin down the observability contract
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
