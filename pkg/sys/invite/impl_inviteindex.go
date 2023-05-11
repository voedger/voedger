/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func provideViewInviteIndex(appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddView(qNameViewInviteIndex).
		AddPartField(field_Dummy, appdef.DataKind_int32).
		AddClustColumn(Field_Login, appdef.DataKind_string).
		AddValueField(field_InviteID, appdef.DataKind_RecordID, true)
}
