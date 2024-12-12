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

func Test_AppDef_AddCDoc(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	var app appdef.IAppDef

	t.Run("should be ok to add document", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		doc := ws.AddCDoc(docName)
		doc.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)
		doc.AddContainer("rec", recName, 0, appdef.Occurs_Unbounded)
		rec := ws.AddCRecord(recName)
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
			require.Equal(appdef.TypeKind_CDoc, typ.Kind())

			doc := appdef.CDoc(tested.Type, docName)
			require.Equal(appdef.TypeKind_CDoc, doc.Kind())
			require.Equal(typ.(appdef.ICDoc), doc)

			require.Equal(wsName, doc.Workspace().QName())
			require.Equal(2, doc.UserFieldCount())
			require.Equal(appdef.DataKind_int64, doc.Field("f1").DataKind())

			require.Equal(appdef.TypeKind_CRecord, doc.Container("rec").Type().Kind())

			t.Run("should be ok to find builded record", func(t *testing.T) {
				typ := tested.Type(recName)
				require.Equal(appdef.TypeKind_CRecord, typ.Kind())

				rec := appdef.CRecord(tested.Type, recName)
				require.Equal(appdef.TypeKind_CRecord, rec.Kind())
				require.Equal(typ.(appdef.ICRecord), rec)

				require.Equal(wsName, rec.Workspace().QName())
				require.Equal(2, rec.UserFieldCount())
				require.Equal(appdef.DataKind_int64, rec.Field("f1").DataKind())

				require.Zero(rec.ContainerCount())
			})
		})

		unknownName := appdef.NewQName("test", "unknown")
		require.Nil(appdef.CDoc(tested.Type, unknownName))
		require.Nil(appdef.CRecord(tested.Type, unknownName))

		t.Run("should be ok to enumerate docs", func(t *testing.T) {
			var docs []appdef.QName
			for doc := range appdef.CDocs(tested.Types()) {
				docs = append(docs, doc.QName())
			}
			require.Len(docs, 1)
			require.Equal(docName, docs[0])
			t.Run("should be ok to enumerate recs", func(t *testing.T) {
				var recs []appdef.QName
				for rec := range appdef.CRecords(tested.Types()) {
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

	wsName := appdef.NewQName("test", "workspace")
	stName := appdef.NewQName("test", "singleton")
	docName := appdef.NewQName("test", "doc")

	var app appdef.IAppDef

	t.Run("should be ok to add singleton", func(t *testing.T) {
		adb := appdef.New()
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

	testWith := func(tested testedTypes) {
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
