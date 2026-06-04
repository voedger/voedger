---
change_id: 2605271242-dedup-appdef-kind-tests
type: test
issue_url: https://untill.atlassian.net/browse/AIR-4087
scope: apps
---

# Change request: Consolidate copy-paste tests in pkg/appdef for enum and doc kinds

Refs:

- [AIR-4087: voedger: consolidate copy-paste tests in pkg/appdef: enum kinds and doc kinds](./issue-AIR-4087.md)

## Why

The `pkg/appdef` test suite carries two clusters of near-identical, copy-pasted tests that scale linearly with every new kind: seven per-enum `MarshalText`/`TrimString` files and four 83-line twin doc-kind files. The duplication makes adding a new kind error-prone, forces N identical edits for any shared-logic fix, and inflates review surface for every kind-related change.

## What

Existing per-kind test coverage in `pkg/appdef` is preserved while the duplication is removed:

- A unified fixture-driven scenario covers `MarshalText` and `TrimString` for every `~uint8` enum kind currently tested individually, at the same assertion granularity as before
- A unified fixture-driven scenario covers build, find-by-name, container-kind, nil-on-unknown, and enumeration for every doc kind currently tested individually, including the system-document filtering asymmetry specific to `CDoc`
- Adding a new enum or doc kind requires appending one fixture entry rather than creating a new test file
- No production code in `pkg/appdef` changes and the full `./pkg/appdef/...` suite stays green

In addition, the actionable `dupl` findings adjacent to the above clusters are eliminated:

- The 126-line and 28-line workspace-with-ACL fixture blocks in `pkg/appdef/acl/provide_test.go` are extracted into shared setup helpers
- The parallel `QName` / `AppQName` JSON marshal/unmarshal and `UnmarshalInvalidString` tests in `pkg/appdef/utils_qname_test.go` are unified via a small generic helper
- The duplicated `States` / `Intents` panic subtests in `pkg/appdef/internal/extensions/storage_test.go` are merged into a table-driven loop

## How

Decisions:

- Place the consolidated enum-kind scenario in a new `pkg/appdef/utils_kinds_test.go` built on a shared `baseEnum` constraint (`~uint8 + String()`), two specialised constraints `enumKind` and `enumTrimmer` for the `MarshalText` / `TrimString` axes, and a single generic case struct `enumCase[T baseEnum]` reused by both helpers
- Place the consolidated doc-kind scenario in a new `pkg/appdef/internal/structures/docs_test.go` built on a `docFixture` struct with function-pointer fields, iterated via a single table loop
- Prefer the function-pointer fixture over a fully generic scenario for doc kinds, because Go generics cannot hold heterogeneous instantiations in one slice; small generic adapters lift typed iterators (`iter.Seq[ICDoc]`, etc.) into the common `iter.Seq[appdef.IDoc]` / `iter.Seq[appdef.IRecord]` used by the fixture
- Encode CDoc's system-document filtering asymmetry as a `skipSystem` flag on the fixture so the other three kinds stay symmetrical
- Drop the original `IsXDoc()` / `IsXRecord()` smoke calls — they assert nothing and would add per-fixture noise

Adjacent `dupl` findings, decisions:

- The `acl/provide_test.go` setup blocks are extracted as package-level helpers `buildAppWithFieldACL` and `buildAppWithAncestors`, each returning the built `appdef.IAppDef` plus any common fixture names the parent tests need
- The `utils_qname_test.go` consolidation uses a single generic `testJSONRoundtrip[T any]` helper parametrised by a constructor, a struct-field accessor, and a sample value — covering both `QName` and `AppQName` round-trips at the same assertion granularity; a parallel `testUnmarshalInvalidJSON[T]` covers the invalid-string cases
- The `storage_test.go` consolidation replaces the twin subtests with a table iterating over a `{name, accessor}` slice that selects either `prj.States()` or `prj.Intents()` — assertions and helpers remain unchanged

Out of scope:

- Production code changes in `pkg/appdef` (including the `gdoc.go`/`wdoc.go` `dupl` warning, which is intrinsic to the interface-required marker pattern)
- `ExampleXxx()` duplication in `pkg/appdef/constraints/example_test.go` and `pkg/appdef/filter/example_types_test.go` — Go's runnable-example convention requires each function to be self-contained with its own `// Output:` block
- Adding `MarshalText()` to `RateScope` to bring it into method-symmetry with sibling enum kinds
- Similar consolidation opportunities outside the clusters above (views, commands, queries, projectors)

References:

- [per-enum tests to be trimmed (DataKind)](../../../../../pkg/appdef/utils_data_test.go)
- [per-enum tests to be trimmed (TypeKind)](../../../../../pkg/appdef/utils_type_test.go)
- [per-enum tests to be trimmed (FilterKind)](../../../../../pkg/appdef/utils_filter_test.go)
- [common doc/record builder and result interfaces consumed by the doc fixture](../../../../../pkg/appdef/interface_structure.go)
- [typed per-kind lookup and enumeration helpers consumed by the doc fixture](../../../../../pkg/appdef/utils_type.go)

## Construction

- [x] create: [appdef/utils_kinds_test.go](../../../../../pkg/appdef/utils_kinds_test.go)
  - Consolidated `appdef_test` scenario for `MarshalText` and `TrimString` of every `~uint8` enum kind
  - Shared constraint `baseEnum` (`~uint8 + String()`); specialised constraints `enumKind` (adds `MarshalText`) and `enumTrimmer` (adds `TrimString`); single generic case struct `enumCase[T baseEnum]`
  - Generic helpers `testEnumMarshalText[T enumKind]` and `testEnumTrimString[T enumTrimmer]` driven by a per-kind table
  - `TestEnumKinds_MarshalText` covers `ConstraintKind`, `DataKind`, `ExtensionEngineKind`, `FilterKind`, `TypeKind`, `LimitFilterOption`
  - `TestEnumKinds_TrimString` covers the six kinds above plus `RateScope`
  - Each kind asserts basic in-range cases plus the out-of-range fallback (`KindName(N)` for `MarshalText`, `String()` parity for `TrimString`)

- [x] create: [structures/docs_test.go](../../../../../pkg/appdef/internal/structures/docs_test.go)
  - Consolidated `structures_test` scenario for `CDoc`, `GDoc`, `ODoc`, `WDoc`
  - Fixture struct `docFixture` with `name`, `docKind`, `recKind`, `skipSystem` and six function-pointer fields (`addDoc`, `addRec`, `findDoc`, `findRec`, `enumDocs`, `enumRecs`)
  - Generic adapters `seqAsDoc[T appdef.IDoc]` / `seqAsRec[T appdef.IRecord]` lift typed iterators into the common `iter.Seq[appdef.IDoc]` / `iter.Seq[appdef.IRecord]` used by the fixture
  - `Test_Docs` iterates `[]docFixture{cdocFix, gdocFix, odocFix, wdocFix}` via `t.Run(fx.name, ...)`
  - Each kind verifies build, find-by-name and kind, container-kind, nil-on-unknown, and enumeration against both the AppDef root and the workspace
  - `skipSystem` filter applied only for `CDoc` to ignore platform-pre-registered system documents

- [x] update: [appdef/utils_data_test.go](../../../../../pkg/appdef/utils_data_test.go)
  - remove: `TestConstraintKind_MarshalText`, `TestConstraintKind_TrimString`, `TestDataKindType_MarshalText`, `TestDataKind_TrimString` (coverage moved to `utils_kinds_test.go`)

- [x] update: [appdef/utils_type_test.go](../../../../../pkg/appdef/utils_type_test.go)
  - remove: `TestTypeKind_MarshalText`, `TestTypeKindTrimString` (coverage moved to `utils_kinds_test.go`)

- [x] update: [appdef/utils_filter_test.go](../../../../../pkg/appdef/utils_filter_test.go)
  - remove: `TestFilterKind_MarshalText`, `TestFilterKindTrimString` (coverage moved to `utils_kinds_test.go`)

- [x] remove: [appdef/utils_extension_test.go](../../../../../pkg/appdef/utils_extension_test.go)
  - File becomes empty after extracting `TestExtensionEngineKind_MarshalText` and `TestExtensionEngineKindTrimString`

- [x] remove: [appdef/utils_ratelimit_test.go](../../../../../pkg/appdef/utils_ratelimit_test.go)
  - File becomes empty after extracting `TestRateScopeTrimString`, `Test_LimitFilterOption_MarshalText`, and `TestLimitFilterOptionTrimString`

- [x] remove: [structures/cdoc_test.go](../../../../../pkg/appdef/internal/structures/cdoc_test.go)
  - Coverage moved to `docs_test.go` under the `CDoc` fixture

- [x] remove: [structures/gdoc_test.go](../../../../../pkg/appdef/internal/structures/gdoc_test.go)
  - Coverage moved to `docs_test.go` under the `GDoc` fixture

- [x] remove: [structures/odoc_test.go](../../../../../pkg/appdef/internal/structures/odoc_test.go)
  - Coverage moved to `docs_test.go` under the `ODoc` fixture

- [x] remove: [structures/wdoc_test.go](../../../../../pkg/appdef/internal/structures/wdoc_test.go)
  - Coverage moved to `docs_test.go` under the `WDoc` fixture

- [x] update: [acl/provide_test.go](../../../../../pkg/appdef/acl/provide_test.go)
  - Extract `buildAppWithFieldACL` covering the 126-line workspace+ACL setup duplicated at 179-305 and 871-997
  - Extract `buildAppWithAncestors` covering the 28-line ancestors-and-grants setup duplicated at 771-799 and 1190-1218

- [x] update: [appdef/utils_qname_test.go](../../../../../pkg/appdef/utils_qname_test.go)
  - Add `testJSONRoundtrip[T comparable, PT *T]` helper covering the marshal/unmarshal/structure-embedding/map-key cases shared by `TestBasicUsage_QName_JSon` and `TestBasicUsage_AppQName_JSon`
  - Add `testUnmarshalInvalidJSON[T comparable, PT *T]` helper covering the cases shared by `TestQName_UnmarshalInvalidString` and `TestAppQName_UnmarshalInvalidString`
  - The duplicated `MustParse*` blocks (`TestMustParseQName` vs `TestMustParseFullQName`) remain as-is (different types, no shared interface)

- [x] update: [structures/extensions/storage_test.go](../../../../../pkg/appdef/internal/extensions/storage_test.go)
  - Replace the twin `States` / `Intents` panic subtests inside `should be panics` with a single table-driven loop selecting the accessor via a closure
