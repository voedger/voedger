package ap

import "github.com/voedger/voedger/pkg/istructs"

type ActualizerID string

// Actualizer Processor
type IActualizersP interface {
	DeployActualizers(app istructs.AppQName, part istructs.PartitionID, map[ActualizerID]pipeline.I) error
}
