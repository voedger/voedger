---
change_id: 2609231156-refresh-sequences-architecture
type: docs
issue_url: https://untill.atlassian.net/browse/AIR-4980
domains: [prod]
scope: [apps]
---

# Change request: Current sequence architecture documentation

Refs:

- [AIR-4980: voedger: actualize arch-sequences.md and arch2-sequences.md](./issue-AIR-4980.md)

## Why

The sequence architecture documentation combines earlier implementation descriptions with a proposed design, making it difficult to identify how sequences currently work. A description derived from the current code will help readers understand sequence management and partition recovery in the Voedger application platform while preserving access to the historical designs.

## What

- Readers get a complete rewrite of the sequence architecture documentation covered by `arch-sequences.md` and `arch2-sequences.md` in the production apps context, derived from the current source code and explaining sequence generation, persistence, and recovery for applications and workspaces.
- Historical context preserves access to the previous versions through GitHub links pinned to the last commit containing each document before its rewrite, clearly distinguishing historical designs from the current implementation.

## How

Decisions:

- Keep `arch-sequences.md` as the canonical architecture for the Sequences subsystem. Retain `arch2-sequences.md` as a short historical entry pointing to the canonical document and the archived proposal, so existing links remain useful without maintaining two competing architecture descriptions.
- Document the active command-processing path without presenting the former persistent-sequencer proposal as current architecture. Describe sequence trust levels as active storage policy and retain the proposal only as pinned historical material.
- Describe command-processor recovery as the active lifecycle: it runs asynchronously under the service lifetime, rebuilds in-memory workspace state by scanning the full PLog, and reapplies the last event. Keep the proposal's checkpoint-based actualization lifecycle in the historical snapshot.
- Use the current uspecs component, scenario, and cross-cutting concern structure, consistent with the surrounding apps architecture. Give each flow one authoritative description with source links, and use existing tests as supporting evidence for observable behavior. Validate document navigation, source references, and historical targets as the verification for this documentation change.

Assumptions:

- None

Out of scope:

- Recovery performance optimizations and integration of the persistent sequencer into command processing.

References (internal):

- [Apps subsystem boundaries and navigation](../../../../specs/prod/apps/arch.md)
- [Command sequence allocation and recovery lifecycle](../../../../../pkg/processors/command/impl.go)
- [Recovery behavior and service-lifetime tests](../../../../../pkg/processors/command/impl_test.go)
- [Application partition contract without a sequencer accessor](../../../../../pkg/appparts/interface.go)
- [Persistent sequencer allocation, flushing, and actualization](../../../../../pkg/isequencer/impl.go)
- [Sequence storage checkpoint and PLog replay adapter](../../../../../pkg/appparts/internal/seqstorage/impl.go)
- [Active event and record write policy](../../../../../pkg/istructsmem/impl.go)
- [Trust-level and event-reapplication behavior tests](../../../../../pkg/istructsmem/impl_seqtrust_test.go)

References (external):

- [Previous sequence architecture at its last modifying commit](https://github.com/voedger/voedger/blob/4899083997529915bf8397e5df9d7f07594da913/uspecs/specs/prod/apps/arch-sequences.md)
- [Previous sequence proposal at its last modifying commit](https://github.com/voedger/voedger/blob/4899083997529915bf8397e5df9d7f07594da913/uspecs/specs/prod/apps/arch2-sequences.md)

## Technical design

- [x] update: [prod/apps/arch-sequences.md](../../../../specs/prod/apps/arch-sequences.md)
  - rewrite: the canonical document as `Context subsystem architecture: prod/apps/sequences`, using the current uspecs Components, Scenarios, and Cross-cutting concerns structure with relative declaration and implementation links.
  - distinguish: the active command processor, workspace ID generator, and application event storage from the historical persistent-sequencer proposal. Keep diagram relationships faithful to current call sites and do not present the proposed `ISequencer` integration as current behavior.
  - document: per-partition PLog offsets, per-workspace WLog offsets and record IDs, their initial values and reserved/singleton ranges, raw-to-persistent ID mapping, and advancement on synchronized records and ODoc argument trees. Explain the allocation and offset advancement points in the command pipeline.
  - document: asynchronous partition recovery, including authentication before recovery can start, first-request and in-progress 503 responses, a failed-attempt 500 response while retry starts, service-lifetime cancellation, the full PLog scan, last-event reapplication, and partition reset after persistence or synchronous projector failures.
  - document: active sequence trust levels and their configuration path, conditional versus unconditional writes for events and new records, sequence-violation errors, and corrupted-event and reapplication exceptions.
  - capture: concurrency, persistence, security, configuration, error-handling, observability, resource-use, and verification rules in one `Operational invariants` cross-cutting subsection. Link existing command recovery, ID-generation, and trust-level tests as behavioral evidence.
  - add: a `Historical context` section linking both previous documents at commit `4899083997529915bf8397e5df9d7f07594da913`, using the pinned GitHub URLs recorded above; keep superseded proposals and historical performance measurements in those archived versions.

- [x] update: [prod/apps/arch2-sequences.md](../../../../specs/prod/apps/arch2-sequences.md)
  - replace: the old proposal with a short historical entry linking to the canonical sequence architecture for current implementation details.
  - add: a `Historical context` section linking the complete proposal at commit `4899083997529915bf8397e5df9d7f07594da913`; identify it as historical design material, while directing readers to the canonical document for the current integration status.
  - remove: duplicated architecture, obsolete proposed APIs, copied implementation sketches, and stale local references from the current entry while preserving them in the archived version.

- [x] update: [prod/apps/arch.md](../../../../specs/prod/apps/arch.md)
  - update: the Sequences subsystem entry to identify the canonical document as covering active command sequencing and separately described sequencer infrastructure.
  - relabel: the secondary link as historical context so the apps architecture does not present the archived proposal as a second current design.
