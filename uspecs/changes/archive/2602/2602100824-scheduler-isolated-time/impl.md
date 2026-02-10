# Implementation plan

## Construction

### Scheduler isolated time

- [x] update: [testingu/mocktime.go](../../pkg/goutils/testingu/mocktime.go)
  - add: `NewIsolatedTime() timeu.ITime` to `IMockTime` interface
  - add: `NewIsolatedTime()` implementation on `mockedTime` that clones current time independently
- [x] update: [schedulers/impl_schedulers.go](../../pkg/processors/schedulers/impl_schedulers.go)
  - add: `time timeu.ITime` field to `schedulers` struct
  - update: `newSchedulers()` to detect `NewIsolatedTime()` capability and create isolated time
  - add: `SchedulersTime() timeu.ITime` method on `schedulers`
  - update: `newSchedulers()` return type from `appparts.ISchedulerRunner` to `*schedulers`

### VIT scheduler time control

- [x] update: [appparts/interface.go](../../pkg/appparts/interface.go)
  - add: `SchedulersTime() timeu.ITime` method to `ISchedulerRunner` interface
- [x] update: [appparts/const_null.go](../../pkg/appparts/const_null.go)
  - add: `SchedulersTime()` implementation on `nullSchedulerRunner`
- [x] update: [appparts/impl_test.go](../../pkg/appparts/impl_test.go)
  - add: `SchedulersTime()` implementation on `mockSchedulerRunner`
- [x] update: [vvm/types.go](../../pkg/vvm/types.go)
  - add: `ISchedulerRunner appparts.ISchedulerRunner` field to `VVM` struct
- [x] update: [vvm/wire_gen.go](../../pkg/vvm/wire_gen.go)
  - update: generated wire code to include `ISchedulerRunner` in `VVM`
- [x] update: [vit/impl.go](../../pkg/vit/impl.go)
  - add: `SchedulerTimeAdd(dur time.Duration)` method on `VIT`
  - add: `SchedulerNow() time.Time` method on `VIT`

### Test updates

- [x] update: [sys/it/impl_jobs_test.go](../../pkg/sys/it/impl_jobs_test.go)
  - update: `TestJobs_BasicUsage_Builtin` to use `vit.SchedulerTimeAdd()` instead of global MockTime
  - update: `TestJobs_BasicUsage_Sidecar` to use `vit.SchedulerTimeAdd()`
  - update: `TestJobs_SendEmail` to advance scheduler time before capturing email
  - update: `isJobFiredForCurrentInstant_builtin` helper to use `vit.SchedulerTimeAdd()`
  - update: `waitForSidecarJobCounter` helper to use `vit.SchedulerTimeAdd()`
