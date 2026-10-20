/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package checkpoints

import "github.com/voedger/voedger/pkg/istructs"

type IRecoveryCheckpointStorage interface {
	GetPartitionCheckpoint(appID istructs.ClusterAppID, partitionID istructs.PartitionID) (PartitionCheckpoint, bool, error)
	PutPartitionCheckpoint(appID istructs.ClusterAppID, partitionID istructs.PartitionID, checkpoint PartitionCheckpoint) error
	GetWorkspaceCheckpoint(appID istructs.ClusterAppID, wsid istructs.WSID) (WorkspaceCheckpoint, bool, error)
	PutWorkspaceCheckpoint(appID istructs.ClusterAppID, wsid istructs.WSID, checkpoint WorkspaceCheckpoint) error
}
