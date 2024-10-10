/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddWDoc(t *testing.T) {
	require := require.New(t)

	docName, recName := NewQName("test", "doc"), NewQName("test", "rec")

	var app IAppDef

	t.Run("must be ok to add document", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		doc := adb.AddWDoc(docName)
		doc.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)
		doc.AddContainer("rec", recName, 0, Occurs_Unbounded)
		rec := adb.AddWRecord(recName)
		rec.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("must be ok to find builded doc", func(t *testing.T) {
		typ := app.Type(docName)
		require.Equal(TypeKind_WDoc, typ.Kind())

		doc := app.WDoc(docName)
		require.Equal(TypeKind_WDoc, doc.Kind())
		require.Equal(typ.(IWDoc), doc)
		require.NotPanics(func() { doc.isWDoc() })

		require.Equal(2, doc.UserFieldCount())
		require.Equal(DataKind_int64, doc.Field("f1").DataKind())

		require.Equal(TypeKind_WRecord, doc.Container("rec").Type().Kind())

		t.Run("must be ok to find builded record", func(t *testing.T) {
			typ := app.Type(recName)
			require.Equal(TypeKind_WRecord, typ.Kind())

			rec := app.WRecord(recName)
			require.Equal(TypeKind_WRecord, rec.Kind())
			require.Equal(typ.(IWRecord), rec)
			require.NotPanics(func() { rec.isWRecord() })

			require.Equal(2, rec.UserFieldCount())
			require.Equal(DataKind_int64, rec.Field("f1").DataKind())

			require.Zero(rec.ContainerCount())
		})
	})

	t.Run("must be ok to enumerate docs", func(t *testing.T) {
		var docs []QName
		for doc := range app.WDocs {
			docs = append(docs, doc.QName())
		}
		require.Len(docs, 1)
		require.Equal(docName, docs[0])
		t.Run("must be ok to enumerate recs", func(t *testing.T) {
			var recs []QName
			for rec := range app.WRecords {
				recs = append(recs, rec.QName())
			}
			require.Len(recs, 1)
			require.Equal(recName, recs[0])
		})
	})

}

func Test_AppDef_AddWDocSingleton(t *testing.T) {
	require := require.New(t)

	docName := NewQName("test", "doc")

	var app IAppDef

	t.Run("must be ok to add singleton", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		doc := adb.AddWDoc(docName)
		doc.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)
		doc.SetSingleton()

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("must be ok to find builded singleton", func(t *testing.T) {
		typ := app.Type(docName)
		require.Equal(TypeKind_WDoc, typ.Kind())

		doc := app.WDoc(docName)
		require.Equal(TypeKind_WDoc, doc.Kind())
		require.Equal(typ.(IWDoc), doc)

		require.True(doc.Singleton())
	})
}
