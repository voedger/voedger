# Implementation plan: Redirect http server internal error log to voedger logger

## Technical design

- [x] update: [apps/logging--td.md](../../specs/prod/apps/logging--td.md)
  - add: under `HTTP` section, entry for forwarded `http.Server.ErrorLog` output — level `Error`, stage `endpoint.http.error`, msg `<payload from net/http internal error log; embedded newlines preserved and escaped by slog text handler>`; attributes limited to per-server root context (`vapp`, `extension`); no `reqid` (see [decs.md](decs.md))
  - note: payloads containing any substring passed via the `WithFilter` option (e.g. `TLS handshake error`) are dropped inside the writer before forwarding

## Construction

### Core helper

- [x] update: [pkg/goutils/logger/loggerctx.go](../../../pkg/goutils/logger/loggerctx.go)
  - add: single exported entry point `NewStdLogBridge(ctx context.Context, stage string, opts ...StdLogBridgeOption) *log.Logger` that internally builds an unexported `stdLogBridgeWriter` and wraps it in `log.New(w, "", 0)`; callers get the exact type `http.Server.ErrorLog` expects and cannot misuse the underlying writer as a generic streaming sink
  - add: functional-option type `StdLogBridgeOption` and option `WithFilter(substrings []string)` that causes payloads containing any of the substrings to be silently dropped
  - internals: `stdLogBridgeWriter.Write` trims trailing `\r`/`\n` bytes, drops payloads that are empty after trimming or match the writer's filter list via the `skip` method, and calls `LogCtx` once per `Write` with the full (possibly multi-line) payload; embedded newlines stay in `msg` and are escaped by slog's text handler to the two-char `\n` on output, keeping `net/http` panic stack traces a single correlatable log event
  - `stdLogBridgeWriter` carries a `skipStackFrames` field initialized in `NewStdLogBridge` from the unexported constant `stdLogBridgeSkipStackFrames` (defined in `pkg/goutils/logger/consts.go`), so the emitted `src` attribute points at the user code that called `*log.Logger.{Println,Print,Printf}` rather than at an internal stdlib or `Write` frame
  - contract: one `Write` call produces at most one log record; partial lines spanning multiple `Write` calls are not buffered (stdlib `*log.Logger` always delivers a complete message per call)
  - no new logger-level attribute constants are introduced; attributes are inherited from the provided `ctx`
- [x] update: [pkg/goutils/logger/consts.go](../../../pkg/goutils/logger/consts.go)
  - add: unexported constant `stdLogBridgeSkipStackFrames = 3` in the existing unexported const block next to `logCtxSkipFrames`; consumed by `NewStdLogBridge`
- [x] update: [pkg/goutils/logger/README.md](../../../pkg/goutils/logger/README.md)
  - add: `Stdlib log bridge` feature entry under `## Features` pointing at `NewStdLogBridge` and `WithFilter`
  - add: before/after collapsible cuts under `## Problem` showing the pain of a DIY stdlib-to-ctx adapter vs. `NewStdLogBridge(..., WithFilter(...))`, and a second pair for log capturing in tests that highlights manual writer/log-level cleanup vs. `StartCapture` auto-restore
  - refresh: shifted line numbers in existing feature links (`WithContextAttrs`, `Ctx logging functions`, `sLogAttrsFromCtx`, `ILogCaptor`)

### Wiring in pkg/router

- [x] update: [pkg/router/consts.go](../../../pkg/router/consts.go)
  - add: package-level var `annoyingHTTPErrorsFilter = logger.WithFilter([]string{"TLS handshake error"})` reused across all server variants; `logger` import added
- [x] update: [pkg/router/impl_http.go](../../../pkg/router/impl_http.go)
  - update: in `httpServer.prepareBasicServer`, drop the `ErrorLog` assignment that currently wraps `log.Default().Writer()`; keep the rest of the `http.Server{...}` initialization
  - update: in `httpServer.preRun`, after `s.rootLogCtx` is built, set `s.server.ErrorLog = logger.NewStdLogBridge(s.rootLogCtx, "endpoint.http.error", annoyingHTTPErrorsFilter)` so that the per-server root context attributes (`vapp`, `extension`) propagate and TLS handshake noise is dropped inside the writer
  - applies to all four server variants (`sys._HTTPServer`, `sys._AdminHTTPServer`, `sys._HTTPSServer`, `sys._ACMEServer`) via shared `httpServer.preRun`
- [x] update: [pkg/router/utils.go](../../../pkg/router/utils.go)
  - remove: `annoyingErrorsFilter` type and its `Write` method; drop the `bytes` and `io` imports that become unused

### Tests

- [x] update: [pkg/goutils/logger/logger_test.go](../../../pkg/goutils/logger/logger_test.go)
  - add: subtests under `Test_NewStdLogBridge` using `StartCapture` that exercise the bridge via `*log.Logger` (`Println`/`Print`)
    - single-line write: one `ErrorCtx` entry with configured stage and ctx attributes
    - multi-line payload: one entry with embedded `\n` preserved in `msg` (asserted against the text-handler-escaped `msg="first\nsecond"` form); trailing CRLF trimmed; exactly one non-empty output line
    - payload that is empty after trimming (e.g. `"\r\n"`) is suppressed
    - level check: disabling `LogLevelError` suppresses forwarded writes
    - `WithFilter` drops entire Writes whose payload contains any configured substring and forwards the rest
- [x] update: [pkg/router/impl_test.go](../../../pkg/router/impl_test.go)
  - add: test that, after router service `Prepare` + `Run` startup, writing to `s.server.ErrorLog` is captured by voedger logger with `stage=endpoint.http.error` and `extension=sys._HTTPServer`, and that a line containing `TLS handshake error` is filtered out (no log entry emitted)
