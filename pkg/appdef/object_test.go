/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Short test form. Full test ref. to gdoc_test.go
func Test_AppDef_AddObject(t *testing.T) {
	require := require.New(t)

	objName, elementName := NewQName("test", "obj"), NewQName("test", "element")

	var app IAppDef

	t.Run("must be ok to add object", func(t *testing.T) {
		appDef := New()
		doc := appDef.AddObject(objName)
		doc.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)
		doc.AddContainer("child", elementName, 0, Occurs_Unbounded)
		rec := appDef.AddElement(elementName)
		rec.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)

		a, err := appDef.Build()
		require.NoError(err)

		app = a
	})

	t.Run("must be ok to find builded object", func(t *testing.T) {
		typ := app.Type(objName)
		require.Equal(TypeKind_Object, typ.Kind())

		doc := app.Object(objName)
		require.Equal(TypeKind_Object, doc.Kind())
		require.Equal(typ.(IObject), doc)

		require.Equal(2, doc.UserFieldCount())
		require.Equal(DataKind_int64, doc.Field("f1").DataKind())

		require.Equal(TypeKind_Element, doc.Container("child").Type().Kind())

		t.Run("must be ok to find builded element", func(t *testing.T) {
			typ := app.Type(elementName)
			require.Equal(TypeKind_Element, typ.Kind())

			rec := app.Element(elementName)
			require.Equal(TypeKind_Element, rec.Kind())
			require.Equal(typ.(IElement), rec)

			require.Equal(2, rec.UserFieldCount())
			require.Equal(DataKind_int64, rec.Field("f1").DataKind())

			require.Equal(0, rec.ContainerCount())
		})
	})
}
