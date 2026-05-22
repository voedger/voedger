---
type: "always_apply"
---

if you you are writting a program on golang then follow these rules:

- Use go 1.24 and above features
- Use `for <idx> = range(<slice>)` whenever appropriate (do not use for <idx> := 0; idx <... ; idx ++)
- in unit tests use require := require.New(t) and then use e.g. require.Equal(1, 1) instead of require.Equal(t, 1, 1)
- in unit tests avoid constructions like if a != b { t.Fatal() }. Use require.Equal(a,b) instead
- in unit tests if you are modifying any global state then save the initial state, then modify, then revert changes using defer func
- in unit tests if the test consists of few parts that do not impact on others then implement subtests with `t.Run(description, func(t *testing.T){})` instead of setting description as a comment
- in unit tests prefer `*testing.T` helpers over their `os` equivalents (respect the `usetesting` linter):
  - use `t.Setenv(key, value)` instead of `os.Setenv` + `defer os.Unsetenv`; drop the surrounding `require.NoError` since `t.Setenv` fails the test on error
  - use `t.TempDir()` instead of `os.MkdirTemp` + `defer os.RemoveAll`
  - use `t.Chdir(dir)` instead of capturing `initialWD, _ := os.Getwd()` + `os.Chdir(dir)` + `defer os.Chdir(initialWD)`
  - when a helper called from a test needs one of the above, refactor it to accept `*testing.T` (and call `t.Helper()`) rather than `*require.Assertions`
  - if the call must stay (e.g. intentionally preserving a temp dir for post-mortem inspection in a verbose-mode branch), annotate it with `//nolint:usetesting` and an inline rationale
- write as short a code as possible
- avoid comments. Produce the code so that it is clear without comments what it does
- never assign a Go error to `_` (no `res, _ := json.Marshal(x)`). Always bind it: `res, err := json.Marshal(x); if err != nil { ... }`. For calls infallible by construction (e.g. `json.Marshal` over a fully-controlled struct, `bytes.Buffer.Write`), use `// notest` on the error branch and either `panic(err)` or `return err`. In all other cases, use `return err` to propagate.
