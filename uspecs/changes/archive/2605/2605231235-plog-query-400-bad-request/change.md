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

## Construction

- [x] update: [it/impl_sqlquery_test.go](../../../../../pkg/sys/it/impl_sqlquery_test.go)
  - update: `TestSqlQuery_readLogParams` subtests - replace `it.Expect500(...)` with `it.Expect400(...)` for all seven cases

- [x] update: [sqlquery/impl.go](../../../../../pkg/sys/sqlquery/impl.go)
  - update: `params()` call site in `provideExecQrySQLQuery` - wrap the returned error with `coreutils.WrapSysError(e, http.StatusBadRequest)`
