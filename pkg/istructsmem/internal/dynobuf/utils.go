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

func NewFieldsScheme(name string, fields appdef.IWithFields) *dynobuffers.Scheme {
	db := dynobuffers.NewScheme()

	db.Name = name
	for _, f := range fields.Fields() {
		if !f.IsSys() { // #18142: extract system fields from dynobuffer
			ft := DataKindToFieldType(f.DataKind())
			if ft == dynobuffers.FieldTypeByte {
				switch f.DataKind() {
				case appdef.DataKind_int8: // #3435 [~server.vsql.smallints/cmp.istructsmem~impl]
					db.AddField(f.Name(), ft, false)
				case appdef.DataKind_QName:
					db.AddArray(f.Name(), ft, false) // two fixed bytes LittleEndian
				default: // bytes, record, event
					db.AddArray(f.Name(), ft, false) // variable length
				}
			} else {
				db.AddField(f.Name(), ft, false)
			}
		}
	}

	return db
}
