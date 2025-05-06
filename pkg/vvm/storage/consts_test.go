/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:unconvert
const (
	_ = uint32(pKeyPrefix_VVMLeader - 1)
	_ = uint32(1 - pKeyPrefix_VVMLeader)

	// [~server.design.sequences/cmp.VVMSeqStorageAdapter.KeyPrefixSeqStoragePart.test~impl]
	_ = uint32(pKeyPrefix_SeqStorage_Part - 2)
	_ = uint32(2 - pKeyPrefix_SeqStorage_Part)

	// [~server.design.sequences/cmp.VVMSeqStorageAdapter.KeyPrefixSeqStorageWS.test~impl]
	_ = uint32(pKeyPrefix_SeqStorage_WS - 3)
	_ = uint32(3 - pKeyPrefix_SeqStorage_WS)
)

func TestConsts(t *testing.T) {
	require := require.New(t)
	require.Equal(uint32(1), pKeyPrefix_VVMLeader)

	// [~server.design.sequences/cmp.VVMSeqStorageAdapter.KeyPrefixSeqStoragePart.test~impl]
	require.Equal(uint32(2), pKeyPrefix_SeqStorage_Part)

	// [~server.design.sequences/cmp.VVMSeqStorageAdapter.KeyPrefixSeqStorageWS.test~impl]
	require.Equal(uint32(3), pKeyPrefix_SeqStorage_WS)

	// [~server.design.sequences/cmp.VVMSeqStorageAdapter.PLogOffsetCC.test~impl]
	require.Equal(uint32(0), PLogOffsetCC)
}
