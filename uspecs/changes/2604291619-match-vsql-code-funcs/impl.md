# Implementation plan: Match vsql and code funcs definitions

## Construction

### Validator

- [x] update: [pkg/iextengine/builtin/impl.go](../../../pkg/iextengine/builtin/impl.go)
  - add: `AppFuncs()` and `StatelessFuncs()` accessors on the BuiltIn factory returning the existing per-app and stateless `BuiltInExtFuncs` maps for deployment-time validation
- [x] Review
- [x] update: [pkg/appparts/impl_app.go](../../../pkg/appparts/impl_app.go)
  - add: package-local structural interface `builtInFuncsRegistry { AppFuncs(); StatelessFuncs() }` so the BuiltIn factory is matched via Go duck typing without exposing a new public type in `pkg/iextengine`
  - add: `validateExtensions(def, eef, extModuleURLs)` function that, before pools are constructed in `appRT.deploy`:
    - iterates `appdef.Extensions(def.Types())` once and switches on `ext.Engine()`:
      - `BuiltIn`: requires the BuiltIn factory accessor (via the package-local interface) to report a matching `FullQName` for `a.name`; collects mismatches as "in vsql, not in code"
      - `WASM`: keeps current behaviour - names are accumulated into `ExtensionModule.ExtensionNames` and validated by `wazero.initModule`; surfaces that error as a typed deployment error rather than a panic
    - inverse pass: every BuiltIn entry whose `FullQName` belongs to `a.name` and was not visited during the AppDef walk is collected as "in code, not in vsql"
    - aggregates all mismatches via `errors.Join` and returns a single composite error listing each offending `FullQName`, kind (projector / command / query / job) and direction
  - update: `appRT.deploy` calls the validator first; on error, panics with the composite error (consistent with the existing `panic(err)` style in `DeployApp`)

### Cleanup of subsumed checks

- [x] update: [pkg/istructsmem/appstruct-types.go](../../../pkg/istructsmem/appstruct-types.go)
  - remove: `validateResources()` and `validateJobs()` together with their two call sites in `AppConfigType.prepare()`
- [x] update: [pkg/processors/actualizers/provide.go](../../../pkg/processors/actualizers/provide.go)
  - note: the unguarded `appdef.Projector(appStructs.AppDef().Type, projector.Name).Sync()` becomes unreachable for missing entries because deployment now fails first; no code change required, but verify the call site stays correct after the validator lands

### Tests

- [x] update: [pkg/appparts/impl_test.go](../../../pkg/appparts/impl_test.go)
  - add: subtest `TestDeployApp_ValidateExtensions_MatchVSQLAndCode`:
    - `vsql declares a builtin projector / command / query with no code implementation` -> `DeployApp` panics with composite error containing the offending FullQName and direction `in vsql, not in code`
    - `code registers a stateless projector / command / query absent from AppDef` -> same composite error, direction `in code, not in vsql`
    - `aligned set` -> `DeployApp` succeeds
    - `wasm-engine extension declared in vsql with name not exported by the wasm module` -> `DeployApp` surfaces a typed deployment error wrapping the wazero engine error

- [x] update: [pkg/sys/it/impl_bootstrap_test.go](../../../pkg/sys/it/impl_bootstrap_test.go)
  - add: VIT-driven `TestVVMLaunch_VSQLCodeMismatch` integration test using `it.NewOwnVITConfig` + `it.WithApp` with a mismatched `test1/app1` builder
    - subtest `in vsql, not in code`: vsql declares an extra builtin command absent from `cfg.Resources` -> `it.NewVIT` panics with `appparts.ErrDeployment` and message containing `in vsql, not in code`
    - subtest `in code, not in vsql`: `cfg.Resources` registers an extra command absent from vsql -> `it.NewVIT` panics with `appparts.ErrDeployment` and message containing `in code, not in vsql`

### Fix existing drift exposed by the new validator

- [x] update: [pkg/vit/shared_cfgs.go](../../../pkg/vit/shared_cfgs.go)
  - register `app1pkg.QryAny` and `app1pkg.CmdAny` via `istructsmem.NullQueryExec` / `NullCommandExec` in `ProvideApp1` to satisfy the validator (they are declared in `pkg/vit/schemaTestApp1.vsql` but had no code stub)
- [x] update: [pkg/parser/impl_test.go](../../../pkg/parser/impl_test.go)
  - `TestIsOperationAllowedOnNestedTable`: register null execs for `sys.CreateLogin`, `sys.UpdateSubscription`, `sys.UPTerminalWebhook` declared in `pkg/parser/sql_example_syspkg/system.vsql`
  - `TestIsOperationAllowedOnGrantRoleToRole`: register null execs for the test's own `pkg.Cmd1` plus the same three sys extensions

- [x] Review
