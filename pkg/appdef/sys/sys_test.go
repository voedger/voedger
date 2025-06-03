/*
 * Copyright (c) 2025-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package sys_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/apps"
	"github.com/voedger/voedger/pkg/appdef/sys"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_SysPackage(t *testing.T) {
	require := require.New(t)

	app := apps.NewAppDef()
	require.NotNil(app)

	adb := apps.NewAppDefBuilder(app)
	require.NotNil(adb)

	sys.MakeSysPackage(adb)

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

			require.Equal(sys.SysWSKind, ws.Descriptor(), "should be ok to read sys workspace descriptor")
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

		t.Run("should be ok to read sys views", func(t *testing.T) {
			t.Run("Projection offsets view", func(t *testing.T) {
				v := appdef.View(app.Type, sys.ProjectionOffsetsView.Name)
				require.NotNil(v)
				require.NotNil(v.Key().PartKey().Field(sys.ProjectionOffsetsView.Fields.Partition))
				require.NotNil(v.Key().ClustCols().Field(sys.ProjectionOffsetsView.Fields.Projector))
				require.NotNil(v.Value().Field(sys.ProjectionOffsetsView.Fields.Offset))
			})

			t.Run("Child workspaces IDs view", func(t *testing.T) {
				v := appdef.View(app.Type, sys.NextBaseWSIDView.Name)
				require.NotNil(v)
				require.NotNil(v.Key().PartKey().Field(sys.NextBaseWSIDView.Fields.PartKeyDummy))
				require.NotNil(v.Key().ClustCols().Field(sys.NextBaseWSIDView.Fields.ClustColDummy))
				require.NotNil(v.Value().Field(sys.NextBaseWSIDView.Fields.NextBaseWSID))
			})
		})
	})
}

func Test_RecordsRegistryViewFields_CrackID(t *testing.T) {
	require := require.New(t)
	require.EqualValues(123, sys.RecordsRegistryView.Fields.CrackID((123<<18)+456)) // 123 is the high part, 456 is the low part
}
