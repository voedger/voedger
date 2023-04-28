/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qrename

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func TestRenameQName(t *testing.T) {

	require := require.New(t)

	old := appdef.NewQName("test", "old")
	new := appdef.NewQName("test", "new")

	storage := teststore.NewStorage()

	t.Run("prepare storage with old QName", func(t *testing.T) {
		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		appDefBuilder := appdef.New()
		_ = appDefBuilder.Add(old, appdef.DefKind_Object)
		appDef, err := appDefBuilder.Build()
		require.NoError(err)

		names := qnames.New()
		err = names.Prepare(storage, versions, appDef, nil)
		require.NoError(err)
	})

	t.Run("basic usage", func(t *testing.T) {
		err := Rename(storage, old, new)
		require.NoError(err)
	})

	t.Run("check result", func(t *testing.T) {
		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		names := qnames.New()
		err = names.Prepare(storage, versions, nil, nil)
		require.NoError(err)

		t.Run("check old is deleted", func(t *testing.T) {
			id, err := names.GetID(old)
			require.ErrorIs(err, qnames.ErrNameNotFound)
			require.Equal(id, qnames.NullQNameID)
		})

		t.Run("check new is not null", func(t *testing.T) {
			id, err := names.GetID(new)
			require.NoError(err)
			require.Greater(id, qnames.QNameIDSysLast)
		})
	})
}
