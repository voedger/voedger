# Context subsystem architecture: prod/routing/debug

Routing debug subsystem architecture: `net/http/pprof` handlers mounted by `registerDebugHandlers` on every `routerService` instance — both the loopback-only admin endpoint and the public HTTP/HTTPS listener — for runtime profiling and diagnostics. Context-level overview: [arch.md](./arch.md).

## External actors

Systems:

- `*Client`
  - Any caller able to open a TCP connection to either listener. The debug routes carry no authentication, role check, or method restriction, so every reachable caller can collect the same profiles. Restricting access on the public listener is an operator-side concern (network ACLs, upstream reverse proxy, or wrapping the listener with an authentication layer); the admin listener is restricted only by its loopback bind address (`httpu.LocalhostIP:AdminPort`).

## Scenarios overview

- **`Collect pprof profile`**
  - `*Client` opens a connection to either listener and reads any of the standard `pprof` handlers (`/debug/pprof`, `/debug/cmdline`, `/debug/profile`, `/debug/symbol`, `/debug/trace`, or `/debug/{cmd}` aliasing into `/debug/pprof/{cmd}`); the handler returns the corresponding profile.

## Components

### Layers

```text
External actors
    |
    +-- *Client
    |
    v
Entry points
    |
    +-- [Debug index]
    +-- [Debug cmdline]
    +-- [Debug profile]
    +-- [Debug symbol]
    +-- [Debug trace]
    +-- [Debug alias]
    |
    v
Cross-subsystem components
    |
    +-- [Admin endpoint]
    +-- [Public listener]
    +-- [Router (gorilla/mux)]
```

### Entry points

- `[Debug index]`
  - `pprof.Index` mounted at `/debug/pprof`. Lists the available profiles.
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

- `[Debug cmdline]`
  - `pprof.Cmdline` mounted at `/debug/cmdline`. Returns the process command line.
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

- `[Debug profile]`
  - `pprof.Profile` mounted at `/debug/profile`. Returns a CPU profile (default 30s).
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

- `[Debug symbol]`
  - `pprof.Symbol` mounted at `/debug/symbol`. Resolves program counters to function names.
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

- `[Debug trace]`
  - `pprof.Trace` mounted at `/debug/trace`. Returns an execution trace.
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

- `[Debug alias]`
  - `/debug/{cmd}` rewrites the request path to `/debug/pprof/{cmd}` and delegates to `pprof.Index`; registered last among the debug handlers. Allows shorter URLs for the per-name pprof handlers (heap, goroutine, block, mutex, threadcreate, allocs).
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

### Cross-subsystem components

- `[Admin endpoint]`
  - Shared with the rest of the routing context; see [arch.md](./arch.md#cross-subsystem-components). All debug handlers are reachable on `localhost:AdminPort`; restricted only by the loopback bind address.
  - Path to file: [pkg/router/provide.go](../../../../pkg/router/provide.go)

- `[Public listener]`
  - Shared with the rest of the routing context; defined in [arch-ingress.md](./arch-ingress.md#listeners). `registerDebugHandlers` is invoked from the same `routerService.Prepare` that runs on the public listener, so the same `/debug/*` routes are mounted there without authentication and are reachable by any `*Client`.
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

- `[Router (gorilla/mux)]`
  - Shared with the rest of the routing context; see [arch.md](./arch.md#cross-subsystem-components). `registerDebugHandlers` runs after `registerHandlersV1` / `registerHandlersV2` and before `registerReverseProxyHandler`, so debug routes take precedence over the reverse-proxy fallback on whichever listener they are mounted.
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

## Scenarios

### Collect pprof profile

```text
*Client curl http://localhost:55555/debug/pprof
  -> [Admin endpoint] -> [Router (gorilla/mux)] match "/debug/pprof"
  -> [Debug index] (pprof.Index) -> respond with profile listing

*Client curl http://localhost:55555/debug/profile?seconds=30
  -> [Admin endpoint] -> [Router (gorilla/mux)] match "/debug/profile"
  -> [Debug profile] (pprof.Profile) -> respond with CPU profile

*Client curl http://localhost:55555/debug/heap
  -> [Admin endpoint] -> [Router (gorilla/mux)] match "/debug/{cmd}"
  -> [Debug alias] rewrite to /debug/pprof/heap -> pprof.Index -> respond with heap profile

*Client curl https://<public-host>/debug/profile?seconds=30
  -> [Public listener] -> [Router (gorilla/mux)] match "/debug/profile"
  -> [Debug profile] (pprof.Profile) -> respond with CPU profile
       (same handler graph; no authentication is performed)
```

## Cross-cutting concerns

`registerDebugHandlers` is invoked from `routerService.Prepare` and is therefore part of the handler graph on every listener that runs a `routerService` — both the admin endpoint (loopback by construction: `httpu.LocalhostIP:AdminPort`) and the public HTTP/HTTPS listener. The debug routes are not gated by `[Query limiter]`, `[Request validator]`, authentication, or any other operator. Restricting external access to `/debug/*` on the public listener is an operator-side concern (network ACLs, upstream reverse proxy, or wrapping the listener with an authentication layer).
