---
change_id: 2607021318-avoid-redundant-log-event-copy
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4383
domains: [prod]
scope: [storage]
---

# Change request: Avoid redundant log-event copy

Refs:

- [AIR-4383: istructsmem: avoid redundant log-event copy for drivers that return owned bytes](./issue-AIR-4383.md)

## Why

bbolt-backed storage can return byte slices backed by memory-mapped database pages, so retaining those slices after a read callback risks observing changed data when the underlying mapping changes. The current partial mitigation copies log-event bytes in `istructsmem`, which protects batched log reads but makes drivers that already return owned bytes pay an unnecessary extra copy.

## What

Symptom: Log-event reads avoid bbolt byte-retention hazards by copying in `istructsmem`, causing redundant copies for storage drivers whose read bytes are already owned.

```text
application reads PLog or WLog events
      |
      v
istructsmem ReadPLog/ReadWLog batch path
      |
      v
storage.Read callback returns driver-provided bytes
      |
      v
newStoredEvent copies bytes unconditionally        <-- fault: ownership fix is applied above the driver boundary
      |
      v
non-bbolt drivers pay an unnecessary second copy   (symptom)
```

Corrected behavior: bbolt read paths provide retain-safe bytes where needed, while `istructsmem` avoids redundant log-event copies for drivers that already return owned bytes.

## How

Decisions:

- Move byte ownership responsibility for bbolt reads to the bbolt storage adapter in `pkg/istorage/bbolt/impl.go`: `Read` and `TTLRead` copy both callback arguments (`ccols` and `viewRecord`), and `TTLGet` copies the returned data.
- Remove the unconditional log-event copy from the `istructsmem` batched log read path once driver-provided bytes are retain-safe where required.
- Keep the existing `ReadCallback` contract explicit: callers must not mutate callback bytes, and any driver-specific ownership strengthening must not weaken existing callback processing semantics.
- Preserve and clarify the regression coverage around batched log events owning their bytes.

Out of scope:

- Standardizing all storage drivers to return owned bytes for every read.
- Changing query or view-record consumers that process values inside the read callback.

References:

- [bbolt storage read paths](../../../../../pkg/istorage/bbolt/impl.go)
- [storage read callback contract](../../../../../pkg/istorage/interface.go)
- [istructsmem log read path](../../../../../pkg/istructsmem/impl.go)
- [stored event byte ownership helper](../../../../../pkg/istructsmem/event-types.go)
- [batched log-event regression test](../../../../../pkg/sys/it/bug_test.go)

## Construction

- [x] update: [sys/it/bug_test.go](../../../../../pkg/sys/it/bug_test.go)
  - add: brief arrange/act/assert comments to `TestBug_BatchedLogEventsMustOwnTheirBytes`
  - keep coverage focused on retained batched log events staying valid after storage read callbacks return

- [x] update: [bbolt/impl_test.go](../../../../../pkg/istorage/bbolt/impl_test.go)
  - add coverage proving `Read` and `TTLRead` copy both callback arguments (`ccols` and `viewRecord`), and `TTLGet` copies returned data, so retained slices remain stable after the bbolt transaction closes
  - cover both regular and TTL-backed read paths so future zero-copy regressions fail at the driver boundary

- [x] update: [bbolt/impl.go](../../../../../pkg/istorage/bbolt/impl.go)
  - copy both callback arguments (`ccols` and `viewRecord`) before exposing them through `Read` and `TTLRead`
  - copy returned data before exposing it through `TTLGet`
  - keep callback ordering and range/TTL filtering behavior unchanged

- [x] update: [istructsmem/impl.go](../../../../../pkg/istructsmem/impl.go)
  - replace the batched `ReadPLog` and `ReadWLog` use of `newStoredEvent` with direct event loading from the read bytes after the driver boundary guarantees stable bytes for the affected storage backend
  - keep single-event read behavior unchanged

- [x] update: [istructsmem/event-types.go](../../../../../pkg/istructsmem/event-types.go)
  - remove `newStoredEvent` when no remaining log read path needs unconditional copy-on-load
  - keep event buffer ownership and release semantics consistent with `eventType.Free`
