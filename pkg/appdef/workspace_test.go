/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddWorkspace(t *testing.T) {
	require := require.New(t)

	wsName, objName := NewQName("test", "ws"), NewQName("test", "object")

	var app IAppDef

	t.Run("must be ok to add workspace", func(t *testing.T) {
		appDef := New()
		ws := appDef.AddWorkspace(wsName)

		t.Run("must be ok to add some object to workspace", func(t *testing.T) {
			_ = appDef.AddObject(objName)
			ws.AddDef(objName)
		})

		a, err := appDef.Build()
		require.NoError(err)

		app = a
	})

	t.Run("must be ok to find workspace", func(t *testing.T) {
		def := app.Def(wsName)
		require.Equal(DefKind_Workspace, def.Kind())

		ws := app.Workspace(wsName)
		require.Equal(DefKind_Workspace, ws.Kind())
		require.Equal(def.(IWorkspace), ws)

		t.Run("must be ok to find object in workspace", func(t *testing.T) {
			def := ws.Def(objName)
			require.NotNil(def)
			require.Equal(DefKind_Object, def.Kind())

			obj, ok := def.(IObject)
			require.True(ok)
			require.NotNil(obj)
			require.Equal(app.Object(objName), obj)

			require.Nil(ws.Def(NewQName("unknown", "definition")), "must be nil if unknown definition")
		})

		t.Run("must be ok to enum workspace definitions", func(t *testing.T) {
			require.Equal(1, func() int {
				cnt := 0
				ws.Defs(func(i IDef) {
					switch i.QName() {
					case objName:
					default:
						require.Fail("unexpected definition in workspace", "unexpected definition «%v» in workspace «%v»", i.QName(), ws.QName())
					}
					cnt++
				})
				return cnt
			}())
		})

		require.Nil(app.Workspace(NewQName("unknown", "workspace")), "must be nil if unknown workspace")
	})

	t.Run("must be panic if add unknown definition to workspace", func(t *testing.T) {
		appDef := New()
		ws := appDef.AddWorkspace(wsName)
		require.Panics(func() { ws.AddDef(NewQName("unknown", "def")) })
	})
}
