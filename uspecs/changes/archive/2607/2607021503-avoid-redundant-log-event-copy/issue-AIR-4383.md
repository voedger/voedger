# istructsmem: avoid redundant log-event copy for drivers that return owned bytes

- URL: https://untill.atlassian.net/browse/AIR-4383
- ID: AIR-4383
- State: In Progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov

## Why

There are cases when data read from bbolt is stored without copying and then modified by bbolt.

### Common problem

The bbolt engine maps the entire database file into memory (`unix.mmap`), then on read returns a slice of that memory. Save it without copy and then use it on, e.g., events batch handling in async projector, and stored data could be changed due to database file growth and re-allocation.

### Examples of problematic places in driver

- `bbolt.TTLGet`: returns slice without copy
- `bbolt.Read`: calls the callback with slice without copy

### Impacted non-driver places

#### ViewRecords().Read

https://github.com/voedger/voedger/blob/main/pkg/istructsmem/viewrecords-types.go#L219-L234

`valRow.loadFromBytes(value)` wraps the zero-copy `value` into the value's dynobuffer without copying, then calls `cb(recKey, valRow)`. This is safe only because every consumer (query2 handlers, sqlquery, `sys/workspace`) processes the `IValue` inside the callback. If any future consumer appends the `IValue` to a slice and uses it after `Read` returns, it reproduces this bug exactly. It is not currently a defect, but it is the same shape and undefended.

#### istoragecache

`TTLRead()`, `Read`

Pure pass-through to the underlying driver, so with bbolt they forward the same zero-copy views; they inherit whatever contract the caller honors.

### Partial solution

https://untill.atlassian.net/browse/AIR-4355

Explicit copy of memory got from bbolt is done for `ReadToTheEnd` branch of `ReadPLog`/`ReadWLog`.

Cons: copy is redundant for `mem` and `Scylla` drivers.

## What

- Revert solution made in https://github.com/voedger/voedger/commit/2895c96efce8e392c8f0945065d28b688ea77720 (always copy on `istructsmem` level).
- Always copy inside the bbolt driver only.
- Update `TestBug_BatchedLogEventsMustOwnTheirBytes`: add brief plan, add comments describing what we are doing.

Pros:

- Non-bbolt drivers untouched; `mem` stops double-copying on log reads.
- One place closes all bbolt retention hazards at once (`Read`, `TTLRead`, `TTLGet`, `ViewRecords`, and any future consumer).

Cons:

- bbolt now copies on every read, including the majority that process in callback and never retain (view-record scans during query processing, startup metadata, TTL cleanup scans). This is a broad, unconditional slowdown that throws away bbolt's zero-copy mmap benefit. In bbolt-dominant single-node/edge deployments this aggregate cost is very likely larger than the current narrow log-path copy.
- The driver cannot know the consumer's retention intent, so it must copy pessimistically for everyone.
- It silently changes the effective `ReadCallback` contract for one driver (bytes become retain-safe) while others still say "temporary", which is inconsistent unless "reads return owned bytes" is standardized across all drivers and `interface.go` is updated. At that point `newStoredEvent` and other copies become dead weight, but every bbolt read pays.
