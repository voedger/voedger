/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package workspaces_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/appdef/internal/workspaces"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Workspaces(t *testing.T) {
	require := require.New(t)

	wsName, descName, objName := appdef.NewQName("test", "ws"), appdef.NewQName("test", "desc"), appdef.NewQName("test", "object")
	wsPName, descPName, objPName := appdef.NewQName("test", "parentWs"), appdef.NewQName("test", "parentDesc"), appdef.NewQName("test", "parentObject")

	var app appdef.IAppDef

	t.Run("should be ok to add workspace with used workspace", func(t *testing.T) {
		adb := builder.New()
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
		for _, ws := range app.Workspaces() {
			switch ws.QName() {
			case appdef.SysWorkspaceQName:
			case wsName:
			case wsPName:
			default:
				require.Fail("unexpected workspace", "unexpected workspace «%v»", ws.QName())
			}
		}
		require.Len(app.Workspaces(), 1+1+1) // system ws + test ws + parent ws
	})

	t.Run("should be ok to find workspace", func(t *testing.T) {
		typ := app.Type(wsName)
		require.Equal(appdef.TypeKind_Workspace, typ.Kind())

		ws := app.Workspace(wsName)
		require.Equal(appdef.TypeKind_Workspace, ws.Kind())
		require.Equal(typ.(appdef.IWorkspace), ws)

		require.Equal(descName, ws.Descriptor())

		t.Run("should be ok to find by Type()", func(t *testing.T) {
			require.Equal(app.Type(objName), ws.Type(objName), "should find local type")
			require.Equal(appdef.SysData(app.Type, appdef.DataKind_int32), appdef.SysData(ws.Type, appdef.DataKind_int32), "should find type from ancestor")
			require.Equal(appdef.NullType, ws.Type(appdef.NewQName("unknown", "object")))
		})

		t.Run("should be ok to enum Types()", func(t *testing.T) {
			types := ws.Types()
			require.Contains(types, app.Type(descName), "ws types should contain descriptor")
			require.Contains(types, app.Type(objName), "ws types should contain local types")
			require.Contains(types, app.Type(appdef.SysData_bool), "ws types should contain types from ancestor")
		})

		t.Run("should be ok to find by LocalType()", func(t *testing.T) {
			require.NotNil(ws.LocalType(objName))
			require.Equal(appdef.NullType, ws.LocalType(appdef.SysData_bool))
		})

		t.Run("should be ok to enum LocalTypes()", func(t *testing.T) {
			types := ws.LocalTypes()
			require.Equal([]appdef.IType{app.Type(descName), app.Type(objName)}, types)
		})
	})

	t.Run("should be ok to find parent workspace", func(t *testing.T) {
		pWs := app.Workspace(wsPName)
		require.Equal(appdef.TypeKind_Workspace, pWs.Kind())

		t.Run("should be ok to find by Type()", func(t *testing.T) {
			require.Equal(app.Type(wsName), pWs.Type(wsName), "should find used workspace in parent")
			require.Equal(appdef.NullType, pWs.Type(objName), "should not found object from used workspace in parent")
		})

		t.Run("should be ok to enum Types()", func(t *testing.T) {
			types := pWs.Types()
			require.Contains(types, app.Type(wsName), "ws types should contain used workspace")
			require.NotContains(types, app.Type(objName), "ws types should not contain type from used workspace", pWs, objName)
		})
	})

	require.Nil(app.Workspace(appdef.NewQName("unknown", "workspace")), "should be nil if unknown workspace")
}

func Test_WorkspacesPanics(t *testing.T) {
	require := require.New(t)

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if type from other package added to workspace", func(t *testing.T) {
			t.Skip("package check is violated by `parser` package")
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			adb.AddPackage("other", "test.com/other")
			ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
			require.Panics(func() { _ = ws.AddObject(appdef.NewQName("other", "doc")) },
				require.Is(appdef.ErrInvalidError), require.HasAll("other.doc", "same package", "test.workspace"))
		})

		t.Run("if unknown descriptor assigned to workspace", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
			require.Panics(func() { ws.SetDescriptor(appdef.NewQName("unknown", "type")) },
				require.Is(appdef.ErrNotFoundError), require.Has("unknown.type"))
		})
	})
}

func Test_AlterWorkspace(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "ws")
	objName := appdef.NewQName("test", "object")

	var adb appdef.IAppDefBuilder

	t.Run("should be ok to add workspace", func(t *testing.T) {
		adb = builder.New()
		adb.AddPackage("test", "test.com/test")

		_ = adb.AddWorkspace(wsName)
	})

	t.Run("should be ok to alter workspace", func(t *testing.T) {
		wsb := adb.AlterWorkspace(wsName)
		_ = wsb.AddObject(objName)
	})

	t.Run("should be panic to alter unknown workspace", func(t *testing.T) {
		require.Panics(func() { _ = adb.AlterWorkspace(appdef.NewQName("test", "unknown")) })
		require.Panics(func() { _ = adb.AlterWorkspace(objName) })
	})

	t.Run("should be ok to build altered workspace", func(t *testing.T) {
		app, err := adb.Build()
		require.NoError(err)

		ws := app.Workspace(wsName)
		require.NotNil(ws, "should be ok to find workspace in app")

		require.NotNil(appdef.Object(ws.Type, objName), "should be ok to find object in workspace")
	})
}

func Test_WorkspaceDescriptor(t *testing.T) {
	require := require.New(t)

	t.Run("should be ok to add workspace with descriptor", func(t *testing.T) {
		wsName, descName := appdef.NewQName("test", "ws"), appdef.NewQName("test", "desc")

		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)
		_ = ws.AddCDoc(descName)
		ws.SetDescriptor(descName)

		app, err := adb.Build()
		require.NoError(err)

		t.Run("should be ok to find workspace by descriptor", func(t *testing.T) {
			ws := app.WorkspaceByDescriptor(descName)
			require.NotNil(ws)
			require.Equal(appdef.TypeKind_Workspace, ws.Kind())

			t.Run("should be nil if can't find workspace by descriptor", func(t *testing.T) {
				ws := app.WorkspaceByDescriptor(appdef.NewQName("test", "unknown"))
				require.Nil(ws)
			})
		})
	})

	t.Run("should be ok to change ws descriptor", func(t *testing.T) {
		wsName, descName, desc1Name := appdef.NewQName("test", "ws"), appdef.NewQName("test", "desc"), appdef.NewQName("test", "desc1")

		adb := builder.New()
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
			require.Equal(appdef.TypeKind_Workspace, ws.Kind())
			require.Equal(wsName, ws.QName())

			require.Nil(app.WorkspaceByDescriptor(descName))
		})

		t.Run("should be ok to clear descriptor", func(t *testing.T) {
			ws.SetDescriptor(appdef.NullQName)

			app, err = adb.Build()
			require.NoError(err)

			require.Nil(app.WorkspaceByDescriptor(descName))
			require.Nil(app.WorkspaceByDescriptor(desc1Name))

			ws := app.Workspace(wsName)
			require.NotNil(ws)
			require.Equal(appdef.NullQName, ws.Descriptor())
		})
	})
}

func Test_WorkspaceAbstract(t *testing.T) {
	require := require.New(t)

	wsName, descName := appdef.NewQName("test", "ws"), appdef.NewQName("test", "desc")

	var app appdef.IAppDef

	t.Run("should be ok to add abstract workspace", func(t *testing.T) {
		adb := builder.New()
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

		desc := appdef.CDoc(app.Type, ws.Descriptor())
		require.True(desc.Abstract())
	})

	t.Run("should be error to set descriptor abstract after assign to workspace", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		desc := ws.AddCDoc(descName)
		ws.SetDescriptor(descName)

		desc.SetAbstract()

		_, err := adb.Build()
		require.ErrorIs(err, appdef.ErrIncompatibleError)
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

	wsName := func(idx int) appdef.QName { return appdef.NewQName("test", fmt.Sprintf("ws%d", idx)) }
	objName := func(idx int) appdef.QName { return appdef.NewQName("test", fmt.Sprintf("obj%d", idx)) }

	testADB := func() appdef.IAppDefBuilder {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		for idx := 0; idx < wsCount; idx++ {
			ws := adb.AddWorkspace(wsName(idx))
			anc := testAncestors[idx].anc
			if len(anc) > 0 {
				ancNames := make([]appdef.QName, len(anc))
				for i, a := range anc {
					ancNames[i] = wsName(a)
				}
				ws.SetAncestors(ancNames[0], ancNames[1:]...)
			}
			_ = ws.AddObject(objName(idx))
		}
		return adb
	}

	var app appdef.IAppDef

	t.Run("should be ok to add workspaces with inheritance", func(t *testing.T) {
		adb := testADB()
		a, err := adb.Build()
		require.NoError(err)
		app = a
	})

	t.Run("should be ok to read ancestors", func(t *testing.T) {
		for wsIdx, test := range testAncestors {
			ws := app.Workspace(wsName(wsIdx))

			want := make([]appdef.QName, len(test.anc))
			for i, a := range test.anc {
				want[i] = wsName(a)
			}
			if len(want) == 0 {
				want = []appdef.QName{appdef.SysWorkspaceQName}
			}
			i := 0
			for _, a := range ws.Ancestors() {
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
			require.True(ws.Inherits(appdef.SysWorkspaceQName), "any workspace should inherit from sys.Workspace")
		}
	})

	t.Run("should be correspondence between Ancestors() and Inherits()", func(t *testing.T) {
		for idx := 0; idx < wsCount; idx++ {
			ws := app.Workspace(wsName(idx))
			for _, a := range ws.Ancestors() {
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
				obj := appdef.Object(ws.Type, objName(o))
				got := obj != nil
				require.Equal(want, got, "unexpected %v.appdef.Object(%v) != nil result: want %v, got: %v", ws, objName(o), want, got)
				if got {
					require.True(ws.Inherits(obj.Workspace().QName()),
						"any type from workspace should be owned by this workspace or by its ancestors")
				}
			}
			require.NotNil(ws.Type(appdef.SysData_int32), "should be ok to find system type in workspace")
		}
	})

	t.Run("should be ok to iterate types", func(t *testing.T) {
		for wsIdx := 0; wsIdx < wsCount; wsIdx++ {
			ws := app.Workspace(wsName(wsIdx))
			types := ws.Types()
			for _, a := range ws.Ancestors() {
				for _, t := range a.Types() {
					require.Contains(types, t, "%v types should contains type %v from ancestor %v", ws, t, a)
				}
			}
		}
	})

	t.Run("Should be rebuild workspace if ancestors changed", func(t *testing.T) {
		adb := testADB()
		_ = adb.MustBuild()

		newObj := appdef.NewQName("test", "newObject")
		find := func() (obj appdef.IObject) {
			ws7 := adb.AlterWorkspace(wsName(7)).Workspace()
			for _, o := range ws7.Types() {
				if o.QName() == newObj {
					obj = o.(appdef.IObject)
					break
				}
			}
			return obj
		}

		require.Nil(find(), "should not found new object before add")

		ws3b := adb.AlterWorkspace(wsName(3))
		_ = ws3b.AddObject(newObj)

		require.NotNil(find(), "should found new object after add")
	})

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if ancestor workspace is not found", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")

			ws := adb.AddWorkspace(wsName(0))
			unknown := appdef.NewQName("test", "unknown")
			require.Panics(func() { ws.SetAncestors(unknown) },
				require.Is(appdef.ErrNotFoundError), require.Has(unknown))
			require.Panics(func() { ws.SetAncestors(appdef.SysData_QName) },
				require.Is(appdef.ErrNotFoundError), require.Has(appdef.SysData_QName))
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
							require.Is(appdef.ErrUnsupportedError), require.Has("Circular inheritance"), require.Has(anc), require.Has(wsName(wsIdx)))
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

	wsName := func(idx int) appdef.QName { return appdef.NewQName("test", fmt.Sprintf("ws%d", idx)) }
	objName := func(idx int) appdef.QName { return appdef.NewQName("test", fmt.Sprintf("obj%d", idx)) }

	testADB := func() appdef.IAppDefBuilder {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		for i := 0; i < wsCount; i++ {
			ws := adb.AddWorkspace(wsName(i))
			_ = ws.AddObject(objName(i))
		}
		for idx, test := range testUses {
			ws := adb.AlterWorkspace(wsName(idx))
			if len(test.use) > 0 {
				useNames := make([]appdef.QName, len(test.use))
				for i, a := range test.use {
					useNames[i] = wsName(a)
				}
				ws.UseWorkspace(useNames[0], useNames[1:]...)
			}
		}
		return adb
	}

	var app appdef.IAppDef

	t.Run("should be ok to add workspaces with usages", func(t *testing.T) {
		adb := testADB()
		a, err := adb.Build()
		require.NoError(err)
		app = a
	})

	t.Run("should be ok to to read workspace usage", func(t *testing.T) {
		for idx, test := range testUses {
			ws := app.Workspace(wsName(idx))
			want := make([]appdef.QName, len(test.use))
			for i, u := range test.use {
				want[i] = wsName(u)
			}
			i := 0
			for _, u := range ws.UsedWorkspaces() {
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
						require.Equal(appdef.TypeKind_Workspace, got.Kind())
					} else {
						require.Equal(appdef.NullType, got)
					}
				}
			})
			t.Run("should be ok to range used workspaces by Types()", func(t *testing.T) {
				wsTypes := ws.Types()
				for u := 0; u < wsCount; u++ {
					usedWs := app.Type(wsName(u))
					if slices.Contains(test.use, u) {
						require.Contains(wsTypes, usedWs, "(%v).Types() should contain %v", ws, usedWs)
					} else {
						require.NotContains(wsTypes, usedWs, "(%v).Types() should not contain %v", ws, usedWs)
					}
					obj := appdef.Object(app.Type, objName(u))
					if u == idx {
						require.Contains(wsTypes, obj, "(%v).Types() should contain %v", ws, obj)
					} else {
						require.NotContains(wsTypes, obj, "(%v).Types() should not contain %v", ws, obj)
					}
				}
			})
		}
	})

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if used workspace is not found", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")

			ws := adb.AddWorkspace(wsName(0))
			unknown := appdef.NewQName("test", "unknown")
			require.Panics(func() { ws.UseWorkspace(unknown) },
				require.Is(appdef.ErrNotFoundError), require.Has(unknown))
			require.Panics(func() { ws.UseWorkspace(appdef.SysData_QName) },
				require.Is(appdef.ErrNotFoundError), require.Has(appdef.SysData_QName))
		})
	})
}

func Test_WorkspaceAddTypes(t *testing.T) {
	require := require.New(t)

	adb := builder.New()

	wsName := appdef.NewQName("test", "workspace")
	wsb := adb.AddWorkspace(wsName)

	tagName := appdef.NewQName("test", "Tag")
	wsb.AddTag(tagName, "tag 1 feature", "tag 1 comment")

	dataName := appdef.NewQName("test", "Data")
	wsb.AddData(dataName, appdef.DataKind_int64, appdef.NullQName).
		SetTag(tagName)

	gDocName := appdef.NewQName("test", "GDoc")
	wsb.AddGDoc(gDocName)
	gRecName := appdef.NewQName("test", "GRecord")
	wsb.AddGRecord(gRecName)

	cDocName := appdef.NewQName("test", "CDoc")
	wsb.AddCDoc(cDocName).SetSingleton()
	cRecName := appdef.NewQName("test", "CRecord")
	wsb.AddCRecord(cRecName)

	wDocName := appdef.NewQName("test", "WDoc")
	wsb.AddWDoc(wDocName)
	wRecName := appdef.NewQName("test", "WRecord")
	wsb.AddWRecord(wRecName)

	oDocName := appdef.NewQName("test", "ODoc")
	wsb.AddODoc(oDocName)
	oRecName := appdef.NewQName("test", "ORecord")
	wsb.AddORecord(oRecName)

	objName := appdef.NewQName("test", "Object")
	wsb.AddObject(objName)

	viewName := appdef.NewQName("test", "View")
	v := wsb.AddView(viewName)
	v.Key().PartKey().AddField("pkf", appdef.DataKind_int64)
	v.Key().ClustCols().AddField("ccf", appdef.DataKind_string)
	v.Value().AddField("vf", appdef.DataKind_bytes, false)

	cmdName := appdef.NewQName("test", "Command")
	wsb.AddCommand(cmdName)

	qryName := appdef.NewQName("test", "Query")
	wsb.AddQuery(qryName)

	prjName := appdef.NewQName("test", "Projector")
	prj1 := wsb.AddProjector(prjName)
	prj1.Events().Add(
		[]appdef.OperationKind{appdef.OperationKind_Execute},
		filter.QNames(cmdName))

	jobName := appdef.NewQName("test", "Job")
	wsb.AddJob(jobName).SetCronSchedule("@every 3s")

	roleName := appdef.NewQName("test", "Role")
	wsb.AddRole(roleName)
	wsb.GrantAll(filter.QNames(cmdName), roleName)
	wsb.RevokeAll(filter.QNames(cmdName), roleName)
	wsb.Grant([]appdef.OperationKind{appdef.OperationKind_Execute},
		filter.QNames(cmdName, cmdName), nil, roleName)
	wsb.Revoke([]appdef.OperationKind{appdef.OperationKind_Execute},
		filter.QNames(cmdName), nil, roleName)

	rateName := appdef.NewQName("test", "Rate")
	wsb.AddRate(rateName, 1, time.Second, []appdef.RateScope{appdef.RateScope_AppPartition})
	limitName := appdef.NewQName("test", "Limit")
	wsb.AddLimit(limitName, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_ALL, filter.QNames(cmdName), rateName)

	app := adb.MustBuild()
	require.NotNil(app)

	t.Run("should be ok to read added types", func(t *testing.T) {
		ws := app.Workspace(wsName)

		unknown := appdef.NewQName("test", "unknown")

		require.Equal(tagName, appdef.Tag(ws.Type, tagName).QName())
		require.Nil(appdef.Tag(ws.Type, unknown))

		require.Equal(dataName, appdef.Data(ws.Type, dataName).QName())
		require.Nil(appdef.Data(ws.Type, unknown))

		require.Equal(gDocName, appdef.GDoc(ws.Type, gDocName).QName())
		require.Nil(appdef.GDoc(ws.Type, unknown))
		require.Equal(gRecName, appdef.GRecord(ws.Type, gRecName).QName())
		require.Nil(appdef.GRecord(ws.Type, unknown))

		require.Equal(cDocName, appdef.CDoc(ws.Type, cDocName).QName())
		require.Nil(appdef.CDoc(ws.Type, unknown))
		require.Equal(cRecName, appdef.CRecord(ws.Type, cRecName).QName())
		require.Nil(appdef.CRecord(ws.Type, unknown))

		require.Equal(wDocName, appdef.WDoc(ws.Type, wDocName).QName())
		require.Nil(appdef.WDoc(ws.Type, unknown))
		require.Equal(wRecName, appdef.WRecord(ws.Type, wRecName).QName())
		require.Nil(appdef.WRecord(ws.Type, unknown))

		require.Equal(oDocName, appdef.ODoc(ws.Type, oDocName).QName())
		require.Nil(appdef.ODoc(ws.Type, unknown))
		require.Equal(oRecName, appdef.ORecord(ws.Type, oRecName).QName())
		require.Nil(appdef.ORecord(ws.Type, unknown))

		require.Equal(objName, appdef.Object(ws.Type, objName).QName())
		require.Nil(appdef.Object(ws.Type, unknown))

		require.Equal(viewName, appdef.View(ws.Type, viewName).QName())
		require.Nil(appdef.View(ws.Type, unknown))

		require.Equal(cmdName, appdef.Command(ws.Type, cmdName).QName())
		require.Nil(appdef.Command(ws.Type, unknown))

		require.Equal(qryName, appdef.Query(ws.Type, qryName).QName())
		require.Nil(appdef.Query(ws.Type, unknown))

		require.Equal(prjName, appdef.Projector(ws.Type, prjName).QName())
		require.Nil(appdef.Projector(ws.Type, unknown))

		require.Equal(jobName, appdef.Job(ws.Type, jobName).QName())
		require.Nil(appdef.Job(ws.Type, unknown))

		require.Equal(roleName, appdef.Role(ws.Type, roleName).QName())
		require.Nil(appdef.Role(ws.Type, unknown))

		require.Equal(rateName, appdef.Rate(ws.Type, rateName).QName())
		require.Nil(appdef.Rate(ws.Type, unknown))
		require.Equal(limitName, appdef.Limit(ws.Type, limitName).QName())
		require.Nil(appdef.Limit(ws.Type, unknown))
	})
}

func Test_WorkspaceGrantRevokePanics(t *testing.T) {
	require := require.New(t)

	adb := builder.New()

	wsName := appdef.NewQName("test", "workspace")
	wsb := adb.AddWorkspace(wsName)

	unknown := appdef.NewQName("test", "unknown")

	require.Panics(func() {
		wsb.Grant([]appdef.OperationKind{appdef.OperationKind_Execute}, filter.AllFunctions(), nil, unknown)
	}, require.Is(appdef.ErrNotFoundError), require.Has(unknown))
	require.Panics(func() { wsb.GrantAll(filter.AllFunctions(), unknown) },
		require.Is(appdef.ErrNotFoundError), require.Has(unknown))

	require.Panics(func() {
		wsb.Revoke([]appdef.OperationKind{appdef.OperationKind_Execute}, filter.AllFunctions(), nil, unknown)
	}, require.Is(appdef.ErrNotFoundError), require.Has(unknown))
	require.Panics(func() { wsb.RevokeAll(filter.AllFunctions(), unknown) },
		require.Is(appdef.ErrNotFoundError), require.Has(unknown))
}

func Test_AddWorkspace(t *testing.T) {
	require := require.New(t)

	adb := builder.New()

	ww := workspaces.MakeWithWorkspaces()

	wsName := appdef.NewQName("test", "workspace")
	ws := workspaces.AddWorkspace(adb.AppDef(), &ww, wsName)

	require.NotNil(ws)

	var _ appdef.IWorkspace = ws
}
