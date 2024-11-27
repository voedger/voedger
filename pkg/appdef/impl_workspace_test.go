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

func requireSortedTypes(t *testing.T, types []IType) {
	require := require.New(t)
	require.True(slices.IsSortedFunc(types, func(a, b IType) int {
		return CompareQName(a.QName(), b.QName())
	}), "types should be sorted")
}

func Test_AppDef_AddWorkspace(t *testing.T) {
	require := require.New(t)

	wsName, descName, objName := NewQName("test", "ws"), NewQName("test", "desc"), NewQName("test", "object")
	wsPName, descPName, objPName := NewQName("test", "parentWs"), NewQName("test", "parentDesc"), NewQName("test", "parentObject")

	var app IAppDef

	t.Run("should be ok to add workspace with used workspace", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		t.Run("should be ok to add workspace", func(t *testing.T) {
			wsb := adb.AddWorkspace(wsName)
			_ = wsb.AddCDoc(descName)
			wsb.SetDescriptor(descName)
			_ = wsb.AddObject(objName)

			require.NotNil(wsb.Workspace(), "should be ok to get workspace definition before build")
			require.Equal(descName, wsb.Workspace().Descriptor(), "should be ok to get workspace descriptor before build")
		})

		t.Run("should be ok to add parent workspace", func(t *testing.T) {
			wsb := adb.AddWorkspace(wsPName)
			_ = wsb.AddCDoc(descPName)
			wsb.SetDescriptor(descPName)
			_ = wsb.AddObject(objPName)

			wsb.UseWorkspace(wsName)
		})

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
			case wsPName:
			default:
				require.Fail("unexpected workspace", "unexpected workspace «%v»", ws.QName())
			}
		}
		require.Equal(1+1+1, cnt) // system ws + test ws + parent ws
	})

	t.Run("should be ok to find workspace", func(t *testing.T) {
		typ := app.Type(wsName)
		require.Equal(TypeKind_Workspace, typ.Kind())

		ws := app.Workspace(wsName)
		require.Equal(TypeKind_Workspace, ws.Kind())
		require.Equal(typ.(IWorkspace), ws)

		require.Equal(descName, ws.Descriptor())

		t.Run("should be ok to find by Type()", func(t *testing.T) {
			require.Equal(app.Type(objName), ws.Type(objName), "should find local type")
			require.Equal(SysData(app.Type, DataKind_int32), SysData(ws.Type, DataKind_int32), "should find type from ancestor")
			require.Equal(NullType, ws.Type(NewQName("unknown", "object")))
		})

		t.Run("should be ok to enum Types()", func(t *testing.T) {
			types := slices.Collect(ws.Types)
			requireSortedTypes(t, types)
			require.Contains(types, app.Type(descName), "ws types should contain descriptor")
			require.Contains(types, app.Type(objName), "ws types should contain local types")
			require.Contains(types, app.Type(SysData_bool), "ws types should contain types from ancestor")
		})

		t.Run("should be ok to find by LocalType()", func(t *testing.T) {
			require.NotNil(ws.LocalType(objName))
			require.Equal(NullType, ws.LocalType(SysData_bool))
		})

		t.Run("should be ok to enum LocalTypes()", func(t *testing.T) {
			types := slices.Collect(ws.LocalTypes)
			requireSortedTypes(t, types)
			require.Equal([]IType{app.Type(descName), app.Type(objName)}, types)
		})
	})

	t.Run("should be ok to find parent workspace", func(t *testing.T) {
		pWs := app.Workspace(wsPName)
		require.Equal(TypeKind_Workspace, pWs.Kind())

		t.Run("should be ok to find by Type()", func(t *testing.T) {
			require.Equal(app.Type(wsName), pWs.Type(wsName), "should find used workspace in parent")
			require.Equal(NullType, pWs.Type(objName), "should not found object from used workspace in parent")
		})

		t.Run("should be ok to enum Types()", func(t *testing.T) {
			types := slices.Collect(pWs.Types)
			requireSortedTypes(t, types)
			require.Contains(types, app.Type(wsName), "ws types should contain used workspace")
			require.NotContains(types, app.Type(objName), "ws types should not contain type from used workspace", pWs, objName)
		})
	})

	require.Nil(app.Workspace(NewQName("unknown", "workspace")), "should be nil if unknown workspace")

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if unknown descriptor assigned to workspace", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			ws := adb.AddWorkspace(wsName)
			require.Panics(func() { ws.SetDescriptor(NewQName("unknown", "type")) },
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

		require.NotNil(Object(ws.Type, objName), "should be ok to find object in workspace")
	})
}

func Test_Workspace_SetDescriptor(t *testing.T) {
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

		desc := CDoc(app.Type, ws.Descriptor())
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

	testAncestors := [wsCount]struct {
		anc []int
	}{
		{[]int{}},     //0
		{[]int{0}},    //1
		{[]int{1}},    //2
		{[]int{}},     //3
		{[]int{3}},    //4
		{[]int{3}},    //5
		{[]int{4}},    //6
		{[]int{5, 6}}, //7
	}

	require := require.New(t)

	wsName := func(idx int) QName { return NewQName("test", fmt.Sprintf("ws%d", idx)) }
	objName := func(idx int) QName { return NewQName("test", fmt.Sprintf("obj%d", idx)) }

	testADB := func() IAppDefBuilder {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		for idx := 0; idx < wsCount; idx++ {
			ws := adb.AddWorkspace(wsName(idx))
			anc := testAncestors[idx].anc
			if len(anc) > 0 {
				ancNames := make([]QName, len(anc), len(anc))
				for i, a := range anc {
					ancNames[i] = wsName(a)
				}
				ws.SetAncestors(ancNames[0], ancNames[1:]...)
			}
			_ = ws.AddObject(objName(idx))
		}
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
		for wsIdx, test := range testAncestors {
			ws := app.Workspace(wsName(wsIdx))

			want := make([]QName, len(test.anc), len(test.anc))
			for i, a := range test.anc {
				want[i] = wsName(a)
			}
			if len(want) == 0 {
				want = []QName{SysWorkspaceQName}
			}
			i := 0
			for a := range ws.Ancestors {
				require.EqualValues(want[i], a.QName(), "unexpected ancestor for «%v»: want %v, got: %v", ws, want[i], a.QName())
				i++
			}
			require.Len(want, i, "unexpected count of ancestors for «%v»", ws)
		}
	})

	t.Run("should be ok to check inheritance", func(t *testing.T) {
		tests := []struct {
			inherits []int
		}{
			{[]int{0}},
			{[]int{0, 1}},
			{[]int{0, 1, 2}},

			{[]int{3}},
			{[]int{3, 4}},
			{[]int{3, 5}},
			{[]int{3, 4, 6}},
			{[]int{3, 4, 5, 6, 7}},
		}
		for wsIdx, test := range tests {
			ws := app.Workspace(wsName(wsIdx))
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
			for a := range ws.Ancestors {
				require.True(ws.Inherits(a.QName()), "%v.Inherits(%v) returns false for ancestor workspace", ws, a.QName())
			}
		}
	})

	t.Run("should be ok to find ancestor types in descendants", func(t *testing.T) {
		tests := []struct {
			objects []int
		}{
			{[]int{0}},
			{[]int{0, 1}},
			{[]int{0, 1, 2}},
			{[]int{3}},
			{[]int{3, 4}},
			{[]int{3, 5}},
			{[]int{3, 4, 6}},
			{[]int{3, 4, 5, 6, 7}},
		}
		for wsIdx, test := range tests {
			ws := app.Workspace(wsName(wsIdx))
			for o := 0; o < wsCount; o++ {
				want := slices.Contains(test.objects, o)
				obj := Object(ws.Type, objName(o))
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

	t.Run("should be ok to iterate types", func(t *testing.T) {
		for wsIdx := 0; wsIdx < wsCount; wsIdx++ {
			ws := app.Workspace(wsName(wsIdx))
			types := slices.Collect(ws.Types)
			requireSortedTypes(t, types)
			for a := range ws.Ancestors {
				for t := range a.Types {
					require.Contains(types, t, "%v types should contains type %v from ancestor %v", ws, t, a)
				}
			}
			require.True(slices.IsSortedFunc(types, func(a, b IType) int {
				return CompareQName(a.QName(), b.QName())
			}), "%v types should be sorted", ws)
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
				circulars []int
			}{
				{[]int{0, 1, 2}},       // 0
				{[]int{1, 2}},          // 1
				{[]int{2}},             // 2
				{[]int{3, 4, 5, 6, 7}}, // 3
				{[]int{4, 6, 7}},       // 4
				{[]int{5, 7}},          // 5
				{[]int{6, 7}},          // 6
				{[]int{7}},             // 7
			}

			for wsIdx, test := range tests {
				for a := 0; a < wsCount; a++ {
					anc := wsName(a)

					adb := testADB()
					ws := adb.AlterWorkspace(wsName(wsIdx))

					if slices.Contains(test.circulars, a) {
						require.Panics(func() { ws.SetAncestors(anc) },
							require.Is(ErrUnsupportedError), require.Has("Circular inheritance"), require.Has(anc), require.Has(wsName(wsIdx)))
					} else {
						require.NotPanics(func() { ws.SetAncestors(anc) },
							"unexpected panics while set ancestors for «%v» from «%v»", wsName(wsIdx), anc)
					}
				}
			}
		})
	})
}

func Test_WorkspaceUsage(t *testing.T) {
	const wsCount = 7

	// WS0 —─► WS1 ──► WS2     ┌———————┐
	//  │              |       |       |
	//  │              ▼       ▼       |
	//  └─———► WS3 ──► WS4 ──► WS5 ──► WS6

	testUses := [wsCount]struct {
		use []int
	}{
		{[]int{1, 3}}, //0
		{[]int{2}},    //1
		{[]int{4}},    //2
		{[]int{4}},    //3
		{[]int{5}},    //4
		{[]int{6}},    //5
		{[]int{5}},    //6
	}

	require := require.New(t)

	wsName := func(idx int) QName { return NewQName("test", fmt.Sprintf("ws%d", idx)) }
	objName := func(idx int) QName { return NewQName("test", fmt.Sprintf("obj%d", idx)) }

	testADB := func() IAppDefBuilder {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		for i := 0; i < wsCount; i++ {
			ws := adb.AddWorkspace(wsName(i))
			_ = ws.AddObject(objName(i))
		}
		for idx, test := range testUses {
			ws := adb.AlterWorkspace(wsName(idx))
			if len(test.use) > 0 {
				useNames := make([]QName, len(test.use), len(test.use))
				for i, a := range test.use {
					useNames[i] = wsName(a)
				}
				ws.UseWorkspace(useNames[0], useNames[1:]...)
			}
		}
		return adb
	}

	var app IAppDef

	t.Run("should be ok to add workspaces with usages", func(t *testing.T) {
		adb := testADB()
		a, err := adb.Build()
		require.NoError(err)
		app = a
	})

	t.Run("should be ok to to read workspace usage", func(t *testing.T) {
		for idx, test := range testUses {
			ws := app.Workspace(wsName(idx))
			want := make([]QName, len(test.use), len(test.use))
			for i, u := range test.use {
				want[i] = wsName(u)
			}
			i := 0
			for u := range ws.UsedWorkspaces {
				require.Equal(want[i], u.QName(), "unexpected %v UsedWorkspaces[%d] for: want %v, got: %v", ws, i, want[i], u.QName())
				i++
			}
			require.Len(want, i, "unexpected count of used workspaces for «%v»", ws)
		}
	})

	t.Run("should be ok to read types from usages", func(t *testing.T) {
		for idx, test := range testUses {
			ws := app.Workspace(wsName(idx))
			t.Run("should be ok to find used workspaces by Type()", func(t *testing.T) {
				for u := 0; u < wsCount; u++ {
					used := wsName(u)
					got := ws.Type(used)
					if slices.Contains(test.use, u) {
						require.Equal(used, got.QName(), "(%v).Type() should find %v", ws, used)
						require.Equal(TypeKind_Workspace, got.Kind())
					} else {
						require.Equal(NullType, got)
					}
				}
			})
			t.Run("should be ok to range used workspaces by Types()", func(t *testing.T) {
				wsTypes := slices.Collect(ws.Types)
				requireSortedTypes(t, wsTypes)
				for u := 0; u < wsCount; u++ {
					usedWs := app.Type(wsName(u))
					if slices.Contains(test.use, u) {
						require.Contains(wsTypes, usedWs, "(%v).Types() should contain %v", ws, usedWs)
					} else {
						require.NotContains(wsTypes, usedWs, "(%v).Types() should not contain %v", ws, usedWs)
					}
					obj := Object(app.Type, objName(u))
					if u == idx {
						require.Contains(wsTypes, obj, "(%v).Types() should contain %v", ws, obj)
					} else {
						require.NotContains(wsTypes, obj, "(%v).Types() should not contain %v", ws, obj)
					}
				}
			})
			t.Run("should be breakable types enumeration", func(t *testing.T) {
				ws := app.Workspace(wsName(0))

				breakAt := func(n QName) {
					var stop IType
					for t := range ws.Types {
						if t.QName() == n {
							stop = t
							break
						}
					}
					require.NotNil(stop)
				}

				t.Run("from ancestor", func(t *testing.T) { breakAt(SysData_int32) })
				t.Run("from local", func(t *testing.T) { breakAt(objName(0)) })
				t.Run("from used", func(t *testing.T) { breakAt(wsName(1)) })
			})
		}
	})

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if used workspace is not found", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")

			ws := adb.AddWorkspace(wsName(0))
			unknown := NewQName("test", "unknown")
			require.Panics(func() { ws.UseWorkspace(unknown) },
				require.Is(ErrNotFoundError), require.Has(unknown))
			require.Panics(func() { ws.UseWorkspace(SysData_QName) },
				require.Is(ErrNotFoundError), require.Has(SysData_QName))
		})
	})
}
