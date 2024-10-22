/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

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
	skbViewSubjectsIdx.PutInt64(Field_LoginHash, coreutils.HashBytes([]byte(login)))
	skbViewSubjectsIdx.PutString(Field_Login, login)
	return skbViewSubjectsIdx, nil
}

// checks cdoc.sys.SubjectIdx existence by login as cdoc.sys.Invite.EMail and as token.Login
func SubjectExistByBothLogins(login string, st istructs.IState) (ok bool, actualLogin string, existingSubjectID istructs.RecordID, _ error) {
	subjectExists, exisingSubjectID, err := SubjectExistsByLogin(login, st) // for backward compatibility
	if err != nil {
		return false, "", 0, err
	}
	skbPrincipal, err := st.KeyBuilder(sys.Storage_RequestSubject, appdef.NullQName)
	if err != nil {
		return false, "", 0, err
	}
	svPrincipal, err := st.MustExist(skbPrincipal)
	if err != nil {
		return
	}
	actualLogin = svPrincipal.AsString(sys.Storage_RequestSubject_Field_Name)
	if !subjectExists {
		subjectExists, exisingSubjectID, err = SubjectExistsByLogin(actualLogin, st)
		if err != nil {
			return false, "", 0, err
		}
	}
	return subjectExists, actualLogin, exisingSubjectID, nil

}

func SubjectExistsByLogin(login string, state istructs.IState) (ok bool, existingSubjectID istructs.RecordID, _ error) {
	skbViewSubjectsIdx, err := GetSubjectIdxViewKeyBuilder(login, state)
	if err != nil {
		// notest
		return false, 0, err
	}
	val, ok, err := state.CanExist(skbViewSubjectsIdx)
	if err != nil {
		// notest
		return false, 0, err
	}
	if ok {
		existingSubjectID = val.AsRecordID("SubjectID")
	}
	return ok, existingSubjectID, nil
}
