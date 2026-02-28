# Implementation plan: Implement VerboseCtx and ErrorCtx in logger package

## Construction

- [x] create: [logger/consts.go](../../../pkg/goutils/logger/consts.go)
  - add: `LogAttr_VApp`, `LogAttr_Feat`, `LogAttr_ReqID`, `LogAttr_WSID`, `LogAttr_Extension` string constants
  - add: `logCtxSkipFrames` const
  - add: `ctxHandlerOpts` with `ReplaceAttr` to emit `VERBOSE`/`TRACE` instead of `DEBUG`/`DEBUG-4`
  - add: pre-built `slogOut`/`slogErr` globals
- [x] create: [logger/loggerctx.go](../../../pkg/goutils/logger/loggerctx.go)
  - add: unexported context key type and package-level key
  - add: `WithContextAttrs(ctx context.Context, name string, value any) context.Context`
  - add: `VerboseCtx`, `ErrorCtx`, `InfoCtx`, `WarningCtx`, `TraceCtx`
  - add: `SetCtxWriters(out, err io.Writer)` – replaces slog writers (tests only)
  - add: shared `logCtx` and `sLogAttrsFromCtx` internal functions
- [x] update: [logger/impl.go](../../../pkg/goutils/logger/impl.go)
  - change: `getFuncName` promoted from `(p *logPrinter)` method to package-level function (shared with loggerctx.go)
- [x] update: [logger/impl_test.go](../../../pkg/goutils/logger/impl_test.go)
  - update: call sites updated to use package-level `getFuncName`
- [x] update: [logger/logger_test.go](../../../pkg/goutils/logger/logger_test.go)
  - add: `captureCtxOutput` helper using `SetCtxWriters` for reliable pipe capture
  - add: `TestLoggerCtx_BasicUsage`, `Test_WithContextAttrs`, `Test_CtxFuncs_StandardAttrs`, `Test_CtxFuncs_LevelFiltering`, `Test_CtxFuncs_EmptyContext`
  - refactor: `TestMultithread` uses `captureCtxOutput` instead of manual pipe setup
