# Implementation plan: Add logging coverage tests for sync projectors

## Construction

- [x] update: [pkg/processors/actualizers/impl.go](../../../pkg/processors/actualizers/impl.go)
  - sync projector log context: `LogAttr_Extension` changed from bare `projector.Name` to `"p." + projector.Name.String()`
- [x] update: [pkg/appparts/internal/actualizers/actualizers.go](../../../pkg/appparts/internal/actualizers/actualizers.go)
  - async projector log context: `LogAttr_Extension` changed from bare `name` to `"ap." + name.String()`
- [x] update: [pkg/processors/actualizers/async_test.go](../../../pkg/processors/actualizers/async_test.go)
  - updated `extension=` assertions to expect `ap.` prefix
- [x] update: [pkg/processors/command/impl_test.go](../../../pkg/processors/command/impl_test.go)
  - extract `projExtension = "p." + projQName.String()` and `addStdRoles` helper shared across sub-tests
  - update: sub-test "sp.triggeredby and sp.success on success" — use `projExtension`, add `woffset`/`poffset`/`evqname` checks in per-projector logs, assert command-processor `sp.success` line (no `extension=p.` prefix), assert total `sp.success` count is 2; add warm-up `sendCUD` before `logCap.Reset()` to absorb partition recovery logs
  - update: sub-test "sp.error on invoke failure" — use `projExtension`, add `woffset`/`poffset`/`evqname` checks, assert command-processor `sp.error` line, assert total `sp.error` count is 2
  - add: sub-test "info level: verbose sync projector logs suppressed" — at info log level `sp.triggeredby` and per-projector/command-processor `sp.success` must not appear; `sp.error` must still be emitted at error level
