---
change_id: 2609280711-drop-unused-sequences
type: refactor
issue_url: https://untill.atlassian.net/browse/AIR-4998
domains: [prod]
scope: [apps, storage]
breaking: true
---

# Change request: Removal of unused sequence storage

Refs:

- [AIR-4998: voedger: drop unused sequences feature](./issue-AIR-4998.md)

## Why

The unused sequence-storage feature adds obsolete code and maintenance burden to the Voedger application platform. Removing it simplifies the storage and partition-recovery implementation while retaining compatibility boundaries required by existing deployments.

## What

- The unused sequence-storage capability and the now-empty `isequencer` package are removed with no change to externally observable sequence allocation, storage, or partition-recovery behavior.
- Existing primary-key prefix values remain reserved and compatible; the legacy sequence-storage prefixes remain declared as deprecated compatibility markers.
- Sequence Trust Level behavior and configuration remain unchanged, while their shared type and constants move to `istructs`.
- Obsolete package-local `isequencer` documentation is deleted with the package; maintained product specifications remain unchanged and are cleaned up separately under AIR-4980.

## How

Decisions:

- Remove the dormant feature end to end instead of retaining compatibility facades: delete the sequencer lifecycle and storage contracts, their implementations and adapters, and the application-parts sequence-storage bridge.
- Remove sequence-type metadata from the application-structures contract and configuration, including its automatic system-sequence registration, rather than replacing it with another abstraction.
- Move the Sequence Trust Level type and constants to the shared `istructs` contract package, preserve their names and numeric values, and migrate every caller before deleting `isequencer` completely.
- Leave any previously persisted sequence-storage rows untouched and inaccessible: retain the deprecated primary-key prefix slots, do not reuse their numeric values, and introduce no data migration or cleanup path.
- Remove tests that exercise only the dormant subsystem while retaining the prefix-value contract and using the existing command-recovery, identifier-generation, and Sequence Trust Level suites to verify the active behavior remains intact.

Assumptions:

- None

References:

- [obsolete sequencer and storage contracts](../../../../../pkg/isequencer/interface.go)
- [obsolete application sequence metadata](../../../../../pkg/istructsmem/appstruct-types.go)
- [application-structures API boundary](../../../../../pkg/istructs/interface.go)
- [legacy sequence-storage persistence adapter](../../../../../pkg/vvm/storage/impl_seqstorage.go)
- [reserved VVM storage key prefixes](../../../../../pkg/vvm/storage/consts.go)
- [active command partition recovery and identifier reconstruction](../../../../../pkg/processors/command/impl.go)
- [active Sequence Trust Level enforcement](../../../../../pkg/istructsmem/impl.go)
- [partition recovery behavior coverage](../../../../../pkg/processors/command/impl_test.go)
- [Sequence Trust Level behavior coverage](../../../../../pkg/istructsmem/impl_seqtrust_test.go)
- [storage key-prefix compatibility coverage](../../../../../pkg/vvm/storage/consts_test.go)

## Construction

### Tests

- [x] delete: [isequencer/impl_race_test.go](../../../../../pkg/isequencer/impl_race_test.go), [isequencer/impl_test.go](../../../../../pkg/isequencer/impl_test.go), and [isequencer/isequencer_test.go](../../../../../pkg/isequencer/isequencer_test.go)
  - remove unit, race, recovery, flushing, caching, and lifecycle coverage owned exclusively by the deleted sequencer implementation

- [x] delete: [internal/seqstorage/impl_test.go](../../../../../pkg/appparts/internal/seqstorage/impl_test.go)
  - remove adapter coverage for reconstructing and persisting the obsolete sequence values and PLog offset

- [x] delete: [vvm/storage/impl_seqstorage_test.go](../../../../../pkg/vvm/storage/impl_seqstorage_test.go)
  - remove persistence-adapter coverage for the deleted sequence-storage read and write paths

- [x] update: [iauthnzimpl/impl_test.go](../../../../../pkg/iauthnzimpl/impl_test.go)
  - remove the obsolete sequence-metadata method from the application-structures test double after that method leaves the interface

- [x] update: [istructs/consts_test.go](../../../../../pkg/istructs/consts_test.go)
  - remove fixed-value assertions for the deleted system sequence QName identifiers while preserving assertions for all remaining identifiers
  - add compile-time assertions that the three Sequence Trust Level numeric values remain `0`, `1`, and `2`

- [x] update: [appparts/example_limit_test.go](../../../../../pkg/appparts/example_limit_test.go), [appparts/example_test.go](../../../../../pkg/appparts/example_test.go), and [appparts/impl_test.go](../../../../../pkg/appparts/impl_test.go)
  - use the relocated `istructs.SequencesTrustLevel_0` constant in application-structures test setup

- [x] update: [istructsmem/appstruct-types_test.go](../../../../../pkg/istructsmem/appstruct-types_test.go), [istructsmem/bench_test.go](../../../../../pkg/istructsmem/bench_test.go), [istructsmem/event-types_test.go](../../../../../pkg/istructsmem/event-types_test.go), [istructsmem/impl_test.go](../../../../../pkg/istructsmem/impl_test.go), and [istructsmem/records-types_test.go](../../../../../pkg/istructsmem/records-types_test.go)
  - replace test and benchmark imports of `isequencer` with the relocated `istructs` trust-level contract

- [x] update: [istructsmem/resources-types_test.go](../../../../../pkg/istructsmem/resources-types_test.go), [istructsmem/test_test.go](../../../../../pkg/istructsmem/test_test.go), [istructsmem/validation_test.go](../../../../../pkg/istructsmem/validation_test.go), and [istructsmem/viewrecords-types_test.go](../../../../../pkg/istructsmem/viewrecords-types_test.go)
  - replace test fixture imports of `isequencer` with `istructs`

- [x] update: [wazero/impl_test.go](../../../../../pkg/iextengine/wazero/impl_test.go), [parser/impl_test.go](../../../../../pkg/parser/impl_test.go), [actualizers/impl_test.go](../../../../../pkg/processors/actualizers/impl_test.go), [command/impl_test.go](../../../../../pkg/processors/command/impl_test.go), and [query/impl_test.go](../../../../../pkg/processors/query/impl_test.go)
  - migrate application-structures test setup to `istructs.SequencesTrustLevel_0`

- [x] update: [collection/collection_test.go](../../../../../pkg/sys/collection/collection_test.go) and [storages/impl_event_storage_test.go](../../../../../pkg/sys/storages/impl_event_storage_test.go)
  - migrate system test setup to the relocated trust-level constant

### Sequencer and application contracts

- [x] delete: [isequencer/types.go](../../../../../pkg/isequencer/types.go) and [isequencer/consts.go](../../../../../pkg/isequencer/consts.go)
  - remove obsolete sequence identifiers, values, parameters, runtime state, mock-storage types, runtime defaults, and retry timing constants
  - relocate the surviving Sequence Trust Level contract to `istructs` before deleting the final package declarations

- [x] delete: [isequencer/errors.go](../../../../../pkg/isequencer/errors.go), [isequencer/interface.go](../../../../../pkg/isequencer/interface.go), [isequencer/impl.go](../../../../../pkg/isequencer/impl.go), [isequencer/provide.go](../../../../../pkg/isequencer/provide.go), and [isequencer/test_utils.go](../../../../../pkg/isequencer/test_utils.go)
  - remove the unused sequencer API, implementation, constructor, errors, and test support

- [x] delete: [isequencer/README.md](../../../../../pkg/isequencer/README.md) and [isequencer/design.md](../../../../../pkg/isequencer/design.md)
  - remove obsolete documentation for the deleted sequencer package and implementation

- [x] update: [istructs/interface.go](../../../../../pkg/istructs/interface.go)
  - remove sequence-type metadata from the application-structures interface

- [x] update: [istructs/consts.go](../../../../../pkg/istructs/consts.go)
  - remove the unused system sequence QNames and their fixed QName identifiers without renumbering any remaining identifier
  - host the three relocated Sequence Trust Level constants with their existing names and values

- [x] update: [istructs/types.go](../../../../../pkg/istructs/types.go)
  - host the relocated `SequencesTrustLevel` shared contract type

### Application structures

- [x] update: [istructsmem/appstruct-types.go](../../../../../pkg/istructsmem/appstruct-types.go)
  - remove sequence-type storage, initialization, automatic workspace sequence registration, and mutation support from application configuration

- [x] update: [istructsmem/impl.go](../../../../../pkg/istructsmem/impl.go)
  - remove the application-structures sequence-metadata accessor
  - preserve all Sequence Trust Level state and enforcement paths while consuming the type and constants from `istructs`

- [x] update: [istructsmem/provide.go](../../../../../pkg/istructsmem/provide.go)
  - accept the relocated `istructs.SequencesTrustLevel` contract without changing provider behavior

### Trust-level consumers and wiring

- [x] update: [teststate/impl.go](../../../../../pkg/state/teststate/impl.go), [teststate/impl_new.go](../../../../../pkg/state/teststate/impl_new.go), and [vit/impl.go](../../../../../pkg/vit/impl.go)
  - update shared test environments to configure Sequence Trust Level through `istructs`

- [x] update: [vvm/impl_cfg.go](../../../../../pkg/vvm/impl_cfg.go), [vvm/provide.go](../../../../../pkg/vvm/provide.go), and [vvm/types.go](../../../../../pkg/vvm/types.go)
  - expose, default, and pass through the relocated `istructs.SequencesTrustLevel` type without changing configuration behavior

- [x] regenerate: [vvm/wire_gen.go](../../../../../pkg/vvm/wire_gen.go)
  - remove the `isequencer` import and wire the relocated trust-level type through the generated provider graph

### Sequence-storage adapters

- [x] delete: [internal/seqstorage/impl.go](../../../../../pkg/appparts/internal/seqstorage/impl.go), [internal/seqstorage/provide.go](../../../../../pkg/appparts/internal/seqstorage/provide.go), and [internal/seqstorage/type.go](../../../../../pkg/appparts/internal/seqstorage/type.go)
  - remove the unused application-parts bridge between event-log scanning and sequence persistence

- [x] delete: [vvm/storage/impl_seqstorage.go](../../../../../pkg/vvm/storage/impl_seqstorage.go)
  - remove sequence-number and partition-offset reads and writes from VVM system storage

- [x] update: [vvm/storage/provide.go](../../../../../pkg/vvm/storage/provide.go)
  - remove construction of the deleted VVM sequence-storage adapter while preserving elections and application-TTL storage providers

- [x] update: [vvm/storage/consts.go](../../../../../pkg/vvm/storage/consts.go)
  - mark both legacy sequence-storage primary-key prefixes as deprecated compatibility slots
  - keep their numeric values and surrounding prefix order unchanged so the slots cannot be reused

### Verification

- [x] verify: [vvm/storage/consts_test.go](../../../../../pkg/vvm/storage/consts_test.go), [command/impl_test.go](../../../../../pkg/processors/command/impl_test.go), and [istructsmem/impl_seqtrust_test.go](../../../../../pkg/istructsmem/impl_seqtrust_test.go)
  - retain coverage of primary-key prefix values `2` and `3`, partition recovery and identifier reconstruction, and every Sequence Trust Level
  - verify no non-documentation source still references the removed sequencer, sequence-storage, or sequence-metadata contracts
  - run the repository-wide Go test suite to catch stale interface implementations and imports across package boundaries
