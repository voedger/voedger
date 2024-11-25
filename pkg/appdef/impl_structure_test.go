/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_StructuresAndRecords(t *testing.T) {
	require := require.New(t)

	wsName := NewQName("test", "workspace")
	docName, recName := NewQName("test", "doc"), NewQName("test", "rec")
	objName := NewQName("test", "obj")

	var app IAppDef

	t.Run("should be ok to add structures", func(t *testing.T) {
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

		obj := wsb.AddObject(objName)
		obj.
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_string, false)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested testedTypes) {
		t.Run("should be ok to find builded structures", func(t *testing.T) {
			findStruct := func(n QName, kind TypeKind) {
				typ := tested.Type(n)
				require.Equal(kind, typ.Kind())

				doc := Structure(tested.Type, n)
				require.Equal(kind, doc.Kind())

				require.Equal(wsName, doc.Workspace().QName())

				require.Equal(2, doc.UserFieldCount())
				require.Equal(DataKind_int64, doc.Field("f1").DataKind())
				require.Equal(DataKind_string, doc.Field("f2").DataKind())
			}
			findStruct(docName, TypeKind_ODoc)
			findStruct(recName, TypeKind_ORecord)
			findStruct(objName, TypeKind_Object)
		})

		require.Nil(Structure(tested.Type, NewQName("test", "unknown")), "should nil if not found")

		t.Run("should be ok to enumerate structures", func(t *testing.T) {
			var str []QName
			for s := range Structures(tested.Types) {
				str = append(str, s.QName())
			}
			require.Equal(str, []QName{docName, objName, recName})
		})

		t.Run("should be ok to find builded records", func(t *testing.T) {
			findRecord := func(n QName, kind TypeKind) {
				typ := tested.Type(n)
				require.Equal(kind, typ.Kind())

				rec := Record(tested.Type, n)
				require.Equal(kind, rec.Kind())

				require.Equal(wsName, rec.Workspace().QName())

				require.Equal(2, rec.UserFieldCount())
				require.Equal(DataKind_int64, rec.Field("f1").DataKind())
				require.Equal(DataKind_string, rec.Field("f2").DataKind())
			}
			findRecord(docName, TypeKind_ODoc)
			findRecord(recName, TypeKind_ORecord)
		})

		require.Nil(Record(tested.Type, NewQName("test", "unknown")), "should nil if not found")
		require.Nil(Record(tested.Type, objName), "should nil if not record")

		t.Run("should be ok to enumerate records", func(t *testing.T) {
			var recs []QName
			for s := range Records(tested.Types) {
				recs = append(recs, s.QName())
			}
			require.Equal(recs, []QName{docName, recName})
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}
