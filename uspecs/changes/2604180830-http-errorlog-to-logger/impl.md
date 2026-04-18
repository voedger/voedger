# Implementation plan: Redirect http server internal error log to voedger logger

## Technical design

- [x] update: [apps/logging--td.md](../../specs/prod/apps/logging--td.md)
  - add: under `HTTP` section, entry for forwarded `http.Server.ErrorLog` output — level `Error`, stage `endpoint.http.error`, msg `<line from net/http internal error log>`; attributes limited to per-server root context (`vapp`, `extension`); no `reqid` (see [decs.md](decs.md))
  - note: lines matching any substring passed via the `WithFilter` option (e.g. `TLS handshake error`) are filtered out inside the writer before forwarding

## Construction

### Core helper

- [x] update: [pkg/goutils/logger/loggerctx.go](../../../pkg/goutils/logger/loggerctx.go)
  - add: single exported entry point `NewStdlibLogBridge(ctx context.Context, stage string, opts ...StdlibLogBridgeOption) *log.Logger` that internally builds an unexported `errorCtxWriter` and wraps it in `log.New(w, "", 0)`; callers get the exact type `http.Server.ErrorLog` expects and cannot misuse the underlying writer as a generic streaming sink
  - add: functional-option type `StdlibLogBridgeOption` and option `WithFilter(substrings []string)` that causes lines containing any of the substrings to be silently dropped
  - internals: `errorCtxWriter.Write` splits the input on `\n`, trims a trailing `\r`, drops lines matching any substring from the writer's filter list via the `skip` method, and calls `LogCtx` once per remaining non-empty line
  - `errorCtxWriter` carries a `skipStackFrames` field initialized in `NewStdlibLogBridge` from the unexported constant `stdlibLogBridgeSkipStackFrames` (defined in `pkg/goutils/logger/consts.go`), so the emitted `src` attribute points at the user code that called `*log.Logger.{Println,Print,Printf}` rather than at an internal stdlib or `Write` frame
  - contract: each `Write` call must contain one complete message (as stdlib `*log.Logger` always does); partial lines spanning multiple `Write` calls are not buffered
  - no new logger-level attribute constants are introduced; attributes are inherited from the provided `ctx`
- [x] update: [pkg/goutils/logger/consts.go](../../../pkg/goutils/logger/consts.go)
  - add: unexported constant `stdlibLogBridgeSkipStackFrames = 3` in the existing unexported const block next to `logCtxSkipFrames`; consumed by `NewStdlibLogBridge`
- [x] update: [pkg/goutils/logger/README.md](../../../pkg/goutils/logger/README.md)
  - add: `Stdlib log bridge` feature entry under `## Features` pointing at `NewStdlibLogBridge` and `WithFilter`
  - add: before/after collapsible cuts under `## Problem` showing the pain of a DIY stdlib-to-ctx adapter vs. `NewStdlibLogBridge(..., WithFilter(...))`, and a second pair for log capturing in tests that highlights manual writer/log-level cleanup vs. `StartCapture` auto-restore
  - refresh: shifted line numbers in existing feature links (`WithContextAttrs`, `Ctx logging functions`, `sLogAttrsFromCtx`, `ILogCaptor`)

### Wiring in pkg/router

- [x] update: [pkg/router/consts.go](../../../pkg/router/consts.go)
  - add: package-level var `annoyingHTTPErrorsFilter = logger.WithFilter([]string{"TLS handshake error"})` reused across all server variants; `logger` import added
- [x] update: [pkg/router/impl_http.go](../../../pkg/router/impl_http.go)
  - update: in `httpServer.prepareBasicServer`, drop the `ErrorLog` assignment that currently wraps `log.Default().Writer()`; keep the rest of the `http.Server{...}` initialization
  - update: in `httpServer.preRun`, after `s.rootLogCtx` is built, set `s.server.ErrorLog = logger.NewStdlibLogBridge(s.rootLogCtx, "endpoint.http.error", annoyingHTTPErrorsFilter)` so that the per-server root context attributes (`vapp`, `extension`) propagate and TLS handshake noise is dropped inside the writer
  - applies to all four server variants (`sys._HTTPServer`, `sys._AdminHTTPServer`, `sys._HTTPSServer`, `sys._ACMEServer`) via shared `httpServer.preRun`
- [x] update: [pkg/router/utils.go](../../../pkg/router/utils.go)
  - remove: `annoyingErrorsFilter` type and its `Write` method; drop the `bytes` and `io` imports that become unused

### Tests

- [x] update: [pkg/goutils/logger/logger_test.go](../../../pkg/goutils/logger/logger_test.go)
  - add: subtests under `Test_NewStdlibLogBridge` using `StartCapture` that exercise the bridge via `*log.Logger` (`Println`/`Print`)
    - single-line write: one `ErrorCtx` entry with configured stage and ctx attributes
    - multi-line write (payload containing embedded `\n`): one entry per non-empty line; trailing `\r` trimmed
    - blank-line payload produces a single entry (empty lines are suppressed)
    - level check: disabling `LogLevelError` suppresses forwarded writes
    - `WithFilter` drops only matching lines and forwards the rest
- [x] update: [pkg/router/impl_test.go](../../../pkg/router/impl_test.go)
  - add: test that, after router service `Prepare` + `Run` startup, writing to `s.server.ErrorLog` is captured by voedger logger with `stage=endpoint.http.error` and `extension=sys._HTTPServer`, and that a line containing `TLS handshake error` is filtered out (no log entry emitted)
