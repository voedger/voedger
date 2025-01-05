/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package structures_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
)

func Test_Objects(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	objName := appdef.NewQName("test", "object")

	var app appdef.IAppDef

	t.Run("should be ok to add object", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		doc := ws.AddObject(objName)
		doc.AddField("f1", appdef.DataKind_int64, true)
		doc.AddContainer("child", objName, 0, appdef.Occurs_Unbounded)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested types.IWithTypes) {
		t.Run("should be ok to find builded object", func(t *testing.T) {
			doc := appdef.Object(tested.Type, objName)
			require.Equal(appdef.TypeKind_Object, doc.Kind())
			doc.IsObject()

			require.Equal(appdef.TypeKind_Object, doc.Container("child").Type().Kind())
		})

		unknownName := appdef.NewQName("test", "unknown")
		require.Nil(appdef.Object(tested.Type, unknownName))

		t.Run("should be ok to enumerate objects", func(t *testing.T) {
			var names []appdef.QName
			for o := range appdef.Objects(tested.Types()) {
				names = append(names, o.QName())
			}
			require.Equal([]appdef.QName{objName}, names)
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}
