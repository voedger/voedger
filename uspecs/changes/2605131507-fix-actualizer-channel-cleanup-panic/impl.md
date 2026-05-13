# Implementation plan: Fix double channel cleanup panic in async actualizer

## Construction

- [x] update: [async.go](../../../../../pkg/processors/actualizers/async.go)
  - fix: reset `a.channelCleanup = nil` after `finit()` in `Run` retry loop, alongside `a.pipeline = nil`, to prevent stale cleanup closure invocation when `init()` returns early before re-acquiring a channel
