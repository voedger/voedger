/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package collection

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/projectors"
)

func Provide(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewQueryFunctionCustomResult(
		qNameQueryCollection,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "CollectionParams")).
			AddField(field_Schema, appdef.DataKind_string, true).
			AddField(field_ID, appdef.DataKind_RecordID, false).(appdef.IType).QName(),
		collectionResultQName,
		collectionFuncExec,
	))

	// Register collection def
	projectors.ProvideViewDef(appDefBuilder, QNameViewCollection, CollectionViewBuilderFunc)

	provideQryCDoc(cfg, appDefBuilder)
	provideStateFunc(cfg, appDefBuilder)

	cfg.AddSyncProjectors(collectionProjectorFactory(appDefBuilder))
}

// should be used in tests only. Sync Actualizer per app will be wired in production
func provideSyncActualizer(ctx context.Context, as istructs.IAppStructs, partitionID istructs.PartitionID) pipeline.ISyncOperator {
	actualizerConfig := projectors.SyncActualizerConf{
		Ctx:        ctx,
		AppStructs: func() istructs.IAppStructs { return as },
		Partition:  partitionID,
		N10nFunc:   func(view appdef.QName, wsid istructs.WSID, offset istructs.Offset) {},
	}
	actualizerFactory := projectors.ProvideSyncActualizerFactory()
	return actualizerFactory(actualizerConfig, collectionProjectorFactory(as.AppDef()))
}
