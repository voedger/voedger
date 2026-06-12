# Implementation plan: Fix user cannot be reinvited after removal

## Construction

### Tests

- [x] update: [pkg/sys/it/impl_invite_test.go](../../../../../pkg/sys/it/impl_invite_test.go)
  - add: `TestReinviteAfterCancelAcceptedInvite` - test that reproduces the bug scenario
- [x] run tests to verify the new test fails before fix is applied
- [x] review

### Fix

- [x] update: [pkg/sys/invite/utils.go](../../../../../pkg/sys/invite/utils.go)
  - refactor: `SubjectExistsByLogin()` returns `(subjectID, isActive, err)` instead of just `(subjectID, err)`
- [x] update: [pkg/sys/invite/impl_applyjoinworkspace.go](../../../../../pkg/sys/invite/impl_applyjoinworkspace.go)
  - fix: use `isActive` to decide skip, use `existingSubjectID` for reactivation
  - fix: reactivate existing Subject by splitting update into two CUD calls (IsActive first, then Roles)
  - refactor: removed separate lookup by `Invite.SubjectID` (no longer needed)
  - refactor: removed unused `collection` import
  - add: comments explaining `Login` vs `ActualLogin` fields
- [x] update: [pkg/sys/invite/impl_initiateinvitationbyemail.go](../../../../../pkg/sys/invite/impl_initiateinvitationbyemail.go)
  - fix: use `subjectIsActive` instead of `existingSubjectID > 0`
- [x] update: [pkg/sys/sys.vsql](../../../../../pkg/sys/sys.vsql)
  - add: comments explaining `Login` and `ActualLogin` fields in Invite table
- [x] run tests to verify that now test passes after fix is applied
