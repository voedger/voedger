---
registered_at: 2026-04-29T16:19:50Z
change_id: 2604291619-match-vsql-code-funcs
baseline: 4f210eb8ab63962037c3dc2e58456093f27056a2
issue_url: https://untill.atlassian.net/browse/AIR-3778
---

# Change request: Match vsql and code funcs definitions

## Why

A stateless function (projector, command, query) registered in code via `IStatelessResources` but not declared in the deployed app's `IAppDef` (built from the bundled vsql baseline) causes voedger to crash on a `nil` dereference deep inside the actualizer factory instead of failing fast with an actionable error. This was reproduced by `Test_FiscalCloud_Vit` panicking at `pkg/processors/actualizers/provide.go:43` after the runtime gained the new `sys.ApplyInviteEvents` projector while the bundled fiscalcloud baseline still lacked it. The same mismatch class can hit any app whose baseline lags the runtime, and is not specific to projectors - commands and queries are equally exposed.

See [issue.md](issue.md) for details.

## What

Enforce the invariant that vsql definitions and code-registered stateless funcs are in sync, and surface any drift as a clear startup error instead of a runtime panic:

- For every projector / command / query require a matching entry in the deployed app's `IAppDef`
- Symmetric check: every vsql-declared func that is expected to have a code implementation must have one; a vsql entry with no code registration must fail the same way (catches the inverse drift)
- Failure mode is a single composite error listing every offending QName together with its kind (projector / command / query) and the direction of the mismatch (in code but not in vsql / in vsql but not in code), produced once during app deployment - well before the first actualizer / command processor / query processor is built
- The dead-on-nil callsite at `pkg/processors/actualizers/provide.go` becomes safe by construction; equivalent unsafe lookups in command and query processor wiring (if any) are reviewed and aligned with the same invariant

Tests:

- Unit test that synthesizes an app with a projector / command / query absent from a stub `IAppDef` and asserts the validator returns the composite error without panicking
- Unit test for the inverse: an `IAppDef` that declares a func with no code registration
- Positive test that a fully-aligned pair passes
