# retrier

`retrier` is a small Go package providing configurable retry logic with exponential backoff, jitter, and reset functionality. It simplifies re-executing operations that may intermittently fail, with flexible error-handling policies.

## Usage

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

initialDelay := 200 * time.Millisecond
maxIntervaDelay := 5 * time.Second
cfg := retrier.NewConfigExponentialBackoff(initialDelay, maxDelay)

res, err := retrier.Retry(ctx, cfg, func() (string, error) {
    return fetchRemoteResource()
})
if err != nil {
    log.Fatalf("Operation failed: %v", err)
}
fmt.Println("Result:", res)
```

## Config

* `InitialDelay    time.Duration`  – starting delay before first retry
* `MaxDelay        time.Duration`  - upper bound for backoff (0 allowed only if Multiplier==1)
* `Multiplier      float64`        – exponential growth factor (>=1)
* `JitterFactor    float64`        – randomness factor in \[0,1]
* `ResetAfter      time.Duration`  – duration after which backoff resets to initial
* `OnError`                        - callback called before each retry
* `RetryOnlyOn []error`            – retry only on listed errors and abort on any other error; empty means retry all non-context errors
* `Acceptable []error`             – treat these errors as success and return no error

## Helper Functions

* `Retry[T any](ctx context.Context, cfg Config, op func() (T, error)) (T, error)` – retries a function returning a value and error.
* `RetryErr(ctx context.Context, cfg Config, op func() error) error` – convenience wrapper for operations that return only an error.
