/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package commandprocessor

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/checkpoints/testutils"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestRecoveryCheckpointBasicUsage(t *testing.T) {
	t.Run("recovery: scn: Partition recovery starts from a checkpoint in basic usage: lastPLogOffset 10", func(t *testing.T) {
		require := require.New(t)
		// | checkpoint        | startOffset |
		// | lastPLogOffset 10 | 10          |
		// Given the PLog contains events through offset 50
		// And workspace 1 has last WLog offset 14 and last allocated ordinary record ID 200005
		app, storage := newCheckpointCommandApp(t, 14)

		// And the stored partition checkpoint is <checkpoint>
		// checkpoint = lastPLogOffset 10
		storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), partitionManagerKeyForWSID(1).partitionID,
			checkpoints.PartitionCheckpoint{LastPLogOffset: 10})

		// And the Command processor has restarted
		restartCmdProc(app)
		app.recoveryTestControl.pLogReads.reset(partitionManagerKeyForWSID(1))
		app.recoveryTestControl.wLogReads.reset(workspaceManagerKey(1))

		// When VAOperator submits an insert into the unrecovered partition, retrying "503 Service Unavailable" responses
		response := sendCUDWithBody(t, 1, *app, emptyRecoveryInsert)
		// Then Partition recovery calls ReadPLog once from offset <startOffset> through the log end
		// startOffset = 10
		require.Equal([]logReadFromTo{{offset: 10, count: istructs.ReadToTheEnd}},
			app.recoveryTestControl.pLogReads.values(partitionManagerKeyForWSID(1)))
		// And the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
		requireIssuedNumbers(t, response, 14+1, 200006)
		require.Len(response["NewIDs"], 1)
	})

	t.Run("recovery: scn: Partition recovery starts from a checkpoint in basic usage: lastPLogOffset 50", func(t *testing.T) {
		require := require.New(t)
		// | checkpoint        | startOffset |
		// | lastPLogOffset 50 | 50          |
		// Given the PLog contains events through offset 50
		// And workspace 1 has last WLog offset 14 and last allocated ordinary record ID 200005
		app, storage := newCheckpointCommandApp(t, 14)

		// And the stored partition checkpoint is <checkpoint>
		// checkpoint = lastPLogOffset 50
		storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), partitionManagerKeyForWSID(1).partitionID,
			checkpoints.PartitionCheckpoint{LastPLogOffset: 50})

		// And the Command processor has restarted
		restartCmdProc(app)
		app.recoveryTestControl.pLogReads.reset(partitionManagerKeyForWSID(1))
		app.recoveryTestControl.wLogReads.reset(workspaceManagerKey(1))

		// When VAOperator submits an insert into the unrecovered partition, retrying "503 Service Unavailable" responses
		response := sendCUDWithBody(t, 1, *app, emptyRecoveryInsert)
		// Then Partition recovery calls ReadPLog once from offset <startOffset> through the log end
		// startOffset = 50
		require.Equal([]logReadFromTo{{offset: 50, count: istructs.ReadToTheEnd}},
			app.recoveryTestControl.pLogReads.values(partitionManagerKeyForWSID(1)))
		// And the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
		requireIssuedNumbers(t, response, 14+1, 200006)
		require.Len(response["NewIDs"], 1)
	})

	t.Run("recovery: scn: Partition recovery starts from a checkpoint in basic usage: lastPLogOffset 0", func(t *testing.T) {
		require := require.New(t)
		// | checkpoint        | startOffset |
		// | lastPLogOffset 0  | 1           |
		// Given the PLog contains events through offset 50
		// And workspace 1 has last WLog offset 14 and last allocated ordinary record ID 200005
		app, storage := newCheckpointCommandApp(t, 14)

		// And the stored partition checkpoint is <checkpoint>
		// checkpoint = lastPLogOffset 0
		storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), partitionManagerKeyForWSID(1).partitionID,
			checkpoints.PartitionCheckpoint{LastPLogOffset: 0})

		// And the Command processor has restarted
		restartCmdProc(app)
		app.recoveryTestControl.pLogReads.reset(partitionManagerKeyForWSID(1))
		app.recoveryTestControl.wLogReads.reset(workspaceManagerKey(1))

		// When VAOperator submits an insert into the unrecovered partition, retrying "503 Service Unavailable" responses
		response := sendCUDWithBody(t, 1, *app, emptyRecoveryInsert)
		// Then Partition recovery calls ReadPLog once from offset <startOffset> through the log end
		// startOffset = 1
		require.Equal([]logReadFromTo{{offset: 1, count: istructs.ReadToTheEnd}},
			app.recoveryTestControl.pLogReads.values(partitionManagerKeyForWSID(1)))
		// And the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
		requireIssuedNumbers(t, response, 14+1, 200006)
		require.Len(response["NewIDs"], 1)
	})

	t.Run("recovery: scn: Partition recovery starts from a checkpoint in basic usage: absent", func(t *testing.T) {
		require := require.New(t)
		// | checkpoint        | startOffset |
		// | absent            | 1           |
		// Given the PLog contains events through offset 50
		// And workspace 1 has last WLog offset 14 and last allocated ordinary record ID 200005
		app, storage := newCheckpointCommandApp(t, 14)

		// And the stored partition checkpoint is <checkpoint>
		// checkpoint = absent
		_, exists, err := storage.GetPartitionCheckpoint(app.appStructs.ClusterAppID(), partitionManagerKeyForWSID(1).partitionID)
		require.NoError(err)
		require.False(exists)

		// And the Command processor has restarted
		restartCmdProc(app)
		app.recoveryTestControl.pLogReads.reset(partitionManagerKeyForWSID(1))
		app.recoveryTestControl.wLogReads.reset(workspaceManagerKey(1))

		// When VAOperator submits an insert into the unrecovered partition, retrying "503 Service Unavailable" responses
		response := sendCUDWithBody(t, 1, *app, emptyRecoveryInsert)
		// Then Partition recovery calls ReadPLog once from offset <startOffset> through the log end
		// startOffset = 1
		require.Equal([]logReadFromTo{{offset: 1, count: istructs.ReadToTheEnd}},
			app.recoveryTestControl.pLogReads.values(partitionManagerKeyForWSID(1)))
		// And the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
		requireIssuedNumbers(t, response, 14+1, 200006)
		require.Len(response["NewIDs"], 1)
	})

	t.Run("recovery: scn: Workspace recovery starts from a checkpoint in basic usage: wlogIDBearingOffset 6, lastWLogOffset 14", func(t *testing.T) {
		require := require.New(t)
		// | checkpoint             | lastWLogOffset | startOffset |
		// | wlogIDBearingOffset 6  | 14             | 6           |
		// Given the WLog has an ID-bearing event at offset 6 with record ID 200001
		// And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
		// And any WLog events after offset 10 are CUD-free
		// And the WLog ends at offset <lastWLogOffset>
		// lastWLogOffset = 14
		app, storage := newCheckpointCommandApp(t, 14)

		// And the stored workspace checkpoint is <checkpoint>
		// checkpoint = wlogIDBearingOffset 6
		storage.ForcePutWorkspaceCheckpoint(t, app.appStructs.ClusterAppID(), 1,
			checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 6})

		// And the Command processor has restarted and completed Partition recovery
		storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), partitionManagerKeyForWSID(1).partitionID,
			checkpoints.PartitionCheckpoint{LastPLogOffset: 50})
		restartCmdProc(app)
		app.recoveryTestControl.pLogReads.reset(partitionManagerKeyForWSID(1))
		app.recoveryTestControl.wLogReads.reset(workspaceManagerKey(1))
		status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(1, *app))
		require.NoError(err)
		require.Equal(http.StatusServiceUnavailable, status)
		require.NoError(app.recoveryTestControl.partitions.wait(app.ctx, partitionManagerKeyForWSID(1)))

		// When VAOperator submits an insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
		response := sendCUDWithBody(t, 1, *app, emptyRecoveryInsert)
		// Then Workspace recovery calls ReadWLog once from offset <startOffset> through the log end
		// startOffset = 6
		require.Equal([]logReadFromTo{{offset: 6, count: istructs.ReadToTheEnd}},
			app.recoveryTestControl.wLogReads.values(workspaceManagerKey(1)))
		// And the insert returns "200 OK" with CurrentWLogOffset <lastWLogOffset> + 1 and new ordinary record ID 200006
		// lastWLogOffset = 14
		requireIssuedNumbers(t, response, 14+1, 200006)
		require.Len(response["NewIDs"], 1)
	})

	t.Run("recovery: scn: Workspace recovery starts from a checkpoint in basic usage: wlogIDBearingOffset 10, lastWLogOffset 14", func(t *testing.T) {
		require := require.New(t)
		// | checkpoint             | lastWLogOffset | startOffset |
		// | wlogIDBearingOffset 10 | 14             | 10          |
		// Given the WLog has an ID-bearing event at offset 6 with record ID 200001
		// And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
		// And any WLog events after offset 10 are CUD-free
		// And the WLog ends at offset <lastWLogOffset>
		// lastWLogOffset = 14
		app, storage := newCheckpointCommandApp(t, 14)

		// And the stored workspace checkpoint is <checkpoint>
		// checkpoint = wlogIDBearingOffset 10
		storage.ForcePutWorkspaceCheckpoint(t, app.appStructs.ClusterAppID(), 1,
			checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 10})

		// And the Command processor has restarted and completed Partition recovery
		storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), partitionManagerKeyForWSID(1).partitionID,
			checkpoints.PartitionCheckpoint{LastPLogOffset: 50})
		restartCmdProc(app)
		app.recoveryTestControl.pLogReads.reset(partitionManagerKeyForWSID(1))
		app.recoveryTestControl.wLogReads.reset(workspaceManagerKey(1))
		status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(1, *app))
		require.NoError(err)
		require.Equal(http.StatusServiceUnavailable, status)
		require.NoError(app.recoveryTestControl.partitions.wait(app.ctx, partitionManagerKeyForWSID(1)))

		// When VAOperator submits an insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
		response := sendCUDWithBody(t, 1, *app, emptyRecoveryInsert)
		// Then Workspace recovery calls ReadWLog once from offset <startOffset> through the log end
		// startOffset = 10
		require.Equal([]logReadFromTo{{offset: 10, count: istructs.ReadToTheEnd}},
			app.recoveryTestControl.wLogReads.values(workspaceManagerKey(1)))
		// And the insert returns "200 OK" with CurrentWLogOffset <lastWLogOffset> + 1 and new ordinary record ID 200006
		// lastWLogOffset = 14
		requireIssuedNumbers(t, response, 14+1, 200006)
		require.Len(response["NewIDs"], 1)
	})

	t.Run("recovery: scn: Workspace recovery starts from a checkpoint in basic usage: wlogIDBearingOffset 10, lastWLogOffset 10", func(t *testing.T) {
		require := require.New(t)
		// | checkpoint             | lastWLogOffset | startOffset |
		// | wlogIDBearingOffset 10 | 10             | 10          |
		// Given the WLog has an ID-bearing event at offset 6 with record ID 200001
		// And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
		// And any WLog events after offset 10 are CUD-free
		// And the WLog ends at offset <lastWLogOffset>
		// lastWLogOffset = 10
		app, storage := newCheckpointCommandApp(t, 10)

		// And the stored workspace checkpoint is <checkpoint>
		// checkpoint = wlogIDBearingOffset 10
		storage.ForcePutWorkspaceCheckpoint(t, app.appStructs.ClusterAppID(), 1,
			checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 10})

		// And the Command processor has restarted and completed Partition recovery
		storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), partitionManagerKeyForWSID(1).partitionID,
			checkpoints.PartitionCheckpoint{LastPLogOffset: 50})
		restartCmdProc(app)
		app.recoveryTestControl.pLogReads.reset(partitionManagerKeyForWSID(1))
		app.recoveryTestControl.wLogReads.reset(workspaceManagerKey(1))
		status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(1, *app))
		require.NoError(err)
		require.Equal(http.StatusServiceUnavailable, status)
		require.NoError(app.recoveryTestControl.partitions.wait(app.ctx, partitionManagerKeyForWSID(1)))

		// When VAOperator submits an insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
		response := sendCUDWithBody(t, 1, *app, emptyRecoveryInsert)
		// Then Workspace recovery calls ReadWLog once from offset <startOffset> through the log end
		// startOffset = 10
		require.Equal([]logReadFromTo{{offset: 10, count: istructs.ReadToTheEnd}},
			app.recoveryTestControl.wLogReads.values(workspaceManagerKey(1)))
		// And the insert returns "200 OK" with CurrentWLogOffset <lastWLogOffset> + 1 and new ordinary record ID 200006
		// lastWLogOffset = 10
		requireIssuedNumbers(t, response, 10+1, 200006)
		require.Len(response["NewIDs"], 1)
	})

	t.Run("recovery: scn: Workspace recovery starts from a checkpoint in basic usage: wlogIDBearingOffset 0, lastWLogOffset 14", func(t *testing.T) {
		require := require.New(t)
		// | checkpoint             | lastWLogOffset | startOffset |
		// | wlogIDBearingOffset 0  | 14             | 1           |
		// Given the WLog has an ID-bearing event at offset 6 with record ID 200001
		// And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
		// And any WLog events after offset 10 are CUD-free
		// And the WLog ends at offset <lastWLogOffset>
		// lastWLogOffset = 14
		app, storage := newCheckpointCommandApp(t, 14)

		// And the stored workspace checkpoint is <checkpoint>
		// checkpoint = wlogIDBearingOffset 0
		storage.ForcePutWorkspaceCheckpoint(t, app.appStructs.ClusterAppID(), 1,
			checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 0})

		// And the Command processor has restarted and completed Partition recovery
		storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), partitionManagerKeyForWSID(1).partitionID,
			checkpoints.PartitionCheckpoint{LastPLogOffset: 50})
		restartCmdProc(app)
		app.recoveryTestControl.pLogReads.reset(partitionManagerKeyForWSID(1))
		app.recoveryTestControl.wLogReads.reset(workspaceManagerKey(1))
		status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(1, *app))
		require.NoError(err)
		require.Equal(http.StatusServiceUnavailable, status)
		require.NoError(app.recoveryTestControl.partitions.wait(app.ctx, partitionManagerKeyForWSID(1)))

		// When VAOperator submits an insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
		response := sendCUDWithBody(t, 1, *app, emptyRecoveryInsert)
		// Then Workspace recovery calls ReadWLog once from offset <startOffset> through the log end
		// startOffset = 1
		require.Equal([]logReadFromTo{{offset: 1, count: istructs.ReadToTheEnd}},
			app.recoveryTestControl.wLogReads.values(workspaceManagerKey(1)))
		// And the insert returns "200 OK" with CurrentWLogOffset <lastWLogOffset> + 1 and new ordinary record ID 200006
		// lastWLogOffset = 14
		requireIssuedNumbers(t, response, 14+1, 200006)
		require.Len(response["NewIDs"], 1)
	})

	t.Run("recovery: scn: Workspace recovery starts from a checkpoint in basic usage: absent, lastWLogOffset 14", func(t *testing.T) {
		require := require.New(t)
		// | checkpoint             | lastWLogOffset | startOffset |
		// | absent                 | 14             | 1           |
		// Given the WLog has an ID-bearing event at offset 6 with record ID 200001
		// And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
		// And any WLog events after offset 10 are CUD-free
		// And the WLog ends at offset <lastWLogOffset>
		// lastWLogOffset = 14
		app, storage := newCheckpointCommandApp(t, 14)

		// And the stored workspace checkpoint is <checkpoint>
		// checkpoint = absent
		_, exists, err := storage.GetWorkspaceCheckpoint(app.appStructs.ClusterAppID(), 1)
		require.NoError(err)
		require.False(exists)

		// And the Command processor has restarted and completed Partition recovery
		storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), partitionManagerKeyForWSID(1).partitionID,
			checkpoints.PartitionCheckpoint{LastPLogOffset: 50})
		restartCmdProc(app)
		app.recoveryTestControl.pLogReads.reset(partitionManagerKeyForWSID(1))
		app.recoveryTestControl.wLogReads.reset(workspaceManagerKey(1))
		status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(1, *app))
		require.NoError(err)
		require.Equal(http.StatusServiceUnavailable, status)
		require.NoError(app.recoveryTestControl.partitions.wait(app.ctx, partitionManagerKeyForWSID(1)))

		// When VAOperator submits an insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
		response := sendCUDWithBody(t, 1, *app, emptyRecoveryInsert)
		// Then Workspace recovery calls ReadWLog once from offset <startOffset> through the log end
		// startOffset = 1
		require.Equal([]logReadFromTo{{offset: 1, count: istructs.ReadToTheEnd}},
			app.recoveryTestControl.wLogReads.values(workspaceManagerKey(1)))
		// And the insert returns "200 OK" with CurrentWLogOffset <lastWLogOffset> + 1 and new ordinary record ID 200006
		// lastWLogOffset = 14
		requireIssuedNumbers(t, response, 14+1, 200006)
		require.Len(response["NewIDs"], 1)
	})

}

func newCheckpointCommandApp(t *testing.T, lastWLogOffset istructs.Offset) (*testApp, *testutils.CheckpointStorage) {
	t.Helper()
	require := require.New(t)
	storage := testutils.NewCheckpointStorageMem()
	app := setUpRecoveryTestApp(t, withCheckpointStorage(storage), withTestWorkspaces(1, 3))
	t.Cleanup(func() { tearDown(app) })
	prepareRecoveryCheckpointWLog(t, app, 1, lastWLogOffset)
	key := partitionManagerKeyForWSID(1)
	for range 51 - nextPLogOffset(t, app.appStructs, key.partitionID) {
		sendNoCUDCommand(t, 3, app)
	}
	require.Equal(istructs.Offset(51), nextPLogOffset(t, app.appStructs, key.partitionID))
	return &app, storage
}
