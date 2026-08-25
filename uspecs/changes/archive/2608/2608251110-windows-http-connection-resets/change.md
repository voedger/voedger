---
change_id: 2608251052-windows-http-connection-resets
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4777
domains: [prod]
scope: [routing]
---

# Change request: Prevent Windows HTTP connection resets

## Why

Voedger integration tests intermittently fail on Windows while the platform sends HTTP requests to its local endpoint, including application deployment during VVM bootstrap. The shared HTTP client forces excessive TCP connection churn and does not consistently complete the response-body lifecycle, making high-volume test traffic vulnerable to Windows connection resets.

## What

Symptom: Integration tests on Windows intermittently panic when a local HTTP request fails with `WSAECONNRESET` because the remote endpoint forcibly closes the connection.

```text
integration test starts a VVM and exercises platform HTTP APIs
      |
      v
bootstrap and test clients send repeated local HTTP requests
      |
      v
httpu.newRequest sets Request.Close = true   <-- fault: disables persistent connection reuse
      |
      v
httpu opens and tears down a TCP connection for every request
      |
      v
httpu.readBody reads a non-streaming response without closing its body   <-- fault: leaves the response lifecycle incomplete
      |
      v
Windows loopback TCP connection churn and reset exposure increase
      |
      v
request fails with WSAECONNRESET and bootstrap repanics   (symptom)
```

Corrected behavior: The shared HTTP client keeps persistent connections reusable and closes every non-streaming response body, so repeated local requests complete without intermittent Windows connection-reset failures.

## How

Decisions:

- Use the existing shared Go HTTP transport's default keep-alive pool; do not force per-request connection closure or introduce another pool, cache, or dependency.
- Make the shared body-reading helper own and close non-streaming response bodies, while response-handler and long-polling paths retain explicit ownership of their streamed bodies.
- Preserve the transport's existing zero-linger socket policy and lifecycle cleanup because they address eventual socket teardown independently of per-request connection reuse.
- Verify body ownership and connection reuse deterministically at the shared HTTP client boundary instead of depending on timing-sensitive Windows integration-test reproduction.

Assumptions:

- None

Out of scope:

- Changing retry policies to retry `WSAECONNRESET`
- Changing router timeouts or connection limits
- Changing the lifecycle of streaming, BLOB, or long-polling responses

References (internal):

- [shared HTTP request lifecycle](../../../../../pkg/goutils/httpu/impl.go)
- [non-streaming response-body helpers](../../../../../pkg/goutils/httpu/utils.go)
- [shared transport and socket cleanup policy](../../../../../pkg/goutils/httpu/provide.go)
- [streaming and discard response ownership options](../../../../../pkg/goutils/httpu/impl_opts.go)
- [shared HTTP client behavior coverage](../../../../../pkg/goutils/httpu/impl_test.go)
- [response helper coverage](../../../../../pkg/goutils/httpu/utils_test.go)
- [bootstrap application deployment requests](../../../../../pkg/btstrp/impl.go)

References (external):

- [Go request connection-close semantics](https://pkg.go.dev/net/http#Request)
- [Go response-body reuse contract](https://pkg.go.dev/net/http#Client.Do)
- [existing Windows ephemeral-port mitigation](https://github.com/voedger/voedger/issues/415)

## Construction

- [x] update: [goutils/httpu/utils_test.go](../../../../../pkg/goutils/httpu/utils_test.go)
  - add: close-tracking response body used to observe ownership without a network dependency
  - add: regression coverage that `readBody` returns the complete body and closes it after a successful read
  - add: regression coverage that `readBody` returns a read error and still closes the response body

- [x] update: [goutils/httpu/impl_test.go](../../../../../pkg/goutils/httpu/impl_test.go)
  - add: deterministic connection-reuse coverage using a custom transport that counts TCP dials
  - verify: two sequential requests to the same endpoint complete through one TCP connection

- [x] update: [goutils/httpu/utils.go](../../../../../pkg/goutils/httpu/utils.go)
  - update: `readBody` always closes the response body after attempting to read it

- [x] update: [goutils/httpu/impl.go](../../../../../pkg/goutils/httpu/impl.go)
  - remove: forced per-request connection closure so the shared transport can reuse persistent connections
