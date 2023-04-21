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

func Test_DynoBufSchemasCache(t *testing.T) {
	require := require.New(t)

	var schemaCache schemas.SchemaCache

	t.Run("must ok to build schemas", func(t *testing.T) {
		bld := schemas.NewSchemaCache()
		rootSchema := bld.Add(schemas.NewQName("test", "rootSchema"), schemas.SchemaKind_Object)
		rootSchema.
			AddField("int32Field", schemas.DataKind_int32, true).
			AddField("int64Field", schemas.DataKind_int64, false).
			AddField("float32Field", schemas.DataKind_float32, false).
			AddField("float64Field", schemas.DataKind_float64, false).
			AddField("bytesField", schemas.DataKind_bytes, false).
			AddField("strField", schemas.DataKind_string, false).
			AddField("qnameField", schemas.DataKind_QName, false).
			AddField("recIDField", schemas.DataKind_RecordID, false).
			AddContainer("child", schemas.NewQName("test", "childSchema"), 1, schemas.Occurs_Unbounded)

		childSchema := bld.Add(schemas.NewQName("test", "childSchema"), schemas.SchemaKind_Element)
		childSchema.
			AddField("int32Field", schemas.DataKind_int32, true).
			AddField("int64Field", schemas.DataKind_int64, false).
			AddField("float32Field", schemas.DataKind_float32, false).
			AddField("float64Field", schemas.DataKind_float64, false).
			AddField("bytesField", schemas.DataKind_bytes, false).
			AddField("strField", schemas.DataKind_string, false).
			AddField("qnameField", schemas.DataKind_QName, false).
			AddField("boolField", schemas.DataKind_bool, false).
			AddField("recIDField", schemas.DataKind_RecordID, false).
			AddContainer("grandChild", schemas.NewQName("test", "grandChild"), 0, 1)

		grandSchema := bld.Add(schemas.NewQName("test", "grandChild"), schemas.SchemaKind_Element)
		grandSchema.
			AddField("recIDField", schemas.DataKind_RecordID, false)

		sch, err := bld.Build()
		require.NoError(err)

		schemaCache = sch
	})

	dynoSchemas := newSchemasCache()
	require.NotNil(dynoSchemas)

	dynoSchemas.Prepare(schemaCache)

	var checkDynoScheme func(dynoScheme *dynobuffers.Scheme)

	checkDynoScheme = func(dynoScheme *dynobuffers.Scheme) {
		require.NotNil(dynoScheme)

		schemaName, err := schemas.ParseQName(dynoScheme.Name)
		require.NoError(err)

		schema := schemaCache.SchemaByName(schemaName)
		require.NotNil(schema)

		for _, dynoField := range dynoScheme.Fields {
			if dynoField.Ft == dynobuffers.FieldTypeObject {
				cont := schema.Container(dynoField.Name)
				require.NotNil(cont)

				require.Equal(dynoField.IsMandatory, cont.MinOccurs() > 0)
				require.Equal(dynoField.IsArray, cont.MaxOccurs() > 1)

				require.NotNil(dynoField.FieldScheme)

				checkDynoScheme(dynoField.FieldScheme)

				continue
			}

			field := schema.Field(dynoField.Name)
			require.NotNil(field)

			require.Equal(DataKindToFieldType(field.DataKind()), dynoField.Ft)
		}
	}

	schemaCache.EnumSchemas(
		func(s schemas.Schema) {
			checkDynoScheme(dynoSchemas[s.QName()])
		})
}
