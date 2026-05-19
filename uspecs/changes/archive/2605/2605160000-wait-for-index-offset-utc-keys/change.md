---
registered_at: 2026-05-15T23:53:26Z
change_id: 2605152353-wait-for-index-offset-utc-keys
baseline: 59247f436a281fd98e8d1cde3265b2fda7063bc9
issue_url: https://untill.atlassian.net/browse/AIR-3952
archived_at: 2026-05-16T00:00:23Z
---

# Change request: Align WaitForIndexOffset polling key with UTC-keyed WLogDates view

## Why

`pkg/sys/it/test_utils.go:WaitForIndexOffset` builds its `Year` / `DayOfYear` filter from `vit.Now()` in the host's local timezone, but `wLogDatesProjector` writes the view keys in UTC. On non-UTC hosts, around the day boundary the helper polls a key the projector never wrote, `vit.WaitFor` exhausts its 10-second budget and returns nil, and the next `SectionRow(0)[0].(string)` panics with a nil-pointer dereference. GitHub-hosted runners are UTC, so the flake is invisible in CI and reproduces only on developer machines.

## What

Make the test helper read the view with the same timezone the projector uses to write it, so `TestBasicUsage_Journal` (and every other test relying on `WaitForIndexOffset`) is stable on developer machines in any timezone.

- Compute `Year` and `DayOfYear` in `WaitForIndexOffset` from `vit.Now().UTC()`

## Construction

- [x] update: [sys/it/test_utils.go](../../../../../pkg/sys/it/test_utils.go)
  - update: `WaitForIndexOffset` to derive `Year` and `DayOfYear` from `vit.Now().UTC()` so the polling key matches the UTC keys written by `wLogDatesProjector`

- [x] Review
