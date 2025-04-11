/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/istructsmem/internal/dynobuf"
)

func TestDynoBufSchemesBasicUsage(t *testing.T) {
	docName := appdef.NewQName("test", "doc")
	viewName := appdef.NewQName("test", "view")

	schemes := dynobuf.New()

	schemes.Prepare(
		func() appdef.IAppDef {
			adb := builder.New()
			wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

			wsb.AddODoc(docName).
				AddField("f_int8", appdef.DataKind_int8, true).    // #3434 [small integers: int8]
				AddField("f_int16", appdef.DataKind_int16, false). // #3434 [small integers: int16]
				AddField("f_int32", appdef.DataKind_int32, false).
				AddField("f_QName", appdef.DataKind_QName, false).
				AddField("f_String", appdef.DataKind_string, false).
				AddField("f_Bytes", appdef.DataKind_bytes, false)

			v := wsb.AddView(viewName)
			v.Key().PartKey().AddField("pkF1", appdef.DataKind_int32)
			v.Key().ClustCols().
				AddField("ccInt8", appdef.DataKind_int8).   // #3434 [small integers: int8]
				AddField("ccInt16", appdef.DataKind_int16). // #3434 [small integers: int16]
				AddField("ccString", appdef.DataKind_string, constraints.MaxLen(100))
			v.Value().AddField("valF1", appdef.DataKind_Event, true)

			return adb.MustBuild()
		}())

	t.Run("should be ok to retrieve schemes", func(t *testing.T) {
		require := require.New(t)

		t.Run("document scheme", func(t *testing.T) {
			scheme := schemes.Scheme(docName)
			require.NotNil(scheme, "schemes.Scheme returns nil", "docName: %q", docName)

			require.Len(scheme.Fields, 6)

			// #3434 [small integers: int8]
			require.Equal("f_int8", scheme.Fields[0].Name)
			require.Equal(dynobuffers.FieldTypeByte, scheme.Fields[0].Ft)

			// #3434 [small integers: int16]
			require.Equal("f_int16", scheme.Fields[1].Name)
			require.Equal(dynobuffers.FieldTypeInt16, scheme.Fields[1].Ft)

			require.Equal("f_int32", scheme.Fields[2].Name)
			require.Equal(dynobuffers.FieldTypeInt32, scheme.Fields[2].Ft)

			require.Equal("f_QName", scheme.Fields[3].Name)
			require.Equal(dynobuffers.FieldTypeByte, scheme.Fields[3].Ft)
			require.True(scheme.Fields[3].IsArray)

			require.Equal("f_String", scheme.Fields[4].Name)
			require.Equal(dynobuffers.FieldTypeString, scheme.Fields[4].Ft)

			require.Equal("f_Bytes", scheme.Fields[5].Name)
			require.Equal(dynobuffers.FieldTypeByte, scheme.Fields[5].Ft)
			require.True(scheme.Fields[5].IsArray)
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

					require.Len(ccols.Fields, 3)

					// #3434 [small integers: int8]
					require.Equal("ccInt8", ccols.Fields[0].Name)
					require.Equal(dynobuffers.FieldTypeByte, ccols.Fields[0].Ft)
					require.False(ccols.Fields[0].IsArray)

					// #3434 [small integers: int16]
					require.Equal("ccInt16", ccols.Fields[1].Name)
					require.Equal(dynobuffers.FieldTypeInt16, ccols.Fields[1].Ft)

					require.Equal("ccString", ccols.Fields[2].Name)
					require.Equal(dynobuffers.FieldTypeString, ccols.Fields[2].Ft)
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
