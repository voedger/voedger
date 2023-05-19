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
	appDef.Defs(
		func(d appdef.IDef) {
			sch.add(d)
		})
}

// Adds scheme
func (sch DynoBufSchemes) add(def appdef.IDef) {
	db := dynobuffers.NewScheme()

	db.Name = def.QName().String()
	def.Fields(
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

	sch[def.QName()] = db
}
