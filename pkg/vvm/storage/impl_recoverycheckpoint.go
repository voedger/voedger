/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package storage

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/istructs"
)

type implCheckpointStorage struct {
	sysVVMStorage ISysVvmStorage
}

func (s *implCheckpointStorage) GetPartitionCheckpoint(appID istructs.ClusterAppID, partitionID istructs.PartitionID) (checkpoints.PartitionCheckpoint, bool, error) {
	data, ok, err := s.get(partitionCheckpointPKey(appID, partitionID), checkpointCCols)
	if err != nil {
		return checkpoints.PartitionCheckpoint{}, false, fmt.Errorf("get partition recovery checkpoint: %w", err)
	}
	if !ok {
		return checkpoints.PartitionCheckpoint{}, false, nil
	}
	checkpoint, _, err := decodePartitionCheckpoint(data)
	if err != nil {
		return checkpoints.PartitionCheckpoint{}, false, fmt.Errorf("decode partition recovery checkpoint: %w", err)
	}
	return checkpoint, true, nil
}

func (s *implCheckpointStorage) PutPartitionCheckpoint(appID istructs.ClusterAppID, partitionID istructs.PartitionID, checkpoint checkpoints.PartitionCheckpoint) error {
	pKey := partitionCheckpointPKey(appID, partitionID)
	value := fmt.Sprintf(`{"%s":%d}`, jsonFieldLastPLogOffset, checkpoint.LastPLogOffset)
	if err := s.sysVVMStorage.Put(pKey, checkpointCCols, []byte(value)); err != nil {
		return fmt.Errorf("put partition recovery checkpoint: %w", err)
	}
	return nil
}

func (s *implCheckpointStorage) GetWorkspaceCheckpoint(appID istructs.ClusterAppID, wsid istructs.WSID) (checkpoints.WorkspaceCheckpoint, bool, error) {
	data, ok, err := s.get(workspaceCheckpointPKey(appID, wsid), checkpointCCols)
	if err != nil {
		return checkpoints.WorkspaceCheckpoint{}, false,
			fmt.Errorf("get workspace recovery checkpoint: %w", err)
	}
	if !ok {
		return checkpoints.WorkspaceCheckpoint{}, false, nil
	}
	checkpoint, _, err := decodeWorkspaceCheckpoint(data)
	if err != nil {
		return checkpoints.WorkspaceCheckpoint{}, false,
			fmt.Errorf("decode workspace recovery checkpoint: %w", err)
	}
	return checkpoint, true, nil
}

func (s *implCheckpointStorage) PutWorkspaceCheckpoint(appID istructs.ClusterAppID,
	wsid istructs.WSID, checkpoint checkpoints.WorkspaceCheckpoint) error {
	pKey := workspaceCheckpointPKey(appID, wsid)
	value := fmt.Sprintf(`{"%s":%d}`, jsonFieldWLogIDBearingOffset, checkpoint.WLogIDBearingOffset)
	if err := s.sysVVMStorage.Put(pKey, checkpointCCols, []byte(value)); err != nil {
		return fmt.Errorf("put workspace recovery checkpoint: %w", err)
	}
	return nil
}

func (s *implCheckpointStorage) get(pKey, cCols []byte) ([]byte, bool, error) {
	var data []byte
	ok, err := s.sysVVMStorage.Get(pKey, cCols, &data)
	return data, ok, err
}

func partitionCheckpointPKey(appID istructs.ClusterAppID, partitionID istructs.PartitionID) []byte {
	pKey := make([]byte, 0, partitionCheckpointPKeySize)
	pKey = binary.BigEndian.AppendUint32(pKey, pKeyPrefix_SeqStorage_Part)
	pKey = binary.BigEndian.AppendUint32(pKey, appID)
	return binary.BigEndian.AppendUint16(pKey, uint16(partitionID))
}

func workspaceCheckpointPKey(appID istructs.ClusterAppID, wsid istructs.WSID) []byte {
	pKey := make([]byte, 0, workspaceCheckpointPKeySize)
	pKey = binary.BigEndian.AppendUint32(pKey, pKeyPrefix_SeqStorage_WS)
	pKey = binary.BigEndian.AppendUint32(pKey, appID)
	return binary.BigEndian.AppendUint64(pKey, uint64(wsid))
}

// PartitionCheckpointKey returns storage partition key and clustering columns for a partition recovery checkpoint.
func PartitionCheckpointKey(appID istructs.ClusterAppID, partitionID istructs.PartitionID) (pKey, cCols []byte) {
	return partitionCheckpointPKey(appID, partitionID), append([]byte(nil), checkpointCCols...)
}

// WorkspaceCheckpointKey returns storage partition key and clustering columns for a workspace recovery checkpoint.
func WorkspaceCheckpointKey(appID istructs.ClusterAppID, wsid istructs.WSID) (pKey, cCols []byte) {
	return workspaceCheckpointPKey(appID, wsid), append([]byte(nil), checkpointCCols...)
}

func decodePartitionCheckpoint(data []byte) (partCheckpoint checkpoints.PartitionCheckpoint, fields map[string]json.RawMessage, err error) {
	err = json.Unmarshal(data, &fields)
	if err != nil {
		return checkpoints.PartitionCheckpoint{}, nil, err
	}
	lastPLogOffset, err := decodeRequiredJSONField[istructs.Offset](fields, jsonFieldLastPLogOffset)
	if err != nil {
		return checkpoints.PartitionCheckpoint{}, nil, err
	}
	return checkpoints.PartitionCheckpoint{LastPLogOffset: lastPLogOffset}, fields, nil
}

func decodeWorkspaceCheckpoint(data []byte) (wsCheckpoint checkpoints.WorkspaceCheckpoint, fields map[string]json.RawMessage, err error) {
	if err = json.Unmarshal(data, &fields); err != nil {
		return checkpoints.WorkspaceCheckpoint{}, nil, err
	}
	wlogIDBearingOffset, err := decodeRequiredJSONField[istructs.Offset](fields, jsonFieldWLogIDBearingOffset)
	if err != nil {
		return checkpoints.WorkspaceCheckpoint{}, nil, err
	}
	return checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: wlogIDBearingOffset}, fields, nil
}

func decodeRequiredJSONField[T ~uint64](fields map[string]json.RawMessage, name string) (T, error) {
	raw, ok := fields[name]
	if !ok {
		return 0, fmt.Errorf("required recovery checkpoint field %q is missing", name)
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, fmt.Errorf("decode recovery checkpoint field %q: %w", name, err)
	}
	return value, nil
}
