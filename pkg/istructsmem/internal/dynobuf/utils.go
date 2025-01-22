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

func NewFieldsScheme(name string, fields appdef.IWithFields) *dynobuffers.Scheme {
	db := dynobuffers.NewScheme()

	db.Name = name
	for _, f := range fields.Fields() {
		if !f.IsSys() { // #18142: extract system fields from dynobuffer
			ft := DataKindToFieldType(f.DataKind())
			if ft == dynobuffers.FieldTypeByte {
				db.AddArray(f.Name(), ft, false)
			} else {
				db.AddField(f.Name(), ft, false)
			}
		}
	}

	return db
}
