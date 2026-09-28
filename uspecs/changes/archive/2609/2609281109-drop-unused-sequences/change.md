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

- The unused sequence-storage capability is removed with no change to externally observable sequence allocation, storage, or partition-recovery behavior.
- Existing primary-key prefix values remain reserved and compatible; the legacy sequence-storage prefixes remain declared as deprecated compatibility markers.
- Sequence Trust Level behavior and configuration remain unchanged.
- Documentation remains unchanged in this change and is cleaned up separately under AIR-4980.

## How

Decisions:

- Remove the dormant feature end to end instead of retaining compatibility facades: delete the sequencer lifecycle and storage contracts, their implementations and adapters, and the application-parts sequence-storage bridge.
- Remove sequence-type metadata from the application-structures contract and configuration, including its automatic system-sequence registration, rather than replacing it with another abstraction.
- Keep the Sequence Trust Level type and constants in the existing sequencer package namespace and preserve all current consumers and configuration signatures; shrink the package to that surviving contract instead of relocating it.
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

### Sequencer and application contracts

- [x] update: [isequencer/types.go](../../../../../pkg/isequencer/types.go)
  - remove obsolete sequence identifiers, values, parameters, runtime state, and mock-storage types
  - retain the Sequence Trust Level type unchanged in its current package

- [x] update: [isequencer/consts.go](../../../../../pkg/isequencer/consts.go)
  - remove sequencer runtime defaults and retry timing constants
  - retain the three Sequence Trust Level constants unchanged

- [x] delete: [isequencer/errors.go](../../../../../pkg/isequencer/errors.go), [isequencer/interface.go](../../../../../pkg/isequencer/interface.go), [isequencer/impl.go](../../../../../pkg/isequencer/impl.go), [isequencer/provide.go](../../../../../pkg/isequencer/provide.go), and [isequencer/test_utils.go](../../../../../pkg/isequencer/test_utils.go)
  - remove the unused sequencer API, implementation, constructor, errors, and test support without changing the package documentation

- [x] update: [istructs/interface.go](../../../../../pkg/istructs/interface.go)
  - remove sequence-type metadata from the application-structures interface

- [x] update: [istructs/consts.go](../../../../../pkg/istructs/consts.go)
  - remove the unused system sequence QNames and their fixed QName identifiers without renumbering any remaining identifier

### Application structures

- [x] update: [istructsmem/appstruct-types.go](../../../../../pkg/istructsmem/appstruct-types.go)
  - remove sequence-type storage, initialization, automatic workspace sequence registration, and mutation support from application configuration

- [x] update: [istructsmem/impl.go](../../../../../pkg/istructsmem/impl.go)
  - remove the application-structures sequence-metadata accessor
  - preserve all Sequence Trust Level state and enforcement paths unchanged

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
