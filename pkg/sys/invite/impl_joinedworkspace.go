/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func provideCDocJoinedWorkspace(appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddCDoc(qNameCDocJoinedWorkspace).
		AddField(Field_Roles, appdef.DataKind_string, true).
		AddField(field_InvitingWorkspaceWSID, appdef.DataKind_int64, true).
		AddField(field_WSName, appdef.DataKind_string, true)
}
