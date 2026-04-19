---
registered_at: 2026-04-18T20:27:48Z
change_id: 2604182027-stdlib-log-bridge
baseline: 9eddaed6a696c253601bdaf175d58dc9e2f50c26
issue_url: https://untill.atlassian.net/browse/AIR-3623
archived_at: 2026-04-19T08:57:44Z
---


# Change request: logger bridge between goutils/logger and stdlib log.Logger

## Why

Consumers of stdlib `*log.Logger` (most notably `http.Server.ErrorLog`, but also acme/autocert, third-party libraries and internal services) currently have no first-class way to forward their byte-oriented output into the voedger structured logger, so operational noise cannot be filtered consistently, `src=` attribute points at the adapter instead of the real caller, and ad-hoc adapters split multi-line payloads into disconnected records.

See [issue.md](issue.md) for details.

## What

Add a single entry point in `pkg/goutils/logger` that produces a stdlib `*log.Logger` forwarding each message to the voedger ctx logger at Error level, preserving ctx attrs from `WithContextAttrs` and keeping one stdlib `Write` = one log record:

- `NewStdErrorLogBridge(ctx context.Context, stage string, opts ...StdLogBridgeOption) *log.Logger` returning the exact type `http.Server.ErrorLog` expects; the writer's level is held in an unexported `logLevel` field initialized to `LogLevelError`
- Unexported writer `stdLogBridgeWriter` (declared in `types.go`) not exposed so it cannot be misused as a generic streaming sink
- Trims trailing `\r`/`\n`, suppresses payloads empty after trimming, forwards remaining payload via `LogCtx(ctx, skipStackFrames, w.logLevel, stage, payload)` exactly once per `Write`
- Embedded newlines kept verbatim in the message so multi-line payloads (e.g. net/http panic stack traces) stay a single correlatable log event
- Contract: one `Write` call produces at most one log record; partial lines spanning multiple `Write` calls are not buffered

Add filtering and caller-frame plumbing:

- Functional-option type `StdLogBridgeOption` and option `WithFilter(substrings []string)` that drops a `Write` whose payload contains any of the given substrings
- Unexported constant `stdLogBridgeSkipStackFrames = 3` in `consts.go` so `src` attribute points at the caller of `*log.Logger.{Println,Print,Printf}`, not at an internal stdlib or bridge frame

Add unit tests in `logger_test.go` (`Test_NewStdLogBridge`) covering:

- Single-line forwarding with stage and ctx attrs
- Multi-line payload produces one entry with embedded `\n` preserved in `msg` (asserted against the text-handler-escaped `msg="first\nsecond"` form), trailing CRLF trimmed, exactly one non-empty output line
- Payload empty after trimming (e.g. `"\r\n"`) is suppressed
- Disabled log level suppresses writes
- `WithFilter` drops entire `Write` calls whose payload contains any configured substring and forwards the rest

Update README:

- New feature entry under `## Features`
- Before/after examples under `## Problem` contrasting a DIY `io.Writer` adapter (CRLF trim + substring filter + `log.New` tuning + wrong src frame) against `NewStdlibLogBridge(..., WithFilter(...))`
