# AIR-3623: logger — implement bridge between goutils/logger and stdlib log.Logger

- **Key**: AIR-3623
- **Type**: Sub-task
- **Status**: In Progress
- **Assignee**: d.gribanov@dev.untill.com
- **URL**: <https://untill.atlassian.net/browse/AIR-3623>

## Why

Consumers of stdlib `*log.Logger` (most notably `http.Server.ErrorLog`, but also acme/autocert, third-party libraries and internal services) currently have no first-class way to forward their byte-oriented output into the voedger structured logger.

Consequences:

- Operational noise from low-value stdlib output (TLS handshake errors, pprof internals) cannot be filtered in a shared way. No attributes, no stages
- `src=...` attribute points at the adapter's `Write` method instead of the code that called `log.Println`, making logs harder to investigate
- No consistent way to express "give me a `*log.Logger` that writes into the voedger logger" — callers keep reinventing it
- Ad-hoc adapters that split multi-line payloads on `\n` fragment a single `net/http` panic stack trace into dozens of disconnected records, breaking alert counts and operator correlation

## What

Add a single entry point to `pkg/goutils/logger` that produces a stdlib `*log.Logger` forwarding each message to the voedger ctx logger at Error level, preserving ctx attrs from `WithContextAttrs` and keeping one stdlib `Write` = one log record.

### API

Add `NewStdErrorLogBridge(ctx context.Context, stage string, opts ...StdLogBridgeOption) *log.Logger`:

- Returns the exact type `http.Server.ErrorLog` expects; the underlying writer (unexported `stdLogBridgeWriter`, declared in `types.go`) is not exposed so it cannot be misused as a generic streaming sink
- Initializes the writer's `logLevel` field to `LogLevelError`; `Write` reads `w.logLevel` so the level is configurable per future constructor without rewriting `Write`
- Skips the entire `Write` when `w.logLevel` is disabled; otherwise trims trailing `\r`/`\n` bytes; suppresses payloads that are empty after trimming; forwards the remaining payload via `LogCtx(ctx, skipStackFrames, w.logLevel, stage, payload)` exactly once per `Write`
- Embedded newlines are kept verbatim in the message and escaped by slog's `TextHandler` to the two-char `\n` on output, so multi-line payloads (e.g. `net/http` panic stack traces) stay a single correlatable log event
- Contract: one `Write` call produces at most one log record; partial lines spanning multiple `Write` calls are not buffered (stdlib `*log.Logger` always delivers a complete message per call)

### Options

Add functional-option type `StdLogBridgeOption` and variadic option `WithFilter(substrings ...string)` that drops a `Write` whose payload contains any of the given substrings. Empty substrings are ignored; the remaining substrings are pre-converted to `[]byte` once and stored as `filters [][]byte` on the writer to avoid a `string->[]byte` conversion on every `Write`.

### Constants

Add unexported constant `stdLogBridgeSkipStackFrames = 3` in `consts.go` so the `src` attribute points at the caller of `*log.Logger.{Println,Print,Printf}`, not at an internal stdlib or bridge frame.

### Tests

Unit tests in `logger_test.go` (`Test_NewStdLogBridge`) covering:

- Single-line forwarding with stage and ctx attrs
- Multi-line payload produces one entry with embedded `\n` preserved in `msg` (asserted against the text-handler-escaped `msg="first\nsecond"` form); trailing CRLF trimmed; exactly one non-empty output line
- Payload that is empty after trimming (e.g. `"\r\n"`) is suppressed
- Disabled log level suppresses writes
- `WithFilter` drops entire `Write` calls whose payload contains any configured substring and forwards the rest

### Documentation

README updates:

- New feature entry under `## Features`
- Before/after examples under `## Problem` contrasting a DIY `io.Writer` adapter (CRLF trim + substring filter + `log.New` tuning + wrong src frame) against `NewStdErrorLogBridge(..., WithFilter(...))`
