---
registered_at: 2026-02-27T17:58:34Z
change_id: 2602271758-fix-bbolt-ttl-key-and-test
baseline: 96d70eed79c988e4645fac2c105e06b0a30875bd
archived_at: 2026-02-27T18:38:24Z
---

# Change request: Fix bbolt TTL key construction and background cleaner test

## Why

`makeTTLKey` allocates a slice with a non-zero length using `make([]byte, totalLength, ...)` and then calls `binary.BigEndian.AppendUint64`, which appends after the pre-allocated zero bytes instead of writing at the start. This produces malformed TTL keys with leading zeros, breaking TTL-based expiry entirely. `TestBackgroundCleaner` passes despite the bug because it verifies expiry via `TTLGet`, which reads the `ExpireAt` field embedded in the stored value rather than confirming the key was physically deleted from the bbolt database.

## What

Fix `makeTTLKey` so it constructs the TTL key correctly:

- Allocate slice with length 0 and the correct capacity (`make([]byte, 0, totalLength)`)
- Use `AppendUint64` to append `expireAt` and `len(pKey)` at the start of the slice

Fix `TestBackgroundCleaner` so it verifies the actual deletion:

- After the background cleaner runs, read the expired key directly from the bbolt data bucket using an internal `db.View` transaction
- Assert the key is absent in the raw bucket, not just via `TTLGet`
