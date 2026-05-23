---
registered_at: 2026-05-23T12:13:12Z
change_id: 2605231213-plog-query-400-bad-request
baseline: 4344f6ebd11c8a3cdf7509027b56717ce5982fed
issue_url: https://untill.atlassian.net/browse/AIR-4029
archived_at: 2026-05-23T12:35:53Z
---

# Change request: Return 400 Bad Request for invalid sys.plog and sys.wlog queries

## Why

Invalid `select` queries against `sys.plog` and `sys.wlog` (malformed limit, non-positive offset, unsupported operator or column, unsupported expression) currently respond with `500 Internal Server Error`. These are client input errors and must be reported as `400 Bad Request`.

## What

- Map log query parameter validation failures from `500 Internal Server Error` to `400 Bad Request`
- Replace unchecked type assertions in `lim()` and `offs()` with ok-form checks so invalid expression shapes also return `400 Bad Request` instead of panicking

## Construction

- [x] update: [it/impl_sqlquery_test.go](../../../../../pkg/sys/it/impl_sqlquery_test.go)
  - update: `TestSqlQuery_readLogParams` subtests - replace `it.Expect500(...)` with `it.Expect400(...)` for all seven cases
  - add: `TestSqlQuery_readLogParams` subtest - `limit sin(5)` expects 400 and `"unsupported limit value expression:"`
  - add: `TestSqlQuery_readLogParams` subtest - `where 5 = 1` expects 400 and `"unsupported column reference expression:"`
  - add: `TestSqlQuery_readLogParams` subtest - `where Offset >= sin(5)` expects 400 and `"unsupported offset value expression:"`

- [x] update: [sqlquery/impl.go](../../../../../pkg/sys/sqlquery/impl.go)
  - update: `params()` call site in `provideExecQrySQLQuery` - wrap the returned error with `coreutils.WrapSysError(e, http.StatusBadRequest)`
  - update: `lim()` - convert `limit.Rowcount.(*sqlparser.SQLVal)` to ok-form; return `"unsupported limit value expression: %T"` on failure
  - update: `offs()` - convert `r.Left.(*sqlparser.ColName)` to ok-form; return `"unsupported column reference expression: %T"` on failure
  - update: `offs()` - convert `r.Right.(*sqlparser.SQLVal)` to ok-form; return `"unsupported offset value expression: %T"` on failure
