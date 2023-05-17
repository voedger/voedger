/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
)

func provideCmdDeactivateJoinedWorkspace(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdDeactivateJoinedWorkspace,
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "DeactivateJoinedWorkspaceParams"), appdef.DefKind_Object).
			AddField(field_InvitingWorkspaceWSID, appdef.DataKind_int64, true).
			QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdDeactivateJoinedWorkspace,
	))
}

func execCmdDeactivateJoinedWorkspace(_ istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	skbViewJoinedWorkspaceIndex, err := args.State.KeyBuilder(state.ViewRecordsStorage, qNameViewJoinedWorkspaceIndex)
	if err != nil {
		return
	}
	skbViewJoinedWorkspaceIndex.PutInt32(field_Dummy, value_Dummy_Two)
	skbViewJoinedWorkspaceIndex.PutInt64(field_InvitingWorkspaceWSID, args.ArgumentObject.AsInt64(field_InvitingWorkspaceWSID))
	svViewJoinedWorkspaceIndex, err := args.State.MustExist(skbViewJoinedWorkspaceIndex)
	if err != nil {
		return
	}

	skbCDocJoinedWorkspace, err := args.State.KeyBuilder(state.RecordsStorage, qNameCDocJoinedWorkspace)
	if err != nil {
		return err
	}
	skbCDocJoinedWorkspace.PutRecordID(state.Field_ID, svViewJoinedWorkspaceIndex.AsRecordID(field_JoinedWorkspaceID))
	svCDocJoinedWorkspace, err := args.State.MustExist(skbCDocJoinedWorkspace)
	if err != nil {
		return err
	}
	svbCDocJoinedWorkspace, err := args.Intents.UpdateValue(skbCDocJoinedWorkspace, svCDocJoinedWorkspace)
	if err != nil {
		return err
	}
	svbCDocJoinedWorkspace.PutBool(appdef.SystemField_IsActive, false)
	return err
}
