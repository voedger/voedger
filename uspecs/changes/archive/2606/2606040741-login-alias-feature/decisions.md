# Decisions: 2605281424-login-alias-feature

## Ambiguity: how the system handles uniqueness when an alias overlaps with another user's login or alias

Decision: An alias must be globally unique across the union of all user logins and aliases; attempts to set a colliding alias are rejected.

- Pros: guarantees unambiguous sign-in; matches typical identity-system invariants; simple mental model for users and operators
- Cons: a desired alias may be unavailable because another account already uses it as a login or alias
- Confidence: high

Alternatives:

1. Alias unique only against other aliases; collisions with logins surfaced at sign-in time
   - Pros: smaller uniqueness scope
   - Cons: introduces sign-in ambiguity that must be resolved at runtime; surprising failure modes
   - Confidence: low
2. No uniqueness constraint; ambiguous sign-in fails with an explicit error
   - Pros: simplest write path; defers conflict resolution to sign-in
   - Cons: lets the system reach an unusable state; poor user experience; hard to recover
   - Confidence: low

## Vagueness: what makes a login alias "active"

Decision: An alias is active for sign-in whenever it is set; clearing the alias is the only way to disable it (there is no separate enabled/disabled state).

- Pros: simplest mental model; one state field per user; matches the "create / update / clean it up" wording in the acceptance criteria; no extra API or UI surface for enable/disable
- Cons: cannot temporarily disable an alias without losing its value; re-enabling later requires retyping
- Confidence: high

Alternatives:

1. Separate set vs. active states, allowing a "set but disabled" alias
   - Pros: allows pausing an alias without losing the value; supports staged rollout per user
   - Cons: adds a second state field and extra management actions; not motivated by the acceptance criteria; risk of operator confusion
   - Confidence: low
2. Alias is automatically active only for a fixed transition window after being set, then expires
   - Pros: models temporary email-change scenarios
   - Cons: introduces time-based behavior nowhere implied by the acceptance criteria; complicates sign-in semantics
   - Confidence: low

## Ambiguity: what the token contains when a user has no alias set

Decision: The authentication token always carries both the login and an alias field; when no alias is set, the alias field is empty.

- Pros: token shape is invariant, so consumers always read both fields; matches the "carries both" wording in `## What`; trivial validation rule ("empty alias means no alias")
- Cons: empty string must be handled consistently by callers and not confused with other sentinel values
- Confidence: high

Alternatives:

1. Alias field is present only when an alias is set; omitted otherwise
   - Pros: distinguishes "no alias" from any sentinel value; smaller tokens for users without an alias
   - Cons: variable token shape complicates consumers and validators; contradicts the "carries both" wording
   - Confidence: medium
2. When no alias is set, the alias field carries the original login (alias defaults to login)
   - Pros: every token has two equally usable identifiers
   - Cons: erases the distinction between real login and alias; surprising semantics; breaks audit logs that rely on the two fields being independent
   - Confidence: low

## Uncertainty: who is authorized to set, update, or clear a user's login alias

Decision: Alias-management operations require a System Principal Token (the `System` role from `sys.vsql`, issued via `payloads.GetSystemPrincipalToken` / `payloads.GetSystemPrincipalTokenApp`). End users and ordinary workspace roles cannot manage aliases directly.

- Pros: matches the existing voedger authorization model for trusted backend operations; ACL checks are skipped under the `System` role, which fits "create/update/clear" being performed by trusted code paths; aligns with the operator-assisted workflow in the parent ticket AIR-3968
- Cons: no self-service alias management for users; callers must hold or be invoked under a System Principal Token, which constrains where this surface can be used
- Confidence: user-provided

Alternatives:

1. User manages their own alias, with administrators also able to manage it on the user's behalf
   - Pros: covers both self-service and operator-assisted scenarios; common pattern in identity products
   - Cons: two entry points to keep consistent; more authorization rules to verify
   - Confidence: high
2. Only a workspace/system administrator role manages the alias (no System Principal Token requirement)
   - Pros: fits the support-ticket origin; simpler than full self-service
   - Cons: still admin-scoped rather than system-scoped; ACL surface different from the existing system-token pattern
   - Confidence: medium
3. Only the user manages their own alias; administrators cannot set it on their behalf
   - Pros: clean ownership model; no admin-impersonation concerns
   - Cons: blocks the operator-assisted workflow that motivated the parent ticket
   - Confidence: low

## Uncertainty: behavior of previously issued tokens when the alias is later updated or cleared

Decision: Previously issued tokens remain valid until their normal expiration; the alias claim is a snapshot at issue time and is not retroactively updated or revoked when the alias changes.

- Pros: matches existing voedger token semantics (claims are a snapshot at issue time, refreshed via `RefreshPrincipalToken`); no per-user revocation infrastructure required; predictable token lifecycle
- Cons: an outdated alias claim can persist in a live token until expiration; consumers must interpret the alias claim as "alias at issue time", not "current alias"
- Confidence: high

Alternatives:

1. Updating or clearing the alias revokes all previously issued tokens for that user
   - Pros: strongest consistency between token contents and current state
   - Cons: requires per-user revocation infrastructure that does not exist today; forces immediate re-sign-in on every alias edit
   - Confidence: low
2. Tokens remain valid, but the alias claim is re-evaluated against current state on each use
   - Pros: no outdated-claim problem
   - Cons: contradicts the stated behavior that the token carries the alias; adds per-request lookups; mismatches existing token semantics
   - Confidence: low

## Inconsistency: `LoginAlias` unique key `(AppName, LoginHash, Alias)` does not enforce the global-uniqueness rule in `## What`

Decision: Hybrid enforcement. Keep `view.registry.LoginIdx` and `ProjectorLoginIdx` unchanged; add a table-level `UNIQUE (AppName, Alias)` on active `LoginAlias` rows for alias-vs-alias uniqueness (handled by the generic uniques mechanism in `pkg/sys/uniques/impl.go`). The internal `c.registry.PutLoginAliasIndex` helper runs in `targetAppWSID = GetPseudoWSID(NullWSID, Alias, clusterID)` and (1) prechecks `LoginIdx` for an existing login with the same string under the same `AppName` (alias-vs-login), then (2) uses `GetRecordIDByUniqueCombination(QNameCDocLoginAlias, {AppName, Alias})` for idempotency (same active source `(SourceAppWSID, CDocLoginID)` pair -> success; different active pair -> reject), then (3) inserts a new `LoginAlias` row or reactivates an existing inactive row for the same alias. The `LoginAlias` back-pointer carries `SourceAppWSID` so sign-in's fallback path (`LoginIdx` miss -> active `LoginAlias` unique lookup -> fetch source `Login` by `CDocLoginID` in `SourceAppWSID`) is navigable.

- Pros: `Login` / `LoginIdx` semantics are untouched, so existing readers (`GetCDocLoginID`, the `SqlQuery` usage in `pkg/vit/utils.go`, etc.) keep their current "name -> primary login id" guarantee; alias uniqueness is a declarative VSQL constraint; clear separation of "logins" vs "aliases" in the schema
- Cons: sign-in adds a miss-fallback lookup (one extra read when the submitted sign-in identifier is an alias); two enforcement mechanisms in one feature (`LoginIdx` read for alias-vs-login, `UNIQUE` for alias-vs-alias)
- Confidence: user-provided

Alternatives:

1. Broaden `ProjectorLoginIdx` to fire `AFTER INSERT ON (Login, LoginAlias)` and write alias rows into `LoginIdx` keyed by `sha256(Alias)`; single precheck covers both halves of the rule and sign-in needs no fallback
   - Pros: one view-key lookup covers both halves of the rule; mirrors the existing `Login` / `LoginIdx` pattern; sign-in path is unchanged
   - Cons: changes `LoginIdx` semantics so existing readers may now resolve alias names to a source login; risk to code that assumes the submitted sign-in identifier equals `Login.LoginHash`
   - Confidence: medium
2. Add a separate parallel view `view.registry.LoginAliasIdx` (sync projector `AFTER INSERT ON LoginAlias`); precheck both `LoginIdx` and `LoginAliasIdx`
   - Pros: keeps the two indexes physically separate; clear "logins vs aliases" boundary
   - Cons: a second view and projector to maintain; two precheck lookups instead of one
   - Confidence: medium
3. Keep `(AppName, LoginHash, Alias)` as declared and weaken `## What` to "unique per source login"
   - Pros: no schema change
   - Cons: loses global uniqueness; reintroduces the sign-in ambiguity already rejected in the first recorded decision in this file
   - Confidence: low

## Gap: `c.registry.CreateLogin` does not validate the new login string against existing aliases

Decision: Promote the four cross-namespace cases into a single shared `#### Sign-in identifier uniqueness rules` subsection and extend `c.registry.CreateLogin` (and transitively `c.registry.CreateEmailLogin`) with a login-vs-alias precheck against active `cdoc.registry.LoginAlias` by unique combination `(AppName, Login)` in `targetAppWSID = GetPseudoWSID(NullWSID, Login, clusterID)`, mirroring the alias-vs-login precheck `c.registry.PutLoginAliasIndex` already does against `view.registry.LoginIdx`. The four rules (login-vs-login, login-vs-alias, alias-vs-login, alias-vs-alias) are stated once and both commands point at the shared subsection.

- Pros: rules are stated once for the whole feature; the two write paths are symmetric and reference the same table; no change to `CreateLogin`'s signature or grantees; new check is one workspace-local read in the same pseudo-WSID the command already routes to
- Cons: `CreateLogin` gains a dependency on `cdoc.registry.LoginAlias` (a new table) -- existing readers of `Login` are unaffected, but the registry now needs `LoginAlias` available before `CreateLogin` will succeed
- Confidence: user-provided

Alternatives:

1. Leave `CreateLogin` unchanged; rely on the `UNIQUE (AppName, Alias)` constraint plus the alias-vs-login precheck in `PutLoginAliasIndex` to keep the namespaces disjoint
   - Pros: smallest change to existing login creation
   - Cons: a new primary login can be created with the same string as an existing alias -- once both rows exist, sign-in resolution against `LoginIdx` would return the primary login and the alias would be silently shadowed; violates the "globally unique across the union of logins and aliases" rule recorded in the first decision in this file
   - Confidence: low
2. Move alias-vs-X and login-vs-X prechecks into a single shared helper (e.g. `assertNameAvailable(state, appName, name, targetAppWSID)`) called from both `createLogin` and `putLoginAliasIndex`
   - Pros: one implementation point; harder to drift between the two commands
   - Cons: implementation-level concern, not a specification one; can be done at coding time without changing the spec; left to the implementer
   - Confidence: medium

## Gap: alias update does not define how the previous alias stops working

Decision: Use `c.registry.InitiateSetLoginAlias` as the public alias-management initiation command. It starts an in-progress alias flow; `aproj.registry.ApplySetLoginAlias` creates, replaces, or clears the single active alias stored on `Login.Alias`. Active `LoginAlias` rows are lookup indexes maintained by internal `PutLoginAliasIndex` and `DeactivateLoginAliasIndex` helpers. Updating from one alias to another first creates the new active alias index, then deactivates the old alias index with `sys.IsActive = false`, then updates `Login.Alias`. Clearing deactivates the old alias index, then writes `Login.Alias = ""`. `q.registry.IssuePrincipalToken` treats `Login.Alias` as authoritative and accepts an alias fallback only when an active `LoginAlias` row dereferences to a `Login` whose `Login.Alias` exactly equals the submitted sign-in identifier, so a prepared new alias cannot work before the final commit and an old alias cannot keep working after deactivation.

- Pros: one public command initiates create/update/clear; replacement semantics are explicit; old alias rows are retained as inactive history; old-index deactivation remains retryable because it happens before the final `Login.Alias` commit
- Cons: update/clear can touch two pseudo workspaces; a prepared new alias index can exist before the source `Login.Alias` commit and is rejected by sign-in validation until the commit succeeds
- Confidence: high

Alternatives:

1. Keep `CreateLoginAlias` as the public command and add a separate `ClearLoginAlias`
   - Pros: each command is mechanically simple
   - Cons: update is still a multi-command workflow for callers; easier to leave the previous alias active by accident
   - Confidence: low
2. Delete old `LoginAlias` rows instead of deactivating them
   - Pros: simpler lookup table with no inactive rows
   - Cons: loses alias history and diverges from the normal `sys.CDoc` lifecycle
   - Confidence: medium
3. Add `UNIQUE(SourceAppWSID, CDocLoginID)` to `LoginAlias` and upsert the row
   - Pros: directly enforces one alias row per user
   - Cons: does not fit alias-based pseudo-WSID routing when the alias value changes; the old and new aliases may live in different workspaces
   - Confidence: low

## Uncertainty: how to source the canonical primary login for `PrincipalPayload.Login` when sign-in is via alias

Decision: Carry the plaintext primary login through the alias index. Add `Login varchar NOT NULL` to `cdoc.registry.LoginAlias`, populated by `c.registry.PutLoginAliasIndex` from the `c.registry.InitiateSetLoginAlias` `Login` argument. On the alias sign-in path, `q.registry.IssuePrincipalToken` sources `PrincipalPayload.Login` from `LoginAlias.Login`; the primary-login path keeps using the submitted login. `cdoc.registry.Login` continues to store only `LoginHash`.

- Pros: leaves `cdoc.registry.Login` and its existing readers untouched; localizes the change to the alias feature; `InitiateSetLoginAlias` already receives the plaintext primary login, so no new input is needed; no dependence on `Login` storing plaintext
- Cons: only the alias path is served by the new field (the primary path still relies on the submitted string equalling the primary login); duplicates the primary login into the alias index across pseudo-workspaces; one more field to keep consistent if a login string could ever change (logins are immutable today, so low risk)
- Confidence: user-provided

Alternatives:

1. Store the plaintext primary login on `cdoc.registry.Login` and source `PrincipalPayload.Login` from the resolved `Login` record on both paths
   - Pros: single source of truth; symmetric for primary and alias sign-in; consistent with the plaintext `Alias` field this change already adds; future-proof for other consumers
   - Cons: changes the `Login` CDoc schema; stores the plaintext login (often an email) on the record; existing rows need backfill or empty-login handling
   - Confidence: high
2. Redefine `PrincipalPayload.Login` to carry the submitted sign-in identifier (drop the canonical-primary-login requirement)
   - Pros: zero schema change; matches current code; smallest diff
   - Cons: breaks the semantic that `Login` is the stable primary identifier; `Login` and `Alias` can be equal on alias sign-in; weakens audit and consumer guarantees; conflicts with the recorded "token carries both" decision
   - Confidence: low

## Uncertainty: how `q.registry.IssuePrincipalToken` verifies credentials when sign-in is routed to the alias pseudo-workspace

Decision: Keep credential verification inside a single `IssuePrincipalToken` invocation in `pseudoWSID(alias)`. On the alias path, read the `LoginAlias` row locally (it already holds `(SourceAppWSID, CDocLoginID)`), then read the source `Login` record from `SourceAppWSID` with one federation call to the existing `q.sys.GetCDoc(ID = CDocLoginID)`, authorized by a System Principal Token. `Login` is a `sys.CDoc`, so `q.sys.GetCDoc` returns its full non-system field set -- `PwdHash`, `WSID`, `WSError`, `Alias`, `SubjectKind`, and `GlobalRoles` -- so the password check, profile-readiness check, alias validation, and token issue all run locally, mirroring the primary-login path. No new registry query is introduced. The salted `PwdHash` travels back transiently in the federation response and is never persisted outside the source workspace.

- Pros: reuses the existing `q.sys.GetCDoc` read surface, so no new query type and no re-dispatch/re-entrancy; one federation hop; password check stays identical to the primary path; `PwdHash` is never persisted into the alias index
- Cons: the salted `PwdHash` crosses the workspace boundary transiently in the federation response (wider exposure than re-dispatching the plaintext into the source workspace, but a salted hash is the lesser concern and it is internal, system-token-authorized); `q.sys.GetCDoc` returns the whole `Login` CDoc, so the read is broader than the fields strictly required
- Confidence: user-provided

Alternatives:

1. Re-dispatch `IssuePrincipalToken` to `pseudoWSID(primary login)` so the password check runs where `PwdHash` lives
   - Pros: `PwdHash` never leaves the source workspace; reuses the whole existing flow at the source WS
   - Cons: re-entrant call to the same handler at a different WSID; moves the plaintext password into the source workspace instead
   - Confidence: medium
2. Add a narrower, registry-owned `ReadLoginByID` query returning only the needed fields (and never re-using `q.sys.GetCDoc`)
   - Pros: returns exactly the required fields; explicit, narrowly-scoped read surface
   - Cons: introduces a new query type that duplicates a CDoc read `q.sys.GetCDoc` already provides
   - Confidence: medium
3. Denormalize `PwdHash` into `cdoc.registry.LoginAlias` so the alias WS checks locally with no federation call
   - Pros: no cross-workspace hop at sign-in time
   - Cons: persists the secret across pseudo-workspaces; widest exposure; consistency burden if the hash rotates
   - Confidence: low

## Vagueness: how `c.registry.InitiateSetLoginAlias` resolves the source `Login` record before updating alias state

Decision: `InitiateSetLoginAlias` must run in `pseudoWSID(Login)` and resolve the source `Login` locally through `view.registry.LoginIdx` by `(AppName, Login)`. The command rejects when no active source `Login` is found or `Login.AliasInProc != 0`, then starts the in-progress alias flow without writing `Login.Alias`; `aproj.registry.ApplySetLoginAlias` performs the create/update/clear state transition. The resolved record supplies `SourceAppWSID` and `CDocLoginID` for the alias-index projector flow.

- Pros: matches the existing primary-login routing model; keeps the public command limited to resolving the source login and starting the in-progress flow; makes `SourceAppWSID` simply the current workspace when the projector writes `LoginAlias`
- Cons: callers must route the command by primary login, not alias
- Confidence: high

Alternatives:

1. Allow `InitiateSetLoginAlias` to run from another workspace and federation-call into `pseudoWSID(Login)` to update the source `Login`
   - Pros: more flexible caller placement; caller may not need to know the source pseudo-workspace
   - Cons: adds another cross-workspace write path before the alias-index write; more failure states around partial updates and retries
   - Confidence: medium
2. Change `InitiateSetLoginAliasParams` to identify the source by `SourceAppWSID` and `CDocLoginID` instead of primary `Login`
   - Pros: direct lookup of the exact `Login` CDoc; avoids ambiguity if login string handling changes later
   - Cons: exposes internal storage identifiers in the public management command; loses the operator-friendly "set alias for login X" contract; still needs a way to populate `PrincipalPayload.Login`
   - Confidence: low

## Uncertainty: whether `aproj.registry.ApplySetLoginAlias` should return success or failure after recording `AliasError`

Decision: `aproj.registry.ApplySetLoginAlias` distinguishes finalized business-rule rejections from retryable source-commit failures. Alias-index command errors, such as alias collisions, are recorded on the source `Login` as `AliasError = <reason>` with `AliasInProc = 0`, and the projector completes so operators can observe the terminal rejection. Source `Login` commit errors still return projector failure after a best-effort `AliasError` write, preserving retry semantics for transient federation or storage failures. To keep source state authoritative, old alias index deactivation happens before the final source `Login.Alias` commit.

- Pros: deterministic business failures do not replay indefinitely; `AliasError` exposes terminal rejections; transient source write errors can still self-heal through projector reapply; a prepared new alias index remains inert until `Login.Alias` is committed
- Cons: alias-index infrastructure errors are treated the same as alias-index business rejections unless error classification is added later
- Confidence: high

Alternatives:

1. Return projector failure on every error
   - Pros: every failure remains visible to projector retry; transient alias-index or cleanup errors can self-heal through reapply
   - Cons: deterministic business failures, such as alias already taken, can be retried repeatedly until corrected; operators may see `AliasInProc` remain set during replay
   - Confidence: medium
2. Record `AliasError` as terminal state and return success for every error
   - Pros: avoids replay loops for permanent business errors; leaves the failed alias operation visible through `AliasError`
   - Cons: transient alias-index or cleanup errors are not retried automatically; callers or operators must initiate a new alias operation after correction
   - Confidence: low
3. Split alias-index errors into deterministic business failures and retryable infrastructure failures
   - Pros: transient failures can self-heal through projector replay while permanent business errors can stop as terminal state
   - Cons: requires reliable error classification; mistakes can cause replay loops or lost retries
   - Confidence: low

## Inconsistency: alias-management command is named `InitiateSetLoginAlias` in some places and `SetLoginAlias` in others

Decision: Standardize the public alias-management initiation command on `c.registry.InitiateSetLoginAlias` with parameter type `InitiateSetLoginAliasParams`. The name reflects that the command resolves the source `Login`, checks `AliasInProc`, and starts the in-progress alias flow; `aproj.registry.ApplySetLoginAlias` performs the actual alias state transition.

- Pros: command name matches its responsibility; avoids implying that the command directly writes `Login.Alias`; aligns the VSQL declaration, grant, projector trigger, and decision records
- Cons: longer API name than `SetLoginAlias`
- Confidence: user-provided

Alternatives:

1. Standardize on `c.registry.SetLoginAlias`
   - Pros: shorter public API name; matches the original command declaration and several older decision entries
   - Cons: name suggests the command directly sets the alias, which previously caused ambiguity around command versus projector responsibility
   - Confidence: medium
2. Use both names intentionally: public command `SetLoginAlias`, projector/event wording `InitiateSetLoginAlias`
   - Pros: preserves the shorter public API name while documenting the implementation flow
   - Cons: confusing unless there is a separate event or command named `InitiateSetLoginAlias`; the spec would continue to look contradictory
   - Confidence: low

## Inconsistency: `CDocLoginID` is workspace-scoped, but `PutLoginAliasIndex` idempotency compared it without `SourceAppWSID`

Decision: Look up the `LoginAlias` row by unique combination `(AppName, Alias)`, then compare the full `(SourceAppWSID, CDocLoginID)` pair with the command args when an active row exists. Equal pair means success without writing; different pair means reject because the alias is already taken by another source login. If the row exists but is inactive, reactivate it and refresh the source back-pointer and login snapshot from the command args.

- Pros: matches the complete `LoginAlias` back-pointer; prevents two source workspaces with the same record id from being treated as the same alias owner; aligns `PutLoginAliasIndex` with `DeactivateLoginAliasIndex`; allows a cleared alias value to be reused even when its inactive unique row remains addressable
- Cons: revises the earlier simplification that compared only `CDocLoginID`; inactive row reuse makes `PutLoginAliasIndex` an upsert-like command rather than insert-only
- Confidence: high

Alternatives:

1. Compare only `CDocLoginID`
   - Pros: simpler comparison
   - Cons: relies on `CDocLoginID` being sufficient to identify the source login even though it is documented as a record id in `SourceAppWSID`
   - Confidence: low
2. Add a globally unique source-login identifier field and compare that instead
   - Pros: gives a single stable identity value for alias ownership
   - Cons: adds schema/API surface not otherwise needed; duplicates the existing `(SourceAppWSID, CDocLoginID)` identity
   - Confidence: low

## Inconsistency: token alias contract conflicts with `RefreshPrincipalToken` being out of scope

Decision: Include `q.sys.RefreshPrincipalToken` in scope for the alias claim contract. A refreshed principal token preserves the existing `PrincipalPayload` identity fields, including `Alias`; the alias value is copied from the input token payload and refresh does not re-read current registry alias state.

- Pros: keeps the token shape invariant after refresh; matches the `PrincipalPayload` extension; aligns with existing authn behavior that refresh preserves authn identity fields
- Cons: requires updating refresh-token behavior and tests even though alias management itself remains registry-side
- Confidence: high

Alternatives:

1. Keep `RefreshPrincipalToken` out of scope, but explicitly state that refreshed tokens may omit or preserve `Alias` until a later change
   - Pros: smallest scope for this change request
   - Cons: weakens the "token always carries both login and alias" rule; leaves consumers with a variable token contract
   - Confidence: low
2. Make refreshed tokens re-read current alias state
   - Pros: refreshed tokens reflect latest alias state
   - Cons: contradicts snapshot semantics; adds registry lookups to refresh; makes refresh behavior diverge from token issue semantics
   - Confidence: low

## Uncertainty: alias format and case-sensitivity are deferred, but alias sign-in needs deterministic matching

Decision: Reuse the existing primary-login identifier rules for aliases. Alias values are validated with the same format rules as primary logins and are matched as exact, case-sensitive strings; no alias-specific normalization such as lowercasing or trimming is applied before routing, hashing, uniqueness checks, storage, or sign-in lookup.

- Pros: one sign-in identifier contract; alias routing, uniqueness, hashing, and sign-in matching use the same exact-string behavior as `Login`; avoids new normalization rules
- Cons: aliases are limited to the current login format, including lowercase-only letters
- Confidence: high

Alternatives:

1. Add alias-specific normalization, such as lowercasing before uniqueness and sign-in lookup
   - Pros: friendlier for email-like aliases where users may type different casing
   - Cons: diverges from existing login hashing/routing; cross-namespace uniqueness becomes harder because primary logins are currently exact-string hashed
   - Confidence: medium
2. Keep format and case-sensitivity deferred, with aliases matched as exact raw strings for now
   - Pros: smallest spec change
   - Cons: leaves alias validation underspecified; callers may create aliases that do not behave like normal sign-in identifiers
   - Confidence: low
