/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"encoding/binary"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/testingu"
)

func TestDataWithExpiration_BasicUsage(t *testing.T) {
	require := require.New(t)
	expireAt_expected := testingu.MockTime.Now().Add(time.Minute).UnixMilli()
	data_expected := []byte("Hello, World!")

	dwe := &DataWithExpiration{
		ExpireAt: expireAt_expected,
		Data:     data_expected,
	}

	bytes := dwe.ToBytes()

	t.Run("read", func(t *testing.T) {
		dwe_actual := ReadWithExpiration(bytes)
		require.Equal(expireAt_expected, dwe_actual.ExpireAt)
		require.Equal(data_expected, dwe_actual.Data)

	})

	t.Run("expiration", func(t *testing.T) {
		require.False(dwe.IsExpired(testingu.MockTime.Now()))
		testingu.MockTime.Add(time.Minute)
		testingu.MockTime.Add(-time.Millisecond)
		require.False(dwe.IsExpired(testingu.MockTime.Now()))
		testingu.MockTime.Add(time.Millisecond)
		require.True(dwe.IsExpired(testingu.MockTime.Now()))
	})

	t.Run("no expiration", func(t *testing.T) {
		expirations := []int64{0, -1}
		for _, exp := range expirations {
			t.Run(strconv.Itoa(int(exp)), func(t *testing.T) {
				dwe := &DataWithExpiration{
					Data:     data_expected,
					ExpireAt: exp,
				}
				require.False(dwe.IsExpired(testingu.MockTime.Now()))
				testingu.MockTime.Add(time.Minute)
				require.False(dwe.IsExpired(testingu.MockTime.Now()))
			})
		}
	})

	t.Run("internal structure", func(t *testing.T) {
		expireAt_actual := binary.BigEndian.Uint64(bytes[:utils.Uint64Size])
		require.Equal(uint64(expireAt_expected), expireAt_actual)
		require.Equal(data_expected, bytes[utils.Uint64Size:])
	})
}
