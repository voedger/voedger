# AIR-3572: fix TestRecovery

- **Type:** Bug
- **Status:** In Progress
- **Assignee:** d.gribanov@dev.untill.com

## Description

CI run failure: <https://github.com/voedger/voedger/actions/runs/24276104634/job/70890230460>

```
FAIL: TestRecovery (0.01s)
    impl_test.go:323: no log line contains all of the expected substrings
```

The test fails intermittently because recovery does not run after `restartCmdProc`. The `appsPartitions` map is not cleared when the service loop exits via the `for vvmCtx.Err() == nil` condition check (as opposed to the `<-vvmCtx.Done()` select case). When `app.cancel()` is called while the service goroutine is still executing post-`sendResponse` code, the loop exits without entering the `select`, skipping the `appsPartitions` clearing. On the next run, the old partition data is found and recovery is skipped entirely — no recovery log lines are written, causing `HasLine` at `impl_test.go:323` to fail.
