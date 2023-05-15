/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package collection

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/projectors"
)

func ProvideCollectionFunc(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewQueryFunctionCustomResult(
		qNameQueryCollection,
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "CollectionParams"), appdef.DefKind_Object).
			AddField(field_Schema, appdef.DataKind_string, true).
			AddField(field_ID, appdef.DataKind_RecordID, false).QName(),
		collectionResultQName,
		collectionFuncExec,
	))

	// Register collection def
	projectors.ProvideViewDef(appDefBuilder, QNameViewCollection, CollectionViewBuilderFunc)
}

func ProvideSyncProjectorFactories(appDef appdef.IAppDef) []istructs.ProjectorFactory {
	return []istructs.ProjectorFactory{
		collectionProjectorFactory(appDef),
	}
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
	projections := ProvideSyncProjectorFactories(as.AppDef())
	return actualizerFactory(actualizerConfig, projections[0], projections[1:]...)
}

func ProvideCDocFunc(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	provideQryCDoc(cfg, appDefBuilder)
}

func ProvideStateFunc(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	provideStateFunc(cfg, appDefBuilder)
}
