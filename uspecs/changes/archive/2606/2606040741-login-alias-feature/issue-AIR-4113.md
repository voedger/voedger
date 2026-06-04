# change-email: voedger: login aliase feature

- URL: https://untill.atlassian.net/browse/AIR-4113
- ID: AIR-4113
- State: in-progress
- Author: Maxim Geraskin
- Labels: none
- Assignees: Maxim Geraskin

## Description

Summary

It should be possible to manage a login alias for each user. When the alias is active, users should be able to sign in with either the original login or the alias.

Acceptance criteria

- It should be possible to create one login alias per user
- It should be possible to update the login alias value or clean it up
- If the alias is active, it should be possible to sign in using either the original login or the alias
- The token contains both login and alias
