# Implementation plan: Support sys.Raw argument in Query Processor V2

## Technical design

In API V2 queries use GET requests with query parameters. The `args` query parameter carries the argument value. For `sys.Raw`, the `args` value is a raw string (not JSON-wrapped). The system must wrap it into `{"Body": rawString}` internally — consistent with how commands handle `sys.Raw` bodies.

Key changes:

- Store the raw `args` string in `QueryParams` so it is available when the param type is known
- In `ParseQueryParams`, tolerate non-JSON `args` values (store raw string, set `Argument` to nil)
- In `newExecQueryArgs`, detect `sys.Raw` param and use `PutChars("Body", rawArgs)` instead of `FillFromJSON`
- In OpenAPI schema generation, skip `sys.Raw` param/result from `ischema` cast (already works since Object implements IWithFields), but adjust the `args` query parameter schema to be a plain string when param is `sys.Raw`

## Construction

- [x] update: [../../pkg/processors/query2/types.go](../../pkg/processors/query2/types.go)
  - add: `RawArgs string` field to `QueryParams` struct

- [x] update: [../../pkg/processors/query2/impl_queryparams.go](../../pkg/processors/query2/impl_queryparams.go)
  - update: `ParseQueryParams` to store raw `args` value in `RawArgs`; if JSON parse fails, keep `Argument` nil instead of returning error

- [x] update: [../../pkg/processors/query2/impl.go](../../pkg/processors/query2/impl.go)
  - update: `newExecQueryArgs` to check if param type is `istructs.QNameRaw`; if so, use `PutChars(processors.Field_RawObject_Body, rawArgs)` instead of `FillFromJSON`

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
