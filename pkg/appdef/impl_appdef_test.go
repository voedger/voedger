/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"iter"
	"testing"

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

	t.Run("must ok to read sys package", func(t *testing.T) {
		require.Equal([]string{SysPackage}, app.PackageLocalNames())
		require.Equal(SysPackagePath, app.PackageFullPath(SysPackage))
	})

	t.Run("must ok to read sys types", func(t *testing.T) {
		require.Equal(NullType, app.TypeByName(NullQName))
		require.Equal(AnyType, app.TypeByName(QNameANY))
	})

	t.Run("must ok to read sys data types", func(t *testing.T) {
		require.Equal(SysData_RecordID, app.Data(SysData_RecordID).QName())
		require.Equal(SysData_String, app.Data(SysData_String).QName())
		require.Equal(SysData_bytes, app.Data(SysData_bytes).QName())
	})
}

func Test_NullAppDef(t *testing.T) {
	require := require.New(t)

	app := NullAppDef
	require.NotNil(app)
	require.Equal(NullType, app.TypeByName(NullQName))

	t.Run("should be ok to get system data types", func(t *testing.T) {
		for k := DataKind_null + 1; k < DataKind_FakeLast; k++ {
			d := app.SysData(k)
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

type tstEnum[T any] struct {
	enum iter.Seq[T]
}

func (e tstEnum[T]) breakable(t *testing.T, name string) {
	cnt := 0
	for range e.enum {
		cnt++
		break
	}
	if cnt != 1 {
		t.Errorf("range by %s should be breakable", name)
	}
}

func Test_AppDef_EnumerationBreakable(t *testing.T) {
	require := require.New(t)

	adb := New()
	adb.AddGDoc(NewQName("test", "doc"))
	adb.AddGRecord(NewQName("test", "rec"))

	app := adb.MustBuild()
	require.NotNil(app)

	t.Run("ranges enumerations should be breakable", func(t *testing.T) {
		tstEnum[IType]{app.Types}.breakable(t, "Types")
		tstEnum[IStructure]{app.Structures}.breakable(t, "Structures")
		tstEnum[IRecord]{app.Records}.breakable(t, "Records")
		tstEnum[IGDoc]{app.GDocs}.breakable(t, "GDocs")
		tstEnum[IGRecord]{app.GRecords}.breakable(t, "GRecords")
	})
}

func Test_appDefBuilder_MustBuild(t *testing.T) {
	require := require.New(t)

	require.NotNil(New().MustBuild(), "Should be ok if no errors in builder")

	t.Run("should panic if errors in builder", func(t *testing.T) {
		adb := New()
		adb.AddView(NewQName("test", "emptyView"))

		require.Panics(func() { _ = adb.MustBuild() },
			require.Is(ErrMissedError),
			require.Has("emptyView"),
		)
	})
}
