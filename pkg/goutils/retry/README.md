# Retrier

Go package providing retry logic with exponential backoff, full jitter, and reset capabilities.

## Usage

```go
import "github.com/untill/retrier"

// Basic retry
cfg := retrier.NewDefaultConfig()
cfg.InitialInterval = 100 * time.Millisecond
cfg.MaxInterval = 5 * time.Second

r, _ := retrier.New(cfg)
err := r.Run(ctx, func() error {
    return doWork()
})

// Generic helper
result, err := retrier.Retry(ctx, cfg, func() (string, error) {
    return fetchData()
})
```

## Configuration

Use `NewDefaultConfig()` for sensible defaults, then configure:

- `InitialInterval`: Starting delay before first retry
- `MaxInterval`: Maximum delay between retries
- `Multiplier`: Factor by which delay increases (≥ 1)
- `JitterFactor`: Randomization factor (0 to 1)
- `ResetAfter`: Duration to reset backoff (0 = disabled)
- `OnRetry`: Optional callback for retry attempts
