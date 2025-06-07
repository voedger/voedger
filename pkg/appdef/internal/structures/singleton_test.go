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

func Test_Singletons(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	stName := appdef.NewQName("test", "singleton")
	docName := appdef.NewQName("test", "doc")

	var app appdef.IAppDef

	t.Run("should be ok to add singleton", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		st := wsb.AddCDoc(stName)
		st.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)
		st.SetSingleton()

		_ = wsb.AddCDoc(docName).
			AddField("f1", appdef.DataKind_int64, true)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested types.IWithTypes) {
		t.Run("should be ok to find builded singleton", func(t *testing.T) {
			typ := tested.Type(stName)
			require.Equal(appdef.TypeKind_CDoc, typ.Kind())

			st := appdef.CDoc(tested.Type, stName)
			require.Equal(appdef.TypeKind_CDoc, st.Kind())
			require.Equal(typ.(appdef.ICDoc), st)

			require.True(st.Singleton())
		})

		t.Run("should be ok to enum singleton", func(t *testing.T) {
			names := appdef.QNames{}
			for st := range appdef.Singletons(tested.Types()) {
				names = append(names, st.QName())
			}
			require.Len(names, 1)
			require.Equal(stName, names[0])
		})

		require.Nil(appdef.Singleton(tested.Type, appdef.NewQName("test", "unknown")), "should be nil if unknown")
		require.Nil(appdef.Singleton(tested.Type, docName), "should be nil if not singleton")
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}
