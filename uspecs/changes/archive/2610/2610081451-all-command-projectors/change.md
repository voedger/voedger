---
change_id: 2610081343-all-command-projectors
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-5054
domains: [prod]
scope: [apps]
---

# Change request: Projectors triggered by all commands

Refs:

- [AIR-5054: voedger: support `AFTER EXECUTE ON ALL COMMANDS` syntax](./issue-AIR-5054.md)

## Why

Voedger application developers need a single projector declaration for behavior that must follow every command. This avoids enumerating commands individually and supports cross-cutting projection behavior in VSQL applications.

## What

In the Voedger application platform's apps context:

- VSQL accepts projector declarations using `AFTER EXECUTE ON ALL COMMANDS`.
- A projector declared for all commands is triggered after any command executes.
- Existing command-specific projector declarations retain their behavior.

## How

Decisions:

- Extend the existing projector trigger grammar with an explicit all-commands target and map it directly into the application definition during schema construction.
- Represent the declaration as the existing execute operation combined with a command-type filter, so the shared projector matcher handles both synchronous and asynchronous execution without a separate runtime path.
- Apply the trigger to every command type visible to the application, including built-in and imported commands, rather than expanding the declaration into a fixed list of command names.
- Preserve existing named-command, parameter-based, error-handling, state, intent, and synchronization semantics, and verify compatibility at the parser/model boundary and across projector execution paths.

Assumptions:

- None

Out of scope:

- New all-queries, tag-filtered, workspace-scoped, or parameter-based all-command projector syntax.

References:

- [projector trigger grammar and syntax model](../../../../../pkg/parser/types.go)
- [projector trigger analysis](../../../../../pkg/parser/impl_analyse.go)
- [projector application-definition construction](../../../../../pkg/parser/impl_build.go)
- [projector event filtering model](../../../../../pkg/appdef/internal/extensions/projector.go)
- [shared runtime trigger matching](../../../../../pkg/processors/actualizers/types.go)
- [existing all-command matcher behavior](../../../../../pkg/processors/actualizers/types_test.go)
- [synchronous projector execution path](../../../../../pkg/processors/actualizers/impl.go)
- [asynchronous projector execution path](../../../../../pkg/processors/actualizers/async.go)

## Functional design

- [x] create: [prod/apps/vsql-projectors.feature](../../../../specs/prod/apps/vsql-projectors.feature)
  - Feature Specification for declaring projectors that execute after all commands while preserving command-specific trigger behavior

## Construction

- [x] update: [parser/impl_test.go](../../../../../pkg/parser/impl_test.go)
  - add: parser and application-definition tests for the new declaration syntax
  - verify: all-command declarations build an execute event with a command-type filter for asynchronous and synchronous projectors
  - verify: existing command-specific declarations continue to build QName filters
  - verify: reject table-action and parameter-based all-command declarations, while scheduled projectors continue to reject an all-command target syntactically

- [x] update: [actualizers/async_test.go](../../../../../pkg/processors/actualizers/async_test.go)
  - add: feature-traceable asynchronous execution coverage for local, imported, and built-in commands and command-specific compatibility
  - preserve exact Gherkin step comments, Scenario Outline example-table rows, placeholder mappings, and scenario identities
  - verify: wait for both independent projector offsets before asserting command-specific exclusion, using a position interval below the pipeline flush interval so filtered-out progress is persisted

- [x] update: [actualizers/impl_test.go](../../../../../pkg/processors/actualizers/impl_test.go)
  - add: feature-traceable synchronous execution coverage for the all-command trigger
  - reuse the established synchronous actualizer fixtures and scenario step wording

- [x] update: [parser/types.go](../../../../../pkg/parser/types.go)
  - extend the projector trigger syntax model to distinguish `ON ALL COMMANDS` from explicit command names

- [x] update: [parser/errors.go](../../../../../pkg/parser/errors.go)
  - add: semantic validation error for all-command targets used outside plain `AFTER EXECUTE`

- [x] update: [parser/impl_analyse.go](../../../../../pkg/parser/impl_analyse.go)
  - reject: table-action and `EXECUTE WITH PARAM` triggers targeting all commands

- [x] update: [parser/impl_build.go](../../../../../pkg/parser/impl_build.go)
  - build `AFTER EXECUTE ON ALL COMMANDS` as an execute projector event filtered by command type
  - retain existing QName- and type-based construction for all other projector trigger forms

## Quick start

Declare an asynchronous projector that runs after every successful command:

```vsql
PROJECTOR MyProjector AFTER EXECUTE ON ALL COMMANDS;
```

Add `SYNC` before `PROJECTOR` when the projector must run synchronously. Existing command-specific declarations continue to use a command name after `ON`.
