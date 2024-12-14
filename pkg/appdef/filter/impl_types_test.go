/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Types(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	dataName := appdef.NewQName("test", "data")

	app := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddData(dataName, appdef.DataKind_int32, appdef.NullQName, appdef.MinIncl(0))

		app, err := adb.Build()

		require.NoError(err)
		return app
	}()

	flt := filter.Types(wsName, appdef.TypeKind_Data)

	ws := app.Workspace(wsName)

	data := appdef.Data(ws.Type, dataName)
	require.NotNil(data, "Data should be found")
	require.True(flt.Match(data), "Data should be matched")

	sysInt32 := appdef.Data(ws.Type, appdef.SysDataName(appdef.DataKind_int32))
	require.NotNil(sysInt32, "system sys.Int32 should be found")
	require.False(flt.Match(sysInt32), "system data should not be matched")

	t.Run("should test filter with no workspace", func(t *testing.T) {
		flt := filter.Types(appdef.NullQName, appdef.TypeKind_Data)
		require.True(flt.Match(sysInt32), "system data should be matched")
	})
}
