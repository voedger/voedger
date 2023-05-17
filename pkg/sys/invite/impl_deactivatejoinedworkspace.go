/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func provideCmdDeactivateJoinedWorkspace(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdDeactivateJoinedWorkspace,
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "DeactivateJoinedWorkspaceParams"), appdef.DefKind_Object).
			AddField(Field_InvitingWorkspaceWSID, appdef.DataKind_int64, true).
			QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdDeactivateJoinedWorkspace,
	))
}

func execCmdDeactivateJoinedWorkspace(_ istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	svbCDocJoinedWorkspace, err := GetCDocJoinedWorkspaceForUpdateRequired(args.State, args.Intents, args.ArgumentObject.AsInt64(Field_InvitingWorkspaceWSID))
	if err == nil {
		svbCDocJoinedWorkspace.PutBool(appdef.SystemField_IsActive, false)
	}
	return err
}
