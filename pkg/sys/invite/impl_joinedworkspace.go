/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	sysshared "github.com/voedger/voedger/pkg/sys/shared"
)

func provideCDocJoinedWorkspace(appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddCDoc(sysshared.QNameCDocJoinedWorkspace).
		AddField(Field_Roles, appdef.DataKind_string, true).
		AddField(Field_InvitingWorkspaceWSID, appdef.DataKind_int64, true).
		AddField(sysshared.Field_WSName, appdef.DataKind_string, true)
}
