# voedger: support alias email in reset-password by email flow

- URL: https://untill.atlassian.net/browse/AIR-4376
- ID: AIR-4376
- State: In Progress
- Author: Maksim Geraskin
- Assignees: Maksim Geraskin
- Labels: none

## Description

Research and design change_id: `2606301014-reset-password-by-alias`

### Context

The reset-password-by-email flow resolves the account only via the primary login index (`registry.LoginIdx`), never via the alias index. An alias-email holder cannot reset the password of the account that owns the alias. The selected approach is verifier decoupling with client re-routing: all cross-workspace logic is resolved server-side during steps 1 and 2, and the final command runs locally in the canonical workspace with no new commands and no cross-workspace writes.

### Step 1 - q.registry.InitiateResetPasswordByEmail

Runs at `pseudoWSID(alias)`.

* If `GetCDocLogin(email)` misses on `LoginIdx`, read `cdoc.registry.LoginAlias` from local state keyed by `(AppName, Alias=email)`
* Derive `CanonicalPseudoWSID = pseudoWSID(LoginAlias.Login)` and federation-read the canonical Login at `LoginAlias.SourceAppWSID` (via `q.sys.GetCDoc`) to obtain `ProfileWSID`
* Add a new field `CanonicalPseudoWSID int64` to `InitiateResetPasswordByEmailResult`. For a primary login it equals `pseudoWSID(email)` (always returned)

### Step 2 - q.registry.IssueVerifiedValueTokenForResetPassword

Runs at `pseudoWSID(alias)`. After the sys verifier confirms the code, re-read `LoginAlias` from local state:

* If alias: re-issue the registry verified-value token with `Value=LoginAlias.Login` (canonical login), keeping Entity, Field and VerificationKind unchanged
* If primary login: return the sys verifier token as-is (existing behaviour)

This is verifier decoupling: the code is proved against the alias email, but the signed token carries the canonical login. The alias -> canonical binding is established server-side; the client never asserts it. Do not accept a client-supplied LoginID (ownership bypass + locality wall - see research).

### Step 3 - c.registry.ResetPasswordByEmail

No backend change. The command runs at `CanonicalPseudoWSID` (client routes it there), where `GetCDocLogin(canonicalLogin)` hits `LoginIdx` locally and the PwdHash write lands in the same workspace.

### Specs

* Add TD scenario _Client resets password by verified alias email_ to authn--td.md (draft in the research doc)
* Add Gherkin scenarios to authn.feature: the alias happy-path plus a rejection for a previous or cleared alias (drafts in the research doc)

### Acceptance criteria

* A reset initiated with an alias email delivers the code to the alias inbox and updates the password of the account that owns the alias
* The primary-login reset flow continues to work unchanged
* A reset attempt using a previously assigned or cleared alias is rejected
* Verification remains the sole proof of ownership; resolving an alias never lets an unverified value alter another account's password
* Integration test in pkg/sys/it/impl_resetpassword_test.go covers the alias-to-canonical flow

Companion frontend subtask: AIR-4372 (client routes step 3 to the returned CanonicalPseudoWSID).

---

Co-authored by [Augment Code](https://www.augmentcode.com/?utm_source=atlassian&utm_medium=jira_issue&utm_campaign=jira)
