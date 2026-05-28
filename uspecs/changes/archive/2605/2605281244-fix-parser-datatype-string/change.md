---
change_id: 2605281228-fix-parser-datatype-string
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4109
scope: apps
---

# Change request: Correct parser data type string names

Refs:

- [AIR-4109: voedger: fix wrong cases in parser/DataType.String()](./issue-AIR-4109.md)

## Why

The parser data type string conversion reports `Float32` and `Float64` as integer type names. This produces misleading string representations for floating-point data types and can propagate wrong type names to parser consumers.

## What

`DataType.String()` returns integer type names for floating-point parser data types.

```text
VSQL schema contains a float32 or float64 data type
      |
      v
parser builds a DataType value
      |
      v
DataType.String() maps Float32/Float64 cases   <-- fault: returns int32/int64
      |
      v
consumer receives the wrong data type name      (symptom)
```

`DataType.String()` returns `float32` for `Float32` and `float64` for `Float64`, while preserving the existing names for all other data types.

## How

Decisions:

- Correct only the `Float32` and `Float64` branches in `DataType.String()`; keep all parser grammar and data-kind analysis behavior unchanged
- Cover the string conversion with focused parser tests so future integer/float mapping regressions are caught directly

Out of scope:

- Changing VSQL parsing aliases such as `real`, `float`, or `double precision`
- Changing appdef data kind mapping or storage representation for floating-point fields

References:

- [parser data type string conversion](../../../../../pkg/parser/types.go)
- [parser data type tests](../../../../../pkg/parser/impl_test.go)

## Construction

- [x] update: [parser data type tests](../../../../../pkg/parser/impl_test.go)
  - add focused assertions that cover every `DataType.String()` return branch, including `float32` for `Float32` and `float64` for `Float64`
  - preserve existing data-kind parsing assertions for VSQL aliases such as `real`, `float`, and `double precision`

- [x] update: [parser data type string conversion](../../../../../pkg/parser/types.go)
  - change only the `Float32` and `Float64` cases in `DataType.String()` to return `float32` and `float64`
  - keep the current return values for all other data type cases
  - keep gofmt-applied struct tag alignment changes in the same file behavior-neutral

- [x] verify: `go test ./pkg/parser -run Test_DataTypes -count=1` passes

- [x] verify: `go test ./pkg/parser -count=1` passes
