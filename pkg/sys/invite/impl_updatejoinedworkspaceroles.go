/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func provideCmdUpdateJoinedWorkspaceRoles(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdUpdateJoinedWorkspaceRoles,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "UpdateJoinedWorkspaceRolesParams")).
			AddField(Field_Roles, appdef.DataKind_string, true).
			AddField(Field_InvitingWorkspaceWSID, appdef.DataKind_int64, true).(appdef.IDef).QName(),
		appdef.NullQName,
		appdef.NullQName,
		execCmdUpdateJoinedWorkspaceRoles,
	))
}

func execCmdUpdateJoinedWorkspaceRoles(_ istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	svbCDocJoinedWorkspace, err := GetCDocJoinedWorkspaceForUpdateRequired(args.State, args.Intents, args.ArgumentObject.AsInt64(Field_InvitingWorkspaceWSID))
	if err != nil {
		// notest
		return err
	}
	svbCDocJoinedWorkspace.PutString(Field_Roles, args.ArgumentObject.AsString(Field_Roles))
	svbCDocJoinedWorkspace.PutBool(appdef.SystemField_IsActive, true)

	return err
}
