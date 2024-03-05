package ap

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type ActualizerID string
type ProjectorQName = appdef.QName

// // Actualizer Processor
type IAsyncActualizers interface {
	// Prepare(partition istructs.PartitionID, appdef appdef.IAppDef) (actualizers map[ProjectorQName]pipeline.ISyncOperator)
	Deploy(appdef appdef.IAppDef, partition istructs.PartitionID) error
}
