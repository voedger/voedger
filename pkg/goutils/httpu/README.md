# httpu

Package httpu provides an HTTP client with reusable request, retry,
and response policies. It also supplies forwarding-header and
listen-address helpers for HTTP servers and reverse proxies.

## Problem

HTTP callers otherwise repeat request setup, status checks, retry
decisions, and response-body cleanup. The examples fetch a response
body with optional authorization, accepting HTTP 200 or 201 and
disabling retries.

<details>
<summary>Without httpu</summary>

```go
package example

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

func fetch(ctx context.Context, url, token string) (string, error) {
	// Boilerplate: every request needs the same retry loop
	client := &http.Client{}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(
			ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		if resp.StatusCode == http.StatusServiceUnavailable {
			// Common mistake: retry without releasing the connection
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			delay := time.Duration(attempt+1) * 100 * time.Millisecond
			if seconds, err := strconv.Atoi(
				resp.Header.Get("Retry-After")); err == nil && seconds > 0 {
				delay = time.Duration(seconds) * time.Second
			}
			time.Sleep(delay)
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return "", readErr
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf(
				"unexpected status: %d", resp.StatusCode)
		}
		return string(body), nil
	}
	return "", errors.New("retry limit reached")
}
```

</details>

<details>
<summary>With httpu</summary>

```go
import (
	"context"
	"net/http"

	"github.com/voedger/voedger/pkg/goutils/httpu"
)

func fetch(url, token string) (string, error) {
	client, cleanup := httpu.NewIHTTPClient()
	defer cleanup()

	retry503 := httpu.WithRetryPolicy(httpu.WithRetryOnStatus(
		http.StatusServiceUnavailable,
		httpu.WithRespectRetryAfter(),
	))
	resp, err := client.Req(context.Background(), url, "",
		httpu.WithAuthorizeBy(token), retry503)
	if err != nil {
		return "", err
	}
	return resp.Body, nil
}
```

</details>

## Features

- **Client lifecycle** - Share request defaults and reuse connections

  - [Default client: provide.go#L15](provide.go#L15)
  - [Custom transport: provide.go#L30](provide.go#L30)
  - [Request interface: types.go#L37](types.go#L37)

- **Retry policies** - Retry selected statuses and errors with backoff

  - [Default policies: consts.go#L45](consts.go#L45)
  - [Policy replacement: impl_opts.go#L158](impl_opts.go#L158)
  - [Status policies: impl_opts.go#L96](impl_opts.go#L96)
  - [Retry-After support: impl_opts.go#L87](impl_opts.go#L87)
  - [Error matchers: impl_opts.go#L149](impl_opts.go#L149)

- **Request options** - Set methods, metadata, and accepted statuses

  - [Methods: impl_opts.go#L143](impl_opts.go#L143)
  - [Headers: impl_opts.go#L66](impl_opts.go#L66)
  - [Authorization: impl_opts.go#L81](impl_opts.go#L81)
  - [Expected statuses: impl_opts.go#L74](impl_opts.go#L74)
  - [Option validation: impl_opts.go#L226](impl_opts.go#L226)

- **Response modes** - Buffer, discard, or stream response bodies

  - [Buffered response: types.go#L20](types.go#L20)
  - [Custom handlers: impl_opts.go#L17](impl_opts.go#L17)
  - [Long polling: impl_opts.go#L30](impl_opts.go#L30)
  - [Response discard: impl_opts.go#L46](impl_opts.go#L46)

- **[Body replay](types.go#L41)** - Buffer `ReqReader` payloads
  for retries

- **[Discard validation](utils.go#L37)** - Reject GET requests with
  discarded responses when installed through `WithOptsValidator`

- **[Proxy forwarding](reverseproxy.go#L19)** - Restore incoming
  `Forwarded` and `X-Forwarded-For/Host/Proto` headers in a
  `httputil.ReverseProxy.Rewrite` callback, excluding incoming values
  named by `Connection`, and append the client IP to `X-Forwarded-For`
  without generating host or protocol headers or changing the URL

- **[Listen addresses](utils.go#L55)** - Use all interfaces for a fixed
  port or localhost with an OS-assigned port when the port is zero

- **[Dynamic localhost](utils.go#L64)** - Return a localhost address
  with an OS-assigned port for tests or local services

- **[Socket errors](utils.go#L44)** - Match wrapped Windows socket
  errors by code for custom error handling or retry policies

## Platform Support

On Windows, response cleanup tolerates `WSAECONNRESET` while
discarding a response body. Callers can match Windows socket errors
with `IsWSAEError` when configuring `WithRetryOnError`.

## Use

See [example](example_test.go)
