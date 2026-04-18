# How: Redirect http server internal error log to voedger logger

## Approach

- Add a small adapter in `pkg/goutils/logger` that forwards stdlib logger output to the voedger logger
  - Single exported entry point `NewStdLogBridge(ctx context.Context, stage string, opts ...StdLogBridgeOption) *log.Logger` builds an unexported writer that trims a trailing `\r`/`\n`, drops payloads matching the filter, and calls `logger.LogCtx(ctx, ..., LogLevelError, stage, payload)` once per `Write` so context attributes (`vapp`, `extension`, `reqid`, `wsid`, ...) are preserved and multi-line payloads (e.g. `net/http` panic stack traces) stay a single correlatable log event; the returned `*log.Logger` is the exact type `http.Server.ErrorLog` expects, and the underlying writer is not exported so it cannot be misused as a generic streaming sink (the writer assumes one `Write` call per complete message)
  - Functional option `WithFilter(substrings []string)` lets callers drop noisy payloads (e.g. TLS handshake errors) without introducing a package-level list
- Wire the adapter into production `http.Server` instantiations
  - `pkg/router/impl_http.go`: in `httpServer.prepareBasicServer`/`preRun`, replace `log.New(&annoyingErrorsFilter{log.Default().Writer()}, ...)` with `logger.NewStdLogBridge(s.rootLogCtx, "endpoint.http.error", annoyingHTTPErrorsFilter)`; the filter option is declared once in `pkg/router/consts.go` as `annoyingHTTPErrorsFilter = logger.WithFilter([]string{"TLS handshake error"})` and reused across all server variants; the obsolete `annoyingErrorsFilter` type in `pkg/router/utils.go` is removed
  - Because `rootLogCtx` is built in `preRun`, assign `s.server.ErrorLog` there (just before `Serve`/`ServeTLS`) instead of in `prepareBasicServer`
  - `pkg/ihttpimpl/impl.go` (`acmeServer`) and `pkg/ihttpimpl/provide.go` (processor `server`): set `ErrorLog` to the same adapter with an appropriate base context built via `logger.WithContextAttrs` (e.g. `vapp=sys`, `extension=sys._ACMEServer` / `sys._HTTPProcessor`)
  - `pkg/vvm/metrics/provide.go`: same treatment with `extension=sys._MetricsServer`
- Conventions
  - Stage: `endpoint.http.error` (aligned with existing `endpoint.listen.start`, `endpoint.unexpectedstop`, `endpoint.shutdown`)
  - Level: `Error` (these messages originate from `net/http` internal error log)
  - Attributes: inherited from the per-server root context (`vapp`, `extension`); no request-scoped attributes, since stdlib `ErrorLog` is invoked by `net/http` without request context
- Tests
  - Unit test in `pkg/goutils/logger` using `StartCapture` to verify that writes to the returned `*log.Logger` produce a single `ErrorCtx` record with the configured stage and ctx attributes, that multi-line payloads stay a single record with embedded `\n` preserved in `msg`, that payloads empty after CRLF trimming are suppressed, and that `WithFilter` drops Writes whose payload contains any configured substring
  - Extend `pkg/router/impl_test.go` (which already injects a capturing `ErrorLog`) to assert that the server's internal error output now goes through the voedger logger with stage and `extension` attributes

References:

- [pkg/goutils/logger/logger.go](../../../pkg/goutils/logger/logger.go)
- [pkg/goutils/logger/loggerctx.go](../../../pkg/goutils/logger/loggerctx.go)
- [pkg/goutils/logger/consts.go](../../../pkg/goutils/logger/consts.go)
- [pkg/router/impl_http.go](../../../pkg/router/impl_http.go)
- [pkg/router/utils.go](../../../pkg/router/utils.go)
- [pkg/router/impl_test.go](../../../pkg/router/impl_test.go)
- [pkg/ihttpimpl/impl.go](../../../pkg/ihttpimpl/impl.go)
- [pkg/ihttpimpl/provide.go](../../../pkg/ihttpimpl/provide.go)
- [pkg/vvm/metrics/provide.go](../../../pkg/vvm/metrics/provide.go)
