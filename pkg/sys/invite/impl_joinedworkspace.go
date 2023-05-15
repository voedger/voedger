/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func provideCDocJoinedWorkspace(appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddStruct(QNameCDocJoinedWorkspace, appdef.DefKind_CDoc).
		AddField(Field_Roles, appdef.DataKind_string, true).
		AddField(Field_InvitingWorkspaceWSID, appdef.DataKind_int64, true).
		AddField(Field_WSName, appdef.DataKind_string, true)
}
