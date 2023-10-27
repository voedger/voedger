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

	wsName, descName, objName := NewQName("test", "ws"), NewQName("test", "desc"), NewQName("test", "object")

	var app IAppDef

	t.Run("must be ok to add workspace", func(t *testing.T) {
		appDef := New()
		ws := appDef.AddWorkspace(wsName)

		t.Run("must be ok to set workspace descriptor", func(t *testing.T) {
			require.Equal(NullQName, ws.Descriptor())

			_ = appDef.AddCDoc(descName)
			ws.SetDescriptor(descName)
		})

		t.Run("must be ok to add some object to workspace", func(t *testing.T) {
			_ = appDef.AddObject(objName)
			ws.AddType(objName)
		})

		a, err := appDef.Build()
		require.NoError(err)

		app = a
	})

	t.Run("must be ok to find workspace", func(t *testing.T) {
		typ := app.Type(wsName)
		require.Equal(TypeKind_Workspace, typ.Kind())

		ws := app.Workspace(wsName)
		require.Equal(TypeKind_Workspace, ws.Kind())
		require.Equal(typ.(IWorkspace), ws)

		require.Equal(descName, ws.Descriptor(), "must be ok to get workspace descriptor")

		t.Run("must be ok to find object in workspace", func(t *testing.T) {
			typ := ws.Type(objName)
			require.NotNil(typ)
			require.Equal(TypeKind_Object, typ.Kind())

			obj, ok := typ.(IObject)
			require.True(ok)
			require.NotNil(obj)
			require.Equal(app.Object(objName), obj)

			require.Equal(NullType, ws.Type(NewQName("unknown", "type")), "must be NullType if unknown type")
		})

		t.Run("must be ok to enum workspace types", func(t *testing.T) {
			require.Equal(1, func() int {
				cnt := 0
				ws.Types(func(typ IType) {
					switch typ.QName() {
					case objName:
					default:
						require.Fail("unexpected type in workspace", "unexpected type «%v» in workspace «%v»", typ.QName(), ws.QName())
					}
					cnt++
				})
				return cnt
			}())
		})

		require.Nil(app.Workspace(NewQName("unknown", "workspace")), "must be nil if unknown workspace")
	})

	t.Run("must be panic if unknown descriptor assigned to workspace", func(t *testing.T) {
		appDef := New()
		ws := appDef.AddWorkspace(wsName)
		require.Panics(func() { ws.SetDescriptor(NewQName("unknown", "type")) })
	})

	t.Run("must be panic if add unknown type to workspace", func(t *testing.T) {
		appDef := New()
		ws := appDef.AddWorkspace(wsName)
		require.Panics(func() { ws.AddType(NewQName("unknown", "type")) })
	})
}

func Test_AppDef_AddWorkspaceAbstract(t *testing.T) {
	require := require.New(t)

	wsName, descName := NewQName("test", "ws"), NewQName("test", "desc")

	var app IAppDef

	t.Run("must be ok to add abstract workspace", func(t *testing.T) {
		appDef := New()
		ws := appDef.AddWorkspace(wsName)

		desc := appDef.AddCDoc(descName)
		desc.SetAbstract()
		ws.SetDescriptor(descName)

		a, err := appDef.Build()
		require.NoError(err)

		app = a
	})

	t.Run("must be ok to read abstract workspace", func(t *testing.T) {
		ws := app.Workspace(wsName)
		require.True(ws.Abstract())

		desc := app.CDoc(ws.Descriptor())
		require.True(desc.Abstract())
	})

	t.Run("must be error to set descriptor abstract after assign to workspace", func(t *testing.T) {
		appDef := New()
		ws := appDef.AddWorkspace(wsName)

		desc := appDef.AddCDoc(descName)
		ws.SetDescriptor(descName)

		desc.SetAbstract()

		_, err := appDef.Build()
		require.ErrorIs(err, ErrWorkspaceShouldBeAbstract)

		t.Run("but must be ok to fix this error by making the workspace abstract", func(t *testing.T) {
			ws.SetAbstract()
			_, err := appDef.Build()
			require.NoError(err)
		})
	})
}
