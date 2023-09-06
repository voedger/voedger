/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func provideViewJoinedWorkspaceIndex(app appdef.IAppDefBuilder) {
	view := app.AddView(QNameViewJoinedWorkspaceIndex)
	view.Key().Partition().AddField(field_Dummy, appdef.DataKind_int32)
	view.Key().ClustCols().AddField(Field_InvitingWorkspaceWSID, appdef.DataKind_int64)
	view.Value().AddRefField(field_JoinedWorkspaceID, true)
}
