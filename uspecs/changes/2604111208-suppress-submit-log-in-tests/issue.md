# AIR-3551: Do not log on submit to processors failure in tests

- **Type:** Sub-task
- **Status:** In Progress
- **Assignee:** d.gribanov@dev.untill.com

## Problem

During stress testing the console is overflowed with "no processors available" error messages. These come from `replyCommandBusy` and `replyQueryBusy` in `pkg/vvm/impl_requesthandler.go` which call `logger.ErrorCtx` every time `procbus.Submit` returns `false`.

In test environments this is expected behaviour under load and produces noise that obscures real failures.

## Constraints

- Do not use the `testing` package to detect test mode
- Do not use command-line args inspection to detect test mode

## Solution

- Add a test-mode detection mechanism that satisfies the constraints above
- Skip `logger.ErrorCtx` calls in `replyCommandBusy` and `replyQueryBusy` when test mode is active
- Scope: `pkg/vvm` (`impl_requesthandler.go`)
