# timeu

Time abstraction interface enabling testable time-dependent code through
dependency injection.

## Problem

Testing time-dependent code with direct time.Now() calls and real timers
creates slow, flaky tests that depend on actual system time.

<details>
<summary>Without timeu</summary>

```go
func ProcessExpiredItems(cache map[string]Item) {
    now := time.Now() // Hard to mock
    for key, item := range cache {
        if now.After(item.ExpiresAt) {
            delete(cache, key)
        }
    }
}

func TestProcessExpiredItems(t *testing.T) {
    cache := map[string]Item{
        "key1": {ExpiresAt: time.Now().Add(-time.Hour)}, // Brittle
        "key2": {ExpiresAt: time.Now().Add(time.Hour)},
    }
    
    ProcessExpiredItems(cache)
    
    // Test depends on system time - may fail due to timing
    if len(cache) != 1 {
        t.Fatal("expected 1 item remaining")
    }
    
    // Cannot test timer-based expiration without real delays
    timer := time.NewTimer(time.Second)
    defer timer.Stop()
    select {
    case <-timer.C:
        // Test takes real time - slow and unreliable
    case <-time.After(2 * time.Second):
        t.Fatal("timeout")
    }
}
```

</details>

<details>
<summary>With timeu</summary>

```go
import "github.com/voedger/voedger/pkg/goutils/timeu"

func ProcessExpiredItems(cache map[string]Item, timeProvider timeu.ITime) {
    now := timeProvider.Now() // Mockable
    for key, item := range cache {
        if now.After(item.ExpiresAt) {
            delete(cache, key)
        }
    }
}

func TestProcessExpiredItems(t *testing.T) {
    mockTime := testingu.MockTime
    cache := map[string]Item{
        "key1": {ExpiresAt: mockTime.Now().Add(-time.Hour)},
        "key2": {ExpiresAt: mockTime.Now().Add(time.Hour)},
    }
    
    ProcessExpiredItems(cache, mockTime)
    require.Equal(t, 1, len(cache)) // Deterministic
}
```

</details>

## Features

- **[Time abstraction](time.go#L12)** - Interface for mockable time operations
  - [Current time: Now() returns current timestamp](time.go#L24)
  - [Timer creation: NewTimerChan() creates channel-based timers](time.go#L28)
  - [Sleep operation: Sleep() pauses execution](time.go#L33)

- **[Real implementation](time.go#L18)** - Production time provider
  - [Factory function: NewITime() creates real time instance](time.go#L18)
  - [Standard library delegation: realTime wraps time package](time.go#L22)

## Use

See [basic usage test](timeu_test.go)
