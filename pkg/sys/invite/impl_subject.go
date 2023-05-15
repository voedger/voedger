/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func provideCDocSubject(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddStruct(QNameCDocSubject, appdef.DefKind_CDoc).
		AddField(Field_Login, appdef.DataKind_string, true).
		AddField(Field_SubjectKind, appdef.DataKind_int32, true).
		AddField(Field_Roles, appdef.DataKind_string, true).
		AddField(Field_ProfileWSID, appdef.DataKind_int64, false)
	cfg.Uniques.Add(QNameCDocSubject, []string{Field_Login})
}
