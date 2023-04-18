/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

func Test_DynoBufSchemasCache(t *testing.T) {
	require := require.New(t)

	schemaCache := schemas.NewSchemaCache()

	t.Run("must ok to build schemas", func(t *testing.T) {
		rootSchema := schemaCache.Add(istructs.NewQName("test", "rootSchema"), istructs.SchemaKind_Object)
		rootSchema.
			AddField("int32Field", istructs.DataKind_int32, true).
			AddField("int64Field", istructs.DataKind_int64, false).
			AddField("float32Field", istructs.DataKind_float32, false).
			AddField("float64Field", istructs.DataKind_float64, false).
			AddField("bytesField", istructs.DataKind_bytes, false).
			AddField("strField", istructs.DataKind_string, false).
			AddField("qnameField", istructs.DataKind_QName, false).
			AddField("recIDField", istructs.DataKind_RecordID, false).
			AddContainer("child", istructs.NewQName("test", "childSchema"), 1, istructs.ContainerOccurs_Unbounded)

		childSchema := schemaCache.Add(istructs.NewQName("test", "childSchema"), istructs.SchemaKind_Element)
		childSchema.
			AddField("int32Field", istructs.DataKind_int32, true).
			AddField("int64Field", istructs.DataKind_int64, false).
			AddField("float32Field", istructs.DataKind_float32, false).
			AddField("float64Field", istructs.DataKind_float64, false).
			AddField("bytesField", istructs.DataKind_bytes, false).
			AddField("strField", istructs.DataKind_string, false).
			AddField("qnameField", istructs.DataKind_QName, false).
			AddField("boolField", istructs.DataKind_bool, false).
			AddField("recIDField", istructs.DataKind_RecordID, false).
			AddContainer("grandChild", istructs.NewQName("test", "grandChild"), 0, 1)

		grandSchema := schemaCache.Add(istructs.NewQName("test", "grandChild"), istructs.SchemaKind_Element)
		grandSchema.
			AddField("recIDField", istructs.DataKind_RecordID, false)

		require.NoError(schemaCache.ValidateSchemas())
	})

	dynoSchemas := newSchemasCache(schemaCache)
	require.NotNil(dynoSchemas)

	var checkDynoScheme func(dynoScheme *dynobuffers.Scheme)

	checkDynoScheme = func(dynoScheme *dynobuffers.Scheme) {
		require.NotNil(dynoScheme)

		schemaName, err := istructs.ParseQName(dynoScheme.Name)
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
		func(s *schemas.Schema) {
			checkDynoScheme(dynoSchemas[s.QName()])
		})
}
