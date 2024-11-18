/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"iter"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestNew(t *testing.T) {
	require := require.New(t)

	adb := New()
	require.NotNil(adb)

	require.NotNil(adb.AppDef(), "should be ok get AppDef before build")

	app, err := adb.Build()
	require.NoError(err)
	require.NotNil(app)

	require.Equal(adb.AppDef(), app, "should be ok get AppDef after build")

	t.Run("should be ok to read sys package", func(t *testing.T) {
		require.Equal([]string{SysPackage}, app.PackageLocalNames())
		require.Equal(SysPackagePath, app.PackageFullPath(SysPackage))
	})

	t.Run("should be ok to read sys types", func(t *testing.T) {
		require.Equal(NullType, app.Type(NullQName))
		require.Equal(AnyType, app.Type(QNameANY))
	})

	t.Run("should be ok to read sys data types", func(t *testing.T) {
		require.Equal(SysData_RecordID, Data(app.Type, SysData_RecordID).QName())
		require.Equal(SysData_String, Data(app.Type, SysData_String).QName())
		require.Equal(SysData_bytes, Data(app.Type, SysData_bytes).QName())
	})
}

func Test_NullAppDef(t *testing.T) {
	require := require.New(t)

	app := NullAppDef
	require.NotNil(app)
	require.Equal(NullType, app.Type(NullQName))

	t.Run("should be ok to get system data types", func(t *testing.T) {
		for k := DataKind_null + 1; k < DataKind_FakeLast; k++ {
			d := SysData(app.Type, k)
			require.NotNil(d)
			require.True(d.IsSystem())
			require.Equal(SysDataName(k), d.QName())
			require.Equal(k, d.DataKind())
		}
	})

	t.Run("should be return sys package only", func(t *testing.T) {
		require.Equal([]string{SysPackage}, app.PackageLocalNames())
		require.Equal(SysPackagePath, app.PackageFullPath(SysPackage))
	})

	t.Run("should be null return other members", func(t *testing.T) {
		for typ := range app.Types {
			if !typ.IsSystem() {
				t.Errorf("unexpected user type %v", typ)
			}
		}
	})
}

func testBreakable[T any](t *testing.T, name string, seq ...iter.Seq[T]) {
	for i, s := range seq {
		t.Run(fmt.Sprintf("%s[%d]", name, i), func(t *testing.T) {
			cnt := 0
			for range s {
				cnt++
				break
			}
			if cnt != 1 {
				t.Errorf("got %d iterations, expected 1", i)
			}
		})
	}
}

func Test_EnumsBreakable(t *testing.T) {
	require := require.New(t)

	adb := New()

	wsName := NewQName("test", "workspace")
	wsb := adb.AddWorkspace(wsName)

	wsb.AddData(NewQName("test", "Data1"), DataKind_int64, NullQName)
	wsb.AddData(NewQName("test", "Data2"), DataKind_string, NullQName)

	wsb.AddGDoc(NewQName("test", "GDoc1"))
	wsb.AddGDoc(NewQName("test", "GDoc2"))
	wsb.AddGRecord(NewQName("test", "GRecord1"))
	wsb.AddGRecord(NewQName("test", "GRecord2"))

	wsb.AddCDoc(NewQName("test", "CDoc1")).
		SetSingleton()
	wsb.AddCDoc(NewQName("test", "CDoc2")).
		SetSingleton()
	wsb.AddCRecord(NewQName("test", "CRecord1"))
	wsb.AddCRecord(NewQName("test", "CRecord2"))

	wsb.AddWDoc(NewQName("test", "WDoc1")).
		SetSingleton()
	wsb.AddWDoc(NewQName("test", "WDoc2")).
		SetSingleton()
	wsb.AddWRecord(NewQName("test", "WRecord1"))
	wsb.AddWRecord(NewQName("test", "WRecord2"))

	wsb.AddODoc(NewQName("test", "ODoc1"))
	wsb.AddODoc(NewQName("test", "ODoc2"))
	wsb.AddORecord(NewQName("test", "ORecord1"))
	wsb.AddORecord(NewQName("test", "ORecord2"))

	wsb.AddObject(NewQName("test", "Object1"))
	wsb.AddObject(NewQName("test", "Object2"))

	for i := 1; i <= 2; i++ {
		v := wsb.AddView(NewQName("test", fmt.Sprintf("View%d", i)))
		v.Key().PartKey().AddField("pkf", DataKind_int64)
		v.Key().ClustCols().AddField("ccf", DataKind_string)
		v.Value().AddField("vf", DataKind_bytes, false)
	}

	cmd1Name, cmd2Name := NewQName("test", "Command1"), NewQName("test", "Command2")
	wsb.AddCommand(cmd1Name)
	wsb.AddCommand(cmd2Name)

	wsb.AddQuery(NewQName("test", "Query1"))
	wsb.AddQuery(NewQName("test", "Query2"))

	wsb.AddProjector(NewQName("test", "Projector1")).
		Events().Add(cmd1Name)
	wsb.AddProjector(NewQName("test", "Projector2")).
		Events().Add(cmd2Name)

	job1name, job2name := NewQName("test", "Job1"), NewQName("test", "Job2")
	wsb.AddJob(job1name).SetCronSchedule("@every 3s").
		States().
		Add(NewQName("test", "State1"), cmd1Name, cmd2Name).
		Add(NewQName("test", "State2"))
	wsb.AddJob(job2name).SetCronSchedule("@every 1h")

	role1Name, role2Name := NewQName("test", "Role1"), NewQName("test", "Role2")
	wsb.AddRole(role1Name).
		GrantAll([]QName{cmd1Name, cmd2Name}).
		RevokeAll([]QName{cmd2Name})
	wsb.AddRole(role2Name).
		GrantAll([]QName{cmd1Name, cmd2Name}).
		RevokeAll([]QName{cmd1Name})

	rate1Name, rate2Name := NewQName("test", "Rate1"), NewQName("test", "Rate2")
	wsb.AddRate(rate1Name, 1, time.Second, []RateScope{RateScope_AppPartition})
	wsb.AddRate(rate2Name, 2, 2*time.Second, []RateScope{RateScope_IP})
	wsb.AddLimit(NewQName("test", "Limit1"), []QName{cmd1Name}, rate1Name)
	wsb.AddLimit(NewQName("test", "Limit2"), []QName{cmd2Name}, rate2Name)

	app := adb.MustBuild()
	require.NotNil(app)

	t.Run("should be breakable", func(t *testing.T) {
		ws := app.Workspace(wsName)

		testBreakable(t, "Workspaces", app.Workspaces)

		testBreakable(t, "Types", app.Types, ws.Types, ws.LocalTypes)

		testBreakable(t, "DataTypes", DataTypes(app.Types), DataTypes(ws.LocalTypes))

		testBreakable(t, "GDocs", GDocs(app.Types), GDocs(ws.LocalTypes))
		testBreakable(t, "GRecords", GRecords(app.Types), GRecords(ws.LocalTypes))

		testBreakable(t, "CDocs", CDocs(app.Types), CDocs(ws.LocalTypes))
		testBreakable(t, "CRecords", CRecords(app.Types), CRecords(ws.LocalTypes))

		testBreakable(t, "WDocs", WDocs(app.Types), WDocs(ws.LocalTypes))
		testBreakable(t, "WRecords", WRecords(app.Types), WRecords(ws.LocalTypes))

		testBreakable(t, "Singletons", Singletons(app.Types), Singletons(ws.LocalTypes))

		testBreakable(t, "ODocs", ODocs(app.Types), ODocs(ws.LocalTypes))
		testBreakable(t, "ORecords", ORecords(app.Types), ORecords(ws.LocalTypes))

		testBreakable(t, "Records", Records(app.Types), Records(ws.LocalTypes))

		testBreakable(t, "Objects", Objects(app.Types), Objects(ws.LocalTypes))

		testBreakable(t, "Structures", Structures(app.Types), Structures(ws.LocalTypes))

		testBreakable(t, "View", Views(app.Types), Views(ws.LocalTypes))

		testBreakable(t, "Commands", Commands(app.Types), Commands(ws.LocalTypes))
		testBreakable(t, "Queries", Queries(app.Types), Queries(ws.LocalTypes))
		testBreakable(t, "Functions", Functions(app.Types), Functions(ws.LocalTypes))

		testBreakable(t, "Projectors", Projectors(app.Types), Projectors(ws.LocalTypes))
		testBreakable(t, "Jobs", Jobs(app.Types), Jobs(ws.LocalTypes))
		testBreakable(t, "IStorages.Enum", Job(app.Type, job1name).States().Enum)

		testBreakable(t, "Extensions", Extensions(app.Types), Extensions(ws.LocalTypes))

		testBreakable(t, "Roles", Roles(app.Types), Roles(ws.LocalTypes))
		testBreakable(t, "ACL", ACL(app), ACL(ws), ACL(Role(app.Type, role1Name)))

		testBreakable(t, "Rates", Rates(app.Types), Rates(ws.LocalTypes))
		testBreakable(t, "Limits", Limits(app.Types), Limits(ws.LocalTypes))
	})
}

func Test_appDefBuilder_MustBuild(t *testing.T) {
	require := require.New(t)

	require.NotNil(New().MustBuild(), "Should be ok if no errors in builder")

	t.Run("should panic if errors in builder", func(t *testing.T) {
		adb := New()
		adb.AddWorkspace(NewQName("test", "workspace")).AddView(NewQName("test", "emptyView"))

		require.Panics(func() { _ = adb.MustBuild() },
			require.Is(ErrMissedError),
			require.Has("emptyView"),
		)
	})
}
