/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package types_test

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Types(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	docName := appdef.NewQName("test", "doc")

	var app appdef.IAppDef

	t.Run("should be ok to add type", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)
		require.Equal("Workspace «test.workspace»", fmt.Sprint(wsb))

		wsb.AddCDoc(docName)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested types.IWithTypes) {
		t.Run("should be ok to find builded type", func(t *testing.T) {
			doc := appdef.CDoc(tested.Type, docName)
			require.NotNil(doc)
			require.Equal(app, doc.App())
			require.False(doc.IsSystem())
			require.Equal(docName, doc.QName())
			require.Equal(appdef.TypeKind_CDoc, doc.Kind())
			require.Equal("CDoc «test.doc»", fmt.Sprint(doc))

			require.Equal(appdef.TypeKind_null, tested.Type(appdef.NewQName("test", "unknown")).Kind(), "should nil if unknown type")
		})

		require.Equal(appdef.TypeKind_null, tested.Type(appdef.NullQName).Kind(), "should be ok to find NullType")
		require.Equal(appdef.TypeKind_Any, tested.Type(appdef.QNameANY).Kind(), "should be ok to find AnyType")
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}

func Test_TypesPanics(t *testing.T) {
	require := require.New(t)

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	wsName := appdef.NewQName("test", "workspace")
	wsb := adb.AddWorkspace(wsName)

	t.Run("should be panics", func(t *testing.T) {
		require.Panics(func() {
			wsb.AddCDoc(appdef.NewQName("test", "naked 🔫"))
		}, require.Is(appdef.ErrInvalidError), require.Has("naked 🔫"),
			"if invalid name")

		require.Panics(func() {
			wsb.AddCDoc(appdef.NullQName)
		}, require.Is(appdef.ErrMissedError),
			"if missed name")

		require.Panics(func() {
			wsb.AddCDoc(wsName)
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has(wsName),
			"if dupe name")
	})
}

func Test_TypeRef(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	docName := appdef.NewQName("test", "doc")

	var app appdef.IAppDef

	t.Run("should be ok to add type", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddCDoc(docName)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("should be ok to make ref to type", func(t *testing.T) {
		ref := types.TypeRef{}
		ref.SetName(docName)
		require.Equal(docName, ref.Name())

		require.True(ref.Valid(app.Type))

		doc := ref.Target(app.Type)
		require.NotNil(doc)
		require.Equal(doc, appdef.CDoc(app.Type, docName))
	})

	t.Run("should be ok to make ref to unknown type", func(t *testing.T) {
		unknown := appdef.NewQName("test", "unknown")

		ref := types.TypeRef{}
		ref.SetName(unknown)
		require.Equal(unknown, ref.Name())

		require.False(ref.Valid(app.Type))

		null := ref.Target(app.Type)
		require.Nil(null)
	})

	t.Run("should be ok to make ref to null", func(t *testing.T) {
		ref := types.TypeRef{}
		ref.SetName(appdef.NullQName)
		require.Equal(appdef.NullQName, ref.Name())

		require.True(ref.Valid(app.Type))

		null := ref.Target(app.Type)
		require.Nil(null)
	})

	t.Run("should be ok to make ref to any", func(t *testing.T) {
		ref := types.TypeRef{}
		ref.SetName(appdef.QNameANY)
		require.Equal(appdef.QNameANY, ref.Name())

		require.True(ref.Valid(app.Type))

		anyRef := ref.Target(app.Type)
		require.NotNil(anyRef)
		require.Equal(appdef.TypeKind_Any, anyRef.Kind())
	})
}
