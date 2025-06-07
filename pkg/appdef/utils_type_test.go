/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"
	"iter"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func testFind[T appdef.IType](t *testing.T, name string, find func(f appdef.FindType, name appdef.QName) T, app appdef.IAppDef) {
	require := require.New(t)

	for i := 1; i <= 2; i++ {
		n := appdef.NewQName("test", fmt.Sprintf("%s%d", name, i))
		v := find(app.Type, n)
		require.NotNil(v, "should be found %s «%s», but not", name, n)
		require.Equal(n, v.QName())
	}

	require.Nil(find(app.Type, appdef.NewQName("test", "unknown")))
}

func testIterator[T appdef.IType](t *testing.T, name string, iter iter.Seq[T]) {
	require := require.New(t)

	t.Run(fmt.Sprintf("should be ok iterate %s", name), func(t *testing.T) {
		cnt := 0
		for v := range iter {
			if v.IsSystem() {
				continue
			}
			require.Contains(fmt.Sprint(v), name)
			cnt++
		}
		require.Equal(2, cnt, "%ss(): got %d iterations, expected 2", name, cnt)
	})

	t.Run(fmt.Sprintf("should be breakable %s", name), func(t *testing.T) {
		cnt := 0
		for range iter {
			cnt++
			break
		}
		require.Equal(1, cnt, "%ss(): got %d iterations, expected 1", name, cnt)
	})
}

func Test_TypeIterators(t *testing.T) {
	require := require.New(t)
	adb := builder.New()

	wsName := appdef.NewQName("test", "workspace")
	wsb := adb.AddWorkspace(wsName)

	qns := func(s string) []appdef.QName {
		return []appdef.QName{
			appdef.NewQName("test", s+"1"),
			appdef.NewQName("test", s+"2"),
		}
	}

	tagName := qns("Tag")
	wsb.AddTag(tagName[0], "tag 1 feature")
	wsb.AddTag(tagName[1], "tag 2 feature")

	dataName := qns("Data")
	wsb.AddData(dataName[0], appdef.DataKind_int64, appdef.NullQName).
		SetTag(tagName[0])
	wsb.AddData(dataName[1], appdef.DataKind_string, appdef.NullQName)

	gDocName := qns("GDoc")
	wsb.AddGDoc(gDocName[0])
	wsb.AddGDoc(gDocName[1])
	gRecName := qns("GRecord")
	wsb.AddGRecord(gRecName[0])
	wsb.AddGRecord(gRecName[1])

	cDocName := qns("CDoc")
	wsb.AddCDoc(cDocName[0]).SetSingleton()
	wsb.AddCDoc(cDocName[1]).SetTag(tagName[1])
	cRecName := qns("CRecord")
	wsb.AddCRecord(cRecName[0])
	wsb.AddCRecord(cRecName[1])

	wDocName := qns("WDoc")
	wsb.AddWDoc(wDocName[0]).SetSingleton()
	wsb.AddWDoc(wDocName[1]).SetTag(tagName[1])
	wRecName := qns("WRecord")
	wsb.AddWRecord(wRecName[0])
	wsb.AddWRecord(wRecName[1])

	oDocName := qns("ODoc")
	wsb.AddODoc(oDocName[0])
	wsb.AddODoc(oDocName[1])
	oRecName := qns("ORecord")
	wsb.AddORecord(oRecName[0])
	wsb.AddORecord(oRecName[1])

	objName := qns("Object")
	wsb.AddObject(objName[0])
	wsb.AddObject(objName[1])

	viewName := qns("View")
	for _, n := range viewName {
		v := wsb.AddView(n)
		v.Key().PartKey().AddField("pkf", appdef.DataKind_int64)
		v.Key().ClustCols().AddField("ccf", appdef.DataKind_string)
		v.Value().AddField("vf", appdef.DataKind_bytes, false)
	}

	cmdName := qns("Command")
	wsb.AddCommand(cmdName[0])
	wsb.AddCommand(cmdName[1])

	qryName := qns("Query")
	wsb.AddQuery(qryName[0])
	wsb.AddQuery(qryName[1])

	prjName := qns("Projector")
	prj1 := wsb.AddProjector(prjName[0])
	prj1.Events().Add(
		[]appdef.OperationKind{appdef.OperationKind_Execute},
		filter.QNames(cmdName[0]))
	prj2 := wsb.AddProjector(prjName[1])
	prj2.Events().Add(
		[]appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Update},
		filter.Tags(tagName[1]))

	jobName := qns("Job")
	wsb.AddJob(jobName[0]).SetCronSchedule("@every 3s")
	wsb.AddJob(jobName[1]).SetCronSchedule("@every 10s")

	roleName := qns("Role")
	_ = wsb.AddRole(roleName[0])
	wsb.GrantAll(filter.QNames(cmdName...), roleName[0])
	wsb.Revoke([]appdef.OperationKind{appdef.OperationKind_Execute}, filter.QNames(cmdName[0]), nil, roleName[0])

	_ = wsb.AddRole(roleName[1])
	wsb.GrantAll(filter.QNames(cDocName...), roleName[1])
	wsb.Revoke([]appdef.OperationKind{appdef.OperationKind_Insert}, filter.QNames(cDocName[1]), nil, roleName[1])

	rateName := qns("Rate")
	wsb.AddRate(rateName[0], 10, time.Second, []appdef.RateScope{appdef.RateScope_AppPartition})
	wsb.AddRate(rateName[1], 5, time.Second, []appdef.RateScope{appdef.RateScope_AppPartition})
	limitName := qns("Limit")
	wsb.AddLimit(limitName[0], []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_ALL, filter.AllFunctions(), rateName[0])
	wsb.AddLimit(limitName[1], []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_EACH, filter.AllFunctions(), rateName[1])

	app, err := adb.Build()
	require.NoError(err)

	t.Run("should be ok to find and iterate types", func(t *testing.T) {
		// ws := app.Workspace(wsName)

		// unknown := qns("unknown")

		testFind(t, "Tag", appdef.Tag, app)
		testIterator(t, "Tag", appdef.Tags(app.Types()))
		testFind(t, "Data", appdef.Data, app)
		testIterator(t, "data", appdef.DataTypes(app.Types()))
		testFind(t, "GDoc", appdef.GDoc, app)
		testIterator(t, "GDoc", appdef.GDocs(app.Types()))
		testFind(t, "GRecord", appdef.GRecord, app)
		testIterator(t, "GRecord", appdef.GRecords(app.Types()))
		testFind(t, "CDoc", appdef.CDoc, app)
		testIterator(t, "CDoc", appdef.CDocs(app.Types()))
		testFind(t, "CRecord", appdef.CRecord, app)
		testIterator(t, "CRecord", appdef.CRecords(app.Types()))
		testFind(t, "WDoc", appdef.WDoc, app)
		testIterator(t, "WDoc", appdef.WDocs(app.Types()))
		testFind(t, "WRecord", appdef.WRecord, app)
		testIterator(t, "WRecord", appdef.WRecords(app.Types()))
		testFind(t, "ODoc", appdef.ODoc, app)
		testIterator(t, "ODoc", appdef.ODocs(app.Types()))
		testFind(t, "ORecord", appdef.ORecord, app)
		testIterator(t, "ORecord", appdef.ORecords(app.Types()))
		testFind(t, "Object", appdef.Object, app)
		testIterator(t, "Object", appdef.Objects(app.Types()))
		testFind(t, "View", appdef.View, app)
		testIterator(t, "View", appdef.Views(app.Types()))
		testFind(t, "Command", appdef.Command, app)
		testIterator(t, "Command", appdef.Commands(app.Types()))
		testFind(t, "Query", appdef.Query, app)
		testIterator(t, "Query", appdef.Queries(app.Types()))
		testFind(t, "Projector", appdef.Projector, app)
		testIterator(t, "Projector", appdef.Projectors(app.Types()))
		testFind(t, "Job", appdef.Job, app)
		testIterator(t, "Job", appdef.Jobs(app.Types()))
		testFind(t, "Role", appdef.Role, app)
		testIterator(t, "Role", appdef.Roles(app.Types()))
		testFind(t, "Rate", appdef.Rate, app)
		testIterator(t, "Rate", appdef.Rates(app.Types()))
		testFind(t, "Limit", appdef.Limit, app)
		testIterator(t, "Limit", appdef.Limits(app.Types()))

		qNames := func(names ...[]appdef.QName) appdef.QNames {
			nn := appdef.QNames{}
			for _, n := range names {
				nn.Add(n...)
			}
			return nn
		}

		t.Run("should be ok to find and iterate extensions", func(t *testing.T) {
			want := qNames(cmdName, qryName, prjName, jobName)
			for _, n := range want {
				require.NotNil(appdef.Extension(app.Type, n))
			}
			got := appdef.QNames{}
			for e := range appdef.Extensions(app.Types()) {
				if !e.IsSystem() {
					got.Add(e.QName())
				}
			}
			require.Equal(want, got)
		})

		t.Run("should be ok to find and iterate extensions", func(t *testing.T) {
			want := qNames(cmdName, qryName)
			for _, n := range want {
				require.NotNil(appdef.Function(app.Type, n))
			}
			got := appdef.QNames{}
			for f := range appdef.Functions(app.Types()) {
				if !f.IsSystem() {
					got.Add(f.QName())
				}
			}
			require.Equal(want, got)
		})

		t.Run("should be ok to find and iterate records", func(t *testing.T) {
			want := qNames(gDocName, gRecName, cDocName, cRecName, wDocName, wRecName, oDocName, oRecName)
			for _, n := range want {
				require.NotNil(appdef.Record(app.Type, n))
			}
			got := appdef.QNames{}
			for f := range appdef.Records(app.Types()) {
				if !f.IsSystem() {
					got.Add(f.QName())
				}
			}
			require.Equal(want, got)
		})

		t.Run("should be ok to find and iterate singletons", func(t *testing.T) {
			want := appdef.QNamesFrom(cDocName[0], wDocName[0])
			for _, n := range want {
				require.NotNil(appdef.Singleton(app.Type, n))
			}

			require.Nil(appdef.Singleton(app.Type, wDocName[1]))

			got := appdef.QNames{}
			for f := range appdef.Singletons(app.Types()) {
				if !f.IsSystem() {
					got.Add(f.QName())
				}
			}
			require.Equal(want, got)

			t.Run("should be ok to break iterate singletons", func(t *testing.T) {
				cnt := 0
				for range appdef.Singletons(app.Types()) {
					cnt++
					break
				}
				require.Equal(1, cnt)
			})
		})

		t.Run("should be ok to find and iterate structures", func(t *testing.T) {
			want := qNames(gDocName, gRecName, cDocName, cRecName, wDocName, wRecName, oDocName, oRecName, objName)
			for _, n := range want {
				require.NotNil(appdef.Structure(app.Type, n))
			}
			got := appdef.QNames{}
			for f := range appdef.Structures(app.Types()) {
				if !f.IsSystem() {
					got.Add(f.QName())
				}
			}
			require.Equal(want, got)
		})
	})
}

func Test_AnyType(t *testing.T) {
	require := require.New(t)

	require.Empty(appdef.AnyType.Comment())
	require.Empty(appdef.AnyType.CommentLines())

	require.Nil(appdef.AnyType.App())
	require.Nil(appdef.AnyType.Workspace())
	require.Equal(appdef.QNameANY, appdef.AnyType.QName())
	require.Equal(appdef.TypeKind_Any, appdef.AnyType.Kind())
	require.True(appdef.AnyType.IsSystem())

	require.Contains(fmt.Sprint(appdef.AnyType), "ANY type")
}

func Test_TypeKind_Records(t *testing.T) {
	require := require.New(t)

	require.True(appdef.TypeKind_Records.ContainsAll(appdef.TypeKind_Docs.AsArray()...), "should contain all docs")

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"GDoc", appdef.TypeKind_CDoc, true},
		{"GRecord", appdef.TypeKind_CRecord, true},
		{"CDoc", appdef.TypeKind_CDoc, true},
		{"CRecord", appdef.TypeKind_CRecord, true},
		{"ODoc", appdef.TypeKind_ODoc, true},
		{"ORecord", appdef.TypeKind_ORecord, true},
		{"WDoc", appdef.TypeKind_WDoc, true},
		{"WRecord", appdef.TypeKind_WRecord, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"count", appdef.TypeKind_count, false},
		{"view", appdef.TypeKind_ViewRecord, false},
		{"command", appdef.TypeKind_Command, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Records.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Records.Set(appdef.TypeKind_ViewRecord)
	}, "should be read-only")
}

func Test_TypeKind_Docs(t *testing.T) {
	require := require.New(t)

	require.True(appdef.TypeKind_Docs.ContainsAll(appdef.TypeKind_GDoc, appdef.TypeKind_CDoc, appdef.TypeKind_ODoc, appdef.TypeKind_WDoc), "should contain all docs")

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"GDoc", appdef.TypeKind_GDoc, true},
		{"CDoc", appdef.TypeKind_CDoc, true},
		{"ODoc", appdef.TypeKind_ODoc, true},
		{"WDoc", appdef.TypeKind_WDoc, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"count", appdef.TypeKind_count, false},
		{"view", appdef.TypeKind_ViewRecord, false},
		{"command", appdef.TypeKind_Command, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Docs.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Docs.Set(appdef.TypeKind_ViewRecord)
	}, "should be read-only")
}

func Test_TypeKind_Structures(t *testing.T) {
	require := require.New(t)

	require.True(appdef.TypeKind_Structures.ContainsAll(appdef.TypeKind_Records.AsArray()...), "should contain all records")

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"Object", appdef.TypeKind_Object, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"count", appdef.TypeKind_count, false},
		{"view", appdef.TypeKind_ViewRecord, false},
		{"command", appdef.TypeKind_Command, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Structures.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Structures.Set(appdef.TypeKind_ViewRecord)
	}, "should be read-only")
}

func Test_TypeKind_Singletons(t *testing.T) {
	require := require.New(t)

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"CDoc", appdef.TypeKind_CDoc, true},
		{"WDoc", appdef.TypeKind_WDoc, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"ODoc", appdef.TypeKind_ODoc, false},
		{"view", appdef.TypeKind_ViewRecord, false},
		{"command", appdef.TypeKind_Command, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Singletons.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Singletons.Clear(appdef.TypeKind_CDoc)
	}, "should be read-only")
}

func Test_TypeKind_Functions(t *testing.T) {
	require := require.New(t)

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"Query", appdef.TypeKind_Query, true},
		{"Command", appdef.TypeKind_Command, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"count", appdef.TypeKind_count, false},
		{"CDoc", appdef.TypeKind_CDoc, false},
		{"Projector", appdef.TypeKind_Projector, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Functions.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Functions.Set(appdef.TypeKind_Job)
	}, "should be read-only")
}

func Test_TypeKind_Extensions(t *testing.T) {
	require := require.New(t)

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"Query", appdef.TypeKind_Query, true},
		{"Command", appdef.TypeKind_Command, true},
		{"Projector", appdef.TypeKind_Projector, true},
		{"Job", appdef.TypeKind_Job, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"count", appdef.TypeKind_count, false},
		{"CDoc", appdef.TypeKind_CDoc, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Extensions.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Extensions.ClearAll()
	}, "should be read-only")
}

func Test_TypeKind_Limitables(t *testing.T) {
	require := require.New(t)

	var tests = []struct {
		name string
		k    appdef.TypeKind
		want bool
	}{
		{"Query", appdef.TypeKind_Query, true},
		{"Command", appdef.TypeKind_Command, true},
		{"CDoc", appdef.TypeKind_CDoc, true},
		{"ORecord", appdef.TypeKind_ORecord, true},

		{"Any", appdef.TypeKind_Any, false},
		{"null", appdef.TypeKind_null, false},
		{"count", appdef.TypeKind_count, false},
		{"Job", appdef.TypeKind_Job, false},
		{"Object", appdef.TypeKind_Object, false},
		{"Role", appdef.TypeKind_Role, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, appdef.TypeKind_Limitables.Contains(tt.k))
		})
	}

	require.Panics(func() {
		appdef.TypeKind_Limitables.ClearAll()
	}, "should be read-only")
}

func TestTypeKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    appdef.TypeKind
		want string
	}{
		{name: `0 —> "appdef.TypeKind_null"`,
			k:    appdef.TypeKind_null,
			want: `TypeKind_null`,
		},
		{name: `2 —> "TypeKind_Data"`,
			k:    appdef.TypeKind_Data,
			want: `TypeKind_Data`,
		},
		{name: `3 —> "appdef.TypeKind_GDoc"`,
			k:    appdef.TypeKind_GDoc,
			want: `TypeKind_GDoc`,
		},
		{name: `TypeKind_FakeLast —> <number>`,
			k:    appdef.TypeKind_count,
			want: utils.UintToString(appdef.TypeKind_count),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("appdef.TypeKind.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("appdef.TypeKind.MarshalText() = %s, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover appdef.TypeKind.String()", func(t *testing.T) {
		const tested = appdef.TypeKind_count + 1
		want := "TypeKind(" + utils.UintToString(tested) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(TypeKind_FakeLast + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestTypeKindTrimString(t *testing.T) {
	tests := []struct {
		name string
		k    appdef.TypeKind
		want string
	}{
		{name: "null", k: appdef.TypeKind_null, want: "null"},
		{name: "basic", k: appdef.TypeKind_CDoc, want: "CDoc"},
		{name: "out of range", k: appdef.TypeKind_count + 1, want: (appdef.TypeKind_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(appdef.TypeKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
