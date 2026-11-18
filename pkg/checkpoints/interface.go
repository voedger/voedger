/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package checkpoints

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type IRecoveryCheckpointStorage interface {
	GetPartitionCheckpoint(appID istructs.ClusterAppID, partitionID istructs.PartitionID) (PartitionCheckpoint, bool, error)
	PutPartitionCheckpoint(appID istructs.ClusterAppID, partitionID istructs.PartitionID, checkpoint PartitionCheckpoint) error
	GetWorkspaceCheckpoint(appID istructs.ClusterAppID, wsid istructs.WSID) (WorkspaceCheckpoint, bool, error)
	PutWorkspaceCheckpoint(appID istructs.ClusterAppID, wsid istructs.WSID, checkpoint WorkspaceCheckpoint) error
}

// IInitialRecovery initializes persistent recovery checkpoints before deployment.
type IInitialRecovery interface {
	// InitializeCheckpoints returns deployable partition IDs, errors keyed by failed
	// partition ID, and a fatal application setup or context error. When the fatal
	// error is nil, the IDs and error-map keys are disjoint and cover 0..numPartitions-1.
	// An empty deployable slice is valid when every partition fails. A non-nil fatal
	// error prohibits deployment from any partial results. A present partition
	// checkpoint, including zero, marks completion and requires no reconstruction.
	InitializeCheckpoints(ctx context.Context, appQName appdef.AppQName, numPartitions istructs.NumAppPartitions) (
		partitionIDsWithInitedCheckpoints []istructs.PartitionID,
		partitionIDsWithFailedCheckpoints map[istructs.PartitionID]error, // error could be about both partition and workspace checkpoints
		err error,
	)
}
