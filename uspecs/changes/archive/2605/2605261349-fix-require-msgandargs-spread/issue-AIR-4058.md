# testingu/require: wrong variadic args usage

- URL: https://untill.atlassian.net/browse/AIR-4058
- ID: AIR-4058
- State: in-progress
- Author: Denis Gribanov
- Assignees: Denis Gribanov
- Labels: none

## Description

### Why

`pkg/goutils/testingu/require` provides a wrapper around `testify/assert` and `testify/require` that adds constraint-based error assertions (`ErrorWith`, `PanicsWith`, `Has`, `Is`, etc.). Two call sites forward a `msgAndArgs []any` value into a variadic `...any` parameter without the `...` spread operator, causing the slice to be boxed as a single `any` argument:

- `pkg/goutils/testingu/require/constraint.go:156` — `assert.Fail(t, "error expected", msgAndArgs)`
- `pkg/goutils/testingu/require/require.go:135` — `errorWith(r.t, err, cc, msgAndArgs)`

`asasalint` flags the first site:

```text
pkg/goutils/testingu/require/constraint.go:156:43:
  pass []any as any to func assert.Fail (asasalint)
```

The defect is reachable only when all of the following are true:

- The caller passed extra `msgAndArgs` past the constraints to `require.Error(err, …)` or `errorWith(t, err, c, …)`
- `err == nil` (the `assert.Fail` branch in `errorWith`)

Because the failing branch is the only consumer of `msgAndArgs`, ordinary callers never notice. When the conditions do meet, testify writes a diagnostic in which:

- format verbs (`%s`, `%d`, …) are not applied — they appear verbatim
- the arguments are printed as a slice literal with surrounding `[ ]` brackets
- when both call sites compose (the `(*Require).Error` → `errorWith` path), the slice is wrapped twice, producing `[[ … ]]`

Observed message text on the buggy code paths:

| Call site reached | Captured `Messages:` content |
|---|---|
| `errorWith` direct | `[ctx=%s id=%d myCase 42]` |
| `(*Require).Error` → `errorWith` | `[[ctx=%s id=%d myCase 42]]` |

Functional outcome is unaffected — the assertion still fails. Only the developer-visible diagnostic is malformed, which makes test failures harder to diagnose and silently invalidates the `msgAndArgs` contract that all `testify/require`-style wrappers promise.

### What

Spread the slice at both forwarding sites:

- `pkg/goutils/testingu/require/constraint.go:156` — change `assert.Fail(t, "error expected", msgAndArgs)` to `assert.Fail(t, "error expected", msgAndArgs...)`
- `pkg/goutils/testingu/require/require.go:135` — change `errorWith(r.t, err, cc, msgAndArgs)` to `errorWith(r.t, err, cc, msgAndArgs...)`

Add a regression test in `pkg/goutils/testingu/require/constraint_msgargs_test.go` that:

- uses a capturing `assert.TestingT` mock (no public `*testing.T` API to read back `Errorf` text)
- exercises both forwarding paths via subtests
- asserts the diagnostic contains the formatted result (`ctx=myCase id=42`) and does not contain unprocessed verbs (`%s`, `%d`)

Verify with `golangci-lint run --default=none --enable asasalint ./pkg/goutils/testingu/require/...` — must report zero issues.

Verify `go test ./pkg/goutils/testingu/require/...` is green.
