---
registered_at: 2026-05-22T13:31:43Z
change_id: 2605221331-respect-usetesting-linter
type: test
baseline: 4344f6ebd11c8a3cdf7509027b56717ce5982fed
issue_url: https://untill.atlassian.net/browse/AIR-4017
archived_at: 2026-05-22T13:44:28Z
---

# Change request: respect `usetesting` linter across the project

## Why

The `usetesting` linter (enabled via `default: all` in `.golangci.yml`) flags `os` package calls inside `*testing.T` functions that have safer testing equivalents introduced in Go 1.17+ and Go 1.24:

- `os.Setenv` / `os.Unsetenv` — manual `defer os.Unsetenv(...)` is error-prone, leaks state on early `t.Fatal`, and is not parallel-safe across subtests; `t.Setenv` auto-restores the prior value and fails fast when `t.Parallel()` is set
- `os.MkdirTemp` / `os.CreateTemp` — manual `defer os.RemoveAll(dir)` is verbose, repeated at every call site, and runs even on `require.FailNow`; `t.TempDir()` ties cleanup to the test lifetime with proper Windows retry semantics and a unique per-test name
- `os.Chdir` (Go 1.24+) — manual `defer os.Chdir(initialWD)` requires capturing the working directory before every call and unwinding in reverse order; `t.Chdir` does this automatically and is goroutine-safe with respect to the test framework

Aligning with `usetesting` keeps the test suite consistent across all five modules registered in `go.work`, removes boilerplate (manual `Getwd` / `Unsetenv` / `RemoveAll` defers), and prevents future regressions because the linter runs in CI.

## What

Replace flagged `os.*` calls in test functions with their `*testing.T` equivalents and drop the now-redundant cleanup defers and error checks:

- `os.Setenv` → `t.Setenv`
- `os.MkdirTemp` → `t.TempDir`
- `os.Chdir` → `t.Chdir`

Retain `os.MkdirTemp` in `cmd/vpm/main_test.go` for the verbose-mode branches that deliberately preserve the temp directory for post-test inspection; annotate each retained site with `//nolint:usetesting` and an inline rationale.

Verify by running `golangci-lint run --enable-only=usetesting ./...` in each `go.work` module and confirming `0 issues`.

## Construction

### Rules

- [x] update: [ar-golang.md](../../../../../.augment/rules/ar-golang.md)
  - add: a new bullet under the existing unit-test rules describing the `usetesting` preferences
  - cover: `t.Setenv` over `os.Setenv` + `defer os.Unsetenv` (with note that `t.Setenv` already fails the test on error, so the surrounding `require.NoError` is to be dropped)
  - cover: `t.TempDir()` over `os.MkdirTemp` + `defer os.RemoveAll`
  - cover: `t.Chdir(dir)` over `initialWD, _ := os.Getwd()` + `os.Chdir(dir)` + `defer os.Chdir(initialWD)`
  - cover: when a helper called from a test needs one of the above, refactor it to accept `*testing.T` (with `t.Helper()`) rather than `*require.Assertions`
  - cover: escape hatch — annotate intentional retentions with `//nolint:usetesting` and an inline rationale (e.g. preserving a temp dir for post-mortem inspection in a verbose-mode branch)

### Tests

- [x] update: [impl_test.go](../../../../../pkg/isecretsimpl/impl_test.go)
  - replace: `os.MkdirTemp` → `t.TempDir()`
  - replace: `os.Setenv` → `t.Setenv`
  - drop: `defer` cleanup block (`os.RemoveAll` + `os.Unsetenv`)

- [x] update: [exec_test.go](../../../../../pkg/goutils/exec/exec_test.go)
  - replace: `os.Setenv("MYVAR", ...)` → `t.Setenv("MYVAR", ...)` in `Test_PassEnvironmentVariable`

- [x] update: [impl_test.go](../../../../../pkg/goutils/filesu/impl_test.go)
  - replace in `TestCopy_BasicUsage/"CopyFile src file without path"`: `os.Getwd` capture + `os.Chdir` + `defer os.Chdir` → `t.Chdir`

- [x] update: [impl_test.go](../../../../../pkg/ihttpimpl/impl_test.go)
  - refactor: `makeTmpContent(require *require.Assertions, ...)` → `makeTmpContent(t *testing.T, ...)` using `t.TempDir()` and `t.Helper()`
  - drop: `defer os.RemoveAll(dir)` at the call site in `TestBasicUsage_HTTPProcessor`

- [x] update: [main_test.go](../../../../../cmd/vpm/main_test.go)
  - replace in `TestPkgRegistryCompile`: `os.Getwd` capture + `os.Chdir` + `defer os.Chdir` → `t.Chdir`
  - annotate the 5 verbose-mode `os.MkdirTemp` sites (`TestCompileBasicUsage_Errors`, `TestOrmBasicUsage`, `TestTidyBasicUsage`, `TestBuildBasicUsage`, `TestGenOrmTestItAndBuildApp`) with `//nolint:usetesting` plus an inline rationale (verbose mode keeps the dir for post-test inspection; `t.TempDir()` auto-removes)

- [x] update: [main_test.go](../../../../../cmd/ctool/main_test.go)
  - replace 3 `os.Setenv` sites (`TestEnvSshKey`, `TestVariableEnvironment` ×2) → `t.Setenv`; drop the paired `require.NoError(err)` checks

### Verification

- [x] run: `golangci-lint run --enable-only=usetesting ./...` in each `go.work` module (`.`, `./cmd/ctool`, `./cmd/edger`, `./pkg/iextengine/wazero/_testdata`, `./pkg/sys/it/testdata/apps/test2.app1/src`) — each must report `0 issues`
- [x] run: `go build ./...` in root, `./cmd/ctool`, `./cmd/edger` — must succeed
- [x] run: tests of affected packages (`pkg/isecretsimpl`, `pkg/goutils/exec`, `pkg/goutils/filesu`, `pkg/ihttpimpl`, `cmd/vpm`) — must pass
