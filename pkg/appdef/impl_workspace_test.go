/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_AddWorkspace(t *testing.T) {
	require := require.New(t)

	wsName, descName, objName := NewQName("test", "ws"), NewQName("test", "desc"), NewQName("test", "object")

	var app IAppDef

	t.Run("should be ok to add workspace", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		t.Run("must be ok to set workspace descriptor", func(t *testing.T) {
			_ = adb.AddCDoc(descName)
			ws.SetDescriptor(descName)
		})

		t.Run("must be ok to add some object to workspace", func(t *testing.T) {
			_ = adb.AddObject(objName)
			ws.AddType(objName)
		})

		require.NotNil(ws.Workspace(), "should be ok to get workspace definition before build")
		require.Equal(ws.Workspace().Descriptor(), descName, "should be ok to get workspace descriptor before build")

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("should be ok to enum workspaces", func(t *testing.T) {
		cnt := 0
		for ws := range app.Workspaces {
			cnt++
			switch ws.QName() {
			case wsName:
			default:
				require.Fail("unexpected workspace", "unexpected workspace «%v»", ws.QName())
			}
		}
		require.Equal(1, cnt)
	})

	t.Run("should be breakable enum workspaces", func(t *testing.T) {
		cnt := 0
		for range app.Workspaces {
			cnt++
			break
		}
		require.Equal(1, cnt)
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

		t.Run("should be ok to enum workspace types", func(t *testing.T) {
			require.Equal(1, func() int {
				cnt := 0
				for typ := range ws.Types {
					switch typ.QName() {
					case objName:
					default:
						require.Fail("unexpected type in workspace", "unexpected type «%v» in workspace «%v»", typ.QName(), ws.QName())
					}
					cnt++
				}
				return cnt
			}())
		})

		t.Run("Should be ok to break enum workspace types", func(t *testing.T) {
			cnt := 0
			for range ws.Types {
				cnt++
				break
			}
			require.Equal(1, cnt)
		})

		require.Nil(app.Workspace(NewQName("unknown", "workspace")), "must be nil if unknown workspace")
	})

	t.Run("must be panic if unknown descriptor assigned to workspace", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(wsName)
		require.Panics(func() { ws.SetDescriptor(NewQName("unknown", "type")) },
			require.Is(ErrNotFoundError), require.Has("unknown.type"))
	})

	t.Run("must be panic if add unknown type to workspace", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(wsName)
		require.Panics(func() { ws.AddType(NewQName("unknown", "type")) },
			require.Is(ErrNotFoundError), require.Has("unknown.type"))
	})
}

func Test_AppDef_SetDescriptor(t *testing.T) {
	require := require.New(t)

	t.Run("must be ok to add workspace with descriptor", func(t *testing.T) {
		wsName, descName := NewQName("test", "ws"), NewQName("test", "desc")

		adb := New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)
		_ = adb.AddCDoc(descName)
		ws.SetDescriptor(descName)

		app, err := adb.Build()
		require.NoError(err)

		t.Run("must be ok to find workspace by descriptor", func(t *testing.T) {
			ws := app.WorkspaceByDescriptor(descName)
			require.NotNil(ws)
			require.Equal(TypeKind_Workspace, ws.Kind())

			t.Run("must be nil if can't find workspace by descriptor", func(t *testing.T) {
				ws := app.WorkspaceByDescriptor(NewQName("test", "unknown"))
				require.Nil(ws)
			})
		})
	})

	t.Run("must be ok to change ws descriptor", func(t *testing.T) {
		wsName, descName, desc1Name := NewQName("test", "ws"), NewQName("test", "desc"), NewQName("test", "desc1")

		adb := New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)
		_ = adb.AddCDoc(descName)
		ws.SetDescriptor(descName)

		t.Run("must be ok to assign descriptor twice", func(t *testing.T) {
			ws.SetDescriptor(descName)
		})

		_ = adb.AddCDoc(desc1Name)
		ws.SetDescriptor(desc1Name)

		app, err := adb.Build()
		require.NoError(err)

		t.Run("must be ok to find workspace by changed descriptor", func(t *testing.T) {
			ws := app.WorkspaceByDescriptor(desc1Name)
			require.NotNil(ws)
			require.Equal(TypeKind_Workspace, ws.Kind())
			require.Equal(wsName, ws.QName())

			require.Nil(app.WorkspaceByDescriptor(descName))
		})

		t.Run("must be ok to clear descriptor", func(t *testing.T) {
			ws.SetDescriptor(NullQName)

			app, err = adb.Build()
			require.NoError(err)

			require.Nil(app.WorkspaceByDescriptor(descName))
			require.Nil(app.WorkspaceByDescriptor(desc1Name))

			ws := app.Workspace(wsName)
			require.NotNil(ws)
			require.Equal(NullQName, ws.Descriptor())
		})
	})
}

func Test_AppDef_AddWorkspaceAbstract(t *testing.T) {
	require := require.New(t)

	wsName, descName := NewQName("test", "ws"), NewQName("test", "desc")

	var app IAppDef

	t.Run("must be ok to add abstract workspace", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		desc := adb.AddCDoc(descName)
		desc.SetAbstract()
		ws.SetDescriptor(descName)

		a, err := adb.Build()
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
		adb := New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		desc := adb.AddCDoc(descName)
		ws.SetDescriptor(descName)

		desc.SetAbstract()

		_, err := adb.Build()
		require.ErrorIs(err, ErrIncompatibleError)

		t.Run("but must be ok to fix this error by making the workspace abstract", func(t *testing.T) {
			ws.SetAbstract()
			_, err := adb.Build()
			require.NoError(err)
		})
	})
}
