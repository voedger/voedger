/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/appdef"
)

// Converts appdef.DataKind to dynobuffers.FieldType
func DataKindToFieldType(kind appdef.DataKind) dynobuffers.FieldType {
	return dataKindToDynoFieldType[kind]
}

// Converts dynobuffers FieldType to string
func FieldTypeToString(ft dynobuffers.FieldType) string {
	return dynobufferFieldTypeToStr[ft]
}

func NewFieldsScheme(name string, fields appdef.IFields) *dynobuffers.Scheme {
	db := dynobuffers.NewScheme()

	db.Name = name
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

	return db
}
