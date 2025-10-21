# httpu

HTTP client with automatic retry handling, flexible
configuration, and sensible defaults for resilient
distributed systems.

## Problem

Making reliable HTTP requests in distributed systems
requires handling transient failures, connection resets,
rate limiting, and status-based retries. Without a
dedicated utility, developers must manually implement
retry logic, error handling, and configuration for each
request.

<details>
<summary>Without httpu</summary>

```go
// Boilerplate: manual retry logic, error handling
client := &http.Client{}
var resp *http.Response
var err error
for attempt := 0; attempt < 3; attempt++ {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err = client.Do(req)
	if err != nil {
		// Handle connection resets manually
		if strings.Contains(err.Error(), "WSAECONNRESET") {
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
			continue
		}
		return nil, err
	}
	if resp.StatusCode == http.StatusServiceUnavailable {
		// Manual retry-after parsing
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			// Parse and sleep...
		}
		continue
	}
	if resp.StatusCode == http.StatusOK {
		break
	}
	return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
}
body, _ := io.ReadAll(resp.Body)
```

</details>

<details>
<summary>With httpu</summary>

```go
import "github.com/voedger/voedger/pkg/goutils/httpu"

httpClient, cleanup := httpu.NewIHTTPClient()
defer cleanup()

resp, err := httpClient.Req(
	context.Background(),
	url,
	"",
	httpu.WithAuthorizeBy(token),
	httpu.WithRetryOnStatus(
		http.StatusServiceUnavailable,
		httpu.WithRespectRetryAfter(),
	),
)
```

</details>

## Features

- **[Automatic retry handling](impl.go#L111)** - Built-in
  retry logic for connection errors and configurable
  HTTP status codes with exponential backoff
  - [{Retry-After header support: impl.go#L139}](impl.go#L139)
  - [{Error matchers: impl_opts.go#L149}](impl_opts.go#L149)

- **[Flexible request options](impl_opts.go)** - Chainable
  option functions for headers, cookies, authentication,
  and response handling
  - [{Headers and cookies: impl_opts.go#L58}](impl_opts.go#L58)
  - [{Authorization: impl_opts.go#L81}](impl_opts.go#L81)
  - [{Custom validators: impl_opts.go#L228}](impl_opts.go#L228)

- **[Status code expectations](impl_opts.go#L74)** - Specify
  expected HTTP status codes with convenience helpers
  (Expect204, Expect404, etc.)
  - [{Expected codes validation: impl.go#L161}](impl.go#L161)

- **[Response handling modes](impl_opts.go#L17)** - Support
  for custom response handlers, long polling, and
  response discarding
  - [{Response handler: impl_opts.go#L17}](impl_opts.go#L17)
  - [{Long polling: impl_opts.go#L30}](impl_opts.go#L30)

- **[Connection optimization](provide.go#L17)** - TCP
  linger configuration and connection pooling for
  efficient resource usage

## Platform Support

Includes Windows-specific socket error handling for WSAECONNRESET and
WSAECONNREFUSED errors to improve retry behavior on Windows systems.

## Use

See [example](example_test.go)

