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

func provideCmdUpdateJoinedWorkspaceRoles(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdUpdateJoinedWorkspaceRoles,
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "UpdateJoinedWorkspaceRolesParams"), appdef.DefKind_Object).
			AddField(Field_Roles, appdef.DataKind_string, true).
			AddField(Field_InvitingWorkspaceWSID, appdef.DataKind_int64, true).
			QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdUpdateJoinedWorkspaceRoles,
	))
}

func execCmdUpdateJoinedWorkspaceRoles(_ istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	skbViewJoinedWorkspaceIndex, err := args.State.KeyBuilder(state.ViewRecordsStorage, qNameViewJoinedWorkspaceIndex)
	if err != nil {
		return
	}
	skbViewJoinedWorkspaceIndex.PutInt32(field_Dummy, value_Dummy_Two)
	skbViewJoinedWorkspaceIndex.PutInt64(Field_InvitingWorkspaceWSID, args.ArgumentObject.AsInt64(Field_InvitingWorkspaceWSID))
	svViewJoinedWorkspaceIndex, err := args.State.MustExist(skbViewJoinedWorkspaceIndex)
	if err != nil {
		return
	}

	skbCDocJoinedWorkspace, err := args.State.KeyBuilder(state.RecordsStorage, QNameCDocJoinedWorkspace)
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
	svbCDocJoinedWorkspace.PutString(Field_Roles, args.ArgumentObject.AsString(Field_Roles))
	svbCDocJoinedWorkspace.PutBool(appdef.SystemField_IsActive, true)

	return err
}
