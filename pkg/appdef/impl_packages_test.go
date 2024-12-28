/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"slices"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_AddPackage(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	t.Run("should be ok to add package", func(t *testing.T) {
		adb := appdef.New()

		adb.AddPackage("test", "test.com/path")
		adb.AddPackage("example", "example.com/path")

		a, err := adb.Build()
		require.NoError(err)
		app = a
	})

	t.Run("should be ok to inspect packages", func(t *testing.T) {
		require.Equal("test", app.PackageLocalName("test.com/path"))
		require.Equal("test.com/path", app.PackageFullPath("test"))

		require.Equal("example", app.PackageLocalName("example.com/path"))
		require.Equal("example.com/path", app.PackageFullPath("example"))

		require.EqualValues([]string{"example", "sys", "test"}, slices.Collect(app.PackageLocalNames()))

		cnt := 0
		for localName, fullPath := range app.Packages() {
			switch cnt {
			case 0:
				require.Equal("example", localName)
				require.Equal("example.com/path", fullPath)
			case 1:
				require.Equal(appdef.SysPackage, localName)
				require.Equal(appdef.SysPackagePath, fullPath)
			case 2:
				require.Equal("test", localName)
				require.Equal("test.com/path", fullPath)
			default:
				require.Fail("unexpected package %v (%v)", localName, fullPath)
			}
			cnt++
		}
		require.Equal(3, cnt)
	})

	t.Run("should be reconvert full-local qualified names", func(t *testing.T) {
		require.Equal(appdef.NewQName(appdef.SysPackage, "name"), app.LocalQName(appdef.NewFullQName(appdef.SysPackagePath, "name")))
		require.Equal(appdef.NewFullQName(appdef.SysPackagePath, "name"), app.FullQName(appdef.NewQName(appdef.SysPackage, "name")))

		require.Equal(appdef.NewQName("test", "name"), app.LocalQName(appdef.NewFullQName("test.com/path", "name")))
		require.Equal(appdef.NewFullQName("test.com/path", "name"), app.FullQName(appdef.NewQName("test", "name")))

		require.Equal(appdef.NewQName("example", "name"), app.LocalQName(appdef.NewFullQName("example.com/path", "name")))
		require.Equal(appdef.NewFullQName("example.com/path", "name"), app.FullQName(appdef.NewQName("example", "name")))

		require.Equal(appdef.NullQName, app.LocalQName(appdef.NewFullQName("unknown.com/path", "name")))
		require.Equal(appdef.NullFullQName, app.FullQName(appdef.NewQName("unknown", "name")))
	})

	t.Run("should be empties if unknown packages", func(t *testing.T) {
		require.Equal("", app.PackageLocalName("unknown.com/path"))
		require.Equal("", app.PackageFullPath("unknown"))
	})

	t.Run("test panics", func(t *testing.T) {
		adb := appdef.New()

		require.Panics(func() { adb.AddPackage("", "test.com/path") },
			require.Is(appdef.ErrMissedError))
		require.Panics(func() { adb.AddPackage("naked 🔫", "test.com/path") },
			require.Is(appdef.ErrInvalidError), require.Has("naked 🔫"))
		require.Panics(func() { adb.AddPackage("test", "") },
			require.Is(appdef.ErrMissedError))

		require.Panics(
			func() {
				adb.AddPackage("test", "test1.com/path")
				adb.AddPackage("test", "test2.com/path")
			}, require.Is(appdef.ErrAlreadyExistsError), require.Has("test"))

		require.Panics(
			func() {
				adb.AddPackage("test1", "test.com/path")
				adb.AddPackage("test2", "test.com/path")
			}, require.Is(appdef.ErrAlreadyExistsError), require.Has("test.com/path"))

		require.Panics(
			func() {
				adb.AddPackage(appdef.SysPackage, "test.com/sys")
			}, require.Is(appdef.ErrAlreadyExistsError), require.Has(appdef.SysPackage))
	})
}
