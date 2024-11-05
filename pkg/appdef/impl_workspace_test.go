/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"slices"
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

		wsb := adb.AddWorkspace(wsName)

		t.Run("should be ok to set workspace descriptor", func(t *testing.T) {
			_ = wsb.AddCDoc(descName)
			wsb.SetDescriptor(descName)
		})

		t.Run("should be ok to add some object to workspace", func(t *testing.T) {
			_ = wsb.AddObject(objName)
			wsb.AddType(objName)
		})

		require.NotNil(wsb.Workspace(), "should be ok to get workspace definition before build")
		require.Equal(descName, wsb.Workspace().Descriptor(), "should be ok to get workspace descriptor before build")

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("should be ok to enum workspaces", func(t *testing.T) {
		cnt := 0
		for ws := range app.Workspaces {
			cnt++
			switch ws.QName() {
			case SysWorkspaceQName:
			case wsName:
			default:
				require.Fail("unexpected workspace", "unexpected workspace «%v»", ws.QName())
			}
		}
		require.Equal(1+1, cnt) // system ws + test ws
	})

	t.Run("should be ok to find workspace", func(t *testing.T) {
		typ := app.Type(wsName)
		require.Equal(TypeKind_Workspace, typ.Kind())

		ws := app.Workspace(wsName)
		require.Equal(TypeKind_Workspace, ws.Kind())
		require.Equal(typ.(IWorkspace), ws)

		require.Equal(descName, ws.Descriptor(), "must be ok to get workspace descriptor")

		t.Run("should be ok to find structures in workspace", func(t *testing.T) {
			typ := ws.Type(objName)
			require.Equal(TypeKind_Object, typ.Kind())
			obj, ok := typ.(IObject)
			require.True(ok)
			require.Equal(Object(app, objName), obj)
			require.Equal(ws, obj.Workspace())

			require.Equal(NullType, ws.Type(NewQName("unknown", "type")), "must be NullType if unknown type")
		})

		t.Run("should be ok to enum workspace types", func(t *testing.T) {
			require.Equal(2, func() int {
				cnt := 0
				for typ := range ws.Types {
					switch typ.QName() {
					case descName, objName:
					default:
						require.Fail("unexpected type in workspace", "unexpected type «%v» in workspace «%v»", typ.QName(), ws.QName())
					}
					cnt++
				}
				return cnt
			}())
		})

		require.Nil(app.Workspace(NewQName("unknown", "workspace")), "must be nil if unknown workspace")
	})

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if unknown descriptor assigned to workspace", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			ws := adb.AddWorkspace(wsName)
			require.Panics(func() { ws.SetDescriptor(NewQName("unknown", "type")) },
				require.Is(ErrNotFoundError), require.Has("unknown.type"))
		})

		t.Run("if add unknown type to workspace", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			ws := adb.AddWorkspace(wsName)
			require.Panics(func() { ws.AddType(NewQName("unknown", "type")) },
				require.Is(ErrNotFoundError), require.Has("unknown.type"))
		})
	})
}

func Test_AppDef_AlterWorkspace(t *testing.T) {
	require := require.New(t)

	wsName := NewQName("test", "ws")
	objName := NewQName("test", "object")

	var adb IAppDefBuilder

	t.Run("should be ok to add workspace", func(t *testing.T) {
		adb = New()
		adb.AddPackage("test", "test.com/test")

		_ = adb.AddWorkspace(wsName)
	})

	t.Run("should be ok to alter workspace", func(t *testing.T) {
		wsb := adb.AlterWorkspace(wsName)
		_ = wsb.AddObject(objName)
	})

	t.Run("should be panic to alter unknown workspace", func(t *testing.T) {
		require.Panics(func() { _ = adb.AlterWorkspace(NewQName("test", "unknown")) })
		require.Panics(func() { _ = adb.AlterWorkspace(objName) })
	})

	t.Run("should be ok to build altered workspace", func(t *testing.T) {
		app, err := adb.Build()
		require.NoError(err)

		ws := app.Workspace(wsName)
		require.NotNil(ws, "should be ok to find workspace in app")

		require.NotNil(Object(ws, objName), "should be ok to find object in workspace")
	})
}

func Test_AppDef_SetDescriptor(t *testing.T) {
	require := require.New(t)

	t.Run("should be ok to add workspace with descriptor", func(t *testing.T) {
		wsName, descName := NewQName("test", "ws"), NewQName("test", "desc")

		adb := New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)
		_ = ws.AddCDoc(descName)
		ws.SetDescriptor(descName)

		app, err := adb.Build()
		require.NoError(err)

		t.Run("should be ok to find workspace by descriptor", func(t *testing.T) {
			ws := app.WorkspaceByDescriptor(descName)
			require.NotNil(ws)
			require.Equal(TypeKind_Workspace, ws.Kind())

			t.Run("should be nil if can't find workspace by descriptor", func(t *testing.T) {
				ws := app.WorkspaceByDescriptor(NewQName("test", "unknown"))
				require.Nil(ws)
			})
		})
	})

	t.Run("should be ok to change ws descriptor", func(t *testing.T) {
		wsName, descName, desc1Name := NewQName("test", "ws"), NewQName("test", "desc"), NewQName("test", "desc1")

		adb := New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)
		_ = ws.AddCDoc(descName)
		ws.SetDescriptor(descName)

		require.NotPanics(
			func() { ws.SetDescriptor(descName) },
			"should be ok to assign descriptor twice")

		_ = ws.AddCDoc(desc1Name)
		ws.SetDescriptor(desc1Name)

		app, err := adb.Build()
		require.NoError(err)

		t.Run("should be ok to find workspace by changed descriptor", func(t *testing.T) {
			ws := app.WorkspaceByDescriptor(desc1Name)
			require.NotNil(ws)
			require.Equal(TypeKind_Workspace, ws.Kind())
			require.Equal(wsName, ws.QName())

			require.Nil(app.WorkspaceByDescriptor(descName))
		})

		t.Run("should be ok to clear descriptor", func(t *testing.T) {
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

	t.Run("should be ok to add abstract workspace", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		desc := ws.AddCDoc(descName)
		desc.SetAbstract()
		ws.SetDescriptor(descName)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("should be ok to read abstract workspace", func(t *testing.T) {
		ws := app.Workspace(wsName)
		require.True(ws.Abstract())

		desc := CDoc(app, ws.Descriptor())
		require.True(desc.Abstract())
	})

	t.Run("should be error to set descriptor abstract after assign to workspace", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		desc := ws.AddCDoc(descName)
		ws.SetDescriptor(descName)

		desc.SetAbstract()

		_, err := adb.Build()
		require.ErrorIs(err, ErrIncompatibleError)
	})
}

func Test_WorkspaceInheritance(t *testing.T) {
	const wsCount = 8

	//                  ┌─── test.ws0 ◄─ test.ws1 ◄─ test.ws2
	//                  │
	// sys.Workspace ◄──┤
	//                  |                ┌─── test.ws4 ◄── test.ws6 ◄──┐
	//                  └─── test.ws3 ◄──┤                             ├─── test.ws7
	//                                   └─—— test.ws5 ◄───────────────┘

	require := require.New(t)

	wsName := func(idx int) QName { return NewQName("test", fmt.Sprintf("ws%d", idx)) }
	objName := func(idx int) QName { return NewQName("test", fmt.Sprintf("obj%d", idx)) }

	testADB := func() IAppDefBuilder {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		newWS := func(idx int, anc []int) {
			ws := adb.AddWorkspace(wsName(idx))
			if len(anc) > 0 {
				ancNames := make([]QName, len(anc), len(anc))
				for i, a := range anc {
					ancNames[i] = wsName(a)
				}
				ws.SetAncestors(ancNames[0], ancNames[1:]...)
			}
			_ = ws.AddObject(objName(idx))
		}

		newWS(0, nil)
		newWS(1, []int{0})
		newWS(2, []int{1})

		newWS(3, nil)
		newWS(4, []int{3})
		newWS(5, []int{3})
		newWS(6, []int{4})
		newWS(7, []int{5, 6})

		return adb
	}

	var app IAppDef

	t.Run("should be ok to add workspaces with inheritance", func(t *testing.T) {
		adb := testADB()
		a, err := adb.Build()
		require.NoError(err)
		app = a
	})

	t.Run("should be ok to read ancestors", func(t *testing.T) {
		tests := []struct {
			ws   int
			anc  []int
			ancR []int
		}{
			{0, []int{}, []int{}},
			{1, []int{0}, []int{0}},
			{2, []int{1}, []int{0, 1}},
			{3, []int{}, []int{}},
			{4, []int{3}, []int{3}},
			{5, []int{3}, []int{3}},
			{6, []int{4}, []int{3, 4}},
			{7, []int{5, 6}, []int{3, 4, 5, 6}},
		}
		for _, test := range tests {
			ws := app.Workspace(wsName(test.ws))

			t.Run("direct ancestors", func(t *testing.T) {
				want := make([]QName, len(test.anc), len(test.anc))
				for i, a := range test.anc {
					want[i] = wsName(a)
				}
				if len(want) == 0 {
					want = []QName{SysWorkspaceQName}
				}
				got := ws.Ancestors(false)
				require.EqualValues(want, got, "unexpected direct ancestors for «%v»: want %v, got: %v", ws, want, got)
			})

			t.Run("ancestors recursively", func(t *testing.T) {
				want := QNamesFrom(SysWorkspaceQName)
				for _, a := range test.ancR {
					want.Add(wsName(a))
				}
				got := ws.Ancestors(true)
				require.EqualValues(want, got, "unexpected recursive ancestors for «%v»: want %v, got: %v", ws, want, got)
			})
		}
	})

	t.Run("should be ok to check inheritance", func(t *testing.T) {
		tests := []struct {
			ws       int
			inherits []int
		}{
			{0, []int{0}},
			{1, []int{0, 1}},
			{2, []int{0, 1, 2}},

			{3, []int{3}},
			{4, []int{3, 4}},
			{5, []int{3, 5}},
			{6, []int{3, 4, 6}},
			{7, []int{3, 4, 5, 6, 7}},
		}
		for _, test := range tests {
			ws := app.Workspace(wsName(test.ws))
			for a := 0; a < wsCount; a++ {
				want := slices.Contains(test.inherits, a)
				got := ws.Inherits(wsName(a))
				require.Equal(want, got, "unexpected inheritance for «%v» from «%v»: want %v, got: %v", ws, wsName(a), want, got)
			}
			require.True(ws.Inherits(ws.QName()), "any workspace should inherit from itself")
			require.True(ws.Inherits(SysWorkspaceQName), "any workspace should inherit from sys.Workspace")
		}
	})

	t.Run("should be correspondence between Ancestors() and Inherits()", func(t *testing.T) {
		for idx := 0; idx < wsCount; idx++ {
			ws := app.Workspace(wsName(idx))
			for _, a := range ws.Ancestors(true) {
				require.True(ws.Inherits(a), "%v.Inherits(%v) returns false for ancestor workspace %v", ws, a, a)
			}
		}
	})

	t.Run("should be ok to find types in descendants", func(t *testing.T) {
		tests := []struct {
			ws      int
			objects []int
		}{
			{0, []int{0}},
			{1, []int{0, 1}},
			{2, []int{0, 1, 2}},

			{3, []int{3}},
			{4, []int{3, 4}},
			{5, []int{3, 5}},
			{6, []int{3, 4, 6}},
			{7, []int{3, 4, 5, 6, 7}},
		}
		for _, test := range tests {
			ws := app.Workspace(wsName(test.ws))
			for o := 0; o < wsCount; o++ {
				want := slices.Contains(test.objects, o)
				obj := Object(ws, objName(o))
				got := obj != nil
				require.Equal(want, got, "unexpected %v.Object(%v) != nil result: want %v, got: %v", ws, objName(o), want, got)
				if got {
					require.True(ws.Inherits(obj.Workspace().QName()),
						"any type from workspace should be owned by this workspace or by its ancestors")
				}
			}
			require.NotNil(ws.Type(SysData_int32), "should be ok to find system type in workspace")
		}
	})

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if ancestor workspace is not found", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")

			ws := adb.AddWorkspace(wsName(0))
			unknown := NewQName("test", "unknown")
			require.Panics(func() { ws.SetAncestors(unknown) },
				require.Is(ErrNotFoundError), require.Has(unknown))
			require.Panics(func() { ws.SetAncestors(SysData_QName) },
				require.Is(ErrNotFoundError), require.Has(SysData_QName))
		})

		t.Run("if circular inheritance detected", func(t *testing.T) {
			tests := []struct {
				ws        int
				circulars []int
			}{
				{0, []int{0, 1, 2}},
				{1, []int{1, 2}},
				{2, []int{2}},

				{3, []int{3, 4, 5, 6, 7}},
				{4, []int{4, 6, 7}},
				{5, []int{5, 7}},
				{6, []int{6, 7}},
				{7, []int{7}},
			}

			for _, test := range tests {
				for a := 0; a < wsCount; a++ {
					anc := wsName(a)

					adb := testADB()
					ws := adb.AlterWorkspace(wsName(test.ws))

					if slices.Contains(test.circulars, a) {
						require.Panics(func() { ws.SetAncestors(anc) },
							require.Is(ErrUnsupportedError), require.Has("Circular inheritance"), require.Has(anc), require.Has(wsName(test.ws)))
					} else {
						require.NotPanics(func() { ws.SetAncestors(anc) },
							"unexpected panics while set ancestors for «%v» from «%v»", wsName(test.ws), anc)
					}
				}
			}
		})
	})
}
