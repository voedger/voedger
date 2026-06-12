---
change_id: 2606031418-derive-routing-context-architecture
type: docs
issue_url: https://untill.atlassian.net/browse/AIR-4160
domains: [prod]
---

# Change request: Derive `routing` context architecture

Refs:

- [AIR-4160: voedger: derive architecture from routing context](./issue-AIR-4160.md)

## Why

The `routing` context listed in `uspecs/specs/prod/domain.md` has no Context Architecture; reviewers and contributors lack a single architecture reference for the HTTP boundary of voedger — request routing, reverse-proxy forwarding, HTTPS/ACME, BLOB and N10N endpoints, the per-workspace concurrent-query limit, domain management, and the admin/debug listener.

## What

Add architecture specifications for the `routing` context, derived from existing documentation and the codebase:

- Add a Context Architecture for `routing` split across five new files under `uspecs/specs/prod/routing/`:
  - `arch.md` — context overview, external actors and dependencies, cross-subsystem components, the admin endpoint, and links to the subsystem chapters below
  - `arch-ingress.md` — the public HTTP/HTTPS listener subsystem
  - `arch-tls.md` — the TLS / ACME subsystem (HTTPS certificate provisioning, HTTP-01 challenge listener, autocert manager and cache)
  - `arch-reverse-proxy.md` — the reverse-proxy subsystem
  - `arch-debug.md` — the debug endpoints (pprof handlers) mounted on the admin endpoint
- Cover every externally observable capability the production `routing` implementation exposes, treating the HTTP boundary as the context boundary:
  - Public HTTP/HTTPS listener that terminates TLS and dispatches API v1 and API v2 requests to the `apps` context, with CORS handling for browser callers (`arch-ingress.md`)
  - BLOB transfer endpoints (upload and download) that validate the HTTP request and delegate to the `apps`-owned `[BLOB processor]` (`arch-ingress.md`)
  - N10N real-time subscription endpoints over Server-Sent Events — subscribe, unsubscribe, channel watch, offset update — that translate HTTP calls into subscriptions on the `apps`-owned `[N10n broker]` and stream the resulting events back to the client (`arch-ingress.md`)
  - HTTPS certificate provisioning via ACME, including the dedicated HTTP-01 challenge listener on port 80 (`arch-tls.md`)
  - Per-workspace concurrent-query limit on API endpoints — a cap on in-flight queries per `WSID` configured via `MaxQueriesPerWS`, with backpressure rejections when the cap is reached (`arch-ingress.md`)
  - Reverse-proxy forwarding to external upstream services, including the four route kinds exposed to operators — path-prefix routes, path-prefix routes with rewrite, host-based domain routes, and the default fallback route (`arch-reverse-proxy.md`)
  - Admin endpoint: localhost-only listener on `AdminPort` (default `55555`) that mounts the same handler set as the public endpoint except BLOB endpoints and the per-WS query limiter; started before bootstrap so the bootstrap operator can invoke internal API functions via `federation.AdminFunc` before the public endpoint opens (`arch.md`); debug endpoints (pprof) mounted on it (`arch-debug.md`)
- Cover domain management as cluster-wide operator configuration supplied by `@Admin` at VVM startup (reverse-proxy route table and the ACME-managed hostname list); per-app dynamic domain registration is not implemented in `pkg/router` and is out of scope
- Keep boundary descriptions boundary-only: name the `apps`-side components that the `routing` context dispatches to (`[BLOB processor]`, `[N10n broker]`, command/query processors via the request bus) and cross-reference `apps/arch-processing.md` for their behavior; do not duplicate how `apps` processes the request once dispatched
- Document only the production implementation wired through the VVM (`pkg/router`); note the separate HTTP abstraction (`pkg/ihttpimpl`) as out of scope
- Give reviewers and contributors a complete architecture reference for the `routing` context that complements `uspecs/specs/prod/domain.md`

## How

Decisions:

- Author five new architecture files under `uspecs/specs/prod/routing/`: `arch.md` (context overview + admin endpoint + cross-subsystem components), `arch-ingress.md` (public HTTP/HTTPS listener subsystem), `arch-tls.md` (TLS / ACME subsystem), `arch-reverse-proxy.md` (reverse-proxy subsystem), `arch-debug.md` (pprof endpoints mounted on the admin endpoint)
- Derive the architecture from the production implementation in `pkg/router` (`provide.go`, `impl_http.go`, `impl_apiv2.go`, `impl_blob.go`, `impl_n10n.go`, `impl_reverseproxy.go`, `impl_acme.go`, `impl_limiter.go`, `types.go`) and the VVM wiring in `pkg/vvm/provide.go::provideServicePipeline`
- Use a layered structure comparable to `uspecs/specs/prod/apps/arch-processing.md` for each subsystem chapter — external actors → listeners / entry points → in-pipeline operators → shared infrastructure / boundary / state — with per-subsystem label adjustments and tier counts as the subsystem shape requires (e.g. `Boundary to apps` in `arch-ingress.md`, `TLS material` in `arch-tls.md`, `Shared infrastructure` in `arch-reverse-proxy.md` and `arch-debug.md`)
- Cross-reference `uspecs/specs/prod/apps/arch-processing.md` for the `apps`-owned `[BLOB processor]` and `[N10n broker]` from `arch-ingress.md` instead of redescribing them
- Document the admin endpoint's bootstrap role in `arch.md` (admin endpoint starts before bootstrap so `pkg/btstrp/impl.go::callDeployApp` can invoke internal API functions via `federation.AdminFunc` over `localhost:AdminPort`)

Out of scope:

- The standalone HTTP processor abstraction (`pkg/ihttp`, `pkg/ihttpctl`, `pkg/ihttpimpl`) used by `cmd/voedger server`
- Per-app dynamic domain registration (not implemented in `pkg/router`)
- Any behavior change in the `routing` context — this is a `docs` change that derives architecture from the existing codebase

References (internal):

- [routing production implementation](../../../../../pkg/router)
- [router service wiring and admin endpoint construction](../../../../../pkg/router/provide.go)
- [HTTP route registration and handler set](../../../../../pkg/router/impl_http.go)
- [API v2 dispatch handlers](../../../../../pkg/router/impl_apiv2.go)
- [BLOB upload and download handlers](../../../../../pkg/router/impl_blob.go)
- [N10N SSE subscription handlers](../../../../../pkg/router/impl_n10n.go)
- [reverse-proxy matcher and handler](../../../../../pkg/router/impl_reverseproxy.go)
- [ACME / autocert manager and HTTP-01 listener](../../../../../pkg/router/impl_acme.go)
- [per-workspace concurrent-query limiter](../../../../../pkg/router/impl_limiter.go)
- [RouterParams configuration surface](../../../../../pkg/router/types.go)
- [VVM service pipeline ordering admin-endpoint → bootstrap → public-endpoint](../../../../../pkg/vvm/provide.go)
- [default AdminPort 55555](../../../../../pkg/vvm/consts.go)
- [bootstrap call to federation.AdminFunc](../../../../../pkg/btstrp/impl.go)
- [domain index of the prod domain](../../../../../uspecs/specs/prod/domain.md)
- [apps context architecture — BLOB processor and N10n broker reference](../../../../../uspecs/specs/prod/apps/arch-processing.md)

References (external):

- [ACME RFC 8555](https://datatracker.ietf.org/doc/html/rfc8555)
- [Server-Sent Events (HTML Living Standard)](https://html.spec.whatwg.org/multipage/server-sent-events.html)

## Technical design

- [x] create: [routing/arch.md](../../../../specs/prod/routing/arch.md)
  - Context Architecture: overview of the `routing` context, external actors and dependencies (`@Admin`, `@ACME`), cross-subsystem components, the cluster-wide operator configuration supplied at VVM startup via `RouterParams` (reverse-proxy route table, ACME-managed hostname list, query limit, ports), and the admin endpoint (localhost-only listener on `AdminPort`, default `55555`) including its bootstrap role (started before bootstrap so `pkg/btstrp` can invoke internal API functions via `federation.AdminFunc` before the public endpoint opens); links to the subsystem chapters below
  - Also lists `*Client` as the external system invoking commands, queries, BLOB transfers, and N10n SSE subscriptions over HTTP/HTTPS
  - Cross-cutting concerns: dependencies on `apps` (request processing via `bus.IRequestSender`, `[BLOB processor]`, `[N10n broker]`), `auth` (principal token validation; routing only carries the `Authorization` header through), and `observability` (structured logs on listener lifecycle, requests, query-limit rejections, N10N errors); `pkg/ihttp` / `pkg/ihttpctl` / `pkg/ihttpimpl` called out as a parallel HTTP stack outside the production VVM wiring

- [x] create: [routing/arch-ingress.md](../../../../specs/prod/routing/arch-ingress.md)
  - Context Subsystem Architecture: public HTTP/HTTPS listener subsystem — API v1/v2 dispatch with CORS, BLOB endpoints delegating to the `apps`-owned `[BLOB processor]`, N10N SSE endpoints subscribing on the `apps`-owned `[N10n broker]`, per-workspace concurrent-query limit (`MaxQueriesPerWS`); TLS termination consumes certificates from the TLS subsystem
  - `[VSqlUpdate v1 shim]` (`dispatchVSqlUpdateShim_V1`) re-routes API v1 `c.cluster.VSqlUpdate` calls to `q.cluster.VSqlUpdate2` so the workpiece runs on the query processor instead of synchronously re-entering the command processor
  - `[Router checker]` on `POST|GET|OPTIONS /api/check` returning `ok`, used by health probes

- [x] create: [routing/arch-tls.md](../../../../specs/prod/routing/arch-tls.md)
  - Context Subsystem Architecture: TLS / ACME subsystem — `[ACME HTTP-01 listener]` on port 80, `[autocert.Manager]` configured with `HostPolicy: autocert.HostWhitelist(HTTP01ChallengeHosts...)` and the Let's Encrypt production directory, `[(autocert.Cache)]` keyed by `RouterParams.CertDir`; scenarios for first-request provisioning, cached-certificate serving, background renewal, and hostname rejection
  - `Notes` section records that the public listener fixes `tls.Config.MinVersion = tls.VersionTLS12` (TLS 1.3 is disabled because of a third-party billing integration, per the inline comment in `pkg/router/impl_http.go`) and links the underlying ACME (RFC 8555) and HTTP-01 challenge references

- [x] create: [routing/arch-reverse-proxy.md](../../../../specs/prod/routing/arch-reverse-proxy.md)
  - Context Subsystem Architecture: reverse-proxy subsystem — the four route kinds (path-prefix, path-prefix with rewrite, host-based domain, default fallback), route matching and upstream forwarding model
  - `Decline unmatched request` scenario: when no domain, path, or default route matches, the matcher returns false and gorilla/mux replies with `404 Not Found`
  - `Notes` section clarifies that the rewrite mechanism is not modeled on Apache `mod_rewrite` (no regex, captures, `RewriteCond`, flags, or chained passes); the closer analogs are Apache `ProxyPass` and Nginx `location { proxy_pass; }`, mapping `Routes` to "preserve prefix" and `RoutesRewrite` to "strip-and-replace prefix"

- [x] create: [routing/arch-debug.md](../../../../specs/prod/routing/arch-debug.md)
  - Context Subsystem Architecture: debug endpoints (pprof handlers — `/debug/pprof`, `cmdline`, `profile`, `symbol`, `trace`) mounted on the admin endpoint
  - `[Debug alias]` on `/debug/{cmd}` rewrites the request path to `/debug/pprof/{cmd}` and delegates to `pprof.Index`, allowing shorter URLs for per-name pprof handlers (heap, goroutine, block, mutex, threadcreate, allocs)
  - Cross-cutting concerns: `registerDebugHandlers` runs unconditionally in `routerService.Prepare`, so the same `/debug/*` routes are also registered on the public endpoint without authentication; safe exposure on the public listener is therefore an operator-side concern (network ACLs)
