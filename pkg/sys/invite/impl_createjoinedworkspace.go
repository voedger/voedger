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

func provideCmdCreateJoinedWorkspace(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdCreateJoinedWorkspace,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "CreateJoinedWorkspaceParams")).
			AddField(Field_Roles, appdef.DataKind_string, true).
			AddField(field_InvitingWorkspaceWSID, appdef.DataKind_int64, true).
			AddField(field_WSName, appdef.DataKind_string, true).
			QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdCreateJoinedWorkspace,
	))
}

func execCmdCreateJoinedWorkspace(_ istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	skbViewJoinedWorkspaceIndex, err := args.State.KeyBuilder(state.ViewRecordsStorage, qNameViewJoinedWorkspaceIndex)
	if err != nil {
		return
	}
	skbViewJoinedWorkspaceIndex.PutInt32(field_Dummy, value_Dummy_Two)
	skbViewJoinedWorkspaceIndex.PutInt64(field_InvitingWorkspaceWSID, args.ArgumentObject.AsInt64(field_InvitingWorkspaceWSID))
	svViewJoinedWorkspaceIndex, ok, err := args.State.CanExist(skbViewJoinedWorkspaceIndex)
	if err != nil {
		return
	}
	if ok {
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
		svbCDocJoinedWorkspace.PutString(Field_Roles, args.ArgumentObject.AsString(Field_Roles))
		svbCDocJoinedWorkspace.PutBool(appdef.SystemField_IsActive, true)

		return nil
	}
	skbCDocJoinedWorkspace, err := args.State.KeyBuilder(state.RecordsStorage, qNameCDocJoinedWorkspace)
	if err != nil {
		return
	}
	svbCDocJoinedWorkspace, err := args.Intents.NewValue(skbCDocJoinedWorkspace)
	if err != nil {
		return err
	}
	svbCDocJoinedWorkspace.PutRecordID(appdef.SystemField_ID, istructs.RecordID(1))
	svbCDocJoinedWorkspace.PutString(Field_Roles, args.ArgumentObject.AsString(Field_Roles))
	svbCDocJoinedWorkspace.PutString(field_WSName, args.ArgumentObject.AsString(field_WSName))
	svbCDocJoinedWorkspace.PutInt64(field_InvitingWorkspaceWSID, args.ArgumentObject.AsInt64(field_InvitingWorkspaceWSID))

	return err
}
