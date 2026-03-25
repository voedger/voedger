# How: QPv1, QPv2, and CP: return 400 bad request on unexpected request fields

## Approach

- In both QPv1 and QPv2, query argument fields arrive as a `map[string]interface{}` and are silently forwarded to `requestArgsBuilder.FillFromJSON(args)` without validating the keys against the schema
- Add a helper (e.g. `checkUnexpectedFields`) that receives the incoming args map and the `appdef.IWithFields` (the resolved `argsType`) and returns an error listing any key that is not a declared field
- Call the helper in `newExecQueryArgs` in `impl.go` for each processor — after `argsType` is resolved and `args` is parsed, but before `FillFromJSON` is called; wrap the error with `http.StatusBadRequest`
- If `argsType` is `nil` (query/command has no declared params) and the client still sends non-empty `args`, return 400; same logic applies to `unloggedArgs` in CP
- Skip fields check if argsType is `sys.Raw`
- For CP root-level validation: insert `checkUnexpectedRequestBodyFields` after `unmarshalRequestBody` in the pipeline; it checks both unknown top-level keys and void-args cases using `iCommand.Param()` and `iCommand.UnloggedParam()`
- Bypass all body checks for `APIPath_Docs` since doc CRUD sends record fields directly in the body
- For void-args integration tests, use app-specific functions (`q.app1pkg.QryVoid`, `c.app1pkg.CmdVoid`) declared in the test schema with `WorkspaceOwnerFuncTag`; these require workspace owner authorization — use `vit.PostWS` (auto-auth) for QPv1/CP and `httpu.WithAuthorizeBy(ws.Owner.Token)` for QPv2 GET requests

Key files:

- QPv1 — `pkg/processors/query/impl.go`: `newExecQueryArgs` (void-args check), pipeline operator for root-level field validation
- QPv2 — `pkg/processors/query2/impl.go`: `newExecQueryArgs` (void-args check + field validation)
- CP — `pkg/processors/command/impl.go`: `checkUnexpectedRequestBodyFields` (unknown keys + void-args check)
- Shared — `pkg/processors/utils.go`: `CheckUnexpectedFields` helper

References:

- [pkg/processors/query/impl.go](../../../pkg/processors/query/impl.go)
- [pkg/processors/query2/impl.go](../../../pkg/processors/query2/impl.go)
- [pkg/processors/command/impl.go](../../../pkg/processors/command/impl.go)
