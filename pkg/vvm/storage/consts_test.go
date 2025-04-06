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
	_ = uint32(pKeyPrefix_SeqStorage - 2)
	_ = uint32(2 - pKeyPrefix_SeqStorage)
)

func TestConsts(t *testing.T) {
	require.Equal(t, uint32(1), pKeyPrefix_VVMLeader)
	require.Equal(t, uint32(2), pKeyPrefix_SeqStorage)
}
