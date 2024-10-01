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
	queryName, parName, resName := NewQName("test", "query"), NewQName("test", "param"), NewQName("test", "res")

	t.Run("must be ok to add query", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		_ = adb.AddObject(parName)
		_ = adb.AddObject(resName)

		query := adb.AddQuery(queryName)

		t.Run("must be ok to assign query params and result", func(t *testing.T) {
			query.
				SetParam(parName).
				SetResult(resName)
		})

		t.Run("must be ok to build", func(t *testing.T) {
			a, err := adb.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	t.Run("must be ok to find builded query", func(t *testing.T) {
		typ := app.Type(queryName)
		require.Equal(TypeKind_Query, typ.Kind())

		q, ok := typ.(IQuery)
		require.True(ok)
		require.Equal(TypeKind_Query, q.Kind())

		query := app.Query(queryName)
		require.Equal(TypeKind_Query, query.Kind())
		require.Equal(q, query)
		require.NotPanics(func() { query.isQuery() })

		require.Equal(queryName.Entity(), query.Name())
		require.Equal(ExtensionEngineKind_BuiltIn, query.Engine())

		require.Equal(parName, query.Param().QName())
		require.Equal(TypeKind_Object, query.Param().Kind())

		require.Equal(resName, query.Result().QName())
		require.Equal(TypeKind_Object, query.Result().Kind())
	})

	t.Run("must be ok to enum queries", func(t *testing.T) {
		cnt := 0
		for q := range app.Queries {
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

	require.Nil(app.Query(NewQName("test", "unknown")), "check nil returns")

	require.Panics(func() {
		New().AddQuery(NullQName)
	}, require.Is(ErrMissedError))

	require.Panics(func() {
		New().AddQuery(NewQName("naked", "🔫"))
	}, require.Is(ErrInvalidError), require.Has("naked.🔫"))

	t.Run("panic if type with name already exists", func(t *testing.T) {
		testName := NewQName("test", "dupe")
		adb := New()
		adb.AddPackage("test", "test.com/test")
		adb.AddObject(testName)
		require.Panics(func() {
			adb.AddQuery(testName)
		}, require.Is(ErrAlreadyExistsError), require.Has(testName))
	})

	t.Run("panic if extension name is empty", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		query := adb.AddQuery(NewQName("test", "query"))
		require.Panics(func() {
			query.SetName("")
		}, require.Is(ErrMissedError))
	})

	t.Run("panic if extension name is invalid", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		query := adb.AddQuery(NewQName("test", "query"))
		require.Panics(func() {
			query.SetName("naked 🔫")
		}, require.Is(ErrInvalidError), require.Has("🔫"))
	})
}

func Test_QueryValidate(t *testing.T) {
	require := require.New(t)

	adb := New()
	adb.AddPackage("test", "test.com/test")

	query := adb.AddQuery(NewQName("test", "query"))

	t.Run("must error if parameter name is unknown", func(t *testing.T) {
		par := NewQName("test", "param")
		query.SetParam(par)
		_, err := adb.Build()
		require.Error(err, require.Is(ErrNotFoundError), require.Has(par))

		_ = adb.AddObject(par)
	})

	t.Run("must error if result object name is unknown", func(t *testing.T) {
		res := NewQName("test", "res")
		query.SetResult(res)
		_, err := adb.Build()
		require.Error(err, require.Is(ErrNotFoundError), require.Has(res))

		_ = adb.AddObject(res)
	})

	_, err := adb.Build()
	require.NoError(err)
}

func Test_AppDef_AddQueryWithAnyResult(t *testing.T) {
	require := require.New(t)

	var app IAppDef
	queryName := NewQName("test", "query")

	t.Run("must be ok to add query with any result", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		query := adb.AddQuery(queryName)
		query.
			SetResult(QNameANY)

		a, err := adb.Build()
		require.NoError(err)
		require.NotNil(a)

		app = a
	})

	require.NotNil(app)

	t.Run("must be ok to find builded query", func(t *testing.T) {
		query := app.Query(queryName)

		require.Equal(AnyType, query.Result())
		require.Equal(QNameANY, query.Result().QName())
		require.Equal(TypeKind_Any, query.Result().Kind())
	})
}
