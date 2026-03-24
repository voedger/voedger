# Decisions: Mockable legacy logger functions

## Interception hook for legacy functions

Add `legacyOut`/`legacyErr` `io.Writer` vars to `DefaultPrintLine`; redirect them in `StartCapture` (confidence: high)

Rationale: `DefaultPrintLine` currently writes directly to `os.Stdout`/`os.Stderr`. Adding package-level `legacyOut` and `legacyErr` vars (initially `os.Stdout`/`os.Stderr`) and having `DefaultPrintLine` write to them instead is the exact same pattern already used for `slogOut`/`slogErr`. `StartCapture` then saves/redirects these vars alongside the slog writers. `PrintLine` itself is never swapped, so the public API is unchanged and no `atomic.Pointer` machinery is needed

Alternatives:

- Replace `PrintLine` with a closure that writes to the captor buffer (confidence: medium)
  - Intercepts even custom `PrintLine` replacements; but forces `atomic.Pointer` on `PrintLine` (breaking `logger.PrintLine = myFunc` direct assignments) and requires two new exported helpers (`SetPrintLine` / `getPrintLine`) — more API surface for no practical gain in a test context where `DefaultPrintLine` is always active
- Replace `printIfLevel` with a function variable (confidence: low)
  - Would intercept at a higher level but shifts `globalSkipStackFramesCount` by +1, breaking caller info in all log lines; requires adjusting the constant or adding a skip-frame parameter

## Unified vs separate capture buffer

Single buffer captures both legacy (`PrintLine`) and ctx (slog) output (confidence: high)

Rationale: `HasLine`, `NotContains`, `EventuallyHasLine` operate on a single `bytes.Buffer`; merging both streams means tests can assert on any log call without knowing which subsystem produced it, which matches real test requirements

Alternatives:

- Separate captors for legacy and ctx output (confidence: medium)
  - Cleaner split but forces tests to maintain two captors, doubling boilerplate; most callers just want "did something get logged"

## Thread-safety of writer vars

Follow the same non-atomic pattern used by `slogOut`/`slogErr` (confidence: high)

Rationale: `slogOut` and `slogErr` are plain package-level vars that `StartCapture` saves and restores without mutexes or atomics; `legacyOut`/`legacyErr` follow the identical pattern. Tests that call `StartCapture` are expected to run sequentially (or each have their own isolated goroutine); the swap window is within `t.Cleanup` which `testing` serialises. Adding atomics here would be inconsistent with the existing slog vars and adds complexity without a concrete benefit

Alternatives:

- `atomic.Pointer` for `legacyOut`/`legacyErr` (confidence: medium)
  - Race-free but inconsistent with `slogOut`/`slogErr` and unnecessary for the sequential-test use case

## Stack frame accuracy under the new hook

Intercept at `PrintLine` level (after `getFuncName` is called) (confidence: high)

Rationale: `logPrinter.print` calls `getFuncName` first, formats the line, then calls `PrintLine`. Replacing `PrintLine` therefore has zero effect on the reported caller file/line, so `globalSkipStackFramesCount` stays at 4 and no existing tests break

Alternatives:

- Intercept at `printIfLevel` level (confidence: low)
  - Requires bumping `globalSkipStackFramesCount` to 5, or threading an extra skip parameter; fragile because the constant is also used by `logger.Log`

## `StartCapture` extension strategy

Extend the existing `StartCapture` function to hook both `PrintLine` and slog writers (confidence: high)

Rationale: `StartCapture` already saves/restores `slogOut`/`slogErr` inside `t.Cleanup`; adding save/restore of `PrintLine` in the same function keeps the API minimal — one call captures everything. Existing call sites require no changes

Alternatives:

- New `StartLegacyCapture` function (confidence: medium)
  - Explicit separation is readable but forces callers to call two capture functions when they want to assert on mixed log output; also splits `Cleanup` registration across two helpers
- `MockLogger` struct implementing a full logger interface (confidence: low)
  - Over-engineered; the package uses package-level functions, not an interface, so a mock struct would require changing all call sites

## Note: unifying `Verbose()` and `VerboseCtx()` under slog

The question is whether `Verbose()` (and the other legacy functions) should delegate to the slog pipeline instead of the `printIfLevel` → `PrintLine` pipeline, while emitting the same textual format as today

Pros:

- Single capture path: `StartCapture` would only need to swap the slog writers; the `PrintLine` hook and all thread-safety work for it become unnecessary
- Structured output from day one: legacy callers would automatically benefit from slog's handler ecosystem (JSON handler, custom handlers, etc.)
- `logCtx` already handles `getFuncName` with a configurable `skipStackFrames`; the legacy functions could reuse it with `context.Background()` and a matching skip count

Cons:

- **Breaking log format**: the current legacy format is `01/02 15:04:05.000: ---: [pkg.FuncName:line]: msg`; slog TextHandler produces `time=... level=DEBUG msg="..." src=pkg.FuncName:line`; tests and external log parsers that rely on the old format (prefixes `*****`, `!!!`, `===`, `---`, `...`) would break
- **`Test_StdoutStderr_LogLevel`** and **`Test_CheckRightPrefix`** explicitly assert on stderr/stdout split and prefix strings; they would need rewriting
- **`logger.Log(skipStackFrames, level, ...)`** exposes a custom skip-frames parameter; mapping this cleanly through slog's `Handler.Handle` source-attribution requires storing the adjusted PC, which is non-trivial
- **`PrintLine` public var** is part of the public API and is actively used by callers (e.g., `Test_BasicUsage_CustomPrintLine`); removing or deprecating it is a separate breaking change
- Scope: doing this correctly is a larger refactor than the current change request; it is a follow-up candidate, not a prerequisite

Recommendation: keep it as a follow-up. For this change, hook `PrintLine` as decided above. Once the codebase is comfortable with slog, a separate change can migrate legacy functions one level at a time, letting the format change be explicit and tested independently
