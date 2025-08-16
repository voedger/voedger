# Retrier

`retrier` is a small Go package providing configurable retry logic with exponential backoff, jitter, and reset functionality. It simplifies re-executing operations that may intermittently fail, with flexible error-handling policies.

## Features

- **Full Jitter** — Random delay adjustments to avoid thundering herd issues.
  - based on [Claudflare Full Jitter algorythm](https://aws.amazon.com/ru/blogs/architecture/exponential-backoff-and-jitter/)
- **Reset Logic** — Reset delay after a period of inactivity.
- **Custom Error Handling** — Fine-grained control over retry decisions via `OnError`
