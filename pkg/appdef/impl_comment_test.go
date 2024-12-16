/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_SetTypeComment(t *testing.T) {
	require := require.New(t)

	t.Run("should be ok to set type comment", func(t *testing.T) {
		wsName, objName := appdef.NewQName("test", "ws"), appdef.NewQName("test", "object")

		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)
		adb.SetTypeComment(wsName, "workspace comment")

		_ = ws.AddObject(objName)
		ws.SetTypeComment(objName, "object comment")

		app, err := adb.Build()
		require.NoError(err)

		t.Run("should be ok to find comment", func(t *testing.T) {
			ws := app.Workspace(wsName)
			require.Equal("workspace comment", ws.Comment())

			obj := appdef.Object(ws.Type, objName)
			require.Equal("object comment", obj.Comment())
		})
	})

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if unknown type", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")

			require.Panics(func() { adb.SetTypeComment(appdef.NewQName("test", "unknown"), "ups") })

			wsb := adb.AddWorkspace(appdef.NewQName("test", "ws"))
			require.Panics(func() { wsb.SetTypeComment(appdef.NewQName("test", "unknown"), "ups") })
			require.Panics(func() { wsb.SetTypeComment(appdef.SysData_bool, "ups") }, "should panic if not local type")
		})
	})
}
