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
		obj := appDef.AddObject(objName)
		obj.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)
		obj.AddContainer("child", elementName, 0, Occurs_Unbounded)
		el := appDef.AddElement(elementName)
		el.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)

		a, err := appDef.Build()
		require.NoError(err)

		app = a
	})

	t.Run("must be ok to find builded object", func(t *testing.T) {
		typ := app.Type(objName)
		require.Equal(TypeKind_Object, typ.Kind())

		obj := app.Object(objName)
		require.Equal(TypeKind_Object, obj.Kind())
		require.Equal(typ.(IObject), obj)

		require.Equal(2, obj.UserFieldCount())
		require.Equal(DataKind_int64, obj.Field("f1").DataKind())

		require.Equal(TypeKind_Element, obj.Container("child").Type().Kind())

		t.Run("must be ok to find builded element", func(t *testing.T) {
			typ := app.Type(elementName)
			require.Equal(TypeKind_Element, typ.Kind())

			el := app.Element(elementName)
			require.Equal(TypeKind_Element, el.Kind())
			require.Equal(typ.(IElement), el)

			require.Equal(2, el.UserFieldCount())
			require.Equal(DataKind_int64, el.Field("f1").DataKind())

			require.Equal(0, el.ContainerCount())
		})
	})
}
