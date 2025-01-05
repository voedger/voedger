/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
)

func TestDynoBufSchemesBasicUsage(t *testing.T) {
	docName := appdef.NewQName("test", "doc")
	viewName := appdef.NewQName("test", "view")

	schemes := New()

	schemes.Prepare(
		func() appdef.IAppDef {
			adb := builder.New()
			wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

			wsb.AddODoc(docName).
				AddField("f1", appdef.DataKind_int32, true).
				AddField("f2", appdef.DataKind_QName, false).
				AddField("f3", appdef.DataKind_string, false).
				AddField("f4", appdef.DataKind_bytes, false)

			v := wsb.AddView(viewName)
			v.Key().PartKey().AddField("pkF1", appdef.DataKind_int32)
			v.Key().ClustCols().AddField("ccF1", appdef.DataKind_string, constraints.MaxLen(100))
			v.Value().AddField("valF1", appdef.DataKind_Event, true)

			return adb.MustBuild()
		}())

	t.Run("should be ok to retrieve schemes", func(t *testing.T) {
		require := require.New(t)

		t.Run("document scheme", func(t *testing.T) {
			scheme := schemes.Scheme(docName)
			require.NotNil(scheme, "schemes.Scheme returns nil", "docName: %q", docName)

			require.Len(scheme.Fields, 4)

			require.Equal("f1", scheme.Fields[0].Name)
			require.Equal(dynobuffers.FieldTypeInt32, scheme.Fields[0].Ft)

			require.Equal("f2", scheme.Fields[1].Name)
			require.Equal(dynobuffers.FieldTypeByte, scheme.Fields[1].Ft)
			require.True(scheme.Fields[1].IsArray)

			require.Equal("f3", scheme.Fields[2].Name)
			require.Equal(dynobuffers.FieldTypeString, scheme.Fields[2].Ft)

			require.Equal("f4", scheme.Fields[3].Name)
			require.Equal(dynobuffers.FieldTypeByte, scheme.Fields[3].Ft)
			require.True(scheme.Fields[1].IsArray)
		})

		t.Run("view scheme", func(t *testing.T) {

			t.Run("key schemes", func(t *testing.T) {

				t.Run("part key", func(t *testing.T) {
					pk := schemes.ViewPartKeyScheme(viewName)
					require.NotNil(pk, "schemes.ViewPartKeyScheme returns nil", "viewName: %q", viewName)

					require.Len(pk.Fields, 1)

					require.Equal("pkF1", pk.Fields[0].Name)
					require.Equal(dynobuffers.FieldTypeInt32, pk.Fields[0].Ft)
				})

				t.Run("ccols", func(t *testing.T) {
					ccols := schemes.ViewClustColsScheme(viewName)
					require.NotNil(ccols, "schemes.ViewClustColsScheme returns nil", "viewName: %q", viewName)

					require.Len(ccols.Fields, 1)

					require.Equal("ccF1", ccols.Fields[0].Name)
					require.Equal(dynobuffers.FieldTypeString, ccols.Fields[0].Ft)
				})
			})

			t.Run("value", func(t *testing.T) {
				val := schemes.Scheme(viewName)
				require.NotNil(val, "schemes.ViewValueScheme returns nil", "viewName: %q", viewName)

				require.Len(val.Fields, 1)

				require.Equal("valF1", val.Fields[0].Name)
				require.Equal(dynobuffers.FieldTypeByte, val.Fields[0].Ft)
				require.True(val.Fields[0].IsArray)
			})
		})
	})
}
