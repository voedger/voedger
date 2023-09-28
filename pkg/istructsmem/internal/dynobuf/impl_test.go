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

	t.Run("must ok to build application", func(t *testing.T) {
		appDefBuilder := appdef.New()
		obj := appDefBuilder.AddObject(appdef.NewQName("test", "obj"))
		obj.
			AddField("int32Field", appdef.DataKind_int32, true).
			AddField("int64Field", appdef.DataKind_int64, false).
			AddField("float32Field", appdef.DataKind_float32, false).
			AddField("float64Field", appdef.DataKind_float64, false).
			AddBytesField("bytesField", false).
			AddStringField("strField", false).
			AddField("qnameField", appdef.DataKind_QName, false).
			AddField("recIDField", appdef.DataKind_RecordID, false)
		obj.
			AddContainer("child", appdef.NewQName("test", "el"), 1, appdef.Occurs_Unbounded)

		el := appDefBuilder.AddElement(appdef.NewQName("test", "el"))
		el.
			AddField("int32Field", appdef.DataKind_int32, true).
			AddField("int64Field", appdef.DataKind_int64, false).
			AddField("float32Field", appdef.DataKind_float32, false).
			AddField("float64Field", appdef.DataKind_float64, false).
			AddBytesField("bytesField", false).
			AddStringField("strField", false).
			AddField("qnameField", appdef.DataKind_QName, false).
			AddField("boolField", appdef.DataKind_bool, false).
			AddField("recIDField", appdef.DataKind_RecordID, false)
		el.
			AddContainer("grandChild", appdef.NewQName("test", "el1"), 0, 1)

		subEl := appDefBuilder.AddElement(appdef.NewQName("test", "el1"))
		subEl.
			AddField("recIDField", appdef.DataKind_RecordID, false)

		sch, err := appDefBuilder.Build()
		require.NoError(err)

		view := appDefBuilder.AddView(appdef.NewQName("test", "view"))
		view.Key().Partition().AddField("pk1", appdef.DataKind_int64)
		view.Key().ClustCols().AddStringField("cc1", 100)
		view.Value().AddRefField("val1", true)
		appDef = sch
	})

	schemes := newSchemes()
	require.NotNil(schemes)

	schemes.Prepare(appDef)

	var checkScheme func(dynoScheme *dynobuffers.Scheme)

	checkScheme = func(dynoScheme *dynobuffers.Scheme) {
		require.NotNil(dynoScheme)

		typeName, err := appdef.ParseQName(dynoScheme.Name)
		require.NoError(err)

		typ := appDef.TypeByName(typeName)
		require.NotNil(typ)

		for _, fld := range dynoScheme.Fields {
			if fld.Ft == dynobuffers.FieldTypeObject {
				cont, ok := typ.(appdef.IContainers)
				require.True(ok)

				c := cont.Container(fld.Name)
				require.NotNil(c)

				require.Equal(fld.IsMandatory, c.MinOccurs() > 0)
				require.Equal(fld.IsArray, c.MaxOccurs() > 1)

				require.NotNil(fld.FieldScheme)

				checkScheme(fld.FieldScheme)

				continue
			}

			field := typ.(appdef.IFields).Field(fld.Name)
			require.NotNil(field)

			require.Equal(DataKindToFieldType(field.DataKind()), fld.Ft)
		}
	}

	appDef.Types(
		func(typ appdef.IType) {
			if _, ok := typ.(appdef.IFields); ok {
				checkScheme(schemes[typ.QName()])
			}
		})
}
