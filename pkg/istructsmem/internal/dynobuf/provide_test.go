/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/schemas"
)

func Test_BasicUsage(t *testing.T) {
	name := schemas.NewQName("test", "test")

	dynoSchemas := NewSchemasCache()

	dynoSchemas.Prepare(
		func() schemas.SchemaCache {
			bld := schemas.NewSchemaCache()
			schema := bld.Add(name, schemas.SchemaKind_CDoc)
			schema.AddField("f1", schemas.DataKind_int32, true)
			schema.AddField("f2", schemas.DataKind_QName, false)
			return bld
		}())

	t.Run("let test basic methods", func(t *testing.T) {
		require := require.New(t)

		schema := dynoSchemas[name]
		require.NotNil(schema, "DynoBufferSchema returns nil")

		require.Len(schema.Fields, 2)

		require.Equal("f1", schema.Fields[0].Name)
		require.Equal(dynobuffers.FieldTypeInt32, schema.Fields[0].Ft)
		require.Equal("int32", FieldTypeToString(schema.Fields[0].Ft))

		require.Equal("f2", schema.Fields[1].Name)
		require.Equal(dynobuffers.FieldTypeByte, schema.Fields[1].Ft)
		require.True(schema.Fields[1].IsArray)
	})
}
