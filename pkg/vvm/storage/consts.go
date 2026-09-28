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

	// Deprecated: retained to preserve value of pKeyPrefix_AppTTL only
	pKeyPrefix_SeqStorage_Part

	// Deprecated: retained to preserve value of pKeyPrefix_AppTTL only
	pKeyPrefix_SeqStorage_WS

	pKeyPrefix_AppTTL
)

const (
	PLogOffsetCC = uint32(0)
)

const (
	MaxKeyLength                = 1024
	MaxValueLength              = 65536
	MaxTTLSeconds               = 31536000
	appTTLPKSize                = 8
	appTTLValidationErrTemplate = "%w: %w"
)
