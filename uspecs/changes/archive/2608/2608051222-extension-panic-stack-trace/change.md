---
change_id: 2608051132-extension-panic-stack-trace
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-4660
domains: [prod]
scope: [apps, extensions]
---

# Change request: Stack traces for extension panics

Refs:

- [AIR-4660: extengine: provide the stack on extension panic](./issue-AIR-4660.md)

## Why

Extension panic logs currently report the panic without enough context to locate the failing application code. Stack context makes extension failures in the Voedger application platform diagnosable.

## What

Extension panic diagnostics in the Voedger application platform provide actionable failure context:

- Panic log messages include the stack trace from the failed extension execution.
- Panic log messages continue to identify the extension and panic value alongside the stack trace.

## How

Decisions:

- Capture the current goroutine stack at the built-in extension engine's existing panic-recovery boundary, before the panic stack unwinds, and append it to the recovered error.
- Preserve the existing panic-error prefix and error-return recovery contract; existing processors remain responsible for logging, retry, and response behavior instead of the extension engine emitting a duplicate log record.
- Keep the stack in the same error message so processor-specific stages and context attributes remain attached to one correlated log event.
- Verify stable diagnostic content: the extension identity, panic value, and an extension call-path frame, without binding tests to generated line numbers or the complete runtime stack.

Assumptions:

- Existing consumers treat the extension identity and panic value as the compatibility boundary and do not require built-in extension panic errors to remain single-line.

Out of scope:

- Changing stack extraction or panic formatting for WASM guest runtimes, which use a separate extension-engine error path.

References:

- [built-in extension panic recovery](../../../../../pkg/iextengine/builtin/impl.go)
- [extension invocation boundary](../../../../../pkg/appparts/impl_app.go)
- [scheduler error propagation](../../../../../pkg/processors/schedulers/impl_scheduler.go)
- [extension panic regression coverage](../../../../../pkg/iextengine/builtin/impl_test.go)
- [processor logging contract](../../../../../uspecs/specs/prod/apps/logging--td.md)

## Technical design

- [x] update: [apps/logging--td.md](../../../../specs/prod/apps/logging--td.md)
  - add: built-in extension panic errors retain the extension identity and panic value and append the current goroutine stack before recovery unwinds
  - clarify: existing processor error stages and context attributes remain responsible for logging the enriched error as one correlated log event
  - clarify: WASM guest-runtime panic extraction and formatting remain unchanged

## Construction

- [x] update: [schedulers/impl_test.go](../../../../../pkg/processors/schedulers/impl_test.go)
  - add: a named panicking scheduler extension and connect the mock app partition to a real built-in extension engine invocation
  - add: an integration test that runs the scheduler and verifies one `job.error` record contains the panic prefix, stack markers, extension frame, and structured `vapp`, `wsid`, and `extension` attributes
  - add: print the captured `job.error` record through `t.Logf` so `go test -v` shows the actual structured log with attributes

- [x] update: [builtin/impl_test.go](../../../../../pkg/iextengine/builtin/impl_test.go)
  - add: a named panicking extension fixture so its function name provides a stable call-path marker in the captured stack
  - update: `Test_Panics` to invoke the named fixture, capture the returned error once, and preserve coverage of the `extension test.ext1 panic: boom` prefix and panic value
  - add: verify current-goroutine stack markers and the named fixture frame without matching generated line numbers or the complete runtime stack
  - add: print the recovered panic error during the test for direct diagnostic inspection

- [x] update: [builtin/impl.go](../../../../../pkg/iextengine/builtin/impl.go)
  - update: the existing deferred panic recovery to capture the current goroutine stack at the recovery boundary and append it on a new line after the existing extension identity and panic value
  - preserve: the error-return recovery contract and absence of engine-level logging; leave the WASM engine path unchanged

- [x] run: `go test ./pkg/iextengine/builtin`
  - verify: built-in extension behavior and panic diagnostics pass at package scope

- [x] run: `go test ./pkg/iextengine/builtin ./pkg/processors/schedulers`
  - verify: built-in panic recovery and scheduler structured-log integration pass together at package scope
