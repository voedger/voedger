# Implementation plan: QPv1 and QPv2: return 400 bad request on unexpected request fields

## Construction

- [x] add: [pkg/processors/utils.go](../../../pkg/processors/utils.go)
  - add: `CheckUnexpectedFields(args map[string]any, argsType appdef.IType) error` — shared helper; returns HTTP 400 for any `args` key not in `argsType.Fields()`

- [x] update: [pkg/processors/query/impl.go](../../../pkg/processors/query/impl.go)
  - update: `newExecQueryArgs` — call `processors.CheckUnexpectedFields` before `FillFromJSON`; skip for nil `argsType`
  - add: pipeline operator `"check unexpected request body fields"` after `"unmarshal request"` — rejects top-level fields other than `args`, `elements`, `filters`, `orderBy`, `count`, `startFrom`

- [x] update: [pkg/processors/query2/impl.go](../../../pkg/processors/query2/impl.go)
  - update: `newExecQueryArgs` — call `processors.CheckUnexpectedFields` before `ObjectBuilder`; skip for `QNameRaw`

- [x] update: [pkg/processors/command/impl.go](../../../pkg/processors/command/impl.go)
  - add: `checkUnexpectedRequestBodyFields` — rejects root-level fields other than `args`, `unloggedArgs`, `cuds`; bypasses check for `APIPath_Docs`

- [x] update: [pkg/processors/command/provide.go](../../../pkg/processors/command/provide.go)
  - wire `checkUnexpectedRequestBodyFields` after `unmarshalRequestBody` in the command pipeline

- [x] update: [pkg/processors/query/impl_test.go](../../../pkg/processors/query/impl_test.go)
  - add: `TestUnexpectedFields` — verifies unexpected args field returns HTTP 400 in QPv1

- [x] update: [pkg/processors/query2/impl_test.go](../../../pkg/processors/query2/impl_test.go)
  - add: `Test_newExecQueryArgs_unexpectedField` — verifies unexpected args field returns error in QPv2

- [x] update: [pkg/sys/it/impl_test.go](../../../pkg/sys/it/impl_test.go)
  - add: `TestUnexpectedFields_400BadRequest` with subtests for QPv1 args, QPv1 root, QPv2 args, and command processor root
