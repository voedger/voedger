/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	sysshared "github.com/voedger/voedger/pkg/sys/shared"
)

func provideCDocSubject(appDefBuilder appdef.IAppDefBuilder) {
	doc := appDefBuilder.AddCDoc(sysshared.QNameCDocSubject)
	doc.
		AddField(Field_Login, appdef.DataKind_string, true).
		AddField(sysshared.Field_SubjectKind, appdef.DataKind_int32, true).
		AddField(Field_Roles, appdef.DataKind_string, true)
	doc.SetUniqueField(Field_Login)
}
