/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package extensions_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Queries(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef
	wsName := appdef.NewQName("test", "workspace")
	queryName, parName, resName := appdef.NewQName("test", "query"), appdef.NewQName("test", "param"), appdef.NewQName("test", "res")

	t.Run("should be ok to add query", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddObject(parName)
		_ = wsb.AddObject(resName)

		query := wsb.AddQuery(queryName)

		t.Run("should be ok to assign query params and result", func(t *testing.T) {
			query.
				SetParam(parName).
				SetResult(resName)
		})

		t.Run("should be ok to build", func(t *testing.T) {
			a, err := adb.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	t.Run("should be ok to find builded query", func(t *testing.T) {
		typ := app.Type(queryName)
		require.Equal(appdef.TypeKind_Query, typ.Kind())

		q, ok := typ.(appdef.IQuery)
		require.True(ok)
		require.Equal(appdef.TypeKind_Query, q.Kind())
		q.IsQuery()

		query := appdef.Query(app.Type, queryName)
		require.Equal(appdef.TypeKind_Query, query.Kind())
		require.Equal(wsName, query.Workspace().QName())
		require.Equal(q, query)

		require.Equal(queryName.Entity(), query.Name())
		require.Equal(appdef.ExtensionEngineKind_BuiltIn, query.Engine())

		require.Equal(parName, query.Param().QName())
		require.Equal(appdef.TypeKind_Object, query.Param().Kind())

		require.Equal(resName, query.Result().QName())
		require.Equal(appdef.TypeKind_Object, query.Result().Kind())
	})

	t.Run("should be ok to enum queries", func(t *testing.T) {
		cnt := 0
		for q := range appdef.Queries(app.Types()) {
			if q.IsSystem() {
				continue
			}
			cnt++
			switch cnt {
			case 1:
				require.Equal(queryName, q.QName())
			default:
				require.Failf("unexpected query", "query: %v", q)
			}
		}
		require.Equal(1, cnt)
	})

	require.Nil(appdef.Query(app.Type, appdef.NewQName("test", "unknown")), "should be nil if unknown")
}

func Test_QueryValidate(t *testing.T) {
	require := require.New(t)

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

	query := wsb.AddQuery(appdef.NewQName("test", "query"))

	t.Run("should be error if parameter name is unknown", func(t *testing.T) {
		par := appdef.NewQName("test", "param")
		query.SetParam(par)
		_, err := adb.Build()
		require.Error(err, require.Is(appdef.ErrNotFoundError), require.Has(par))

		_ = wsb.AddObject(par)
	})

	t.Run("should be error if result object name is unknown", func(t *testing.T) {
		res := appdef.NewQName("test", "res")
		query.SetResult(res)
		_, err := adb.Build()
		require.Error(err, require.Is(appdef.ErrNotFoundError), require.Has(res))

		_ = wsb.AddObject(res)
	})

	_, err := adb.Build()
	require.NoError(err)
}

func Test_QueryWithAnyResult(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef
	wsName := appdef.NewQName("test", "workspace")
	queryName := appdef.NewQName("test", "query")

	t.Run("should be ok to add query with any result", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		query := wsb.AddQuery(queryName)
		query.
			SetResult(appdef.QNameANY)

		a, err := adb.Build()
		require.NoError(err)
		require.NotNil(a)

		app = a
	})

	require.NotNil(app)

	t.Run("should be ok to find builded query", func(t *testing.T) {
		query := appdef.Query(app.Type, queryName)

		require.Equal(appdef.AnyType, query.Result())
		require.Equal(appdef.QNameANY, query.Result().QName())
		require.Equal(wsName, query.Workspace().QName())

		require.Equal(query, appdef.Query(app.Workspace(wsName).Type, queryName))
	})
}
