/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddCDoc(t *testing.T) {
	require := require.New(t)

	wsName := NewQName("test", "workspace")
	docName, recName := NewQName("test", "doc"), NewQName("test", "rec")

	var app IAppDef

	t.Run("should be ok to add document", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		doc := ws.AddCDoc(docName)
		doc.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)
		doc.AddContainer("rec", recName, 0, Occurs_Unbounded)
		rec := ws.AddCRecord(recName)
		rec.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWithCDocs := func(cDocs IWithCDocs) {
		t.Run("should be ok to find builded doc", func(t *testing.T) {
			typ := app.Type(docName)
			require.Equal(TypeKind_CDoc, typ.Kind())

			doc := cDocs.CDoc(docName)
			require.Equal(TypeKind_CDoc, doc.Kind())
			require.Equal(typ.(ICDoc), doc)
			require.NotPanics(func() { doc.isCDoc() })

			require.Equal(2, doc.UserFieldCount())
			require.Equal(DataKind_int64, doc.Field("f1").DataKind())

			require.Equal(TypeKind_CRecord, doc.Container("rec").Type().Kind())

			t.Run("should be ok to find builded record", func(t *testing.T) {
				typ := app.Type(recName)
				require.Equal(TypeKind_CRecord, typ.Kind())

				rec := cDocs.CRecord(recName)
				require.Equal(TypeKind_CRecord, rec.Kind())
				require.Equal(typ.(ICRecord), rec)

				require.Equal(2, rec.UserFieldCount())
				require.Equal(DataKind_int64, rec.Field("f1").DataKind())

				require.Zero(rec.ContainerCount())
			})
		})

		unknownName := NewQName("test", "unknown")
		require.Nil(cDocs.CDoc(unknownName))
		require.Nil(cDocs.CRecord(unknownName))

		t.Run("should be ok to enumerate docs", func(t *testing.T) {
			var docs []QName
			for doc := range cDocs.CDocs {
				docs = append(docs, doc.QName())
			}
			require.Len(docs, 1)
			require.Equal(docName, docs[0])
			t.Run("should be ok to enumerate recs", func(t *testing.T) {
				var recs []QName
				for rec := range cDocs.CRecords {
					recs = append(recs, rec.QName())
				}
				require.Len(recs, 1)
				require.Equal(recName, recs[0])
			})
		})
	}

	testWithCDocs(app)
	testWithCDocs(app.Workspace(wsName))
}

func Test_AppDef_AddCDocSingleton(t *testing.T) {
	require := require.New(t)

	docName := NewQName("test", "doc")

	var app IAppDef

	t.Run("should be ok to add singleton", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(NewQName("test", "workspace"))

		doc := ws.AddCDoc(docName)
		doc.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)
		doc.SetSingleton()

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("should be ok to find builded singleton", func(t *testing.T) {
		typ := app.Type(docName)
		require.Equal(TypeKind_CDoc, typ.Kind())

		doc := app.CDoc(docName)
		require.Equal(TypeKind_CDoc, doc.Kind())
		require.Equal(typ.(ICDoc), doc)

		require.True(doc.Singleton())
	})
}
