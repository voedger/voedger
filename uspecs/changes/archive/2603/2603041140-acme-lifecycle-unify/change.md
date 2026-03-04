---
registered_at: 2026-03-04T10:19:38Z
change_id: 2603041019-acme-lifecycle-unify
baseline: d36ff04c9d60dbebdd5129eb498d46d0bbee21e0
archived_at: 2026-03-04T11:40:06Z
---

# Change request: Refactor pkg/router service and parameter hierarchy

## Why

`pkg/router` had several structural problems:

- `acmeService` had a different lifecycle: `Prepare()` was a no-op and `Run()` both bound the listener and served, while `httpService` bound the listener in `Prepare()` and only served in `Run()`
- `httpService` was a monolith mixing TCP/HTTP infrastructure with application-level routing logic
- `RouterParams` mixed basic server settings (timeouts, port) with router-specific config (routes, cert dir), making it impossible to construct an ACME server without passing irrelevant route fields
- Route configuration fields were public on `RouterParams` but accessed directly on the struct, leaking internals

## What

Refactor `pkg/router` to separate concerns at both the type and parameter level:

- Split `httpService` into `httpServer` (TCP/HTTP infrastructure) and `routerService` (application routing); `acmeService` and `routerService` embed `httpServer` by value, `httpsService` embeds `*routerService`
- Extract `prepareBasicServer` onto `httpServer` to eliminate duplicated listener/server init between `routerService` and `acmeService`
- Split `RouterParams` into `HTTPServerParams` (`Port`, `WriteTimeout`, `ReadTimeout`, `ConnectionsLimit`) and `RouterParams` (embeds `HTTPServerParams`, adds `AdminPort`, `CertDir`, route fields)
- Move route fields (`RouteDefault`, `Routes`, `RoutesRewrite`, `RouteDomains`) to private fields on `routerService`; `acmeService` is constructed with `HTTPServerParams` only
- Update `pkg/vvm` callers to use the new nested struct syntax
