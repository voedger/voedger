/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package provider

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)
	asf := mem.Provide()
	asi := Provide(asf)

	app1 := istructs.NewAppQName("sys", "_") // SafeAppName is "sys"
	app2 := istructs.NewAppQName("sys", "/") // SafeAppName is "sys{uuid}"

	t.Run("not inited -> storage not inited error", func(t *testing.T) {
		_, err := asi.AppStorage(app1)
		require.ErrorIs(err, ErrStorageNotInited)
	})

	// init storage before obtaining
	require.NoError(asi.Init(app1))
	require.NoError(asi.Init(app2))

	t.Run("init again -> storage inited already error", func(t *testing.T) {
		require.ErrorIs(asi.Init(app1), ErrStorageInitedAlready)
		require.ErrorIs(asi.Init(app2), ErrStorageInitedAlready)
	})

	// basic IAppStorage obtain
	storage, err := asi.AppStorage(app1)
	require.NoError(err)
	storageApp2, err := asi.AppStorage(app2)
	require.NoError(err)

	t.Run("same IAppStorage instances on AppStorage calls for the same app", func(t *testing.T) {
		storage2, err := asi.AppStorage(app1)
		require.NoError(err)
		require.Same(storage, storage2)

		storageApp3, err := asi.AppStorage(app2)
		require.NoError(err)
		require.Same(storageApp2, storageApp3)
	})

	t.Run("safe app name is obtained once -> read it from sysmeta in future", func(t *testing.T) {
		// store something for app2
		require.NoError(storageApp2.Put([]byte{1}, []byte{1}, []byte{2}))

		// re-initialize
		asi = Provide(asf, asi.(*implIAppStorageInitializer).suffix)

		// obtain IAppStorage for app2
		// it should be the same as before
		asi.Init(app2)
		storage, err := asi.AppStorage(app2)
		require.NoError(err)

		// now check we've got into sysab for app2, not sysaa that could be if there was just single app2
		// because we've get sysab once for app2 so it should be stored in sysmeta
		val := []byte{}
		_, err = storage.Get([]byte{1}, []byte{1}, &val)
		require.NoError(err)
		require.Equal([]byte{2}, val)
	})
}

func TestInitErrorPersistence(t *testing.T) {
	require := require.New(t)
	asf := mem.Provide()
	asp := Provide(asf)

	app1 := istructs.NewAppQName("sys", "_")
	app1SafeName, err := istorage.NewSafeAppName(app1, func(name string) (bool, error) { return true, nil })
	require.NoError(err)

	// init the storage manually to force the error
	app1SafeName = asp.(*implIAppStorageInitializer).clarifyKeyspaceName(app1SafeName)
	require.NoError(asf.Init(app1SafeName))

	// expect an error
	storage, err := asp.AppStorage(app1)
	require.ErrorIs(err, ErrStorageInitError)
	require.Nil(storage)

	// re-init
	asp = Provide(asf, asp.(*implIAppStorageInitializer).suffix)

	// expect Init() error is stored in sysmeta
	storage, err = asp.AppStorage(app1)
	require.ErrorIs(err, ErrStorageInitError)
	require.Nil(storage)
}
