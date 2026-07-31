---
change_id: 2607301440-replace-active-member-invite
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4631
domains: [prod]
scope: [auth]
---

# Change request: Replace accepted invitations without revoking membership

## Why

An active workspace member can accept another invitation addressed to the member's canonical login or active login alias, causing multiple invitations to share one membership. The acceptance flow must retire the previous accepted invitation before transferring control of the membership to the new invitation.

## What

Symptom: Accepting an additional invitation can leave multiple joined invitations controlling the same active membership, so cancelling any one of them can revoke the user's workspace access.

```text
active member accepts another valid invitation
      |
      v
invitation acceptance attaches the new invitation to the existing membership without retiring the previous invitation  <-- fault: multiple invitations control one membership
      |
      v
either accepted invitation can subsequently cancel the membership
      |
      v
cancelling one invitation revokes the user's existing workspace access   (symptom)
```

Corrected behavior: When an active member accepts another valid invitation, the system retires the uniquely identified previous controlling invitation without interrupting membership, accepts the new invitation as the sole active controller, and applies its roles; if the previous invitation cannot be identified uniquely, the response is `409 Conflict` and nothing changes, while first-time joins and former-member rejoins remain unchanged.

When multiple valid invitation acceptances are queued before projection, they are applied in PLog order and the last successfully applied invitation becomes the sole controller of the membership.

## How

`InitiateJoinWorkspace` validates the invitation and authenticated recipient and records the canonical membership identity, but does not retire another invitation or write final invitation state. `ApplyInviteEvents` remains the sole writer of final invitation states and performs replacement in PLog order as an idempotent transition, preserving an active Subject and JoinedWorkspace while retiring the previous controlling invitation.

`Subject.InviteEmail` may be empty on Subjects created before the field was introduced. Every successful join, rejoin, or replacement writes the accepted `Invite.Email` to this field; when populated, it is the authoritative single link from a Subject to its controlling accepted invitation. `ApplyInviteEvents` resolves an existing value through `InviteIndexView` when replacing an invitation, retires the resolved invitation without deactivating the membership, and replaces `Subject.InviteEmail` with the newly accepted invitation's email.

At most one invitation is actively linked to a Subject: it is in `Joined` state and its `Email` equals `Subject.InviteEmail`. Retired invitations keep their `SubjectID` as historical provenance, but their non-`Joined` state means they no longer control roles or membership cancellation.

When membership cancellation or leave makes a Subject inactive, its `InviteEmail` remains as provenance for the last controlling invitation. An inactive Subject has no active invitation link regardless of that stored value; a successful rejoin overwrites `InviteEmail` with the newly accepted invitation's email.

For an existing active Subject whose `InviteEmail` is absent or cannot be resolved, `InitiateJoinWorkspace` uses the authenticated canonical login and active login alias to look up candidates through `InviteIndexView`, excluding the invitation currently being accepted. Replacement proceeds only when exactly one candidate is joined and references the same Subject; `ApplyInviteEvents` repeats the authoritative check before applying the transition. Zero or multiple matches cause `409 Conflict` with no state changes.

For each accepted join event, `ApplyInviteEvents` loads the invitation referenced by that event. A different queued invitation remains eligible while its state is `Invited`; applying it retires the Subject's current controlling invitation, changes the event's invitation to `Joined`, updates the Subject roles and `InviteEmail`, and leaves the membership active. These workspace-local changes are written together. Replaying an event for the same invitation is a no-op because that invitation is no longer in a valid source state for joining. Cross-workspace JoinedWorkspace role updates remain idempotent.

## Functional design

- [x] update: [auth/invites.feature](../../../../specs/prod/auth/invites.feature)
  - update: "Alias-addressed invitation reuses an existing canonical membership" scenario -> cover both canonical-to-alias and alias-to-canonical replacement while keeping the existing membership active
  - add: assertions that the previous accepted invitation is retired, the new invitation becomes the sole active controller, and the new invitation's roles take effect
  - add: scenario verifying that cancelling or updating a retired invitation is rejected without affecting the active membership
  - add: scenario verifying `409 Conflict` with no state changes when an active member's previous controlling invitation cannot be identified uniquely

## Construction

### Tests

- [x] update: [sys/it/impl_invites_feature_test.go](../../../../../pkg/sys/it/impl_invites_feature_test.go)
  - extend the Subject test projection with `InviteEmail`
  - update the canonical-to-alias replacement scenario to assert that the previous invitation becomes `Cancelled` while retaining its historical `SubjectID`, the new invitation becomes `Joined`, the membership and JoinedWorkspace stay active, the new roles take effect, and `Subject.InviteEmail` identifies the new invitation
  - add the reciprocal alias-to-canonical replacement scenario and verify that cancelling or updating the retired invitation is rejected without affecting membership
  - cover an existing active Subject with empty, missing, non-`Joined`, or wrong-Subject `InviteEmail`: accept only when canonical-login/active-alias fallback finds exactly one different joined invitation for the same Subject, otherwise return `409 Conflict` with no changes
  - verify that cancelling the current invitation and member-initiated leave both target the invitation identified by `Subject.InviteEmail`, retain that field on the inactive Subject, and allow rejoin to overwrite it

- [x] update: [invite/impl_applyinviteevents_test.go](../../../../../pkg/sys/invite/impl_applyinviteevents_test.go)
  - cover sequential join events for different invitations and assert deterministic PLog-order replacement with the last event as the sole active controller
  - verify that replaying a join event for an invitation already in `Joined` state is a no-op
  - verify that replacement writes the previous Invite, current Invite, and Subject changes together and does not deactivate Subject or JoinedWorkspace

### Schema and shared helpers

- [x] update: [sys/sys.vsql](../../../../../pkg/sys/sys.vsql)
  - add optional `InviteEmail varchar` to `TABLE Subject`
  - document that it stores the exact email of the current or last controlling accepted invitation and is empty on records created before the field exists

- [x] update: [invite/consts.go](../../../../../pkg/sys/invite/consts.go)
  - add the `InviteEmail` field-name constant used by Subject reads and writes

- [x] update: [invite/errors.go](../../../../../pkg/sys/invite/errors.go)
  - add the conflict error returned when an active Subject's controlling invitation cannot be identified uniquely

- [x] update: [invite/utils.go](../../../../../pkg/sys/invite/utils.go)
  - add exact invitation lookup through `InviteIndexView`
  - add controlling-invitation resolution from `Subject.InviteEmail`, requiring the resolved invitation to be `Joined` and linked to the same Subject
  - add legacy fallback resolution by authenticated canonical login and active alias, excluding the invitation being accepted and requiring exactly one joined invitation for the same Subject

### Command validation

- [x] update: [invite/impl_initiatejoinworkspace.go](../../../../../pkg/sys/invite/impl_initiatejoinworkspace.go)
  - look up Subject by authenticated canonical login after recipient validation
  - when the Subject is active, pre-validate the controlling invitation through `Subject.InviteEmail` or the legacy fallback and return `409 Conflict` without emitting changes when resolution is not unique
  - preserve first-join and inactive-Subject rejoin behavior and continue recording canonical `Invite.ActualLogin`

- [x] update: [invite/impl_initiateleaveworkspace.go](../../../../../pkg/sys/invite/impl_initiateleaveworkspace.go)
  - resolve the active controlling invitation through `Subject.InviteEmail` instead of assuming its address is the canonical login
  - use the same legacy fallback for existing Subjects with no resolvable `InviteEmail`

### Projector

- [x] update: [invite/impl_applyinviteevents.go](../../../../../pkg/sys/invite/impl_applyinviteevents.go)
  - write `Subject.InviteEmail` on first join, rejoin, and replacement
  - for an active Subject, repeat authoritative controlling-invitation resolution and retire the previous invitation without invoking membership deactivation
  - apply previous Invite `Cancelled` while preserving its historical `SubjectID`, current Invite `Joined`, Subject roles, current Invite SubjectID linkage, and `Subject.InviteEmail` as one workspace-local CUD
  - retain `Subject.InviteEmail` when cancel or leave deactivates membership
  - preserve idempotent JoinedWorkspace role updates and PLog-order last-wins behavior
