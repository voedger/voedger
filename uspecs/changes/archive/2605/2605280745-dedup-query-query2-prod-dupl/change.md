---
change_id: 2605280725-dedup-query-query2-prod-dupl
type: refactor
issue_url: https://untill.atlassian.net/browse/AIR-4089
scope: apps
---

# Change request: Deduplicate query and query2 production code flagged by dupl

Refs:

- [AIR-4089: voedger: Deduplicate query / query2 production code (dupl linter cleanup)](./issue-AIR-4089.md)

## Why

The `dupl` linter flags three independent copy-paste clones in the production code of the query and query2 processor packages. Each clone duplicates a small but non-trivial behaviour fragment, so any future change must be mirrored across the duplicated sites or the branches will silently drift apart.

## What

Collapse the three clone sites in the query and query2 processor packages into single shared implementations:

- Unify the `Greater` / `Less` comparison filter behaviour into one ordered-comparison helper while preserving the existing public filter types
- Unify the `And` / `Or` logical-filter factories through a shared binary-filter builder
- Unify the Command and Query Execute OpenAPI schema branches via a single dispatch through the common function interface, preserving the existing skip-on-any-qname semantics
- No behaviour change: the externally observable behaviour to preserve is the exact result of `IsMatch` for every supported `DataKind`, the error messages produced by the filter factories, and the OpenAPI document generated for command and query Execute endpoints

## How

Decisions:

- Site 1: keep `GreaterFilter` and `LessFilter` as the public types; both delegate `IsMatch` to one shared unexported routine carrying a comparison discriminator (`gt` / `lt`) so the `switch` over `appdef.DataKind_*` lives in one place and adding a new kind happens once
- Site 2: extract `buildBinaryFilter` in `factory-filter-impl.go` that takes the operands, the filter kind constant, and a constructor closure; `newAndFilter` and `newOrFilter` become thin wrappers around it
- Site 3: dispatch the Execute branch in `impl_openapi.go` through `appdef.IFunction` (the common ancestor of `ICommand` and `IQuery`) and a new `generateFunctionExecuteSchemas` helper. The helper preserves the existing semantics that, when `Param().QName() == QNameANY`, both the param-schema component generation and the matching `Result()` processing are skipped for that op (originally achieved via `continue` on the outer `for op, fields := range ops` loop)
- Do not change the public API of `GreaterFilter`, `LessFilter`, `AndFilter`, `OrFilter`, `newAndFilter`, `newOrFilter`, `CreateOpenAPISchema`, or `schemaGenerator`

Out of scope:

- Test-file duplication (addressed separately under the `pkg/appdef` / `pkg/processors/query` test-dedup work)
- Sub-threshold (<75 token) handler boilerplate clones in `query2/impl.go`, `impl_cdocs_handler.go`, `impl_query_handler.go`, `impl_view_handler.go`, `interface.go`, `util.go`
- The `query2/types.go` `case DataKind_int32` / `case DataKind_string` filter-loader pair

References:

- [GreaterFilter implementation](../../../../../pkg/processors/query/filter-greater-impl.go)
- [LessFilter implementation](../../../../../pkg/processors/query/filter-less-impl.go)
- [And/Or filter factories](../../../../../pkg/processors/query/factory-filter-impl.go)
- [OpenAPI schema generator](../../../../../pkg/processors/query2/impl_openapi.go)
- [appdef.IFunction interface](../../../../../pkg/appdef/interface_function.go)
- [Greater filter tests](../../../../../pkg/processors/query/filter-greater-impl_test.go)
- [Less filter tests](../../../../../pkg/processors/query/filter-less-impl_test.go)
- [And filter tests](../../../../../pkg/processors/query/filter-and-impl_test.go)
- [Or filter tests](../../../../../pkg/processors/query/filter-or-impl_test.go)
- [OpenAPI integration tests](../../../../../pkg/sys/it/impl_qpv2_test.go)

## Construction

- [x] update: [query/filter-greater-impl.go](../../../../../pkg/processors/query/filter-greater-impl.go)
  - rewrite `GreaterFilter.IsMatch` to delegate to a shared unexported ordered-comparison routine carrying a `gt` discriminator
  - preserve the public `GreaterFilter` struct and its field shape

- [x] update: [query/filter-less-impl.go](../../../../../pkg/processors/query/filter-less-impl.go)
  - rewrite `LessFilter.IsMatch` to delegate to the same shared routine with an `lt` discriminator
  - preserve the public `LessFilter` struct and its field shape

- [x] create: [query/filter-ordering-impl.go](../../../../../pkg/processors/query/filter-ordering-impl.go)
  - shared single `switch` over `appdef.DataKind_*` that compares row value against filter value for both `gt` and `lt`
  - returns `false, nil` for `DataKind_null` and `false, ErrWrongType`-wrapped error for unsupported kinds, parameterized by the filter-kind constant

- [x] update: [query/factory-filter-impl.go](../../../../../pkg/processors/query/factory-filter-impl.go)
  - extract `buildBinaryFilter(operands []interface{}, kind string, ctor func([]IFilter) IFilter) (IFilter, error)`
  - rewrite `newAndFilter` and `newOrFilter` as thin wrappers around `buildBinaryFilter`

- [x] update: [query2/impl_openapi.go](../../../../../pkg/processors/query2/impl_openapi.go)
  - replace the two `if t.Kind() == TypeKind_Command/_Query && op == Execute { ... }` branches in `generateComponents` with a single dispatch through `appdef.IFunction`
  - add `generateFunctionExecuteSchemas(fn appdef.IFunction, op appdef.OperationKind, schemas map[string]interface{})` that handles `Param()` then `Result()`; preserves the original semantics that, when `Param().QName() == QNameANY`, both the param-schema generation and the subsequent `Result()` processing are skipped for that op

- [x] verify: `go test ./pkg/processors/query/... ./pkg/processors/query2/...` passes

- [x] verify: affected integration tests under `pkg/sys/it` pass (`TestOpenAPI`, `TestQueryProcessor2_*`)

- [x] verify: `golangci-lint run --no-config --default=none --enable=dupl --tests=false ./pkg/processors/query/... ./pkg/processors/query2/...` reports 0 findings at the default threshold, and the three sites above no longer appear at threshold 75

- [x] verify: `golangci-lint run ./pkg/processors/query/... ./pkg/processors/query2/...` reports no new findings in any other linter
