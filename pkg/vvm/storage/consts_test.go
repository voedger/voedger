/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	_ = uint32(pKeyPrefix_Elections - 1)
	_ = uint32(1 - pKeyPrefix_Elections)
)

func TestConsts(t *testing.T) {
	require.Equal(t, uint32(1), pKeyPrefix_Elections)
}
