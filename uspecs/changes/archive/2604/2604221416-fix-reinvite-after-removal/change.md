---
registered_at: 2026-04-22T09:14:48Z
change_id: 2604220914-fix-reinvite-after-removal
baseline: cde799cbe15807fbd753b5c4a00c53efc1b8d4af
issue_url: https://untill.atlassian.net/browse/AIR-3664
archived_at: 2026-04-22T14:16:32Z
---

# Change request: Fix user cannot be reinvited after removal

## Why

When a joined user is removed from a workspace (via cancel accepted invite), the `view.sys.SubjectsIdx` still contains the old Subject entry. On reinvitation, `SubjectExistsByLogin()` finds this entry and `ApplyJoinWorkspace` skips creating the Subject, leaving the invite stuck.

The protection message appears:
```text
skip aproj.sys.ApplyJoinWorkspace because cdoc.sys.SubjectID.{id} exists already by cdoc.sys.Invite.Login "{email}"
```

This happens because:

- User joins workspace -> `cdoc.sys.Subject` created, `view.sys.SubjectsIdx` populated with login -> SubjectID mapping
- Admin removes user -> `cdoc.sys.Subject.sys.IsActive` set to `false`, but `view.sys.SubjectsIdx` entry remains
- Admin reinvites same email -> new invite created, user clicks link
- `ap.sys.ApplyJoinWorkspace` checks `SubjectExistsByLogin()` which queries `view.sys.SubjectsIdx`
- View returns old (now inactive) SubjectID -> projector skips Subject creation to prevent unique constraint violation
- Invite never transitions to `State_Joined`, user stuck

## What

Refactor `SubjectExistsByLogin()` to return both SubjectID and isActive status:

- Return `(subjectID, isActive, err)` instead of just `(subjectID, err)`
- Callers can decide: skip if active, reactivate if inactive, create if not found
- `ApplyJoinWorkspace` uses `existingSubjectID` directly for reactivation (no separate lookup needed)

## How

- Write failing integration test first (`TestReinviteAfterCancelAcceptedInvite`)
- Refactor `SubjectExistsByLogin()` to return `(subjectID, isActive, err)`
- Update `ApplyJoinWorkspace` to use `isActive` for skip check, `existingSubjectID` for reactivation
- Update `InitiateInvitationByEMail` to use `subjectIsActive` instead of `existingSubjectID > 0`
- Add comments explaining `Login` vs `ActualLogin` fields in vsql and projector

References:

- [pkg/sys/it/impl_invite_test.go](../../../../../pkg/sys/it/impl_invite_test.go)
- [pkg/sys/invite/utils.go](../../../../../pkg/sys/invite/utils.go)
- [pkg/sys/invite/impl_applyjoinworkspace.go](../../../../../pkg/sys/invite/impl_applyjoinworkspace.go)
- [pkg/sys/invite/impl_initiateinvitationbyemail.go](../../../../../pkg/sys/invite/impl_initiateinvitationbyemail.go)
- [pkg/sys/sys.vsql](../../../../../pkg/sys/sys.vsql)
