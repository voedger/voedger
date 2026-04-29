---
registered_at: 2026-04-29T08:00:13Z
change_id: 2604290800-guard-empty-actuallogin-join
baseline: 82dba9f71f8a2248b7f8ffc81b1d312628dfff9c
pre_pr_head: cc6a7eb0d927ec67cad1990adfc40f6cc379c2cf
archived_at: 2026-04-29T09:59:41Z
---

# Change request: Skip pre-refactor invite events in ap.sys.ApplyInviteEvents via Version marker

## Why

After PR [#4521](https://github.com/voedger/voedger/pull/4521) (commit `82dba9f7`, AIR-3704) replaced the per-command invite projectors with the single `ap.sys.ApplyInviteEvents` projector, the actualizer has no stored offset under the new projector QName and replays every workspace's invite history from WLog offset 0.

The pre-refactor events were already fully processed by the deprecated per-command projectors at the time they were written. Replaying them now re-executes their side effects against today's database state. Those side effects are not internal bookkeeping - they are externally visible:

- re-send invitation emails to the original invitees (`handleApplyInvitation` -> `sendEmail`)
- re-issue role-update HTTP calls into the invitee's profile workspace and re-send role-change emails (`handleApplyUpdateInviteRoles` -> `fed.Func("c.sys.UpdateJoinedWorkspaceRoles")` + `sendEmail`)
- resurrect cancelled subjects (`handleApplyJoinWorkspace`)
- flip active memberships back to inactive (`handleApplyCancelAcceptedInvite`, `handleApplyLeaveWorkspace`)

The crash currently observed in production (`field is empty: view «sys.ViewSubjectsIdx» key string-field «Login»: validate error code 4`, `evqname=sys.InitiateJoinWorkspace`, `extension=ap.sys.ApplyInviteEvents`) is a stop-gap accident that happens to halt the actualizer before it can re-send emails on the affected event. Once that single field is patched, every other side-effecting handler will silently re-execute on every replayed pre-refactor event in every workspace.

The correct fix is therefore to refuse to replay any pre-refactor invite event at all, not to patch individual symptom fields.

See [analysis.md](analysis.md) for the full root cause analysis.

## What

Add a `Version` discriminator on `cdoc.sys.Invite` so the projector can cheaply tell a pre-refactor event apart from a post-refactor one and skip the former.

Schema:

- Add field `Version int32` to `cdoc.sys.Invite` (default `0` for all existing records)

Commands (every current invite cmd writes `Version = 1` in its `cdoc.sys.Invite` CUD):

- `c.sys.InitiateInvitationByEMail` (`pkg/sys/invite/impl_initiateinvitationbyemail.go`) - writes `Version = 1` on both create and update branches
- `c.sys.InitiateJoinWorkspace` (`pkg/sys/invite/impl_initiatejoinworkspace.go`) - writes `Version = 1` on its update
- `c.sys.InitiateUpdateInviteRoles` (`pkg/sys/invite/impl_initiateupdateinviteroles.go`) - currently has no CUD; add a `args.Intents.UpdateValue` carrying `Version = 1`
- `c.sys.InitiateCancelAcceptedInvite` (`pkg/sys/invite/impl_initiatecancelacceptedinvite.go`) - currently has no CUD; add the same
- `c.sys.InitiateLeaveWorkspace` (`pkg/sys/invite/impl_initiateleaveworkspace.go`) - already has a no-op CUD ("keep UpdateValue so projector can discover InviteID from event.CUDs"); extend it to write `Version = 1`
- `c.sys.CancelSentInvite` (`pkg/sys/invite/impl_cancelsentinvite.go`) - currently has no CUD; add the same

Projector (`pkg/sys/invite/impl_applyinviteevents.go`):

- At the dispatch site (before `loadInviteByID`), iterate `event.CUDs` looking for the cdoc.sys.Invite CUD
- If no `cdoc.sys.Invite` CUD is present, or its `Version` reads `0`, `return nil` (pre-refactor event - already applied by deprecated projector, must not be replayed)
- Otherwise dispatch to the per-command handler as today

Tests:

- Drive the projector with a synthesised pre-refactor event (CUD on cdoc.sys.Invite without `Version`) and assert the projector returns `nil` and writes nothing
- Drive the projector with a post-refactor event (`Version = 1`) and assert normal processing
- Re-invite scenario: assert that an old `sys.InitiateJoinWorkspace` event in the WLog is skipped after re-invite, while the new join's event proceeds to create the subject
