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

	wsName := NewQName("test", "workspace")
	rootName, childName := NewQName("test", "root"), NewQName("test", "child")

	var app IAppDef

	t.Run("should be ok to add objects", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		root := wsb.AddObject(rootName)
		root.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)
		root.AddContainer("child", childName, 0, Occurs_Unbounded)
		child := wsb.AddObject(childName)
		child.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWithObjects := func(tested IWithObjects) {

		t.Run("should be ok to find builded root object", func(t *testing.T) {
			typ := tested.(IWithTypes).Type(rootName)
			require.Equal(TypeKind_Object, typ.Kind())

			root := tested.Object(rootName)
			require.Equal(TypeKind_Object, root.Kind())
			require.Equal(typ.(IObject), root)
			require.NotPanics(func() { root.isObject() })

			require.NotNil(root.Field(SystemField_QName))

			require.Equal(2, root.UserFieldCount())
			require.Equal(DataKind_int64, root.Field("f1").DataKind())

			require.Equal(TypeKind_Object, root.Container("child").Type().Kind())

			t.Run("should be ok to find builded child object", func(t *testing.T) {
				typ := tested.(IWithTypes).Type(childName)
				require.Equal(TypeKind_Object, typ.Kind())

				child := tested.Object(childName)
				require.Equal(TypeKind_Object, child.Kind())
				require.Equal(typ.(IObject), child)

				require.NotNil(child.Field(SystemField_QName))
				require.NotNil(child.Field(SystemField_Container))

				require.Equal(2, child.UserFieldCount())
				require.Equal(DataKind_int64, child.Field("f1").DataKind())

				require.Zero(child.ContainerCount())
			})
		})

		t.Run("should be ok to enumerate objects", func(t *testing.T) {
			var objects []QName
			for obj := range app.Objects {
				objects = append(objects, obj.QName())
			}
			require.Len(objects, 2)
			require.Equal(childName, objects[0])
			require.Equal(rootName, objects[1])
		})

		require.Nil(tested.Object(NewQName("test", "unknown")), "should be nil if unknown")
	}

	testWithObjects(app)
	testWithObjects(app.Workspace(wsName))
}
