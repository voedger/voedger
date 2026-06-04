# Context subsystem architecture: prod/routing/ingress

Routing ingress subsystem architecture for the public HTTP/HTTPS listener: API v1 and API v2 dispatch with CORS, BLOB transfer endpoints, N10N Server-Sent Events subscription endpoints, and the per-workspace concurrent-query limit. TLS termination uses certificates supplied by the TLS subsystem ([arch-tls.md](./arch-tls.md)). Context-level overview: [arch.md](./arch.md).

## External actors

Roles:

- `@Admin`
  - Configures the public listener (port, TLS, ACME-managed hostname list, proxy-protocol, connection cap, per-WS query limit) via `RouterParams`. See [arch.md](./arch.md#configuration).

Systems:

- `*Client`
  - Browser, mobile, or backend caller invoking commands, queries, BLOB transfers, and N10n subscriptions.

## Scenarios overview

- **`Dispatch API v1 request`**
  - Validate the URL placeholders (`appOwner/appName/wsid/resourceName`), apply CORS, acquire the per-WS query limit on `q.*` and on the `q.cluster.VSqlUpdate2` shim, forward the request through `bus.IRequestSender` to the `apps` command or query processor, and stream the response back.

- **`Dispatch API v2 request`**
  - Match the path against the API v2 catalog (`docs`, `cdocs`, `commands`, `queries`, `views`, `schemas`, BLOB temporary/persistent), acquire the per-WS query limit, dispatch through `bus.IRequestSender` to the v2 processor, and stream rows.

- **`Transfer BLOB`**
  - Validate the BLOB URL and headers, delegate to the `apps`-owned `[BLOB processor]` (`blobprocessor.IRequestHandler.HandleWrite` / `HandleRead`) which streams bytes between the client and `[(BLOB storage)]`; return `503 Service Unavailable` with `Retry-After` when the handler is busy.

- **`Subscribe to N10n over SSE`**
  - Parse the subscription payload, open an SSE stream (`Content-Type: text/event-stream`), create a channel on the `apps`-owned `[N10n broker]`, subscribe to the requested projection keys, and forward broker updates to the client until the request context is cancelled.

- **`Reject excess concurrent query`**
  - When the per-`WSID` query count reaches `MaxQueriesPerWS`, reply `503 Service Unavailable` and aggregate per-extension rejection counters logged every `10s`.

## Components

### Layers

```text
External actors
    |
    +-- *Client
    |
    v
Listeners
    |
    +-- [Public listener]
    |
    v
Entry points
    |
    +-- [API v1 handler]
    +-- [API v2 handler]
    +-- [BLOB handler]
    +-- [N10N SSE handler]
    +-- [Router checker]
    |
    v
In-pipeline operators
    |
    +-- [Request validator]
    +-- [CORS wrapper]
    +-- [Query limiter]
    +-- [VSqlUpdate v1 shim]
    +-- [Response writer]
    |
    v
Boundary to apps
    |
    +-- [bus.IRequestSender]
    +-- [blobprocessor.IRequestHandler]
    +-- [in10n.IN10nBroker]
```

### Listeners

- `[Public listener]`
  - `[HTTP server]` bound to `RouterParams.Port`. On `Port == 443` (`HTTPSPort`) wrapped by `httpsService` which configures `tls.Config{GetCertificate: crtMgr.GetCertificate, MinVersion: tls.VersionTLS12}` and serves via `ServeTLS`; otherwise serves plain HTTP. Optional PROXY-protocol decoration and connection-count cap are applied by the shared `[HTTP server]` from [arch.md](./arch.md#cross-subsystem-components). Certificate provisioning and `crtMgr` lifecycle: [arch-tls.md](./arch-tls.md).
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

### Entry points

- `[API v1 handler]`
  - `RequestHandler_V1` registered on `POST|PATCH|OPTIONS /api/{appOwner}/{appName}/{wsid:[0-9]+}/{resourceName}`. Builds a `bus.Request`, optionally acquires `[Query limiter]` (for `q.*` and the `[VSqlUpdate v1 shim]`), sends through `[bus.IRequestSender]`, and replies via `[Response writer]`.
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

- `[API v2 handler]`
  - Family of handlers registered by `registerHandlersV2` covering `docs`, `cdocs`, `commands`, `queries`, `views`, `schemas` (`/api/v2/apps/{owner}/{app}/...`), workspace roles, BLOB temporary and persistent endpoints, and a generic catch-all for unknown v2 paths. Every handler acquires `[Query limiter]` and forwards through `[bus.IRequestSender]`.
  - Path to file: [pkg/router/impl_apiv2.go](../../../../pkg/router/impl_apiv2.go)

- `[BLOB handler]`
  - `blobHTTPRequestHandler_Write` on `POST|OPTIONS /blob/{appOwner}/{appName}/{wsid}` and `blobHTTPRequestHandler_Read` on `POST|GET|OPTIONS /blob/{appOwner}/{appName}/{wsid}/{blobIDOrSUUID}`. Each calls `[blobprocessor.IRequestHandler]` and falls back to `503 Service Unavailable` with `Retry-After: 1` when the handler reports it is overloaded.
  - Path to file: [pkg/router/impl_blob.go](../../../../pkg/router/impl_blob.go)

- `[N10N SSE handler]`
  - Four handlers on `GET /n10n/...`: `subscribeAndWatchHandler` (`/n10n/channel` - opens an SSE stream and subscribes), `subscribeHandler` (`/n10n/subscribe`), `unSubscribeHandler` (`/n10n/unsubscribe`), and `updateHandler` (`/n10n/update/{offset}`). Each manipulates a channel on `[in10n.IN10nBroker]` and streams events over `text/event-stream` until the request context is cancelled. `routerService.Stop` waits for `MetricNumSubscriptions() == 0` before completing shutdown.
  - Path to file: [pkg/router/impl_n10n.go](../../../../pkg/router/impl_n10n.go)

- `[Router checker]`
  - `POST|GET|OPTIONS /api/check` replies `ok`; used by health probes.
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

### In-pipeline operators

- `[Request validator]`
  - `withValidateForFuncs` / `withValidateForBLOBs` parse URL placeholders into a `validatedData` record (`appQName`, `wsid`, headers, body) and short-circuit malformed requests before any apps-side dispatch.
  - Path to file: [pkg/router/impl_validation.go](../../../../pkg/router/impl_validation.go)

- `[CORS wrapper]`
  - Shared with the rest of the routing context; see [arch.md](./arch.md#cross-subsystem-components).
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

- `[Query limiter]`
  - Shared with the rest of the routing context; see [arch.md](./arch.md#cross-subsystem-components). On the public listener it wraps every API v2 handler and the `q.*` / `q.cluster.VSqlUpdate2` paths of `[API v1 handler]`; the admin endpoint mounts the same handler set but with the limiter disabled (`limiter is nil for Admin and ACME services`, see [arch.md](./arch.md#cross-subsystem-components)). Rejections reply `503 Service Unavailable`.
  - Path to file: [pkg/router/impl_limiter.go](../../../../pkg/router/impl_limiter.go)

- `[VSqlUpdate v1 shim]`
  - `dispatchVSqlUpdateShim_V1` re-routes API v1 `c.cluster.VSqlUpdate` calls to `q.cluster.VSqlUpdate2` so the workpiece runs on the query processor instead of synchronously re-entering the command processor.
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

- `[Response writer]`
  - `initResponse` / `reply_v1` / `reply_v2` translate the `bus.ResponseMeta` mode (`Single`, `StreamJSON`, `StreamEvents`) into HTTP headers and stream-or-buffer the response body; `applySysErrorHeaders` propagates `coreutils.SysError` headers.
  - Path to file: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)

### Boundary to apps

- `[bus.IRequestSender]`
  - Asynchronous request bus carrying every API v1 / API v2 request from ingress to the matching processor in the `apps` context; the response channel is consumed by `[Response writer]`. Definition and processors: see [../apps/arch-processing.md](../apps/arch-processing.md).
  - Path to file: [pkg/bus/interface.go](../../../../pkg/bus/interface.go)

- `[blobprocessor.IRequestHandler]`
  - HandleWrite / HandleRead surface implemented by the `apps`-owned `[BLOB processor]`; ingress streams bytes through it without owning BLOB metadata or storage. See [../apps/arch-processing.md](../apps/arch-processing.md).
  - Path to file: [pkg/processors/blobber/interface.go](../../../../pkg/processors/blobber/interface.go)

- `[in10n.IN10nBroker]`
  - Channel / subscription / watch surface implemented by the `apps`-owned `[N10n broker]`; ingress opens channels, subscribes to projection keys, and forwards updates over SSE. See [../apps/arch-processing.md](../apps/arch-processing.md).
  - Path to file: [pkg/in10n/interface.go](../../../../pkg/in10n/interface.go)

## Scenarios

### Dispatch API v1 request

```text
*Client POST /api/{owner}/{app}/{wsid}/{resource}
  -> [Public listener] -> [Router (gorilla/mux)] match "api"
  -> [CORS wrapper] -> [Request validator]
  -> if q.* or VSqlUpdate shim: [Query limiter].acquire(WSID)
  -> if VSqlUpdate shim: [VSqlUpdate v1 shim] -> q.cluster.VSqlUpdate2
  -> else: [bus.IRequestSender].SendRequest -> apps processor
  -> [Response writer] (Single / StreamJSON / StreamEvents)
  -> *Client
```

### Dispatch API v2 request

```text
*Client {GET|POST|PATCH|DELETE} /api/v2/apps/{owner}/{app}/...
  -> [Public listener] -> [Router (gorilla/mux)] match by API v2 path
  -> [CORS wrapper] -> [Request validator]
  -> [Query limiter].acquire(WSID)
  -> [API v2 handler] -> [bus.IRequestSender] -> apps v2 processor
  -> [Response writer]
  -> *Client
```

### Transfer BLOB

```text
*Client {POST|GET} /blob/{owner}/{app}/{wsid}[/{blobID}]
  -> [Public listener] -> [Router (gorilla/mux)] match "blob write" / "blob read"
  -> [CORS wrapper] -> [Request validator (BLOBs)]
  -> [BLOB handler] -> [blobprocessor.IRequestHandler].HandleWrite / HandleRead
       (apps-owned [BLOB processor] streams bytes to/from [(BLOB storage)])
  -> if handler returns false: HTTP 503 Service Unavailable + Retry-After
  -> *Client
```

### Subscribe to N10n over SSE

```text
*Client GET /n10n/channel?payload={SubjectLogin,ProjectionKey[]}
  -> [Public listener] -> [Router (gorilla/mux)] match "/n10n/channel"
  -> [CORS wrapper] -> [N10N SSE handler].subscribeAndWatchHandler
  -> set headers: Content-Type=text/event-stream, Cache-Control=no-cache, Connection=keep-alive
  -> [in10n.IN10nBroker].NewChannel -> Subscribe(channel, projectionKey)
  -> stream "event: <name>\ndata: <payload>\n\n" until req.Context() is done
  -> on Stop: routerService.Stop waits for MetricNumSubscriptions() == 0
```

### Reject excess concurrent query

```text
*Client query while [Query limiter] count for WSID == MaxQueriesPerWS
  -> [Public listener] -> [Router (gorilla/mux)] -> [CORS wrapper] (sets CORS headers, also on rejection)
  -> [Query limiter].acquire returns false
  -> onQueryDrop: increment per (WSID, extension) counter; flush aggregated log every 10s
  -> replyServiceUnavailable: HTTP 503 Service Unavailable
  -> *Client
```
