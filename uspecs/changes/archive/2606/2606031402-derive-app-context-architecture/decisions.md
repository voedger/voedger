# Decisions: derive-app-context-architecture

## Inconsistency: `change.md` says "App context", but `uspecs/specs/prod/domain.md` and the folder layout name the context `apps`

Decision: Rename "App context" to "`apps` context" throughout `change.md` (title, Why, What)

- Pros: matches `domain.md` and the `uspecs/specs/prod/apps/` folder verbatim; avoids introducing a second name for the same context; aligns with the uspecs convention of using the lowercase identifier in prose
- Cons: change request title becomes slightly less prose-like ("Derive `apps` context architecture")
- Confidence: high

Alternatives:

1. Keep "App context" wording but add a one-line clarification in Why that "App context" refers to the `apps` context in `domain.md`
   - Pros: keeps the title human-readable; preserves the issue's wording
   - Cons: introduces a synonym readers must learn; risks the same drift recurring in the derived specs
   - Confidence: low
2. Rename the context in `domain.md` from `apps` to `App` to match the change request
   - Pros: aligns documentation with the issue title
   - Cons: out-of-scope rename touching many references; breaks existing folder name `apps/` and existing files (`arch-sequences.md`, `arch-vvm-orch.md`, `vsql-*.feature`, `logging--td.md`); not what the issue asks for
   - Confidence: low

## Inconsistency: `change.md` claims the `apps` context "has no architecture specification", but `uspecs/specs/prod/apps/` already contains architecture files

Decision: Add a new Context Architecture (`apps/arch.md`) plus two new Context Subsystem Architectures (`apps/arch-deployment.md` and `apps/arch-processing.md`); leave the existing `arch-sequences.md`, `arch-vvm-orch.md`, `arch2-sequences.md`, and `logging--td.md` untouched and reference them from `arch.md` where relevant

- Pros: matches uspecs-td layering (Context Architecture + Context Subsystem Architecture); minimal blast radius; preserves prior work and lets it be reorganized later if needed; aligns Why ("no overall architecture reference") with What ("split into deployment and processing")
- Cons: temporary overlap between `arch.md` overview and the older `arch-*.md` files until they are reconciled; reader must follow links to get full picture
- Confidence: high

Alternatives:

1. Add only `apps/arch.md` containing both deployment and processing as sections inside one file; do not create separate subsystem files
   - Pros: single document, easier for readers to scan
   - Cons: violates the uspecs-td convention that subsystem architectures live in `arch-{subsystem}.md`; contradicts the explicit "split into deployment subsystem and processing subsystem" wording in What
   - Confidence: low
2. Replace the existing `arch-sequences.md`, `arch-vvm-orch.md`, and `arch2-sequences.md` with the new deployment/processing architecture by folding their content into the new files
   - Pros: leaves a cleaner final layout without overlap
   - Cons: substantially out of scope for a `docs` change; risks losing detail or context; not requested by the issue; high review burden
   - Confidence: low

## Ambiguity: "the app partitions engine" is listed under both the deployment subsystem and the processing subsystem

Decision: Keep the engine in both subsystem files, split by role -- deployment covers deploy and undeploy of app partitions, processing covers invoking extensions on app partitions during request processing; introduce the engine once in `arch.md` as a shared component and link to both subsystem chapters

- Pros: matches the codebase, where the app partitions engine genuinely participates in both phases; gives readers a single conceptual entry point in `arch.md`; keeps each subsystem self-contained; user-provided role split is precise and actionable
- Cons: requires explicit cross-references and a brief shared definition in `arch.md` to avoid duplication drift
- Confidence: user-provided

Alternatives:

1. Place the app partitions engine only in the deployment subsystem; processing references it but does not describe it
   - Pros: avoids duplication; one canonical location
   - Cons: processing pipelines are inseparable from how requests reach the right app partition, so the processing chapter would be incomplete without describing the engine's runtime contract
   - Confidence: medium
2. Place the app partitions engine only in the processing subsystem; deployment references it but does not describe it
   - Pros: avoids duplication; one canonical location
   - Cons: bootstrap and deployment-time concerns (partition allocation, engine startup, descriptor changes) genuinely belong in deployment; placing them under processing misclassifies them
   - Confidence: low
3. Promote the app partitions engine to its own Context Subsystem Architecture (`apps/arch-app-partitions-engine.md`), referenced by both deployment and processing
   - Pros: cleanest separation; no duplication; engine gets a dedicated specification
   - Cons: expands scope beyond the two subsystems the issue asks for; adds a third deliverable; risks fragmenting deployment and processing narratives
   - Confidence: medium

## Ambiguity: "authnz, ACL" in the processing subsystem overlaps with the `auth` context defined in `domain.md`

Decision: Reword to describe `apps` processing as covering "authnz/ACL enforcement points within request pipelines (principal token validation and ACL checks delegated to the `auth` context)"; the `apps` processing subsystem describes where and how the pipeline applies authnz/ACL, while the rules and identity model remain owned by the `auth` context

- Pros: respects the existing context boundary in `domain.md`; aligns with the codebase pattern where processors invoke auth/iauthnz to validate tokens and check ACL; gives clear scope to `arch-processing.md` (enforcement, not policy)
- Cons: slightly longer wording; requires `arch-processing.md` to cross-link the `auth` context
- Confidence: high

Alternatives:

1. Keep "authnz, ACL" verbatim and let `arch-processing.md` cover both enforcement and policy mechanics for the `apps` context
   - Pros: shortest wording in `change.md`
   - Cons: duplicates ownership claimed by the `auth` context; readers cannot tell which context's spec is authoritative; risks contradictions between `apps/arch-processing.md` and `auth/arch.md`
   - Confidence: low
2. Drop "authnz, ACL" from the processing subsystem and place them entirely in the `auth` context architecture (out of scope for this change)
   - Pros: cleanest separation by context
   - Cons: `arch-processing.md` would omit a central concern of every request flow (the pipeline cannot be described without naming its authnz/ACL steps); contradicts the issue body which explicitly lists "authnz, acl"
   - Confidence: low

## Vagueness: "processor pipelines" in the processing subsystem does not enumerate which processors are in scope

Decision: List the in-scope processors explicitly in the What bullet -- Command, Query v1, Query v2, Sync Actualizer, Async Actualizer, Scheduler, BLOB, and N10N

- Pros: matches the codebase under `pkg/processors/*`; gives `arch-processing.md` a concrete table of contents; avoids re-litigating scope later
- Cons: longest bullet in the What section; if a new processor appears later, the change request becomes stale (acceptable for a docs change)
- Confidence: high

Alternatives:

1. Cover only the request-driven processors (Command, Query v1, Query v2, BLOB, N10N) and treat Sync Actualizer, Async Actualizer, and Scheduler as separate "background processors" mentioned briefly with a forward reference
   - Pros: keeps `arch-processing.md` focused on the user-request flow; matches how readers usually approach a processing chapter
   - Cons: actualizers and scheduler are central to event sourcing in Voedger; deferring them risks omitting authnz/ACL/response concerns that differ for non-request processors
   - Confidence: medium
2. Cover only Command and Query processors; everything else (BLOB, N10N, actualizers, scheduler) is out of scope and listed as future architecture work
   - Pros: smallest scope; fastest to deliver
   - Cons: contradicts the issue body which says "all processors pipelines"; leaves the processing chapter incomplete
   - Confidence: low

## Vagueness: "bootstrap" in the deployment subsystem does not say what is being bootstrapped

Decision: Scope "bootstrap" to the VVM startup bootstrap performed by `pkg/btstrp` -- initializing the cluster-app workspace, registering built-in and sidecar apps via `c.cluster.DeployApp`, and deploying their app partitions

- Pros: matches the deployment subsystem's existing focus (vsql DDL, deployment descriptors, app partitions engine -- all driven by the same `c.cluster.DeployApp` path); aligns with the codebase package `pkg/btstrp`; keeps cluster provisioning (`ctool`) in the `routing`/infrastructure space where it belongs
- Cons: explicitly excludes cluster-provisioning bootstrap; readers seeking that topic must look in `ctool`
- Confidence: high

Alternatives:

1. Cover both VVM startup bootstrap (`pkg/btstrp`) and cluster provisioning bootstrap (`ctool`) in `arch-deployment.md`
   - Pros: single chapter for any "bootstrap" question a reader might ask
   - Cons: `ctool` is an infrastructure/provisioning tool that lives outside the `apps` context; mixing it into `apps/arch-deployment.md` blurs context boundaries set by `domain.md`
   - Confidence: low
2. Leave "bootstrap" generic; let `arch-deployment.md` decide later
   - Pros: keeps `change.md` short
   - Cons: defers the vagueness instead of resolving it; the same question will resurface during spec authoring
   - Confidence: low
