# deduplicate test mocks in stateprovide and storages using coreutils

- URL: https://untill.atlassian.net/browse/AIR-5221
- ID: AIR-5221
- State: in-progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov
- Linked issues: AIR-376, AIR-5213

## Why

`stateprovide` and `storages` duplicate mocks for the same interfaces (e.g. [here](https://github.com/voedger/voedger/blob/5c45e847270a93a7c5c45f7f4026c9d9415aeb6d/pkg/sys/storages/types_test.go#L45) and [here](https://github.com/voedger/voedger/blob/5c45e847270a93a7c5c45f7f4026c9d9415aeb6d/pkg/state/stateprovide/impl_test.go#L80)). These copies increase maintenance effort and can develop inconsistent behavior. Reusing shared mocks will simplify test infrastructure

## What

* Implement missing mocks in `coreutils`: `MockAppStructs`, `MockRecords`, `MockViewRecords`, `MockRecord`, and `MockCUD`
* Replace compatible local mocks with existing or newly implemented `coreutils` mocks
* Adapt shared mocks and test expectations to preserve required behavior, including nil `BuildValue()` results, configurable `Equal()`, and event `Release()` calls
* Keep `mockStorage` outside `coreutils` because moving it there would introduce an import cycle
* Retain stateful storage helpers, map-backed builders, and lightweight benchmark doubles where shared mocks would change their purpose
* Remove superseded mock definitions and verify interface compliance and affected package tests
