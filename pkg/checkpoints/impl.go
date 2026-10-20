/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package checkpoints

import (
	"github.com/voedger/voedger/pkg/istructs"
)

func applyRecoveryCheckpoint(storage IRecoveryCheckpointStorage) func(istructs.IPLogEvent, istructs.IState, istructs.IIntents) error {
	return func(event istructs.IPLogEvent, state istructs.IState, _ istructs.IIntents) error {
		clusterAppID := state.AppStructs().ClusterAppID()
		if event.PLogOffset()%partitionCheckpointUpdateInterval == 0 {
			if err := storage.PutPartitionCheckpoint(clusterAppID, event.HandlingPartition(), PartitionCheckpoint{
				LastPLogOffset: event.PLogOffset(),
			}); err != nil {
				return err
			}
		}

		if !IsIDBearingEvent(event) {
			return nil
		}
		return storage.PutWorkspaceCheckpoint(clusterAppID, event.Workspace(), WorkspaceCheckpoint{
			WLogIDBearingOffset: event.WLogOffset(),
		})
	}
}

func IsIDBearingEvent(event istructs.IAbstractEvent) bool {
	containsNewRecordIDs := false
	event.CUDs(func(row istructs.ICUDRow) bool {
		containsNewRecordIDs = row.IsNew() && row.ID() >= istructs.FirstUserRecordID
		return !containsNewRecordIDs
	})
	return containsNewRecordIDs
}
