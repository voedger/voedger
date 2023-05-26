/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func provideViewJoinedWorkspaceIndex(appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddView(QNameViewJoinedWorkspaceIndex).
		AddPartField(field_Dummy, appdef.DataKind_int32).
		AddClustColumn(Field_InvitingWorkspaceWSID, appdef.DataKind_int64).
		AddValueField(field_JoinedWorkspaceID, appdef.DataKind_RecordID, true)
}
