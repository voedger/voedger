---
registered_at: 2026-05-14T10:38:43Z
change_id: 2605141038-recreate-deactivated-login
baseline: 4184d623f617f80924fec35b0351252c59686887
issue_url: https://untill.atlassian.net/browse/AIR-3915
archived_at: 2026-05-14T11:49:37Z
---

# Change request: Allow recreating a login that was deactivated

## Why

Once a login is deactivated, `IssuePrincipalToken` returns "login or password is incorrect" (per AIR-3892), but attempting to register the same login again fails with "login exists already". This violates GDPR statements because the user cannot re-register under the same name. See [issue.md](issue.md) for details.

## What

Registering a login again after it was deactivated must succeed:

- Creating a login with a name that previously existed but is now deactivated produces a new bare login with the same name
- Existing key in `LoginIdx` is rewritten to point to the newly created login
