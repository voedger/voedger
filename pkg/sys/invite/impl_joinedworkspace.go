/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func provideCDocJoinedWorkspace(appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddCDoc(QNameCDocJoinedWorkspace).
		AddField(Field_Roles, appdef.DataKind_string, true).
		AddField(Field_InvitingWorkspaceWSID, appdef.DataKind_int64, true).
		AddField(authnz.Field_WSName, appdef.DataKind_string, true)
}
