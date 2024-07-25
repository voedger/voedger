/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package collection

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(sr istructsmem.IStatelessResources) {
	sr.AddQueries(appdef.SysPackagePath, istructsmem.NewQueryFunctionCustomResult(
		qNameQueryCollection,
		collectionResultQName,
		collectionFuncExec,
	))

	provideQryCDoc(sr)
	provideStateFunc(sr)

	sr.AddProjectors(appdef.SysPackagePath, collectionProjector)
}
