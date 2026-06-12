# Implementation plan: Skip pre-refactor invite events in ap.sys.ApplyInviteEvents via Version marker

## Technical design

- [x] update: [auth/invites--td.md](../../../../specs/prod/auth/invites--td.md)
  - add: `Version int32` field to the `cdoc.sys.Invite` document description in section "Documents"
  - update: section "Single projector design" with a forward reference to the dedicated Versioning subsection
  - update: decision "Single projector as sole writer of final states" to note that the three commands without prior CDoc writes (`InitiateUpdateInviteRoles`, `InitiateCancelAcceptedInvite`, `CancelSentInvite`) now carry a no-op CUD writing `Version = 1`, mirroring the existing `InitiateLeaveWorkspace` pattern
  - add: new subsection "Versioning" under "Technical design" explaining the replay risk that motivates the discriminator, the rule (commands write `Version = 1`; projector skips `Version == 0` via `event.CUDs`), why the discriminator lives on the CUD rather than the merged record (`event.CUDs` yields per-CUD changes only via `cudType.enumRecs`; never-`Put*`-d field reads back as zero through the dynobuffer's set/unset encoding; event-scoped vs state-scoped; single dispatch-level filter protects all side-effecting handlers; no external contract change), and the rejected alternatives (per-field `ActualLogin` guard, CUD-shape filter, command-argument version)

## Construction

### Schema and constants

- [x] update: [pkg/sys/sys.vsql](../../../../../pkg/sys/sys.vsql)
  - add: `Version int32` field to `TABLE Invite` (default `0` for existing records; current commands write `1`)

- [x] update: [pkg/sys/invite/consts.go](../../../../../pkg/sys/invite/consts.go)
  - add: `Field_Version = "Version"` constant in the field-name `const` block

### Command handlers writing Version = 1

- [x] update: [pkg/sys/invite/impl_initiateinvitationbyemail.go](../../../../../pkg/sys/invite/impl_initiateinvitationbyemail.go)
  - add: `svbCDocInvite.PutInt32(Field_Version, 1)` on both the existing-invite update branch and the create branch

- [x] update: [pkg/sys/invite/impl_initiatejoinworkspace.go](../../../../../pkg/sys/invite/impl_initiatejoinworkspace.go)
  - add: `svbCDocInvite.PutInt32(Field_Version, 1)` on the `Intents.UpdateValue` already produced by the handler

- [x] update: [pkg/sys/invite/impl_initiateleaveworkspace.go](../../../../../pkg/sys/invite/impl_initiateleaveworkspace.go)
  - update: extend the existing no-op CUD - obtain the `svbCDocInvite` from `args.Intents.UpdateValue(skbCDocInvite, svCDocInvite)` and write `Field_Version = 1`

- [x] update: [pkg/sys/invite/impl_initiateupdateinviteroles.go](../../../../../pkg/sys/invite/impl_initiateupdateinviteroles.go)
  - add: no-op CUD on `cdoc.sys.Invite` (mirroring the existing pattern in `impl_initiateleaveworkspace.go`) writing only `Field_Version = 1`

- [x] update: [pkg/sys/invite/impl_initiatecancelacceptedinvite.go](../../../../../pkg/sys/invite/impl_initiatecancelacceptedinvite.go)
  - add: no-op CUD on `cdoc.sys.Invite` writing only `Field_Version = 1`

- [x] update: [pkg/sys/invite/impl_cancelsentinvite.go](../../../../../pkg/sys/invite/impl_cancelsentinvite.go)
  - add: no-op CUD on `cdoc.sys.Invite` writing only `Field_Version = 1`

### Projector skip logic

- [x] update: [pkg/sys/invite/impl_applyinviteevents.go](../../../../../pkg/sys/invite/impl_applyinviteevents.go)
  - add: helper `findInviteCUD(event)` that iterates `event.CUDs` and returns the first `QNameCDocInvite` CUD (or `nil`)
  - add: at the top of `applyInviteEvents` returned func (before `inviteIDFromEvent`), call `findInviteCUD` and `return nil` when the CUD is absent or its `Field_Version` reads `0` (pre-refactor event - already applied by deprecated per-command projectors, must not be replayed)
  - update: change `inviteIDFromEvent` signature to accept the located `inviteCUD` and, for `qNameCmdInitiateLeaveWorkspace`, return `inviteCUD.ID()` directly, removing the redundant `event.CUDs` pass that the previous implementation performed

### Tests

- [x] update: [pkg/sys/it/impl_invite_test.go](../../../../../pkg/sys/it/impl_invite_test.go)
  - add: dedicated `TestInvite_VersionMarker` with a local `getInviteVersion` helper (q.sys.Collection query projecting `Version`), asserting `Version == 1` after each of the six commands runs:
    - create branch of `c.sys.InitiateInvitationByEMail`
    - `c.sys.CancelSentInvite`
    - re-invite update branch of `c.sys.InitiateInvitationByEMail` (after `CancelSentInvite`)
    - `c.sys.InitiateJoinWorkspace`
    - `c.sys.InitiateUpdateInviteRoles`
    - `c.sys.InitiateLeaveWorkspace`
    - `c.sys.InitiateCancelAcceptedInvite`

- [x] create: [pkg/sys/invite/impl_applyinviteevents_test.go](../../../../../pkg/sys/invite/impl_applyinviteevents_test.go)
  - add: `newMockEventWithCUDs` / `newInviteCUD` helpers building a `coreutils.MockPLogEvent` and `coreutils.TestObject` with the configured CUD list and Version
  - add: `TestApplyInviteEvents_SkipsEventsWithoutVersionMarker` - drives `applyInviteEvents` with strict `coreutils.MockState` and `coreutils.MockIntents` (no `On(...)` registrations, so any state/intents call fails the test) for four sub-cases: no `cdoc.sys.Invite` CUD; invite CUD with `Version == 0`; only non-invite CUDs; invite CUD with `Version == 0` listed after a non-invite CUD. All sub-cases assert `projectorFn` returns `nil`
  - add: `TestApplyInviteEvents_Version1ReachesDispatch` - positive-case unit test for `qNameCmdInitiateLeaveWorkspace` with `Version == 1`; `MockState.KeyBuilder(sys.Storage_Record, QNameCDocInvite)` is wired to return an injected error, and the test asserts the projector propagates that error back, proving execution advanced past the Version check into `loadInviteByID`
