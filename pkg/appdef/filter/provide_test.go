/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_QNames(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	docName := appdef.NewQName("test", "doc")

	app := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddCDoc(docName)

		app, err := adb.Build()

		require.NoError(err)
		return app
	}()

	flt := filter.QNames(docName)

	doc := appdef.CDoc(app.Type, docName)
	require.NotNil(doc, "Doc should be found")
	require.True(flt.Match(doc), "Doc should be matched")

	require.False(flt.Match(app.Workspace(wsName)), "Workspace should not be matched")

	t.Run("should be panics", func(t *testing.T) {
		require.Panics(func() {
			_ = filter.QNames()
		}, "if no qnames are provided")
	})
}

func Test_Tags(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	doc1Name, doc2Name := appdef.NewQName("test", "doc1"), appdef.NewQName("test", "doc2")
	tagName := appdef.NewQName("test", "tag")

	app := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddTag(tagName)
		wsb.AddCDoc(doc1Name).SetTag(tagName)
		_ = wsb.AddCDoc(doc2Name)

		app, err := adb.Build()

		require.NoError(err)
		return app
	}()

	flt := filter.Tags(tagName)

	require.True(flt.Match(appdef.CDoc(app.Type, doc1Name)), "Doc1 should be matched")
	require.False(flt.Match(appdef.CDoc(app.Type, doc2Name)), "Doc2 should not be matched")

	t.Run("should be panics", func(t *testing.T) {
		require.Panics(func() {
			_ = filter.Tags()
		}, "if no tags are provided")
	})
}

func Test_WSTypes(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	dataName := appdef.NewQName("test", "data")

	app := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddData(dataName, appdef.DataKind_int32, appdef.NullQName, appdef.MinIncl(0))

		app, err := adb.Build()

		require.NoError(err)
		return app
	}()

	flt := filter.WSTypes(wsName, appdef.TypeKind_Data)

	ws := app.Workspace(wsName)

	data := appdef.Data(ws.Type, dataName)
	require.NotNil(data, "Data should be found")
	require.True(flt.Match(data), "Data should be matched")

	sysInt32 := appdef.Data(ws.Type, appdef.SysDataName(appdef.DataKind_int32))
	require.NotNil(sysInt32, "system sys.Int32 should be found")
	require.False(flt.Match(sysInt32), "system data should not be matched")

	t.Run("should be panics", func(t *testing.T) {
		require.Panics(func() {
			_ = filter.WSTypes(appdef.NullQName, appdef.TypeKind_Data)
		}, "if workspace name is null")
		require.Panics(func() {
			_ = filter.WSTypes(wsName)
		}, "if no type kinds are provided")
	})
}
