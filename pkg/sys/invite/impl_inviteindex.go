/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func provideViewInviteIndex(app appdef.IAppDefBuilder) {
	view := app.AddView(qNameViewInviteIndex)
	view.Key().Partition().AddField(field_Dummy, appdef.DataKind_int32)
	view.Key().ClustCols().AddStringField(Field_Login, appdef.DefaultFieldMaxLength)
	view.Value().AddRefField(field_InviteID, true)
}
