/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func provideViewJoinedWorkspaceIndex(app appdef.IAppDefBuilder) {
	view := app.AddView(QNameViewJoinedWorkspaceIndex)
	view.KeyBuilder().PartKeyBuilder().AddField(field_Dummy, appdef.DataKind_int32)
	view.KeyBuilder().ClustColsBuilder().AddField(Field_InvitingWorkspaceWSID, appdef.DataKind_int64)
	view.ValueBuilder().AddRefField(field_JoinedWorkspaceID, true)
}
