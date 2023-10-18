/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func provideCmdCreateJoinedWorkspace(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdCreateJoinedWorkspace,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "CreateJoinedWorkspaceParams")).
			AddField(Field_Roles, appdef.DataKind_string, true).
			AddField(Field_InvitingWorkspaceWSID, appdef.DataKind_int64, true).
			AddField(authnz.Field_WSName, appdef.DataKind_string, true).(appdef.IType).QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdCreateJoinedWorkspace,
	))
}

func execCmdCreateJoinedWorkspace(args istructs.ExecCommandArgs) (err error) {
	svbCDocJoinedWorkspace, ok, err := GetCDocJoinedWorkspaceForUpdate(args.State, args.Intents, args.ArgumentObject.AsInt64(Field_InvitingWorkspaceWSID))
	if err != nil {
		// notest
		return err
	}
	if ok {
		svbCDocJoinedWorkspace.PutString(Field_Roles, args.ArgumentObject.AsString(Field_Roles))
		svbCDocJoinedWorkspace.PutBool(appdef.SystemField_IsActive, true)

		return nil
	}
	skbCDocJoinedWorkspace, err := args.State.KeyBuilder(state.Record, QNameCDocJoinedWorkspace)
	if err != nil {
		return
	}
	svbCDocJoinedWorkspace, err = args.Intents.NewValue(skbCDocJoinedWorkspace)
	if err != nil {
		return err
	}
	svbCDocJoinedWorkspace.PutRecordID(appdef.SystemField_ID, istructs.RecordID(1))
	svbCDocJoinedWorkspace.PutString(Field_Roles, args.ArgumentObject.AsString(Field_Roles))
	svbCDocJoinedWorkspace.PutString(authnz.Field_WSName, args.ArgumentObject.AsString(authnz.Field_WSName))
	svbCDocJoinedWorkspace.PutInt64(Field_InvitingWorkspaceWSID, args.ArgumentObject.AsInt64(Field_InvitingWorkspaceWSID))

	return err
}
