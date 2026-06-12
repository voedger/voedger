# Context subsystem architecture: prod/routing/tls

Routing TLS subsystem architecture for HTTPS certificate provisioning via ACME: the dedicated HTTP-01 challenge listener on port 80, the autocert manager that orders and caches certificates from `@ACME` (Let's Encrypt by default), and the certificate cache consumed by the public listener during TLS handshakes. Context-level overview: [arch.md](./arch.md). Public listener that consumes the certificates: [arch-ingress.md](./arch-ingress.md).

## External actors

Roles:

- `@Admin`
  - Configures the ACME-managed hostname whitelist and certificate cache directory via `RouterParams.HTTP01ChallengeHosts` and `RouterParams.CertDir`. See [arch.md](./arch.md#configuration).

Systems:

- `*Client`
  - HTTPS caller whose TLS handshake against the public listener triggers `GetCertificate` lookups in `[autocert.Manager]`.

- `@ACME`
  - ACME directory (Let's Encrypt production by default, `https://acme-v02.api.letsencrypt.org/directory`) that issues certificates after validating control over each host via the HTTP-01 challenge.

## Scenarios overview

- **`Provision certificate on first request`**
  - On a TLS handshake whose SNI matches `HTTP01ChallengeHosts` but has no cached certificate, `[autocert.Manager]` opens an ACME order, publishes the HTTP-01 key-authorization on `[ACME HTTP-01 listener]`, receives the signed certificate, persists it in `[(autocert.Cache)]`, and returns it to the public listener.

- **`Serve cached certificate`**
  - Subsequent TLS handshakes hit the in-memory or on-disk cache and complete without contacting `@ACME`.

- **`Renew certificate`**
  - `[autocert.Manager]` refreshes certificates near expiry by repeating the HTTP-01 flow in the background.

- **`Reject hostname outside the whitelist`**
  - SNI values not in `HTTP01ChallengeHosts` are refused by `[autocert.Manager]`'s `HostPolicy`, preventing rogue ACME orders that would burn the CA's per-account rate limits.

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
Listeners
    |
    +-- [ACME HTTP-01 listener]
    |
    v
TLS material
    |
    +-- [autocert.Manager]
    +-- [(autocert.Cache)]
```

### Listeners

- `[ACME HTTP-01 listener]`
  - Separate `[HTTP server]` from [arch.md](./arch.md#cross-subsystem-components) bound to `:80` whose only handler is `autocert.Manager.HTTPHandler(nil)`, which answers `/.well-known/acme-challenge/<token>` with the key-authorization string and returns 404 for all other paths. Provisioned only when the public listener runs on `HTTPSPort` (`443`); on any other public port the TLS subsystem is not wired at all. Port `80` is mandatory: the HTTP-01 challenge spec forbids redirecting validation to a non-standard port. Uses defaults `DefaultACMEServerWriteTimeout` / `DefaultACMEServerReadTimeout`.
  - Path to file: [pkg/router/impl_acme.go](../../../../pkg/router/impl_acme.go), [pkg/router/provide.go](../../../../pkg/router/provide.go)

### TLS material

- `[autocert.Manager]`
  - `golang.org/x/crypto/acme/autocert.Manager` with `Prompt: autocert.AcceptTOS`, `HostPolicy: autocert.HostWhitelist(HTTP01ChallengeHosts...)`, `Cache: autocertCache` (caller-injected) or `autocert.DirCache(CertDir)` when nil. Exposes two surfaces consumed by routing: `GetCertificate` (wired into the public listener's `tls.Config` with `MinVersion: tls.VersionTLS12` — see [arch-ingress.md](./arch-ingress.md#listeners)) and `HTTPHandler(nil)` (wired into `[ACME HTTP-01 listener]`). Talks to the Let's Encrypt production directory by default; a comment in `provide.go` describes switching to the staging directory (`https://acme-staging-v02.api.letsencrypt.org/directory`) when issuance volume risks Let's Encrypt rate limits during testing.
  - Path to file: [pkg/router/provide.go](../../../../pkg/router/provide.go)

- `[(autocert.Cache)]`
  - Certificate and ACME account-key store consumed by `[autocert.Manager]`. Supplied by the caller; defaults to `autocert.DirCache(RouterParams.CertDir)` when nil. Must persist across VVM restarts; losing it forces a fresh ACME account on next start and risks hitting Let's Encrypt's per-domain weekly issuance limit.
  - Path to file: [pkg/router/provide.go](../../../../pkg/router/provide.go)

## Scenarios

### Provision certificate on first request

```text
*Client TLS ClientHello (SNI = host, host in HTTP01ChallengeHosts)
  -> public [HTTP server] (see arch-ingress.md) tls.Config.GetCertificate
  -> [autocert.Manager].GetCertificate(SNI)
       -> [(autocert.Cache)].Get(host) returns cache miss
       -> ACME newOrder(host) -> @ACME returns authorization with http-01 challenge {token}
       -> [autocert.Manager] computes keyAuthorization = token.thumbprint(accountKey)
       -> @ACME GET http://host/.well-known/acme-challenge/{token}
            -> [ACME HTTP-01 listener] (:80) -> [autocert.Manager].HTTPHandler
            -> respond 200 + keyAuthorization
       -> @ACME validates control -> [autocert.Manager] submits CSR -> @ACME returns signed cert
       -> [(autocert.Cache)].Put(host, cert+key)
  -> *tls.Certificate returned to public listener
  -> TLS handshake completes -> *Client
```

### Serve cached certificate

```text
*Client TLS ClientHello (SNI = host)
  -> public [HTTP server] tls.Config.GetCertificate
  -> [autocert.Manager].GetCertificate(SNI)
       -> [(autocert.Cache)].Get(host) returns cert
  -> *tls.Certificate returned -> TLS handshake completes -> *Client
```

### Renew certificate

```text
[autocert.Manager] background refresh detects expiry approaching
  -> repeats `Provision certificate on first request` flow
  -> [(autocert.Cache)].Put(host, new cert+key)
```

### Reject hostname outside the whitelist

```text
*Client TLS ClientHello (SNI = host, host NOT in HTTP01ChallengeHosts)
  -> public [HTTP server] tls.Config.GetCertificate
  -> [autocert.Manager].GetCertificate(SNI)
       -> HostPolicy returns error
  -> tls.Config returns no certificate -> TLS handshake aborted by Go runtime
```

## Notes

The TLS subsystem is built on the [Automatic Certificate Management Environment protocol (RFC 8555)](https://datatracker.ietf.org/doc/html/rfc8555) and the [HTTP-01 challenge](https://letsencrypt.org/docs/challenge-types/#http-01-challenge); see [`golang.org/x/crypto/acme/autocert`](https://pkg.go.dev/golang.org/x/crypto/acme/autocert) for the Go implementation that voedger consumes without modification.

The minimum TLS version is fixed at TLS 1.2 (`tls.VersionTLS12`) at the public listener; TLS 1.3 is disabled because of a third-party billing integration, as noted in the inline comment in `pkg/router/impl_http.go`.
