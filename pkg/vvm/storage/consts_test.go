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

	_ = uint32(pKeyPrefix_SeqStorage_Part - 2)
	_ = uint32(2 - pKeyPrefix_SeqStorage_Part)

	_ = uint32(pKeyPrefix_SeqStorage_WS - 3)
	_ = uint32(3 - pKeyPrefix_SeqStorage_WS)

	// Recovery checkpoints reuse the retained sequence-storage namespaces. Keep
	// both their values and their position before AppTTL fixed.
	_ = uint32(pKeyPrefix_SeqStorage_Part - pKeyPrefix_VVMLeader - 1)
	_ = uint32(pKeyPrefix_SeqStorage_WS - pKeyPrefix_SeqStorage_Part - 1)
	_ = uint32(pKeyPrefix_AppTTL - pKeyPrefix_SeqStorage_WS - 1)
)

func TestConsts(t *testing.T) {
	require := require.New(t)
	require.Equal(uint32(1), pKeyPrefix_VVMLeader)

	require.Equal(uint32(2), pKeyPrefix_SeqStorage_Part)

	require.Equal(uint32(3), pKeyPrefix_SeqStorage_WS)
	require.Equal(uint32(4), pKeyPrefix_AppTTL)
}
