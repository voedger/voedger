/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func provideCmdCancelSentInvite(sr istructsmem.IStatelessResources, time timeu.ITime) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(
		qNameCmdCancelSentInvite,
		execCmdCancelSentInvite(time),
	))
}

func execCmdCancelSentInvite(_ timeu.ITime) func(args istructs.ExecCommandArgs) (err error) {
	return execCmdCancelInvite(qNameCmdCancelSentInvite)
}
