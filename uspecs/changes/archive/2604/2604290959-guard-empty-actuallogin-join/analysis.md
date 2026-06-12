# Root cause analysis: empty `Login` key on `sys.ViewSubjectsIdx`

## Symptom

Async actualizer log entries of the form:

```text
level=ERROR
msg="field is empty: view «sys.ViewSubjectsIdx» key string-field «Login»: validate error code 4"
extension=ap.sys.ApplyInviteEvents
evqname=sys.InitiateJoinWorkspace
woffset=6 poffset=7583
wsid=140737488488661
vapp=untill/airs-bp
```

The view-records storage rejects a key whose required string field `Login` is empty. The error surfaces from `pkg/sys/invite/impl_applyinviteevents.go::handleApplyJoinWorkspace` -> `SubjectExistsByLogin(login, s)` -> `GetSubjectIdxViewKeyBuilder(login, s)`, which calls `skb.PutString(Field_Login, login)` with `login == ""`.

## Impact of commit `82dba9f7` (AIR-3704)

PR [#4521](https://github.com/voedger/voedger/pull/4521) refactored the invite flow:

- The five per-command async projectors (`ApplyInvitation`, `ApplyJoinWorkspace`, `ApplyUpdateInviteRoles`, `ApplyCancelAcceptedInvite`, `ApplyLeaveWorkspace`) were replaced with a single `ap.sys.ApplyInviteEvents` projector.
- The legacy projectors are kept as `deprecatedNullProjector` placeholders for backward compatibility (their QNames must keep existing in `provide.go` so previously stored offsets remain valid).
- All invite state transitions and side effects moved out of the commands into the new projector.
- `c.sys.InitiateJoinWorkspace` no longer writes `Field_State = State_ToBeJoined`; the cmd only persists `ActualLogin`, `InviteeProfileWSID`, `SubjectKind`, `Updated`. The state transition to `State_Joined` is performed by the projector.

The new projector dispatches by `event.QName()` and validates the current `cdoc.sys.Invite.State` against `projectorValidStates[cmd]`. For `qNameCmdInitiateJoinWorkspace` this set is `{State_Invited, State_ToBeJoined}`. The projector then loads the *current* `cdoc.sys.Invite` via `loadInviteByID` and reads `ActualLogin` from it.

## Why the new projector replays from offset 0

Async actualizers persist their last-applied PLog/WLog offset per projector QName. When a new projector QName is registered, the platform has no stored offset for it and starts from offset 0, walking forward through every event the projector subscribes to.

`ap.sys.ApplyInviteEvents` is a brand new QName (`qNameAPApplyInviteEvents`). Even though the body of the deprecated projectors is now `deprecatedNullProjector`, their offsets are not transferable to a different QName. Consequently, in every workspace that ever contained invite events, the new projector replays the full history of `sys.InitiateInvitationByEMail`, `sys.InitiateJoinWorkspace`, `sys.InitiateUpdateInviteRoles`, `sys.InitiateCancelAcceptedInvite`, `sys.InitiateLeaveWorkspace`, and `sys.CancelSentInvite` events from WLog offset 0 onwards. The log line `woffset=6` confirms this is a very early historical event being replayed.

Because the projector reads the *current* state of the `cdoc.sys.Invite` (not the state it had when the event was originally produced), a replay of an old event sees today's record and decides what to do based on it.

## Replay is not just "noisy" - it has externally visible side effects

The crash on empty `Login` is one symptom. The deeper problem is that `ap.sys.ApplyInviteEvents` produces real side effects in *other* workspaces and to *external systems*. Replaying historical events re-executes those side effects against the current world. Concretely:

- `handleApplyInvitation` calls `sendEmail(...)` (`pkg/sys/invite/impl_applyinviteevents.go`) - replaying every historical `c.sys.InitiateInvitationByEMail` would re-send invitation emails to the original invitees.
- `handleApplyUpdateInviteRoles` calls `fed.Func(".../c.sys.UpdateJoinedWorkspaceRoles", ...)` against the invitee's profile workspace and `sendEmail(...)` for the role-change notification - replaying re-issues role-update HTTP calls and re-sends emails for changes that may have since been reverted, cancelled, or further updated.
- `handleApplyJoinWorkspace` creates/activates a `cdoc.sys.Subject` and calls `c.sys.CreateJoinedWorkspace` in the invitee's profile workspace - replaying can resurrect subjects whose membership has since been cancelled/left.
- `handleApplyCancelAcceptedInvite` and `handleApplyLeaveWorkspace` deactivate the subject and joined workspace - replaying can flip an active membership back to inactive.

None of these effects are guarded by idempotency against the *historical* intent of the event. The deprecated per-command projectors already produced these effects when the events were first written; running them a second time against today's database state is observable to end users (unwanted emails, ghost subjects, role flapping).

The empty-`Login` crash is therefore a stop-gap accident: it happens to halt the actualizer before it can re-send emails. Once that single field is patched, every other side-effecting handler will silently re-execute. The correct behaviour is to refuse to replay any pre-refactor invite event at all.

## Why `ActualLogin` can be empty

Two independent paths produce an empty `ActualLogin` on a record whose state is in `{State_Invited, State_ToBeJoined}`:

- Re-invitation. `pkg/sys/invite/impl_initiateinvitationbyemail.go` (line 89) explicitly resets the field on the existing `cdoc.sys.Invite`:

  ```go
  svbCDocInvite.PutString(field_ActualLogin, "") // to be filled with Invitee's login by ap.sys.Apply
  ```

  After re-invite, state passes through `State_ToBeInvited` -> (projector) -> `State_Invited`. While the record sits in `State_Invited` waiting for the new invitee to accept, the projector replays the *old* `sys.InitiateJoinWorkspace` event from the previous join cycle. The state guard `{State_Invited, State_ToBeJoined}` admits the event, `handleApplyJoinWorkspace` runs, and `ActualLogin` is `""`.

- Pre-`d53d48e9` history. `field_ActualLogin` was first written by `c.sys.InitiateJoinWorkspace` in commit `d53d48e9` (#1107). Records created and joined before that commit never had the field populated. If such a record was ever re-invited and is currently in `State_Invited`, the same replay path applies.

## Proposed fix: skip pre-refactor events outright via a `Version` marker

The empty `ActualLogin` is one symptom of a wider invariant break: the new projector should not be applying pre-refactor events at all. Their effects were already produced by the deprecated per-command projectors. Replaying them against the *current* cdoc state (which has moved on through re-invites and re-joins) is meaningless at best and crashing at worst.

Discriminator. Add field `Version int32` to `cdoc.sys.Invite`. Every current invite command writes `Version = 1` to its `cdoc.sys.Invite` CUD. Pre-refactor events written before the discriminator existed have no `Version` write in their CUD, so the field reads `0`.

Why a CUD-side marker works mechanically. `event.CUDs(...)` iterates the per-CUD changes recorded by the command handler, not the merged record (`pkg/istructsmem/event-types.go::cudType.enumRecs` yields `&rec.changes`). A field that the command never `Put*`-d reads back as the type's zero value, even after a PLog round-trip - the dynobuffer encodes set vs unset.

Projector dispatch (`pkg/sys/invite/impl_applyinviteevents.go`). Before `loadInviteByID`, walk `event.CUDs`:

```go
var version int32
for cud := range event.CUDs {
    if cud.QName() == QNameCDocInvite {
        version = cud.AsInt32(Field_Version)
        break
    }
}
if version == 0 {
    return nil // pre-refactor event - effects already applied by deprecated projector
}
```

Command audit. Of the six current invite commands:

- `InitiateInvitationByEMail`, `InitiateJoinWorkspace`, `InitiateLeaveWorkspace` already update `cdoc.sys.Invite`; they only need the extra `PutInt32(Field_Version, 1)`. `InitiateLeaveWorkspace` already established the "no-op CUD to surface the InviteID" pattern.
- `InitiateUpdateInviteRoles`, `InitiateCancelAcceptedInvite`, `CancelSentInvite` currently have no `Intents.UpdateValue` on `cdoc.sys.Invite`; they must add one (mirroring the `InitiateLeaveWorkspace` no-op pattern) carrying `Version = 1`. This also means subsequent handlers can rely on the cdoc.Invite CUD always being present in invite events.

Why this is preferred over a per-field guard

- It is event-scoped, not state-scoped. The decision is made from the immutable event payload, not from current cdoc fields that may have been re-written since.
- It generalises. The same dispatch-level filter protects `handleApplyCancelAcceptedInvite` and `handleApplyLeaveWorkspace` from analogous replays without per-handler patches.
- It is self-documenting. A single explicit field signals "this event came from the post-refactor command set"; future invite commands need only write `Version = 1` to participate.
- It does not require auditing which `State_*` values still appear in new CUDs, unlike a state-shape filter.

Trade-offs

- Schema change on a system cdoc. Because the field is added with a zero default, existing records remain valid; only the projector's interpretation of `Version == 0` changes.
- Six command handlers must be updated (three of which gain a no-op `Intents.UpdateValue` on `cdoc.sys.Invite`). This is a one-time, localised change and does not affect the public command argument contract.
- Future invite commands must remember to set `Version = 1`. Mitigated by adding a small helper or comment in `consts.go`.

Considered and rejected

- Per-field guard `if svCDocInvite.AsString(field_ActualLogin) == "" { return nil }`. Fixes only the empty-`ActualLogin` crash; leaves analogous replay risks in `handleApplyCancelAcceptedInvite` and `handleApplyLeaveWorkspace`; reads from current cdoc state instead of the event.
- CUD shape (presence of legacy `State_*` values). Works mechanically but depends on auditing every current command for which `State_*` it writes; the marker subset is non-obvious and brittle to future refactors.
- Version on the command argument. Would require updating every external client that calls these commands.
