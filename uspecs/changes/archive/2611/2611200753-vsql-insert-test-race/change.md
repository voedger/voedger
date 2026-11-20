---
change_id: 2611200703-vsql-insert-test-race
type: test
issue_url: https://untill.atlassian.net/browse/AIR-5309
domains: [prod]
scope: [observability, apps]
---

# Change request: VSQL direct-insert test race correction

Refs:

- [AIR-5309: fix data race in TestVSqlUpdate_BasicUsage_DirectInsert](./issue-AIR-5309.md)

## Why

The CI race detector reports that log-capture cleanup in the `basic` subtest of `TestVSqlUpdate_BasicUsage_DirectInsert` restores global context writers while an asynchronous actualizer reads them through `logCtx`. The test uses a shared VIT fixture whose background services remain active when the subtest finishes, making the direct-insert regression coverage fail under the race detector.

## What

- VSQL direct-insert integration coverage runs without capture setup or cleanup racing with background logging
- Successful unlogged insertion still verifies the inserted view fields and the VSQL update shim log contract
- Integration coverage retains rejection of incomplete keys and unknown fields with HTTP 400 and duplicate view keys with HTTP 409
- The test correction preserves production VSQL behavior and does not suppress race detection or remove the logging assertions

## How

Decisions:

- Keep the shared VIT fixture and make logger destination access safe during capture installation and restoration; test-scope ordering alone cannot stop the shared server's background log producers
- Synchronize destination snapshots and replacement consistently across context-aware logging, legacy logging, and capture cleanup, without holding configuration synchronization while invoking output writers
- Preserve existing capture restoration semantics, log formatting, and level filtering; concurrent logging is supported, while independently overlapping capture sessions remain outside the supported test model
- Validate with the existing logger tests and repeated race-enabled runs of the affected VSQL integration test, retaining its data, rejection, and shim-log assertions without additional concurrency scaffolding

Assumptions:

- Capture sessions are sequential or nested with cleanup in reverse installation order, rather than independently overlapping across parallel tests

Out of scope:

- Replacing shared VIT lifecycle management or isolating the test in a subprocess
- Redesigning capture as per-request or per-test log routing

References:

- [Direct-insert scenarios and shim-log assertions](../../../../../pkg/sys/it/impl_vsqlupdate_test.go)
- [Shared fixture acquisition and teardown](../../../../../pkg/vit/impl.go)
- [Shared server retention during cleanup](../../../../../pkg/vit/restart.go)
- [Capture installation and restoration](../../../../../pkg/goutils/logger/logcapture.go)
- [Context logger destination reads and replacement](../../../../../pkg/goutils/logger/loggerctx.go)

## Construction

- [x] Update: [goutils/logger/consts.go](../../../../../pkg/goutils/logger/consts.go)
  - Define one shared synchronization boundary for context and legacy destination access, preserving the default stdout and stderr destinations
- [x] Update: [goutils/logger/loggerctx.go](../../../../../pkg/goutils/logger/loggerctx.go)
  - Synchronize replacement by `SetCtxWriters` and destination selection by `logCtx` through the shared boundary
  - Release configuration synchronization before formatting or invoking the selected logger, preserving context attributes, levels, and formatting
- [x] Update: [goutils/logger/logger.go](../../../../../pkg/goutils/logger/logger.go)
  - Snapshot the selected legacy destination under the same synchronization and release it before invoking the writer
  - Preserve existing output and error routing and log-level behavior
- [x] Update: [goutils/logger/logcapture.go](../../../../../pkg/goutils/logger/logcapture.go)
  - Save and replace all capture destinations as one synchronized operation and restore them through the same boundary during cleanup
  - Preserve nested cleanup order and existing atomic log-level handling without recursively acquiring the configuration lock
  - Audit all context and legacy destination accesses in the logger package so no unsynchronized read or assignment remains
  - Run formatting and `go test -race ./pkg/goutils/logger`, then `go test -race ./pkg/sys/it -run '^TestVSqlUpdate_BasicUsage_DirectInsert$' -count=20`
  - Retain the existing direct-insert test's view-value, rejection, and shim-log assertions
