# Context subsystem architecture: prod/routing/debug

Routing debug subsystem architecture: `net/http/pprof` handlers mounted on the admin endpoint (and, for compatibility, also on the public router) for runtime profiling and diagnostics. Context-level overview: [arch.md](./arch.md).

## External actors

Roles:

- `@Admin`
  - Operator collecting CPU / heap / goroutine / trace profiles for diagnostics over loopback.

## Scenarios overview

- **`Collect pprof profile`**
  - The operator opens a loopback connection to the admin endpoint and reads any of the standard `pprof` handlers (`/debug/pprof`, `/debug/cmdline`, `/debug/profile`, `/debug/symbol`, `/debug/trace`, or `/debug/{cmd}` aliasing into `/debug/pprof/{cmd}`); the handler returns the corresponding profile.

## Components

### Layers

```text
External actors
    |
    +-- @Admin
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
  - Shared with the rest of the routing context; see [arch.md](./arch.md#cross-subsystem-components). All debug handlers are reachable on `localhost:AdminPort`.
  - Path to file: [pkg/router/provide.go](../../../../pkg/router/provide.go)

- `[Router (gorilla/mux)]`
  - Shared with the rest of the routing context; see [arch.md](./arch.md#cross-subsystem-components). `registerDebugHandlers` runs after `registerHandlersV1` / `registerHandlersV2` and before `registerReverseProxyHandler`, so debug routes take precedence over the reverse-proxy fallback.
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

## Scenarios

### Collect pprof profile

```text
@Admin curl http://localhost:55555/debug/pprof
  -> [Admin endpoint] -> [Router (gorilla/mux)] match "/debug/pprof"
  -> [Debug index] (pprof.Index) -> respond with profile listing

@Admin curl http://localhost:55555/debug/profile?seconds=30
  -> [Admin endpoint] -> [Router (gorilla/mux)] match "/debug/profile"
  -> [Debug profile] (pprof.Profile) -> respond with CPU profile

@Admin curl http://localhost:55555/debug/heap
  -> [Admin endpoint] -> [Router (gorilla/mux)] match "/debug/{cmd}"
  -> [Debug alias] rewrite to /debug/pprof/heap -> pprof.Index -> respond with heap profile
```

## Cross-cutting concerns

`registerDebugHandlers` runs unconditionally in `routerService.Prepare`, so the same `/debug/*` routes are also registered on the public endpoint without authentication. The admin endpoint is loopback-only by construction (`httpu.LocalhostIP:AdminPort`); exposing the debug routes on the public listener safely is therefore an operator-side concern (network ACLs).
