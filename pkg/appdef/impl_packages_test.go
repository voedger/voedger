/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddPackage(t *testing.T) {
	require := require.New(t)

	var app IAppDef

	t.Run("should be ok to add package", func(t *testing.T) {
		appDef := New()

		appDef.AddPackage("test", "test.com/path")
		appDef.AddPackage("example", "example.com/path")

		a, err := appDef.Build()
		require.NoError(err)
		app = a
	})

	t.Run("should be ok to inspect packages", func(t *testing.T) {
		require.Equal("test", app.PackageLocalName("test.com/path"))
		require.Equal("test.com/path", app.PackageFullPath("test"))

		require.Equal("example", app.PackageLocalName("example.com/path"))
		require.Equal("example.com/path", app.PackageFullPath("example"))

		require.EqualValues([]string{"example", "test"}, app.PackageLocalNames())

		cnt := 0
		app.Packages(func(localName, fullPath string) {
			switch cnt {
			case 0:
				require.Equal("example", localName)
				require.Equal("example.com/path", fullPath)
			case 1:
				require.Equal("test", localName)
				require.Equal("test.com/path", fullPath)
			default:
				require.Fail("unexpected package %v (%v)", localName, fullPath)
			}
			cnt++
		})
		require.Equal(2, cnt)
	})

	t.Run("should be reconvert full-local qualified names", func(t *testing.T) {
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
		appDef := New()

		require.Panics(func() { appDef.AddPackage("", "test.com/path") }, "should be panic if empty name")
		require.Panics(func() { appDef.AddPackage("naked 🔫", "test.com/path") }, "should be panic if invalid name")
		require.Panics(func() { appDef.AddPackage("test", "") }, "should be panic if empty path")

		require.Panics(
			func() {
				appDef.AddPackage("test", "test1.com/path")
				appDef.AddPackage("test", "test2.com/path")
			}, "should be panic if reuse local name")

		require.Panics(
			func() {
				appDef.AddPackage("test1", "test.com/path")
				appDef.AddPackage("test2", "test.com/path")
			}, "should be panic if reuse path")
	})
}
