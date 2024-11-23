/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_AddQuery(t *testing.T) {
	require := require.New(t)

	var app IAppDef
	wsName := NewQName("test", "workspace")
	queryName, parName, resName := NewQName("test", "query"), NewQName("test", "param"), NewQName("test", "res")

	t.Run("should be ok to add query", func(t *testing.T) {
		adb := New()
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

	testWith := func(tested testedTypes) {
		t.Run("should be ok to find builded query", func(t *testing.T) {
			typ := tested.Type(queryName)
			require.Equal(TypeKind_Query, typ.Kind())

			q, ok := typ.(IQuery)
			require.True(ok)
			require.Equal(TypeKind_Query, q.Kind())

			query := Query(tested.Type, queryName)
			require.Equal(TypeKind_Query, query.Kind())
			require.Equal(wsName, query.Workspace().QName())
			require.Equal(q, query)
			require.NotPanics(func() { query.isQuery() })

			require.Equal(queryName.Entity(), query.Name())
			require.Equal(ExtensionEngineKind_BuiltIn, query.Engine())

			require.Equal(parName, query.Param().QName())
			require.Equal(TypeKind_Object, query.Param().Kind())

			require.Equal(resName, query.Result().QName())
			require.Equal(TypeKind_Object, query.Result().Kind())
		})

		t.Run("should be ok to enum queries", func(t *testing.T) {
			cnt := 0
			for q := range Queries(tested.Types) {
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

		require.Nil(Query(tested.Type, NewQName("test", "unknown")), "should be nil if unknown")
	}

	testWith(app)
	testWith(app.Workspace(wsName))

	t.Run("should be panics ", func(t *testing.T) {
		require.Panics(func() {
			New().AddWorkspace(wsName).AddQuery(NullQName)
		}, require.Is(ErrMissedError))

		require.Panics(func() {
			New().AddWorkspace(wsName).AddQuery(NewQName("naked", "🔫"))
		}, require.Is(ErrInvalidError), require.Has("naked.🔫"))

		t.Run("if type with name already exists", func(t *testing.T) {
			testName := NewQName("test", "dupe")
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			wsb.AddObject(testName)
			require.Panics(func() {
				wsb.AddQuery(testName)
			}, require.Is(ErrAlreadyExistsError), require.Has(testName))
		})

		t.Run("if extension name is empty", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			query := wsb.AddQuery(queryName)
			require.Panics(func() {
				query.SetName("")
			}, require.Is(ErrMissedError))
		})

		t.Run("if extension name is invalid", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			query := wsb.AddQuery(queryName)
			require.Panics(func() {
				query.SetName("naked 🔫")
			}, require.Is(ErrInvalidError), require.Has("🔫"))
		})
	})
}

func Test_QueryValidate(t *testing.T) {
	require := require.New(t)

	adb := New()
	adb.AddPackage("test", "test.com/test")

	wsb := adb.AddWorkspace(NewQName("test", "workspace"))

	query := wsb.AddQuery(NewQName("test", "query"))

	t.Run("should be error if parameter name is unknown", func(t *testing.T) {
		par := NewQName("test", "param")
		query.SetParam(par)
		_, err := adb.Build()
		require.Error(err, require.Is(ErrNotFoundError), require.Has(par))

		_ = wsb.AddObject(par)
	})

	t.Run("should be error if result object name is unknown", func(t *testing.T) {
		res := NewQName("test", "res")
		query.SetResult(res)
		_, err := adb.Build()
		require.Error(err, require.Is(ErrNotFoundError), require.Has(res))

		_ = wsb.AddObject(res)
	})

	_, err := adb.Build()
	require.NoError(err)
}

func Test_AppDef_AddQueryWithAnyResult(t *testing.T) {
	require := require.New(t)

	var app IAppDef
	wsName := NewQName("test", "workspace")
	queryName := NewQName("test", "query")

	t.Run("should be ok to add query with any result", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		query := wsb.AddQuery(queryName)
		query.
			SetResult(QNameANY)

		a, err := adb.Build()
		require.NoError(err)
		require.NotNil(a)

		app = a
	})

	require.NotNil(app)

	t.Run("should be ok to find builded query", func(t *testing.T) {
		query := Query(app.Type, queryName)

		require.Equal(AnyType, query.Result())
		require.Equal(QNameANY, query.Result().QName())
		require.Equal(TypeKind_Any, query.Result().Kind())
		require.Equal(wsName, query.Workspace().QName())

		require.Equal(query, Query(app.Workspace(wsName).Type, queryName))
	})
}
