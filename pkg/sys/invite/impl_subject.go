/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func provideCDocSubject(appDefBuilder appdef.IAppDefBuilder) {
	doc := appDefBuilder.AddCDoc(QNameCDocSubject)
	doc.
		AddField(Field_Login, appdef.DataKind_string, true).
		AddField(Field_SubjectKind, appdef.DataKind_int32, true).
		AddField(Field_Roles, appdef.DataKind_string, true).
		AddField(Field_ProfileWSID, appdef.DataKind_int64, true)
	doc.SetUniqueField(Field_Login)
}
