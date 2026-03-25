# How: QPv1 and QPv2: return 400 bad request on unexpected request fields

## Approach

- In both QPv1 and QPv2, query argument fields arrive as a `map[string]interface{}` and are silently forwarded to `requestArgsBuilder.FillFromJSON(args)` without validating the keys against the schema
- Add a helper (e.g. `checkUnexpectedFields`) that receives the incoming args map and the `appdef.IWithFields` (the resolved `argsType`) and returns an error listing any key that is not a declared field
- Call the helper in `newExecQueryArgs` in `impl.go` for each processor — after `argsType` is resolved and `args` is parsed, but before `FillFromJSON` is called; wrap the error with `http.StatusBadRequest`
- If `argsType` is `nil` (query has no declared params) and the client still sends non-empty args, also return 400
- Skip fields check if argsType is `sys.Raw`

Key files:

- QPv1 — `pkg/processors/query/impl.go`: `newExecQueryArgs`, args come from `data.AsObject("args")`
- QPv2 — `pkg/processors/query2/impl.go`: `newExecQueryArgs`, args come from `qw.queryParams.Argument`
- Field enumeration — `appdef.IWithFields.Fields()` iterator yields declared field names

References:

- [pkg/processors/query/impl.go](../../../pkg/processors/query/impl.go)
- [pkg/processors/query2/impl.go](../../../pkg/processors/query2/impl.go)

