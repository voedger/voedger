# How: Stdlib log.Logger bridge in pkg/goutils/logger

## Approach

- Add a new file `pkg/goutils/logger/stdlogbridge.go` that exposes `NewStdErrorLogBridge(ctx, stage, opts...) *log.Logger` and keeps the writer unexported
  - The unexported `stdLogBridgeWriter` (declared in `types.go` next to other package types) stores `ctx`, `stage`, `logLevel`, and `filters []string`; its `Write([]byte)` is the single point that converts one stdlib message into one structured record
  - `NewStdErrorLogBridge` initializes `logLevel: LogLevelError`; the field is read by `Write` so the level is configurable per future constructor without rewriting `Write`
  - Inside `Write`: apply `WithFilter` substring check first (early return with `len(p), nil` so stdlib sees a successful write), then trim trailing `\r`/`\n` with `bytes.TrimRight(p, "\r\n")`, return if empty, otherwise call `logger.LogCtx(ctx, stdLogBridgeSkipStackFrames, w.logLevel, stage, string(trimmed))` exactly once
  - Construct the returned logger as `log.New(w, "", 0)` so stdlib adds no prefix/flags and the slog `TextHandler` fully owns timestamp/level/src formatting
- Wire the caller-frame constant in `consts.go`
  - Add `stdLogBridgeSkipStackFrames = 3`, positioned next to `logCtxSkipFrames`; value chosen so `getFuncName` (called from `logCtx`) skips the bridge's `Write`, stdlib `*log.Logger.output`, and stdlib `Println/Printf/Print` frame, landing on the user code that invoked the stdlib logger
  - The exact frame count is verified by the unit test asserting the `src=` attribute in the captured output
- Functional options
  - Define exported `type StdLogBridgeOption func(*stdLogBridgeWriter)` in `types.go`; implement `WithFilter(substrings []string) StdLogBridgeOption` in `stdlogbridge.go` that appends to the writer's filters
  - Keep the writer type unexported so options are the only way to mutate its state, preserving the "cannot be misused as a generic streaming sink" invariant
- Tests live in `logger_test.go` as `Test_NewStdLogBridge` using the existing `StartCapture` helper (see `logcapture.go`) with `t.Run` subtests for the five cases from the issue
  - Use `StartCapture(t, LogLevelNone)` for the "disabled level suppresses writes" subtest (level is auto-restored on cleanup)
  - Assert the text-handler-escaped form `msg="first\nsecond"` verbatim for the multi-line case; assert `src=` points at the test function for the single-line case
- README updates in `pkg/goutils/logger/README.md`
  - Add feature bullet under `## Features` with links into `stdlogbridge.go`
  - Add a new `<details>` pair under `## Problem` showing the DIY `io.Writer` adapter (CRLF trim, substring skip, `log.New` tuning, and the resulting wrong `src`) versus the one-liner `http.Server{ErrorLog: logger.NewStdErrorLogBridge(ctx, "http", logger.WithFilter([]string{"TLS handshake error"}))}`

References:

- [pkg/goutils/logger/loggerctx.go](../../../../../pkg/goutils/logger/loggerctx.go)
- [pkg/goutils/logger/consts.go](../../../../../pkg/goutils/logger/consts.go)
- [pkg/goutils/logger/types.go](../../../../../pkg/goutils/logger/types.go)
- [pkg/goutils/logger/stdlogbridge.go](../../../../../pkg/goutils/logger/stdlogbridge.go)
- [pkg/goutils/logger/logger.go](../../../../../pkg/goutils/logger/logger.go)
- [pkg/goutils/logger/logger_test.go](../../../../../pkg/goutils/logger/logger_test.go)
- [pkg/goutils/logger/README.md](../../../../../pkg/goutils/logger/README.md)
