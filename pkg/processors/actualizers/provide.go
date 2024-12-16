/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package actualizers

import (
	"context"
	"maps"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/pipeline"
)

func ProvideActualizers(cfg BasicAsyncActualizerConfig) IActualizersService {
	return newActualizers(cfg)
}

func ProvideSyncActualizerFactory() SyncActualizerFactory {
	return syncActualizerFactory
}

func ProvideViewDef(wsb appdef.IWorkspaceBuilder, qname appdef.QName, buildFunc ViewTypeBuilder) {
	provideViewDefImpl(wsb, qname, buildFunc)
}

func NewSyncActualizerFactoryFactory(actualizerFactory SyncActualizerFactory, secretReader isecrets.ISecretReader,
	n10nBroker in10n.IN10nBroker, statelessResources istructsmem.IStatelessResources) func(appStructs istructs.IAppStructs, partitionID istructs.PartitionID) pipeline.ISyncOperator {
	return func(appStructs istructs.IAppStructs, partitionID istructs.PartitionID) pipeline.ISyncOperator {
		projectors := maps.Clone(appStructs.SyncProjectors())
		for _, projector := range statelessResources.Projectors {
			if appdef.Projector(appStructs.AppDef().Type, projector.Name).Sync() {
				projectors[projector.Name] = projector
			}
		}
		if len(projectors) == 0 {
			return &pipeline.NOOP{}
		}
		conf := SyncActualizerConf{
			Ctx:          context.Background(), // it is needed for sync pipeline and GMP believes it is enough
			SecretReader: secretReader,
			Partition:    partitionID,
			N10nFunc: func(view appdef.QName, wsid istructs.WSID, offset istructs.Offset) {
				n10nBroker.Update(in10n.ProjectionKey{
					App:        appStructs.AppQName(),
					Projection: view,
					WS:         wsid,
				}, offset)
			},
			IntentsLimit: DefaultIntentsLimit,
		}

		return actualizerFactory(conf, projectors)
	}
}
