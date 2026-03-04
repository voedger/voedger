---
registered_at: 2026-03-04T10:19:38Z
change_id: 2603041019-acme-lifecycle-unify
baseline: d36ff04c9d60dbebdd5129eb498d46d0bbee21e0
---

# Change request: Unify acmeService lifecycle with HTTP/HTTPS services

## Why

`acmeService` has a different lifecycle from `httpService` and `httpsService`: its `Prepare()` is a no-op and its `Run()` both binds the listener and serves (via `ListenAndServe()`), while the HTTP/HTTPS services bind the listener in `Prepare()` and only serve in `Run()`. This inconsistency makes the codebase harder to reason about and maintain.

## What

Align `acmeService` lifecycle with `httpService`/`httpsService` so all three services are constructed and managed the same way:

- Change `acmeService` to embed `*httpService` (same as `httpsService`) instead of `http.Server` directly
- Construct `acmeService` via `getHTTPService()` in `Provide()`, same as `httpService`, `httpsService`, and `adminSrv`
- Override `acmeService.Prepare()` to bind the listener and create `http.Server` with the ACME handler — skip route registration since `acmeService` has no routes
- Override `acmeService.Run()` to call `server.Serve(listener)` instead of `ListenAndServe()`
- Remove the standalone `acmeService.Stop()` — delegate to the embedded `httpService.Stop()`
