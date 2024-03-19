/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
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
		require.Empty(app.Comment())

		require.Zero(
			func() int {
				cnt := 0
				app.Types(func(t IType) {
					if !t.IsSystem() {
						cnt++
					}
				})
				return cnt
			}(),
			"must be no user types")

		require.Zero(
			func() int {
				cnt := 0
				app.Records(func(r IRecord) {
					if !r.IsSystem() {
						cnt++
					}
				})
				return cnt
			}(),
			"must be no user records")

		require.Zero(
			func() int {
				cnt := 0
				app.Structures(func(s IStructure) {
					if !s.IsSystem() {
						cnt++
					}
				})
				return cnt
			}(),
			"must be no user structures")

		require.Zero(
			func() int {
				cnt := 0
				app.Projectors(func(p IProjector) {
					if !p.IsSystem() {
						cnt++
					}
				})
				return cnt
			}(),
			"must be no user projectors")

		require.Zero(
			func() int {
				cnt := 0
				app.Extensions(func(e IExtension) {
					if !e.IsSystem() {
						cnt++
					}
				})
				return cnt
			}(),
			"must be no user extensions")
	})
}
