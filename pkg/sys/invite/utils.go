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

// SubjectExistsByLogin returns SubjectID and isActive status for a Subject with the given login.
// Returns (0, false, nil) if no Subject exists for this login.
func SubjectExistsByLogin(login string, state istructs.IState) (subjectID istructs.RecordID, isActive bool, err error) {
	skbViewSubjectsIdx, err := GetSubjectIdxViewKeyBuilder(login, state)
	if err != nil {
		// notest
		return 0, false, err
	}
	val, ok, err := state.CanExist(skbViewSubjectsIdx)
	if err != nil || !ok {
		return 0, false, err
	}
	subjectID = val.AsRecordID(Field_SubjectID)

	skbSubject, err := state.KeyBuilder(sys.Storage_Record, QNameCDocSubject)
	if err != nil {
		// notest
		return 0, false, err
	}
	skbSubject.PutRecordID(sys.Storage_Record_Field_ID, subjectID)
	svSubject, ok, err := state.CanExist(skbSubject)
	if err != nil || !ok {
		return 0, false, err
	}
	return subjectID, svSubject.AsBool(appdef.SystemField_IsActive), nil
}
