/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func provideCmdUpdateJoinedWorkspaceRoles(cfg *istructsmem.AppConfigType) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdUpdateJoinedWorkspaceRoles,
		appdef.NullQName,
		appdef.NullQName,
		appdef.NullQName,
		execCmdUpdateJoinedWorkspaceRoles,
	))
}

func execCmdUpdateJoinedWorkspaceRoles(args istructs.ExecCommandArgs) (err error) {
	svbCDocJoinedWorkspace, err := GetCDocJoinedWorkspaceForUpdateRequired(args.State, args.Intents, args.ArgumentObject.AsInt64(Field_InvitingWorkspaceWSID))
	if err != nil {
		// notest
		return err
	}
	svbCDocJoinedWorkspace.PutString(Field_Roles, args.ArgumentObject.AsString(Field_Roles))
	svbCDocJoinedWorkspace.PutBool(appdef.SystemField_IsActive, true)

	return err
}
