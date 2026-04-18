# How: Redirect http server internal error log to voedger logger

## Approach

- Add a small adapter in `pkg/goutils/logger` that returns an `io.Writer` whose writes are forwarded to the voedger logger
  - New exported helper `NewErrorCtxWriter(ctx context.Context, stage string) io.Writer`
  - Internally an `io.Writer` type splits the payload by `\n`, trims a trailing `\r`, drops lines matching any substring from the package-private `httpErrorsToSkip` list, and calls `logger.LogCtx(ctx, ..., LogLevelError, stage, line)` so context attributes (`vapp`, `extension`, `reqid`, `wsid`, ...) are preserved
  - Callers wrap it with `log.New(writer, "", 0)` to disable stdlib prefix/flags and avoid duplicated timestamps (slog adds its own timestamp)
- Wire the adapter into production `http.Server` instantiations
  - `pkg/router/impl_http.go`: in `httpServer.prepareBasicServer`/`preRun`, replace `log.New(&annoyingErrorsFilter{log.Default().Writer()}, ...)` with `log.New(loggerWriter, "", 0)`, where `loggerWriter` is produced by the new helper bound to `s.rootLogCtx`; TLS handshake noise is dropped inside the writer via `httpErrorsToSkip`, so `annoyingErrorsFilter` is removed
  - Because `rootLogCtx` is built in `preRun`, assign `s.server.ErrorLog` there (just before `Serve`/`ServeTLS`) instead of in `prepareBasicServer`
  - `pkg/ihttpimpl/impl.go` (`acmeServer`) and `pkg/ihttpimpl/provide.go` (processor `server`): set `ErrorLog` to the same adapter with an appropriate base context built via `logger.WithContextAttrs` (e.g. `vapp=sys`, `extension=sys._ACMEServer` / `sys._HTTPProcessor`)
  - `pkg/vvm/metrics/provide.go`: same treatment with `extension=sys._MetricsServer`
- Conventions
  - Stage: `endpoint.http.error` (aligned with existing `endpoint.listen.start`, `endpoint.unexpectedstop`, `endpoint.shutdown`)
  - Level: `Error` (these messages originate from `net/http` internal error log)
  - Attributes: inherited from the per-server root context (`vapp`, `extension`); no request-scoped attributes, since stdlib `ErrorLog` is invoked by `net/http` without request context
- Tests
  - Unit test in `pkg/goutils/logger` using `StartCapture` to verify that writes to the returned `*log.Logger` produce a single `ErrorCtx` line with the configured stage and ctx attributes, and that multi-line payloads are split into one log entry per non-empty line
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
