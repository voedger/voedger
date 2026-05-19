---
registered_at: 2026-05-18T08:35:12Z
change_id: 2605180835-actualizer-log-cancel-cause
baseline: 1fd58491a65eda9ce4263c44f09aad0210daf6a4
issue_url: https://untill.atlassian.net/browse/AIR-3956
archived_at: 2026-05-18T10:01:58Z
---

# Change request: Async actualizer: include readCtx error in the retrier error log

## Why

When an async actualizer's `readCtx` is canceled because of a real error (projector failure, storage error, etc.), in-flight reads on the now-canceled context return `context.Canceled`, which is what the retrier's `OnError` at `pkg/processors/actualizers/async.go:96` ends up logging. The original error stored in `asyncActualizerContextState.err` is not included in the log, so the actual reason for the cancellation is lost — as observed in production (`ap.air.UpdateTerminalsOverview`).

## What

Include the stored `readCtx` error in the actualizer retrier error log so the root cause is visible alongside the masking `context.Canceled`.

## Construction

- [x] update: [actualizers/async.go](../../../../../pkg/processors/actualizers/async.go)
  - update: `retrierCfg.OnError` to append `a.readCtx.error()` to the error logged at line 96 when it is non-nil, so the original cancel cause is visible together with `opErr`
  - update: change the log stage from `a.name` to a fixed identifier to distinguish from the in-projector `"ap.error"` branch and make it obvious that the error did not originate inside the projector function
  - update: final stage name is `"actualizer.error"`, and the log context is decorated with `logger.LogAttr_Extension = "sys._Actualizer"` via `logger.WithContextAttrs`, so the line is grouped under a synthetic extension consistent with other actualizer-emitted logs
- [x] update: [actualizers/async_test.go](../../../../../pkg/processors/actualizers/async_test.go)
  - add: `Test_AsycActualizerErrorCause` exercising the new branch via the existing `flakyAppParts` decorator (failing `WaitForBorrow` on the 2nd call), asserting the log line carries `extension=sys._Actualizer`, `stage=actualizer.error` and contains `cause: flaky WaitForBorrow failure`
