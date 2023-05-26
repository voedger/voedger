/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package invite

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	sysshared "github.com/voedger/voedger/pkg/sys/shared"
)

func GetCDocJoinedWorkspaceForUpdateRequired(st istructs.IState, intents istructs.IIntents, invitingWorkspaceWSID int64) (svbCDocJoinedWorkspace istructs.IStateValueBuilder, err error) {
	skbViewJoinedWorkspaceIndex, err := st.KeyBuilder(state.ViewRecordsStorage, QNameViewJoinedWorkspaceIndex)
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
	skb, err := st.KeyBuilder(state.RecordsStorage, sysshared.QNameCDocJoinedWorkspace)
	if err != nil {
		// notest
		return nil, err
	}
	skb.PutRecordID(state.Field_ID, svViewJoinedWorkspaceIndex.AsRecordID(field_JoinedWorkspaceID))
	svCDocJoinedWorkspace, err := st.MustExist(skb)
	if err != nil {
		return nil, err
	}
	svbCDocJoinedWorkspace, err = intents.UpdateValue(skb, svCDocJoinedWorkspace)
	return
}

func GetCDocJoinedWorkspace(st istructs.IState, intents istructs.IIntents, invitingWorkspaceWSID int64) (svbCDocJoinedWorkspace istructs.IStateValue, skb istructs.IStateKeyBuilder, ok bool, err error) {
	skbViewJoinedWorkspaceIndex, err := st.KeyBuilder(state.ViewRecordsStorage, QNameViewJoinedWorkspaceIndex)
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

	skb, err = st.KeyBuilder(state.RecordsStorage, sysshared.QNameCDocJoinedWorkspace)
	if err != nil {
		// notest
		return nil, nil, false, err
	}
	skb.PutRecordID(state.Field_ID, svViewJoinedWorkspaceIndex.AsRecordID(field_JoinedWorkspaceID))
	svbCDocJoinedWorkspace, ok, err = st.CanExist(skb)
	return svbCDocJoinedWorkspace, skb, ok, err
}

func GetCDocJoinedWorkspaceForUpdate(st istructs.IState, intents istructs.IIntents, invitingWorkspaceWSID int64) (svbCDocJoinedWorkspace istructs.IStateValueBuilder, ok bool, err error) {
	svCDocJoinedWorkspace, skb, ok, err := GetCDocJoinedWorkspace(st, intents, invitingWorkspaceWSID)
	if err != nil || !ok {
		return nil, false, err
	}
	svbCDocJoinedWorkspace, err = intents.UpdateValue(skb, svCDocJoinedWorkspace)
	return svbCDocJoinedWorkspace, true, err
}
