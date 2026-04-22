# Implementation plan: Support sys.Raw argument in Query Processor V2

## Technical design

In API V2 queries use GET requests with query parameters. The `args` query parameter carries the argument value. For `sys.Raw`, the `args` value is a raw string (not JSON-wrapped). The system must wrap it into `{"Body": rawString}` internally — consistent with how commands handle `sys.Raw` bodies.

Key changes:

- Add `RawArg string` field to `QueryParams`; `IQueryMessage` carries raw params (`map[string]string`) instead of pre-parsed `QueryParams`
- `ParseQueryParams` accepts `argQName appdef.QName`: if `sys.Raw` → store arg in `RawArg`; else → JSON unmarshal into `Argument` or return error `"invalid 'args' parameter"`
- A `"parse query params"` pipeline operator runs after `"set request type"` (when `iQuery` is known), populates `qw.queryParams`
- In `newExecQueryArgs`, detect `sys.Raw` param and use `PutChars("Body", RawArg)` instead of `FillFromJSON(Argument)`
- In OpenAPI schema generation, skip `sys.Raw` param/result from `ischema` cast (already works since Object implements IWithFields), but adjust the `args` query parameter schema to be a plain string when param is `sys.Raw`

## Construction

- [x] update: [../../pkg/processors/query2/types.go](../../pkg/processors/query2/types.go)
  - add: `RawArg string` field to `QueryParams` struct
  - update: `IQueryMessage` interface: `QueryParams() QueryParams` → `RawParams() map[string]string`
  - update: `implIQueryMessage` struct: `queryParams QueryParams` → `rawParams map[string]string`

- [x] update: [../../pkg/processors/query2/impl_queryparams.go](../../pkg/processors/query2/impl_queryparams.go)
  - update: `ParseQueryParams` signature: add `argQName appdef.QName` param; if `sys.Raw` → set `RawArg`; else → JSON unmarshal or return error `"invalid 'args' parameter"`

- [x] update: [../../pkg/processors/query2/util.go](../../pkg/processors/query2/util.go)
  - update: `NewIQueryMessage`: `queryParams QueryParams` → `rawParams map[string]string`
  - update: `newQueryWork`: remove `queryParams: msg.QueryParams()` initialisation (populated later in pipeline)

- [x] update: [../../pkg/processors/query2/impl.go](../../pkg/processors/query2/impl.go)
  - add: `"parse query params"` pipeline operator after `"set request type"`: extracts `argQName` from `qw.iQuery.Param()` (or `appdef.NullQName`), calls `ParseQueryParams`, stores result in `qw.queryParams`
  - update: `newExecQueryArgs` to check if param type is `istructs.QNameRaw`; if so, use `PutChars(processors.Field_RawObject_Body, qw.queryParams.RawArg)` instead of `FillFromJSON(qw.queryParams.Argument)`

- [x] update: [../../pkg/vvm/impl_requesthandler.go](../../pkg/vvm/impl_requesthandler.go)
  - remove: `ParseQueryParams` call and error handling; pass `request.Query` directly to `NewIQueryMessage`

- [x] update: [../../pkg/processors/query2/impl_openapi.go](../../pkg/processors/query2/impl_openapi.go)
  - update: query `args` parameter generation to use string schema when param is `sys.Raw` instead of `$ref`

- [x] update: [../../pkg/vit/schemaTestApp1.vsql](../../pkg/vit/schemaTestApp1.vsql)
  - add: `QUERY TestQryRawArg(sys.Raw) RETURNS MockQryResult` and GRANT

- [x] update: [../../pkg/vit/shared_cfgs.go](../../pkg/vit/shared_cfgs.go)
  - add: register `TestQryRawArg` query function

- [x] update: [../../pkg/sys/it/impl_qpv2_test.go](../../pkg/sys/it/impl_qpv2_test.go)
  - add: test case for query with `sys.Raw` argument type via API V2

- [x] update: [../../pkg/processors/query2/impl_queryparams_test.go](../../pkg/processors/query2/impl_queryparams_test.go)
  - add: test case for `ParseQueryParams` with non-JSON `args` value (sys.Raw scenario)
