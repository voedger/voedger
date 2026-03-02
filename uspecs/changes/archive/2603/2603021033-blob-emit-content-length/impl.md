# Implementation plan: Emit blob size in Content-Length header on read

## Construction

### Refactor read path

- [x] update: [pkg/processors/blobber/impl_read.go](../../../pkg/processors/blobber/impl_read.go)
  - update: `initResponse` — add `httpu.ContentLength` (from `bw.blobState.Size`) to the `okResponseIniter` call alongside `Content-Type` and `BlobName`

### Add `BLOBSize` to `BLOBReader`

- [x] update: [pkg/iblobstorage/types.go](../../../pkg/iblobstorage/types.go)
  - add: `BLOBSize uint64` field to `BLOBReader`

- [x] update: [pkg/coreutils/federation/impl.go](../../../pkg/coreutils/federation/impl.go)
  - add: private `readBLOB` method with `blobSizeFromHeader` helper — populates `BLOBSize` from `Content-Length` response header
  - update: `ReadBLOB` and `ReadTempBLOB` — delegate to `readBLOB`

### Tests

- [x] update: [pkg/sys/it/impl_blob_test.go](../../../pkg/sys/it/impl_blob_test.go)
  - add: `require.EqualValues(len(expBLOB), blobReader.BLOBSize)` in 6 locations across `TestBasicUsage_Persistent` (2 reads), `TestBasicUsage_Temporary`, `TestAPIv1v2BackwardCompatibility`, `TestODocWithBLOB`, `TestBLOBsPseudoWSIDToAppWSID`
