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

func Test_GDocs(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	var app appdef.IAppDef

	t.Run("should be ok to add document", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		doc := ws.AddGDoc(docName)
		doc.AddField("f1", appdef.DataKind_int64, true)
		doc.AddContainer("rec", recName, 0, appdef.Occurs_Unbounded)
		rec := ws.AddGRecord(recName)
		rec.AddField("f1", appdef.DataKind_int64, true)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith := func(tested types.IWithTypes) {
		t.Run("should be ok to find builded doc", func(t *testing.T) {
			doc := appdef.GDoc(tested.Type, docName)
			require.Equal(appdef.TypeKind_GDoc, doc.Kind())
			doc.IsGDoc()

			require.Equal(appdef.TypeKind_GRecord, doc.Container("rec").Type().Kind())

			t.Run("should be ok to find builded record", func(t *testing.T) {
				rec := appdef.GRecord(tested.Type, recName)
				require.Equal(appdef.TypeKind_GRecord, rec.Kind())
				rec.IsGRecord()

				require.Zero(rec.ContainerCount())
			})
		})

		unknownName := appdef.NewQName("test", "unknown")
		require.Nil(appdef.GDoc(tested.Type, unknownName))
		require.Nil(appdef.GRecord(tested.Type, unknownName))

		t.Run("should be ok to enumerate docs", func(t *testing.T) {
			var docs []appdef.QName
			for doc := range appdef.GDocs(tested.Types()) {
				docs = append(docs, doc.QName())
			}
			require.Equal([]appdef.QName{docName}, docs)
			t.Run("should be ok to enumerate recs", func(t *testing.T) {
				var recs []appdef.QName
				for rec := range appdef.GRecords(tested.Types()) {
					recs = append(recs, rec.QName())
				}
				require.Equal([]appdef.QName{recName}, recs)
			})
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}
