# How: registry — allow recreating a login after its profile was deactivated

## Approach

- Lift the "treat inactive as missing" rule into the create-login path in `pkg/registry/impl_createlogin.go`: replace the `GetCDocLoginID > 0` precheck inside `createLogin` with a `GetCDocLogin` call. Since `GetCDocLogin` in `pkg/registry/utils.go` already returns `loginExists=false` for `cdoc.registry.Login` with `sys.IsActive=false` (per the prior AIR-3892 change), the 409 "login already exists" response stays for active logins and is bypassed for deactivated ones — covering both `c.registry.CreateLogin` and `c.registry.CreateEmailLogin`, which share `createLogin`
- Rely on the existing `projectorLoginIdx` in `impl_createlogin.go` to "rewrite the existing key in `LoginIdx`": its PK is `(AppWSID, AppIDLoginHash)`, so the new `cdoc.registry.Login` produced for the same login name simply overwrites the stale `CDocLoginID` value pointing at the deactivated record. No projector or `appws.vsql` change required
- The new `cdoc.registry.Login` is `IsNew()` and triggers `invokeCreateWorkspaceIDProjector` in `impl_invokecreateworkspaceid.go`, which creates a fresh profile workspace — matching the issue's "new bare login is created" requirement; the deactivated profile workspace from the previous incarnation is left as-is (it is already `Status=Inactive` after the deactivation cascade)
- Relax the `c.sys.CreateWorkspaceID` deduplication in `pkg/sys/workspace/impl.go > execCmdCreateWorkspaceID` so the 409 "already exists" response is returned only while the existing `cdoc.sys.WorkspaceID` (referenced by `view.sys.WorkspaceIDIdx[(OwnerWSID, WSName)].IDOfCDocWorkspaceID`) is `sys.IsActive=true`. If the previous profile was deactivated (so its `cdoc.sys.WorkspaceID.sys.IsActive=false`), re-creation under the same `(OwnerWSID, WSName)` proceeds and `workspaceIDIdxProjector` overwrites the index entry by its PK `(OwnerWSID, WSName)`. The deduplication for active workspaces (and for child-workspace re-creation under an active owner) is preserved
- Close the deactivation gap for login profiles in `pkg/sys/workspace/impl_deactivate.go > projectorApplyDeactivateWorkspace` so the IsActive check above can actually fire: route the existing `c.sys.OnWorkspaceDeactivated` call to the app and pseudoWSID where `cdoc.sys.WorkspaceID` actually lives — `ownerApp` at `pseudoWSID(ownerWSID, wsName)` for child workspaces (where `ownerAppQName == projectorAppQName`), and `projectorAppQName` (= `targetApp`) at `pseudoWSID(NullWSID, wsName)` for login profiles (where `ownerAppQName != projectorAppQName`, per `pkg/registry/impl_invokecreateworkspaceid.go`). The previous unconditional call to `ownerApp` was a silent no-op for login profiles since no `cdoc.sys.WorkspaceID` exists there
- Cover the recreate-after-deactivate scenario with an IT in `pkg/sys/it/impl_deactivateworkspace_test.go` (next to the existing deactivation cascade IT): sign up a login, deactivate its profile WS via `c.sys.InitiateDeactivateWorkspace`, wait for `cdoc.registry.Login.sys.IsActive=false`, then sign up the same login again and assert it succeeds, the resulting `cdoc.registry.Login` is a new `RecordID`, the `view.registry.LoginIdx` entry now points at it, and `q.registry.IssuePrincipalToken` returns a token for the new profile

References:

- [pkg/registry/impl_createlogin.go](../../../../../pkg/registry/impl_createlogin.go)
- [pkg/registry/utils.go](../../../../../pkg/registry/utils.go)
- [pkg/registry/impl_invokecreateworkspaceid.go](../../../../../pkg/registry/impl_invokecreateworkspaceid.go)
- [pkg/registry/appws.vsql](../../../../../pkg/registry/appws.vsql)
- [pkg/sys/workspace/impl.go](../../../../../pkg/sys/workspace/impl.go)
- [pkg/sys/workspace/impl_deactivate.go](../../../../../pkg/sys/workspace/impl_deactivate.go)
- [pkg/sys/it/impl_deactivateworkspace_test.go](../../../../../pkg/sys/it/impl_deactivateworkspace_test.go)
- [pkg/sys/it/impl_signupin_test.go](../../../../../pkg/sys/it/impl_signupin_test.go)
