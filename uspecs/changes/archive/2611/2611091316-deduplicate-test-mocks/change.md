---
change_id: 2611091232-deduplicate-test-mocks
type: refactor
issue_url: https://untill.atlassian.net/browse/AIR-5221
domains: [prod]
scope: [apps, storage]
---

# Change request: Shared interface mocks across backend tests

Refs:

- [AIR-5221: Deduplicate test mocks in stateprovide and storages using coreutils](./issue-AIR-5221.md)

## Why

State providers, storages, query processors, and the record registry duplicate mocks for the same interfaces, increasing maintenance effort and allowing test behavior to diverge. Shared mocks should make these tests easier to maintain while preserving the behavior each test relies on.

## What

- State-provider, storage, query-processor, and record-registry tests use shared interface mocks consistently, with missing reusable mocks available from coreutils
- Query-processor tests use a shared record-backed state-value mock that preserves their record access contract
- Shared mocks support the existing tests' required nil builder results, configurable equality, and event release expectations
- No behavior change to Application or Workspace state access, storage operations, public APIs, or existing test outcomes
- Dependency-specific storage mocks remain outside coreutils to avoid import cycles, while stateful storage helpers, map-backed builders, and lightweight benchmark doubles retain their existing purpose
- Superseded local mock definitions are removed, with interface compliance and affected package tests verifying the shared replacements

## How

Decisions:

- Reuse existing shared mocks through compatible narrower interfaces instead of introducing separate mocks for every reader, writer, builder, value, or event interface
- Keep record-backed state-value mocks separate from field-addressed state-value mocks because their record accessors have different signatures
- Follow the existing testify mock convention, forwarding the original method arguments and callbacks and using explicit expectations for supported calls, including equality, nil builder results, and release, without silently falling back to real implementations or fixed defaults
- Make interface-valued mock returns accept configured nil results alongside errors, preserving absence and fallback behavior without substituting typed nil pointers
- Migrate test construction and expectations directly to shared mocks rather than retaining local aliases or adapter types that obscure ownership
- Verify compatibility primarily through existing state-provider, storage, query-processor, and record-registry tests, with interface checks for every interface used by the migrated tests; avoid direct mock invocation tests that merely check configured behavior

Assumptions:

- None

References:

- [Existing shared mock conventions and return handling](../../../../../pkg/coreutils/mock.go)
- [Shared mock interface compliance checks](../../../../../pkg/coreutils/mock_test.go)
- [State-provider mock implementations and contracts](../../../../../pkg/state/stateprovide/impl_test.go)
- [Storage mock implementations and fixture setup](../../../../../pkg/sys/storages/types_test.go)
- [Bundled state fallback for unavailable builder values](../../../../../pkg/state/stateprovide/impl_bundled_host_state.go)
- [Record insertion expectations at the CUD boundary](../../../../../pkg/sys/storages/impl_records_storage_test.go)
- [WLog event mock and release behavior](../../../../../pkg/sys/storages/impl_wlog_storage_test.go)
- [Query processor state, key-builder, record-value, and record mocks](../../../../../pkg/processors/query/impl_test.go)
- [Record registry view, key-builder, and value mocks](../../../../../pkg/istructsmem/internal/recreg/registry_test.go)
- [Distinct record-backed and field-addressed state-value contracts](../../../../../pkg/istructs/recources-types.go)

## Construction

### Tests

- [x] update: [coreutils/mock_test.go](../../../../../pkg/coreutils/mock_test.go)
  - extend: interface compliance checks for the new shared AppStructs, Records, ViewRecords, Record, CUD, and record-backed state-value mocks
  - verify: shared mocks used through narrower interfaces satisfy IValue, IValueBuilder, IRowReader, IRowWriter, IKeyBuilder, and IWLogEvent
  - retain: compile-time interface compliance while relying on existing production-consumer tests for runtime behavior
  - remove: direct mock invocation tests for configured lookup results, nil builder results, equality, record access, and CUD creation under the rule against self-fulfilling tests

- [x] update: [stateprovide/impl_test.go](../../../../../pkg/state/stateprovide/impl_test.go)
  - remove: local AppStructs, Records, ViewRecords, Record, Value, KeyBuilder, ValueBuilder, and StateValue mocks after migrating their consumers
  - retain: the dependency-specific mockStorage and stateful or intentionally inert helpers whose behavior is not represented by shared interface mocks

- [x] update: [stateprovide/types_test.go](../../../../../pkg/state/stateprovide/types_test.go)
  - migrate: fixture construction and helper return types to shared AppStructs, Records, ViewRecords, and Record mocks
  - update: fixture construction to use MustBuild and mark the fixture function as a test helper, avoiding helper-owned assertion instances
  - retain: the map-backed key builder and its equality and field-storage behavior

- [x] update: [stateprovide/impl_host_state_test.go](../../../../../pkg/state/stateprovide/impl_host_state_test.go)
  - replace: local value, key-builder, value-builder, and state-value mocks with compatible coreutils mocks
  - adapt: expectations for nil builder results and any additional calls made explicit by the shared mocks, preserving the existing storage and error assertions

- [x] update: [stateprovide/impl_bundled_host_state_test.go](../../../../../pkg/state/stateprovide/impl_bundled_host_state_test.go)
  - migrate: shared interface mocks and configure nil BuildValue results explicitly where bundled reads must fall back to storage
  - preserve: bundle behavior, stored values, and assertions on storage calls without substituting fixed equality or builder defaults

- [x] update: [storages/types_test.go](../../../../../pkg/sys/storages/types_test.go)
  - migrate: fixture construction and helper return types to shared mocks
  - update: fixture construction to use MustBuild and mark the fixture function as a test helper, with root-owned assertions for the field-copy tests
  - replace: local row readers, row writers, values, and value builders with compatible shared mock types
  - remove: superseded AppStructs, Records, ViewRecords, Record, CUD, reader, writer, value, and builder definitions after migrating all consumers
  - retain: the lightweight helpers needed by unchanged benchmark doubles and the stateful storage implementations

- [x] update: [storages/impl_records_storage_test.go](../../../../../pkg/sys/storages/impl_records_storage_test.go)
  - migrate: record and CUD dependencies to shared mocks while retaining insert, update, batch, singleton, and failure assertions
  - adapt: CUD creation expectations to include the actual entity QName and preserve existing row-write assertions

- [x] update: [storages/impl_view_records_storage_test.go](../../../../../pkg/sys/storages/impl_view_records_storage_test.go)
  - migrate: shared AppStructs, ViewRecords, record, value, and builder mocks wherever their local equivalents are used
  - preserve: existing view key, value, callback, and error behavior through explicit expectations

- [x] update: [storages/impl_wlog_storage_test.go](../../../../../pkg/sys/storages/impl_wlog_storage_test.go)
  - replace: local CUD-row and WLog-event mocks with MockCUDRow and MockPLogEvent through their compatible interfaces
  - configure: event release expectations only where Release is called, preserving callback, field-access, and error assertions
  - remove: superseded local event and CUD-row definitions

- [x] update: [query/impl_test.go](../../../../../pkg/processors/query/impl_test.go)
  - replace: local State, StateKeyBuilder, StateValue, and Record mocks with shared mocks, using the separate record-backed value mock for argument-free AsRecord access
  - adapt: state method expectations to return both values and errors
  - remove: local mock definitions after migrating both row-processing and reference-enrichment consumers
  - preserve: row-processing output, reference resolution, and existing error assertions
  - normalize: assertion ownership in modified root tests while leaving the unrelated skipped numeric-comparison test unchanged

- [x] update: [query/operator-enrichment-impl_test.go](../../../../../pkg/processors/query/operator-enrichment-impl_test.go)
  - replace: remaining local State, StateKeyBuilder, StateValue, and Record mocks with shared mocks, including argument-free record access
  - adapt: state expectations to include explicit error results while preserving nested reference-enrichment assertions
  - verify: no references remain to the local mock definitions removed by the query implementation-test item
  - clarify: scenario names, actual and expected work variables, and purpose comments, with one root assertion instance and a nil-safe cancellation error assertion

- [x] update: [recreg/registry_test.go](../../../../../pkg/istructsmem/internal/recreg/registry_test.go)
  - replace: local ViewRecords, key-builder, and value mocks with shared mocks
  - preserve: registry lookup values, key-field expectations, not-found handling, and failure propagation
  - remove: superseded local mock definitions
  - clarify: test and scenario names, actual lookup result variables, and purpose comments, moving assertion initialization to the first statement of each root test

### Shared mocks

- [x] update: [coreutils/mock.go](../../../../../pkg/coreutils/mock.go)
  - add: MockAppStructs, MockRecords, MockViewRecords, MockRecord, and MockCUD with interface-compatible signatures, original arguments, callbacks, and configured error results
  - add: MockStateRecordValue implementing IStateValue and IStateRecordValue with argument-free AsRecord access, reusing existing state-value behavior where compatible
  - compose: MockRecord from MockKey and MockStateRecordValue from MockStateValue so inherited behavior shares one expectation recorder
  - support: configured nil interface returns through the common mockResult helper in BuildValue and the new interface-returning methods, preserving strict type assertions for non-nil results
  - update: MockStateValueBuilder.Equal to use configured expectations instead of always returning true
  - preserve: strict supported-call expectations and existing event release recording, without adding state-dependent imports

### Verification

- [x] verify: shared mock migration and preserved behavior
  - run: `go test ./pkg/coreutils ./pkg/state/stateprovide ./pkg/sys/storages ./pkg/processors/query ./pkg/istructsmem/internal/recreg`
  - run: `golangci-lint run ./pkg/coreutils/... ./pkg/state/stateprovide/... ./pkg/sys/storages/... ./pkg/processors/query/... ./pkg/istructsmem/internal/recreg/... --new-from-rev HEAD`
  - run: `git diff --check`
  - audit: all consumers of changed shared mock methods for compatible expectations and all remaining local mocks in scope for duplicated responsibilities
  - verify: each modified root test using assertions initializes require exactly once as its first statement and reuses it in nested scenarios
  - confirm: stateful storage helpers, map-backed builders, benchmark doubles, and excluded dependency-specific mocks retain their existing behavior
  - result: all five affected package suites pass, scoped lint reports zero issues, and diff checks pass; assertion ownership is verified in 41 modified root tests and excluded helpers retain their existing behavior
