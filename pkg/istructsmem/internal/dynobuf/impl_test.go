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

func Test_DynoBufSchemasCache(t *testing.T) {
	require := require.New(t)

	var schemaCache appdef.SchemaCache

	t.Run("must ok to build schemas", func(t *testing.T) {
		bld := appdef.NewSchemaCache()
		rootSchema := bld.Add(appdef.NewQName("test", "rootSchema"), appdef.SchemaKind_Object)
		rootSchema.
			AddField("int32Field", appdef.DataKind_int32, true).
			AddField("int64Field", appdef.DataKind_int64, false).
			AddField("float32Field", appdef.DataKind_float32, false).
			AddField("float64Field", appdef.DataKind_float64, false).
			AddField("bytesField", appdef.DataKind_bytes, false).
			AddField("strField", appdef.DataKind_string, false).
			AddField("qnameField", appdef.DataKind_QName, false).
			AddField("recIDField", appdef.DataKind_RecordID, false).
			AddContainer("child", appdef.NewQName("test", "childSchema"), 1, appdef.Occurs_Unbounded)

		childSchema := bld.Add(appdef.NewQName("test", "childSchema"), appdef.SchemaKind_Element)
		childSchema.
			AddField("int32Field", appdef.DataKind_int32, true).
			AddField("int64Field", appdef.DataKind_int64, false).
			AddField("float32Field", appdef.DataKind_float32, false).
			AddField("float64Field", appdef.DataKind_float64, false).
			AddField("bytesField", appdef.DataKind_bytes, false).
			AddField("strField", appdef.DataKind_string, false).
			AddField("qnameField", appdef.DataKind_QName, false).
			AddField("boolField", appdef.DataKind_bool, false).
			AddField("recIDField", appdef.DataKind_RecordID, false).
			AddContainer("grandChild", appdef.NewQName("test", "grandChild"), 0, 1)

		grandSchema := bld.Add(appdef.NewQName("test", "grandChild"), appdef.SchemaKind_Element)
		grandSchema.
			AddField("recIDField", appdef.DataKind_RecordID, false)

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

		schemaName, err := appdef.ParseQName(dynoScheme.Name)
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

	schemaCache.Schemas(
		func(s appdef.Schema) {
			checkDynoScheme(dynoSchemas[s.QName()])
		})
}
