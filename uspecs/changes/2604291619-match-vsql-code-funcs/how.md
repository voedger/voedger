# How: Match vsql and code funcs definitions

## Approach

### Why not `AppConfigType.prepare()`

The pre-existing `validateResources()` / `validateJobs()` in
`pkg/istructsmem/appstruct-types.go` look like the right place at first glance, but
they are structurally incomplete and cannot be the unified validator:

- They only see per-app Go-registered things (`cfg.Resources.Resources`,
  `cfg.syncProjectors`, `cfg.asyncProjectors`, `cfg.jobs`). Stateless funcs
  (`IStatelessResources`) are invisible here, which is exactly the gap that produced
  the fiscalcloud panic
- `AppConfigType` has no view of wasm modules. Sidecar apps carry their command /
  query / projector implementations in wasm (`extModuleURLs`), and that information
  lands in `appparts.DeployApp` only - never inside `prepare()`. Adding stateless
  checks here still leaves wasm-side drift unvalidated
- `validateResources()` walks code -> AppDef. It does not catch the inverse drift
  (vsql declares an extension that has no implementation), which is the more dangerous
  case at deployment time

Conclusion: the existing `validateResources()` / `validateJobs()` should be removed
from `AppConfigType.prepare()` and the validation relocated to the deployment stage,
where every kind of implementation surface is reachable.

### Why `appparts.appRT.deploy` is the right place

`pkg/appparts/impl_app.go::appRT.deploy` is the single choke point that already:

- Runs uniformly for built-in apps (via `appStructsProviderType.BuiltIn`) and sidecar
  apps (via `appStructsProviderType.New`) - both reach `appparts.DeployApp` ->
  `appRT.deploy`
- Walks every extension declared in the AppDef via `appdef.Extensions(def.Types())`,
  which is the AppDef -> code direction we need
- Has access to `a.apps.extEngineFactories`, the unified entry point that already
  groups all known implementations:
  - `pkg/vvm/engines/provide.go::ProvideExtEngineFactories` builds the BuiltIn engine
    factory from `provideAppsBuiltInExtFuncs(cfg.AppConfigs)` (per-app Go funcs)
    merged with `provideStatelessFuncs(cfg.StatelessResources)` (process-wide stateless
    funcs). Both surface as a single `BuiltInExtFuncs` map keyed by `FullQName`
  - The WASM engine factory receives `iextengine.ExtensionModule{Path, ModuleURL,
    ExtensionNames}` and validates them inside `wazero.initModule(...)` against the
    actual wasm exports - that side is already covered

So everything the validator needs is reachable from `deploy`, without nil-guards
against `IAppDef.Type(...)` lookups.

### Validator shape

Add a single function (e.g. `validateExtensions(def, eef, extModuleURLs)`) called from
`appRT.deploy` before pools are constructed. It iterates `appdef.Extensions(def.Types())`
once and switches on `ext.Engine()`:

- `ExtensionEngineKind_BuiltIn`: require that `eef[BuiltIn]` exposes an implementation
  for `def.FullQName(ext.QName())`. The BuiltIn factory already has the merged
  `BuiltInExtFuncs` map (per-app + stateless); expose it via a small read-only
  accessor on the BuiltIn factory (or pass the merged map directly into `deploy` next
  to `eef`). Lookup is a plain map check, no nil-guard on AppDef
- `ExtensionEngineKind_WASM`: keep the current behaviour - the existing wasm path in
  `deploy` already collects names into `ExtensionModule.ExtensionNames`, and
  `wazero.initModule` errors out when an expected name is not exported by the module.
  No additional code needed beyond surfacing that error as a typed deployment error
  rather than a panic

Jobs (`IJob`) are covered by the same pass without any special branch:

- `IJob` inherits `IExtension`, so `appdef.Extensions(def.Types())` already yields
  jobs alongside commands / queries / projectors. The parser
  (`pkg/parser/impl_build.go::jobs`) assigns `ExtensionEngineKind_BuiltIn` or
  `ExtensionEngineKind_WASM` to each job, identical to other extensions
- The merged `BuiltInExtFuncs` map already includes per-app jobs via
  `pkg/vvm/engines/impl.go::writeJobs` (keyed by the same `FullQName`), so the
  `BuiltIn` branch lookup catches missing job implementations with no extra code
- There is no stateless-jobs surface (`IStatelessResources` exposes only commands,
  queries and projectors; jobs are always added per-app via `cfg.AddJobs(...)`), so
  jobs are strictly a subset of what the unified validator already covers
- The existing `AppConfigType.validateJobs()` is fully subsumed by the deployment
  walk and is removed together with `validateResources()`

Inverse direction (code with no AppDef entry) is handled by the same pass plus a
final consistency assertion: every entry in the merged `BuiltInExtFuncs` map whose
package path belongs to `def` must have been visited during the AppDef walk. This
catches "registered in code, missing from vsql" - the original fiscalcloud failure
mode - directly at the deployment stage.

All mismatches are aggregated with `errors.Join` and returned as a single deployment
error listing each offending `FullQName`, kind (projector / command / query) and
direction (in code but not in vsql / in vsql but not in code).

### Failure mode

`appparts.DeployApp` currently panics on internal errors. The new validator returns
its composite error to `DeployApp`, which surfaces it the same way - but with an
actionable message instead of a `nil pointer dereference` deeper in the actualizer
factory. The unguarded
`pkg/processors/actualizers/provide.go::NewSyncActualizerFactoryFactory` loop becomes
unreachable for missing entries because `deploy` would have already failed.

### Cleanup

- Delete `validateResources()` and `validateJobs()` from `AppConfigType.prepare()` and
  any wiring that exists only to support them - the deployment-stage validator
  subsumes both
- Leave `provideAppsBuiltInExtFuncs` / `provideStatelessFuncs` panic-on-missing
  behaviour as is for now; once `deploy` validates first, those panics are unreachable
  and can be turned into asserts in a follow-up

### Tests

- Unit test in `pkg/appparts` covering `appRT.deploy`:
  - AppDef declares a `BuiltIn`-engine projector / command / query that has no entry
    in the merged `BuiltInExtFuncs` -> `DeployApp` returns the composite error
  - Inverse: `BuiltInExtFuncs` carries an entry whose `FullQName` is not in the AppDef
    -> same composite error
  - Aligned set passes
- WASM-engine drift is already covered by `wazero` engine tests; add one
  deployment-level test confirming the deploy-time error wraps the engine error
- Integration: re-run `Test_FiscalCloud_Vit` with the pre-`sys.ApplyInviteEvents`
  baseline pinned in `airs-bp3` and assert it now fails with the new typed deployment
  error, not with a `nil` pointer deref

## References

- [pkg/appparts/impl_app.go](../../../pkg/appparts/impl_app.go)
- [pkg/appparts/impl.go](../../../pkg/appparts/impl.go)
- [pkg/vvm/engines/provide.go](../../../pkg/vvm/engines/provide.go)
- [pkg/vvm/engines/impl.go](../../../pkg/vvm/engines/impl.go)
- [pkg/iextengine/interface.go](../../../pkg/iextengine/interface.go)
- [pkg/iextengine/wazero/impl.go](../../../pkg/iextengine/wazero/impl.go)
- [pkg/parser/impl_build.go](../../../pkg/parser/impl_build.go)
- [pkg/istructsmem/appstruct-types.go](../../../pkg/istructsmem/appstruct-types.go)
- [pkg/istructsmem/resources-types.go](../../../pkg/istructsmem/resources-types.go)
- [pkg/processors/actualizers/provide.go](../../../pkg/processors/actualizers/provide.go)
- [pkg/cluster/impl_deployapp.go](../../../pkg/cluster/impl_deployapp.go)
