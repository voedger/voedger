/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func provideCDocInvite(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	doc := appDefBuilder.AddCDoc(qNameCDocInvite)
	doc.
		AddField(Field_SubjectKind, appdef.DataKind_int32, false).
		AddField(Field_Login, appdef.DataKind_string, true).
		AddField(field_Email, appdef.DataKind_string, true).
		AddField(Field_Roles, appdef.DataKind_string, false).
		AddField(field_ExpireDatetime, appdef.DataKind_int64, false).
		AddField(field_VerificationCode, appdef.DataKind_string, false).
		AddField(field_State, appdef.DataKind_int32, true).
		AddField(field_Created, appdef.DataKind_int64, false).
		AddField(field_Updated, appdef.DataKind_int64, true).
		AddField(field_SubjectID, appdef.DataKind_RecordID, false).
		AddField(field_InviteeProfileWSID, appdef.DataKind_int64, false)
	doc.SetUniqueField(field_Email)
}
