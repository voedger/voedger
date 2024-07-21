/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package collection

import (
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(sprb istructsmem.IStatelessPkgResourcesBuilder) {
	sprb.AddFunc(istructsmem.NewQueryFunctionCustomResult(
		qNameQueryCollection,
		collectionResultQName,
		collectionFuncExec,
	))

	provideQryCDoc(sprb)
	provideStateFunc(sprb)

	sprb.AddSyncProjectors(collectionProjector)
}
