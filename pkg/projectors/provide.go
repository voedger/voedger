/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/sys/builtin"
)

func ProvideAsyncActualizerFactory() AsyncActualizerFactory {
	return asyncActualizerFactory
}

func ProvideSyncActualizerFactory() SyncActualizerFactory {
	return syncActualizerFactory
}

func ProvideOffsetsDef(appDef appdef.IAppDefBuilder) {
	provideOffsetsDefImpl(appDef)
}

func ProvideViewDef(appDef appdef.IAppDefBuilder, qname appdef.QName, buildFunc ViewTypeBuilder) {
	provideViewDefImpl(appDef, qname, buildFunc)
}

func NewSyncActualizerFactoryFactory(actualizerFactory SyncActualizerFactory, secretReader isecrets.ISecretReader, n10nBroker in10n.IN10nBroker) func(appStructs istructs.IAppStructs, partitionID istructs.PartitionID) pipeline.ISyncOperator {
	return func(appStructs istructs.IAppStructs, partitionID istructs.PartitionID) pipeline.ISyncOperator {
		if len(appStructs.SyncProjectors()) == 0 {
			return &pipeline.NOOP{}
		}
		conf := SyncActualizerConf{
			Ctx:          context.Background(), // it is needed for sync pipeline and GMP believes it is enough
			AppStructs:   func() istructs.IAppStructs { return appStructs },
			SecretReader: secretReader,
			Partition:    partitionID,
			WorkToEvent: func(work interface{}) istructs.IPLogEvent {
				return work.(interface{ Event() istructs.IPLogEvent }).Event()
			},
			N10nFunc: func(view appdef.QName, wsid istructs.WSID, offset istructs.Offset) {
				n10nBroker.Update(in10n.ProjectionKey{
					App:        appStructs.AppQName(),
					Projection: view,
					WS:         wsid,
				}, offset)
			},
			IntentsLimit: builtin.MaxCUDs,
		}
		return actualizerFactory(conf, appStructs.SyncProjectors())
	}
}
