/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddObject(t *testing.T) {
	require := require.New(t)

	rootName, childName := NewQName("test", "root"), NewQName("test", "child")

	var app IAppDef

	t.Run("must be ok to add objects", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		root := adb.AddObject(rootName)
		root.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)
		root.AddContainer("child", childName, 0, Occurs_Unbounded)
		child := adb.AddObject(childName)
		child.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("must be ok to find builded root object", func(t *testing.T) {
		typ := app.Type(rootName)
		require.Equal(TypeKind_Object, typ.Kind())

		root := app.Object(rootName)
		require.Equal(TypeKind_Object, root.Kind())
		require.Equal(typ.(IObject), root)
		require.NotPanics(func() { root.isObject() })

		require.NotNil(root.Field(SystemField_QName))

		require.Equal(2, root.UserFieldCount())
		require.Equal(DataKind_int64, root.Field("f1").DataKind())

		require.Equal(TypeKind_Object, root.Container("child").Type().Kind())

		t.Run("must be ok to find builded child object", func(t *testing.T) {
			typ := app.Type(childName)
			require.Equal(TypeKind_Object, typ.Kind())

			child := app.Object(childName)
			require.Equal(TypeKind_Object, child.Kind())
			require.Equal(typ.(IObject), child)

			require.NotNil(child.Field(SystemField_QName))
			require.NotNil(child.Field(SystemField_Container))

			require.Equal(2, child.UserFieldCount())
			require.Equal(DataKind_int64, child.Field("f1").DataKind())

			require.Zero(child.ContainerCount())
		})
	})

	t.Run("must be ok to enumerate objects", func(t *testing.T) {
		var objects []QName
		for obj := range app.Objects {
			objects = append(objects, obj.QName())
		}
		require.Len(objects, 2)
		require.Equal(childName, objects[0])
		require.Equal(rootName, objects[1])
	})
}
