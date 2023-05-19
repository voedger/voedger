/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem"
	sysshared "github.com/voedger/voedger/pkg/sys/shared"
)

func provideCDocSubject(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddStruct(sysshared.QNameCDocSubject, appdef.DefKind_CDoc).
		AddField(Field_Login, appdef.DataKind_string, true).
		AddField(sysshared.Field_SubjectKind, appdef.DataKind_int32, true).
		AddField(Field_Roles, appdef.DataKind_string, true).
		AddField(sysshared.Field_ProfileWSID, appdef.DataKind_int64, true)
	cfg.Uniques.Add(sysshared.QNameCDocSubject, []string{Field_Login})
}
