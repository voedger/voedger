/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_AddPackage(t *testing.T) {
	require := require.New(t)

	var app IAppDef

	t.Run("should be ok to add package", func(t *testing.T) {
		adb := New()

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

		require.EqualValues([]string{"example", "sys", "test"}, app.PackageLocalNames())

		cnt := 0
		for localName, fullPath := range app.Packages {
			switch cnt {
			case 0:
				require.Equal("example", localName)
				require.Equal("example.com/path", fullPath)
			case 1:
				require.Equal(SysPackage, localName)
				require.Equal(SysPackagePath, fullPath)
			case 2:
				require.Equal("test", localName)
				require.Equal("test.com/path", fullPath)
			default:
				require.Fail("unexpected package %v (%v)", localName, fullPath)
			}
			cnt++
		}
		require.Equal(3, cnt)

		t.Run("range for packages should be breakable", func(t *testing.T) {
			cnt := 0
			for _, _ = range app.Packages {
				cnt++
				break
			}
			require.Equal(1, cnt)
		})
	})

	t.Run("should be reconvert full-local qualified names", func(t *testing.T) {
		require.Equal(NewQName(SysPackage, "name"), app.LocalQName(NewFullQName(SysPackagePath, "name")))
		require.Equal(NewFullQName(SysPackagePath, "name"), app.FullQName(NewQName(SysPackage, "name")))

		require.Equal(NewQName("test", "name"), app.LocalQName(NewFullQName("test.com/path", "name")))
		require.Equal(NewFullQName("test.com/path", "name"), app.FullQName(NewQName("test", "name")))

		require.Equal(NewQName("example", "name"), app.LocalQName(NewFullQName("example.com/path", "name")))
		require.Equal(NewFullQName("example.com/path", "name"), app.FullQName(NewQName("example", "name")))

		require.Equal(NullQName, app.LocalQName(NewFullQName("unknown.com/path", "name")))
		require.Equal(NullFullQName, app.FullQName(NewQName("unknown", "name")))
	})

	t.Run("should be empties if unknown packages", func(t *testing.T) {
		require.Equal("", app.PackageLocalName("unknown.com/path"))
		require.Equal("", app.PackageFullPath("unknown"))
	})

	t.Run("test panics", func(t *testing.T) {
		adb := New()

		require.Panics(func() { adb.AddPackage("", "test.com/path") },
			require.Is(ErrMissedError))
		require.Panics(func() { adb.AddPackage("naked 🔫", "test.com/path") },
			require.Is(ErrInvalidError), require.Has("naked 🔫"))
		require.Panics(func() { adb.AddPackage("test", "") },
			require.Is(ErrMissedError))

		require.Panics(
			func() {
				adb.AddPackage("test", "test1.com/path")
				adb.AddPackage("test", "test2.com/path")
			}, require.Is(ErrAlreadyExistsError), require.Has("test"))

		require.Panics(
			func() {
				adb.AddPackage("test1", "test.com/path")
				adb.AddPackage("test2", "test.com/path")
			}, require.Is(ErrAlreadyExistsError), require.Has("test.com/path"))

		require.Panics(
			func() {
				adb.AddPackage(SysPackage, "test.com/sys")
			}, require.Is(ErrAlreadyExistsError), require.Has(SysPackage))
	})
}
