/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func provideViewInviteIndex(app appdef.IAppDefBuilder) {
	view := app.AddView(qNameViewInviteIndex)
	view.KeyBuilder().PartKeyBuilder().AddField(field_Dummy, appdef.DataKind_int32)
	view.KeyBuilder().ClustColsBuilder().AddField(Field_Login, appdef.DataKind_string)
	view.ValueBuilder().AddRefField(field_InviteID, true)
}
