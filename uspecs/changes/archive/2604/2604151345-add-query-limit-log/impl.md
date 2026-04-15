# Implementation plan: Add log when query execs per workspace limit is reached

## Technical design

- [x] update: [logging--td.md](../../../../specs/prod/apps/logging--td.md)
  - add: `routing.qp.limit` entry to Router section

## Construction

- [x] update: [pkg/router/impl_apiv2.go](../../../../../../pkg/router/impl_apiv2.go)
  - update: `sendRequestAndReadResponse()` — move `withLogAttribs` call before the limiter check; add `logger.WarningCtx` with stage `routing.qp.limit` when `limiter.acquire` returns false

- [x] update: [pkg/router/impl_http.go](../../../../../../pkg/router/impl_http.go)
  - update: `RequestHandler_V1()` — move `withLogAttribs` call before the limiter check; add `logger.WarningCtx` with stage `routing.qp.limit` when `limiter.acquire` returns false

- [x] update: [pkg/sys/it/impl_rates_test.go](../../../../../../pkg/sys/it/impl_rates_test.go)
  - add: `logCap.HasLine("stage=routing.qp.limit")` assertions in existing `qpv1` and `qpv2` subtests of `TestQueryLimiter_BasicUsage`

- [x] Review
