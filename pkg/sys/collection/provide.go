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
		collectionResultQName,
		collectionFuncExec,
	))

	provideQryCDoc(cfg)
	provideStateFunc(cfg, appDefBuilder)

	cfg.AddSyncProjectors(collectionProjector(appDefBuilder.AppDef()))
}
