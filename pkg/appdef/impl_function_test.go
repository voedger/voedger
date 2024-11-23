/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDefFunctions(t *testing.T) {

	require := require.New(t)

	var app IAppDef

	wsName := NewQName("test", "workspace")

	cmdName := NewQName("test", "cmd")
	qrName := NewQName("test", "query")
	parName := NewQName("test", "param")
	resName := NewQName("test", "res")

	t.Run("Should be ok to build application with functions", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddObject(parName)
		_ = wsb.AddObject(resName)

		cmd := wsb.AddCommand(cmdName)
		cmd.SetEngine(ExtensionEngineKind_WASM)
		cmd.
			SetParam(parName).
			SetResult(resName)

		qry := wsb.AddQuery(qrName)
		qry.
			SetParam(parName).
			SetResult(QNameANY)

		a, err := adb.Build()
		require.NoError(err)

		app = a
		require.NotNil(app)
	})

	testWith := func(tested testedTypes) {
		t.Run("should be ok to enumerate functions", func(t *testing.T) {
			var names []QName
			for f := range Functions(tested.Types) {
				require.Equal(wsName, f.Workspace().QName())
				names = append(names, f.QName())
			}
			require.Len(names, 2)
			require.Equal([]QName{cmdName, qrName}, names)
		})

		t.Run("should be ok to find function by name", func(t *testing.T) {
			f := Function(tested.Type, cmdName)
			require.NotNil(f)
			require.Equal(cmdName, f.QName())
		})

		require.Nil(Function(tested.Type, NewQName("test", "unknown")), "Should be nil if unknown")
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}
