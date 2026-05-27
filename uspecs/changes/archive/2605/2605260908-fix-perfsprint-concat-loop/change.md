---
registered_at: 2026-05-26T08:51:37Z
change_id: 2605260851-fix-perfsprint-concat-loop
type: refactor
baseline: 3f3122a8e3a8dd3b7b6428d0983134b608d69693
issue_url: https://untill.atlassian.net/browse/AIR-4052
archived_at: 2026-05-26T09:08:07Z
---

# Change request: Fix perfsprint concat-loop warnings

## Why

The `perfsprint` linter flagged 9 occurrences of `concat-loop` across the codebase — string concatenation inside `for` loops using `+=`. This pattern runs in O(n²) (every `+=` allocates a fresh string and copies the previous buffer), repeats `if i > 0 { s += sep }` separator boilerplate at every site, and blocks a clean lint baseline.

## What

Replace each in-loop string concatenation with the idiomatic O(n) alternative, without changing observable output:

- "Join values with a separator" patterns: build `[]string` and use `strings.Join`
- Multi-section text/HTML accumulators: use `strings.Builder` together with `fmt.Fprintf` / `fmt.Fprint` / `WriteString`

## Construction

- [x] update: [appdef/filter/impl_and.go](../../../../../pkg/appdef/filter/impl_and.go)
  - replace: `s += " AND " + c.String()` loop in `String()` with `strings.Join` over a pre-sized `[]string`

- [x] update: [appdef/filter/impl_or.go](../../../../../pkg/appdef/filter/impl_or.go)
  - replace: `s += " OR " + c.String()` loop in `String()` with `strings.Join` over a pre-sized `[]string`

- [x] update: [appdef/filter/impl_qnames.go](../../../../../pkg/appdef/filter/impl_qnames.go)
  - replace: `s += ", "` loop in `String()` with `"QNAMES(" + strings.Join(parts, ", ") + ")"`

- [x] update: [appdef/filter/impl_tags.go](../../../../../pkg/appdef/filter/impl_tags.go)
  - replace: `s += ", "` loop in `String()` with `"TAGS(" + strings.Join(parts, ", ") + ")"`

- [x] update: [appdef/filter/impl_types.go](../../../../../pkg/appdef/filter/impl_types.go)
  - replace: `s += ", "` loop in `String()` with `"TYPES(" + strings.Join(parts, ", ") + ")"`

- [x] update: [goutils/logger/impl.go](../../../../../pkg/goutils/logger/impl.go)
  - replace: `s += fmt.Sprint(" ", arg)` loop in `getFormattedMsg` with `strings.Builder` + `fmt.Fprint`

- [x] update: [istructsmem/errors.go](../../../../../pkg/istructsmem/errors.go)
  - replace: `enrich += " " + fmt.Sprint(args[i])` loop in `enrichError` with `strings.Builder` + `fmt.Fprint`

- [x] update: [query2/impl_schemas_handler.go](../../../../../pkg/processors/query2/impl_schemas_handler.go)
  - replace: two `generatedHTML += fmt.Sprintf(...)` loops with `strings.Builder` + `fmt.Fprintf` / `WriteString`

- [x] update: [query2/impl_schemas_roles_handler.go](../../../../../pkg/processors/query2/impl_schemas_roles_handler.go)
  - replace: two `generatedHTML += fmt.Sprintf(...)` loops with `strings.Builder` + `fmt.Fprintf` / `WriteString`

- [x] verify: `golangci-lint run --default=none --enable perfsprint --max-same-issues 0 --max-issues-per-linter 0 ./...` reports no `concat-loop` issues
- [x] verify: `go test ./pkg/appdef/filter/... ./pkg/processors/query2/... ./pkg/goutils/logger/... ./pkg/istructsmem/...` is green
