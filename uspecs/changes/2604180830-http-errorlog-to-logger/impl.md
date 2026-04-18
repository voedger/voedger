# Implementation plan: Redirect http server internal error log to voedger logger

## Technical design

- [x] update: [apps/logging--td.md](../../specs/prod/apps/logging--td.md)
  - add: under `HTTP` section, entry for forwarded `http.Server.ErrorLog` output — level `Error`, stage `endpoint.http.error`, msg `<line from net/http internal error log>`; attributes limited to per-server root context (`vapp`, `extension`); no `reqid` (see [decs.md](decs.md))
  - note: lines matching any substring from the package-private `httpErrorsToSkip` list (e.g. `TLS handshake error`) are filtered out inside `NewErrorCtxWriter` before forwarding

## Construction

### Core helper

- [x] update: [pkg/goutils/logger/loggerctx.go](../../../pkg/goutils/logger/loggerctx.go)
  - add: exported `NewCtxErrorWriter(ctx context.Context, stage string) io.Writer` returning a writer whose `Write` forwards to `ErrorCtx(ctx, stage, line)`
  - internals: `Write` splits the input on `\n`, trims a trailing `\r`, calls `ErrorCtx` once per non-empty line; no prefix/flags transformation — callers using stdlib `log.Logger` should create it with `log.New(writer, "", 0)` so slog provides the timestamp and the attributes come from ctx
  - no new logger-level attribute constants are introduced; attributes are inherited from the provided `ctx`

### Wiring in pkg/router

- [x] update: [pkg/router/impl_http.go](../../../pkg/router/impl_http.go)
  - update: in `httpServer.prepareBasicServer`, drop the `ErrorLog` assignment that currently wraps `log.Default().Writer()`; keep the rest of the `http.Server{...}` initialization
  - update: in `httpServer.preRun`, after `s.rootLogCtx` is built, set `s.server.ErrorLog = log.New(&annoyingErrorsFilter{w: logger.NewCtxErrorWriter(s.rootLogCtx, "endpoint.http.error")}, "", 0)` so that the per-server root context attributes (`vapp`, `extension`) propagate and TLS handshake noise is still dropped by `annoyingErrorsFilter`
  - applies to all four server variants (`sys._HTTPServer`, `sys._AdminHTTPServer`, `sys._HTTPSServer`, `sys._ACMEServer`) via shared `httpServer.preRun`
- [x] update: [pkg/router/utils.go](../../../pkg/router/utils.go)
  - update: `annoyingErrorsFilter.w` remains `io.Writer`; no API change expected — verify type still accepts the new logger-backed writer

### Tests

- [x] update: [pkg/goutils/logger/logger_test.go](../../../pkg/goutils/logger/logger_test.go)
  - add: subtests under a new top-level test function for `NewCtxErrorWriter` using `StartCapture`
    - single-line write: one `ErrorCtx` entry with configured stage and ctx attributes
    - multi-line write (payload containing embedded `\n`): one entry per non-empty line
    - trailing newline from stdlib `log.Logger.Output` is trimmed, not emitted as an empty entry
    - level check: disabling `LogLevelError` suppresses forwarded writes
- [x] update: [pkg/router/impl_test.go](../../../pkg/router/impl_test.go)
  - add: test that, after router service `Prepare` + `Run` startup, writing to `s.server.ErrorLog` is captured by voedger logger with `stage=endpoint.http.error` and `extension=sys._HTTPServer`, and that a line containing `TLS handshake error` is filtered out (no log entry emitted)
