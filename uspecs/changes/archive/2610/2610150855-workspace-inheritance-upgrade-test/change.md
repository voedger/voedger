---
change_id: 2610150815-workspace-inheritance-upgrade-test
type: test
issue_url: https://untill.atlassian.net/browse/AIR-5077
domains: [prod]
scope: [apps, storage]
---

# Change request: Workspace inheritance storage upgrade coverage

## Why

It is unclear whether workspace-scoped data and schema structures remain usable when an existing Workspace is changed to inherit from a new abstract Workspace. Integration coverage is needed to protect persisted data and schema compatibility across this upgrade scenario.

## What

Add VIT-level coverage for workspace inheritance across a storage-preserving restart:

- The restart-preserving-storage test facility supports using a new VVM configuration for the second server run.
- Integration coverage persists multiple records under the original Workspace schema, restarts the server with the same storage and a schema that adds an abstract ancestor, and verifies that the records remain readable.
- Integration coverage verifies that the existing structures of the inherited Workspace remain available after the schema upgrade.
- Backward-compatibility validation accepts the covered schema transition while continuing to reject incompatible transitions.

## How

Decisions:

- Allow the storage-preserving restart facility to launch the second run from a distinct own VIT configuration, while retaining the first-run configuration as the default for existing callers; bind both runs to the same storage factory and keyspace isolation suffix.
- Represent the pre-upgrade and post-upgrade application schemas as separate test application definitions under the same application identity, with the upgraded definition adding a common abstract Workspace ancestor without changing the existing concrete Workspace or document identities.
- Cover two concrete Workspaces that gain the common ancestor, persist records in their existing document structures before restart, and verify both record readability and effective Workspace type availability after restart through supported VIT APIs.
- Treat adding an abstract Workspace ancestor as backward-compatible when every previously effective type keeps its identity and contract; preserve the existing compatibility restrictions on removals, field changes, and other contract mutations.

Assumptions:

- The schema upgrade changes only Workspace ancestry and adds the abstract ancestor; existing application, Workspace, document, and field identities remain stable.
- Persisted records require no rewrite or storage migration when their effective document definitions are unchanged by the new ancestry relationship.

Out of scope:

- Compatibility of upgrades that rename, remove, reorder, or otherwise change existing document structures.
- Data migration for schema changes that alter persisted record layouts.

References:

- [storage-preserving VIT restart lifecycle](../../../../../pkg/vit/utils.go)
- [VIT configuration construction](../../../../../pkg/vit/impl_vitcfg.go)
- [fresh VVM creation for each VIT launch](../../../../../pkg/vit/impl.go)
- [existing restart integration coverage](../../../../../pkg/sys/it/impl_test.go)
- [test application schema provisioning](../../../../../pkg/vit/shared_cfgs.go)
- [workspace inheritance contract](../../../../../pkg/appdef/interface_workspace.go)
- [backward-compatibility tree and constraints](../../../../../pkg/appdefcompat/impl.go)

## Construction

- [x] update: [sys/it/impl_test.go](../../../../../pkg/sys/it/impl_test.go)
  - extend: restart-helper coverage to prove that the original call form still uses the same configuration for both launches.
  - add: coverage with a distinct second-run configuration, asserting that its VVM settings take effect while the original records remain readable in the same workspace.

- [x] create: [sys/it/impl_workspace_inheritance_test.go](../../../../../pkg/sys/it/impl_workspace_inheritance_test.go)
  - Provide focused VIT coverage for adding a common abstract ancestor to two existing concrete Workspace kinds across a server restart.
  - Include test-local application builders and original/upgraded VSQL fixtures under an existing test application identity so both launches build their respective schemas without reusing a cached test schema.
  - Keep the ticket's `Work`/`Doc1` shape with `IntFld int32` and `StrFld varchar`, and include a second concrete Workspace kind that gains the same empty abstract `Base` ancestor; keep existing type and field identities unchanged.
  - Before restart, create workspace instances of both kinds, persist multiple documents with distinct integer/string values, and retain their workspace IDs, record IDs, and effective type definitions for comparison.
  - After restart with the upgraded application definition and preserved storage, assert that both workspace IDs and document IDs still address the original data and compare every existing document's field values through normal query APIs.
  - Verify that both upgraded Workspace definitions inherit from `Base` and retain every previously effective type, including descriptors and system-provided structures, with the same kind and relevant field/container contracts.
  - Exercise the original persisted records and workspace instances directly; do not recreate their data during the second-run setup.

- [x] update: [pkg/appdefcompat/impl_test.go](../../../../../pkg/appdefcompat/impl_test.go)
  - add: a focused old/new application-definition comparison proving that introducing an empty abstract ancestor for two concrete Workspaces passes backward-compatibility validation.
  - add: negative comparisons showing that removing an existing effective type or changing an existing field's data kind is still rejected, including when combined with the ancestry addition.

- [x] update: [pkg/vit/utils.go](../../../../../pkg/vit/utils.go)
  - extend: `TestRestartPreservingStorage` with an optional trailing second-run own `VITConfig`, preserving the existing call form and same-config default.
  - validate: both selected configurations are owned, and reject multiple second-run configurations with a clear assertion.
  - apply: the same captured storage factory and isolation suffix to both launches after their ordinary configuration options, so second-run options cannot replace the preserved storage.
  - isolate: helper-added options in local configuration copies with independent option slices so caller configurations remain reusable without accumulating restart state.
  - retain: teardown of the first VIT before constructing the second and deferred cleanup for each launch; document the optional configuration and storage guarantees.

- [x] update if required: [pkg/appdefcompat/impl.go](../../../../../pkg/appdefcompat/impl.go)
  - verified: the new ancestry-addition and incompatible-transition tests pass with the existing validator; no rule change is required.
  - change: only the rule demonstrated by the new compatibility tests to reject the supported ancestry-only transition; leave the validator unchanged if that transition already passes.
  - preserve: checks on existing effective types, descriptors, abstractness, fields, containers, and packages; do not bypass validation or introduce general ancestor compatibility rules beyond this case.

- [x] verify: targeted restart, inheritance-upgrade, and compatibility coverage
  - run: the existing restart test and the new inheritance-upgrade tests in `./pkg/sys/it`, selecting their test names with `go test -run`.
  - run: `go test ./pkg/appdefcompat` and format changed Go files with `gofmt`.
  - confirm: existing restart callers compile with the additive helper API and the original same-configuration restart test still passes.
  - passed: `go test ./pkg/sys/it -run '^(TestVITResetPreservingStorage|TestWorkspaceInheritanceUpgrade)$' -count=1 -timeout=60s` and `go test ./pkg/appdefcompat`; changed Go files formatted with `gofmt`.
  - reviewed: construction stages A and B; made the processor-count change explicit (three to two) and asserted that the new ancestor is absent before upgrade and abstract afterward. No outstanding review findings.
