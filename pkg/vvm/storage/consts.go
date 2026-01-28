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

	// [~server.design.sequences/cmp.VVMSeqStorageAdapter.KeyPrefixSeqStoragePart~impl]
	pKeyPrefix_SeqStorage_Part

	// [~server.design.sequences/cmp.VVMSeqStorageAdapter.KeyPrefixSeqStorageWS~impl]
	pKeyPrefix_SeqStorage_WS

	pKeyPrefix_AppTTL
)

const (
	// [~server.design.sequences/cmp.VVMSeqStorageAdapter.PLogOffsetCC~impl]
	PLogOffsetCC = uint32(0)
)

const (
	MaxKeyLength                = 1024
	MaxValueLength              = 65536
	MaxTTLSeconds               = 31536000
	appTTLPKSize                = 8
	appTTLValidationErrTemplate = "%w: %w"
)
