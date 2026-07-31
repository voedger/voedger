/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package invite

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func validateInviteRoles(rolesStr string, ws appdef.IWorkspace) error {
	trimmed := strings.TrimSpace(rolesStr)
	if len(trimmed) == 0 {
		return coreutils.NewHTTPError(http.StatusBadRequest, ErrRolesEmpty)
	}
	seen := make(map[appdef.QName]struct{})
	for role := range strings.SplitSeq(trimmed, ",") {
		role = strings.TrimSpace(role)
		if len(role) == 0 {
			return coreutils.NewHTTPError(http.StatusBadRequest, ErrRolesEmpty)
		}
		qName, err := appdef.ParseQName(role)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, fmt.Errorf("%w: %s: %w", ErrRoleInvalid, role, err))
		}
		if _, ok := seen[qName]; ok {
			return coreutils.NewHTTPError(http.StatusBadRequest, fmt.Errorf("%w: %s", ErrRoleDuplicate, role))
		}
		seen[qName] = struct{}{}
		if iauthnz.IsSystemRole(qName) {
			return coreutils.NewHTTPError(http.StatusBadRequest, fmt.Errorf("%w: %s", ErrSystemRole, role))
		}
		if appdef.Role(ws.Type, qName) == nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, fmt.Errorf("%w: %s", ErrRoleNotFound, role))
		}
	}
	return nil
}

func GetCDocJoinedWorkspaceForUpdateRequired(st istructs.IState, intents istructs.IIntents, invitingWorkspaceWSID int64) (svbCDocJoinedWorkspace istructs.IStateValueBuilder, err error) {
	skbViewJoinedWorkspaceIndex, err := st.KeyBuilder(sys.Storage_View, QNameViewJoinedWorkspaceIndex)
	if err != nil {
		// notest
		return nil, err
	}
	skbViewJoinedWorkspaceIndex.PutInt32(field_Dummy, value_Dummy_Two)
	skbViewJoinedWorkspaceIndex.PutInt64(Field_InvitingWorkspaceWSID, invitingWorkspaceWSID)
	svViewJoinedWorkspaceIndex, err := st.MustExist(skbViewJoinedWorkspaceIndex)
	if err != nil {
		return nil, err
	}
	skb, err := st.KeyBuilder(sys.Storage_Record, QNameCDocJoinedWorkspace)
	if err != nil {
		// notest
		return nil, err
	}
	skb.PutRecordID(sys.Storage_Record_Field_ID, svViewJoinedWorkspaceIndex.AsRecordID(field_JoinedWorkspaceID))
	svCDocJoinedWorkspace, err := st.MustExist(skb)
	if err != nil {
		return nil, err
	}
	svbCDocJoinedWorkspace, err = intents.UpdateValue(skb, svCDocJoinedWorkspace)
	return
}

func GetCDocJoinedWorkspace(st istructs.IState, invitingWorkspaceWSID int64) (svbCDocJoinedWorkspace istructs.IStateValue, skb istructs.IStateKeyBuilder, ok bool, err error) {
	skbViewJoinedWorkspaceIndex, err := st.KeyBuilder(sys.Storage_View, QNameViewJoinedWorkspaceIndex)
	if err != nil {
		// notest
		return nil, nil, false, err
	}
	skbViewJoinedWorkspaceIndex.PutInt32(field_Dummy, value_Dummy_Two)
	skbViewJoinedWorkspaceIndex.PutInt64(Field_InvitingWorkspaceWSID, invitingWorkspaceWSID)
	svViewJoinedWorkspaceIndex, ok, err := st.CanExist(skbViewJoinedWorkspaceIndex)
	if err != nil {
		// notest
		return nil, nil, false, err
	}
	if !ok {
		return nil, nil, false, nil
	}

	skb, err = st.KeyBuilder(sys.Storage_Record, QNameCDocJoinedWorkspace)
	if err != nil {
		// notest
		return nil, nil, false, err
	}
	skb.PutRecordID(sys.Storage_Record_Field_ID, svViewJoinedWorkspaceIndex.AsRecordID(field_JoinedWorkspaceID))
	svbCDocJoinedWorkspace, ok, err = st.CanExist(skb)
	return svbCDocJoinedWorkspace, skb, ok, err
}

func GetCDocJoinedWorkspaceForUpdate(st istructs.IState, intents istructs.IIntents, invitingWorkspaceWSID int64) (svbCDocJoinedWorkspace istructs.IStateValueBuilder, ok bool, err error) {
	svCDocJoinedWorkspace, skb, ok, err := GetCDocJoinedWorkspace(st, invitingWorkspaceWSID)
	if err != nil || !ok {
		return nil, false, err
	}
	svbCDocJoinedWorkspace, err = intents.UpdateValue(skb, svCDocJoinedWorkspace)
	return svbCDocJoinedWorkspace, true, err
}

func GetSubjectIdxViewKeyBuilder(login string, s istructs.IState) (istructs.IStateKeyBuilder, error) {
	skbViewSubjectsIdx, err := s.KeyBuilder(sys.Storage_View, QNameViewSubjectsIdx)
	if err != nil {
		// notest
		return nil, err
	}
	skbViewSubjectsIdx.PutInt64(Field_LoginHash, coreutils.LoginHash(login))
	skbViewSubjectsIdx.PutString(Field_Login, login)
	return skbViewSubjectsIdx, nil
}

func LoginFromToken(st istructs.IState) (loginFromToken string, err error) {
	skbPrincipal, err := st.KeyBuilder(sys.Storage_RequestSubject, appdef.NullQName)
	if err != nil {
		return "", err
	}
	svPrincipal, err := st.MustExist(skbPrincipal)
	if err != nil {
		return "", err
	}
	return svPrincipal.AsString(sys.Storage_RequestSubject_Field_Name), nil
}

// subjectByLogin returns the Subject ID and record for a canonical login.
func subjectByLogin(login string, state istructs.IState) (subjectID istructs.RecordID, subject istructs.IStateValue, ok bool, err error) {
	skbViewSubjectsIdx, err := GetSubjectIdxViewKeyBuilder(login, state)
	if err != nil {
		// notest
		return 0, nil, false, err
	}
	val, ok, err := state.CanExist(skbViewSubjectsIdx)
	if err != nil || !ok {
		return 0, nil, false, err
	}
	subjectID = val.AsRecordID(Field_SubjectID)

	skbSubject, err := state.KeyBuilder(sys.Storage_Record, QNameCDocSubject)
	if err != nil {
		// notest
		return 0, nil, false, err
	}
	skbSubject.PutRecordID(sys.Storage_Record_Field_ID, subjectID)
	subject, ok, err = state.CanExist(skbSubject)
	if err != nil || !ok {
		return 0, nil, false, err
	}
	return subjectID, subject, true, nil
}

// SubjectExistsByLogin returns SubjectID and isActive status for a Subject with the given login.
// Returns (0, false, nil) if no Subject exists for this login.
func SubjectExistsByLogin(login string, state istructs.IState) (subjectID istructs.RecordID, isActive bool, err error) {
	subjectID, subject, ok, err := subjectByLogin(login, state)
	if err != nil || !ok {
		return 0, false, err
	}
	return subjectID, subject.AsBool(appdef.SystemField_IsActive), nil
}

// findInviteByEmail resolves an exact invitation email through InviteIndexView.
func findInviteByEmail(email string, state istructs.IState) (inviteID istructs.RecordID, invite istructs.IStateValue, ok bool, err error) {
	if email == "" {
		return 0, nil, false, nil
	}

	skbViewInviteIndex, err := state.KeyBuilder(sys.Storage_View, qNameViewInviteIndex)
	if err != nil {
		// notest
		return 0, nil, false, err
	}
	skbViewInviteIndex.PutInt32(field_Dummy, value_Dummy_One)
	skbViewInviteIndex.PutString(Field_Login, email)
	svViewInviteIndex, ok, err := state.CanExist(skbViewInviteIndex)
	if err != nil || !ok {
		return 0, nil, false, err
	}

	inviteID = svViewInviteIndex.AsRecordID(field_InviteID)
	if inviteID == istructs.NullRecordID {
		return 0, nil, false, nil
	}

	skbInvite, err := state.KeyBuilder(sys.Storage_Record, QNameCDocInvite)
	if err != nil {
		// notest
		return 0, nil, false, err
	}
	skbInvite.PutRecordID(sys.Storage_Record_Field_ID, inviteID)
	invite, ok, err = state.CanExist(skbInvite)
	if err != nil || !ok {
		return 0, nil, false, err
	}
	return inviteID, invite, true, nil
}

// resolveControllingInvite returns the sole Joined invitation controlling subject.
// Subject.InviteEmail is authoritative when it resolves to a valid controller; otherwise
// canonical login and active alias provide the legacy fallback candidates.
func resolveControllingInvite(subjectID istructs.RecordID, subject istructs.IStateValue, canonicalLogin, activeAlias string, excludedInviteID istructs.RecordID, state istructs.IState) (inviteID istructs.RecordID, invite istructs.IStateValue, err error) {
	type inviteLookupResult struct {
		id     istructs.RecordID
		invite istructs.IStateValue
		ok     bool
		err    error
	}

	lookupCache := map[string]inviteLookupResult{}
	lookup := func(email string) inviteLookupResult {
		if result, found := lookupCache[email]; found {
			return result
		}
		id, value, ok, err := findInviteByEmail(email, state)
		result := inviteLookupResult{id: id, invite: value, ok: ok, err: err}
		lookupCache[email] = result
		return result
	}

	isValidController := func(result inviteLookupResult) bool {
		return result.ok &&
			result.id != excludedInviteID &&
			State(result.invite.AsInt32(Field_State)) == State_Joined &&
			result.invite.AsRecordID(field_SubjectID) == subjectID
	}

	if result := lookup(subject.AsString(Field_InviteEmail)); result.err != nil {
		return 0, nil, result.err
	} else if isValidController(result) {
		return result.id, result.invite, nil
	}

	candidates := map[istructs.RecordID]istructs.IStateValue{}
	for _, email := range []string{canonicalLogin, activeAlias} {
		if email == "" {
			continue
		}
		result := lookup(email)
		if result.err != nil {
			return 0, nil, result.err
		}
		if isValidController(result) {
			candidates[result.id] = result.invite
		}
	}

	if len(candidates) != 1 {
		return 0, nil, ErrControllingInviteNotIdentified
	}
	for id, value := range candidates {
		return id, value, nil
	}
	return 0, nil, ErrControllingInviteNotIdentified
}
