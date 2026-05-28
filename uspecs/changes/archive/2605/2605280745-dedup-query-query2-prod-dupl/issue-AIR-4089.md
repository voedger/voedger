# voedger: Deduplicate query / query2 production code (dupl linter cleanup)

- URL: https://untill.atlassian.net/browse/AIR-4089
- ID: AIR-4089
- State: in-progress
- Author: d.gribanov@dev.untill.com
- Labels: none

## Why

`dupl` flags three independent copy-paste clones in the production code of `pkg/processors/query` and `pkg/processors/query2`. Each clone duplicates a small but non-trivial piece of behaviour, meaning a change to one branch must be mirrored in the other or the two will silently drift apart.

Site 1 — `Greater` vs `Less` filter `IsMatch`

Two 30-line `switch` statements over `appdef.DataKind_*` that differ only by the comparison operator (`>` vs `<`) and the error-message filter-kind constant (`filterKind_Gt` vs `filterKind_Lt`). Each new supported `DataKind` must be added in both files in lock-step.

Site 2 — `newAndFilter` vs `newOrFilter` in `factory-filter-impl.go`

Two 15-line factories with identical operand-validation loops; they differ only in the struct constructed (`AndFilter` vs `OrFilter`) and the kind constant used in the error message.

Site 3 — Command vs Query `Execute` schema generation in `impl_openapi.go`

Two 14-line branches inside `generateComponents` that handle `TypeKind_Command + Execute` and `TypeKind_Query + Execute` respectively. They differ only in the `TypeKind`, the concrete interface used for the type assertion (`appdef.ICommand` vs `appdef.IQuery`) and the local variable name (`cmd` vs `qry`). Both interfaces embed `appdef.IFunction`, which already exposes `Param()` and `Result()`.

Consequences

- 6 `dupl` warnings across the two packages (2 at default threshold, 4 more at threshold 75)
- High drift risk on any change to a comparison filter, logical-filter factory, or function-schema rule
- One latent edge case worth verifying during the refactor: in site 3 the existing `continue` skips the whole `for op, fields := range ops` iteration when `param.QName() == QNameANY`, so the matching `result` is never processed for that op. The new helper must preserve this behaviour

## What

Refactor each clone site so that the shared behaviour lives in one place. No behaviour change is intended; existing unit and integration tests must continue to pass without modification.

1. Unify `GreaterFilter` / `LessFilter` `IsMatch`
   - Extract one `compareOrdered[T constraints.Ordered](a, b T, op orderingOp) bool` (or an equivalent unexported helper) and route both `GreaterFilter.IsMatch` and `LessFilter.IsMatch` through it
   - Alternative: keep the two structs but back them with one unexported `orderingFilter` type carrying a `kind` discriminator (`Gt` / `Lt`) and a single `IsMatch` method; expose `GreaterFilter` and `LessFilter` as thin constructors / type aliases to preserve the public API and existing test fixtures
   - Acceptance: `TestGreaterFilter_IsMatch` and `TestLessFilter_IsMatch` pass unchanged; the clone between the two files no longer appears in `dupl`
2. Unify `newAndFilter` / `newOrFilter`
   - Extract `buildBinaryFilter(operands []interface{}, kind string, ctor func([]IFilter) IFilter) (IFilter, error)` and have both factories call it with their respective `kind` constant and constructor
   - Acceptance: `TestAndFilter_IsMatch` and `TestOrFilter_IsMatch` pass unchanged; the clone within `factory-filter-impl.go` no longer appears in `dupl`
3. Unify the Command/Query `Execute` schema branch in `impl_openapi.go`
   - Replace the two `if t.Kind() == … && op == Execute { … }` blocks with a single dispatch through `appdef.IFunction`:

     ```go
     if op == appdef.OperationKind_Execute {
         if fn, ok := t.(appdef.IFunction); ok {
             g.generateFunctionExecuteSchemas(fn, op, schemas)
         }
     }
     ```

   - Add `generateFunctionExecuteSchemas(fn appdef.IFunction, op appdef.OperationKind, schemas map[string]interface{})` that handles `fn.Param()` and `fn.Result()` and returns early on `QNameANY` to preserve current semantics
   - Acceptance: the OpenAPI integration tests in `pkg/sys/it` (e.g. `TestOpenAPI`, `TestQpv2*`) pass unchanged; the two clones inside `impl_openapi.go` no longer appear in `dupl`

Verification

- `go test ./pkg/processors/query/... ./pkg/processors/query2/...` passes
- Affected integration tests under `pkg/sys/it` pass
- `golangci-lint run --no-config --default=none --enable=dupl --tests=false ./pkg/processors/query/... ./pkg/processors/query2/...` reports 0 findings at the default threshold and ≤0 of the three sites above at threshold 75
- `golangci-lint run ./pkg/processors/query/... ./pkg/processors/query2/...` reports no new findings in any other linter
