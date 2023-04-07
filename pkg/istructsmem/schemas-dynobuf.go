/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	dynobuffers "github.com/untillpro/dynobuffers"
	"github.com/untillpro/voedger/pkg/istructs"
)

var dataKindToDynoFieldType = map[istructs.DataKindType]dynobuffers.FieldType{
	istructs.DataKind_null:     dynobuffers.FieldTypeUnspecified,
	istructs.DataKind_int32:    dynobuffers.FieldTypeInt32,
	istructs.DataKind_int64:    dynobuffers.FieldTypeInt64,
	istructs.DataKind_float32:  dynobuffers.FieldTypeFloat32,
	istructs.DataKind_float64:  dynobuffers.FieldTypeFloat64,
	istructs.DataKind_bytes:    dynobuffers.FieldTypeByte,
	istructs.DataKind_string:   dynobuffers.FieldTypeString,
	istructs.DataKind_QName:    dynobuffers.FieldTypeByte, // two fixed bytes LittleEndian
	istructs.DataKind_bool:     dynobuffers.FieldTypeBool,
	istructs.DataKind_RecordID: dynobuffers.FieldTypeInt64,
	istructs.DataKind_Record:   dynobuffers.FieldTypeByte,
	istructs.DataKind_Event:    dynobuffers.FieldTypeByte,
}

// createDynoBufferScheme: recreate schema *dynobuffers.Scheme
func (sch *SchemaType) createDynoBufferScheme() *dynobuffers.Scheme {
	sch.dynoScheme = dynobuffers.NewScheme()
	sch.dynoScheme.Name = sch.name.String()

	addFieldToDynoSchema := func(fieldName string, kind istructs.DataKindType) {
		if sysField(fieldName) {
			return // #18142: extract system fields from dynobuffer
		}
		dynoType := dataKindToDynoFieldType[kind]
		if dynoType == dynobuffers.FieldTypeByte {
			sch.dynoScheme.AddArray(fieldName, dynoType, false)
		} else {
			sch.dynoScheme.AddField(fieldName, dynoType, false)
		}
	}

	sch.Fields(addFieldToDynoSchema)

	return sch.dynoScheme
}

// createDynoBufferSchemes: recreate *dynobuffers.Scheme for all schemas.  Must calls at application start
func (cache *SchemasCacheType) createDynoBufferSchemes() {
	for _, sch := range cache.schemas {
		sch.createDynoBufferScheme()
	}
}
