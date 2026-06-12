# Implementation plan: Redirect http server internal error log to voedger logger

## Technical design

- [x] update: [apps/logging--td.md](../../../../../specs/prod/apps/logging--td.md)
  - add: HTTP server internal error log routing under the HTTP subsection - bridged via `logger.NewStdErrorLogBridge` using the http server's root log context; define stage and attributes; document the preserved `TLS handshake error` filter via `logger.WithFilter`

## Construction

- [x] update: [pkg/router/consts.go](../../../../../../pkg/router/consts.go)
  - add: `skipAnnoyingErrors = []string{"TLS handshake error"}` package-level var
- [x] update: [pkg/router/impl_http.go](../../../../../../pkg/router/impl_http.go)
  - remove: `ErrorLog: log.New(&annoyingErrorsFilter{...}, ...)` from `http.Server` literal in `prepareBasicServer`
  - add: `s.server.ErrorLog = logger.NewStdErrorLogBridge(s.rootLogCtx, "endpoint.http.error", logger.WithFilter(skipAnnoyingErrors...))` in `preRun` after `s.rootLogCtx` is built
- [x] update: [pkg/router/utils.go](../../../../../../pkg/router/utils.go)
  - remove: `annoyingErrorsFilter` struct and its `Write` method
  - remove: unused `bytes` and `io` imports
- [x] update: [pkg/router/impl_test.go](../../../../../../pkg/router/impl_test.go)
  - add: `Test_HTTPErrorLog_ForwardedToLogger` covering direct bridge writes, `skipAnnoyingErrors` filtering, and real http internal logging via handler panic
