# Implementation: Temporary BLOBs DurationType stores days directly

## Construction

- [x] update: [pkg/iblobstorage/utils.go](../../../pkg/iblobstorage/utils.go)
  - Change `Seconds()` method formula from `dt*dt*secondsInDay` to `dt*secondsInDay`
- [x] update: [pkg/iblobstorage/types.go](../../../pkg/iblobstorage/types.go)
  - Update comment on DurationType to reflect new behavior (stores days directly)
- [x] update: [pkg/iblobstorage/utils_test.go](../../../pkg/iblobstorage/utils_test.go)
  - Update `TestDurationSeconds` expectations: `DurationType(2)` should return `86400*2` instead of `86400*4`
- [x] update: [pkg/sys/it/impl_blob_test.go](../../../pkg/sys/it/impl_blob_test.go)
  - No change needed: uses `DurationType_1Day` (value=1), and 1*1=1 equals 1*1
