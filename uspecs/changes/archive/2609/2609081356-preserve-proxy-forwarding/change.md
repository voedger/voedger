---
change_id: 2609081145-preserve-proxy-forwarding
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4881
domains: [prod]
scope: [routing]
---

# Change request: Reverse-proxy forwarding compatibility

Refs:

- [AIR-4881: voedger, reverse proxy: migrate to Rewrite and fix drops forwarding headers](./issue-AIR-4881.md)

## Why

Both HTTP proxy implementations in the Voedger platform routing context previously used the deprecated `httputil.ReverseProxy.Director` field. Replacing its empty callback with an empty `Rewrite` callback strips incoming forwarding headers and stops Go from automatically appending the client IP, changing the information delivered to upstream services. The migration must preserve the established forwarding behavior and add regression coverage.

## What

Symptom: With an empty `Rewrite` callback, upstream services receive proxied requests without the incoming forwarding headers or the appended client IP.

```text
Client sends an HTTP request to an upstream route configured by Admin
      |
      v
pkg/router: routerService.getRedirectMatcher
or pkg/ihttpimpl: newRouter
      |
      v
httputil.ReverseProxy.ServeHTTP selects the Rewrite path
and removes Forwarded and X-Forwarded-* headers
      |
      v
Migration to an empty Rewrite callback in
pkg/router/impl_reverseproxy.go or pkg/ihttpimpl/impl.go
      <-- fault: forwarding headers are not restored and client IP is not appended
      |
      v
Upstream service receives no forwarding headers or appended client IP (symptom)
```

Corrected behavior: Both HTTP proxy implementations use `Rewrite` while preserving incoming `X-Forwarded-For`, `X-Forwarded-Host`, `X-Forwarded-Proto`, and `Forwarded` headers and appending the client IP to the existing `X-Forwarded-For` chain, with regression assertions bound to the corresponding subtest.

## How

Decisions:

- Place one forwarding-header compatibility callback in the shared `pkg/goutils/httpu` utilities and use it in both proxies; keep destination selection and URL rewriting in the existing routing matchers.
- Build outgoing forwarding headers explicitly, because Go's convenience generation also replaces host and protocol metadata with values from the current request.
- Apply header changes only to the outbound request and preserve Go's hop-by-hop filtering: incoming forwarding values designated by the `Connection` header must remain excluded when restoring headers.
- Verify header semantics with focused utility tests and verify callback integration through both routers using their existing reverse-proxy tests.

Assumptions:

- None

Out of scope:

- Restoring legacy handling of malformed query parameters; query normalization follows the standard `Rewrite` behavior.

References:

- [Shared HTTP utility conventions](../../../../../pkg/goutils/httpu/utils.go)
- [Main router destination selection](../../../../../pkg/router/impl_reverseproxy.go)
- [Legacy HTTP router destination selection](../../../../../pkg/ihttpimpl/impl.go)
- [Main router integration coverage](../../../../../pkg/sys/it/reverseproxy_test.go)
- [Legacy HTTP router integration coverage](../../../../../pkg/ihttpimpl/impl_test.go)
- [Go proxy rewriting, forwarding headers, and hop-by-hop filtering](https://pkg.go.dev/net/http/httputil#ReverseProxy)

## Construction

- [x] create: [goutils/httpu/reverseproxy_test.go](../../../../../pkg/goutils/httpu/reverseproxy_test.go)
  - Add `TestRestoreForwardedHeaders` to exercise the shared callback through `httputil.ReverseProxy` with a buffered-channel transport, so assertions cover the standard library's header cleanup order without a shared request variable.
  - Cover absent forwarding headers, existing single and multiple header values, IPv4 and IPv6 peers, invalid remote addresses, and explicit nil versus empty `X-Forwarded-For` values.
  - Cover comma-separated, mixed-case `Connection` tokens across multiple header values: nominated forwarding values stay excluded, while a valid peer IP is still appended when incoming `X-Forwarded-For` is nominated.
  - Compare the complete outgoing header map to check that absent host/protocol headers are not generated, unrelated hop-by-hop headers remain removed, and end-to-end headers survive; retain URL, host, and inbound-header mutation and aliasing checks.
  - After implementation, run `go test ./pkg/goutils/httpu -run '^TestRestoreForwardedHeaders$' -count=1 -timeout=3m`.

- [x] update: [sys/it/reverseproxy_test.go](../../../../../pkg/sys/it/reverseproxy_test.go)
  - Extend `TestBasicUsage_ReverseProxy` with absent and existing forwarding-header cases for ordinary, rewritten, default, and domain routes; assert the incoming chain plus the connected client IP and preservation of the other forwarding headers.
  - Send domain-route requests with a host matching the configured `localhost` entry, so the test exercises domain routing rather than another matching path rule.
  - Send cloned upstream requests through a buffered channel and assert on the active subtest goroutine; retain method, path, and host checks, verify query combination and trailing or double slashes, and echo the POST body to verify request delivery and response forwarding.
  - Use `httptest.NewServer` and its client for lifecycle management; send observations without blocking on a full channel and consume available observations before response assertions, so failed cases cannot block server cleanup.
  - After implementation, run `go test ./pkg/sys/it -run '^TestBasicUsage_ReverseProxy$' -count=1 -timeout=3m`.

- [x] update: [pkg/ihttpimpl/impl_test.go](../../../../../pkg/ihttpimpl/impl_test.go)
  - Extend `TestReverseProxy` to assert forwarding headers for configured and default proxy routes with and without incoming forwarding metadata.
  - Use a table of paths, upstream destinations, response statuses, and bodies to retain all 14 route cases, including the local static file and missing-file 404; assert that static requests never reach the upstream.
  - Capture cloned upstream requests through a buffered channel, bind assertions to their subtests, and use `httptest.NewServer` and its client for cleanup; nonblocking sends and receives prevent failed cases from blocking later handlers.
  - Run both forwarding-header cases for every path; specify the expected `X-Forwarded-For` chain and compare the other forwarding headers directly with their incoming values, preserving absent-header checks.
  - After implementation, run `go test ./pkg/ihttpimpl -run '^TestReverseProxy$' -count=1 -timeout=3m`.

- [x] create: [goutils/httpu/reverseproxy.go](../../../../../pkg/goutils/httpu/reverseproxy.go)
  - Provide `RestoreForwardedHeaders(*httputil.ProxyRequest)` as the shared `Rewrite` callback for forwarding-header compatibility.
  - Restore only the four forwarding headers from independent copies of eligible inbound values; exclude values nominated by any case-insensitive, whitespace-trimmed `Connection` token.
  - Append a successfully parsed peer IP to the retained `X-Forwarded-For` chain, folding multiple values with comma-space separators; retain explicit nil suppression and preserve existing values when the remote address cannot be parsed.
  - Apply suppression after hop-by-hop exclusions: a removed incoming `X-Forwarded-For` value does not suppress generation of a valid peer IP.
  - Leave inbound data, destination selection, and standard query normalization untouched.

- [x] update: [pkg/router/impl_reverseproxy.go](../../../../../pkg/router/impl_reverseproxy.go)
  - Replace the empty `Director` with the shared forwarding-header `Rewrite` callback and import the HTTP utility package.
  - Keep existing route matching and destination rewriting, and update the callback comment to describe that responsibility split.

- [x] update: [pkg/ihttpimpl/impl.go](../../../../../pkg/ihttpimpl/impl.go)
  - Construct the proxy with the same shared `Rewrite` callback while retaining existing redirection matching and URL rewriting.
  - Format changed Go files and run the focused tests listed above plus `go test ./pkg/router -count=1 -timeout=3m`; check the diff for whitespace errors and run lint scoped to changed Go code using the repository configuration.
  - Verify the simplified tests with the race detector and compare covered production blocks before and after simplification: all 51 subtests pass, the same 306 production blocks remain covered, and both forwarding-helper functions retain 100% statement coverage.

- [x] update: [goutils/httpu/README.md](../../../../../pkg/goutils/httpu/README.md)
  - Expand the package description and feature list to cover forwarding-header restoration, listener addresses, discard validation, and socket-error matching alongside the HTTP client policies.
  - Link features to public API entry points, including the forwarding callback and reader-body replay interface; retain the Windows cleanup notes and example-test link.

- [x] fix: [goutils/httpu/README.md](../../../../../pkg/goutils/httpu/README.md)
  - Align both code examples with the stated optional authorization, accepted statuses, and retry behavior; include each snippet's package declaration and required imports so the committed examples compile independently.
  - Verify both snippets compile and meet [ar-generate-readme.md](../../../../../.augment/rules/ar-generate-readme.md): at most 50 lines without the package and 20 with it, basic and configured usage in the latter, and all README lines at most 72 characters.
