/*
 * Copyright (c) 2025-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestSetEmptyWSDesc(t *testing.T) {

	require := require.New(t)

	t.Run("should be ok to assign default empty descriptor", func(t *testing.T) {
		wsName := appdef.NewQName("test", "ws")

		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)
		appdef.SetEmptyWSDesc(wsb)

		app, err := adb.Build()
		require.NoError(err)

		ws := app.Workspace(wsName)
		require.NotNil(ws)

		descName := ws.Descriptor()
		require.Contains(descName.String(), wsName.String())
	})

	t.Run("should panic if descriptor already assigned", func(t *testing.T) {
		wsName, descName := appdef.NewQName("test", "ws"), appdef.NewQName("test", "wsDesc")

		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)
		_ = wsb.AddCDoc(descName)
		wsb.SetDescriptor(descName)

		require.Panics(
			func() { appdef.SetEmptyWSDesc(wsb) },
			require.Is(appdef.ErrAlreadyExistsError),
			require.HasAll(wsName, descName),
		)

		app, err := adb.Build()
		require.NoError(err)

		ws := app.Workspace(wsName)
		require.NotNil(ws)
		require.Equal(ws.Descriptor(), descName)
	})
}
