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
	name := appdef.NewQName("test", "test")

	schemes := New()

	schemes.Prepare(
		func() appdef.IAppDef {
			app := appdef.New()
			def := app.Add(name, appdef.DefKind_CDoc)
			def.AddField("f1", appdef.DataKind_int32, true)
			def.AddField("f2", appdef.DataKind_QName, false)
			return app
		}())

	t.Run("let test basic methods", func(t *testing.T) {
		require := require.New(t)

		scheme := schemes[name]
		require.NotNil(scheme, "DynoBufferScheme returns nil")

		require.Len(scheme.Fields, 2)

		require.Equal("f1", scheme.Fields[0].Name)
		require.Equal(dynobuffers.FieldTypeInt32, scheme.Fields[0].Ft)
		require.Equal("int32", FieldTypeToString(scheme.Fields[0].Ft))

		require.Equal("f2", scheme.Fields[1].Name)
		require.Equal(dynobuffers.FieldTypeByte, scheme.Fields[1].Ft)
		require.True(scheme.Fields[1].IsArray)
	})
}
