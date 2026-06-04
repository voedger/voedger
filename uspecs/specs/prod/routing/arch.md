# Context architecture: prod/routing

Routing context architecture for the HTTP boundary of voedger: terminating client connections, dispatching API v1 and v2 requests, streaming BLOB transfers and N10N notifications, provisioning HTTPS certificates, forwarding configured paths and hosts to upstream services, and exposing the localhost-only admin endpoint used by bootstrap and operators.

## External actors

Roles:

- `@Admin`
  - Supplies cluster-wide operator configuration at VVM startup (reverse-proxy route table, ACME-managed hostname list, query limit, ports, TLS / proxy-protocol switches) via `RouterParams`.

Systems:

- `*Client`
  - External caller invoking commands, queries, BLOB transfers, and N10n SSE subscriptions over HTTP/HTTPS.

- `@ACME`
  - ACME directory (e.g. Let's Encrypt) issuing TLS certificates via the HTTP-01 challenge served on port 80.

## Scenarios overview

- **`Serve client request`**
  - Public listener accepts a TLS connection, the gorilla/mux router matches the request against API v1 / API v2 / BLOB / N10N / reverse-proxy routes, applies CORS and the per-workspace query limit where applicable, and dispatches to the matching subsystem.

- **`Provision TLS certificate`**
  - When the public listener runs on `HTTPSPort`, the autocert manager fetches and renews certificates from `@ACME` for the operator-supplied hostname whitelist; the HTTP-01 challenge listener on port 80 answers the validation requests. Details: [arch-tls.md](./arch-tls.md).

- **`Forward to upstream`**
  - The reverse-proxy matcher (registered last on the public router) inspects the request host and path and forwards unmatched requests to the configured upstream (path-prefix, path-prefix with rewrite, host-based domain, or default fallback). Details: [arch-reverse-proxy.md](./arch-reverse-proxy.md).

- **`Serve bootstrap and operator request`**
  - The admin endpoint accepts loopback requests for internal API calls used by `pkg/btstrp` (via `federation.AdminFunc`) and for debug endpoints used by operators. Details: [arch-debug.md](./arch-debug.md).

## Components

### Layers

```text
External actors
    |
    +-- @Admin
    +-- *Client
    +-- @ACME
    |
    v
Routing subsystems
    |
    +-- [[Ingress]]
    +-- [[TLS]]
    +-- [[Reverse proxy]]
    +-- [[Debug]]
    |
    v
Cross-subsystem components
    |
    +-- [Admin endpoint]
    +-- [HTTP server]
    +-- [Router (gorilla/mux)]
    +-- [CORS wrapper]
    +-- [Query limiter]
    |
    v
Configuration
    |
    +-- [RouterParams]
```

### Routing subsystems

- `[[Ingress]]`
  - Public HTTP/HTTPS listener that terminates TLS and dispatches API v1 / API v2 / BLOB / N10N requests to the `apps` context.
  - Path to file: [arch-ingress.md](./arch-ingress.md)

- `[[TLS]]`
  - HTTPS certificate provisioning via ACME: the autocert manager that orders, caches, and renews certificates from `@ACME` (Let's Encrypt) for the operator-supplied hostname whitelist, and the dedicated HTTP-01 challenge listener on port 80 that answers validation requests. Wired only when the public listener runs on `HTTPSPort`.
  - Path to file: [arch-tls.md](./arch-tls.md)

- `[[Reverse proxy]]`
  - Last-resort matcher on the public router that forwards requests not matched by ingress routes to operator-configured upstreams.
  - Path to file: [arch-reverse-proxy.md](./arch-reverse-proxy.md)

- `[[Debug]]`
  - `net/http/pprof` endpoints (`/debug/pprof`, `/debug/cmdline`, `/debug/profile`, `/debug/symbol`, `/debug/trace`) mounted on the admin endpoint (and also on the public router, currently without authentication).
  - Path to file: [arch-debug.md](./arch-debug.md)

### Cross-subsystem components

- `[Admin endpoint]`
  - Localhost-only listener on `AdminPort` (default `55555`) sharing the same handler registration sequence as the public endpoint (router checker, API v1, API v2, debug, reverse-proxy) with two carve-outs: BLOB routes are skipped because `blobRequestHandler` is nil, and the `[Query limiter]` is disabled (the comment in source: `limiter is nil for Admin and ACME services`). The VVM service pipeline (`provideServicePipeline`) runs the admin endpoint operator strictly before the bootstrap operator so `pkg/btstrp.callDeployApp` can invoke `c.cluster.DeployApp` via `federation.AdminFunc` (which targets `localhost:AdminPort`) before the public endpoint accepts external traffic.
  - Path to file: [pkg/router/provide.go](../../../../pkg/router/provide.go)

- `[HTTP server]`
  - Thin wrapper over `net/http.Server` used by every listener (public, admin, ACME). Owns the TCP listener, optional PROXY-protocol decoration (`pires/go-proxyproto`), connection-count cap (`golang.org/x/net/netutil.LimitListener`), and read/write timeouts. Logs `endpoint.listen.start` on startup and `endpoint.shutdown` on graceful stop.
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

- `[Router (gorilla/mux)]`
  - Per-listener `mux.Router` populated in fixed order by `routerService.Prepare`: `registerRouterCheckerHandler` (`/api/check`) → `registerHandlersV1` (`/blob/...` when `blobRequestHandler != nil`, `/api/...`, `/n10n/...`) → `registerHandlersV2` (`/api/v2/...`) → `registerDebugHandlers` (`/debug/...`) → `registerReverseProxyHandler` (matcher-only, must be last).
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

- `[CORS wrapper]`
  - Wraps every API, BLOB, and N10N handler to set `Access-Control-Allow-Origin: *`, allow the headers used by the browser SDK, and short-circuit `OPTIONS` preflight requests.
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

- `[Query limiter]`
  - Per-`WSID` concurrent-query cap (`MaxQueriesPerWS`, default `10`) gating API v1 queries and the `q.cluster.VSqlUpdate2` shim as well as every API v2 handler; rejects excess requests with `503 Service Unavailable` and logs aggregated rejections every `10s`. Disabled on the admin endpoint (`limiter is nil for Admin and ACME services`).
  - Path to file: [pkg/router/impl_limiter.go](../../../../pkg/router/impl_limiter.go)

### Configuration

- `[RouterParams]`
  - Single configuration record supplied by `@Admin` at VVM startup: public port (`Port`), `AdminPort`, read/write timeouts, connections cap, `UseProxyProtocol`, `HTTP01ChallengeHosts` and `CertDir` for ACME, `RouteDefault` / `Routes` / `RoutesRewrite` / `RouteDomains` for the reverse proxy, `MaxQueriesPerWS`. Per-app dynamic registration of routes or hosts is not implemented.
  - Path to file: [pkg/router/types.go](../../../../pkg/router/types.go)

## Scenarios

### Serve client request

```text
*Client
  -> [HTTP server] (public listener: Port=443 with TLS, or Port=DefaultPort plain)
  -> [Router (gorilla/mux)] match by method + path
  -> [CORS wrapper] (set CORS headers, short-circuit OPTIONS)
  -> [Query limiter].acquire(WSID) on q.* / API v2 calls
  -> [[Ingress]] handler -> apps context via bus.IRequestSender / blobprocessor.IRequestHandler / in10n.IN10nBroker
  -> response back to *Client
```

Details: [arch-ingress.md](./arch-ingress.md).

### Provision TLS certificate

```text
@ACME
  -> HTTP-01 challenge on :80
  -> ACME challenge listener (autocert.Manager.HTTPHandler) -> respond with token
  -> autocert.Manager caches the certificate in autocert.Cache (or autocert.DirCache(CertDir))
  -> public [HTTP server] serves TLS using crtMgr.GetCertificate for hosts in HTTP01ChallengeHosts
```

Details: [arch-tls.md](./arch-tls.md).

### Forward to upstream

```text
*Client request
  -> [Router (gorilla/mux)] no API / BLOB / N10N / debug match
  -> [[Reverse proxy]] matcher resolves route (domain, path-prefix, rewrite, or default)
  -> httputil.ReverseProxy forwards to upstream
```

Details: [arch-reverse-proxy.md](./arch-reverse-proxy.md).

### Serve bootstrap and operator request

```text
pkg/btstrp.callDeployApp
  -> federation.AdminFunc("api/sys.cluster/.../c.cluster.DeployApp", body)
  -> [Admin endpoint] (localhost:AdminPort) [HTTP server]
  -> [Router (gorilla/mux)] -> registerHandlersV1 -> RequestHandler_V1
  -> bus.IRequestSender -> command processor

@Admin / operator
  -> [Admin endpoint] /debug/...
  -> [[Debug]] pprof handler
```

Details: [arch-debug.md](./arch-debug.md).

## Cross-cutting concerns

### Context dependencies

- The `apps` context owns request processing once dispatched: the `[BLOB processor]`, the `[N10n broker]`, and the command / query processors invoked via `bus.IRequestSender`. See [../apps/arch-processing.md](../apps/arch-processing.md).
- The `auth` context owns principal token validation; routing only carries the `Authorization` header through to `apps`. See [../auth/arch.md](../auth/arch.md).
- The `observability` context receives structured logs emitted on every listener lifecycle event, request, query-limit rejection, and N10N error.

### Out-of-context implementations

The standalone HTTP processor abstraction in `pkg/ihttp`, `pkg/ihttpctl`, and `pkg/ihttpimpl` (used by `cmd/voedger server`) is a parallel HTTP stack outside the production VVM wiring and is not covered here.
