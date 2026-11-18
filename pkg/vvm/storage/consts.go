/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package storage

const (
	// nolint: unused
	pKeyPrefix_null pKeyPrefix = iota

	// [~server.design.orch/KeyPrefix_VVMLeader~impl]
	pKeyPrefix_VVMLeader

	// Partition recovery checkpoints append the partition ID to the application partition key.
	pKeyPrefix_SeqStorage_Part

	// Workspace recovery checkpoints append the WSID to the application partition key.
	pKeyPrefix_SeqStorage_WS

	pKeyPrefix_AppTTL
)

const (
	MaxKeyLength                 = 1024
	MaxValueLength               = 65536
	MaxTTLSeconds                = 31536000
	appTTLPKSize                 = 8
	appTTLValidationErrTemplate  = "%w: %w"
	partitionCheckpointPKeySize  = 4 + 4 + 2 // pkeyPrefix+ClusterAppID+PartitionID
	workspaceCheckpointPKeySize  = 4 + 4 + 8 // pkeyPrefix+ClusterAppID+WSID
	jsonFieldLastPLogOffset      = "lastPLogOffset"
	jsonFieldWLogIDBearingOffset = "wlogIDBearingOffset"
)

var (
	checkpointCCols = []byte{1}
)
