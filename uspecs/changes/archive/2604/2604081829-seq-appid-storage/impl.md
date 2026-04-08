# Implementation plan: Consider AppID in VVM sequence storage

## Technical design

- [x] update: [apps/sequences--arch.md](../../../../specs/prod/apps/sequences--arch.md)
  - add: VVM storage key structure section documenting per-app scoping of PLog offset and sequence number keys
  - update: `IVVMSeqStorageAdapter` description to note that `GetPLogOffset`/`PutPLogOffset` accept `appID` parameter

## Construction

### Interface change

- [x] update: [pkg/isequencer/interface.go](../../../../../pkg/isequencer/interface.go)
  - update: Add `appID ClusterAppID` parameter to `GetPLogOffset` and `PutPLogOffset` in `IVVMSeqStorageAdapter`

### Implementation

- [x] update: [pkg/vvm/storage/impl_seqstorage.go](../../../../../pkg/vvm/storage/impl_seqstorage.go)
  - update: `pLogOffsetPKeySize` from `4 + 2` to `4 + 4 + 2` to accommodate AppID
  - update: `GetPLogOffset` to accept `appID` and encode it into PKey
  - update: `PutPLogOffset` to accept `appID` and encode it into PKey

### Caller update

- [x] update: [pkg/appparts/internal/seqstorage/impl.go](../../../../../pkg/appparts/internal/seqstorage/impl.go)
  - update: `WriteValuesAndNextPLogOffset` to pass `ss.appID` to `PutPLogOffset`
  - update: `ReadNextPLogOffset` to pass `ss.appID` to `GetPLogOffset`

### Tests

- [x] update: [pkg/vvm/storage/impl_seqstorage_test.go](../../../../../pkg/vvm/storage/impl_seqstorage_test.go)
  - update: `TestPutPLogOffset` to pass `appID` to all `GetPLogOffset`/`PutPLogOffset` calls
  - add: Subtest verifying different apps on same partition get independent PLog offsets
