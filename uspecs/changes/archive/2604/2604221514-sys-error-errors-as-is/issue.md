# AIR-3666: Support errors.As and errors.Is by coreutils.SysError

- **Key**: AIR-3666
- **Type**: Task
- **Status**: In Progress
- **Assignee**: d.gribanov@dev.untill.com
- **URL**: <https://untill.atlassian.net/browse/AIR-3666>

## Why

`errors.Is(processors.ErrWSNotInited, processors.ErrWSNotInited)` returns `false`.

## What

Support `errors.As` and `errors.Is` in `coreutils.SysError`.
