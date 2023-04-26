/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qrename

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
	"github.com/voedger/voedger/pkg/schemas"
)

func TestRenameQName(t *testing.T) {

	require := require.New(t)

	old := schemas.NewQName("test", "old")
	new := schemas.NewQName("test", "new")

	storage := teststore.NewStorage()

	t.Run("prepare storage with old QName", func(t *testing.T) {
		versions := vers.New()
		err := versions.Prepare(storage)
		require.NoError(err)

		bld := schemas.NewSchemaCache()
		_ = bld.Add(old, schemas.SchemaKind_Object)
		schemas, err := bld.Build()
		require.NoError(err)

		names := qnames.New()
		err = names.Prepare(storage, versions, schemas, nil)
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
