# Implementation plan: QPv1, QPv2 and CP: return 400 bad request on unexpected request fields

## Construction

- [x] add: [pkg/processors/utils.go](../../../pkg/processors/utils.go)
  - add: `CheckUnexpectedFields(args map[string]any, argsType appdef.IType) error` — shared helper; returns HTTP 400 for any `args` key not in `argsType.Fields()`

- [x] update: [pkg/processors/query/impl.go](../../../pkg/processors/query/impl.go)
  - update: `newExecQueryArgs` — return 400 if `argsType` is nil and non-empty `args` provided; else call `processors.CheckUnexpectedFields` before `FillFromJSON`
  - add: pipeline operator `"check unexpected request body fields"` after `"unmarshal request"` — rejects top-level fields other than `args`, `elements`, `filters`, `orderBy`, `count`, `startFrom`

- [x] update: [pkg/processors/query2/impl.go](../../../pkg/processors/query2/impl.go)
  - update: `newExecQueryArgs` — return 400 if `argsType` is nil and non-empty `Argument` provided; else call `processors.CheckUnexpectedFields` before `ObjectBuilder`; skip field check for `QNameRaw`

- [x] update: [pkg/processors/command/impl.go](../../../pkg/processors/command/impl.go)
  - add: `checkUnexpectedRequestBodyFields` — rejects root-level fields other than `args`, `unloggedArgs`, `cuds`; returns 400 if `args` is non-empty but `Param()` is nil; returns 400 if `unloggedArgs` is non-empty but `UnloggedParam()` is nil; bypasses all checks for `APIPath_Docs`

- [x] update: [pkg/processors/command/provide.go](../../../pkg/processors/command/provide.go)
  - wire `checkUnexpectedRequestBodyFields` after `unmarshalRequestBody` in the command pipeline

- [x] update: [pkg/processors/query/impl_test.go](../../../pkg/processors/query/impl_test.go)
  - add: `TestUnexpectedFields` — verifies unexpected args field returns HTTP 400 in QPv1

- [x] update: [pkg/processors/query2/impl_test.go](../../../pkg/processors/query2/impl_test.go)
  - add: `Test_newExecQueryArgs_unexpectedField` — verifies unexpected args field returns error in QPv2

- [x] update: [pkg/vit/schemaTestApp1.vsql](../../../pkg/vit/schemaTestApp1.vsql)
  - add: `QUERY QryVoid() RETURNS void WITH Tags=(WorkspaceOwnerFuncTag)` — void query for integration tests
  - add: `COMMAND CmdVoid() WITH Tags=(WorkspaceOwnerFuncTag)` — void command for integration tests

- [x] update: [pkg/vit/shared_cfgs.go](../../../pkg/vit/shared_cfgs.go)
  - register `QryVoid` via `istructsmem.NewQueryFunction` with `istructsmem.NullQueryExec`
  - register `CmdVoid` via `istructsmem.NewCommandFunction` with `istructsmem.NullCommandExec`

- [x] update: [pkg/sys/it/impl_test.go](../../../pkg/sys/it/impl_test.go)
  - add: `TestUnexpectedFields_400BadRequest` with subtests:
    - `QPv1 unexpected field inside args` — `q.sys.Echo`, unexpected key in `args`
    - `QPv1 unexpected field at root level` — `q.sys.Echo`, unknown top-level key
    - `QPv2` — `sys.Echo`, unknown key in `args`
    - `command processor` — `c.sys.CUD`, unknown top-level key
    - `QPv1 args provided for void query` — `q.app1pkg.QryVoid` via `vit.PostWS` (auto-auth)
    - `QPv2 args provided for void query` — `q.app1pkg.QryVoid` with `httpu.WithAuthorizeBy(ws.Owner.Token)`
    - `CP args provided for void command` — `c.app1pkg.CmdVoid` via `vit.PostWS`
