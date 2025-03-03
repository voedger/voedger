/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
)

// [~server.design.orch/TTLStorageTest~impl]

const seconds10 = 10

func TestInsertIfNotExist(t *testing.T) {
	require := require.New(t)
	appStorageProvider := provider.Provide(mem.Provide(coreutils.MockTime))
	sysVvmAppStorage, err := appStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(err)
	const (
		ttlStorageImplKey TTLStorageImplKey = 12345
		expectedVal                         = "some-value"
	)
	ttlStorage := NewElectionsTTLStorage(sysVvmAppStorage)

	// basic usage
	ok, err := ttlStorage.InsertIfNotExist(ttlStorageImplKey, "some-value", seconds10)
	require.NoError(err)
	require.True(ok)

	// check the underlying storage
	pKey, cCols := ttlStorage.(*implElectionsITTLStorage).buildKeys(ttlStorageImplKey)
	valBytes := []byte{}
	ok, err = sysVvmAppStorage.TTLGet(pKey, cCols, &valBytes)
	require.NoError(err)
	require.True(ok)
	require.Equal(expectedVal, string(expectedVal))

	// check using ITTLStorage.Get
	ok, storedVal, err := ttlStorage.Get(ttlStorageImplKey)
	require.NoError(err)
	require.True(ok)
	require.Equal(expectedVal, storedVal)

	// expire the value
	coreutils.MockTime.Sleep(seconds10 * time.Second)
	ok, err = sysVvmAppStorage.TTLGet(pKey, cCols, &valBytes)
	require.NoError(err)
	require.False(ok)
}

func TestCompareAndSwap(t *testing.T) {
	require := require.New(t)
	appStorageProvider := provider.Provide(mem.Provide(coreutils.MockTime))
	sysVvmAppStorage, err := appStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(err)
	const (
		ttlStorageImplKey TTLStorageImplKey = 12345
		oldVal                              = "old-val"
		newVal                              = "new-val"
	)
	ttlStorage := NewElectionsTTLStorage(sysVvmAppStorage)

	// no stored value -> !ok
	t.Run("no stored value -> !ok", func(t *testing.T) {
		ok, err := ttlStorage.CompareAndSwap(ttlStorageImplKey, oldVal, newVal, seconds10)
		require.NoError(err)
		require.False(ok)
	})

	// insert a value
	ok, err := ttlStorage.InsertIfNotExist(ttlStorageImplKey, oldVal, seconds10)
	require.NoError(err)
	require.True(ok)

	pKey, cCols := ttlStorage.(*implElectionsITTLStorage).buildKeys(ttlStorageImplKey)

	t.Run("basic usage", func(t *testing.T) {
		// swap stored oldVal->newVal -> ok
		ok, err = ttlStorage.CompareAndSwap(ttlStorageImplKey, oldVal, newVal, seconds10)
		require.NoError(err)
		require.True(ok)

		// check the newVal is stored indeed after CompareAndSwap
		valBytes := []byte{}
		ok, err = sysVvmAppStorage.TTLGet(pKey, cCols, &valBytes)
		require.NoError(err)
		require.True(ok)
		require.Equal(newVal, string(valBytes))
	})

	t.Run("CompareAndSwap failure if the stored value is modified", func(t *testing.T) {
		modifiedVal := "sdfsdfsd"
		// remove+insert new to simulate the value has changed
		// do not check ok for the case this test is run separately
		_, err := ttlStorage.CompareAndDelete(ttlStorageImplKey, newVal)
		require.NoError(err)
		_, err = ttlStorage.CompareAndDelete(ttlStorageImplKey, oldVal)
		require.NoError(err)

		ok, err = ttlStorage.InsertIfNotExist(ttlStorageImplKey, modifiedVal, seconds10)
		require.NoError(err)
		require.True(ok)

		// stored modifiedVal, trying to update oldVal->newVal -> fail
		ok, err = ttlStorage.CompareAndSwap(ttlStorageImplKey, oldVal, newVal, seconds10)
		require.NoError(err)
		require.False(ok)
	})
}

func TestCompareAndDelete(t *testing.T) {
	require := require.New(t)
	appStorageProvider := provider.Provide(mem.Provide(coreutils.MockTime))
	sysVvmAppStorage, err := appStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(err)
	const (
		ttlStorageImplKey TTLStorageImplKey = 12345
		val                                 = "val"
	)
	ttlStorage := NewElectionsTTLStorage(sysVvmAppStorage)

	t.Run("no stored value -> !ok", func(t *testing.T) {
		ok, err := ttlStorage.CompareAndDelete(ttlStorageImplKey, val)
		require.NoError(err)
		require.False(ok)
	})

	t.Run("basic usage", func(t *testing.T) {
		ok, err := ttlStorage.InsertIfNotExist(ttlStorageImplKey, val, seconds10)
		require.NoError(err)
		require.True(ok)

		ok, err = ttlStorage.CompareAndDelete(ttlStorageImplKey, val)
		require.NoError(err)
		require.True(ok)

		// check the value is removed indeed from the underlying storage
		pKey, cCols := ttlStorage.(*implElectionsITTLStorage).buildKeys(ttlStorageImplKey)
		valBytes := []byte{}
		ok, err = sysVvmAppStorage.TTLGet(pKey, cCols, &valBytes)
		require.NoError(err)
		require.False(ok)
	})

}
