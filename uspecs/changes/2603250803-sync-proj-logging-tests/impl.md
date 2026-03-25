# Implementation plan: Add logging coverage tests for sync projectors

## Construction

- [x] update: [pkg/processors/command/impl_test.go](../../../pkg/processors/command/impl_test.go)
  - update: sub-test "sp.triggeredby and sp.success on success" — fix `extension=sys.TestProj` → `extension=p.sys.TestProj` to verify the `p.` prefix, add checks for `woffset`, `poffset`, `evqname` attributes in per-projector logs, add check for command-processor level `sp.success` (no `extension=p.` prefix, contains `woffset`/`poffset`/`evqname`)
  - update: sub-test "sp.error on invoke failure" — fix `extension=sys.TestProj` → `extension=p.sys.TestProj`, add check for command-processor level `sp.error` (no `extension=p.` prefix, contains error message and `woffset`/`poffset`/`evqname`), verify `woffset`/`poffset`/`evqname` in per-projector logs
