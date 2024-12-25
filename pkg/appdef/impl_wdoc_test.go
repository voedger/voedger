/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

func Test_AppDef_AddWDoc(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	var app appdef.IAppDef

	t.Run("should be ok to add document", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		doc := wsb.AddWDoc(docName)
		doc.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)
		doc.AddContainer("rec", recName, 0, appdef.Occurs_Unbounded)
		rec := wsb.AddWRecord(recName)
		rec.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested testedTypes) {
		t.Run("should be ok to find builded doc", func(t *testing.T) {
			typ := tested.Type(docName)
			require.Equal(appdef.TypeKind_WDoc, typ.Kind())

			doc := appdef.WDoc(tested.Type, docName)
			require.Equal(appdef.TypeKind_WDoc, doc.Kind())
			require.Equal(typ.(appdef.IWDoc), doc)

			require.Equal(wsName, doc.Workspace().QName())

			require.Equal(2, doc.UserFieldCount())
			require.Equal(appdef.DataKind_int64, doc.Field("f1").DataKind())

			require.Equal(appdef.TypeKind_WRecord, doc.Container("rec").Type().Kind())

			t.Run("should be ok to find builded record", func(t *testing.T) {
				typ := app.Type(recName)
				require.Equal(appdef.TypeKind_WRecord, typ.Kind())

				rec := appdef.WRecord(tested.Type, recName)
				require.Equal(appdef.TypeKind_WRecord, rec.Kind())
				require.Equal(typ.(appdef.IWRecord), rec)

				require.Equal(wsName, rec.Workspace().QName())

				require.Equal(2, rec.UserFieldCount())
				require.Equal(appdef.DataKind_int64, rec.Field("f1").DataKind())

				require.Zero(rec.ContainerCount())
			})
		})

		t.Run("should be ok to enumerate docs", func(t *testing.T) {
			var docs []appdef.QName
			for doc := range appdef.WDocs(tested.Types()) {
				docs = append(docs, doc.QName())
			}
			require.Len(docs, 1)
			require.Equal(docName, docs[0])
			t.Run("should be ok to enumerate recs", func(t *testing.T) {
				var recs []appdef.QName
				for rec := range appdef.WRecords(tested.Types()) {
					recs = append(recs, rec.QName())
				}
				require.Len(recs, 1)
				require.Equal(recName, recs[0])
			})
		})

		t.Run("should nil if not found", func(t *testing.T) {
			unknown := appdef.NewQName("test", "unknown")
			require.Nil(appdef.WDoc(tested.Type, unknown))
			require.Nil(appdef.WRecord(tested.Type, unknown))
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}

func Test_AppDef_AddWDocSingleton(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	docName := appdef.NewQName("test", "doc")

	var app appdef.IAppDef

	t.Run("should be ok to add singleton", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		doc := wsb.AddWDoc(docName)
		doc.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)
		doc.SetSingleton()

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested appdef.FindType) {
		t.Run("should be ok to find builded singleton", func(t *testing.T) {
			typ := tested(docName)
			require.Equal(appdef.TypeKind_WDoc, typ.Kind())

			doc := appdef.WDoc(tested, docName)
			require.Equal(appdef.TypeKind_WDoc, doc.Kind())
			require.Equal(typ.(appdef.IWDoc), doc)
			require.True(doc.Singleton())
		})
	}

	testWith(app.Type)
	testWith(app.Workspace(wsName).Type)
}
