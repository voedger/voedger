# Implementation plan: Fix bbolt TTL key construction and background cleaner test

## Construction

- [x] update: [pkg/istorage/bbolt/impl.go](../../../../pkg/istorage/bbolt/impl.go)
  - fix: `makeTTLKey` — allocate slice with length 0 (`make([]byte, 0, totalLength)`) so `AppendUint64` writes at the start instead of after pre-allocated zero bytes
- [x] update: [pkg/istorage/bbolt/impl_test.go](../../../../pkg/istorage/bbolt/impl_test.go)
  - fix: `TestBackgroundCleaner` — verify expired key is physically deleted from the bbolt data bucket using a raw `db.View` transaction instead of relying on `TTLGet`

