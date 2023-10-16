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

		view := appDefBuilder.AddView(appdef.NewQName("test", "view"))
		view.KeyBuilder().PartKeyBuilder().AddField("pk1", appdef.DataKind_int64)
		view.KeyBuilder().ClustColsBuilder().AddStringField("cc1", 100)
		view.ValueBuilder().AddRefField("val1", true)

		sch, err := appDefBuilder.Build()
		require.NoError(err)

		appDef = sch
	})

	schemes := newSchemes()
	require.NotNil(schemes)

	schemes.Prepare(appDef)

	checkScheme := func(name appdef.QName, fields appdef.IFields, dynoScheme *dynobuffers.Scheme) {
		require.NotNil(dynoScheme, "dynobuffer scheme for «%v» not found", name)

		require.EqualValues(len(dynoScheme.FieldsMap), fields.UserFieldCount())

		for _, fld := range dynoScheme.Fields {
			field := fields.Field(fld.Name)
			require.NotNil(field)
			require.Equal(DataKindToFieldType(field.DataKind()), fld.Ft)
		}

		fields.Fields(func(field appdef.IField) {
			if !field.IsSys() {
				fld, ok := dynoScheme.FieldsMap[field.Name()]
				require.True(ok)
				require.Equal(DataKindToFieldType(field.DataKind()), fld.Ft)
			}
		})
	}

	appDef.Types(
		func(typ appdef.IType) {
			name := typ.QName()
			if view, ok := typ.(appdef.IView); ok {
				checkScheme(name, view.Key().PartKey(), schemes.ViewPartKeyScheme(name))
				checkScheme(name, view.Key().ClustCols(), schemes.ViewClustColsScheme(name))
				checkScheme(name, view.Value(), schemes.Scheme(name))
				return
			}
			if fld, ok := typ.(appdef.IFields); ok {
				checkScheme(name, fld, schemes.Scheme(name))
			}
		})
}
