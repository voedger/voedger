# Implementation plan: Skip singletons in ActualizeSequencesFromPLog

## Technical design

- [x] update: [apps/sequences--arch.md](../../../../specs/prod/apps/sequences--arch.md)
  - add: Document that `ActualizeSequencesFromPLog` must skip singleton CUDs because they have predefined IDs (65536–66047) that would corrupt the `RecordIDSequence` state

## Construction

- [x] update: [pkg/appparts/internal/seqstorage/impl.go](../../../../../pkg/appparts/internal/seqstorage/impl.go)
  - update: Skip singleton CUDs in `ActualizeSequencesFromPLog` by checking `appdef.ISingleton` type assertion on `ss.appDef.Type(cud.QName())`
- [x] update: [pkg/appparts/internal/seqstorage/impl_test.go](../../../../../pkg/appparts/internal/seqstorage/impl_test.go)
  - add: Singleton CDoc type (`testSingletonCDocQName`) to `setupTestAppDef`
  - add: Test case in `TestSequenceActualization` with a singleton CUD verifying it does not appear in the batch
