/*
 * Copyright (c) 2025-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package coreutils

import (
	"encoding/binary"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDataWithExpiration_ToBytesAndRead(t *testing.T) {
	// Prepare test data
	original := &DataWithExpiration{
		Data:     []byte("hello"),
		ExpireAt: 1672522560123, // Some UnixMilli timestamp
	}

	// Call ToBytes
	encoded := original.ToBytes()

	// Check length: Data bytes + 8 bytes for uint64
	require.Len(t, original.Data, len(encoded)-utils.Uint64Size, "encoded length mismatch")

	// Manually check the last 8 bytes match the big-endian expireAt
	last8 := encoded[len(encoded)-8:]
	expireVal := binary.BigEndian.Uint64(last8)
	require.Equal(t, uint64(original.ExpireAt), expireVal, "expireAt mismatch in encoded data")

	// Decode into a new struct
	decoded := &DataWithExpiration{}
	decoded.Read(encoded)

	// Check that fields match
	require.Equal(t, original.Data, decoded.Data, "Data field should match after Read")
	require.Equal(t, original.ExpireAt, decoded.ExpireAt, "ExpireAt field should match after Read")
}

func TestDataWithExpiration_IsExpired(t *testing.T) {
	now := time.Now()

	// 1) ExpireAt = 0 => never expired
	d1 := DataWithExpiration{
		Data:     []byte("no expiry"),
		ExpireAt: 0,
	}
	require.False(t, d1.IsExpired(now), "expected not to be expired if ExpireAt=0")

	// 2) ExpireAt in the future => not expired
	futureTime := now.Add(24 * time.Hour).UnixMilli()
	d2 := DataWithExpiration{
		Data:     []byte("future"),
		ExpireAt: futureTime,
	}
	require.False(t, d2.IsExpired(now), "expected not to be expired if ExpireAt is in the future")

	// 3) ExpireAt in the past => expired
	pastTime := now.Add(-24 * time.Hour).UnixMilli()
	d3 := DataWithExpiration{
		Data:     []byte("past"),
		ExpireAt: pastTime,
	}
	require.True(t, d3.IsExpired(now), "expected to be expired if ExpireAt is in the past")

	// 4) Edge case: ExpireAt exactly 'now' => isExpired should be true
	exactNow := now.UnixMilli()
	d4 := DataWithExpiration{
		Data:     []byte("exact now"),
		ExpireAt: exactNow,
	}
	require.True(t, d4.IsExpired(now), "expected to be expired if ExpireAt == now")
}

func TestDataWithExpiration_ReadShortData(t *testing.T) {
	// (Optional) Test if data is too short to contain an 8-byte ExpireAt.
	// The current Read method does not handle this with an error, so it
	// may lead to a panic. Let's demonstrate how to test for that scenario.

	d := &DataWithExpiration{}
	shortData := []byte("12345") // only 5 bytes, definitely < 8 needed

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic due to short data, but got none")
		}
	}()
	d.Read(shortData)
}
