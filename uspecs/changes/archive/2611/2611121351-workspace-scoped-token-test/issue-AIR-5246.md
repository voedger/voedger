# Reseller access, voedger: implement a VIT integration test showing how to issue a token scoped to one Workspace

- URL: https://untill.atlassian.net/browse/AIR-5246
- ID: AIR-5246
- State: In Progress⚒️
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov
- Linked PRs/issues: AIR-1808 (parent): Enable Reseller Diagnostic Access to Clients Backoffice (first stage)

## Description

* ceate a foreign workspace
* show that we can not work in that workspace
* issue a token with a role scoped to a that foreign workspace 
* show that we can work in that foreign workspace with this token
* show that we can not work neighter in siblings of that workspace nor in the owner of these workspaces with this token

