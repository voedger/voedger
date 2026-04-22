# Implementation plan: Replace c.cluster.VSqlUpdate with q.cluster.VSqlUpdate2 to avoid command processor deadlock

## Construction

### Failing regression test (red first)

- [x] update: [pkg/sys/it/impl_vsqlupdate_test.go](../../../pkg/sys/it/impl_vsqlupdate_test.go)
  - add: deterministic regression test using `it.NewOwnVITConfig` with `cfg.NumCommandProcessors = 1` (forces `c.cluster.VSqlUpdate` and the downstream `c.sys.CUD` to share the single command processor); post with a short HTTP deadline; expect a successful response within the deadline (fails with deadline-exceeded before the fix, passes after)

### Schema and constants

- [x] update: [pkg/cluster/appws.vsql](../../../pkg/cluster/appws.vsql)
  - add: `TYPE VSqlUpdate2Result` with `LogWLogOffset int64`, `CUDWLogOffset int64` (no `NewID` - `VSqlUpdate2` is for updates, not inserts)
  - add: `COMMAND LogVSqlUpdate(VSqlUpdateParams)` and `QUERY VSqlUpdate2(VSqlUpdateParams) RETURNS VSqlUpdate2Result` to the existing `EXTENSION ENGINE BUILTIN` block
  - note: no GRANT statements added; existing `c.cluster.VSqlUpdate` is not granted either — access is gated by the sys principal check
- [x] update: [pkg/cluster/consts.go](../../../pkg/cluster/consts.go)
  - add: QName constants `qNameQryVSqlUpdate2`, `qNameCmdLogVSqlUpdate`, `qNameVSqlUpdate2Result`; field constants `field_LogWLogOffset`, `field_CUDWLogOffset`

### Handler

- [x] create: [pkg/cluster/impl_vsqlupdate2.go](../../../pkg/cluster/impl_vsqlupdate2.go)
  - add: `provideExecQryVSqlUpdate2` that parses/validates via shared helper (reuses `parseAndValidateQuery`, which goes through `args.Workpiece.(processors.IProcessorWorkpiece)` - implemented by both command and query workpieces), rejects `InsertTable` (query path is update-only, `Intents` unavailable), invokes `c.cluster.LogVSqlUpdate` through `federation.IFederation` against `args.WSID` (ClusterAppWSID in practice), then dispatches the DML through the shared helper, emitting a single result row with `LogWLogOffset` and `CUDWLogOffset`
- no handler file is needed for `c.cluster.LogVSqlUpdate`: it is a pure no-op that relies on WLog and is wired directly in `provide.go` using `istructsmem.NullCommandExec`

### Refactor and wiring

- [x] update: [pkg/cluster/impl_vsqlupdate.go](../../../pkg/cluster/impl_vsqlupdate.go)
  - refactor: extract the `switch update.Kind { ... }` dispatch into an unexported `dispatchDML` helper reused by both `provideExecCmdVSqlUpdate` and `provideExecQryVSqlUpdate2` (DRY)
  - refactor: change `parseAndValidateQuery` to take `workpiece interface{}` instead of `istructs.ExecCommandArgs` so it is callable from both command and query handlers
- [x] update: [pkg/cluster/impl_table.go](../../../pkg/cluster/impl_table.go)
  - refactor: `updateTable` now returns `(cudWLogOffset istructs.Offset, err error)` by reading `CurrentWLogOffset` from the `c.sys.CUD` response (removed `WithDiscardResponse`) so that `q.cluster.VSqlUpdate2` can expose the CUD WLog offset to callers
- [x] update: [pkg/cluster/provide.go](../../../pkg/cluster/provide.go)
  - add: register `c.cluster.LogVSqlUpdate` via `istructsmem.NewCommandFunction(..., istructsmem.NullCommandExec)`
  - add: register `q.cluster.VSqlUpdate2` via `istructsmem.NewQueryFunction`

### Router compatibility shim

The shim is implemented per API version because the response shape must match the entry point: a client calling v1 must get a v1 command response, a client calling v2 must get a v2 command response. In both versions the shim runs on the query processor, so its dispatch is placed inside the same `wsQueryLimiter`-protected block as native queries to share the gating.

- [x] create: [pkg/router/impl_vsqlupdate_shim.go](../../../pkg/router/impl_vsqlupdate_shim.go)
  - add: `isVSqlUpdateV1Call`, `isVSqlUpdateV2Call` predicates that gate the shim on `isUpdateTableBody`, which uses `dml.ParseQuery` and accepts only `dml.OpKind_UpdateTable` (insert/unlogged/update-corrupted stay on the original command path)
  - add: `capturingResponseWriter` (buffers headers/status/body, implements `http.Flusher`) so the shim can reuse `reply_v1` / `reply_v2` unchanged and then rewrite the final response shape
  - add: `rewriteVSqlUpdateBody` — splices `"elements":[{"fields":["LogWLogOffset","CUDWLogOffset"]}]` into the incoming body via `bytes.Buffer` (no JSON round-trip)
  - add: `dispatchVSqlUpdateShim_V1` — rewrites `busRequest.Resource` to `q.cluster.VSqlUpdate2`, calls `reply_v1` into the capturing writer, then emits `{"CurrentWLogOffset":<LogWLogOffset>}` on success
  - add: `dispatchVSqlUpdateShim_V2` — rewrites `busRequest` to `APIPath_Queries` with `cluster.VSqlUpdate2` QName, moves `args` to the `args` query param and sets `keys` to `LogWLogOffset,CUDWLogOffset`, calls `reply_v2` into the capturing writer, then emits `{"currentWLogOffset":<LogWLogOffset>}` on success
  - add: `extractLogWLogOffsetFromV1Body` / `extractLogWLogOffsetFromV2Body` — unmarshal the captured body via `federation.QueryResponse` / `federation.FuncResponse` and read the first `LogWLogOffset`; unreachable error branches are marked `// notest`
  - add: logging stages `routing.vsqlupdate` (Info, reroute announcement) and `routing.vsqlupdate.error` (Error, body parse / args marshal / downstream reply failures); transport error on `SendRequest` is logged under the existing `routing.send2vvm.error` stage with a shim-specific message `forwarding <source> to <target> failed: <err>`
- [x] update: [pkg/router/impl_http.go](../../../pkg/router/impl_http.go)
  - add: in `RequestHandler_V1`, compute `isVSqlUpdateV1Call` once, include it alongside the `q.*` predicate in the `wsQueryLimiter` gate, and dispatch `dispatchVSqlUpdateShim_V1` from inside the limiter-protected block instead of `SendRequest`
- [x] update: [pkg/router/impl_apiv2.go](../../../pkg/router/impl_apiv2.go)
  - add: in shared `sendRequestAndReadResponse`, compute `isVSqlUpdateV2Call` once, include it alongside the GET-on-QP-bound-path predicate in the `wsQueryLimiter` gate, and dispatch `dispatchVSqlUpdateShim_V2` from inside the limiter-protected block (covers every v2 entry point that calls `sendRequestAndReadResponse`, not only `requestHandlerV2_extension`)
- [x] update: [.golangci.yml](../../../.golangci.yml)
  - add: `"to"` to the `revive` `add-constant` `allowStrs` allowlist to admit the `"rerouting X to Y"` log message literal

### Tests

- [x] update: [pkg/sys/it/impl_vsqlupdate_test.go](../../../pkg/sys/it/impl_vsqlupdate_test.go)
  - add: `TestVSqlUpdate_NoDeadlockOnSharedCommandProcessor` — deterministic regression test that passes only with the shim in place
  - add: `TestVSqlUpdate2_DirectQuery` — posts directly to `q.cluster.VSqlUpdate2` and asserts the returned `LogWLogOffset` and `CUDWLogOffset` are positive
  - change: `TestVSqlUpdate_BasicUsage_UpdateTable` — un-skipped and split into `apiv1` / `apiv2` subtests sharing a single VIT; `apiv2` posts to `/api/v2/.../commands/cluster.VSqlUpdate` and asserts the v2 command response carries `currentWLogOffset`; both subtests use `logger.StartCapture` + `EventuallyHasLine` to assert the `routing.vsqlupdate` reroute log line
  - change: un-skipped `TestVSqlUpdate_BasicUsage_InsertTable` and `TestDirectUpdateManyTypes`
  - note: the remaining tests exercise the v1 shim end-to-end because they post to `c.cluster.VSqlUpdate` via `vit.PostApp` (the v1 entry point)

## Quick start

Call the new query directly (replaces `c.cluster.VSqlUpdate`):

```bash
curl -X POST \
  -H "Authorization: Bearer ${SYS_TOKEN}" \
  -d '{"args":{"Query":"update untill.app1.140737488486400.app1pkg.Customer.322685000131099 set ClientID = '\'''\'', ClientConfigured = false"},"elements":[{"fields":["LogWLogOffset","CUDWLogOffset"]}]}' \
  "http://${HOST}/api/untill/cluster/${CLUSTER_APP_WSID}/q.cluster.VSqlUpdate2"
```

Existing callers may keep posting to `c.cluster.VSqlUpdate`; the router transparently reroutes the request to `q.cluster.VSqlUpdate2` until the old command is removed in a follow-up change.
