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
	appDef.Types(
		func(t appdef.IType) {
			if fld, ok := t.(appdef.IFields); ok {
				sch.add(t.QName(), fld)
			}
		})
}

// Adds scheme
func (sch DynoBufSchemes) add(name appdef.QName, fields appdef.IFields) {
	db := dynobuffers.NewScheme()

	db.Name = name.String()
	fields.Fields(
		func(f appdef.IField) {
			if !f.IsSys() { // #18142: extract system fields from dynobuffer
				fieldType := DataKindToFieldType(f.DataKind())
				if fieldType == dynobuffers.FieldTypeByte {
					db.AddArray(f.Name(), fieldType, false)
				} else {
					db.AddField(f.Name(), fieldType, false)
				}
			}
		})

	sch[name] = db
}
