/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddODoc(t *testing.T) {
	require := require.New(t)

	wsName := NewQName("test", "workspace")
	docName, recName := NewQName("test", "doc"), NewQName("test", "rec")

	var app IAppDef

	t.Run("should be ok to add document", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		doc := wsb.AddODoc(docName)
		doc.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)
		doc.AddContainer("rec", recName, 0, Occurs_Unbounded)
		rec := wsb.AddORecord(recName)
		rec.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested testedTypes) {
		t.Run("should be ok to find builded doc", func(t *testing.T) {
			typ := tested.Type(docName)
			require.Equal(TypeKind_ODoc, typ.Kind())

			doc := ODoc(tested.Type, docName)
			require.Equal(TypeKind_ODoc, doc.Kind())
			require.Equal(typ.(IODoc), doc)
			require.NotPanics(func() { doc.isODoc() })

			require.Equal(wsName, doc.Workspace().QName())

			require.Equal(2, doc.UserFieldCount())
			require.Equal(DataKind_int64, doc.Field("f1").DataKind())

			require.Equal(TypeKind_ORecord, doc.Container("rec").Type().Kind())

			t.Run("should be ok to find builded record", func(t *testing.T) {
				typ := app.Type(recName)
				require.Equal(TypeKind_ORecord, typ.Kind())

				rec := ORecord(tested.Type, recName)
				require.Equal(TypeKind_ORecord, rec.Kind())
				require.Equal(typ.(IORecord), rec)
				require.NotPanics(func() { rec.isORecord() })

				require.Equal(wsName, rec.Workspace().QName())

				require.Equal(2, rec.UserFieldCount())
				require.Equal(DataKind_int64, rec.Field("f1").DataKind())

				require.Equal(0, rec.ContainerCount())
			})
		})

		t.Run("should be ok to enumerate docs", func(t *testing.T) {
			var docs []QName
			for doc := range ODocs(tested.Types) {
				docs = append(docs, doc.QName())
			}
			require.Len(docs, 1)
			require.Equal(docName, docs[0])
			t.Run("should be ok to enumerate recs", func(t *testing.T) {
				var recs []QName
				for rec := range ORecords(tested.Types) {
					recs = append(recs, rec.QName())
				}
				require.Len(recs, 1)
				require.Equal(recName, recs[0])
			})
		})

		t.Run("should nil if not found", func(t *testing.T) {
			unknown := NewQName("test", "unknown")
			require.Nil(ODoc(tested.Type, unknown))
			require.Nil(ORecord(tested.Type, unknown))
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}
