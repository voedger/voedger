# Decisions: Derive `auth` context architecture

## Inconsistency: `change.md` lists four `auth` scope areas but only three subsystem architecture chapters

Decision: Add a fourth Context Subsystem Architecture for token management covering issue, refresh, validation, and the principal and verified-value payload contracts

- Pros: one-stop reference for token shape, lifetime, and validation; mirrors `domain.md`'s four-way split exactly; payload contract is a real cross-cutting concern between authn and authz
- Cons: more files to maintain; risks duplicating material that will already appear in the authn (issue/refresh) and authz (validation) chapters
- Confidence: user-provided

Alternatives:

1. Drop "token lifecycle" from the overview list; document the split explicitly (token issue/refresh in the authentication subsystem, token validation in the authorization subsystem)
   - Pros: KISS — three chapters match the natural enforcement-vs-issuance boundary already present in the codebase (`pkg/registry` issues; `pkg/iauthnz` validates); avoids a thin chapter that mostly cross-references the other two
   - Cons: readers scanning for "token management" have to look in two places; the JWT payload contract has no single home
   - Confidence: high
2. Drop "token lifecycle" from the overview list AND drop "principal token validation" from the authorization bullet, treating tokens as an internal authn implementation detail referenced only from the authn chapter
   - Pros: smallest scope; clearest single-owner model
   - Cons: misrepresents reality — `pkg/iauthnz` validates tokens on every request as part of authorization, not authentication; would force authz to silently depend on authn internals
   - Confidence: low

## Ambiguity: how the existing `auth/arch.md` content is handled when it is "replaced"

Decision: Derive all chapters from scratch from the current codebase across the entire repository; the new `arch.md` is a summary linking to subsystem chapters and replaces the current `arch.md` at the same path; treat the existing `auth/arch.md`, `authn--td.md`, `authn.feature`, and `invites--td.md` as reference material consulted during derivation to extract key architectural points the codebase alone might not surface, but not as authoritative sources, since they may be outdated

- Pros: codebase-grounded chapters cannot inherit stale wording from the existing docs; reference material is still mined so key historical points are not lost; the end state's authority on architecture is the code itself
- Cons: largest review surface — each chapter is written from scratch; reviewers must verify both code coverage and that nothing important from the reference docs was missed; derivation effort is highest of the considered options
- Confidence: user-provided

Alternatives:

1. Migrate the existing `arch.md` material verbatim into the new authentication subsystem chapter (split only between the new `arch.md` overview and the authn chapter), with token-related sections moved to the token-management chapter; no wording changes beyond the split
   - Pros: minimum review surface; zero risk of behavior-level wording drift; lets reviewers verify the change is purely a restructure plus three new chapters
   - Cons: leaves the authn chapter slightly long and partially redundant with `authn--td.md` (which is the deeper feature-level technical design); inherits any staleness already present in `arch.md`
   - Confidence: high
2. Rewrite the authentication chapter as a concise architecture-level summary and treat `authn--td.md` and `authn.feature` as the source of truth for the detailed scenarios; cross-reference rather than duplicate
   - Pros: cleaner separation between architecture (arch-\*) and feature TD layers per uspecs-td conventions; smaller authn chapter
   - Cons: larger review surface; reviewers must verify the rewrite preserves the architectural claims of today's `arch.md`; risk of unintended wording drift; depends on TD documents being accurate
   - Confidence: medium
3. Leave the existing `arch.md` in place as the authentication subsystem chapter (rename it to the authn chapter file name with no content change) and add a fresh `arch.md` overview plus the three other subsystem chapters alongside it
   - Pros: smallest possible diff for the authn material — pure rename plus token-section excision; one less file rewrite
   - Cons: `arch.md` rename leaves stale anchor links from any external doc that points at `auth/arch.md#…`; mixing rename with new files makes the diff harder to read; inherits any staleness already present in `arch.md`
   - Confidence: medium

## Ambiguity: whether profile workspace lifecycle is inside the auth chapter scope or a cross-context reference

Decision: Authentication chapter covers only the profile workspace readiness gate visible to sign-in; profile workspace lifecycle stays owned by the `apps` context; additionally, the authentication chapter must explicitly document two login-lifecycle behaviors observable to clients — (a) when a profile workspace is deactivated, the deactivation propagates to `[(registry.Login)].IsActive=false` via `c.sys.OnChildWorkspaceDeactivated` (`pkg/sys/workspace/impl_deactivate.go`); the deactivated login is treated as a missing login by `GetCDocLogin` (`pkg/registry/utils.go` `IsActive` check returns `(nil, false, nil)` and logs "is deactivated, treating as missing login"), so subsequent `[q.registry.IssuePrincipalToken]` calls return the shared `errLoginOrPasswordIsIncorrect` response (HTTP 401, "login or password is incorrect") — the same response used for missing logins and wrong passwords to prevent enumeration; (b) re-creating a login with the same name creates a new login with a fresh profile workspace, and the previously deactivated login and its profile workspace remain in storage but become unreachable

- Pros: keeps the auth/apps boundary clean and matches the current `[[Profile workspace lifecycle]]` cross-context marker; surfaces two non-obvious sign-in failure modes that integrators routinely hit; the architecture chapter, not just deeper TDs, carries these client-visible facts
- Cons: edge-case behavior claims are typically the domain of feature TDs (`authn--td.md`) or feature scenarios, not architecture chapters; risks the architecture chapter accumulating similar specific facts over time
- Confidence: user-provided

Alternatives:

1. Authentication chapter covers the full profile workspace lifecycle (creation, readiness, profile fields write-back), pulling that material into the auth context
   - Pros: end-to-end login-to-sign-in story in one place; no cross-context dependency for the most common reader question
   - Cons: duplicates / contradicts material that belongs to the `apps` context; widens this change's blast radius to touch apps-context architecture; risks contradicting future updates to apps workspace specs
   - Confidence: low
2. Drop "profile workspace readiness" from the authentication chapter scope entirely; treat readiness as a runtime concern of the sign-in scenario and reference apps for everything workspace-related
   - Pros: smallest authn chapter scope; cleanest single-owner model
   - Cons: hides an externally observable behavior of sign-in (sign-in fails until readiness); readers lose a key architectural fact from the authn chapter
   - Confidence: medium

## Ambiguity: authorization chapter's "role resolution from VSQL" wording does not cover invite-granted roles

Decision: Expand the authorization chapter to cover runtime ACL evaluation and effective-roles computation from four role sources — invite-granted roles, roles read from `cdoc.sys.Subject` records in the request workspace, roles carried by the principal token (per-app `Roles` and cross-workspace `GlobalRoles`), and ACL-engine-emitted contextual roles based on request context (`role.sys.ProfileOwner`, `role.sys.WorkspaceOwner`, `role.sys.WorkspaceDevice`, `role.sys.System`) plus VSQL-declared role inheritance expansion (e.g., `ProfileOwner` implies `WorkspaceOwner`), all confirmed in `pkg/iauthnzimpl/impl.go`; VSQL schema parsing of role/grant declarations stays owned by the `apps` context, and the management of `cdoc.sys.Subject` / `cdoc.sys.JoinedWorkspace` records stays owned by the workspace membership chapter

- Pros: captures the unconditional contextual-role emission that `pkg/iauthnzimpl` performs on every request — a key architectural fact integrators routinely miss; documents the VSQL role-inheritance chain that determines whether a permission grant actually applies; the four-source model maps 1:1 to how the code composes principals
- Cons: broadest chapter scope of the considered options; describing role-inheritance expansion requires referencing VSQL syntax that is owned by the `apps` context, increasing the risk of duplicating role-inheritance grammar specs
- Confidence: user-provided

Alternatives:

1. Three sources only (invite-granted, `cdoc.sys.Subject`, principal token) without enumerating ACL-engine-emitted contextual roles or inheritance expansion
   - Pros: smaller chapter scope; avoids touching VSQL inheritance grammar at all
   - Cons: hides the contextual emission (`ProfileOwner`, `WorkspaceOwner`, etc.) that happens on every request and the role-inheritance expansion that turns those into the actual ACL-matched roles
   - Confidence: medium
2. Re-scope to runtime ACL evaluation only, listing role sources as a black-box input but not enumerating them
   - Pros: tightest chapter scope; no cross-chapter coupling on role shape
   - Cons: hides the four-source composition that is the core architectural fact of `iauthnzimpl`
   - Confidence: low
3. Expand further to also cover VSQL schema parsing of role and grant declarations end-to-end
   - Pros: single chapter answers all "who can do what" questions
   - Cons: pulls VSQL ACL parsing material into authz, overlapping the `apps` context architecture
   - Confidence: low

## Inconsistency: workspace membership chapter scope (line 34) omits the subjects doc that the authz chapter (line 27) delegates to it

Decision: Rewrite both lines to drop explicit QNames (`cdoc.sys.Subject`, `cdoc.sys.JoinedWorkspace`) and use conceptual wording instead — "subjects doc" and "joined-workspace records"; line 27 becomes "the management of the subjects doc and joined-workspace records stays owned by the workspace membership chapter"; line 30 becomes "roles read from the subjects doc in the request workspace"; line 34 becomes "invite lifecycle, the subjects doc, joined-workspace records, role updates, and member removal"; role QNames in line 32 stay since they are conceptual role identifiers, not records

- Pros: scope statements stay at the architectural-concept level instead of binding the change request to specific VSQL QNames that may evolve; consistent terminology across all three lines; the omission flagged by the inconsistency is closed because "subjects doc" now appears in both line 27 and line 34
- Cons: readers who grep for `cdoc.sys.Subject` in `change.md` will no longer find it; the conceptual term "subjects doc" is not yet defined in `domain.md` and will need a one-line glossary entry in the membership chapter when it is written
- Confidence: user-provided

Alternatives:

1. Rewrite line 34 to explicitly name both record types and clarify the lifecycle stages, keeping line 27 unchanged
   - Pros: terminology matches line 27 verbatim; reader can map each delegated responsibility to a concrete chapter section
   - Cons: pins the change request to specific QNames; longest bullet in the file
   - Confidence: high
2. Rewrite line 34 minimally to add `cdoc.sys.Subject` alongside joined-workspace records, leaving line 27 unchanged
   - Pros: smallest diff to fix the omission
   - Cons: pins both lines to specific QNames; leaves "role updates" still ambiguous
   - Confidence: high
3. Rewrite line 27 instead to drop the record-name delegation, deferring all wording to line 34
   - Pros: tightens line 27 and removes the cross-bullet coupling
   - Cons: weakens the boundary statement at the place where readers will look for it
   - Confidence: medium

## Ambiguity: file paths for the four new Context Subsystem Architecture chapters are not specified

Decision: Add explicit file paths to each of the four subsystem bullets, using the `arch-{subsystem}.md` convention (mirroring `uspecs/specs/prod/apps/`) with short identifiers — `uspecs/specs/prod/auth/arch-authn.md`, `arch-authz.md`, `arch-tokens.md`, `arch-membership.md`

- Pros: matches the apps convention verbatim; short identifiers stay readable in markdown link text and file lists; eliminates implementer ambiguity about both convention and identifier choice
- Cons: identifier shorthands (`authn`, `authz`, `tokens`, `membership`) diverge from the longer `domain.md` wording ("authentication", "authorization", "token management", "workspace membership")
- Confidence: medium

Alternatives:

1. Add explicit file paths using full-word names (`arch-authentication.md`, `arch-authorization.md`, `arch-token-management.md`, `arch-workspace-membership.md`)
   - Pros: file names match the chapter titles and `domain.md` wording word-for-word
   - Cons: longer paths in link references and CLI; diverges from apps's shorter `arch-vvm-orch.md` style
   - Confidence: medium
2. State the convention once without listing the four exact paths
   - Pros: smallest diff
   - Cons: leaves the four identifiers undecided, just relocating the ambiguity
   - Confidence: low

## Ambiguity: how to handle potential staleness in `authn--td.md`, `authn.feature`, and `invites--td.md` after the new arch chapters land

Decision: Expand the scope of this change to update/rewrite the three existing TD files (`auth/authn--td.md`, `auth/authn.feature`, `auth/invites--td.md`) in lockstep with the new arch chapters, reconciling every divergence found during derivation in this change so no stale or contradictory material is left behind; each TD stays at its current path and keeps its scope (feature/technical design, not architecture)

- Pros: no stale material remains after the change merges; readers cannot land on contradictory information depending on which file they open; the derivation work that surfaces the divergence happens at the same time as the reconciliation, avoiding context loss between two separate changes
- Cons: significantly widens the change's blast radius; TD revision requires the `uspecs-fd` / `uspecs-td` skills in addition to the arch derivation skills; risks the change growing too large to review in one pass
- Confidence: user-provided

Alternatives:

1. Add a one-line "See also" pointer at the top of each of the three TD files; list any contradictions found in a follow-up section of `change.md` (or a separate follow-up issue) rather than fixing them in this change
   - Pros: zero risk of scope creep on TD content; readers get a clear "newer reference exists" signal; contradictions are surfaced rather than silently propagated
   - Cons: introduces a tiny content edit to files this change otherwise leaves alone; contradictions remain in storage until separate change requests address them
   - Confidence: high
2. Leave the three TD files completely untouched and log any divergence found in `change.md` under a "Follow-ups" section, to be addressed by separate change requests later
   - Pros: smallest possible footprint on existing files; cleanest separation between "derive arch" and "reconcile TDs"
   - Cons: leaves TDs as silent contradictions with no in-file pointer to the newer material; readers may continue citing stale TDs indefinitely
   - Confidence: medium

## Inconsistency: `## Why` does not motivate three of the deliverables in `## What`

Decision: Expand `## Why` with three additional sentences so every `## What` deliverable is traceable to a stated motivation — (i) the absence of a Context Architecture overview that ties the four subsystems together, (ii) the absence of any arch coverage for principal token issue/refresh (not just validation), (iii) the risk of stale TDs diverging from the new arch chapters as derivation surfaces contradictions, requiring lockstep reconciliation

- Pros: every What deliverable is traceable to a stated Why; reviewers see one justification per scope item; the four-subsystem split in What is anchored in the gap list in Why; locks in the three previous clarification decisions (overview rewrite, token lifecycle scope, TD lockstep) at the motivation level
- Cons: lengthens `## Why` from one paragraph to four sentences; risks Why duplicating What if the wording drifts
- Confidence: user-provided

Alternatives:

1. Narrow `## What` instead of expanding `## Why` — drop the context overview rewrite, drop token issue/refresh from the tokens chapter, drop the TD lockstep deliverable
   - Pros: tightest scope; Why and What stay aligned at the smallest possible footprint
   - Cons: reverses three previous decisions in this clarification session; leaves the auth context without an overview, the tokens chapter without lifecycle, and the TDs unreconciled
   - Confidence: low
2. Leave `## Why` as a high-level motivation and accept that some What deliverables are structural/hygiene work that does not need a per-item Why entry
   - Pros: smallest diff
   - Cons: violates the implicit traceability between Why and What; reviewers have to infer the rationale for the unmotivated deliverables
   - Confidence: medium

## Inconsistency: token management chapter wrongly describes the verified-value payload as "shared by the authentication and authorization subsystems"

Decision: Drop the verified-value payload from the token management chapter scope and move it to the authentication chapter as the verifier sub-flow (issue and consume verified-value tokens for email verification and password reset); the tokens chapter keeps only the principal payload contract (which is genuinely shared between authn issuer and authz validator); `domain.md`'s listing of "Verified Value Token" as an auth-context vocabulary term stays unchanged — verified-value remains an auth concept, just owned by the authn subsystem instead of the tokens subsystem; the authn exclusion list is narrowed from "token issue/refresh" to "principal token issue/refresh" so that verified-value token issuance is not excluded from the authn chapter

- Pros: removes the incorrect "shared by authn and authz" claim; aligns chapter ownership with actual code consumers (`pkg/registry/impl_resetpassword.go` and the sign-up flow are the only verified-value consumers, and both are authn flows); keeps the tokens chapter focused on the one payload that is genuinely cross-subsystem
- Cons: the tokens chapter no longer covers all auth-context token types in one place, so a reader looking for "all token contracts" has to also open the authn chapter; introduces an explicit exclusion note in the tokens bullet
- Confidence: user-provided

Alternatives:

1. Split the two payload contracts in the bullet, naming the correct consumer set for each
   - Pros: matches actual consumer relationships in code; makes cross-subsystem contracts explicit
   - Cons: longest bullet variant; pairs two payloads with very different lifecycles inside a single chapter
   - Confidence: high
2. Keep both payloads in the tokens chapter but drop the "shared by…" qualifier from the bullet
   - Pros: minimal fix to the inconsistency without expanding the bullet
   - Cons: loses the architectural fact (which payload is shared with what) at the level where a reader scanning `change.md` would benefit from it most
   - Confidence: medium

## Ambiguity: `arch.md` overview chapter scope does not list the cross-cutting concepts that need a single home

Decision: Add a `## What` bullet that names the four cross-cutting auth-context concepts to be introduced once in `arch.md` and cross-linked from the subsystem chapters that produce or consume them — Principal Token (authn/tokens producer → authz consumer), the subjects doc (membership producer → authz consumer), the login record (authn producer → tokens consumer), and the `pkg/iauthnz` apps→auth enforcement boundary (used at every enforcement point in the apps context processors); per-subsystem chapters reference the shared definition rather than restating it

- Pros: matches the apps precedent (`arch.md` line 24: "Introduce the app partitions engine once in `arch.md` as a shared component"); eliminates duplication risk across the four subsystem chapters; gives readers a single landing page for the cross-cutting vocabulary; all four concepts are code-verified producer-consumer pairs
- Cons: lengthens `## What` by one bullet; commits the implementer to a specific shared-concepts list before derivation has surfaced what the right set actually is
- Confidence: user-provided

Alternatives:

1. Add a softer bullet that names the constraint without listing the concepts ("introduce any cross-cutting auth-context concepts once in `arch.md` and cross-link them from the subsystem chapters")
   - Pros: smaller commitment up front; lets derivation determine the exact list
   - Cons: leaves the implementer with full discretion over what counts as "cross-cutting", which may produce inconsistent results across reviewers
   - Confidence: medium
2. Leave the overview chapter scope as-is and let the implementer decide what goes in `arch.md` vs subsystem chapters during derivation
   - Pros: smallest diff to `change.md`
   - Cons: diverges from the apps precedent; risks duplication across multiple chapters with no single source of truth
   - Confidence: low

## Ambiguity: Login Alias feature is not explicitly placed in any subsystem chapter scope

Decision: Expand the authentication chapter scope to explicitly cover login-alias set/update/clear and sign-in by alias, and expand the token management chapter scope to call out the alias snapshot carried by the principal payload contract at token-issue time and its immunity to subsequent alias changes; the `authn.feature` alias scenarios (set/replace/clear under System Principal Token, sign-in by original login while alias is active, sign-in by active alias, rejection of previous or cleared aliases, alias snapshot retention in existing principal tokens) drive the derivation in both chapters

- Pros: makes the Login Alias feature a first-class concern with a clear home in two chapters rather than an implicit detail; matches the existing `authn.feature` coverage so derivation will not omit it; keeps the alias-as-data (authn write/read) separate from alias-as-token-payload (tokens) following the same producer-consumer split used elsewhere
- Cons: lengthens two `## What` bullets; pairs alias management with login creation in the same chapter scope even though alias is set via System Principal Token (a privileged path), which may warrant its own sub-section during derivation
- Confidence: user-provided

Alternatives:

1. Put Login Alias entirely in the authentication chapter and only reference the alias snapshot from the tokens chapter without listing it as token-payload concern
   - Pros: single owner for the feature; shortest change to `## What`
   - Cons: hides the fact that the principal payload carries an alias snapshot with specific immutability semantics, which is a token-contract concern that survives alias changes
   - Confidence: medium
2. Leave both chapter scopes as-is and rely on derivation to pick up the alias scenarios from `authn.feature` without explicit mention in `## What`
   - Pros: smallest diff
   - Cons: relies on the implementer to remember a non-trivial feature not named in scope; risks splitting alias coverage inconsistently between chapters
   - Confidence: low

## Inconsistency: `arch-tokens.md` and `arch-authz.md` disagree about whether API tokens emit the `AuthenticatedUser` role

Decision: Fix `arch-tokens.md` to match `arch-authz.md` and `pkg/iauthnzimpl/impl.go`; the `IsAPIToken=true` bullet now states that principal composition emits `AuthenticatedUser` and `Roles` filtered to `RequestWSID`, and omits `Host`, `GlobalRoles`, the `WorkspaceOwner`/`ProfileOwner`/`WorkspaceDevice` derivations, the `System` short-circuit, and the `[(cdoc.sys.Subject)]` read

- Pros: aligns the token chapter with the authoritative source (`pkg/iauthnzimpl/impl.go` lines 53-75) and with the authz chapter; smallest, most local fix; preserves the at-a-glance composition summary in the token chapter
- Cons: none material
- Confidence: user-provided

Alternatives:

1. Fix `arch-authz.md` to match `arch-tokens.md`: state `AuthenticatedUser` is also omitted for API tokens
   - Pros: would not require touching the tokens chapter
   - Cons: contradicts the codebase; would make the spec wrong about observable behavior; ACL rules that gate API-token endpoints with `AuthenticatedUser` would be described as inaccessible even though they are reachable
   - Confidence: low
2. Reframe the bullet in `arch-tokens.md` to defer all composition semantics to `arch-authz.md` and only state that `IsAPIToken` is a payload flag the token subsystem owns
   - Pros: cleanest DRY split — token chapter owns the payload shape, authz chapter owns the rules driven by the flag
   - Cons: removes a useful at-a-glance summary from the token chapter; readers must jump to `arch-authz.md` for any composition detail
   - Confidence: medium

## Inconsistency: `arch-authz.md` Scenarios overview omits the `Host` principal for anonymous and non-API-token branches

Decision: Add `Host` to both Scenarios overview bullets — the anonymous bullet now lists Guest, Anonymous, Host, and matching Subject rows; the user/device bullet now lists AuthenticatedUser, Host, all four role sources, and the workspace-derived roles

- Pros: aligns the overview with `pkg/iauthnzimpl/impl.go` lines 20-27 (deferred `Host` emission whenever `!IsAPIToken`) and with the chapter's own detailed Scenarios block (lines 108, 124); readers scanning the overview no longer miss a principal that ACL rules may key off
- Cons: lengthens two bullets by one principal each
- Confidence: user-provided

Alternatives:

1. Drop the explicit principal enumeration from the overview entirely and have each bullet say "emits the principal set defined in the Scenarios block below"
   - Pros: removes the redundancy that caused the divergence; single source of truth
   - Cons: overview becomes less informative; readers have to scroll to the detailed scenario for any principal-set question
   - Confidence: medium
2. Add a single up-front line in the Scenarios overview: "`Host` is emitted for every branch except `IsAPIToken=true`", and keep the per-branch bullets focused on the branch-specific principals
   - Pros: factors out the cross-branch invariant; both bullets stay short
   - Cons: introduces a fourth structural element to the overview; readers of a single bullet still might miss the cross-cutting line
   - Confidence: medium

## Inconsistency: the "four sources of effective roles" enumeration differs across `change.md`, `arch.md`, and `arch-authz.md`

Decision: Adopt a hybrid origin-perspective vocabulary across all three docs — invite-granted, token-carried, request-context, anonymous-grants — and rewrite `arch-authz.md`'s `Role sources` section to use the same labels with per-source origin + composition notes; VSQL role inheritance expansion is documented as an ACL-engine-evaluation concern, not as a composition source; `change.md` ## What and `arch.md` Scenarios overview are updated to match

- Pros: single vocabulary across the three docs; ends the double-count of "invite-granted vs subjects doc" (subjects doc is the persistence boundary for invite-granted, not a separate source); ends the over-split of "principal token" into per-app vs global at the top level (both are token-carried, snapshotted at the same point); origin perspective is easier for product readers to scan; per-source notes inside each bullet preserve the per-app vs global filtering rules that the previous `Token roles` / `Global roles` split made visible
- Cons: requires coordinated edits to `change.md`, `arch.md`, and `arch-authz.md` (plus the layers diagram and the Authenticate-request scenario block in arch-authz.md); supersedes the earlier "four role sources" wording in `decisions.md` (the entry "Authorization chapter's 'role resolution from VSQL' wording does not cover invite-granted roles" above)
- Confidence: user-provided

Alternatives:

1. Align `change.md` and `arch.md` to the previous arch-authz.md model (Token / Global / Subject / Contextual) instead of changing all three to a new vocabulary
   - Pros: aligns spec deliverable wording with code-grounded model; smaller scope (arch-authz.md unchanged)
   - Cons: keeps the per-app/global split as top-level sources; less accessible for product readers; "Subject" label hides that the rows are invite-produced
   - Confidence: high
2. Align `arch-authz.md` to the change.md model (invite-granted / subjects doc / principal token / contextual)
   - Pros: smallest diff to change.md
   - Cons: makes arch-authz.md less precise; readers must mentally reconcile "Subject roles" with "Invite-granted roles" since they are the same rows
   - Confidence: low

## Inconsistency: `change.md` ## What and `decisions.md` describe the deactivated-profile error path inaccurately

Decision: Adopt hybrid wording across `change.md` ## What and `decisions.md` — preserve the "treated as a missing login" narrative from the original ticket while explicitly naming the actual user-facing response (`errLoginOrPasswordIsIncorrect`, HTTP 401) and the correct propagation path (`InitiateDeactivateWorkspace` → `c.sys.OnChildWorkspaceDeactivated` → `cdoc.registry.Login.IsActive=false` → `GetCDocLogin` treats as missing → `IssuePrincipalToken` returns the shared error); the earlier entry "Ambiguity: whether profile workspace lifecycle is inside the auth chapter scope" above is rewritten in place to reflect this; `arch-authn.md` already states the correct error and remains unchanged

- Pros: preserves the original ticket framing ("treated as missing") that makes the security intent clear; replaces the inaccurate `errLoginDoesNotExist` citation with the actual `errLoginOrPasswordIsIncorrect` response that `IssuePrincipalToken` returns; documents the full propagation chain so readers can verify the claim end-to-end; aligns `change.md` ## What with `arch-authn.md` line 73 which already had it right; preserves the enumeration-prevention property as a stated security claim
- Cons: longer wording in both files (each line grows from a single clause to a propagation chain); two entries in `decisions.md` now describe the same login-lifecycle behavior (the original ambiguity entry plus this inconsistency entry)
- Confidence: user-provided

Alternatives:

1. Fix `change.md` ## What line 25 and `decisions.md` line 47 to cite `errLoginOrPasswordIsIncorrect` directly without keeping the "treated as missing" framing
   - Pros: shortest precise wording; single error name everywhere
   - Cons: loses the conceptual reason (the login is intentionally indistinguishable from a missing one); reader has to infer the enumeration-prevention motivation
   - Confidence: high
2. Fix `arch-authn.md` to match the original `change.md`/`decisions.md` wording (`errLoginDoesNotExist`)
   - Pros: would not require touching change.md or decisions.md
   - Cons: contradicts `pkg/registry/impl_issueprincipaltoken.go` lines 65-66; would document an error that the sign-in endpoint never returns
   - Confidence: low
