/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qnames

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func TestRenameQName(t *testing.T) {

	require := require.New(t)

	appName := appdef.NewAppQName("test", "app")

	oldQName := appdef.NewQName("test", "old")
	newQName := appdef.NewQName("test", "new")

	storage := teststore.NewStorage(appName)

	t.Run("prepare storage with old QName", func(t *testing.T) {
		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		_ = wsb.AddObject(oldQName)
		appDef, err := adb.Build()
		require.NoError(err)

		names := New()
		err = names.Prepare(storage, versions, appDef)
		require.NoError(err)
	})

	t.Run("basic usage", func(t *testing.T) {
		err := Rename(storage, oldQName, newQName)
		require.NoError(err)
	})

	t.Run("check result", func(t *testing.T) {
		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		names := New()
		err = names.Prepare(storage, versions, nil)
		require.NoError(err)

		t.Run("check old is deleted", func(t *testing.T) {
			id, err := names.ID(oldQName)
			require.ErrorIs(err, ErrNameNotFound)
			require.Equal(NullQNameID, id)
		})

		t.Run("check new is not null", func(t *testing.T) {
			id, err := names.ID(newQName)
			require.NoError(err)
			require.Greater(id, QNameIDSysLast)
		})
	})
}

func TestRenameQName_Errors(t *testing.T) {
	require := require.New(t)

	appName := appdef.NewAppQName("test", "app")

	oldQName := appdef.NewQName("test", "old")
	newQName := appdef.NewQName("test", "new")
	other := appdef.NewQName("test", "other")

	storage := teststore.NewStorage(appName)

	t.Run("prepare storage with old QName", func(t *testing.T) {
		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		_ = wsb.AddObject(oldQName)
		_ = wsb.AddObject(other)
		appDef, err := adb.Build()
		require.NoError(err)

		names := New()
		err = names.Prepare(storage, versions, appDef)
		require.NoError(err)
	})

	t.Run("should be error if old and new are equals", func(t *testing.T) {
		err := Rename(storage, oldQName, oldQName)
		require.ErrorContains(err, "equals")
	})

	t.Run("should be error if twice rename", func(t *testing.T) {
		err := Rename(storage, oldQName, newQName)
		require.NoError(err)

		err = Rename(storage, oldQName, newQName)
		require.ErrorIs(err, ErrNameNotFound)

		t.Run("but must ok reverse rename", func(t *testing.T) {
			err = Rename(storage, newQName, oldQName)
			require.NoError(err)
		})
	})

	t.Run("should be error if old name not found", func(t *testing.T) {
		err := Rename(storage, appdef.NewQName("test", "unknown"), newQName)
		require.ErrorIs(err, ErrNameNotFound)
	})

	t.Run("should be error if new name is already exists", func(t *testing.T) {
		err := Rename(storage, oldQName, other)
		require.ErrorContains(err, "exists")
	})
}

func TestRenameQName_Fails(t *testing.T) {
	require := require.New(t)

	appName := appdef.NewAppQName("test", "app")

	oldQName := appdef.NewQName("test", "old")
	newQName := appdef.NewQName("test", "new")

	t.Run("should be error if unsupported version of Versions system view", func(t *testing.T) {
		testError := errors.New("error read versions")
		storage := teststore.NewStorage(appName)

		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)
		versions.Put(vers.SysQNamesVersion, latestVersion+1) // future version

		storage.ScheduleGetError(testError, utils.ToBytes(consts.SysView_Versions), nil)

		err = Rename(storage, oldQName, newQName)
		require.ErrorIs(err, testError)
	})

	t.Run("should be error if unsupported version of QNames system view", func(t *testing.T) {
		storage := teststore.NewStorage(appName)

		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)
		versions.Put(vers.SysQNamesVersion, latestVersion+1) // future version

		err = Rename(storage, oldQName, newQName)
		require.ErrorIs(err, vers.ErrorInvalidVersion)
	})

	storage := teststore.NewStorage(appName)

	t.Run("prepare storage with old QName", func(t *testing.T) {
		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		_ = wsb.AddObject(oldQName)
		appDef, err := adb.Build()
		require.NoError(err)

		names := New()
		err = names.Prepare(storage, versions, appDef)
		require.NoError(err)
	})

	t.Run("should be error if storage read failed", func(t *testing.T) {
		testError := errors.New("can not read old qname")

		storage.ScheduleGetError(testError, nil, []byte(oldQName.String()))

		err := Rename(storage, oldQName, newQName)
		require.ErrorIs(err, testError)
	})

	t.Run("should be error if storage put failed", func(t *testing.T) {
		testError := errors.New("can not delete old qname")

		storage.SchedulePutError(testError, nil, []byte(newQName.String()))

		err := Rename(storage, oldQName, newQName)
		require.ErrorIs(err, testError)
	})
}
