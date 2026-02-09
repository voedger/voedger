# MockTime

Controllable time implementation for deterministic testing of
time-dependent code.

## Problem

Testing time-dependent code with real timers creates slow, flaky tests
that wait for actual time to pass and can fail due to timing variations.

<details>
<summary>Without MockTime</summary>

```go
func TestCacheExpiration(t *testing.T) {
    cache := NewCache(5 * time.Second) // 5 second TTL
    cache.Set("key", "value")

    // Must wait real time - slow and unreliable
    time.Sleep(4 * time.Second)
    if !cache.Has("key") {
        t.Fatal("key should still exist")
    }

    // Wait more real time - test takes 6+ seconds
    time.Sleep(2 * time.Second)
    if cache.Has("key") {
        t.Fatal("key should be expired") // May fail on slow systems
    }
}

func TestRetryWithBackoff(t *testing.T) {
    start := time.Now()
    retrier := NewRetrier(100*time.Millisecond, 3) // 100ms, 300ms, 900ms

    err := retrier.Do(func() error {
        return errors.New("always fails")
    })

    elapsed := time.Since(start)
    // Flaky assertion - timing dependent
    if elapsed < 1300*time.Millisecond || elapsed > 1500*time.Millisecond {
        t.Fatal("unexpected retry timing")
    }
}
```

</details>

<details>
<summary>With MockTime</summary>

```go
import "github.com/voedger/voedger/pkg/goutils/testingu"

func TestCacheExpiration(t *testing.T) {
    cache := NewCache(testingu.MockTime, 5*time.Second)
    cache.Set("key", "value")

    testingu.MockTime.Add(4 * time.Second) // Instant
    require.True(t, cache.Has("key"))

    testingu.MockTime.Add(2 * time.Second) // Instant
    require.False(t, cache.Has("key")) // Deterministic
}
```

</details>

## Features

- **[Time control](mocktime.go#L22)** - Advance time instantly with Add()
  - [Frozen time: Now() returns consistent values](mocktime.go#L47)
  - [Timer management: tracks and fires timers precisely](mocktime.go#L102)
  - [Sleep simulation: Add() advances time without waiting](mocktime.go#L98)

- **[Timer mocking](mocktime.go#L53)** - Replace real timers with controllable ones
  - [Timer creation: NewTimerChan() creates mockable timers](mocktime.go#L53)
  - [Timer firing: checkTimers() fires expired timers](mocktime.go#L102)
  - [Immediate firing: FireNextTimerImmediately() for edge cases](mocktime.go#L71)

- **[Global instance](mocktime.go#L16)** - Shared MockTime prevents timing inconsistencies

- **[Isolated time](mocktime.go#L25)** - Independent mock time unaffected by global time advances

## Use

See [basic usage test](mocktime_test.go)
