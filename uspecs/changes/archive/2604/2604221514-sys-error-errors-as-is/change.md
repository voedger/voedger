---
registered_at: 2026-04-22T15:03:05Z
change_id: 2604221503-sys-error-errors-as-is
baseline: f91d1438f6fbf47ffd247bb8d06e47cfdff55e6e
issue_url: https://untill.atlassian.net/browse/AIR-3666
archived_at: 2026-04-22T15:14:57Z
---

# Change request: Support errors.As and errors.Is by coreutils.SysError

## Why

`errors.Is(processors.ErrWSNotInited, processors.ErrWSNotInited)` returns `false` because `coreutils.SysError` does not implement the `Is` and `As` methods required by the standard `errors` package. See [issue.md](issue.md) for details.

## What

Extend `coreutils.SysError`:

- Implement `Is(target error) bool` method
- Implement `As(target any) bool` method
