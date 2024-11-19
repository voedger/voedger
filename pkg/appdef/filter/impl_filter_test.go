/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"iter"
	"slices"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_filter(t *testing.T) {
	require := require.New(t)
	f := filter{}
	for range f.And() {
		require.Fail("filter.And() should be empty")
	}
	require.Nil(f.Not(), "filter.Not() should be nil")
	for range f.Or() {
		require.Fail("filter.Or() should be empty")
	}
	for range f.QNames() {
		require.Fail("filter.QNames() should be empty")
	}
	for range f.Tags() {
		require.Fail("filter.Tags() should be empty")
	}
	for range f.Types() {
		require.Fail("filter.Types() should be empty")
	}
}

func Test_allMatches(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	doc, obj, cmd := appdef.NewQName("test", "doc"), appdef.NewQName("test", "object"), appdef.NewQName("test", "command")

	app := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddODoc(doc)
		_ = wsb.AddObject(obj)
		_ = wsb.AddCommand(cmd)

		return adb.MustBuild()
	}()

	ws := app.Workspace(wsName)

	flt := Or(AllTables(), Types(appdef.TypeKind_Command))

	t.Run("should return all matches", func(t *testing.T) {
		require.Equal([]appdef.IType{
			ws.Type(cmd),
			ws.Type(doc),
			ws.Type(obj),
		}, slices.Collect(iter.Seq[appdef.IType](Matches(flt, ws.LocalTypes))))
	})

	t.Run("should be breakable", func(t *testing.T) {
		cnt := 0
		for range Matches(flt, ws.LocalTypes) {
			cnt++
			break
		}
		require.Equal(1, cnt)
	})
}
