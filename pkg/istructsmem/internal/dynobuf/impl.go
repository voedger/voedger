/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/appdef"
)

func newSchemes() DynoBufSchemes {
	cache := DynoBufSchemes{}
	return cache
}

// Prepares schemes
func (sch DynoBufSchemes) Prepare(appDef appdef.IAppDef) {
	appDef.Schemas(
		func(schema appdef.Schema) {
			sch.add(schema)
		})
}

// Adds schema
func (sch DynoBufSchemes) add(schema appdef.Schema) {
	db := dynobuffers.NewScheme()

	db.Name = schema.QName().String()
	schema.Fields(
		func(f appdef.Field) {
			if !f.IsSys() { // #18142: extract system fields from dynobuffer
				fieldType := DataKindToFieldType(f.DataKind())
				if fieldType == dynobuffers.FieldTypeByte {
					db.AddArray(f.Name(), fieldType, false)
				} else {
					db.AddField(f.Name(), fieldType, false)
				}
			}
		})

	sch[schema.QName()] = db
}
