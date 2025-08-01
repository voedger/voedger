# Retrier

Go package providing retry logic with exponential backoff, full jitter, and reset capabilities.

## Features

* Exponential backoff with configurable multiplier
* Full jitter to randomize delays
* Automatic reset of backoff interval after a configurable duration
* `Run` method to retry operations until success or context cancellation
* Generic `Retry` helper for functions returning values
* Hook for observing retry attempts via `OnRetry` callback

## Installation

```bash
go get github.com/untill/retrier
```

## Usage

```go
import (
  "context"
  "fmt"
  "time"

  "github.com/untill/retrier"
)

func main() {
  ctx := context.Background()
  cfg := retrier.NewDefaultConfig()
  cfg.InitialInterval = 100 * time.Millisecond
  cfg.MaxInterval = 5 * time.Second
  cfg.Multiplier = 2
  cfg.JitterFactor = 0.5
  cfg.ResetAfter = 30 * time.Second
  cfg.OnRetry = func(attempt int, delay time.Duration) {
    fmt.Printf("Retry attempt %d, next delay %v\n", attempt, delay)
  }

  // Using Retrier.Run
  r, err := retrier.New(cfg)
  if err != nil {
    log.Fatalf("invalid config: %v", err)
  }

  err = r.Run(ctx, func() error {
    // perform the operation that may fail
    return doWork()
  })
  if err != nil {
    log.Fatalf("operation failed: %v", err)
  }

  // Using generic Retry helper
  result, err := retrier.Retry(ctx, cfg, func() (string, error) {
    return fetchData()
  })
  if err != nil {
    log.Fatalf("fetch data failed: %v", err)
  }
  fmt.Println("Received:", result)
}
```

## Configuration

The `Config` struct controls retry behavior:

| Field           | Type                       | Description                                                                          |
| --------------- | -------------------------- | ------------------------------------------------------------------------------------ |
| InitialInterval | `time.Duration`            | Starting delay before first retry                                                    |
| MaxInterval     | `time.Duration`            | Maximum delay between retries                                                        |
| Multiplier      | `float64`                  | Factor by which the delay increases after each retry (must be ≥ 1)                   |
| JitterFactor    | `float64`                  | Fraction of `base` interval used to randomize delays (0 to 1)                        |
| ResetAfter      | `time.Duration`            | Duration after which the backoff interval resets to `InitialInterval` (0 = disabled) |
| OnRetry         | `func(int, time.Duration)` | Optional callback invoked before each retry with attempt number and upcoming delay   |

Use `NewDefaultConfig()` to get a `Config` with sensible defaults:

* `InitialInterval`: 0
* `MaxInterval`: 0
* `Multiplier`: 2
* `JitterFactor`: 0.5
* `ResetAfter`: 0 (disabled)

Configure individual fields after calling `NewDefaultConfig()`.

## API Reference

### `func NewDefaultConfig() Config`

Returns a `Config` initialized with default `Multiplier` and `JitterFactor`.

### `func New(cfg Config) (*Retrier, error)`

Creates a `Retrier` instance with the provided configuration. Returns `ErrInvalidConfig` if any parameters are out of valid range.

### `func (r *Retrier) NextDelay() time.Duration`

Calculates the next retry delay based on exponential backoff, full jitter, and reset logic.

### `func (r *Retrier) Run(ctx context.Context, operation func() error) error`

Retries the given operation until it succeeds or the context is canceled. Invokes `OnRetry` before sleeping between attempts.

### `func Retry[T any](ctx context.Context, cfg Config, fn func() (T, error)) (T, error)`

Convenience function combining `New` and `Run` for operations that return a value.

## Error Handling

* `ErrInvalidConfig` is returned by `New` when any configuration parameter is invalid.
* `Run` and `Retry` return the context error if the context is canceled.
