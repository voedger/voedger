---
registered_at: 2026-02-20T10:53:47Z
change_id: 2602201053-http-storage-return-cleanup
baseline: 37f1a7f2d67a3f6ed5c7a6f05f5cb1b476b083b5
archived_at: 2026-02-20T11:57:58Z
---

# Change request: Return httpStorage cleanup to caller

## Why

`httpStorage` creates an `httpu.IHTTPClient` via `httpu.NewIHTTPClient()` and stores the returned `cleanup` function, but never exposes it. The cleanup is never called, which means the underlying HTTP client resources are leaked and never released on shutdown.

## What

Make `NewHTTPStorage` return the cleanup function to callers so the server can invoke it on shutdown:

- Update `NewHTTPStorage` to return both `state.IStateStorage` and `func()` (cleanup)
- Update all call sites (async actualizer, scheduler, query processor state providers) to capture and propagate the cleanup function
- Ensure the cleanup function is called during VVM shutdown
