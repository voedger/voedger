/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package extensions_test

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Extensions(t *testing.T) {

	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")

	cmdName := appdef.NewQName("test", "cmd")
	qrName := appdef.NewQName("test", "query")
	prjName := appdef.NewQName("test", "projector")
	parName := appdef.NewQName("test", "param")
	resName := appdef.NewQName("test", "res")

	sysLog := appdef.NewQName("sys", "plog")
	sysRecords := appdef.NewQName("sys", "records")
	sysViews := appdef.NewQName("sys", "views")
	viewName := appdef.NewQName("test", "view")

	t.Run("Should be ok to build application with extensions", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddObject(parName)
		_ = wsb.AddObject(resName)

		cmd := wsb.AddCommand(cmdName)
		cmd.SetEngine(appdef.ExtensionEngineKind_WASM).
			SetName(`command`)
		cmd.States().Add(sysLog)
		cmd.Intents().Add(sysRecords)
		cmd.
			SetParam(parName).
			SetResult(resName)

		qry := wsb.AddQuery(qrName)
		qry.
			SetParam(parName).
			SetResult(appdef.QNameANY)

		prj := wsb.AddProjector(prjName)
		prj.Events().Add([]appdef.OperationKind{appdef.OperationKind_Execute}, filter.QNames(cmdName))
		prj.Intents().
			Add(sysViews, viewName)

		v := wsb.AddView(viewName)
		v.Key().PartKey().AddField("pk", appdef.DataKind_int64)
		v.Key().ClustCols().AddField("cc", appdef.DataKind_string)
		v.Value().AddField("f1", appdef.DataKind_int64, true)

		a, err := adb.Build()
		require.NoError(err)

		app = a
		require.NotNil(app)
	})

	t.Run("should be ok to enumerate extensions", func(t *testing.T) {
		var extNames []appdef.QName
		for ex := range appdef.Extensions(app.Types()) {
			require.Equal(wsName, ex.Workspace().QName())
			extNames = append(extNames, ex.QName())
		}
		require.Len(extNames, 3)
		require.Equal([]appdef.QName{cmdName, prjName, qrName}, extNames)
	})

	t.Run("should be ok to find extension by name", func(t *testing.T) {
		ext := appdef.Extension(app.Type, cmdName)
		require.NotNil(ext)
		require.Equal(cmdName, ext.QName())
		require.Equal(appdef.ExtensionEngineKind_WASM, ext.Engine())
		require.Equal(`command`, ext.Name())
		require.Equal(`WASM-Command «test.cmd»`, fmt.Sprint(ext))
	})

	require.Nil(appdef.Extension(app.Type, appdef.NewQName("test", "unknown")), "should be nil if unknown")
}

func Test_ExtensionsPanics(t *testing.T) {
	require := require.New(t)

	t.Run("should be panics", func(t *testing.T) {
		apb := builder.New()
		apb.AddPackage("test", "test.com/test")
		wsb := apb.AddWorkspace(appdef.NewQName("test", "workspace"))

		cmd := wsb.AddCommand(appdef.NewQName("test", "cmd"))

		require.Panics(func() {
			cmd.SetEngine(appdef.ExtensionEngineKind_count) // <-- out of bounds
		}, require.Is(appdef.ErrOutOfBoundsError), require.HasAll(cmd, "count"))

		require.Panics(func() {
			cmd.SetName("") // <-- missed
		}, require.Is(appdef.ErrMissedError), require.HasAll(cmd, "name"))

		require.Panics(func() {
			cmd.SetName("naked 🔫") // <-- invalid
		}, require.Is(appdef.ErrInvalidError), require.HasAll(cmd, "naked 🔫"))
	})
}
