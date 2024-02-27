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

		appDef.AddPackage("test", "test/path")
		appDef.AddPackage("example", "example/path")

		a, err := appDef.Build()
		require.NoError(err)
		app = a
	})

	t.Run("should be ok to inspect packages", func(t *testing.T) {
		require.Equal("test", app.PackageLocalName("test/path"))
		require.Equal("test/path", app.PackageFullPath("test"))

		require.Equal("example", app.PackageLocalName("example/path"))
		require.Equal("example/path", app.PackageFullPath("example"))

		require.EqualValues([]string{"example", "test"}, app.PackageLocalNames())
	})

	t.Run("should be empties if unknown packages", func(t *testing.T) {
		require.Equal("", app.PackageLocalName("unknown/path"))
		require.Equal("", app.PackageFullPath("unknown"))
	})

	t.Run("test panics", func(t *testing.T) {
		appDef := New()

		require.Panics(func() { appDef.AddPackage("", "test/path") }, "should be panic if empty name")
		require.Panics(func() { appDef.AddPackage("naked 🔫", "test/path") }, "should be panic if invalid name")
		require.Panics(func() { appDef.AddPackage("test", "") }, "should be panic if empty path")

		require.Panics(
			func() {
				appDef.AddPackage("test", "test/path1")
				appDef.AddPackage("test", "test/path2")
			}, "should be panic if reuse local name")

		require.Panics(
			func() {
				appDef.AddPackage("test1", "test/path")
				appDef.AddPackage("test2", "test/path")
			}, "should be panic if reuse path")
	})
}
