# voedger: principal token Login reflects alias used at sign-in

- URL: https://untill.atlassian.net/browse/AIR-4270
- ID: AIR-4270
- State: in-progress
- Author: Unknown user name (712020:89894cda-e18f-4f7e-989d-8540111610e5)
- Labels: none
- Parent: AIR-3968 (Grandhotel Bad Pyrmont: Change email address)
- Assignees: Unknown user name (712020:89894cda-e18f-4f7e-989d-8540111610e5)

## Why

The frontend displays the Login value taken from the principal token. When a user signs in with their alias, the token currently carries the canonical primary login, so the UI shows an identifier different from the one the user just typed. This confuses the user, who expects to see the alias they signed in with.

Carrying the alias (when set) in the token's Login field lets the frontend display the identifier the user recognizes, while the canonical primary login remains available separately for backend identity needs such as subject-based role resolution.

## What

Change the principal token identity contract so the token's Login reflects the identifier the user is known by, and the canonical primary login is carried separately.

- The principal token's Login field carries the user's active alias when one exists, and falls back to the canonical primary login when no alias is set
- The canonical primary login is always carried in a separate field (CanonicalLogin), replacing the current Alias field in the token payload
- This applies on both sign-in paths (sign-in by primary login and sign-in by alias) and is preserved across token refresh as a snapshot taken at issue time
- During workspace authorization both fields, Login and CanonicalLogin, are evaluated for subject-based role resolution, so a user's workspace roles are matched whether the subject was registered under the alias or the canonical primary login

Behavior when no alias is set is unchanged from the caller's perspective: Login equals the primary login.
