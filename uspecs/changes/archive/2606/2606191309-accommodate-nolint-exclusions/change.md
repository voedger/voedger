---
change_id: 2606191059-accommodate-nolint-exclusions
type: chore
issue_url: https://untill.atlassian.net/browse/AIR-4322
domains: [devops]
---

# Change request: Accommodate directory-level lint exclusion markers

Refs:

- [AIR-4322: voedger: accomodate .nolint files according to exclusions](./issue-AIR-4322.md)

## Why

CI action in Voedger now provides exclusions for ci-action to be used in lint-all.sh. Need to change the approach to “convetion over configuration”

## What

- create empty `.nolint` files in the dirs to be excluded, according to the paths currently provided to ci-action
- eliminate the `lint_exclude` input passed to ci-action, since `.nolint` files are already supported by ci-action

## How

Decisions:

- Adopt convention over configuration: mark each lint-excluded directory with an empty `.nolint` file instead of maintaining the `lint_exclude` input passed to `untillpro/ci-action`
- Place one empty `.nolint` marker at the root of each directory currently listed in `lint_exclude` (enumerated under Provisioning and configuration)
- Drop the now-redundant `lint_exclude` input from the three workflows that call the reusable ci-action, so the discovered `.nolint` markers become the single source of truth

Out of scope:

- The `untillpro/ci-action` reusable workflow that discovers `.nolint` markers (maintained in a separate repository)
- The local `.golangci.yml` exclusions block used for developer runs

References (internal):

- [CI workflow invoking ci-action](../../../../../.github/workflows/ci.yml)
- [PR CI workflow](../../../../../.github/workflows/ci_pr.yml)
- [full CI workflow](../../../../../.github/workflows/ci-full.yml)
- [local golangci-lint configuration](../../../../../.golangci.yml)

References (external):

- [untillpro/ci-action](https://github.com/untillpro/ci-action)
- [golangci-lint exclusions](https://golangci-lint.run/usage/configuration/)

## Provisioning and configuration

- [x] create: [cmd/vpm/testdata/.nolint](../../../../../cmd/vpm/testdata/.nolint)
  - empty marker file; signals ci-action to skip linting this directory
- [x] create: [pkg/iextengine/wazero/\_testdata/.nolint](../../../../../pkg/iextengine/wazero/_testdata/.nolint)
  - empty marker file; signals ci-action to skip linting this directory
- [x] create: [pkg/sys/it/testdata/.nolint](../../../../../pkg/sys/it/testdata/.nolint)
  - empty marker file; signals ci-action to skip linting this directory
- [x] create: [examples/airs-bp2/air/.nolint](../../../../../examples/airs-bp2/air/.nolint)
  - empty marker file; signals ci-action to skip linting this directory
- [x] update: [.github/workflows/ci.yml](../../../../../.github/workflows/ci.yml): remove the `lint_exclude` input from the ci-action call (manual edit - no CLI available)
- [x] update: [.github/workflows/ci_pr.yml](../../../../../.github/workflows/ci_pr.yml): remove the `lint_exclude` input from the ci-action call (manual edit - no CLI available)
- [x] update: [.github/workflows/ci-full.yml](../../../../../.github/workflows/ci-full.yml): remove the `lint_exclude` input from the ci-action call (manual edit - no CLI available)
- [x] update: [.github/workflows/README.md](../../../../../.github/workflows/README.md): drop `lint_exclude` from the documented ci-action call examples and workflow descriptions (manual edit - no CLI available)
