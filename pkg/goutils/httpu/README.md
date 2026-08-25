# httpu

Package httpu provides a configurable HTTP client for requests with
reusable retry, status-validation, and response-handling policies.

## Problem

The standard HTTP client leaves callers to repeat request setup, retry
decisions, status validation, and response-body cleanup. httpu
centralizes those policies behind one request API.

<details>
<summary>Without httpu</summary>

```go
import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
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

- **Retry policies** - Retry transient statuses with configurable rules
  - [Default policies: consts.go#L45](consts.go#L45)
  - [Policy replacement: impl_opts.go#L158](impl_opts.go#L158)
  - [Status policies: impl_opts.go#L96](impl_opts.go#L96)
  - [Retry-After support: impl_opts.go#L87](impl_opts.go#L87)
  - [Error matchers: impl_opts.go#L149](impl_opts.go#L149)

- **Request options** - Configure methods, metadata, and
  accepted statuses
  - [Cookies: impl_opts.go#L58](impl_opts.go#L58)
  - [Headers: impl_opts.go#L66](impl_opts.go#L66)
  - [Authorization: impl_opts.go#L81](impl_opts.go#L81)
  - [Expected statuses: impl_opts.go#L74](impl_opts.go#L74)
  - [Option validation: impl_opts.go#L226](impl_opts.go#L226)

- **Response modes** - Buffer, discard, or stream response bodies
  - [Custom handlers: impl_opts.go#L17](impl_opts.go#L17)
  - [Long polling: impl_opts.go#L30](impl_opts.go#L30)
  - [Response discard: impl_opts.go#L46](impl_opts.go#L46)

- **[Body replay](impl.go#L48)** - Preserve reader payloads on retries

- **Client lifecycle** - Reuse connections or inject a transport
  - [Default client: provide.go#L15](provide.go#L15)
  - [Custom transport: provide.go#L30](provide.go#L30)

## Platform Support

On Windows, response cleanup tolerates `WSAECONNRESET` while
discarding a response body. Callers can match other Windows socket
errors with `IsWSAEError` when configuring `WithRetryOnError`.

## Use

See [example](example_test.go)
