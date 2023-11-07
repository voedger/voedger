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
)

func TestDynoBufSchemesBasicUsage(t *testing.T) {
	docName := appdef.NewQName("test", "cdoc")
	viewName := appdef.NewQName("test", "view")

	schemes := New()

	schemes.Prepare(
		func() appdef.IAppDef {
			app := appdef.New()
			app.AddCDoc(docName).
				AddField("f1", appdef.DataKind_int32, true).
				AddField("f2", appdef.DataKind_QName, false)

			v := app.AddView(viewName)
			v.KeyBuilder().PartKeyBuilder().AddField("pkF1", appdef.DataKind_int32)
			v.KeyBuilder().ClustColsBuilder().AddField("ccF1", appdef.DataKind_string, appdef.MaxLen(100))
			v.ValueBuilder().AddField("valF1", appdef.DataKind_Event, true)

			a, _ := app.Build()
			return a
		}())

	t.Run("let test basic methods", func(t *testing.T) {
		require := require.New(t)

		t.Run("document scheme", func(t *testing.T) {
			scheme := schemes.Scheme(docName)
			require.NotNil(scheme, "schemes.Scheme returns nil", "docName: %q", docName)

			require.Len(scheme.Fields, 2)

			require.Equal("f1", scheme.Fields[0].Name)
			require.Equal(dynobuffers.FieldTypeInt32, scheme.Fields[0].Ft)

			require.Equal("f2", scheme.Fields[1].Name)
			require.Equal(dynobuffers.FieldTypeByte, scheme.Fields[1].Ft)
			require.True(scheme.Fields[1].IsArray)
		})

		t.Run("view scheme", func(t *testing.T) {

			t.Run("key scheme", func(t *testing.T) {

				t.Run("part key scheme", func(t *testing.T) {
					pk := schemes.ViewPartKeyScheme(viewName)
					require.NotNil(pk, "schemes.ViewPartKeyScheme returns nil", "viewName: %q", viewName)

					require.Len(pk.Fields, 1)

					require.Equal("pkF1", pk.Fields[0].Name)
					require.Equal(dynobuffers.FieldTypeInt32, pk.Fields[0].Ft)
				})

				t.Run("ccols scheme", func(t *testing.T) {
					ccols := schemes.ViewClustColsScheme(viewName)
					require.NotNil(ccols, "schemes.ViewClustColsScheme returns nil", "viewName: %q", viewName)

					require.Len(ccols.Fields, 1)

					require.Equal("ccF1", ccols.Fields[0].Name)
					require.Equal(dynobuffers.FieldTypeString, ccols.Fields[0].Ft)
				})
			})

			t.Run("value scheme", func(t *testing.T) {
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
