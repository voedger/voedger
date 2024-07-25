/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func provideCmdDeactivateJoinedWorkspace(sr istructsmem.IStatelessResources) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(
		qNameCmdDeactivateJoinedWorkspace,
		execCmdDeactivateJoinedWorkspace,
	))
}

func execCmdDeactivateJoinedWorkspace(args istructs.ExecCommandArgs) (err error) {
	svbCDocJoinedWorkspace, err := GetCDocJoinedWorkspaceForUpdateRequired(args.State, args.Intents, args.ArgumentObject.AsInt64(Field_InvitingWorkspaceWSID))
	if err == nil {
		svbCDocJoinedWorkspace.PutBool(appdef.SystemField_IsActive, false)
	}
	return err
}
