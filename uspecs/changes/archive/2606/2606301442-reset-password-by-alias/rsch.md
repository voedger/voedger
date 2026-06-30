# Research: reset password by login alias

## Problem

The reset-by-verified-email flow resolves the account only via the primary login index (`view.registry.LoginIdx`), never via the alias index. An alias-email holder cannot reset the password of the account that owns the alias.

The structural reason is a workspace locality constraint. `LoginAlias` lives at `pseudoWSID(alias)` (the workspace the alias hash maps to). The canonical `Login` CDoc and its `LoginIdx` entry live at `pseudoWSID(canonicalLogin)`. These are different workspaces. No single command can simultaneously read from one and write to the other using only local state and intents.

## Selected approach: verifier decoupling with client re-routing

The three-step flow (`InitiateResetPasswordByEmail` -> `IssueVerifiedValueTokenForResetPassword` -> `ResetPasswordByEmail`) is extended so that all cross-workspace logic is resolved server-side during the first two steps, and the final command runs locally in the canonical workspace without any new commands or federation writes.

### Proposed TD scenario

To add to `authn--td.md` as a sibling of `#### Client resets password by verified email`, following the `#### User signs in with active alias` convention (the alias path is a discrete scenario, not a branch folded into the primary flow).

#### Client resets password by verified alias email

```text
@Client
  -> [/q.registry.InitiateResetPasswordByEmail/]: AppName, alias, Language
      -> [(registry.LoginIdx)]: GetLoginHash(alias); primary-login miss
      -> [(registry.LoginAlias)]: (AppName, Alias=alias); active alias hit
          - canonicalLogin = LoginAlias.Login
          - SourceAppWSID = LoginAlias.SourceAppWSID
          - CanonicalPseudoWSID = pseudoWSID(canonicalLogin)
      -> [/q.sys.GetCDoc/]: read canonical [(registry.Login)] at SourceAppWSID; ProfileWSID = Login.WSID
      -> [/q.sys.InitiateEmailVerification/]: at loginApp/ProfileWSID; alias, TargetWSID=ProfileWSID, ForRegistry=true
          - code emailed to the alias inbox; VerificationToken Value = alias
      -> out: InitiateResetPasswordByEmailResult
          - VerificationToken, ProfileWSID
          - CanonicalPseudoWSID

  -> [/q.registry.IssueVerifiedValueTokenForResetPassword/]: AppName, VerificationToken, code, ProfileWSID
      -> [/q.sys.IssueVerifiedValueToken/]: at loginApp/ProfileWSID; confirm code; VerifiedValueToken Value = alias
      -> [(registry.LoginAlias)]: (AppName, Alias=alias); active alias hit
      -> re-issue VerifiedValueToken under sys/registry with Value = canonicalLogin
      -> out: VerifiedValueToken (Value = canonicalLogin)

  -> [/c.registry.ResetPasswordByEmail/]: at CanonicalPseudoWSID; VerifiedValueToken, NewPwd (UNLOGGED), AppName
      -> [(registry.Login)]: login = token.Value (= canonicalLogin); write PwdHash
      -> out: 200 OK
```

### Proposed Gherkin scenario

To add to `authn.feature` alongside `Scenario: Client resets password by verified email`, mirroring `Scenario: User signs in with active alias`:

```gherkin
    Scenario: Client resets password by verified alias email
      Given a user login exists with an active login alias
      When Client initiates password reset using the alias email
      And Client verifies the reset code sent to the alias email
      And Client resets the password with the verified value token
      Then Client can sign in with the new password
```

A reset attempt using a previously assigned or cleared alias must be rejected, mirroring the sign-in rejection scenarios:

```gherkin
    Scenario: Password reset rejects a previous or cleared alias
      Given a user login exists
      And System updated or cleared the user's login alias
      When Client initiates password reset using the previous alias
      Then the response status is "400 Bad Request"
```

### Step 1 - `q.registry.InitiateResetPasswordByEmail` at `pseudoWSID(alias)`

Current behaviour: calls `GetCDocLogin(email)` via `LoginIdx` keyed by `GetLoginHash(email)` at the current workspace; fails for an alias because `LoginIdx` has no entry for the alias hash.

Change: if `GetCDocLogin` returns not-found, read `cdoc.registry.LoginAlias` from local state (it lives at `pseudoWSID(alias)`, which is exactly the current workspace) keyed by `(AppName, Alias=email)`. If a matching active row exists:

- `LoginAlias.Login` is the plaintext canonical login
- `LoginAlias.SourceAppWSID` is the appWSID of the canonical `Login` CDoc
- `LoginAlias.CDocLoginID` is the record id of that CDoc
- Derive `CanonicalPseudoWSID = pseudoWSID(LoginAlias.Login)`
- Federation-read the canonical `Login` CDoc at `SourceAppWSID` (using `q.sys.GetCDoc`) to obtain its `profileWSID`
- Call `q.sys.InitiateEmailVerification` with `Email=alias-email` (code delivered to alias inbox) and `TargetWSID=profileWSID` (same as today); the `VerificationToken` payload binds `Value=alias-email`

Return `VerificationToken`, `ProfileWSID` (existing fields), and `CanonicalPseudoWSID` (new field). For a primary login `CanonicalPseudoWSID = pseudoWSID(email)` (same workspace the call is already running in).

### Step 2 - `q.registry.IssueVerifiedValueTokenForResetPassword` at `pseudoWSID(alias)`

The client keeps routing this step to `pseudoWSID(alias)` (same workspace as step 1), so `LoginAlias` is still local.

Current behaviour: forwards `VerificationToken + code` to `q.sys.IssueVerifiedValueToken` at `loginApp/profileWSID`; the sys verifier confirms the code and re-issues the payload as a `sys/registry` app token with `Value=alias-email`.

Change: after the sys verifier call confirms the code, check whether `alias-email` is an alias by re-reading `LoginAlias` from local state:

- If alias: re-issue the registry verified-value token with `Value=LoginAlias.Login` (the canonical login) using `args.State.AppStructs().AppTokens()`, keeping all other payload fields (Entity, Field, VerificationKind) unchanged
- If primary login: return the token from the sys verifier as-is (existing behaviour)

This is verifier decoupling: the code is delivered to and proved against the alias-email, but the signed verified value token carries the canonical login. The binding alias -> canonical login is established server-side by reading `LoginAlias`; the client never asserts it.

### Step 3 - client re-routing

The client always routes `c.registry.ResetPasswordByEmail` to the returned `CanonicalPseudoWSID`. For a primary login this is the same workspace as steps 1-2; for an alias it is the canonical workspace.

### Step 4 - `c.registry.ResetPasswordByEmail` at `CanonicalPseudoWSID` (no changes)

```go
email := args.ArgumentUnloggedObject.AsString(field_Email)  // verified value = canonicalLogin
login := email
return ChangePassword(login, args.State, args.Intents, args.WSID, appName, newPwd)
```

`GetCDocLogin(canonicalLogin)` hits `LoginIdx` locally at `CanonicalPseudoWSID`. The `PwdHash` write lands in the same workspace. No federation, no cross-workspace write, no new command.

## Alternative approach: cross-workspace write from the alias workspace

The reset command runs at `pseudoWSID(alias)`, validates the verified-email token locally, resolves alias to the canonical login via local `LoginAlias`, then issues a system-authorized federated command to write the `PwdHash` in the canonical workspace (new internal command, mirroring `c.registry.PutLoginAliasIndex`).

- Pros: client contract and verifier untouched; no step-2 re-issuance
- Cons: adds a new internal command; wires `federation` and `itokens` into the reset command exec; the write is not local to the command's workspace

## Rejected: a password-bearing `q.registry.ResetPasswordByEmailEx`

Wrapping the flow in an `Ex` query that accepts the new password and forwards it is rejected:

- A query cannot write the `Login` CDoc - a CUD via `Intents` is command-only - so it must forward to a command anyway, adding a layer without removing the command
- The password would travel as a query argument; on API v2 a query is GET-only with args in the URL query string (a POST to a v2 query route is dispatched to the command processor), so a public v2 route would expose the password in URLs and access logs
- The `UNLOGGED` guarantee that keeps a secret out of durable logs exists only for commands; queries have no equivalent

It is acceptable only as a secret-free orchestrator (alias resolution and workspace selection); the password-bearing step must stay a command.

### Why a plain `LoginID` parameter would not be safe

Returning `CanonicalPseudoWSID` and `CDocLoginID` from step 1 is fine (mild pre-auth disclosure). Accepting `LoginID` back from the client as a trusted parameter to `ResetPasswordByEmail` is not, for two reasons:

- Ownership bypass: any attacker who verifies a code for any email they control can then pass a victim's `LoginID` to change the victim's password
- Locality wall: the command at `CanonicalPseudoWSID` cannot read `LoginAlias` to re-validate the claimed mapping, because `LoginAlias` lives at `pseudoWSID(alias)`, a different workspace; a command can only read from its own workspace's state

The alias-to-canonical binding must be encoded in the cryptographically signed verified value token, not in a client-supplied parameter.