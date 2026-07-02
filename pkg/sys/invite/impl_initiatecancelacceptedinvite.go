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

func provideCmdInitiateCancelAcceptedInvite(sr istructsmem.IStatelessResources, time timeu.ITime) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(
		qNameCmdInitiateCancelAcceptedInvite,
		execCmdInitiateCancelAcceptedInvite(time),
	))
}

func execCmdInitiateCancelAcceptedInvite(_ timeu.ITime) func(args istructs.ExecCommandArgs) (err error) {
	return execCmdCancelInvite(qNameCmdInitiateCancelAcceptedInvite)
}
