/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package collection

import (
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(cfg *istructsmem.AppConfigType) {
	cfg.Resources.Add(istructsmem.NewQueryFunctionCustomResult(
		qNameQueryCollection,
		collectionResultQName,
		collectionFuncExec,
	))

	provideQryCDoc(cfg)
	provideStateFunc(cfg)

	cfg.AddSyncProjectors(collectionProjector)
}
