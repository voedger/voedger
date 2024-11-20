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

	testWith := func(tested testedTypes) {
		t.Run("should be ok to find builded doc", func(t *testing.T) {
			typ := tested.Type(docName)
			require.Equal(TypeKind_CDoc, typ.Kind())

			doc := CDoc(tested.Type, docName)
			require.Equal(TypeKind_CDoc, doc.Kind())
			require.Equal(typ.(ICDoc), doc)
			require.NotPanics(func() { doc.isCDoc() })

			require.Equal(wsName, doc.Workspace().QName())
			require.Equal(2, doc.UserFieldCount())
			require.Equal(DataKind_int64, doc.Field("f1").DataKind())

			require.Equal(TypeKind_CRecord, doc.Container("rec").Type().Kind())

			t.Run("should be ok to find builded record", func(t *testing.T) {
				typ := tested.Type(recName)
				require.Equal(TypeKind_CRecord, typ.Kind())

				rec := CRecord(tested.Type, recName)
				require.Equal(TypeKind_CRecord, rec.Kind())
				require.Equal(typ.(ICRecord), rec)

				require.Equal(wsName, rec.Workspace().QName())
				require.Equal(2, rec.UserFieldCount())
				require.Equal(DataKind_int64, rec.Field("f1").DataKind())

				require.Zero(rec.ContainerCount())
			})
		})

		unknownName := NewQName("test", "unknown")
		require.Nil(CDoc(tested.Type, unknownName))
		require.Nil(CRecord(tested.Type, unknownName))

		t.Run("should be ok to enumerate docs", func(t *testing.T) {
			var docs []QName
			for doc := range CDocs(tested.Types) {
				docs = append(docs, doc.QName())
			}
			require.Len(docs, 1)
			require.Equal(docName, docs[0])
			t.Run("should be ok to enumerate recs", func(t *testing.T) {
				var recs []QName
				for rec := range CRecords(tested.Types) {
					recs = append(recs, rec.QName())
				}
				require.Len(recs, 1)
				require.Equal(recName, recs[0])
			})
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}

func Test_AppDef_AddCDocSingleton(t *testing.T) {
	require := require.New(t)

	wsName := NewQName("test", "workspace")
	stName := NewQName("test", "singleton")
	docName := NewQName("test", "doc")

	var app IAppDef

	t.Run("should be ok to add singleton", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		st := wsb.AddCDoc(stName)
		st.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)
		st.SetSingleton()

		_ = wsb.AddCDoc(docName).
			AddField("f1", DataKind_int64, true)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested testedTypes) {
		t.Run("should be ok to find builded singleton", func(t *testing.T) {
			typ := tested.Type(stName)
			require.Equal(TypeKind_CDoc, typ.Kind())

			st := CDoc(tested.Type, stName)
			require.Equal(TypeKind_CDoc, st.Kind())
			require.Equal(typ.(ICDoc), st)

			require.True(st.Singleton())
		})

		t.Run("should be ok to enum singleton", func(t *testing.T) {
			names := QNames{}
			for st := range Singletons(tested.Types) {
				names = append(names, st.QName())
			}
			require.Len(names, 1)
			require.Equal(stName, names[0])
		})

		require.Nil(Singleton(tested.Type, NewQName("test", "unknown")), "should be nil if unknown")
		require.Nil(Singleton(tested.Type, docName), "should be nil if not singleton")
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}
