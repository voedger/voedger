# Implementation plan: Singleton IDs corrupt RecordID sequence after recovery

## Technical design

- [x] update: [sequences--arch.md](../../../../specs/prod/apps/sequences--arch.md)
  - update: `Next()` flow documentation to reflect `initialValue` floor enforcement in `incrementNumber()`
  - update: `Next()` comment block — replace "If number is 0 then initial value is used" with `max(number+1, initialValue)` logic
  - update: Mermaid sequence diagram for sequencing transaction to show `initialValue` enforcement

## Construction

- [x] update: [pkg/isequencer/impl_test.go](../../../../../../pkg/isequencer/impl_test.go)
  - add: test that demonstrates the problem with singleton IDs
- [x] Review
- [x] update: [pkg/isequencer/impl.go](../../../../../../pkg/isequencer/impl.go)
  - update: `incrementNumber()` — add `initialValue` parameter, return `max(number+1, initialValue)` instead of `number+1`
  - update: all 4 call sites in `Next()` — pass `initialValue` to `incrementNumber()`
  - remove: `if nextNumber == 0 { nextNumber = initialValue - 1 }` special case in the storage path
  - update: `Next()` comment block to reflect new logic
