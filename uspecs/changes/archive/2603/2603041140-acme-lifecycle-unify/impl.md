# Implementation plan: Unify acmeService lifecycle with HTTP/HTTPS services

## Type hierarchy refactor

- [x] update: [pkg/router/types.go](../../../pkg/router/types.go)
  - add: `httpServer` struct — infrastructure-only: `HTTPServerParams`, `listenAddress`, `server`, `listener`, `name`, `listeningPort`, `rootLogCtx`
  - rename: `httpService` → `routerService`; embed `httpServer` by value; add private route fields (`routeDefault`, `routes`, `routesRewrite`, `routeDomains`)
  - update: `httpsService` — embed `*routerService` instead of `*httpService`
  - update: `acmeService` — embed `httpServer` by value and add `handler http.Handler`; remove embedded `http.Server`

## Parameter hierarchy refactor

- [x] update: [pkg/router/types.go](../../../pkg/router/types.go)
  - add: `HTTPServerParams` struct — `Port`, `WriteTimeout`, `ReadTimeout`, `ConnectionsLimit`
  - update: `RouterParams` — embed `HTTPServerParams`; keep router-specific fields (`AdminPort`, `HTTP01ChallengeHosts`, `CertDir`, route fields)
  - update: `httpServer` — embed `HTTPServerParams` instead of `RouterParams`

## Lifecycle deduplication

- [x] update: [pkg/router/impl_http.go](../../../pkg/router/impl_http.go)
  - add: `httpServer.prepareBasicServer(handler http.Handler) error` — shared: binds listener, stores port, applies connection limit, creates `http.Server`
  - update: `routerService.Prepare()` — delegates to `prepareBasicServer(s.router)`
  - update: `routerService.Stop()` — calls `httpServer.Stop()` then drains n10n subscriptions
  - move: `Run`, `Stop`, `GetPort`, `preRun`, `log` receivers from `*httpService` to `*httpServer`
  - move: route/handler methods (`registerDebugHandlers`, `registerReverseProxyHandler`, etc.) receivers to `*routerService`
- [x] update: [pkg/router/impl_acme.go](../../../pkg/router/impl_acme.go)
  - update: `Prepare()` — calls `s.prepareBasicServer(s.handler)`, sets `s.server.ErrorLog` to `filteringLogger`
  - remove: `Run()` and `Stop()` — inherited from embedded `httpServer`

## Construction

- [x] update: [pkg/router/provide.go](../../../pkg/router/provide.go)
  - rename: `getHTTPService` → `getRouterService`; maps `RouterParams` route fields to private `routerService` fields
  - add: `getHTTPServer(name, addr, HTTPServerParams) httpServer` — used for ACME construction
  - update: `acmeService` constructed with `getHTTPServer` using `HTTPServerParams` with ACME timeouts; `filteringLogger` moved into `acmeService.Prepare()`
  - update: `httpsService` constructed with `routerService:` field

## Private route fields

- [x] update: [pkg/router/impl_reverseproxy.go](../../../pkg/router/impl_reverseproxy.go)
  - update: `s.Routes/RouteDefault/RoutesRewrite/RouteDomains` → lowercase private fields on `routerService`
- [x] update: [pkg/router/impl_n10n.go](../../../pkg/router/impl_n10n.go), [impl_blob.go](../../../pkg/router/impl_blob.go), [impl_apiv2.go](../../../pkg/router/impl_apiv2.go)
  - update: all method receivers `*httpService` → `*routerService`

## Downstream callers

- [x] update: [pkg/router/impl_test.go](../../../pkg/router/impl_test.go)
  - update: `RouterParams{}` literal uses nested `HTTPServerParams{}` syntax
- [x] update: [pkg/vvm/provide.go](../../../pkg/vvm/provide.go), [wire_gen.go](../../../pkg/vvm/wire_gen.go)
  - update: `provideRouterParams` uses nested `HTTPServerParams{}` syntax; `Port` moved into `HTTPServerParams`
