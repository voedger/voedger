---
registered_at: 2026-05-17T12:34:13Z
change_id: 2605171234-vit-seal-option-types
baseline: 59247f436a281fd98e8d1cde3265b2fda7063bc9
issue_url: https://untill.atlassian.net/browse/AIR-3953
archived_at: 2026-05-17T12:58:39Z
---

# Change request: VIT — seal functional-option types to silence revive unexported-return

## Why

Exported option constructors in `pkg/vit` (e.g. `DoNotFailOnTimeout`, `WithClusterID`, `WithReqOpt`) return unexported function types such as `signInOptFunc` and `signUpOptFunc`. `revive`'s `unexported-return` rule flags every such function, producing persistent linter noise, and consumers in other packages cannot name the return type to store it in a variable. We do not want to suppress the warning (hides the smell) or to export the underlying `signInOpts` / `signUpOpts` structs (would leak internal fields and freeze them as public API).

## What

Adopt the canonical sealed functional-options pattern in `pkg/vit`:

- Introduce an exported interface per option family (`ISignInOpt`, `ISignUpOpt`, `IVITOpt`) with a single unexported `apply` method
- Keep the underlying option structs (`signInOpts`, `signUpOpts`) and the func carriers (`signInOptFunc`, `signUpOptFunc`) unexported; have the func types implement the sealed interfaces
- Change exported option constructors to return the new interface types
- Update the consuming methods/constructors (`(*VIT).SignIn`, `(*VIT).SignUp`, `(*VIT).SignUpDevice`, `NewVIT`, `NewVITLocalCassandra`) to accept the interfaces as variadic arguments

## Construction

- [x] update: [vit/types.go](../../../../../pkg/vit/types.go)
  - add: exported sealed interfaces `ISignInOpt`, `ISignUpOpt`, `IVITOpt`, each with a single unexported `apply…` method
  - update: keep `signInOptFunc`, `signUpOptFunc` unexported; add `apply…` methods so they implement the corresponding sealed interface
  - remove: unused `vitOptFunc` func type (no constructor produces it; future `IVITOpt` carriers can reintroduce a typed func when needed)

- [x] update: [vit/utils.go](../../../../../pkg/vit/utils.go)
  - update: `DoNotFailOnTimeout` to return `ISignInOpt`
  - update: `WithClusterID`, `WithReqOpt` to return `ISignUpOpt`
  - update: `(*VIT).SignIn` to accept `...ISignInOpt` and invoke the sealed apply method
  - update: `(*VIT).SignUp`, `(*VIT).SignUpDevice`, `getSignUpOpts` to accept `...ISignUpOpt` and invoke the sealed apply method

- [x] update: [vit/impl.go](../../../../../pkg/vit/impl.go)
  - update: `NewVIT`, `NewVITLocalCassandra` to accept `...IVITOpt` and invoke the sealed apply method
