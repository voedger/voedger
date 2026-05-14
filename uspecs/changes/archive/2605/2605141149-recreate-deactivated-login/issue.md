# AIR-3915: voedger: make possible to create login again if it was deactivated

- **Key**: AIR-3915
- **Type**: Sub-task
- **Status**: In Progress
- **Assignee**: d.gribanov@dev.untill.com
- **URL**: https://untill.atlassian.net/browse/AIR-3915

## Why

- Deactivate login
- `IssuePrincipalToken` returns "login or password is incorrect" according to [AIR-3892](https://untill.atlassian.net/browse/AIR-3892)
- Try to create the login again with the same name → "login exists already" error that violates GDPR statements

## What

Create the login again with the same name → new bare login is created, the same name.

## How

Rewrite the existing key in `LoginIdx`.
