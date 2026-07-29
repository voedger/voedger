# Voedger: accept workspace invitations addressed to login aliases

- URL: https://untill.atlassian.net/browse/AIR-4612
- ID: AIR-4612
- State: To Do
- Author: Maksim Geraskin
- Labels: none
- Assignees: Denis Gribanov

## Problem

An invitation sent to an active login alias passes normal authentication and `Subjects` role resolution, but `c.sys.InitiateJoinWorkspace` independently compares the canonical request-subject name with `Invite.Email`. For an alias-addressed invitation those values differ, so the command returns HTTP 400 once the portal allows the request through.

Generic `Subjects` authorization already resolves both canonical `Login` and token `Alias`; it is not part of this change.

## Scope

* Update recipient validation in `pkg/sys/invite/impl_initiatejoinworkspace.go` so `Invite.Email` may match either the authenticated canonical `Login` or authenticated token `Alias`.
* Continue storing the canonical `Login` in `ActualLogin` so joined-workspace and `Subject` identity remains canonical.
* Preserve rejection when neither authenticated identifier matches the invitation recipient.
* Leave `InitiateInvitationByEMail` and generic `Subjects` authorization unchanged.

## Acceptance criteria

* An invitation addressed to the authenticated active alias can pass `InitiateJoinWorkspace`.
* A canonical-login invitation continues to work.
* An invitation for a different identity is rejected.
* `ActualLogin` and any created or updated `Subject.Login` remain canonical.
* Accepting an alias invitation for an already joined canonical account does not create a duplicate `Subject`.
* Automated tests cover canonical, matching-alias, mismatched-identity, and existing-member cases.