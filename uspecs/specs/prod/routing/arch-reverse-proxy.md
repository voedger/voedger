# Context subsystem architecture: prod/routing/reverse-proxy

Routing reverse-proxy subsystem architecture: a single matcher registered last on the public router that forwards any request not consumed by API, BLOB, N10N, or debug routes to an operator-configured upstream. Context-level overview: [arch.md](./arch.md).

## External actors

Roles:

- `@Admin`
  - Configures the route table at VVM startup via `RouterParams.RouteDefault`, `Routes`, `RoutesRewrite`, and `RouteDomains`. Per-app dynamic registration is not implemented.

Systems:

- `*Client`
  - External caller whose request is not matched by any API / BLOB / N10N / debug handler.

- `*Upstream`
  - External HTTP service (e.g. Grafana, reseller portal, fallback "not found" page) reached by the reverse proxy.

## Scenarios overview

- **`Forward host-based domain route`**
  - When the request host (without port) is a key in `RouteDomains`, rewrite the request to the configured target host and forward.

- **`Forward path-prefix route`**
  - When the request path starts with a prefix in `Routes`, forward to the configured target preserving the original path.

- **`Forward path-prefix route with rewrite`**
  - When the request path starts with a prefix in `RoutesRewrite`, replace the matched prefix with the target path before forwarding.

- **`Forward to default route`**
  - When no host or path matches and `RouteDefault` is configured, prepend the default target path to the original request path and forward.

- **`Decline unmatched request`**
  - When neither a domain, path, nor default route matches, the matcher returns false and gorilla/mux replies with `404 Not Found`.

## Components

### Layers

```text
External actors
    |
    +-- @Admin
    +-- *Client
    +-- *Upstream
    |
    v
Entry points
    |
    +-- [Reverse-proxy matcher]
    |
    v
In-pipeline operators
    |
    +-- [Route table parser]
    +-- [Domain rule]
    +-- [Path-prefix rule]
    +-- [Rewrite rule]
    +-- [Default rule]
    +-- [Request redirector]
    |
    v
Shared infrastructure
    |
    +-- [httputil.ReverseProxy]
```

### Entry points

- `[Reverse-proxy matcher]`
  - `mux.MatcherFunc` registered as the final route on the public router by `registerReverseProxyHandler`. Acts as both matcher and director: when it returns true it stores `httputil.ReverseProxy` on the route match and the rewritten request is forwarded; when it returns false gorilla/mux falls through to its default 404 handler.
  - Path to file: [pkg/router/impl_reverseproxy.go](../../../../pkg/router/impl_reverseproxy.go)

### In-pipeline operators

- `[Route table parser]`
  - `parseRoutes` validates every path prefix starts with `/` and parses every target URL once at `Prepare` time. Failures abort listener startup.
  - Path to file: [pkg/router/impl_reverseproxy.go](../../../../pkg/router/impl_reverseproxy.go)

- `[Domain rule]`
  - Looks up the request host (with the port stripped) in `RouteDomains`; on hit, rewrites the request host to the target host and forwards regardless of path. Highest precedence _within the reverse-proxy matcher_ — domain matches short-circuit `[Path-prefix rule]` and `[Default rule]`, but only requests that fall through the earlier API / BLOB / N10N / debug routes reach the matcher in the first place (the reverse proxy is registered last on the gorilla/mux router).
  - Path to file: [pkg/router/impl_reverseproxy.go](../../../../pkg/router/impl_reverseproxy.go)

- `[Path-prefix rule]`
  - Walks the request path one slash-separated segment at a time and looks up the accumulated prefix in the merged `Routes` + `RoutesRewrite` table; first hit wins. Non-rewrite hit forwards preserving the original path.
  - Path to file: [pkg/router/impl_reverseproxy.go](../../../../pkg/router/impl_reverseproxy.go)

- `[Rewrite rule]`
  - On a hit whose route has `isRewrite = true`, replaces the matched prefix in the request path with the target URL's path before forwarding.
  - Path to file: [pkg/router/impl_reverseproxy.go](../../../../pkg/router/impl_reverseproxy.go)

- `[Default rule]`
  - When neither `[Domain rule]` nor `[Path-prefix rule]` matches and `RouteDefault` is set, prepends the default target path to the original request path and forwards.
  - Path to file: [pkg/router/impl_reverseproxy.go](../../../../pkg/router/impl_reverseproxy.go)

- `[Request redirector]`
  - `redirect` mutates the inbound `*http.Request`: sets `URL.Scheme`, `URL.Host`, `Host`, the rewritten `URL.Path`, and merges target query string with the inbound one. `httputil.ReverseProxy` is constructed with an empty `Director` because the matcher has already performed the rewrite.
  - Path to file: [pkg/router/impl_reverseproxy.go](../../../../pkg/router/impl_reverseproxy.go)

### Shared infrastructure

- `[httputil.ReverseProxy]`
  - Single reused `net/http/httputil.ReverseProxy` instance attached to every matched request. Performs the upstream round-trip and copies the response back to the client.
  - Path to package: [net/http/httputil](https://pkg.go.dev/net/http/httputil#ReverseProxy)

## Scenarios

### Forward host-based domain route

```text
*Client https://resellerportal.dev.untill.ru/foo
  -> [Reverse-proxy matcher]
  -> [Domain rule] resellerportal.dev.untill.ru -> http://resellerportal
  -> [Request redirector] req.URL = http://resellerportal/foo
  -> [httputil.ReverseProxy] -> *Upstream
```

### Forward path-prefix route

```text
*Client https://alpha.dev.untill.ru/grafana/foo
  -> [Reverse-proxy matcher]
  -> [Path-prefix rule] /grafana -> http://10.0.0.3:3000
  -> [Request redirector] req.URL = http://10.0.0.3:3000/grafana/foo
  -> [httputil.ReverseProxy] -> *Upstream
```

### Forward path-prefix route with rewrite

```text
*Client https://alpha.dev.untill.ru/grafana-rewrite/foo
  -> [Reverse-proxy matcher]
  -> [Path-prefix rule] /grafana-rewrite -> http://10.0.0.3:3000/rewritten (isRewrite)
  -> [Rewrite rule] /grafana-rewrite/foo -> /rewritten/foo
  -> [Request redirector] req.URL = http://10.0.0.3:3000/rewritten/foo
  -> [httputil.ReverseProxy] -> *Upstream
```

### Forward to default route

```text
*Client https://alpha.dev.untill.ru/unknown/foo
  -> [Reverse-proxy matcher]
  -> no [Domain rule] hit, no [Path-prefix rule] hit
  -> [Default rule] RouteDefault = http://10.0.0.3:3000/not-found
  -> [Request redirector] req.URL = http://10.0.0.3:3000/not-found/unknown/foo
  -> [httputil.ReverseProxy] -> *Upstream
```

### Decline unmatched request

```text
*Client request with no matching route and no RouteDefault
  -> [Reverse-proxy matcher] returns false
  -> gorilla/mux default 404 Not Found
  -> *Client
```

## Notes

The rewrite mechanism here is not modeled on Apache `mod_rewrite`. `mod_rewrite` is a regex engine with capture groups, `RewriteCond`, per-rule flags (`[P]`, `[L]`, `[QSA]`, `[R]`), and chained rewrite passes; the matcher in `pkg/router` does literal prefix matching on host or path with a single substitution per route in a single pass. The closer analogs are Apache [`ProxyPass`](https://httpd.apache.org/docs/current/mod/mod_proxy.html#proxypass) and Nginx [`location { proxy_pass; }`](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_pass), which expose the same "preserve or strip the matched prefix" choice as `Routes` and `RoutesRewrite` respectively.
