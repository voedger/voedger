/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddQuery(t *testing.T) {
	require := require.New(t)

	var app IAppDef
	queryName, argName, resName := NewQName("test", "query"), NewQName("test", "arg"), NewQName("test", "res")

	t.Run("must be ok to add query", func(t *testing.T) {
		appDef := New()

		_ = appDef.AddObject(argName)
		_ = appDef.AddObject(resName)

		query := appDef.AddQuery(queryName)
		require.Equal(TypeKind_Query, query.Kind())
		require.Equal(query, appDef.Query(queryName))
		require.Nil(query.Arg())
		require.Nil(query.Result())

		t.Run("must be ok to assign query args and result", func(t *testing.T) {
			query.
				SetArg(argName).
				SetResult(resName).
				SetExtension("QueryExt", ExtensionEngineKind_BuiltIn)
		})

		t.Run("must be ok to build", func(t *testing.T) {
			a, err := appDef.Build()
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

		require.Equal(argName, query.Arg().QName())
		require.Equal(TypeKind_Object, query.Arg().Kind())

		require.Equal(resName, query.Result().QName())
		require.Equal(TypeKind_Object, query.Result().Kind())

		require.Equal("QueryExt", query.Extension().Name())
		require.Equal(ExtensionEngineKind_BuiltIn, query.Extension().Engine())
	})

	t.Run("must be ok to enum functions", func(t *testing.T) {
		cnt := 0
		app.Functions(func(f IFunction) {
			cnt++
			switch cnt {
			case 1:
				require.Equal(TypeKind_Query, f.Kind())
				require.Equal(queryName, f.QName())
			default:
				require.Failf("unexpected function", "kind: %v, name: %v", f.Kind(), f.QName())
			}
		})
		require.Equal(1, cnt)
	})

	t.Run("check nil returns", func(t *testing.T) {
		unknown := NewQName("test", "unknown")
		require.Nil(app.Query(unknown))
	})

	t.Run("panic if name is empty", func(t *testing.T) {
		apb := New()
		require.Panics(func() {
			apb.AddQuery(NullQName)
		})
	})

	t.Run("panic if name is invalid", func(t *testing.T) {
		apb := New()
		require.Panics(func() {
			apb.AddQuery(NewQName("naked", "🔫"))
		})
	})

	t.Run("panic if type with name already exists", func(t *testing.T) {
		testName := NewQName("test", "dupe")
		apb := New()
		apb.AddObject(testName)
		require.Panics(func() {
			apb.AddQuery(testName)
		})
	})

	t.Run("panic if extension name is empty", func(t *testing.T) {
		apb := New()
		query := apb.AddQuery(NewQName("test", "query"))
		require.Panics(func() {
			query.SetExtension("", ExtensionEngineKind_BuiltIn)
		})
	})

	t.Run("panic if extension name is invalid", func(t *testing.T) {
		apb := New()
		query := apb.AddQuery(NewQName("test", "query"))
		require.Panics(func() {
			query.SetExtension("naked 🔫", ExtensionEngineKind_BuiltIn)
		})
	})
}

func Test_QueryValidate(t *testing.T) {
	require := require.New(t)

	appDef := New()

	query := appDef.AddQuery(NewQName("test", "query"))

	t.Run("must error if argument name is unknown", func(t *testing.T) {
		arg := NewQName("test", "arg")
		query.SetArg(arg)
		_, err := appDef.Build()
		require.ErrorIs(err, ErrNameNotFound)
		require.ErrorContains(err, arg.String())

		_ = appDef.AddObject(arg)
	})

	t.Run("must error if result object name is unknown", func(t *testing.T) {
		res := NewQName("test", "res")
		query.SetResult(res)
		_, err := appDef.Build()
		require.ErrorIs(err, ErrNameNotFound)
		require.ErrorContains(err, res.String())

		_ = appDef.AddObject(res)
	})

	t.Run("must error if extension name or engine is missed", func(t *testing.T) {
		_, err := appDef.Build()
		require.ErrorIs(err, ErrNameMissed)
		require.ErrorContains(err, "extension name")

		require.ErrorIs(err, ErrExtensionEngineKindMissed)
	})

	query.SetExtension("QueryExt", ExtensionEngineKind_BuiltIn)
	_, err := appDef.Build()
	require.NoError(err)
}
