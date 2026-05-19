# Implementation plan: Allow recreating a login that was deactivated

## Construction

- [x] update: [pkg/registry/impl_createlogin.go](../../../../../pkg/registry/impl_createlogin.go)
  - update: in `createLogin`, replace the `GetCDocLoginID > 0` precheck with `GetCDocLogin` so the existing IsActive filter (introduced for AIR-3892) is honored — an existing-but-deactivated `cdoc.registry.Login` no longer triggers the 409 "login already exists" response and a new `cdoc.registry.Login` is created instead; the `projectorLoginIdx` already overwrites `view.registry.LoginIdx` by primary key `(AppWSID, AppIDLoginHash)`, so the stale `CDocLoginID` is rewritten without any projector or `appws.vsql` change
- [x] update: [pkg/sys/workspace/impl.go](../../../../../pkg/sys/workspace/impl.go)
  - update: in `execCmdCreateWorkspaceID`, when `view.sys.WorkspaceIDIdx[(OwnerWSID, WSName)]` already exists, additionally read `cdoc.sys.WorkspaceID` by `IDOfCDocWorkspaceID` and return the 409 conflict only if its `sys.IsActive=true`; if it is `false` (previous profile was deactivated), proceed with creation — `workspaceIDIdxProjector` then overwrites the index entry by its PK `(OwnerWSID, WSName)`
- [x] update: [pkg/sys/workspace/impl_deactivate.go](../../../../../pkg/sys/workspace/impl_deactivate.go)
  - update: in `projectorApplyDeactivateWorkspace`, route the single `c.sys.OnWorkspaceDeactivated` call to the app and pseudoWSID where `cdoc.sys.WorkspaceID` actually lives — `ownerApp` at `pseudoWSID(ownerWSID, wsName)` for child workspaces, and `projectorAppQName` (= `targetApp`) at `pseudoWSID(NullWSID, wsName)` for login profiles (`ownerAppQName != projectorAppQName`, per `pkg/registry/impl_invokecreateworkspaceid.go`) — so its `sys.IsActive` is flipped to `false` and the new `execCmdCreateWorkspaceID` IsActive check can detect the deactivated state on re-creation; the previous call to `ownerApp` was a silent no-op for login profiles since no `cdoc.sys.WorkspaceID` exists there
- [x] update: [pkg/sys/it/impl_deactivateworkspace_test.go](../../../../../pkg/sys/it/impl_deactivateworkspace_test.go)
  - add: subtest under `TestDeactivateUserProfile` that signs up the same login again after deactivation completes and asserts:
    - `c.registry.CreateLogin` succeeds (no 409)
    - the resulting `cdoc.registry.Login` has a different `RecordID` than the deactivated one
    - `view.registry.LoginIdx` for the same `(AppWSID, AppName/LoginHash)` now points at the new `CDocLoginID`
    - `q.registry.IssuePrincipalToken` returns a token for the recreated login and resolves a new `ProfileWSID` distinct from the deactivated profile
- [x] Review
