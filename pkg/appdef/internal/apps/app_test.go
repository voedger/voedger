/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apps_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/internal/apps"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_NewAppDef(t *testing.T) {
	require := require.New(t)

	app := apps.NewAppDef()
	require.NotNil(app)
	var _ appdef.IAppDef = app // check interface compatibility

	adb := apps.NewAppDefBuilder(app)
	require.NotNil(adb)
	var _ appdef.IAppDefBuilder = adb // check interface compatibility

	require.Equal(app, adb.AppDef(), "should be ok to obtain AppDef(*) before build")

	t.Run("Should be ok to obtain empty app", func(t *testing.T) {
		app, err := adb.Build()
		require.NoError(err)
		require.NotNil(app)

		t.Run("should be ok to read sys package", func(t *testing.T) {
			require.Equal([]string{appdef.SysPackage}, app.PackageLocalNames())
			require.Equal(appdef.SysPackagePath, app.PackageFullPath(appdef.SysPackage))
			require.Equal(appdef.SysPackage, app.PackageLocalName(appdef.SysPackagePath))
		})

		t.Run("should be ok to read sys workspace", func(t *testing.T) {
			ws := app.Workspace(appdef.SysWorkspaceQName)
			require.NotNil(ws)
			require.Equal(appdef.SysWorkspaceQName, ws.QName())
			require.Equal(appdef.TypeKind_Workspace, ws.Kind())

			require.Equal(ws, app.Type(appdef.SysWorkspaceQName))
		})

		t.Run("should be ok to read sys types", func(t *testing.T) {
			require.Equal(appdef.NullType, app.Type(appdef.NullQName))
			require.Equal(appdef.AnyType, app.Type(appdef.QNameANY))
		})

		t.Run("should be ok to read sys data types", func(t *testing.T) {
			require.Equal(appdef.SysData_RecordID, appdef.Data(app.Type, appdef.SysData_RecordID).QName())
			require.Equal(appdef.SysData_String, appdef.Data(app.Type, appdef.SysData_String).QName())
			require.Equal(appdef.SysData_bytes, appdef.Data(app.Type, appdef.SysData_bytes).QName())
		})
	})

	t.Run("Should be ok to alter workspace after build", func(t *testing.T) {
		wsb := adb.AlterWorkspace(appdef.SysWorkspaceQName)
		require.NotNil(wsb)
	})
}

func Test_AppDefBuilder_MustBuild(t *testing.T) {
	require := require.New(t)

	require.NotNil(builder.New().MustBuild(), "Should be ok if no errors in builder")

	t.Run("should panic if errors in builder", func(t *testing.T) {
		adb := builder.New()
		adb.AddWorkspace(appdef.NewQName("test", "workspace")).AddView(appdef.NewQName("test", "emptyView"))

		require.Panics(func() { _ = adb.MustBuild() },
			require.Is(appdef.ErrMissedError),
			require.Has("emptyView"),
		)
	})
}
