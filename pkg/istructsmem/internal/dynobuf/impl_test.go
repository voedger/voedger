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

func TestDynoBufSchemes(t *testing.T) {
	require := require.New(t)

	var appDef appdef.IAppDef

	t.Run("must ok to build application definition", func(t *testing.T) {
		appDefBuilder := appdef.New()
		rootDef := appDefBuilder.AddStruct(appdef.NewQName("test", "obj"), appdef.DefKind_Object)
		rootDef.
			AddField("int32Field", appdef.DataKind_int32, true).
			AddField("int64Field", appdef.DataKind_int64, false).
			AddField("float32Field", appdef.DataKind_float32, false).
			AddField("float64Field", appdef.DataKind_float64, false).
			AddField("bytesField", appdef.DataKind_bytes, false).
			AddField("strField", appdef.DataKind_string, false).
			AddField("qnameField", appdef.DataKind_QName, false).
			AddField("recIDField", appdef.DataKind_RecordID, false).
			AddContainer("child", appdef.NewQName("test", "el"), 1, appdef.Occurs_Unbounded)

		childDef := appDefBuilder.AddStruct(appdef.NewQName("test", "el"), appdef.DefKind_Element)
		childDef.
			AddField("int32Field", appdef.DataKind_int32, true).
			AddField("int64Field", appdef.DataKind_int64, false).
			AddField("float32Field", appdef.DataKind_float32, false).
			AddField("float64Field", appdef.DataKind_float64, false).
			AddField("bytesField", appdef.DataKind_bytes, false).
			AddField("strField", appdef.DataKind_string, false).
			AddField("qnameField", appdef.DataKind_QName, false).
			AddField("boolField", appdef.DataKind_bool, false).
			AddField("recIDField", appdef.DataKind_RecordID, false).
			AddContainer("grandChild", appdef.NewQName("test", "el1"), 0, 1)

		grandDef := appDefBuilder.AddStruct(appdef.NewQName("test", "el1"), appdef.DefKind_Element)
		grandDef.
			AddField("recIDField", appdef.DataKind_RecordID, false)

		sch, err := appDefBuilder.Build()
		require.NoError(err)

		appDef = sch
	})

	schemes := newSchemes()
	require.NotNil(schemes)

	schemes.Prepare(appDef)

	var checkScheme func(dynoScheme *dynobuffers.Scheme)

	checkScheme = func(dynoScheme *dynobuffers.Scheme) {
		require.NotNil(dynoScheme)

		defName, err := appdef.ParseQName(dynoScheme.Name)
		require.NoError(err)

		def := appDef.DefByName(defName)
		require.NotNil(def)

		for _, fld := range dynoScheme.Fields {
			if fld.Ft == dynobuffers.FieldTypeObject {
				cont := def.Container(fld.Name)
				require.NotNil(cont)

				require.Equal(fld.IsMandatory, cont.MinOccurs() > 0)
				require.Equal(fld.IsArray, cont.MaxOccurs() > 1)

				require.NotNil(fld.FieldScheme)

				checkScheme(fld.FieldScheme)

				continue
			}

			field := def.Field(fld.Name)
			require.NotNil(field)

			require.Equal(DataKindToFieldType(field.DataKind()), fld.Ft)
		}
	}

	appDef.Defs(
		func(s appdef.IDef) {
			checkScheme(schemes[s.QName()])
		})
}
