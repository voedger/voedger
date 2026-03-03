# Implementation plan: Implement VerboseCtx and ErrorCtx in logger package

## Construction

- [x] create: [logger/consts.go](../../../pkg/goutils/logger/consts.go)
  - add: `LogAttr_App`, `LogAttr_Feat`, `LogAttr_ReqID`, `LogAttr_WSID`, `LogAttr_Extension` string constants
  - add: `logCtxSkipFrames` const
  - add: `ctxHandlerOpts` with `ReplaceAttr` to emit `VERBOSE`/`TRACE` instead of `DEBUG`/`DEBUG-4`
  - add: pre-built `slogOut`/`slogErr` globals
- [x] create: [logger/types.go](../../../pkg/goutils/logger/types.go)
  - add: `ctxKey` unexported context key type
  - add: `logAttrs` struct with `attrs map[string]any` and `parent *logAttrs` for linked-list chain
- [x] create: [logger/loggerctx.go](../../../pkg/goutils/logger/loggerctx.go)
  - add: `WithContextAttrs(ctx context.Context, attrs map[string]any) context.Context` – O(1), prepends a new chain node
  - add: `VerboseCtx`, `ErrorCtx`, `InfoCtx`, `WarningCtx`, `TraceCtx`
  - add: `SetCtxWriters(out, err io.Writer)` – replaces slog writers (tests only)
  - add: shared `logCtx` and `sLogAttrsFromCtx`; the latter walks the parent chain newest→oldest, first-seen-wins per key
- [x] update: [logger/impl.go](../../../pkg/goutils/logger/impl.go)
  - change: `getFuncName` promoted from `(p *logPrinter)` method to package-level function (shared with loggerctx.go)
- [x] update: [logger/impl_test.go](../../../pkg/goutils/logger/impl_test.go)
  - update: call sites updated to use package-level `getFuncName`
- [x] update: [logger/logger_test.go](../../../pkg/goutils/logger/logger_test.go)
  - add: `captureCtxOutput` helper using `SetCtxWriters` for reliable pipe capture
  - add: `TestLoggerCtx_BasicUsage`, `Test_WithContextAttrs`, `Test_CtxFuncs_StandardAttrs`, `Test_CtxFuncs_LevelFiltering`, `Test_CtxFuncs_EmptyContext`
  - refactor: `TestMultithread` uses `captureCtxOutput` instead of manual pipe setup
  - update: `WithContextAttrs` call sites use `map[string]any{...}` argument
- [x] update: [logger/README.md](../../../pkg/goutils/logger/README.md)
  - update: synopsis, problem statement, and example to cover context-aware logging
