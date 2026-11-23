/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package commandprocessor

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/checkpoints/testutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestPartitionRecoveryLoggingRetainsArguments(t *testing.T) {
	require := require.New(t)
	logs := logger.StartCapture(t, logger.LogLevelVerbose)
	app := setUpRecoveryTestApp(t)
	defer func() { tearDown(app) }() // Restart replaces the cancellation function and completion channel.
	sendNoCUDCommand(t, 1, app)
	restartCmdProc(&app)
	logs.Reset()
	require.NoError(triggerAndWaitForRecovery(t, app, 1))
	logs.HasLine("stage=cp.partition_recovery.reapply", "evqname=sys.TestNoCUD",
		"woffset=", "poffset=", "args=", `\"Marker\":1`)
}

func TestRecoveryLogsDurationOnCheckpointFailure(t *testing.T) {
	require := require.New(t)
	for _, kind := range []testutils.OperationKind{testutils.GetPartition, testutils.GetWorkspace} {
		stage := "cp.partition_recovery.error"
		if kind == testutils.GetWorkspace {
			stage = "cp.workspace_recovery.error"
		}
		t.Run(stage, func(t *testing.T) {
			logs := logger.StartCapture(t, logger.LogLevelInfo)
			storage := testutils.NewCheckpointStorageMem()
			app := setUpRecoveryTestApp(t, withCheckpointStorage(storage))
			defer tearDown(app)
			expectedError := errors.New("checkpoint unavailable")
			storage.SetError(testutils.Operation{Kind: kind, AppID: app.appStructs.ClusterAppID(), PartitionID: 1, WSID: 1}, expectedError)
			if kind == testutils.GetWorkspace {
				require.NoError(triggerAndWaitForRecovery(t, app, 1))
				triggerWorkspaceRecovery(t, app, 1)
				require.ErrorIs(app.recoveryTestControl.workspaces.wait(app.ctx, workspaceManagerKey(1)), expectedError)
			} else {
				require.ErrorIs(triggerAndWaitForRecovery(t, app, 1), expectedError)
			}
			// Failures before log scanning still report elapsed time inside the message
			logs.HasLine("stage="+stage, "level=ERROR", `msg="checkpoint unavailable, duration `)
			require.Regexp(`msg="checkpoint unavailable, duration [0-9]+ms"`, logs.String())
		})
	}
}

func TestRecoveryCheckpointBasicUsage(t *testing.T) {
	require := require.New(t)
	t.Run("recovery: scn: Normal partion recovery from checkpoint: lastPLogOffset 10", func(t *testing.T) {
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

	t.Run("recovery: scn: Normal partion recovery from checkpoint: lastPLogOffset 50", func(t *testing.T) {
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

	t.Run("recovery: scn: Normal partion recovery from checkpoint: lastPLogOffset 0", func(t *testing.T) {
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

	t.Run("recovery: scn: Normal partion recovery from checkpoint: absent", func(t *testing.T) {
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

	t.Run("recovery: scn: Normal workspace recovery from checkpoint: wlogIDBearingOffset 6, lastWLogOffset 14", func(t *testing.T) {
		// | checkpoint             | lastWLogOffset | startOffset |
		// | wlogIDBearingOffset 6  | 14             | 6           |
		// Given the WLog has an ID-bearing event at offset 6 with record ID 200001
		// And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
		// And any WLog events after offset 10 are CUD-free
		// And the WLog ends at offset <lastWLogOffset>
		// lastWLogOffset = 14
		app, storage := newCheckpointCommandApp(t, 14)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)

		// And the stored workspace checkpoint is <checkpoint>
		// checkpoint = wlogIDBearingOffset 6
		storage.ForcePutWorkspaceCheckpoint(t, app.appStructs.ClusterAppID(), 1,
			checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 6})

		// And the Command processor has restarted and completed Partition recovery
		storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), partitionManagerKeyForWSID(1).partitionID,
			checkpoints.PartitionCheckpoint{LastPLogOffset: 10})
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
		// Both scans must honor valid checkpoints without falling back to the beginning
		require.Equal([]logReadFromTo{{offset: 10, count: istructs.ReadToTheEnd}},
			app.recoveryTestControl.pLogReads.values(partitionManagerKeyForWSID(1)))
		logCap.NotContains("cp.partition_recovery.checkpoint.invalid", "cp.workspace_recovery.checkpoint.invalid")
	})

	t.Run("recovery: scn: Normal workspace recovery from checkpoint: wlogIDBearingOffset 10, lastWLogOffset 14", func(t *testing.T) {
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

	t.Run("recovery: scn: Normal workspace recovery from checkpoint: wlogIDBearingOffset 10, lastWLogOffset 10", func(t *testing.T) {
		// | checkpoint             | lastWLogOffset | startOffset |
		// | wlogIDBearingOffset 10 | 10             | 10          |
		// Given the WLog has an ID-bearing event at offset 6 with record ID 200001
		// And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
		// And any WLog events after offset 10 are CUD-free
		// And the WLog ends at offset <lastWLogOffset>
		// lastWLogOffset = 10
		app, storage := newCheckpointCommandApp(t, 10)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)

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
		// Both scans must honor valid checkpoints without falling back to the beginning
		require.Equal([]logReadFromTo{{offset: 50, count: istructs.ReadToTheEnd}},
			app.recoveryTestControl.pLogReads.values(partitionManagerKeyForWSID(1)))
		logCap.NotContains("cp.partition_recovery.checkpoint.invalid", "cp.workspace_recovery.checkpoint.invalid")
	})

	t.Run("recovery: scn: Normal workspace recovery from checkpoint: wlogIDBearingOffset 0, lastWLogOffset 14", func(t *testing.T) {
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

	t.Run("recovery: scn: Normal workspace recovery from checkpoint: absent, lastWLogOffset 14", func(t *testing.T) {
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

	t.Run("recovery: scn: The recovery checkpoint projector overwrites the initial recovery checkpoint without changing numbers issued by the next insert: tail recovery", func(t *testing.T) {
		const (
			projectedCheckpoint = istructs.Offset(200)
			pLogTail            = istructs.Offset(250)
		)

		// Given the PLog contains events through offset 250
		// And workspace "Orders" in that partition has last WLog offset 14 and last allocated ordinary record ID 200005
		app, storage := newCheckpointCommandAppWithPLogTail(t, 14, pLogTail)
		partitionKey := partitionManagerKeyForWSID(1)
		// The checkpoint-writes layer establishes the projector result consumed by recovery.
		storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), partitionKey.partitionID,
			checkpoints.PartitionCheckpoint{LastPLogOffset: projectedCheckpoint})

		// When the following lifecycle occurs in order:
		// | order | action                                                                                             |
		// | 2     | VAOperator restarts the VVM                                                                        |
		// | 3     | VAOperator inserts one ordinary record into "Orders", retrying "503 Service Unavailable" responses |
		restartCmdProc(app)
		app.recoveryTestControl.pLogReads.reset(partitionKey)
		app.recoveryTestControl.wLogReads.reset(workspaceManagerKey(1))
		response := sendCUDWithBody(t, 1, *app, emptyRecoveryInsert)

		// And Partition recovery calls ReadPLog once from offset 200 through the log end
		require.Equal([]logReadFromTo{{offset: projectedCheckpoint, count: istructs.ReadToTheEnd}},
			app.recoveryTestControl.pLogReads.values(partitionKey))
		// And the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
		requireIssuedNumbers(t, response, 15, 200006)
		require.Len(response["NewIDs"], 1)
		// And the insert writes the next PLog event at offset 251
		require.Equal(istructs.Offset(251), nextPLogOffset(t, app.appStructs, partitionKey.partitionID)-1)
	})
}

func newCheckpointCommandApp(t *testing.T, lastWLogOffset istructs.Offset) (*testApp, *testutils.CheckpointStorage) {
	t.Helper()
	return newCheckpointCommandAppWithPLogTail(t, lastWLogOffset, 50)
}

func newCheckpointCommandAppWithPLogTail(t *testing.T, lastWLogOffset,
	pLogTail istructs.Offset) (*testApp, *testutils.CheckpointStorage) {
	t.Helper()
	require := require.New(t)
	storage := testutils.NewCheckpointStorageMem()
	app := setUpRecoveryTestApp(t, withCheckpointStorage(storage), withTestWorkspaces(1, 3))
	t.Cleanup(func() { tearDown(app) })
	prepareRecoveryCheckpointWLog(t, app, 1, lastWLogOffset)
	key := partitionManagerKeyForWSID(1)
	nextOffset := nextPLogOffset(t, app.appStructs, key.partitionID)
	require.LessOrEqual(nextOffset, pLogTail+1)
	for nextOffset <= pLogTail {
		sendNoCUDCommand(t, 3, app)
		nextOffset++
	}
	require.Equal(pLogTail+1, nextPLogOffset(t, app.appStructs, key.partitionID))
	return &app, storage
}
