# Retrier

`retrier` is a small Go package providing configurable retry logic with exponential backoff, jitter, and reset functionality. It simplifies re-executing operations that may intermittently fail, with flexible error-handling policies.

## Features

- **Configurable Backoff**
  - **Constant backoff** (`NewConfigConstantBackoff`)
  - **Exponential backoff** (`NewConfigExponentialBackoff`)
- **Jitter** — Random delay adjustments to avoid thundering herd issues.
- **Reset Logic** — Reset delay after a period of inactivity.
- **Custom Error Handling** via `HandleError` callback:

| `HandleError` result    | Meaning                                           |
| --------- | --------------------------------------------------------------- |
| `DoRetry` | Wait the computed delay, then try again                         |
| `Accept`  | Stop retrying, return success (even if the last attempt failed) |
| `Abort`   | Stop retrying and return the last error                         |

## Timing Confguration

| Field          | Type            | Description                                                     |
| -------------- | --------------- | --------------------------------------------------------------- |
| `InitialDelay` | `time.Duration` | Starting delay before the first retry                           |
| `MaxDelay`     | `time.Duration` | Maximum delay cap (`0` allowed only if `Multiplier == 1`)       |
| `Multiplier`   | `float64`       | Exponential growth factor (must be ≥ 1)                         |
| `JitterFactor` | `float64`       | Randomness factor in the range \[0, 1]                          |
| `ResetAfter`   | `time.Duration` | Time period after which the delay resets back to `InitialDelay` |
