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

func Test_ODocs(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	var app appdef.IAppDef

	t.Run("should be ok to add document", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		doc := ws.AddODoc(docName)
		doc.AddField("f1", appdef.DataKind_int64, true)
		doc.AddContainer("rec", recName, 0, appdef.Occurs_Unbounded)
		rec := ws.AddORecord(recName)
		rec.AddField("f1", appdef.DataKind_int64, true)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested types.IWithTypes) {
		t.Run("should be ok to find builded doc", func(t *testing.T) {
			doc := appdef.ODoc(tested.Type, docName)
			require.Equal(appdef.TypeKind_ODoc, doc.Kind())
			doc.IsODoc()

			require.Equal(appdef.TypeKind_ORecord, doc.Container("rec").Type().Kind())

			t.Run("should be ok to find builded record", func(t *testing.T) {
				rec := appdef.ORecord(tested.Type, recName)
				require.Equal(appdef.TypeKind_ORecord, rec.Kind())
				rec.IsORecord()

				require.Zero(rec.ContainerCount())
			})
		})

		unknownName := appdef.NewQName("test", "unknown")
		require.Nil(appdef.ODoc(tested.Type, unknownName))
		require.Nil(appdef.ORecord(tested.Type, unknownName))

		t.Run("should be ok to enumerate docs", func(t *testing.T) {
			var docs []appdef.QName
			for doc := range appdef.ODocs(tested.Types()) {
				docs = append(docs, doc.QName())
			}
			require.Equal([]appdef.QName{docName}, docs)
			t.Run("should be ok to enumerate recs", func(t *testing.T) {
				var recs []appdef.QName
				for rec := range appdef.ORecords(tested.Types()) {
					recs = append(recs, rec.QName())
				}
				require.Equal([]appdef.QName{recName}, recs)
			})
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}
