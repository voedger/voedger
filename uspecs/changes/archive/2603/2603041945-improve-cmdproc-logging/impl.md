# Implementation plan: Improve logging in command processor

## Construction

- [x] update: [pkg/processors/command/provide.go](../../../../pkg/processors/command/provide.go)
  - add: `logEventAndCUDs` pipeline step between `putPLog` and `store`

- [x] update: [pkg/processors/command/impl.go](../../../../pkg/processors/command/impl.go)
  - add: `logEventAndCUDs(_ context.Context, cmd *cmdWorkpiece) error` function:
    - guarded by `logger.IsVerbose()`
    - `woffset` (from `cmd.pLogEvent.WLogOffset()`) and `poffset` (from `cmd.rawEvent.PLogOffset()`) set as context attributes via `logger.WithContextAttrs()`
    - event log: `logger.VerboseCtx` with `"args=<JSON>"` (args from `cmd.argsObject` via `coreutils.ObjectToMap`)
    - CUD log entries (one per `cmd.parsedCUDs` entry): `cud<i> rectype=<qName> recid=<actualID> op=<TrimString> newfields=<JSON> oldfields=<JSON>`
      - insert actual ID resolved via `idGeneratorReporter.generatedIDs`; update/activate/deactivate use `parsedCUD.id` directly
      - `oldfields` populated from `coreutils.FieldsToMap(existingRecord)`, empty `{}` for inserts

- [x] update: [pkg/processors/command/impl_test.go](../../../../pkg/processors/command/impl_test.go)
  - add: `TestLogEventAndCUDs` with two subtests using `stubPLogEvent`/`stubRawEvent` stubs
    - `verbose level emits event and cud log entries`: verifies woffset, poffset, rectype, recid (resolved), op=Insert
    - `info level emits nothing`: verifies zero output when level < verbose

