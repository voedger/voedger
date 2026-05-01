# AIR-3801: sqlquery: support int16 to make possible to read from view.air.FdmLog

- Key: AIR-3801
- Type: Task
- Status: In Progress
- Assignee: d.gribanov@dev.untill.com
- URL: <https://untill.atlassian.net/browse/AIR-3801>

## Description

Reading the `FdmLog` rows via `q.sys.SqlQuery` runs into a real gap in voedger that has to be addressed first.

### The blocker

`FdmLog` partition key is `(Year int16, Month int8, Day int8)`, with `int16` clustering keys (`RequestType`, `PartNr`). But `pkg/sys/sqlquery/impl_viewrecords.go` only handles a fixed set of `DataKind`s when binding WHERE literals to the `KeyBuilder`:

```go
// pkg/sys/sqlquery/impl_viewrecords.go
switch f.DataKind() {
case appdef.DataKind_int32:
    fallthrough
case appdef.DataKind_int64:
    fallthrough
case appdef.DataKind_float32:
    fallthrough
case appdef.DataKind_float64:
    fallthrough
case appdef.DataKind_RecordID:
    n := json.Number(string(k.value))
    kb.PutNumber(k.name, n)
```

`int8` and `int16` fall to default → `errUnsupportedDataKind`. And per `IViewRecords.Read` contract — "All fields of `key.PartitionKey` MUST be specified" — WHERE cannot just be dropped either: an empty PK is rejected (`ErrFieldIsEmptyError`).

`q.sys.Collection` is also out — collection works on `CDoc` trees, not views.

### Three options

| # | Approach | Pros | Cons |
|---|----------|------|------|
| 1 | Add `DataKind_int8` / `DataKind_int16` to the fallthrough chain in `impl_viewrecords.go` (voedger), then write the airs-bp3 test using `select Body from air.FdmLog where Year=… and Month=… and Day=… and Time=… and ScreenId=… and RequestType=… and PartNr=…` | Trivial 2-line fix; brings sqlquery in line with the int8/int16 storage support added under #3435; unlocks any view with small-int keys | Cross-repo change (voedger then airs-bp3) |
| 2 | Use the new HTTP `views/{view}` API (`api/v2`) — `TestQueryProcessor2_Views` shows it already filters on `Year`/`Month`/`Day` cleanly | No voedger change | Different API surface; deviates from "sqlquery or collection" |
| 3 | Keep the current direct `IAppStructs` access | No new code | Already rejected by the user |

Option 1 is the right one — it is a one-line fallthrough extension, matches what `kb.PutNumber` already supports under the hood, and the bug is small enough that a focused fix is cleaner than working around it.

### Proposed plan

- In voedger: extend the switch in `pkg/sys/sqlquery/impl_viewrecords.go` to fallthrough `DataKind_int8` and `DataKind_int16` into the `PutNumber` case
- Add a small unit/integration test in `pkg/sys/it/impl_sqlquery_test.go` (or a focused unit test) covering the new kinds
- Switch `airs-bp3/packages/air/it/impl_fdm_test.go` "big body" subtest to query via `q.sys.SqlQuery`, base64-decoding `Body` and asserting `len == 30000`
