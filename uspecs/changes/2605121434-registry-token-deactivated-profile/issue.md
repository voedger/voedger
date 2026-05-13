# AIR-3892: registry: consider user profile status on IssuePrincipalToken

- **Type**: Sub-task
- **Status**: In Progress
- **Assignee**: d.gribanov@dev.untill.com
- **URL**: <https://untill.atlassian.net/browse/AIR-3892>

## Why

if a user profile (login, not cdoc.registry.Login) is deactivated via c.sys.InitiateDeactivateWorkspace then IssuePrincipalToken now returns 410 gone status code with message workspace status is not active. That is wrong according to GDPR requirements

## What

check on q.registry.IssuePrincipalToken: if the profile has been deactivated then return login does not exists like if it it is missing

## How

if the profile has been deactivated then cdoc.registry.Login.IsActive set to false (already done)
