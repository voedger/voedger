/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */
package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
)

const (
	testTTLSeconds = 60
	testClusterApp = istructs.ClusterAppID(1)
)

func TestAppTTLStorage_BasicUsage(t *testing.T) {
	require := require.New(t)
	asf := mem.Provide(testingu.MockTime)
	sp := provider.Provide(asf, "")
	storage, err := sp.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(err)

	ttlStorage := NewAppTTLStorage(storage, testClusterApp)

	t.Run("InsertIfNotExists", func(t *testing.T) {
		ok, err := ttlStorage.InsertIfNotExists("insertKey1", "testValue", testTTLSeconds)
		require.NoError(err)
		require.True(ok)

		t.Run("TTLGet returns inserted value", func(t *testing.T) {
			value, ok, err := ttlStorage.TTLGet("insertKey1")
			require.NoError(err)
			require.True(ok)
			require.Equal("testValue", value)
		})

		t.Run("insert again fails", func(t *testing.T) {
			ok, err := ttlStorage.InsertIfNotExists("insertKey1", "otherValue", testTTLSeconds)
			require.NoError(err)
			require.False(ok)
		})

		testingu.MockTime.Add(time.Duration(testTTLSeconds+1) * time.Second)

		t.Run("TTLGet returns nothing after expiration", func(t *testing.T) {
			_, ok, err := ttlStorage.TTLGet("insertKey1")
			require.NoError(err)
			require.False(ok)
		})

		t.Run("insert succeeds after expiration", func(t *testing.T) {
			ok, err := ttlStorage.InsertIfNotExists("insertKey1", "newValue", testTTLSeconds)
			require.NoError(err)
			require.True(ok)
		})
	})

	t.Run("CompareAndSwap", func(t *testing.T) {
		ok, err := ttlStorage.InsertIfNotExists("swapKey1", "initialValue", testTTLSeconds)
		require.NoError(err)
		require.True(ok)

		t.Run("swap with wrong expected value fails", func(t *testing.T) {
			ok, err := ttlStorage.CompareAndSwap("swapKey1", "wrongValue", "newValue", testTTLSeconds)
			require.NoError(err)
			require.False(ok)
		})

		t.Run("swap with correct expected value succeeds", func(t *testing.T) {
			ok, err := ttlStorage.CompareAndSwap("swapKey1", "initialValue", "swappedValue", testTTLSeconds)
			require.NoError(err)
			require.True(ok)

			value, ok, err := ttlStorage.TTLGet("swapKey1")
			require.NoError(err)
			require.True(ok)
			require.Equal("swappedValue", value)
		})

		testingu.MockTime.Add(time.Duration(testTTLSeconds+1) * time.Second)

		t.Run("swap fails after expiration", func(t *testing.T) {
			ok, err := ttlStorage.CompareAndSwap("swapKey1", "swappedValue", "newValue", testTTLSeconds)
			require.NoError(err)
			require.False(ok)
		})
	})

	t.Run("CompareAndDelete", func(t *testing.T) {
		ok, err := ttlStorage.InsertIfNotExists("deleteKey1", "deleteValue", testTTLSeconds)
		require.NoError(err)
		require.True(ok)

		t.Run("delete with wrong expected value fails", func(t *testing.T) {
			ok, err := ttlStorage.CompareAndDelete("deleteKey1", "wrongValue")
			require.NoError(err)
			require.False(ok)
		})

		t.Run("delete with correct expected value succeeds", func(t *testing.T) {
			ok, err := ttlStorage.CompareAndDelete("deleteKey1", "deleteValue")
			require.NoError(err)
			require.True(ok)

			_, ok, err = ttlStorage.TTLGet("deleteKey1")
			require.NoError(err)
			require.False(ok)
		})

		ok, err = ttlStorage.InsertIfNotExists("deleteKey2", "value2", testTTLSeconds)
		require.NoError(err)
		require.True(ok)

		testingu.MockTime.Add(time.Duration(testTTLSeconds+1) * time.Second)

		t.Run("delete fails after expiration", func(t *testing.T) {
			ok, err := ttlStorage.CompareAndDelete("deleteKey2", "value2")
			require.NoError(err)
			require.False(ok)
		})
	})
}

func TestAppTTLStorage_Validation(t *testing.T) {
	require := require.New(t)
	asf := mem.Provide(testingu.MockTime)
	sp := provider.Provide(asf, "")
	storage, err := sp.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(err)

	ttlStorage := NewAppTTLStorage(storage, testClusterApp)

	t.Run("empty key", func(t *testing.T) {
		_, err := ttlStorage.InsertIfNotExists("", "value", testTTLSeconds)
		require.ErrorIs(err, ErrAppTTLValidation)
		require.ErrorIs(err, ErrKeyEmpty)

		_, _, err = ttlStorage.TTLGet("")
		require.ErrorIs(err, ErrAppTTLValidation)
		require.ErrorIs(err, ErrKeyEmpty)

		_, err = ttlStorage.CompareAndSwap("", "old", "new", testTTLSeconds)
		require.ErrorIs(err, ErrAppTTLValidation)
		require.ErrorIs(err, ErrKeyEmpty)

		_, err = ttlStorage.CompareAndDelete("", "value")
		require.ErrorIs(err, ErrAppTTLValidation)
		require.ErrorIs(err, ErrKeyEmpty)
	})

	t.Run("key too long", func(t *testing.T) {
		longKey := string(make([]byte, MaxKeyLength+1))
		_, err := ttlStorage.InsertIfNotExists(longKey, "value", testTTLSeconds)
		require.ErrorIs(err, ErrAppTTLValidation)
		require.ErrorIs(err, ErrKeyTooLong)
	})

	t.Run("key invalid UTF-8", func(t *testing.T) {
		invalidUTF8Key := string([]byte{0xff, 0xfe, 0xfd})
		_, err := ttlStorage.InsertIfNotExists(invalidUTF8Key, "value", testTTLSeconds)
		require.ErrorIs(err, ErrAppTTLValidation)
		require.ErrorIs(err, ErrKeyInvalidUTF8)
	})

	t.Run("value too long", func(t *testing.T) {
		longValue := string(make([]byte, MaxValueLength+1))
		_, err := ttlStorage.InsertIfNotExists("key", longValue, testTTLSeconds)
		require.ErrorIs(err, ErrAppTTLValidation)
		require.ErrorIs(err, ErrValueTooLong)
	})

	t.Run("invalid TTL", func(t *testing.T) {
		_, err := ttlStorage.InsertIfNotExists("key", "value", 0)
		require.ErrorIs(err, ErrAppTTLValidation)
		require.ErrorIs(err, ErrInvalidTTL)

		_, err = ttlStorage.InsertIfNotExists("key", "value", -1)
		require.ErrorIs(err, ErrAppTTLValidation)
		require.ErrorIs(err, ErrInvalidTTL)

		_, err = ttlStorage.InsertIfNotExists("key", "value", MaxTTLSeconds+1)
		require.ErrorIs(err, ErrAppTTLValidation)
		require.ErrorIs(err, ErrInvalidTTL)
	})
}
