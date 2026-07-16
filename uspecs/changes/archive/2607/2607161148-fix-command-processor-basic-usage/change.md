---
change_id: 2607161122-fix-command-processor-basic-usage
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4529
---

# Change request: Reliable command processor basic usage test

Refs:

- [AIR-4529: Fix command processor TestBasicUsage](./issue-AIR-4529.md)

## Why

The command processor basic usage test can fail in CI even when the command completes successfully. The test must tolerate the asynchronous ordering between response delivery and success logging so that it reports actual command processor regressions rather than timing-dependent failures.

## What

Symptom: `TestBasicUsage/basic_usage` intermittently fails because no captured log line contains `stage=cp.success`.

```text
CI runs command processor TestBasicUsage
      |
      v
the command processor sends the response
      |
      v
the test finishes consuming the response
      |
      v
impl_test.go calls logCap.HasLine immediately   <-- fault: does not wait for the subsequent success log
      |
      v
the command processor emits cp.success after response delivery
      |
      v
the test reports a missing cp.success log line   (symptom)
```

Corrected behavior: `TestBasicUsage` waits for the asynchronously emitted `cp.success` log entry and succeeds after a valid command execution.

## How

Decisions:

- Replace the immediate success-log assertion in `pkg/processors/command/impl_test.go` with the existing eventually-consistent log assertion
- Keep the command processor response and success-log ordering unchanged
- Reuse the logger capture helper's standard timeout and polling behavior without introducing test-specific synchronization

Out of scope:

- Changing command processor runtime behavior or log ordering
- Changing logger capture timeouts or polling intervals

References:

- [command processor basic usage test](../../../../../pkg/processors/command/impl_test.go)
- [eventual log assertion helper](../../../../../pkg/goutils/logger/logcapture.go)
- [command response and success logging flow](../../../../../pkg/processors/command/provide.go)

## Construction

- [x] update: [processors/command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - replace the immediate `HasLine` check for `stage=cp.success` with `EventuallyHasLine`
  - preserve the success-log assertion while allowing the command processor's asynchronous post-response logging to complete
