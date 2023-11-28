/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package collection

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewQueryFunctionCustomResult(
		qNameQueryCollection,
		// local tests -> params will be used as declared here
		// runtime -> params will be replaced with ones from sql
		appdef.NullQName,
		// appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "CollectionParams_local")).
		// 	AddField(field_Schema, appdef.DataKind_string, true).
		// 	AddField(field_ID, appdef.DataKind_RecordID, false).(appdef.IType).QName(),
		collectionResultQName,
		collectionFuncExec,
	))

	provideQryCDoc(cfg, appDefBuilder)
	provideStateFunc(cfg, appDefBuilder)

	cfg.AddSyncProjectors(collectionProjectorFactory(appDefBuilder))
}
