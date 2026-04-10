# Implementation plan: Log failed submit to processors

## Technical design

- [x] update: [apps/logging--td.md](../../../../../uspecs/specs/prod/apps/logging--td.md)
  - add: VVM Request Handler section with `vvm.submit` stage logging for query and command processor submit failures

## Construction

- [x] update: [impl_requesthandler.go](../../../../../pkg/vvm/impl_requesthandler.go)
  - add: `logger` import
  - add: `replyQueryBusy(ctx, isAPIv2, responder)` helper — logs `logger.ErrorCtx` with stage `vvm.submit`, API version, and replies 503
  - add: `replyCommandBusy(ctx, responder, partitionID)` helper — logs `logger.ErrorCtx` with stage `vvm.submit`, partition ID, and replies 503
  - update: replace inline `bus.ReplyErrf` calls on `procbus.Submit` failure with helper calls (4 places)
- [x] update: [impl_test.go](../../../../../pkg/sys/it/impl_test.go)
  - update: `Test503OnNoQueryProcessorsAvailable` — add `logger.StartCapture` and verify `stage=vvm.submit` log line with `no query processors v1 available`
