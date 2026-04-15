# Implementation plan: Fix TestRecovery

## Construction

- [x] update: [pkg/processors/command/provide.go](../../../../../pkg/processors/command/provide.go)
  - fix: race condition where `appsPartitions` was not cleared on service stop; `appsPartitions` clearing was only inside the `select`'s `<-vvmCtx.Done()` case, but the `for vvmCtx.Err() == nil` loop condition could exit the loop before reaching the `select` (when context was cancelled between loop iterations after `sendResponse` returned); moved clearing after the `for` loop so both exit paths converge on it
- [x] update: [pkg/processors/command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - fix: use `requestCtx` parameter instead of captured `vvmCtx` in the request handler callback passed to `bus.NewIRequestSender`; after `restartCmdProc` cancels `vvmCtx`, subsequent `NewCommandMessage` calls received a cancelled context as `RequestCtx()` — matching the production code pattern in `impl_requesthandler.go` which correctly uses the `requestCtx` parameter
