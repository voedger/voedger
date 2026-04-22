# Implementation plan: Mockable legacy logger functions

## Construction

- [x] update: [pkg/goutils/logger/logger.go](../../../pkg/goutils/logger/logger.go)
  - add: `legacyOut` and `legacyErr` package-level `io.Writer` vars (initially `os.Stdout` / `os.Stderr`)
  - update: `DefaultPrintLine` to write to `legacyOut`/`legacyErr` instead of `os.Stdout`/`os.Stderr` directly

- [x] update: [pkg/goutils/logger/logcapture.go](../../../pkg/goutils/logger/logcapture.go)
  - update: `StartCapture` to also save `legacyOut`/`legacyErr`, redirect both to the captor, and restore originals in `t.Cleanup` (mirroring the existing `slogOut`/`slogErr` save-restore)

- [x] update: [pkg/goutils/logger/logger_test.go](../../../pkg/goutils/logger/logger_test.go)
  - removed: `Test_BasicUsage`, `Test_BasicUsage_SetLogLevelWithRestore`, `Test_BasicUsage_SkipStackFrames`, `TestLoggerCtx_BasicUsage` (demonstration-, no assertions)
  - removed: `captureCtxOutput` helper and `testingu`/`os` imports (replaced by `StartCapture` everywhere)
  - add: `TestLegacyFuncs_BasicUsage` — calls `Verbose`, `Info`, `Warning`, `Error`; asserts each message is captured with `HasLine`
  - add: `TestSlogFuncs_BasicUsage` — calls `VerboseCtx`, `InfoCtx`, `ErrorCtx`, `LogCtx` with context attrs; asserts message and attrs with `HasLine`
  - converted: `Test_SetLogLevelWithRestore`, `Test_BasicUsage_CustomPrintLine`, `Test_SkipStackFrames`, `Test_StdoutStderr_LogLevel`, `TestMultithread` to use `StartCapture`
  - converted: `Test_WithContextAttrs`, `Test_CtxFuncs_StandardAttrs`, `Test_CtxFuncs_SLogLevels`, `Test_CtxFuncs_LevelFiltering`, `Test_CtxFuncs_EmptyContext` to use `StartCapture`
  - converted: `Test_CheckSetLevels` to a table-driven test using `activeIdx` to derive cumulative level expectations
  - converted: `Test_CtxFuncs_SLogLevels` to table-driven, dropping `wantStdErr` field (stream routing is implicit in `level=`/`msg=` assertions)

- [x] update: [pkg/goutils/logger/logcapture_test.go](../../../pkg/goutils/logger/logcapture_test.go)
  - add: `TestLegacyFunctions` as a table-driven test; each row carries `captureLevel`, `logFn`, `wantLine []string`, and `notContains` flag; covers all five legacy levels, level-based suppression, and error-prefix verification

- [x] update: [pkg/goutils/logger/README.md](../../../pkg/goutils/logger/README.md)
  - updated: description to mention both legacy and slog-based logging modes
  - updated: all architecture-point line numbers after code changes shifted them
  - updated: Use section to reference `TestLegacyFuncs_BasicUsage` and `TestSlogFuncs_BasicUsage`
