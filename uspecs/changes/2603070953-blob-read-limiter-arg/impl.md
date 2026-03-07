# Implementation plan: Add read limiter arg to blobprocessor IRequestHandler blob reads

## Construction

### Read API surface

- [x] update: [pkg/processors/blobber/interface.go](../../../pkg/processors/blobber/interface.go)
  - update: add `iblobstorage.RLimiterType` arg to `HandleRead`, `HandleRead_V2`, and `HandleReadTemp_V2`

### Read request plumbing

- [x] update: [pkg/processors/blobber/impl_requesthandler.go](../../../pkg/processors/blobber/impl_requesthandler.go)
  - update: accept the read limiter in read handlers and pass it into the read message
- [x] update: [pkg/processors/blobber/types.go](../../../pkg/processors/blobber/types.go)
  - add: `rLimiter iblobstorage.RLimiterType` to the read message payload
- [x] update: [pkg/processors/blobber/impl_read.go](../../../pkg/processors/blobber/impl_read.go)
  - update: use the request-provided read limiter when calling `blobStorage.ReadBLOB`

### Caller updates

- [x] update: [pkg/router/impl_blob.go](../../../pkg/router/impl_blob.go)
  - update: pass `iblobstoragestg.RLimiter_Null` to API v1 blob reads
- [x] update: [pkg/router/impl_apiv2.go](../../../pkg/router/impl_apiv2.go)
  - update: pass `iblobstoragestg.RLimiter_Null` to API v2 blob reads

### Tests

- [x] verify: [pkg/sys/it/impl_blob_test.go](../../../pkg/sys/it/impl_blob_test.go)
  - verify: existing coverage for persistent, temporary, and API v2 blob reads passes after threading `iblobstoragestg.RLimiter_Null` through the read handlers