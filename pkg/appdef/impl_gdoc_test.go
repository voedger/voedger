/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_AddGDoc(t *testing.T) {
	require := require.New(t)

	var app IAppDef
	docName, recName := NewQName("test", "doc"), NewQName("test", "rec")

	t.Run("must be ok to add document", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		doc := adb.AddGDoc(docName)

		t.Run("must be ok to add doc fields", func(t *testing.T) {
			doc.
				AddField("f1", DataKind_int64, true).
				AddField("f2", DataKind_string, false)
		})

		t.Run("must be ok to add child", func(t *testing.T) {
			rec := adb.AddGRecord(recName)
			doc.AddContainer("rec", recName, 0, Occurs_Unbounded)

			t.Run("must be ok to add rec fields", func(t *testing.T) {
				rec.
					AddField("f1", DataKind_int64, true).
					AddField("f2", DataKind_string, false)
			})
		})

		t.Run("must be ok to make doc abstract", func(t *testing.T) {
			doc.SetAbstract()
		})

		t.Run("must be ok to build", func(t *testing.T) {
			a, err := adb.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})

		require.Equal(fmt.Sprint(doc), fmt.Sprint(app.GDoc(docName)))
	})

	require.NotNil(app)

	t.Run("must be ok to find builded doc", func(t *testing.T) {
		typ := app.Type(docName)
		require.Equal(TypeKind_GDoc, typ.Kind())

		d, ok := typ.(IGDoc)
		require.True(ok)
		require.Equal(TypeKind_GDoc, d.Kind())

		doc := app.GDoc(docName)
		require.Equal(TypeKind_GDoc, doc.Kind())
		require.Equal(d, doc)
		require.NotPanics(func() { doc.isDoc() })
		require.NotPanics(func() { doc.isGDoc() })

		require.NotNil(doc.Field(SystemField_QName))
		require.Equal(doc.SystemField_QName(), doc.Field(SystemField_QName))
		require.NotNil(doc.Field(SystemField_ID))
		require.Equal(doc.SystemField_ID(), doc.Field(SystemField_ID))

		require.Equal(2, doc.UserFieldCount())
		require.Equal(DataKind_int64, doc.Field("f1").DataKind())

		require.True(doc.Abstract())

		require.Equal(1, doc.ContainerCount())
		require.Equal(recName, doc.Container("rec").QName())
		require.Equal(TypeKind_GRecord, doc.Container("rec").Type().Kind())

		t.Run("must be ok to find builded record", func(t *testing.T) {
			typ := app.Type(recName)
			require.Equal(TypeKind_GRecord, typ.Kind())

			r, ok := typ.(IGRecord)
			require.True(ok)
			require.Equal(TypeKind_GRecord, r.Kind())

			rec := app.GRecord(recName)
			require.Equal(TypeKind_GRecord, rec.Kind())
			require.Equal(r, rec)
			require.NotPanics(func() { rec.isGRecord() })

			require.NotNil(rec.Field(SystemField_QName))
			require.Equal(rec.SystemField_QName(), rec.Field(SystemField_QName))
			require.NotNil(rec.Field(SystemField_ID))
			require.Equal(rec.SystemField_ID(), rec.Field(SystemField_ID))
			require.NotNil(rec.Field(SystemField_ParentID))
			require.Equal(rec.SystemField_ParentID(), rec.Field(SystemField_ParentID))
			require.NotNil(rec.Field(SystemField_Container))
			require.Equal(rec.SystemField_Container(), rec.Field(SystemField_Container))

			require.Equal(2, rec.UserFieldCount())
			require.Equal(DataKind_int64, rec.Field("f1").DataKind())

			require.Zero(rec.ContainerCount())
		})
	})

	t.Run("should be ok to enumerate docs", func(t *testing.T) {
		var docs []QName
		for doc := range app.GDocs {
			docs = append(docs, doc.QName())
		}
		require.Len(docs, 1)
		require.Equal(docName, docs[0])
		t.Run("should be ok to enumerate recs", func(t *testing.T) {
			var recs []QName
			for rec := range app.GRecords {
				recs = append(recs, rec.QName())
			}
			require.Len(recs, 1)
			require.Equal(recName, recs[0])
		})
	})

	t.Run("check nil returns", func(t *testing.T) {
		unknown := NewQName("test", "unknown")
		require.Equal(NullType, app.Type(unknown))
		require.Nil(app.GDoc(unknown))
		require.Nil(app.GRecord(unknown))
		require.Nil(app.CDoc(unknown))
		require.Nil(app.CRecord(unknown))
		require.Nil(app.WDoc(unknown))
		require.Nil(app.WRecord(unknown))
		require.Nil(app.ODoc(unknown))
		require.Nil(app.ORecord(unknown))
		require.Nil(app.Object(unknown))
	})

	require.Panics(func() {
		New().AddGDoc(NullQName)
	}, require.Is(ErrMissedError))

	require.Panics(func() {
		New().AddGDoc(NewQName("naked", "🔫"))
	}, require.Is(ErrInvalidError), require.Has("naked.🔫"))

	t.Run("panic if type with name already exists", func(t *testing.T) {
		testName := NewQName("test", "dupe")
		adb := New()
		adb.AddPackage("test", "test.com/test")
		adb.AddGDoc(testName)
		require.Panics(func() {
			adb.AddGDoc(testName)
		}, require.Is(ErrAlreadyExistsError), require.Has(testName.String()))
	})
}
