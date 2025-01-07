/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qrename

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
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

		names := qnames.New()
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

		names := qnames.New()
		err = names.Prepare(storage, versions, nil)
		require.NoError(err)

		t.Run("check old is deleted", func(t *testing.T) {
			id, err := names.ID(oldQName)
			require.ErrorIs(err, qnames.ErrNameNotFound)
			require.Equal(qnames.NullQNameID, id)
		})

		t.Run("check new is not null", func(t *testing.T) {
			id, err := names.ID(newQName)
			require.NoError(err)
			require.Greater(id, qnames.QNameIDSysLast)
		})
	})
}
