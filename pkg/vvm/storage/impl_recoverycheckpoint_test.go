/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package storage

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestRecoveryCheckpointStorage(t *testing.T) {
	require := require.New(t)
	sysVVMStorage, checkpointStorage := newRecoveryCheckpointStorageForTest(t)

	const (
		appID       = istructs.ClusterAppID(101)
		partitionID = istructs.PartitionID(7)
		wsid        = istructs.WSID(7001)
	)

	t.Run("no checkpoints", func(t *testing.T) {
		partition, ok, err := checkpointStorage.GetPartitionCheckpoint(appID, partitionID)
		require.NoError(err)
		require.False(ok)
		require.Zero(partition)

		workspace, ok, err := checkpointStorage.GetWorkspaceCheckpoint(appID, wsid)
		require.NoError(err)
		require.False(ok)
		require.Zero(workspace)
	})

	t.Run("values format", func(t *testing.T) {
		partition := checkpoints.PartitionCheckpoint{LastPLogOffset: 42}
		workspace := checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 43}

		require.NoError(checkpointStorage.PutPartitionCheckpoint(appID, partitionID, partition))
		require.NoError(checkpointStorage.PutWorkspaceCheckpoint(appID, wsid, workspace))

		actualPartition, ok, err := checkpointStorage.GetPartitionCheckpoint(appID, partitionID)
		require.NoError(err)
		require.True(ok)
		require.Equal(partition, actualPartition)

		actualWorkspace, ok, err := checkpointStorage.GetWorkspaceCheckpoint(appID, wsid)
		require.NoError(err)
		require.True(ok)
		require.Equal(workspace, actualWorkspace)

		assertCheckpointJSON(t, sysVVMStorage, partitionCheckpointPKeyForTest(appID, partitionID),
			checkpointCColsForTest(), map[string]any{"lastPLogOffset": float64(42)})
		assertCheckpointJSON(t, sysVVMStorage, workspaceCheckpointPKeyForTest(appID, wsid),
			checkpointCColsForTest(), map[string]any{"wlogIDBearingOffset": float64(43)})
	})

	t.Run("partition-key suffixes isolate applications partitions and workspaces", func(t *testing.T) {
		t.Run("partition", func(t *testing.T) {
			partitionCases := []struct {
				appID       istructs.ClusterAppID
				partitionID istructs.PartitionID
				offset      istructs.Offset
			}{
				{appID: 201, partitionID: 1, offset: 11},
				{appID: 202, partitionID: 1, offset: 12},
				{appID: 201, partitionID: 2, offset: 13},
			}
			for _, tc := range partitionCases {
				require.NoError(checkpointStorage.PutPartitionCheckpoint(tc.appID, tc.partitionID,
					checkpoints.PartitionCheckpoint{LastPLogOffset: tc.offset}))
			}
			for _, tc := range partitionCases {
				actual, ok, err := checkpointStorage.GetPartitionCheckpoint(tc.appID, tc.partitionID)
				require.NoError(err)
				require.True(ok)
				require.Equal(tc.offset, actual.LastPLogOffset)
				assertCheckpointJSON(t, sysVVMStorage,
					partitionCheckpointPKeyForTest(tc.appID, tc.partitionID), checkpointCColsForTest(),
					map[string]any{"lastPLogOffset": float64(tc.offset)})
			}
		})

		t.Run("workspaces", func(t *testing.T) {
			workspaceCases := []struct {
				appID  istructs.ClusterAppID
				wsid   istructs.WSID
				offset istructs.Offset
			}{
				{appID: 201, wsid: 1, offset: 21},
				{appID: 202, wsid: 1, offset: 22},
				{appID: 201, wsid: 2, offset: 23},
			}
			for _, tc := range workspaceCases {
				require.NoError(checkpointStorage.PutWorkspaceCheckpoint(tc.appID, tc.wsid,
					checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: tc.offset}))
			}
			for _, tc := range workspaceCases {
				actual, ok, err := checkpointStorage.GetWorkspaceCheckpoint(tc.appID, tc.wsid)
				require.NoError(err)
				require.True(ok)
				require.Equal(tc.offset, actual.WLogIDBearingOffset)
				assertCheckpointJSON(t, sysVVMStorage,
					workspaceCheckpointPKeyForTest(tc.appID, tc.wsid), checkpointCColsForTest(),
					map[string]any{"wlogIDBearingOffset": float64(tc.offset)})
			}
		})
	})

	t.Run("malformed or incomplete values return errors", func(t *testing.T) {
		malformedPartitionID := istructs.PartitionID(77)
		malformedWSID := istructs.WSID(7701)
		require.NoError(sysVVMStorage.Put(partitionCheckpointPKeyForTest(appID, malformedPartitionID),
			checkpointCColsForTest(), []byte("not-json")))
		require.NoError(sysVVMStorage.Put(workspaceCheckpointPKeyForTest(appID, malformedWSID),
			checkpointCColsForTest(), []byte(`{"other":1}`)))

		_, ok, err := checkpointStorage.GetPartitionCheckpoint(appID, malformedPartitionID)
		require.Error(err)
		require.False(ok)

		_, ok, err = checkpointStorage.GetWorkspaceCheckpoint(appID, malformedWSID)
		require.Error(err)
		require.False(ok)
	})
}

func TestRecoveryCheckpointStorageUsesLastWriteWins(t *testing.T) {
	require := require.New(t)
	_, checkpointStorage := newRecoveryCheckpointStorageForTest(t)

	const (
		appID       = istructs.ClusterAppID(301)
		partitionID = istructs.PartitionID(3)
		wsid        = istructs.WSID(3001)
	)

	require.NoError(checkpointStorage.PutPartitionCheckpoint(appID, partitionID,
		checkpoints.PartitionCheckpoint{LastPLogOffset: 200}))
	require.NoError(checkpointStorage.PutWorkspaceCheckpoint(appID, wsid,
		checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 50}))

	partition := checkpoints.PartitionCheckpoint{LastPLogOffset: 100}
	workspace := checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 40}
	require.NoError(checkpointStorage.PutPartitionCheckpoint(appID, partitionID, partition))
	require.NoError(checkpointStorage.PutWorkspaceCheckpoint(appID, wsid, workspace))

	actualPartition, ok, err := checkpointStorage.GetPartitionCheckpoint(appID, partitionID)
	require.NoError(err)
	require.True(ok)
	require.Equal(partition, actualPartition)

	actualWorkspace, ok, err := checkpointStorage.GetWorkspaceCheckpoint(appID, wsid)
	require.NoError(err)
	require.True(ok)
	require.Equal(workspace, actualWorkspace)
}

func partitionCheckpointPKeyForTest(appID istructs.ClusterAppID, partitionID istructs.PartitionID) []byte {
	pKey := binary.BigEndian.AppendUint32(nil, pKeyPrefix_SeqStorage_Part)
	pKey = binary.BigEndian.AppendUint32(pKey, appID)
	return binary.BigEndian.AppendUint16(pKey, uint16(partitionID))
}

func workspaceCheckpointPKeyForTest(appID istructs.ClusterAppID, wsid istructs.WSID) []byte {
	pKey := binary.BigEndian.AppendUint32(nil, pKeyPrefix_SeqStorage_WS)
	pKey = binary.BigEndian.AppendUint32(pKey, appID)
	return binary.BigEndian.AppendUint64(pKey, uint64(wsid))
}

func checkpointCColsForTest() []byte {
	return []byte{1}
}

func assertCheckpointJSON(t *testing.T, storage ISysVvmStorage, pKey, cCols []byte, expected map[string]any) {
	t.Helper()
	require := require.New(t)
	value := []byte{}
	ok, err := storage.Get(pKey, cCols, &value)
	require.NoError(err)
	require.True(ok)
	actual := map[string]any{}
	require.NoError(json.Unmarshal(value, &actual))
	require.Equal(expected, actual)
}

func newRecoveryCheckpointStorageForTest(t *testing.T) (ISysVvmStorage, checkpoints.IRecoveryCheckpointStorage) {
	t.Helper()
	appStorageProvider := provider.Provide(mem.Provide(testingu.MockTime))
	sysVVMStorage, err := appStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(t, err)
	return sysVVMStorage, NewRecoveryCheckpointStorage(sysVVMStorage)
}
