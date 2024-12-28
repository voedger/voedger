/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

func Test_AppDefFunctions(t *testing.T) {

	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")

	cmdName := appdef.NewQName("test", "cmd")
	qrName := appdef.NewQName("test", "query")
	parName := appdef.NewQName("test", "param")
	resName := appdef.NewQName("test", "res")

	t.Run("Should be ok to build application with functions", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddObject(parName)
		_ = wsb.AddObject(resName)

		cmd := wsb.AddCommand(cmdName)
		cmd.SetEngine(appdef.ExtensionEngineKind_WASM)
		cmd.
			SetParam(parName).
			SetResult(resName)

		qry := wsb.AddQuery(qrName)
		qry.
			SetParam(parName).
			SetResult(appdef.QNameANY)

		a, err := adb.Build()
		require.NoError(err)

		app = a
		require.NotNil(app)
	})

	testWith := func(tested testedTypes) {
		t.Run("should be ok to enumerate functions", func(t *testing.T) {
			var names []appdef.QName
			for f := range appdef.Functions(tested.Types()) {
				require.Equal(wsName, f.Workspace().QName())
				names = append(names, f.QName())
			}
			require.Len(names, 2)
			require.Equal([]appdef.QName{cmdName, qrName}, names)
		})

		t.Run("should be ok to find function by name", func(t *testing.T) {
			f := appdef.Function(tested.Type, cmdName)
			require.NotNil(f)
			require.Equal(cmdName, f.QName())
		})

		require.Nil(appdef.Function(tested.Type, appdef.NewQName("test", "unknown")), "Should be nil if unknown")
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}
