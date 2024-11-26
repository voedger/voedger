/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_SetTypeComment(t *testing.T) {
	require := require.New(t)

	t.Run("should be ok to set type comment", func(t *testing.T) {
		wsName, objName := NewQName("test", "ws"), NewQName("test", "object")

		adb := New()
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

			obj := Object(ws.Type, objName)
			require.Equal("object comment", obj.Comment())
		})
	})

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if unknown type", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")

			require.Panics(func() { adb.SetTypeComment(NewQName("test", "unknown"), "ups") })

			wsb := adb.AddWorkspace(NewQName("test", "ws"))
			require.Panics(func() { wsb.SetTypeComment(NewQName("test", "unknown"), "ups") })
			require.Panics(func() { wsb.SetTypeComment(SysData_bool, "ups") }, "should panic if not local type")
		})
	})
}
