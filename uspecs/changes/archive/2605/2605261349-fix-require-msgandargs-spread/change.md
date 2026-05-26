---
change_id: 2605261332-fix-require-msgandargs-spread
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4058
---

# Change request: Spread msgAndArgs in require wrapper forwarding sites

Refs:

- [AIR-4058: testingu/require: wrong variadic args usage](./issue-AIR-4058.md)

## Why

The `pkg/goutils/testingu/require` wrapper forwards `msgAndArgs []any` into variadic `...any` parameters at two sites without the `...` spread, boxing the slice as a single `any`. When an assertion of the form `require.Error(nil, constraint, "ctx=%s", v)` fails, testify prints the raw slice (`[ctx=%s v]` or doubly wrapped `[[ctx=%s v]]`) instead of the formatted message, which makes diagnostics misleading and silently breaks the documented `msgAndArgs` contract.

## What

The failure message produced when `require.Error` / `errorWith` reports a nil error contains the raw `msgAndArgs` slice literal with unprocessed format verbs instead of the formatted text.

```text
caller invokes require.Error(nil, Has(...), "ctx=%s id=%d", v1, v2)
      |
      v
(*Require).Error                 <-- fault: forwards msgAndArgs without ...
      |
      v
errorWith (err == nil branch)    <-- fault: forwards msgAndArgs without ...
      |
      v
assert.Fail receives a single boxed []any
      |
      v
testify prints "Messages: [[ctx=%s id=%d v1 v2]]"   (symptom)
```

Failure messages reported by both forwarding paths render `msgAndArgs` through their format string, so the printed `Messages:` line matches the caller's intent and contains no unprocessed format verbs or slice-literal brackets.

## How

Decisions:

- Drive the change test-first: land a failing regression test that captures the malformed `Messages:` text from both forwarding paths, then apply the fix and watch it turn green
- Spread `msgAndArgs` with `...` at both forwarding sites instead of restructuring the helper signatures, keeping the public API of `(*Require).Error` and `errorWith` unchanged
- Capture failure-message text with a minimal in-package `assert.TestingT` mock (`captureT`), since `*testing.T` exposes no way to read back `Errorf` content; mirrors testify's own self-test pattern
- Exercise both bug sites through subtests of one parent test (`errorWith` direct, `(*Require).Error` through `errorWith`) so the doubly wrapped `[[ … ]]` symptom is visible alongside the singly wrapped `[ … ]` one

Out of scope:

- Auditing `msgAndArgs` forwarding patterns elsewhere in the repository (whole-repo `asasalint` run confirmed these are the only two sites)
- The latent issue that `(*Require).Error` discards `errorWith`'s return value on the constraint branch and never calls `FailNow()` on failure
- Adjacent linter warnings in the same files (`modernize`, `gocritic`, etc.) that are unrelated to variadic forwarding

References:

- [forwarding fault site in errorWith](../../../../../pkg/goutils/testingu/require/constraint.go)
- [forwarding fault site in (\*Require).Error](../../../../../pkg/goutils/testingu/require/require.go)
- [existing constraint tests for the package](../../../../../pkg/goutils/testingu/require/constraint_test.go)
- [asasalint linter rationale](https://github.com/alingse/asasalint)

## Construction

### Tests

- [x] update: [require/constraint_test.go](../../../../../pkg/goutils/testingu/require/constraint_test.go)
  - add: regression test that demonstrates the malformed `Messages:` diagnostic and locks the fixed behavior
  - add: `captureT` minimal `assert.TestingT` mock that records each `Errorf` call (no public `*testing.T` API exposes failure-message text)
  - add: `assertFormattedMessage` helper asserting the captured message contains `ctx=myCase id=42` and does not contain unprocessed verbs (`%s`, `%d`)
  - add: parent test `TestErrorWith_NilErrorMsgAndArgs_FormatsCorrectly` with two subtests:
    - `errorWith direct (single wrap site)` — exercises `errorWith(mockT, nil, []Constraint{}, "ctx=%s id=%d", "myCase", 42)`
    - `(*Require).Error (two wrap sites stacked)` — exercises `New(mockT).Error(nil, Has("never-matches"), "ctx=%s id=%d", "myCase", 42)` inside a recover to swallow the `FailNow` panic
  - add: `strings` import for the contains-check helper

### Implementation

- [x] update: [require/constraint.go](../../../../../pkg/goutils/testingu/require/constraint.go)
  - fix: line 156 — spread `msgAndArgs` when forwarding to `assert.Fail`: `assert.Fail(t, "error expected", msgAndArgs...)`

- [x] update: [require/require.go](../../../../../pkg/goutils/testingu/require/require.go)
  - fix: line 135 — spread `msgAndArgs` when forwarding to `errorWith`: `errorWith(r.t, err, cc, msgAndArgs...)`

### Verification

- [x] verify: `go test ./pkg/goutils/testingu/require/...` passes (both subtests green)
- [x] verify: `golangci-lint run --default=none --enable asasalint ./pkg/goutils/testingu/require/...` reports zero issues
