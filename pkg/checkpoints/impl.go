/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package checkpoints

import (
	"github.com/voedger/voedger/pkg/istructs"
)

func ApplyRecoveryCheckpoint(storage IRecoveryCheckpointStorage) func(istructs.IPLogEvent, istructs.IState, istructs.IIntents) error {
	return func(event istructs.IPLogEvent, state istructs.IState, _ istructs.IIntents) error {
		clusterAppID := state.AppStructs().ClusterAppID()
		if event.PLogOffset()%partitionCheckpointUpdateInterval == 0 {
			if err := storage.PutPartitionCheckpoint(clusterAppID, event.HandlingPartition(), PartitionCheckpoint{
				LastPLogOffset: event.PLogOffset(),
			}); err != nil {
				return err
			}
		}

		if !canAdvanceWorkspaceCheckpoint(event) {
			return nil
		}
		return storage.PutWorkspaceCheckpoint(clusterAppID, event.Workspace(), WorkspaceCheckpoint{
			WLogIDBearingOffset: event.WLogOffset(),
		})
	}
}

// Locally generated IDs exceed all previously allocated IDs. Synchronized
// inserts can carry lower IDs, so advancing past earlier events would lose
// coverage of the allocator's maximum. Keep scanning from the previous checkpoint.
func canAdvanceWorkspaceCheckpoint(event istructs.IAbstractEvent) bool {
	return IsIDBearingEvent(event) && !event.Synced()
}

func IsIDBearingEvent(event istructs.IAbstractEvent) bool {
	return HighestNewRecordID(event) != istructs.NullRecordID
}

// HighestNewRecordID returns the greatest ordinary ID allocated by the event,
// including its logged argument tree. Singleton IDs and CUD updates are excluded.
func HighestNewRecordID(event istructs.IAbstractEvent) istructs.RecordID {
	highest := highestArgumentRecordID(event.ArgumentObject())
	for row := range event.CUDs {
		if row.IsNew() && row.ID() >= istructs.FirstUserRecordID {
			highest = max(highest, row.ID())
		}
	}
	return highest
}

func highestArgumentRecordID(object istructs.IObject) istructs.RecordID {
	highest := istructs.NullRecordID
	if id := object.AsRecord().ID(); id >= istructs.FirstUserRecordID {
		highest = id
	}
	for child := range object.Children() {
		highest = max(highest, highestArgumentRecordID(child))
	}
	return highest
}
