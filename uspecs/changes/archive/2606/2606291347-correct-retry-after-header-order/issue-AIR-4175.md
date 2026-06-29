# correct order for Retry-After header

- URL: https://untill.atlassian.net/browse/AIR-4175
- ID: AIR-4175
- State: in-progress
- Author: Denis Gribanov
- Assignees: Denis Gribanov
- Labels: none

## Why

In BLOB handler (and probably in other places) `Retry-After` header is sent after status code. So it is dropped by the client.

## What

- use `httpu.RetryAfter` const instead of `"Retry-After"` string
- first send `Retry-After` and any others, status code the last
